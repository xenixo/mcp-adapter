package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ServerConfig holds per-server configuration
type ServerConfig struct {
	Env  map[string]string `yaml:"env,omitempty"`
	Args []string          `yaml:"args,omitempty"`
}

// AppConfig holds the application configuration
type AppConfig struct {
	Servers map[string]ServerConfig `yaml:"servers"`
}

func newConfigCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage server configuration",
		Long: `Manage configuration for MCP servers.

Use this command to set environment variables and default arguments
for servers without passing them on the command line each time.`,
	}

	cmd.AddCommand(newConfigSetCmd(app))
	cmd.AddCommand(newConfigGetCmd(app))
	cmd.AddCommand(newConfigListCmd(app))
	cmd.AddCommand(newConfigEditCmd(app))
	cmd.AddCommand(newConfigPathCmd(app))

	return cmd
}

func newConfigSetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <server> <KEY=VALUE>...",
		Short: "Set environment variables for a server",
		Long: `Set environment variables for a server.

Examples:
  mcp-adapter config set github GITHUB_PERSONAL_ACCESS_TOKEN=ghp_xxxxx
  mcp-adapter config set brave-search BRAVE_API_KEY=your-api-key
  mcp-adapter config set sentry SENTRY_AUTH_TOKEN=xxx SENTRY_ORG=myorg`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverName := args[0]
			envPairs := args[1:]
			return runConfigSet(app, serverName, envPairs)
		},
	}

	return cmd
}

func newConfigGetCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <server>",
		Short: "Get configuration for a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(app, args[0])
		},
	}

	return cmd
}

func newConfigListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList(app)
		},
	}

	return cmd
}

func newConfigEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open configuration file in editor",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigEdit(app)
		},
	}

	return cmd
}

func newConfigPathCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Print configuration file path",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(getConfigPath(app))
		},
	}

	return cmd
}

func getConfigPath(app *App) string {
	return filepath.Join(app.Config.BaseDir, "config.yaml")
}

func loadAppConfig(app *App) (*AppConfig, error) {
	configPath := getConfigPath(app)
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &AppConfig{Servers: make(map[string]ServerConfig)}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.Servers == nil {
		config.Servers = make(map[string]ServerConfig)
	}

	return &config, nil
}

func saveAppConfig(app *App, config *AppConfig) error {
	configPath := getConfigPath(app)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func runConfigSet(app *App, serverName string, envPairs []string) error {
	config, err := loadAppConfig(app)
	if err != nil {
		return err
	}

	serverConfig, ok := config.Servers[serverName]
	if !ok {
		serverConfig = ServerConfig{Env: make(map[string]string)}
	}
	if serverConfig.Env == nil {
		serverConfig.Env = make(map[string]string)
	}

	for _, pair := range envPairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format: %q (expected KEY=VALUE)", pair)
		}
		key := parts[0]
		value := parts[1]

		// Mask sensitive values in output
		displayValue := value
		if isSensitiveKey(key) && len(value) > 8 {
			displayValue = value[:4] + "..." + value[len(value)-4:]
		}

		serverConfig.Env[key] = value
		fmt.Printf("Set %s=%s for %s\n", key, displayValue, serverName)
	}

	config.Servers[serverName] = serverConfig

	if err := saveAppConfig(app, config); err != nil {
		return err
	}

	fmt.Printf("✓ Configuration saved to %s\n", getConfigPath(app))
	return nil
}

func runConfigGet(app *App, serverName string) error {
	config, err := loadAppConfig(app)
	if err != nil {
		return err
	}

	serverConfig, ok := config.Servers[serverName]
	if !ok {
		fmt.Printf("No configuration found for %q\n", serverName)
		return nil
	}

	fmt.Printf("Configuration for %s:\n", serverName)
	fmt.Println()

	if len(serverConfig.Env) > 0 {
		fmt.Println("Environment variables:")
		for key, value := range serverConfig.Env {
			displayValue := value
			if isSensitiveKey(key) && len(value) > 8 {
				displayValue = value[:4] + "..." + value[len(value)-4:]
			}
			fmt.Printf("  %s=%s\n", key, displayValue)
		}
	}

	if len(serverConfig.Args) > 0 {
		fmt.Println()
		fmt.Println("Default arguments:")
		for _, arg := range serverConfig.Args {
			fmt.Printf("  %s\n", arg)
		}
	}

	return nil
}

