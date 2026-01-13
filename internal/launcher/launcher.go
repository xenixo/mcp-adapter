// Package launcher provides MCP server process lifecycle management.
package launcher

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/xenixo/mcp-adapter/internal/config"
	"github.com/xenixo/mcp-adapter/internal/manifest"
	"github.com/xenixo/mcp-adapter/internal/runtime"
)

// State represents the state of a launched server.
type State int

const (
	StateUnknown State = iota
	StateStarting
	StateRunning
	StateStopping
	StateStopped
	StateFailed
)

func (s State) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Process represents a launched MCP server process.
type Process struct {
	Server     *manifest.Server
	Cmd        *exec.Cmd
	State      State
	StartTime  time.Time
	StopTime   time.Time
	ExitCode   int
	Error      error
	Stdin      io.WriteCloser
	Stdout     io.ReadCloser
	Stderr     io.ReadCloser
	cancelFunc context.CancelFunc
	mu         sync.RWMutex
}

// Launcher handles MCP server process lifecycle.
type Launcher struct {
	cfg      *config.Config
	detector *runtime.Detector
	logger   *zap.Logger
	mu       sync.RWMutex
	procs    map[string]*Process
}

// NewLauncher creates a new launcher.
func NewLauncher(cfg *config.Config, logger *zap.Logger) *Launcher {
	return &Launcher{
		cfg:      cfg,
		detector: runtime.NewDetector(),
		logger:   logger,
		procs:    make(map[string]*Process),
	}
}

// LaunchOptions configures how a server is launched.
type LaunchOptions struct {
	// Args are additional arguments to pass to the server.
	Args []string

	// Env are additional environment variables.
	Env map[string]string

	// WorkDir is the working directory for the server.
	WorkDir string

	// Stdin is the reader for stdin (for stdio transport).
	Stdin io.Reader

	// Stdout is the writer for stdout.
	Stdout io.Writer

	// Stderr is the writer for stderr.
	Stderr io.Writer
}

// Launch starts an MCP server process.
func (l *Launcher) Launch(ctx context.Context, server *manifest.Server, opts *LaunchOptions) (*Process, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if already running
	if proc, ok := l.procs[server.Name]; ok {
		if proc.State == StateRunning {
			return nil, fmt.Errorf("server %q is already running", server.Name)
		}
	}

	// Validate runtime
	rt, err := l.detector.DetectForServer(server)
	if err != nil {
		return nil, fmt.Errorf("runtime detection failed: %w", err)
	}

	// Check runtime version requirement
	var versionReq string
	switch server.Type {
	case manifest.ServerTypeNode:
		versionReq = server.Runtime.Node
	case manifest.ServerTypePython:
		versionReq = server.Runtime.Python
	}

	if versionReq != "" && !runtime.MeetsRequirement(rt.Version, versionReq) {
		return nil, fmt.Errorf("runtime version %s does not meet requirement %s", rt.Version, versionReq)
	}

	// Determine entrypoint
	installDir := l.cfg.ServerInstallPath(server.Name)
	entrypoint, err := l.resolveEntrypoint(server, installDir)
	if err != nil {
		return nil, err
	}

	// Build command
	cmdCtx, cancel := context.WithCancel(ctx)
	args := append(server.Args, opts.Args...)

	var cmd *exec.Cmd
	switch server.Type {
	case manifest.ServerTypeNode:
		// For node, run with node if entrypoint is a .js file
		if filepath.Ext(entrypoint) == ".js" {
			allArgs := append([]string{entrypoint}, args...)
			cmd = exec.CommandContext(cmdCtx, rt.Path, allArgs...)
		} else {
			cmd = exec.CommandContext(cmdCtx, entrypoint, args...)
		}
	case manifest.ServerTypePython:
		cmd = exec.CommandContext(cmdCtx, entrypoint, args...)
	case manifest.ServerTypeBinary:
		cmd = exec.CommandContext(cmdCtx, entrypoint, args...)
	}

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range server.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if opts.Env != nil {
		for k, v := range opts.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Set working directory
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	} else {
		cmd.Dir = installDir
	}

	// Set up I/O
	proc := &Process{
		Server:     server,
		Cmd:        cmd,
		State:      StateStarting,
		cancelFunc: cancel,
	}

	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	} else {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
		}
		proc.Stdin = stdin
	}

	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	} else {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
		}
		proc.Stdout = stdout
	}

	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	} else {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
		}
		proc.Stderr = stderr
	}

	// Start process
	l.logger.Info("launching server",
		zap.String("server", server.Name),
		zap.String("entrypoint", entrypoint),
		zap.Strings("args", args),
	)

	if err := cmd.Start(); err != nil {
		cancel()
		proc.State = StateFailed
		proc.Error = err
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	proc.StartTime = time.Now()
	proc.State = StateRunning
	l.procs[server.Name] = proc

	// Monitor process in background
	go l.monitorProcess(proc)

	return proc, nil
}

