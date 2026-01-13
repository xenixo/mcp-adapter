package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/mcpadapter/mcp-adapter/internal/runtime"
)

func newDoctorCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check system requirements and configuration",
		Long: `Check the system for required runtimes and configuration.

This command validates that all required tools are installed and
properly configured for running MCP servers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(app)
		},
	}

	return cmd
}

func runDoctor(app *App) error {
	fmt.Println("mcp-adapter doctor")
	fmt.Println("==================")
	fmt.Println()

	allOk := true

	// Check configuration directory
	fmt.Println("Configuration:")
	fmt.Printf("  Base directory: %s\n", app.Config.BaseDir)
	fmt.Printf("  Servers directory: %s\n", app.Config.ServersDir)

	if _, err := os.Stat(app.Config.BaseDir); os.IsNotExist(err) {
		fmt.Printf("  ⚠ Base directory does not exist (will be created on first install)\n")
	} else {
		fmt.Printf("  ✓ Base directory exists\n")
	}
	fmt.Println()

	// Check runtimes
	fmt.Println("Runtimes:")
	detector := runtime.NewDetector()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Node.js
	if rt, err := detector.DetectNode(); err == nil {
		fmt.Fprintf(w, "  ✓ node\t%s\t%s\n", rt.Version, rt.Path)
	} else {
		fmt.Fprintf(w, "  ✗ node\tnot found\t(required for Node.js MCP servers)\n")
	}

	// npm
	if rt, err := detector.DetectNPM(); err == nil {
		fmt.Fprintf(w, "  ✓ npm\t%s\t%s\n", rt.Version, rt.Path)
	} else {
		fmt.Fprintf(w, "  ✗ npm\tnot found\t(required for Node.js MCP servers)\n")
	}

	// Python
	if rt, err := detector.DetectPython(); err == nil {
		fmt.Fprintf(w, "  ✓ python\t%s\t%s\n", rt.Version, rt.Path)
	} else {
		fmt.Fprintf(w, "  ✗ python\tnot found\t(required for Python MCP servers)\n")
	}

	// pip
	if rt, err := detector.DetectPip(); err == nil {
		fmt.Fprintf(w, "  ✓ pip\t%s\t%s\n", rt.Version, rt.Path)
	} else {
		fmt.Fprintf(w, "  ✗ pip\tnot found\t(required for Python MCP servers)\n")
	}

	w.Flush()
	fmt.Println()

	// Check for installed servers
	fmt.Println("Installed Servers:")
	serversDir := app.Config.ServersDir
	if _, err := os.Stat(serversDir); os.IsNotExist(err) {
		fmt.Println("  No servers installed yet.")
	} else {
		entries, err := os.ReadDir(serversDir)
		if err != nil {
			fmt.Printf("  ✗ Error reading servers directory: %v\n", err)
			allOk = false
		} else if len(entries) == 0 {
			fmt.Println("  No servers installed yet.")
		} else {
			for _, entry := range entries {
				if entry.IsDir() {
					serverPath := filepath.Join(serversDir, entry.Name())
					fmt.Printf("  • %s (%s)\n", entry.Name(), serverPath)
				}
			}
		}
	}
	fmt.Println()

	// Check for user manifests
	fmt.Println("User Manifests:")
	userManifestsDir := filepath.Join(app.Config.BaseDir, "manifests")
	if _, err := os.Stat(userManifestsDir); os.IsNotExist(err) {
		fmt.Printf("  No user manifests directory (create %s to add custom servers)\n", userManifestsDir)
	} else {
		entries, err := os.ReadDir(userManifestsDir)
		if err != nil {
			fmt.Printf("  ✗ Error reading manifests directory: %v\n", err)
			allOk = false
		} else {
			count := 0
			for _, entry := range entries {
				name := entry.Name()
				if filepath.Ext(name) == ".yaml" || filepath.Ext(name) == ".yml" {
					fmt.Printf("  • %s\n", name)
					count++
				}
			}
			if count == 0 {
				fmt.Println("  No manifest files found.")
			}
		}
	}
	fmt.Println()

	// Summary
	if allOk {
		fmt.Println("✓ All checks passed!")
	} else {
		fmt.Println("✗ Some checks failed. Please resolve the issues above.")
		return fmt.Errorf("doctor checks failed")
	}

	return nil
}