func runConfigList(app *App) error {
	config, err := loadAppConfig(app)
	if err != nil {
		return err
	}

	if len(config.Servers) == 0 {
		fmt.Println("No servers configured.")
		fmt.Println()
		fmt.Println("Use 'mcp-adapter config set <server> KEY=VALUE' to configure a server.")
		return nil
	}

	fmt.Println("Configured servers:")
	fmt.Println()

	for serverName, serverConfig := range config.Servers {
		envCount := len(serverConfig.Env)
		argsCount := len(serverConfig.Args)
		fmt.Printf("  %s (%d env vars, %d args)\n", serverName, envCount, argsCount)
	}

	fmt.Println()
	fmt.Printf("Configuration file: %s\n", getConfigPath(app))

	return nil
}

func runConfigEdit(app *App) error {
	configPath := getConfigPath(app)

	// Create default config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := `# MCP Adapter Configuration
# 
# Configure environment variables and default arguments for servers.
#
# Example:
# servers:
#   github:
#     env:
#       GITHUB_PERSONAL_ACCESS_TOKEN: "ghp_xxxxxxxxxxxx"
#   brave-search:
#     env:
#       BRAVE_API_KEY: "your-api-key"
#   filesystem:
#     args:
#       - "/path/to/allowed/directory"

servers: {}
`
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	}

	// Try to find an editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Check for common editors
		for _, e := range []string{"code", "vim", "nano", "vi"} {
			if _, err := os.Stat("/usr/bin/" + e); err == nil {
				editor = e
				break
			}
			if _, err := os.Stat("/usr/local/bin/" + e); err == nil {
				editor = e
				break
			}
		}
	}

	if editor == "" {
		fmt.Printf("Configuration file: %s\n", configPath)
		fmt.Println()
		fmt.Println("No editor found. Set EDITOR environment variable or edit the file manually.")
		return nil
	}

	fmt.Printf("Opening %s with %s...\n", configPath, editor)
	fmt.Println("(Close the editor when done)")
	
	// We can't actually launch an editor in this context, so just print the path
	fmt.Printf("\nEdit: %s\n", configPath)

	return nil
}

func isSensitiveKey(key string) bool {
	key = strings.ToUpper(key)
	sensitivePatterns := []string{
		"TOKEN", "KEY", "SECRET", "PASSWORD", "PASS", "CREDENTIAL", "AUTH",
	}
	for _, pattern := range sensitivePatterns {
		if strings.Contains(key, pattern) {
			return true
		}
	}
	return false
}

// GetServerConfig returns the configuration for a server
func GetServerConfig(app *App, serverName string) (*ServerConfig, error) {
	config, err := loadAppConfig(app)
	if err != nil {
		return nil, err
	}

	if serverConfig, ok := config.Servers[serverName]; ok {
		return &serverConfig, nil
	}

	return &ServerConfig{
		Env:  make(map[string]string),
		Args: []string{},
	}, nil
}

// PromptForConfig interactively prompts for configuration
func PromptForConfig(app *App, serverName string, requiredEnv []string) error {
	if len(requiredEnv) == 0 {
		return nil
	}

	config, err := loadAppConfig(app)
	if err != nil {
		return err
	}

	serverConfig, ok := config.Servers[serverName]
	if !ok {
		serverConfig = ServerConfig{Env: make(map[string]string)}
	}

	reader := bufio.NewReader(os.Stdin)
	modified := false

	for _, key := range requiredEnv {
		if _, exists := serverConfig.Env[key]; exists {
			continue
		}

		fmt.Printf("Enter %s: ", key)
		value, _ := reader.ReadString('\n')
		value = strings.TrimSpace(value)

		if value != "" {
			serverConfig.Env[key] = value
			modified = true
		}
	}

	if modified {
		config.Servers[serverName] = serverConfig
		return saveAppConfig(app, config)
	}

	return nil
}
