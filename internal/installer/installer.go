// Package installer provides MCP server installation logic.
package installer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/xenixo/mcp-adapter/internal/manifest"
	"github.com/xenixo/mcp-adapter/internal/security"
)

// Result represents the result of an installation.
type Result struct {
	ServerName  string
	InstallPath string
	Entrypoint  string
	Success     bool
	Error       error
}

// Installer defines the interface for installers.
type Installer interface {
	Install(ctx context.Context, server *manifest.Server, installDir string) (*Result, error)
	Name() string
}

// Manager manages installers for different server types.
type Manager struct {
	installers map[manifest.ServerType]Installer
	validator  *security.Validator
	verifier   *security.Verifier
}

// NewManager creates a new installer manager.
func NewManager() *Manager {
	m := &Manager{
		installers: make(map[manifest.ServerType]Installer),
		validator:  security.NewValidator(),
		verifier:   security.NewVerifier(),
	}

	m.installers[manifest.ServerTypeNode] = NewNPMInstaller(m.validator)
	m.installers[manifest.ServerTypePython] = NewPipInstaller(m.validator)
	m.installers[manifest.ServerTypeBinary] = NewBinaryInstaller(m.validator, m.verifier)

	return m
}

// Install installs an MCP server.
func (m *Manager) Install(ctx context.Context, server *manifest.Server, installDir string) (*Result, error) {
	installer, ok := m.installers[server.Type]
	if !ok {
		return nil, fmt.Errorf("no installer for server type: %s", server.Type)
	}

	return installer.Install(ctx, server, installDir)
}

// NPMInstaller installs Node.js MCP servers via npm.
type NPMInstaller struct {
	validator *security.Validator
}

// NewNPMInstaller creates a new npm installer.
func NewNPMInstaller(validator *security.Validator) *NPMInstaller {
	return &NPMInstaller{validator: validator}
}

// Name returns the installer name.
func (i *NPMInstaller) Name() string {
	return "npm"
}

// Install installs a Node.js MCP server.
func (i *NPMInstaller) Install(ctx context.Context, server *manifest.Server, installDir string) (*Result, error) {
	result := &Result{
		ServerName:  server.Name,
		InstallPath: installDir,
	}

	// Validate package name and version
	if err := i.validator.ValidatePackageName(server.Source.NPM); err != nil {
		result.Error = err
		return result, err
	}

	if err := i.validator.ValidateVersion(server.Source.Version); err != nil {
		result.Error = err
		return result, err
	}

	// Create installation directory
	if err := os.MkdirAll(installDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create install directory: %w", err)
		return result, result.Error
	}

	// Initialize npm project
	initCmd := exec.CommandContext(ctx, "npm", "init", "-y")
	initCmd.Dir = installDir
	if err := initCmd.Run(); err != nil {
		result.Error = fmt.Errorf("failed to initialize npm project: %w", err)
		return result, result.Error
	}

	// Install package
	packageSpec := fmt.Sprintf("%s@%s", server.Source.NPM, server.Source.Version)
	installCmd := exec.CommandContext(ctx, "npm", "install", "--save", packageSpec)
	installCmd.Dir = installDir
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		result.Error = fmt.Errorf("failed to install npm package: %w", err)
		return result, result.Error
	}

	// Determine entrypoint path
	entrypoint := filepath.Join(installDir, "node_modules", ".bin", server.Entrypoint)
	if runtime.GOOS == "windows" {
		entrypoint += ".cmd"
	}

	if _, err := os.Stat(entrypoint); os.IsNotExist(err) {
		// Try the package directory
		entrypoint = filepath.Join(installDir, "node_modules", server.Source.NPM, "dist", server.Entrypoint+".js")
		if _, err := os.Stat(entrypoint); os.IsNotExist(err) {
			result.Error = fmt.Errorf("entrypoint not found: %s", server.Entrypoint)
			return result, result.Error
		}
	}

	result.Entrypoint = entrypoint
	result.Success = true

	return result, nil
}

// PipInstaller installs Python MCP servers via pip.
type PipInstaller struct {
	validator *security.Validator
}

// NewPipInstaller creates a new pip installer.
func NewPipInstaller(validator *security.Validator) *PipInstaller {
	return &PipInstaller{validator: validator}
}

// Name returns the installer name.
func (i *PipInstaller) Name() string {
	return "pip"
}

