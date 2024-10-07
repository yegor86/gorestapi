package cmd

import (
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	cli "github.com/spf13/cobra"

	"github.com/snowzach/golib/version"
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")
}

var (
	Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true, // Enables logging the file and line number
	}))

	pidFile string
	cfgFile string

	// The Root Cli Handler
	rootCmd = &cli.Command{
		Version: version.GitVersion,
		Use:     version.Executable,
		PersistentPreRunE: func(cmd *cli.Command, args []string) error {

			// Parse defaults, config file and environment.
			conf, err := Load()
			if err != nil {
				Logger.Error(fmt.Sprintf("could not parse YAML config: %v", err))
				os.Exit(1)
			}

			// Load the metrics server
			if conf.Metrics.Enabled {
				hostPort := net.JoinHostPort(conf.Metrics.Host, strconv.Itoa(conf.Metrics.Port))
				r := http.NewServeMux()
				r.Handle("/metrics", promhttp.Handler())
				if conf.Profiler.Enabled {
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
			pidFile = conf.Profiler.Pidfile
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
