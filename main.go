package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/jmoiron/sqlx"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/sony/sonyflake"

	"nginx_auth_server/server/configuration"
	"nginx_auth_server/server/handlers"
	"nginx_auth_server/server/lookup"
	"nginx_auth_server/server/metrics"
)

var ApplicationDescription = "Nginx Mail Auth HTTP Server"
var BuildVersion = "0.0.0"
var ShowSecretsInLog = false

var Debug bool = false
var DebugMetricsNotifierPeriod time.Duration = 60
var ListLengthWatcherPeriod time.Duration = 60

type handleSignalParamsStruct struct {
	httpServer http.Server
	db         *sqlx.DB
}

var handleSignalParams = handleSignalParamsStruct{}

var MetricsNotifierPeriod int = 60

// var Metrics = metrics.MetricsStruct{
// 	AuthRequests:             0,
// 	AuthRequestsFailed:       0,
// 	AuthRequestsFailedRelay:  0,
// 	AuthRequestsFailedLogin:  0,
// 	AuthRequestsSuccess:      0,
// 	AuthRequestsSuccessRelay: 0,
// 	AuthRequestsSuccessLogin: 0,
// 	AuthRequestsRelay:        0,
// 	AuthRequestsLogin:        0,
// 	InternalErrors:           0,

// 	Entities: make(map[string]int32),
// }

var Lookup = lookup.LookupStruct{
	ShowSecretsInLog: Debug,
}

var ctx = context.Background()
var flake = sonyflake.NewSonyflake(sonyflake.Settings{})
var DB *sqlx.DB

var httpHandlers = handlers.Handlers{
	ApplicationDescription: ApplicationDescription,
	BuildVersion:           BuildVersion,
	DB:                     DB,
	Flake:                  flake,
	MetricsCounter:         &metrics.Metrics,
	Lookup:                 &Lookup,
}

func MetricsNotifier() {
	go func() {
		for {
			time.Sleep(DebugMetricsNotifierPeriod * time.Second)
			log.Debug().
				Int32("AuthRequests", metrics.Metrics.AuthRequests).
				Int32("AuthRequestsFailed", metrics.Metrics.AuthRequestsFailed).
				Int32("AuthRequestsFailedRelay", metrics.Metrics.AuthRequestsFailedRelay).
				Int32("AuthRequestsFailedLogin", metrics.Metrics.AuthRequestsFailedLogin).
				Int32("AuthRequestsSuccess", metrics.Metrics.AuthRequestsSuccess).
				Int32("AuthRequestsSuccessRelay", metrics.Metrics.AuthRequestsSuccessRelay).
				Int32("AuthRequestsSuccessLogin", metrics.Metrics.AuthRequestsSuccessLogin).
				Int32("AuthRequestsRelay", metrics.Metrics.AuthRequestsRelay).
				Int32("AuthRequestsLogin", metrics.Metrics.AuthRequestsLogin).
				Int32("InternalErrors", metrics.Metrics.InternalErrors).
				Msg("Metrics")
		}
	}()
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
			log.Error().Err(err).Msgf("HTTP server Shutdown: %v", err)

		} else {
			log.Info().Msgf("HTTP server Shutdown complete")
		}

		log.Warn().Msg("SIGINT")
		os.Exit(0)

	}()
}

func init() {

	configPtr := flag.String("config", "nginx-mail-auth-http-server.conf", "Path to configuration file")
	verbosePtr := flag.Bool("verbose", false, "Verbose output")
	logSecretsPtr := flag.Bool("log-secrets", false, "Show plaintext passwords in logs")
	showVersionPtr := flag.Bool("version", false, "Show version")

	flag.Parse()

	if *showVersionPtr {
		fmt.Printf("%s\n", ApplicationDescription)
		fmt.Printf("Version: %s\n", BuildVersion)
		os.Exit(0)
	}

	if *logSecretsPtr {
		lookup.ShowSecretsInLog = true
	}

	if *verbosePtr {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		MetricsNotifier()
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Debug().Msg("Logger initialised")

	configuration.ReadConfigurationFile(*configPtr, &configuration.Configuration)

	listen_address, err := net.ResolveTCPAddr("tcp4", configuration.Configuration.ListenAddress)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error while resolving listen address")
	}

	configuration.Configuration.ListenNetTCPAddr = listen_address

	configuration.Configuration.ApplicationDescription = ApplicationDescription
	configuration.Configuration.BuildVersion = BuildVersion

	handleSignal()

	log.Debug().Msg("Initialising database connection")
	mysqlConnectionURI := configuration.Configuration.Database.URI
	db, err := sqlx.Open(configuration.Configuration.Database.Driver, mysqlConnectionURI)
	if err != nil {
		log.Fatal().Msgf("Error while initialising db: %v", err)
	}

	DB = db
	handleSignalParams.db = DB
	httpHandlers.DB = DB
	configuration.Configuration.DB = DB

	if err := DB.Ping(); err != nil {
		log.Fatal().
			Err(err).
			Str("stage", "init").
			Msgf("Error while pinging db: %v", err)
	}

	log.Debug().Msg("Finished initialising database connection")

}

func main() {

	log.Info().Msgf("Listening on %s", configuration.Configuration.ListenNetTCPAddr.String())

	srv := &http.Server{
		Addr:         configuration.Configuration.ListenNetTCPAddr.String(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	handleSignalParams.httpServer = *srv

	http.HandleFunc("/", httpHandlers.Index)
	http.HandleFunc("/auth", httpHandlers.Auth)
	http.HandleFunc("/metrics", httpHandlers.Metrics)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal().Err(err).Msgf("HTTP server ListenAndServe error")
	}

}