// Stop stops a running MCP server.
func (l *Launcher) Stop(serverName string, timeout time.Duration) error {
	l.mu.Lock()
	proc, ok := l.procs[serverName]
	l.mu.Unlock()

	if !ok {
		return fmt.Errorf("server %q is not running", serverName)
	}

	proc.mu.Lock()
	if proc.State != StateRunning {
		proc.mu.Unlock()
		return fmt.Errorf("server %q is not running (state: %s)", serverName, proc.State)
	}
	proc.State = StateStopping
	proc.mu.Unlock()

	// Send SIGTERM
	if proc.Cmd.Process != nil {
		proc.Cmd.Process.Signal(syscall.SIGTERM)
	}

	// Wait for graceful shutdown
	done := make(chan struct{})
	go func() {
		proc.Cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Graceful shutdown
	case <-time.After(timeout):
		// Force kill
		if proc.Cmd.Process != nil {
			proc.Cmd.Process.Kill()
		}
	}

	proc.mu.Lock()
	proc.StopTime = time.Now()
	proc.State = StateStopped
	proc.mu.Unlock()

	return nil
}

// Get returns a running process by server name.
func (l *Launcher) Get(serverName string) (*Process, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	proc, ok := l.procs[serverName]
	return proc, ok
}

// ListRunning returns all running processes.
func (l *Launcher) ListRunning() []*Process {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var result []*Process
	for _, proc := range l.procs {
		if proc.State == StateRunning {
			result = append(result, proc)
		}
	}
	return result
}

// StopAll stops all running servers.
func (l *Launcher) StopAll(timeout time.Duration) {
	l.mu.RLock()
	names := make([]string, 0, len(l.procs))
	for name := range l.procs {
		names = append(names, name)
	}
	l.mu.RUnlock()

	for _, name := range names {
		l.Stop(name, timeout)
	}
}

func (l *Launcher) monitorProcess(proc *Process) {
	err := proc.Cmd.Wait()

	proc.mu.Lock()
	defer proc.mu.Unlock()

	proc.StopTime = time.Now()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			proc.ExitCode = exitErr.ExitCode()
		}
		proc.Error = err
		if proc.State != StateStopping && proc.State != StateStopped {
			proc.State = StateFailed
		} else {
			proc.State = StateStopped
		}
	} else {
		proc.ExitCode = 0
		proc.State = StateStopped
	}

	l.logger.Info("server stopped",
		zap.String("server", proc.Server.Name),
		zap.Int("exitCode", proc.ExitCode),
		zap.Duration("runtime", proc.StopTime.Sub(proc.StartTime)),
	)
}

func (l *Launcher) resolveEntrypoint(server *manifest.Server, installDir string) (string, error) {
	switch server.Type {
	case manifest.ServerTypeNode:
		// Check node_modules/.bin first
		binPath := filepath.Join(installDir, "node_modules", ".bin", server.Entrypoint)
		if _, err := os.Stat(binPath); err == nil {
			return binPath, nil
		}

		// Check for .js file in package
		parts := []string{installDir, "node_modules"}
		if server.Source.NPM != "" {
			parts = append(parts, server.Source.NPM)
		}
		parts = append(parts, "dist", server.Entrypoint+".js")
		jsPath := filepath.Join(parts...)
		if _, err := os.Stat(jsPath); err == nil {
			return jsPath, nil
		}

		return "", fmt.Errorf("entrypoint not found for server %q", server.Name)

	case manifest.ServerTypePython:
		binPath := filepath.Join(installDir, "venv", "bin", server.Entrypoint)
		if _, err := os.Stat(binPath); err == nil {
			return binPath, nil
		}
		return "", fmt.Errorf("entrypoint not found for server %q", server.Name)

	case manifest.ServerTypeBinary:
		binPath := filepath.Join(installDir, server.Entrypoint)
		if _, err := os.Stat(binPath); err == nil {
			return binPath, nil
		}
		return "", fmt.Errorf("entrypoint not found for server %q", server.Name)

	default:
		return "", fmt.Errorf("unknown server type: %s", server.Type)
	}
}
