// Package config provides global configuration for mcp-adapter.
package config

import (
	"os"
	"path/filepath"
	"sync"
)

// Config holds the global configuration for mcp-adapter.
type Config struct {
	// BaseDir is the root directory for mcp-adapter data.
	BaseDir string

	// ServersDir is the directory where MCP servers are installed.
	ServersDir string

	// ManifestPaths contains paths to manifest files.
	ManifestPaths []string

	// LogLevel controls logging verbosity.
	LogLevel string

	// Verbose enables verbose output.
	Verbose bool
}

var (
	defaultConfig *Config
	configOnce    sync.Once
)

// Default returns the default configuration.
func Default() *Config {
	configOnce.Do(func() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = os.TempDir()
		}

		baseDir := filepath.Join(homeDir, ".mcp-adapter")
		defaultConfig = &Config{
			BaseDir:       baseDir,
			ServersDir:    filepath.Join(baseDir, "servers"),
			ManifestPaths: []string{},
			LogLevel:      "info",
			Verbose:       false,
		}
	})
	return defaultConfig
}

// New creates a new Config with the given base directory.
func New(baseDir string) *Config {
	return &Config{
		BaseDir:       baseDir,
		ServersDir:    filepath.Join(baseDir, "servers"),
		ManifestPaths: []string{},
		LogLevel:      "info",
		Verbose:       false,
	}
}

// EnsureDirs creates the required directories if they don't exist.
func (c *Config) EnsureDirs() error {
	dirs := []string{
		c.BaseDir,
		c.ServersDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// ServerInstallPath returns the installation path for a given server.
func (c *Config) ServerInstallPath(serverName string) string {
	return filepath.Join(c.ServersDir, serverName)
}
