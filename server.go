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
	"sync/atomic"
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
var DebugMetricsNotifierPeriod time.Duration = 300

type DatabaseStruct struct {
	URI              string `json:"uri"`
	AuthLookupQuery  string `json:"auth_lookup_query"`
	RelayLookupQuery string `json:"relay_lookup_query"`
}

type ConfigurationStruct struct {
	Listen   string         `json:"listen"`
	Logfile  string         `json:"logfile"`
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
	AuthViaRelay  bool
	AuthViaLogin  bool
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

type MetricsStruct struct {
	AuthRequests             int32
	AuthRequestsFailed       int32
	AuthRequestsFailedRelay  int32
	AuthRequestsFailedLogin  int32
	AuthRequestsSuccess      int32
	AuthRequestsSuccessRelay int32
	AuthRequestsSuccessLogin int32
	AuthRequestsRelay        int32
	AuthRequestsLogin        int32
	InternalErrors           int32
}

var Configuration = ConfigurationStruct{}
var handleSignalParams = handleSignalParamsStruct{}
var flagParams = flagParamsStruct{}

var Metrics = MetricsStruct{
	AuthRequests:             0,
	AuthRequestsFailed:       0,
	AuthRequestsFailedRelay:  0,
	AuthRequestsFailedLogin:  0,
	AuthRequestsSuccess:      0,
	AuthRequestsSuccessRelay: 0,
	AuthRequestsSuccessLogin: 0,
	AuthRequestsRelay:        0,
	AuthRequestsLogin:        0,
	InternalErrors:           0,
}

func MetricsNotifier() {
	go func() {
		for {
			time.Sleep(DebugMetricsNotifierPeriod * time.Second)
			log.Debug().
				Int32("AuthRequests", Metrics.AuthRequests).
				Int32("AuthRequestsFailed", Metrics.AuthRequestsFailed).
				Int32("AuthRequestsFailedRelay", Metrics.AuthRequestsFailedRelay).
				Int32("AuthRequestsFailedLogin", Metrics.AuthRequestsFailedLogin).
				Int32("AuthRequestsSuccess", Metrics.AuthRequestsSuccess).
				Int32("AuthRequestsSuccessRelay", Metrics.AuthRequestsSuccessRelay).
				Int32("AuthRequestsSuccessLogin", Metrics.AuthRequestsSuccessLogin).
				Int32("AuthRequestsRelay", Metrics.AuthRequestsRelay).
				Int32("AuthRequestsLogin", Metrics.AuthRequestsLogin).
				Int32("InternalErrors", Metrics.InternalErrors).
				Msg("Metrics")
		}
	}()
}

func ReadConfigurationFile(configPtr string, configuration *ConfigurationStruct) {

	log.Debug().Msgf("Loading configuration file '%s'", configPtr)

	configFile, _ := os.Open(configPtr)
	defer configFile.Close()

	JSONDecoder := json.NewDecoder(configFile)

	err := JSONDecoder.Decode(&configuration)
	if err != nil {
		log.Error().
			Err(err).
			Str("stage", "init").
			Msgf("Error while loading configuration file '%s'", configPtr)
	}

	log.Debug().Msg("Finished loading configuration file")

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

	log.Info().
		Str("user", user).
		Str("pass", pass).
		Str("protocol", protocol).
		Str("mailFrom", mailFrom).
		Str("rcptTo", rcptTo).
		Msgf("Processing authentication request")

	if user == "" &&
		pass == "" &&
		mailFrom != "" &&
		rcptTo != "" {

		_ = atomic.AddInt32(&Metrics.AuthRequestsRelay, 1)

		result.AuthViaRelay = true

		log.Info().
			Str("protocol", protocol).
			Str("mailFrom", mailFrom).
			Str("rcptTo", rcptTo).
			Msg("Authenticating by relay access ('user' and 'pass' are empty)")

		mailHeaderRegex := regexp.MustCompile(`<(.*?)(\+.*?)?@(.*?)>`)
		mailFromEmailMatch := mailHeaderRegex.FindStringSubmatch(mailFrom)
		rcptToEmailMatch := mailHeaderRegex.FindStringSubmatch(rcptTo)

		if len(mailFromEmailMatch) == 4 {

			nonVERPAddress.WriteString(mailFromEmailMatch[1])
			nonVERPAddress.WriteString("@")
			nonVERPAddress.WriteString(mailFromEmailMatch[3])

			queryParams.MailFrom = nonVERPAddress.String()

			log.Debug().
				Str("queryParams.MailFrom", queryParams.MailFrom).
				Str("mailFrom", mailFrom).
				Msgf("Fetched an email address out of mailFrom header (VERP section stripped)")

			nonVERPAddress.Reset()

		} else {

			_ = atomic.AddInt32(&Metrics.InternalErrors, 1)

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

			log.Debug().
				Str("queryParams.RcptTo", queryParams.RcptTo).
				Str("rcptTo", rcptTo).
				Msgf("Fetched an email address out of rcptTo header (VERP section stripped)")

			nonVERPAddress.Reset()

		} else {

			_ = atomic.AddInt32(&Metrics.InternalErrors, 1)

			log.Error().
				Str("rcptTo", rcptTo).
				Int("rcptToEmailMatchLen", len(rcptToEmailMatch)).
				Msg("Can't parse RCPT TO command")

			result.AuthStatus = "Temporary server problem, try again later"
			result.AuthErrorCode = "451 4.3.0"
			result.AuthWait = "5"

			return false, result, err

		}

		log.Debug().
			Str("User", queryParams.User).
			Str("Pass", queryParams.Pass).
			Str("MailFrom", queryParams.MailFrom).
			Str("RcptTo", queryParams.RcptTo).
			Msg("Relay lookup query parameters prepared")

		query = Configuration.Database.RelayLookupQuery

	} else if user != "" &&
		pass != "" &&
		mailFrom != "" &&
		rcptTo != "" {

		_ = atomic.AddInt32(&Metrics.AuthRequestsLogin, 1)

		result.AuthViaRelay = true

		log.Info().
			Str("protocol", protocol).
			Str("user", user).
			Str("pass", pass).
			Msg("Authenticating by credentials")

		query = Configuration.Database.AuthLookupQuery

	} else {
		_ = atomic.AddInt32(&Metrics.InternalErrors, 1)

		log.Error().
			Str("User", queryParams.User).
			Str("Pass", queryParams.Pass).
			Str("MailFrom", queryParams.MailFrom).
			Str("RcptTo", queryParams.RcptTo).
			Msg("Can't authenticate via relay nor login")

		result.AuthStatus = "Temporary server problem, try again later"
		result.AuthErrorCode = "451 4.3.0"
		result.AuthWait = "5"

		return false, result, err

	}

	log.Debug().
		Str("query", query).
		Msg("Lookup query prepared")

	queryResult, err := DB.NamedQuery(query, queryParams)

	if err != nil {

		_ = atomic.AddInt32(&Metrics.InternalErrors, 1)

		log.Error().Err(err).Msgf("Error while executing query: %v", err)

		result.AuthStatus = "Temporary server problem, try again later"
		result.AuthErrorCode = "451 4.3.0"
		result.AuthWait = "5"

		return false, result, err
	}

	defer queryResult.Close()

	for queryResult.Next() {
		log.Debug().
			Str("protocol", protocol).
			Str("user", user).
			Str("pass", pass).
			Str("mailFrom", mailFrom).
			Str("rcptTo", rcptTo).
			Msgf("Found results after lookup query execution")

		var upstream = UpstreamStruct{}

		if err = queryResult.StructScan(&upstream); err != nil {

			_ = atomic.AddInt32(&Metrics.InternalErrors, 1)

			log.Error().Err(err).Msgf("Error while parsing lookup query result")

			result.AuthStatus = "Temporary server problem, try again later"
			result.AuthErrorCode = "451 4.3.0"
			result.AuthWait = "5"

			return false, result, err

		} else {
			log.Debug().Msgf("Lookup query results parsed successfully")
		}

		log.Info().
			Str("protocol", protocol).
			Str("user", user).
			Str("pass", pass).
			Str("mailFrom", mailFrom).
			Str("rcptTo", rcptTo).
			Str("upstreamAddress", upstream.Address).
			Int("upstreamPort", upstream.Port).
			Msgf("Found upstream")

		result.AuthStatus = "OK"
		result.AuthServer = upstream.Address
		result.AuthPort = upstream.Port

		return true, result, nil

	}

	log.Info().
		Str("protocol", protocol).
		Str("user", user).
		Str("pass", pass).
		Str("mailFrom", mailFrom).
		Str("rcptTo", rcptTo).
		Msgf("No results after lookup")

	result.AuthStatus = "Error: authentication failed."
	result.AuthErrorCode = "535 5.7.8"
	result.AuthWait = "5"

	return false, result, nil
}

func handleSignal() {

	log.Debug().Msg("Initialising signal handling function")

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

func handlerMetrics(rw http.ResponseWriter, req *http.Request) {

	fmt.Fprintf(rw, "# TYPE AuthRequests counter\n")
	fmt.Fprintf(rw, "AuthRequests{result=\"started\"} %v\n", Metrics.AuthRequests)
	fmt.Fprintf(rw, "AuthRequests{result=\"fail\"} %v\n", Metrics.AuthRequestsFailed)
	fmt.Fprintf(rw, "AuthRequests{result=\"success\"} %v\n", Metrics.AuthRequestsSuccess)
	fmt.Fprintf(rw, "AuthRequests{kind=\"relay\"} %v\n", Metrics.AuthRequestsRelay)
	fmt.Fprintf(rw, "AuthRequests{kind=\"login\"} %v\n", Metrics.AuthRequestsLogin)

	fmt.Fprintf(rw, "# TYPE InternalErrors counter\n")
	fmt.Fprintf(rw, "InternalErrors %v\n", Metrics.InternalErrors)

}

func handlerAuth(rw http.ResponseWriter, req *http.Request) {

	_ = atomic.AddInt32(&Metrics.AuthRequests, 1)

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

		_ = atomic.AddInt32(&Metrics.InternalErrors, 1)

		log.Error().
			Err(err).
			Str("event", "auth").
			Msgf("Can't authenticate %s:%s", authUser, authPass)
	}

	rw.Header().Set("Auth-Status", result.AuthStatus)

	log.Debug().
		Str("AuthStatus", result.AuthStatus).
		Str("AuthServer", result.AuthServer).
		Int("AuthPort", result.AuthPort).
		Str("AuthWait", result.AuthWait).
		Str("AuthErrorCode", result.AuthErrorCode).
		Bool("AuthViaRelay", result.AuthViaRelay).
		Bool("AuthViaLogin", result.AuthViaLogin).
		Msgf("Got result from authentication function")

	if success {

		log.Debug().
			Msgf("Registering successful authentication in metrics")

		_ = atomic.AddInt32(&Metrics.AuthRequestsSuccess, 1)

		if result.AuthViaRelay {
			_ = atomic.AddInt32(&Metrics.AuthRequestsSuccessRelay, 1)

		} else if result.AuthViaLogin {
			_ = atomic.AddInt32(&Metrics.AuthRequestsSuccessLogin, 1)
		}

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
			Str("AuthStatus", result.AuthStatus).
			Str("AuthServer", result.AuthServer).
			Int("AuthPort", result.AuthPort).
			Str("AuthWait", result.AuthWait).
			Str("AuthErrorCode", result.AuthErrorCode).
			Str("event", "auth_ok").
			Msgf("Successful authentication")

		rw.Header().Set("Auth-Server", result.AuthServer)
		rw.Header().Set("Auth-Port", strconv.Itoa(result.AuthPort))

	} else {

		_ = atomic.AddInt32(&Metrics.AuthRequestsFailed, 1)

		if result.AuthViaRelay {
			_ = atomic.AddInt32(&Metrics.AuthRequestsFailedRelay, 1)

		} else if result.AuthViaLogin {
			_ = atomic.AddInt32(&Metrics.AuthRequestsFailedLogin, 1)
		}

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
			Str("AuthStatus", result.AuthStatus).
			Str("AuthServer", result.AuthServer).
			Int("AuthPort", result.AuthPort).
			Str("AuthWait", result.AuthWait).
			Str("AuthErrorCode", result.AuthErrorCode).
			Str("event", "auth_failed").
			Msgf("Failed to authenticate")

		rw.Header().Set("Auth-Wait", result.AuthWait)

		if result.AuthErrorCode != "" {
			rw.Header().Set("Auth-Error-Code", result.AuthErrorCode)
		}
	}

}

func init() {

	configPtr := flag.String("config", "nginx-mail-auth-http-server.conf", "Path to configuration file")
	verbosePtr := flag.Bool("verbose", false, "Verbose output")
	showVersionPtr := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersionPtr {
		fmt.Printf("%s\n", ApplicationDescription)
		fmt.Printf("Version: %s\n", BuildVersion)
		os.Exit(0)
	}

	if *verbosePtr {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		MetricsNotifier()
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Debug().Msg("Logger initialised")

	ReadConfigurationFile(*configPtr, &Configuration)

	log.Debug().Msg("Initialising database connection")
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
	}

	log.Debug().Msg("Finished initialising database connection")

	handleSignal()
}

func main() {

	log.Info().Msgf("Strating server on %s", Configuration.Listen)

	if err := DB.Ping(); err != nil {
		log.Fatal().Err(err).Msgf("Error while pinging db: %v", err)
	}

	srv := &http.Server{
		Addr:         Configuration.Listen,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	handleSignalParams.httpServer = *srv

	http.HandleFunc("/", handlerIndex)
	http.HandleFunc("/auth", handlerAuth)
	http.HandleFunc("/metrics", handlerMetrics)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal().Err(err).Msgf("HTTP server ListenAndServe: %v", err)
	}

}
