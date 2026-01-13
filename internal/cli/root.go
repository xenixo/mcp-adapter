// Package cli provides the command-line interface for mcp-adapter.
package cli

import (
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/xenixo/mcp-adapter/internal/config"
)

var (
	// Version is set at build time.
	Version = "dev"
	// Commit is set at build time.
	Commit = "unknown"
	// BuildDate is set at build time.
	BuildDate = "unknown"
)

// App holds the application state for the CLI.
type App struct {
	Config   *config.Config
	Logger   *zap.Logger
	Verbose  bool
	LogLevel string
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{
		Config: config.Default(),
	}
}

// InitLogger initializes the logger based on configuration.
func (a *App) InitLogger() error {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(a.LogLevel)); err != nil {
		level = zapcore.InfoLevel
	}

	if a.Verbose {
		level = zapcore.DebugLevel
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{"stderr"}

	logger, err := cfg.Build()
	if err != nil {
		return err
	}

	a.Logger = logger
	return nil
}

// NewRootCmd creates the root command.
func NewRootCmd() *cobra.Command {
	app := NewApp()

	rootCmd := &cobra.Command{
		Use:   "mcp-adapter",
		Short: "MCP adapter for discovering, installing, and running MCP servers",
		Long: `mcp-adapter is a command-line tool for managing Model Context Protocol (MCP) servers.

It provides a unified interface for discovering, installing, and running MCP servers
written in different languages (Node.js, Python, binary) through a single CLI.

Features:
  - Discover available MCP servers from manifests
  - Install servers using npm, pip, or binary downloads
  - Launch servers with proper runtime detection
  - Support for stdio and HTTP transports`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return app.InitLogger()
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if app.Logger != nil {
				app.Logger.Sync()
			}
		},
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&app.Verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&app.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&app.Config.BaseDir, "config-dir", app.Config.BaseDir, "Configuration directory")

	// Add subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newListCmd(app))
	rootCmd.AddCommand(newInstallCmd(app))
	rootCmd.AddCommand(newRunCmd(app))
	rootCmd.AddCommand(newDoctorCmd(app))
	rootCmd.AddCommand(newUninstallCmd(app))
	rootCmd.AddCommand(newRegistryCmd(app))

	return rootCmd
}

// Execute runs the CLI.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