// Install installs a Python MCP server.
func (i *PipInstaller) Install(ctx context.Context, server *manifest.Server, installDir string) (*Result, error) {
	result := &Result{
		ServerName:  server.Name,
		InstallPath: installDir,
	}

	// Validate package name and version
	if err := i.validator.ValidatePackageName(server.Source.PyPI); err != nil {
		result.Error = err
		return result, err
	}

	if err := i.validator.ValidateVersion(server.Source.Version); err != nil {
		result.Error = err
		return result, result.Error
	}

	// Create virtual environment directory
	venvDir := filepath.Join(installDir, "venv")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create install directory: %w", err)
		return result, result.Error
	}

	// Create virtual environment
	pythonPath, err := exec.LookPath("python3")
	if err != nil {
		pythonPath, err = exec.LookPath("python")
		if err != nil {
			result.Error = fmt.Errorf("python not found in PATH")
			return result, result.Error
		}
	}

	venvCmd := exec.CommandContext(ctx, pythonPath, "-m", "venv", venvDir)
	if err := venvCmd.Run(); err != nil {
		result.Error = fmt.Errorf("failed to create virtual environment: %w", err)
		return result, result.Error
	}

	// Determine pip path in venv
	var pipPath string
	if runtime.GOOS == "windows" {
		pipPath = filepath.Join(venvDir, "Scripts", "pip.exe")
	} else {
		pipPath = filepath.Join(venvDir, "bin", "pip")
	}

	// Install package
	packageSpec := fmt.Sprintf("%s==%s", server.Source.PyPI, server.Source.Version)
	installCmd := exec.CommandContext(ctx, pipPath, "install", packageSpec)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		result.Error = fmt.Errorf("failed to install pip package: %w", err)
		return result, result.Error
	}

	// Determine entrypoint path
	var entrypoint string
	if runtime.GOOS == "windows" {
		entrypoint = filepath.Join(venvDir, "Scripts", server.Entrypoint+".exe")
	} else {
		entrypoint = filepath.Join(venvDir, "bin", server.Entrypoint)
	}

	if _, err := os.Stat(entrypoint); os.IsNotExist(err) {
		result.Error = fmt.Errorf("entrypoint not found: %s", server.Entrypoint)
		return result, result.Error
	}

	result.Entrypoint = entrypoint
	result.Success = true

	return result, nil
}

// BinaryInstaller installs binary MCP servers.
type BinaryInstaller struct {
	validator *security.Validator
	verifier  *security.Verifier
}

// NewBinaryInstaller creates a new binary installer.
func NewBinaryInstaller(validator *security.Validator, verifier *security.Verifier) *BinaryInstaller {
	return &BinaryInstaller{
		validator: validator,
		verifier:  verifier,
	}
}

// Name returns the installer name.
func (i *BinaryInstaller) Name() string {
	return "binary"
}

// Install installs a binary MCP server.
func (i *BinaryInstaller) Install(ctx context.Context, server *manifest.Server, installDir string) (*Result, error) {
	result := &Result{
		ServerName:  server.Name,
		InstallPath: installDir,
	}

	// Validate URL
	if err := i.validator.ValidateURL(server.Source.URL); err != nil {
		result.Error = err
		return result, err
	}

	// Checksum is required for binary downloads
	if server.Source.Checksum == "" {
		result.Error = fmt.Errorf("checksum is required for binary downloads")
		return result, result.Error
	}

	// Create installation directory
	if err := os.MkdirAll(installDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create install directory: %w", err)
		return result, result.Error
	}

	// Download binary
	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.Source.URL, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result, result.Error
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to download binary: %w", err)
		return result, result.Error
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("download failed with status: %d", resp.StatusCode)
		return result, result.Error
	}

	// Write to temporary file
	tmpFile, err := os.CreateTemp(installDir, "download-*")
	if err != nil {
		result.Error = fmt.Errorf("failed to create temp file: %w", err)
		return result, result.Error
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		result.Error = fmt.Errorf("failed to save download: %w", err)
		return result, result.Error
	}
	tmpFile.Close()

	// Verify checksum
	checksumType := security.ChecksumType(server.Source.ChecksumType)
	if checksumType == "" {
		checksumType = security.ChecksumSHA256
	}

	if err := i.verifier.VerifyFile(tmpPath, server.Source.Checksum, checksumType); err != nil {
		os.Remove(tmpPath)
		result.Error = fmt.Errorf("checksum verification failed: %w", err)
		return result, result.Error
	}

	// Move to final location
	entrypoint := filepath.Join(installDir, server.Entrypoint)
	if err := os.Rename(tmpPath, entrypoint); err != nil {
		os.Remove(tmpPath)
		result.Error = fmt.Errorf("failed to move binary: %w", err)
		return result, result.Error
	}

	// Make executable
	if err := os.Chmod(entrypoint, 0755); err != nil {
		result.Error = fmt.Errorf("failed to make binary executable: %w", err)
		return result, result.Error
	}

	result.Entrypoint = entrypoint
	result.Success = true

	return result, nil
}

// IsInstalled checks if a server is installed.
func IsInstalled(installDir string) bool {
	info, err := os.Stat(installDir)
	return err == nil && info.IsDir()
}
