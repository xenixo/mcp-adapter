package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/xenixo/mcp-adapter/internal/installer"
	"github.com/xenixo/mcp-adapter/internal/registry"
	"github.com/xenixo/mcp-adapter/manifests"
)

func newUninstallCmd(app *App) *cobra.Command {
	var (
		force bool
		all   bool
	)

	cmd := &cobra.Command{
		Use:     "uninstall <server>",
		Aliases: []string{"remove", "rm"},
		Short:   "Uninstall an MCP server",
		Long: `Uninstall an installed MCP server.

This command removes the specified MCP server from the local installation
directory. The server definition remains in the registry and can be
reinstalled at any time.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				return runUninstallAll(app, force)
			}
			if len(args) == 0 {
				return fmt.Errorf("server name required (or use --all)")
			}
			return runUninstall(app, args[0], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&all, "all", false, "Uninstall all servers")

	return cmd
}

func runUninstall(app *App, serverName string, force bool) error {
	installDir := app.Config.ServerInstallPath(serverName)

	if !installer.IsInstalled(installDir) {
		return fmt.Errorf("server %q is not installed", serverName)
	}

	// Confirm uninstallation
	if !force {
		fmt.Printf("Uninstall server %q from %s? [y/N] ", serverName, installDir)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Remove installation directory
	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("failed to remove server: %w", err)
	}

	fmt.Printf("✓ Uninstalled %s\n", serverName)
	return nil
}

func runUninstallAll(app *App, force bool) error {
	// Load registry to get server names
	reg := registry.New()

	if err := reg.LoadFromEmbed(manifests.FS, "*.yaml"); err != nil {
		return fmt.Errorf("failed to load embedded manifests: %w", err)
	}

	userManifestsDir := filepath.Join(app.Config.BaseDir, "manifests")
	_ = reg.LoadFromDirectory(userManifestsDir)

	// Find installed servers
	var installed []string
	for _, server := range reg.List() {
		installDir := app.Config.ServerInstallPath(server.Name)
		if installer.IsInstalled(installDir) {
			installed = append(installed, server.Name)
		}
	}

	// Also check for servers not in registry
	if _, err := os.Stat(app.Config.ServersDir); err == nil {
		entries, _ := os.ReadDir(app.Config.ServersDir)
		for _, entry := range entries {
			if entry.IsDir() {
				found := false
				for _, name := range installed {
					if name == entry.Name() {
						found = true
						break
					}
				}
				if !found {
					installed = append(installed, entry.Name())
				}
			}
		}
	}

	if len(installed) == 0 {
		fmt.Println("No servers are installed.")
		return nil
	}

	// Confirm
	if !force {
		fmt.Println("The following servers will be uninstalled:")
		for _, name := range installed {
			fmt.Printf("  • %s\n", name)
		}
		fmt.Print("\nContinue? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Uninstall each server
	for _, name := range installed {
		installDir := app.Config.ServerInstallPath(name)
		if err := os.RemoveAll(installDir); err != nil {
			fmt.Printf("✗ Failed to uninstall %s: %v\n", name, err)
		} else {
			fmt.Printf("✓ Uninstalled %s\n", name)
		}
	}

	return nil
}
