package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/xenixo/mcp-adapter/internal/manifest"
)

// Known MCP registries/marketplaces
var knownRegistries = map[string]string{
	"official":  "https://raw.githubusercontent.com/modelcontextprotocol/servers/main/registry.json",
	"community": "https://raw.githubusercontent.com/punkpeye/awesome-mcp-servers/main/registry.json",
}

func newRegistryCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage MCP server registries",
		Long: `Manage MCP server registries and sync servers from remote sources.

Use this command to discover and add new MCP servers from community registries
and marketplaces.`,
	}

	cmd.AddCommand(newRegistrySyncCmd(app))
	cmd.AddCommand(newRegistryListCmd(app))
	cmd.AddCommand(newRegistryAddCmd(app))

	return cmd
}

func newRegistrySyncCmd(app *App) *cobra.Command {
	var (
		source string
		output string
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync servers from a remote registry",
		Long: `Sync MCP servers from a remote registry URL.

This downloads the registry and saves it to your local manifests directory.

Known registries:
  - official:  Official MCP servers from modelcontextprotocol
  - community: Community servers from awesome-mcp-servers

Example:
  mcp-adapter registry sync --source official
  mcp-adapter registry sync --source https://example.com/mcp-registry.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRegistrySync(app, source, output)
		},
	}

	cmd.Flags().StringVarP(&source, "source", "s", "official", "Registry source (name or URL)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: ~/.mcp-adapter/manifests/<source>.yaml)")

	return cmd
}

func newRegistryListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List known registries",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Known MCP registries:")
			fmt.Println()
			for name, url := range knownRegistries {
				fmt.Printf("  %-12s %s\n", name, url)
			}
			fmt.Println()
			fmt.Println("Use 'mcp-adapter registry sync --source <name>' to sync a registry.")
			return nil
		},
	}

	return cmd
}

func newRegistryAddCmd(app *App) *cobra.Command {
	var (
		name        string
		description string
		serverType  string
		npm         string
		pypi        string
		version     string
		entrypoint  string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a custom server to local registry",
		Long: `Add a custom MCP server to your local registry.

Example:
  mcp-adapter registry add --name my-server --type node --npm @myorg/mcp-server --version 1.0.0 --entrypoint mcp-server`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRegistryAdd(app, name, description, serverType, npm, pypi, version, entrypoint)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Server name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Server description")
	cmd.Flags().StringVar(&serverType, "type", "node", "Server type (node, python, binary)")
	cmd.Flags().StringVar(&npm, "npm", "", "NPM package name")
	cmd.Flags().StringVar(&pypi, "pypi", "", "PyPI package name")
	cmd.Flags().StringVar(&version, "version", "", "Package version (required)")
	cmd.Flags().StringVar(&entrypoint, "entrypoint", "", "Entrypoint command (required)")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("version")
	cmd.MarkFlagRequired("entrypoint")

	return cmd
}

func runRegistrySync(app *App, source, output string) error {
	// Resolve source URL
	url := source
	sourceName := source
	if registryURL, ok := knownRegistries[source]; ok {
		url = registryURL
		sourceName = source
	}

	fmt.Printf("Syncing from: %s\n", url)

	// Download registry
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch registry: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read registry: %w", err)
	}

	// Parse registry (try JSON first, then YAML)
	var servers []manifest.Server
	if err := parseRegistryJSON(body, &servers); err != nil {
		if err := parseRegistryYAML(body, &servers); err != nil {
			return fmt.Errorf("failed to parse registry: %w", err)
		}
	}

	if len(servers) == 0 {
		fmt.Println("No servers found in registry.")
		return nil
	}

	// Validate servers
	validServers := make([]manifest.Server, 0)
	for _, server := range servers {
		if err := server.Validate(); err != nil {
			app.Logger.Warn("skipping invalid server",
				zap.String("name", server.Name),
				zap.Error(err),
			)
			continue
		}
		validServers = append(validServers, server)
	}

	// Create manifest
	m := manifest.Manifest{
		Version: "1",
		Servers: validServers,
	}

	// Determine output path
	if output == "" {
		manifestsDir := filepath.Join(app.Config.BaseDir, "manifests")
		if err := os.MkdirAll(manifestsDir, 0755); err != nil {
			return fmt.Errorf("failed to create manifests directory: %w", err)
		}
		output = filepath.Join(manifestsDir, sourceName+".yaml")
	}

	// Write manifest
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	fmt.Printf("✓ Synced %d servers to %s\n", len(validServers), output)
	return nil
}

func parseRegistryJSON(data []byte, servers *[]manifest.Server) error {
	// Try parsing as array of servers
	if err := json.Unmarshal(data, servers); err == nil {
		return nil
	}

	// Try parsing as object with servers field
	var registry struct {
		Servers []manifest.Server `json:"servers"`
	}
	if err := json.Unmarshal(data, &registry); err == nil {
		*servers = registry.Servers
		return nil
	}

	return fmt.Errorf("unable to parse JSON registry")
}

func parseRegistryYAML(data []byte, servers *[]manifest.Server) error {
	var m manifest.Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return err
	}
	*servers = m.Servers
	return nil
}

func runRegistryAdd(app *App, name, description, serverType, npm, pypi, version, entrypoint string) error {
	// Validate inputs
	if name == "" || version == "" || entrypoint == "" {
		return fmt.Errorf("name, version, and entrypoint are required")
	}

	st := manifest.ServerType(serverType)
	switch st {
	case manifest.ServerTypeNode:
		if npm == "" {
			return fmt.Errorf("npm package name is required for node servers")
		}
	case manifest.ServerTypePython:
		if pypi == "" {
			return fmt.Errorf("pypi package name is required for python servers")
		}
	default:
		return fmt.Errorf("unsupported server type: %s", serverType)
	}

	// Create server entry
	server := manifest.Server{
		Name:        name,
		Description: description,
		Type:        st,
		Source: manifest.Source{
			NPM:     npm,
			PyPI:    pypi,
			Version: version,
		},
		Entrypoint: entrypoint,
		Transport:  manifest.TransportStdio,
	}

	if err := server.Validate(); err != nil {
		return fmt.Errorf("invalid server configuration: %w", err)
	}

	// Load or create custom manifest
	manifestsDir := filepath.Join(app.Config.BaseDir, "manifests")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return fmt.Errorf("failed to create manifests directory: %w", err)
	}

	customPath := filepath.Join(manifestsDir, "custom.yaml")
	var m manifest.Manifest

	if data, err := os.ReadFile(customPath); err == nil {
		if err := yaml.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("failed to parse existing manifest: %w", err)
		}
	} else {
		m = manifest.Manifest{Version: "1", Servers: []manifest.Server{}}
	}

	// Check for duplicate
	for i, s := range m.Servers {
		if s.Name == name {
			m.Servers[i] = server
			fmt.Printf("✓ Updated server %q in %s\n", name, customPath)
			goto save
		}
	}

	m.Servers = append(m.Servers, server)
	fmt.Printf("✓ Added server %q to %s\n", name, customPath)

save:
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(customPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}
