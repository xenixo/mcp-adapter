package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/xenixo/mcp-adapter/internal/installer"
	"github.com/xenixo/mcp-adapter/internal/launcher"
	"github.com/xenixo/mcp-adapter/internal/manifest"
	"github.com/xenixo/mcp-adapter/internal/registry"
	"github.com/xenixo/mcp-adapter/internal/runtime"
	"github.com/xenixo/mcp-adapter/manifests"
)

func newRunCmd(app *App) *cobra.Command {
	var (
		args    []string
		envVars []string
		stdio   bool
	)

	cmd := &cobra.Command{
		Use:   "run <server> [-- args...]",
		Short: "Run an MCP server",
		Long: `Run an installed MCP server.

This command launches the specified MCP server and manages its lifecycle.
For stdio transport, stdin/stdout are connected to the server for MCP
communication.

The server must be installed before running. Use 'mcp-adapter install' first.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			serverName := cmdArgs[0]
			if len(cmdArgs) > 1 {
				args = append(args, cmdArgs[1:]...)
			}
			return runServer(app, serverName, args, envVars, stdio)
		},
	}

	cmd.Flags().StringArrayVarP(&args, "arg", "a", nil, "Additional arguments to pass to the server")
	cmd.Flags().StringArrayVarP(&envVars, "env", "e", nil, "Environment variables (KEY=VALUE)")
	cmd.Flags().BoolVar(&stdio, "stdio", true, "Connect stdio for MCP communication")

	return cmd
}

func runServer(app *App, serverName string, args, envVars []string, stdio bool) error {
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

	// Check if installed
	installDir := app.Config.ServerInstallPath(serverName)
	if !installer.IsInstalled(installDir) {
		return fmt.Errorf("server %q is not installed; run 'mcp-adapter install %s' first", serverName, serverName)
	}

	// Validate runtime
	detector := runtime.NewDetector()
	rt, err := detector.DetectForServer(server)
	if err != nil {
		return fmt.Errorf("runtime not available: %w", err)
	}

	// Check version requirements
	var versionReq string
	switch server.Type {
	case manifest.ServerTypeNode:
		versionReq = server.Runtime.Node
	case manifest.ServerTypePython:
		versionReq = server.Runtime.Python
	}

	if versionReq != "" && !runtime.MeetsRequirement(rt.Version, versionReq) {
		return fmt.Errorf("runtime version %s does not meet requirement %s", rt.Version, versionReq)
	}

	// Parse environment variables
	env := make(map[string]string)
	for _, e := range envVars {
		for i, c := range e {
			if c == '=' {
				env[e[:i]] = e[i+1:]
				break
			}
		}
	}

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create launcher
	launcherInst := launcher.NewLauncher(app.Config, app.Logger)

	// Configure launch options
	opts := &launcher.LaunchOptions{
		Args: args,
		Env:  env,
	}

	// For stdio transport, connect stdin/stdout directly
	if stdio && server.Transport == manifest.TransportStdio {
		opts.Stdin = os.Stdin
		opts.Stdout = os.Stdout
		opts.Stderr = os.Stderr
	}

	// Launch server
	app.Logger.Info("launching server",
		zap.String("server", serverName),
		zap.String("runtime", rt.Name),
		zap.String("runtimeVersion", rt.Version),
	)

	proc, err := launcherInst.Launch(ctx, server, opts)
	if err != nil {
		return fmt.Errorf("failed to launch server: %w", err)
	}

	// For non-stdio, stream output
	if !stdio || server.Transport != manifest.TransportStdio {
		if proc.Stdout != nil {
			go io.Copy(os.Stdout, proc.Stdout)
		}
		if proc.Stderr != nil {
			go io.Copy(os.Stderr, proc.Stderr)
		}
	}

	// Wait for signal or process exit
	go func() {
		sig := <-sigChan
		app.Logger.Info("received signal", zap.String("signal", sig.String()))
		launcherInst.Stop(serverName, 10*time.Second)
		cancel()
	}()

	// Wait for process to exit
	<-ctx.Done()

	// Get final state
	if p, ok := launcherInst.Get(serverName); ok {
		if p.Error != nil {
			return fmt.Errorf("server exited with error: %w", p.Error)
		}
		if p.ExitCode != 0 {
			return fmt.Errorf("server exited with code %d", p.ExitCode)
		}
	}

	return nil
}
