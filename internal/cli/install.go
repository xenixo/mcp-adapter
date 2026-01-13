package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/xenixo/mcp-adapter/internal/installer"
	"github.com/xenixo/mcp-adapter/internal/registry"
	"github.com/xenixo/mcp-adapter/manifests"
)

func newInstallCmd(app *App) *cobra.Command {
	var (
		force   bool
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "install <server>",
		Short: "Install an MCP server",
		Long: `Install an MCP server from the registry.

This command downloads and installs the specified MCP server using the
appropriate package manager (npm for Node.js, pip for Python, or direct
download for binaries).

Servers are installed to ~/.mcp-adapter/servers/<server-name>/`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(app, args[0], force, timeout)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force reinstall even if already installed")
	cmd.Flags().DurationVarP(&timeout, "timeout", "t", 10*time.Minute, "Installation timeout")

	return cmd
}

func runInstall(app *App, serverName string, force bool, timeout time.Duration) error {
	// Ensure directories exist
	if err := app.Config.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Load registry
	reg := registry.New()

	if err := reg.LoadFromEmbed(manifests.FS, "*.yaml"); err != nil {
		return fmt.Errorf("failed to load embedded manifests: %w", err)
	}

	userManifestsDir := filepath.Join(app.Config.BaseDir, "manifests")
	if err := reg.LoadFromDirectory(userManifestsDir); err != nil {
		app.Logger.Debug("user manifests not loaded", zap.Error(err))
	}

	// Find server
	server, ok := reg.Get(serverName)
	if !ok {
		return fmt.Errorf("server %q not found in registry", serverName)
	}

	installDir := app.Config.ServerInstallPath(serverName)

	// Check if already installed
	if installer.IsInstalled(installDir) && !force {
		fmt.Printf("Server %q is already installed at %s\n", serverName, installDir)
		fmt.Println("Use --force to reinstall.")
		return nil
	}

	// Remove existing installation if force
	if force && installer.IsInstalled(installDir) {
		app.Logger.Info("removing existing installation", zap.String("path", installDir))
		if err := os.RemoveAll(installDir); err != nil {
			return fmt.Errorf("failed to remove existing installation: %w", err)
		}
	}

	// Create context with timeout and signal handling
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nInstallation cancelled.")
		cancel()
	}()

	// Install
	fmt.Printf("Installing %s (%s) version %s...\n", serverName, server.Type, server.Source.Version)

	mgr := installer.NewManager()
	result, err := mgr.Install(ctx, server, installDir)
	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("installation failed: %v", result.Error)
	}

	fmt.Printf("✓ Successfully installed %s\n", serverName)
	fmt.Printf("  Location: %s\n", result.InstallPath)
	fmt.Printf("  Entrypoint: %s\n", result.Entrypoint)

	return nil
}
