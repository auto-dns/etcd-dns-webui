package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/auto-dns/etcd-dns-webui/internal/app"
	"github.com/auto-dns/etcd-dns-webui/internal/config"
	"github.com/auto-dns/etcd-dns-webui/internal/logger"
)

type contextKey string

const configKey = contextKey("config")

var rootCmd = &cobra.Command{
	Use:   "etcd-dns-webui",
	Short: "A web app server for viewing CoreDNS records stored in etcd",
	Long:  "A server for a web app used to view CoreDNS records stored in an etcd datastore",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		ctx := context.WithValue(cmd.Context(), configKey, cfg)
		cmd.SetContext(ctx)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration.
		cfg := cmd.Context().Value(configKey).(*config.Config)

		// Set up logger.
		logInstance := logger.SetupLogger(&cfg.Log)

		// Create the application.
		application, err := app.New(cfg, logInstance)
		if err != nil {
			return fmt.Errorf("failed to create app: %w", err)
		}

		// Create a context with cancellation for graceful shutdown.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Listen for OS signals.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigCh
			logInstance.Info().Msgf("Received signal: %v", sig)
			cancel()
		}()

		// Run the application. When context is canceled, Run returns.
		if err := application.Run(ctx); err != nil {
			return fmt.Errorf("app run error: %w", err)
		}
		return nil
	},
}

func init() {
	// Persistent config file override
	rootCmd.PersistentFlags().String("config", "", "Path to config file (e.g. ./config.yaml)")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	// Etcd Flags
	rootCmd.PersistentFlags().String("etcd.host", "", "etcd host to connect to")
	viper.BindPFlag("etcd.host", rootCmd.PersistentFlags().Lookup("etcd.host"))

	rootCmd.PersistentFlags().Int("etcd.port", 0, "etcd port")
	viper.BindPFlag("etcd.port", rootCmd.PersistentFlags().Lookup("etcd.port"))

	rootCmd.PersistentFlags().String("etcd.path-prefix", "", "etcd key path prefix (e.g., /skydns)")
	viper.BindPFlag("etcd.path_prefix", rootCmd.PersistentFlags().Lookup("etcd.path-prefix"))

	// Log Flags
	rootCmd.PersistentFlags().String("log.level", "", "Log level (e.g., TRACE, DEBUG, INFO, WARN, ERROR, FATAL)")
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log.level"))

	// Server Flags
	rootCmd.PersistentFlags().Int("server.port", 0, "the server port (e.g. 8080)")
	viper.BindPFlag("server.port", rootCmd.PersistentFlags().Lookup("server.port"))

	rootCmd.PersistentFlags().Bool("server.proxy.enable", false, "enable webui proxying to a Vite server for local development")
	viper.BindPFlag("server.proxy.enable", rootCmd.PersistentFlags().Lookup("server.proxy.enable"))

	rootCmd.PersistentFlags().String("server.proxy.hostname", "", "the vite web server hostname (e.g. localhost)")
	viper.BindPFlag("server.proxy.hostname", rootCmd.PersistentFlags().Lookup("server.proxy.hostname"))

	rootCmd.PersistentFlags().Int("server.proxy.port", 0, "the vite web server port (e.g. 5173)")
	viper.BindPFlag("server.proxy.port", rootCmd.PersistentFlags().Lookup("server.proxy.port"))
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", err)
		os.Exit(1)
	}
}
