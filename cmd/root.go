package cmd

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	cli "github.com/spf13/cobra"
)

func init() {
	// Parse defaults, config file and environment.
	_, _, err := Load()
	if err != nil {
		Logger.Error(fmt.Sprintf("could not parse YAML config: %v", err))
		os.Exit(1)
	}
}

var (
	Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true, // Enables logging the file and line number
	}))

	pidFile string

	// The Root Cli Handler
	rootCmd = &cli.Command{
		Version: GitVersion,
		Use:     Executable,
		PersistentPreRunE: func(cmd *cli.Command, args []string) error {

			// Load the metrics server
			if config.Metrics.Enabled {
				hostPort := net.JoinHostPort(config.Metrics.Host, strconv.Itoa(config.Metrics.Port))
				r := http.NewServeMux()
				r.Handle("/metrics", promhttp.Handler())
				if config.Profiler.Enabled {
					r.HandleFunc("/debug/pprof/", pprof.Index)
					r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
					r.HandleFunc("/debug/pprof/profile", pprof.Profile)
					r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
					r.HandleFunc("/debug/pprof/trace", pprof.Trace)
					Logger.Info("Profiler enabled", "profiler_path", fmt.Sprintf("http://%s/debug/pprof/", hostPort))
				}
				go func() {
					if err := http.ListenAndServe(hostPort, r); err != nil {
						Logger.Error(fmt.Sprintf("Metrics server error: %v", err))
					}
				}()
				Logger.Info("Metrics enabled", "address", hostPort)
			}

			// Create Pid File
			pidFile = config.Profiler.Pidfile
			if pidFile != "" {
				file, err := os.OpenFile(pidFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
				if err != nil {
					return fmt.Errorf("could not create pid file: %s error:%v", pidFile, err)
				}
				defer file.Close()
				_, err = fmt.Fprintf(file, "%d\n", os.Getpid())
				if err != nil {
					return fmt.Errorf("could not create pid file: %s error:%v", pidFile, err)
				}
			}
			return nil
		},
		PersistentPostRun: func(cmd *cli.Command, args []string) {
			// Remove Pid file
			if pidFile != "" {
				os.Remove(pidFile)
			}
		},
	}
)

// Execute starts the program
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}
