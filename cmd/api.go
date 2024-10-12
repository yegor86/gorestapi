package cmd

import (
	"os"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	cli "github.com/spf13/cobra"
	"github.com/jmoiron/sqlx"

	"github.com/snowzach/golib/httpserver"
	"github.com/snowzach/golib/httpserver/metrics"
	"github.com/snowzach/golib/log"
	"github.com/snowzach/golib/signal"
	"github.com/snowzach/golib/version"
	"github.com/snowzach/gorestapi/embed"
	// "github.com/snowzach/gorestapi/gorestapi/mainrpc"
	"github.com/snowzach/golib/store/driver/postgres"
)

func init() {
	// Parse defaults, config file and environment.
	_, _, err := Load()
	if err != nil {
		Logger.Error(fmt.Sprintf("could not parse YAML config: %v", err))
		os.Exit(1)
	}
	rootCmd.AddCommand(apiCmd)
}

var (
	apiCmd = &cli.Command{
		Use:   "api",
		Short: "Start API",
		Long:  `Start API`,
		Run: func(cmd *cli.Command, args []string) { // Initialize the databse

			var err error

			// Create the router and server config
			router, err := newRouter()
			if err != nil {
				log.Fatalf("router config error: %v", err)
			}

			// Create the database
			// db, err := newDatabase()
			_, err = newDatabase()
			if err != nil {
				log.Fatalf("database config error: %v", err)
			}

			// Version endpoint
			router.Get("/version", version.GetVersion())

			// MainRPC
			// if err = mainrpc.Setup(router, db); err != nil {
			// 	log.Fatalf("Could not setup mainrpc: %v", err)
			// }

			// Serve embedded public html
			htmlFilesFS := embed.PublicHTMLFS()
			htmlFilesServer := http.FileServer(http.FS(htmlFilesFS))
			// Serve swagger docs
			router.Mount("/api-docs", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Vary", "Accept-Encoding")
				w.Header().Set("Cache-Control", "no-cache")
				htmlFilesServer.ServeHTTP(w, r)
			}))
			// Serve embedded webapp
			router.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// See if the file exists
				file, err := htmlFilesFS.Open(strings.TrimLeft(r.URL.Path, "/"))
				if err != nil {
					// If the file is not found, serve the root index.html file
					r.URL.Path = "/"
				} else {
					file.Close()
				}
				w.Header().Set("Vary", "Accept-Encoding")
				w.Header().Set("Cache-Control", "no-cache")
				htmlFilesServer.ServeHTTP(w, r)
			}))

			// Create a server
			s, err := newServer(router)
			if err != nil {
				log.Fatalf("could not create server error: %v", err)
			}

			// Start the listener and service connections.
			go func() {
				if err = s.ListenAndServe(); err != nil {
					log.Errorf("Server error: %v", err)
					signal.Stop.Stop()
				}
			}()
			log.Infof("API listening on %s", s.Addr)

			// Register signal handler and wait
			signal.Stop.OnSignal(signal.DefaultStopSignals...)
			<-signal.Stop.Chan() // Wait until Stop
			signal.Stop.Wait()   // Wait until everyone cleans up
		},
	}
)

func newRouter() (chi.Router, error) {

	router := chi.NewRouter()
	router.Use(
		middleware.Recoverer, // Recover from panics
		middleware.RequestID, // Inject request-id
	)

	// Request logger
	if config.Server.Log.Enabled {
		// router.Use(logger.LoggerStandardMiddleware(log.Logger.With("context", "server"), loggerConfig))
	}

	// CORS handler
	if config.Server.CORS.Enabled {
		var corsOptions cors.Options
		if err := koanfConfig.Unmarshal("server.cors", &corsOptions); err != nil {
			return nil, fmt.Errorf("could not parser server.cors config: %w", err)
		}
		router.Use(cors.New(corsOptions).Handler)
	}

	// If we have server metrics enabled, enable the middleware to collect them on the server.
	if config.Server.Metrics.Enabled {
		var metricsConfig metrics.Config
		if err := koanfConfig.Unmarshal("server.metrics", &metricsConfig); err != nil {
			return nil, fmt.Errorf("could not parser server.metrics config: %w", err)
		}
		router.Use(metrics.MetricsMiddleware(metricsConfig))
	}

	return router, nil
}

func newServer(handler http.Handler) (*httpserver.Server, error) {

	// Parse the config
	var serverConfig = &httpserver.Config{Handler: handler}
	if err := koanfConfig.Unmarshal("server", serverConfig); err != nil {
		return nil, fmt.Errorf("could not parse server config: %w", err)
	}

	// Create the server
	s, err := httpserver.New(httpserver.WithConfig(serverConfig))
	if err != nil {
		return nil, fmt.Errorf("could not create server: %w", err)
	}

	return s, nil

}

func newDatabase() (*sqlx.DB, error) {

	var err error

	// Database config
	var postgresConfig = &postgres.Config{}
	if err = koanfConfig.Unmarshal("database", postgresConfig); err != nil {
		return nil, fmt.Errorf("could not parse database config: %w", err)
	}

	// Loggers
	postgresConfig.Logger = log.NewWrapper(log.Logger.With("context", "database.postgres"), slog.LevelInfo)
	if config.Database.LogQueries {
		postgresConfig.QueryLogger = log.NewWrapper(log.Logger.With("context", "database.postgres.query"), slog.LevelDebug)
	}

	// Migrations
	postgresConfig.MigrationSource, err = embed.MigrationSource()
	if err != nil {
		return nil, fmt.Errorf("could not get database migrations error: %w", err)
	}

	// Create database
	db, err := postgres.New(postgresConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create database client: %w", err)
	}

	return db, nil

}
