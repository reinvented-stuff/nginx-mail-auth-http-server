package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var ApplicationDescription = "Nginx Mail Auth HTTP Server"
var BuildVersion = "0.0.0"
var DB *sqlx.DB

type DatabaseStruct struct {
	URI              string `json:"uri"`
	AuthLookupQuery  string `json:"auth_lookup_query"`
	RelayLookupQuery string `json:"relay_lookup_query"`
}

type ConfigurationStruct struct {
	Listen   string         `json:"listen"`
	Database DatabaseStruct `json:"database"`
}

type flagParamsStruct struct {
	address string
	port    string
}

type handleSignalParamsStruct struct {
	httpServer http.Server
	db         *sqlx.DB
}

type authResultStruct struct {
	AuthStatus    string
	AuthServer    string
	AuthPort      int
	AuthWait      string
	AuthErrorCode string
}

var Configuration = ConfigurationStruct{}
var handleSignalParams = handleSignalParamsStruct{}
var flagParams = flagParamsStruct{}

func ReadConfigurationFile(configPtr string, configuration *ConfigurationStruct) {

	configFile, _ := os.Open(configPtr)
	defer configFile.Close()

	JSONDecoder := json.NewDecoder(configFile)

	err := JSONDecoder.Decode(&configuration)
	if err != nil {
		log.Error().
			Err(err).
			Str("stage", "init").
			Msgf("Error while reading config file")
	}
}

type UpstreamStruct struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type QueryParamsStruct struct {
	User     string `db:"User"`
	Pass     string `db:"Pass"`
	RcptTo   string `db:"RcptTo"`
	MailFrom string `db:"MailFrom"`
}

func authenticate(user string, pass string, protocol string, mailFrom string, rcptTo string) (success bool, result authResultStruct, err error) {

	result = authResultStruct{}

	var nonVERPAddress bytes.Buffer
	var query string
	var queryParams = QueryParamsStruct{
		User:     user,
		Pass:     pass,
		RcptTo:   "",
		MailFrom: "",
	}

	if user == "" &&
		pass == "" &&
		mailFrom != "" &&
		rcptTo != "" {
		log.Info().
			Str("protocol", protocol).
			Str("mailFrom", mailFrom).
			Str("rcptTo", rcptTo).
			Msg("Authenticating by relay access")

		mailHeaderRegex := regexp.MustCompile(`<(.*?)(\+.*?)?@(.*?)>`)
		mailFromEmailMatch := mailHeaderRegex.FindStringSubmatch(mailFrom)
		rcptToEmailMatch := mailHeaderRegex.FindStringSubmatch(rcptTo)

		if len(mailFromEmailMatch) == 4 {

			nonVERPAddress.WriteString(mailFromEmailMatch[1])
			nonVERPAddress.WriteString("@")
			nonVERPAddress.WriteString(mailFromEmailMatch[3])

			queryParams.MailFrom = nonVERPAddress.String()

			nonVERPAddress.Reset()

		} else {
			log.Error().
				Str("mailFrom", mailFrom).
				Int("mailFromEmailMatchLen", len(mailFromEmailMatch)).
				Msg("Can't parse MAIL FROM command")

			result.AuthStatus = "Temporary server problem, try again later"
			result.AuthErrorCode = "451 4.3.0"
			result.AuthWait = "5"

			return false, result, err

		}

		if len(rcptToEmailMatch) == 4 {

			nonVERPAddress.WriteString(rcptToEmailMatch[1])
			nonVERPAddress.WriteString("@")
			nonVERPAddress.WriteString(rcptToEmailMatch[3])

			queryParams.RcptTo = nonVERPAddress.String()

			nonVERPAddress.Reset()

		} else {
			log.Error().
				Str("rcptTo", rcptTo).
				Int("rcptToEmailMatchLen", len(rcptToEmailMatch)).
				Msg("Can't parse RCPT TO command")

			result.AuthStatus = "Temporary server problem, try again later"
			result.AuthErrorCode = "451 4.3.0"
			result.AuthWait = "5"

			return false, result, err

		}

		query = Configuration.Database.RelayLookupQuery

	} else {

		log.Info().
			Str("protocol", protocol).
			Str("user", user).
			Str("pass", pass).
			Msg("Authenticating by credentials")

		query = Configuration.Database.AuthLookupQuery
	}

	log.Info().
		Str("query", query).
		Msg("Lookup query")

	queryResult, err := DB.NamedQuery(query, queryParams)

	if err != nil {
		log.Error().Err(err).Msgf("Error while preparing query: %v", err)

		result.AuthStatus = "Temporary server problem, try again later"
		result.AuthErrorCode = "451 4.3.0"
		result.AuthWait = "5"

		return false, result, err
	}

	defer queryResult.Close()

	for queryResult.Next() {
		log.Debug().Msgf("Found results for '%s':'%s'", user, pass)

		var upstream = UpstreamStruct{}

		if err = queryResult.StructScan(&upstream); err != nil {
			log.Fatal().Err(err).Msgf("Error while scanning query results: %v", err)

			result.AuthStatus = "Temporary server problem, try again later"
			result.AuthErrorCode = "451 4.3.0"
			result.AuthWait = "5"

			return false, result, err

		} else {
			log.Debug().Msgf("Successfully scanned query results")
		}

		log.Info().Msgf("Found upstream: %v", upstream.Address)

		result.AuthStatus = "OK"
		result.AuthServer = upstream.Address
		result.AuthPort = upstream.Port

		return true, result, nil

	}

	result.AuthStatus = "Error: authentication failed."
	result.AuthErrorCode = "535 5.7.8"
	result.AuthWait = "5"

	return false, result, nil
}

