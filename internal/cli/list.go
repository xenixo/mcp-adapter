package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/xenixo/mcp-adapter/internal/installer"
	"github.com/xenixo/mcp-adapter/internal/manifest"
	"github.com/xenixo/mcp-adapter/internal/registry"
	"github.com/xenixo/mcp-adapter/manifests"
)

func newListCmd(app *App) *cobra.Command {
	var (
		showInstalled bool
		jsonOutput    bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available MCP servers",
		Long: `List all MCP servers defined in the registry.

This command displays information about available MCP servers including
their name, type, description, and installation status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(app, showInstalled, jsonOutput)
		},
	}

	cmd.Flags().BoolVarP(&showInstalled, "installed", "i", false, "Show only installed servers")
	cmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")

	return cmd
}

func runList(app *App, showInstalled, jsonOutput bool) error {
	reg := registry.New()

	// Load embedded manifests
	if err := reg.LoadFromEmbed(manifests.FS, "*.yaml"); err != nil {
		return fmt.Errorf("failed to load embedded manifests: %w", err)
	}

	// Load user manifests
	userManifestsDir := filepath.Join(app.Config.BaseDir, "manifests")
	if err := reg.LoadFromDirectory(userManifestsDir); err != nil {
		app.Logger.Warn("failed to load user manifests", 
			zap.String("dir", userManifestsDir),
			zap.Error(err),
		)
	}

	servers := reg.List()

	if len(servers) == 0 {
		fmt.Println("No servers found in registry.")
		return nil
	}

	if jsonOutput {
		return printServersJSON(app, servers, showInstalled)
	}

	return printServersTable(app, servers, showInstalled)
}

func printServersTable(app *App, servers []*manifest.Server, showInstalled bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tVERSION\tTRANSPORT\tINSTALLED\tDESCRIPTION")
	fmt.Fprintln(w, "----\t----\t-------\t---------\t---------\t-----------")

	for _, server := range servers {
		installDir := app.Config.ServerInstallPath(server.Name)
		installed := installer.IsInstalled(installDir)

		if showInstalled && !installed {
			continue
		}

		installedStr := "no"
		if installed {
			installedStr = "yes"
		}

		desc := server.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			server.Name,
			server.Type,
			server.Source.Version,
			server.Transport,
			installedStr,
			desc,
		)
	}

	return w.Flush()
}

func printServersJSON(app *App, servers []*manifest.Server, showInstalled bool) error {
	type serverInfo struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Version     string `json:"version"`
		Transport   string `json:"transport"`
		Installed   bool   `json:"installed"`
		Description string `json:"description"`
	}

	var output []serverInfo

	for _, server := range servers {
		installDir := app.Config.ServerInstallPath(server.Name)
		installed := installer.IsInstalled(installDir)

		if showInstalled && !installed {
			continue
		}

		output = append(output, serverInfo{
			Name:        server.Name,
			Type:        string(server.Type),
			Version:     server.Source.Version,
			Transport:   string(server.Transport),
			Installed:   installed,
			Description: server.Description,
		})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