func handleSignal() {
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	go func() {

		<-signalChannel

		err := handleSignalParams.httpServer.Shutdown(context.Background())
		defer handleSignalParams.db.Close()

		if err != nil {
			log.Fatal().Err(err).Msgf("HTTP server Shutdown: %v", err)

		} else {
			log.Info().Msgf("HTTP server Shutdown complete")
		}

		log.Warn().Msg("SIGINT")
		os.Exit(0)

	}()
}

func handlerIndex(rw http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(rw, "%s v%s\n", html.EscapeString(ApplicationDescription), html.EscapeString(BuildVersion))
}

func handlerAuth(rw http.ResponseWriter, req *http.Request) {

	authMethod := req.Header.Get("Auth-Method")
	authUser := req.Header.Get("Auth-User")
	authPass := req.Header.Get("Auth-Pass")
	authProtocol := req.Header.Get("Auth-Protocol")
	authLoginAttempt := req.Header.Get("Auth-Login-Attempt")
	clientIP := req.Header.Get("Client-IP")
	clientHost := req.Header.Get("Client-Host")
	authSMTPHelo := req.Header.Get("Auth-SMTP-Helo")
	authSMTPFrom := req.Header.Get("Auth-SMTP-From")
	authSMTPTo := req.Header.Get("Auth-SMTP-To")

	log.Info().
		Str("authMethod", authMethod).
		Str("authUser", authUser).
		Str("authPass", authPass).
		Str("authProtocol", authProtocol).
		Str("authLoginAttempt", authLoginAttempt).
		Str("clientIP", clientIP).
		Str("clientHost", clientHost).
		Str("authSMTPHelo", authSMTPHelo).
		Str("authSMTPFrom", authSMTPFrom).
		Str("authSMTPTo", authSMTPTo).
		Str("event", "auth").
		Msgf("Incoming auth request")

	success, result, err := authenticate(authUser, authPass, authProtocol, authSMTPFrom, authSMTPTo)
	if err != nil {
		log.Error().
			Err(err).
			Str("event", "auth").
			Msgf("Can't authenticate %s:%s", authUser, authPass)
	}

	log.Info().
		Str("AuthStatus", result.AuthStatus).
		Str("AuthServer", result.AuthServer).
		Int("AuthPort", result.AuthPort).
		Str("AuthWait", result.AuthWait).
		Str("AuthErrorCode", result.AuthErrorCode).
		Str("event", "auth_result").
		Msgf("Received auth result for '%s':'%s'", authUser, authPass)

	rw.Header().Set("Auth-Status", result.AuthStatus)

	if success {
		log.Info().
			Str("event", "auth_ok").
			Msgf("Successfully authenticated '%s':'%s'", authUser, authPass)

		rw.Header().Set("Auth-Server", result.AuthServer)
		rw.Header().Set("Auth-Port", strconv.Itoa(result.AuthPort))

	} else {

		log.Info().
			Str("event", "auth_failed").
			Msgf("Access denied '%s':'%s'", authUser, authPass)

		rw.Header().Set("Auth-Wait", result.AuthWait)

		if result.AuthErrorCode != "" {
			rw.Header().Set("Auth-Error-Code", result.AuthErrorCode)
		}
	}

}

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Debug().Msg("Logger initialised")

	configPtr := flag.String("config", "nginx-mail-auth-http-server.conf", "Path to configuration file")
	showVersionPtr := flag.Bool("version", false, "Show version")

	ReadConfigurationFile(*configPtr, &Configuration)

	flag.Parse()

	if *showVersionPtr {
		fmt.Printf("%s\n", ApplicationDescription)
		fmt.Printf("Version: %s\n", BuildVersion)
		os.Exit(0)
	}

	mysqlConnectionURI := Configuration.Database.URI
	db, err := sqlx.Open("mysql", mysqlConnectionURI)
	if err != nil {
		log.Fatal().Msgf("Error while initialising db: %v", err)
	}

	DB = db
	handleSignalParams.db = DB

	if err := DB.Ping(); err != nil {
		log.Fatal().
			Err(err).
			Str("stage", "init").
			Msgf("Error while pinging db: %v", err)
	} else {
		log.Info().
			Str("stage", "init").
			Msg("Database ping ok")
	}

	handleSignal()
}

func main() {

	log.Info().Msg("Strating server...")

	if err := DB.Ping(); err != nil {
		log.Fatal().Err(err).Msgf("Error while pinging db: %v", err)
	} else {
		log.Info().Msg("Database ping ok")
	}

	srv := &http.Server{
		Addr:         Configuration.Listen,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	handleSignalParams.httpServer = *srv

	http.HandleFunc("/", handlerIndex)
	http.HandleFunc("/auth", handlerAuth)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal().Err(err).Msgf("HTTP server ListenAndServe: %v", err)
	}

}
