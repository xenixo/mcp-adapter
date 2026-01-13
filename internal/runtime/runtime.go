// Package runtime provides runtime detection for MCP servers.
package runtime

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/mcpadapter/mcp-adapter/internal/manifest"
)

// Runtime represents a detected runtime.
type Runtime struct {
	Name    string
	Path    string
	Version string
}

// Detector handles runtime detection.
type Detector struct{}

// NewDetector creates a new runtime detector.
func NewDetector() *Detector {
	return &Detector{}
}

// DetectNode detects Node.js installation.
func (d *Detector) DetectNode() (*Runtime, error) {
	path, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("node not found in PATH")
	}

	cmd := exec.Command(path, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get node version: %w", err)
	}

	version := strings.TrimPrefix(strings.TrimSpace(out.String()), "v")

	return &Runtime{
		Name:    "node",
		Path:    path,
		Version: version,
	}, nil
}

// DetectNPM detects npm installation.
func (d *Detector) DetectNPM() (*Runtime, error) {
	path, err := exec.LookPath("npm")
	if err != nil {
		return nil, fmt.Errorf("npm not found in PATH")
	}

	cmd := exec.Command(path, "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get npm version: %w", err)
	}

	version := strings.TrimSpace(out.String())

	return &Runtime{
		Name:    "npm",
		Path:    path,
		Version: version,
	}, nil
}

// DetectPython detects Python installation.
func (d *Detector) DetectPython() (*Runtime, error) {
	// Try python3 first, then python
	for _, name := range []string{"python3", "python"} {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}

		cmd := exec.Command(path, "--version")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			continue
		}

		// Python 3.x.x
		version := strings.TrimPrefix(strings.TrimSpace(out.String()), "Python ")

		return &Runtime{
			Name:    "python",
			Path:    path,
			Version: version,
		}, nil
	}

	return nil, fmt.Errorf("python not found in PATH")
}

// DetectPip detects pip installation.
func (d *Detector) DetectPip() (*Runtime, error) {
	// Try pip3 first, then pip
	for _, name := range []string{"pip3", "pip"} {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}

		cmd := exec.Command(path, "--version")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			continue
		}

		// pip 23.0 from ...
		parts := strings.Fields(strings.TrimSpace(out.String()))
		if len(parts) >= 2 {
			return &Runtime{
				Name:    "pip",
				Path:    path,
				Version: parts[1],
			}, nil
		}
	}

	return nil, fmt.Errorf("pip not found in PATH")
}

// DetectForServer detects the required runtime for a server.
func (d *Detector) DetectForServer(server *manifest.Server) (*Runtime, error) {
	switch server.Type {
	case manifest.ServerTypeNode:
		return d.DetectNode()
	case manifest.ServerTypePython:
		return d.DetectPython()
	case manifest.ServerTypeBinary:
		return &Runtime{Name: "binary", Path: "", Version: ""}, nil
	default:
		return nil, fmt.Errorf("unknown server type: %s", server.Type)
	}
}

// CheckAll returns the status of all runtimes.
func (d *Detector) CheckAll() map[string]*Runtime {
	result := make(map[string]*Runtime)

	if rt, err := d.DetectNode(); err == nil {
		result["node"] = rt
	}
	if rt, err := d.DetectNPM(); err == nil {
		result["npm"] = rt
	}
	if rt, err := d.DetectPython(); err == nil {
		result["python"] = rt
	}
	if rt, err := d.DetectPip(); err == nil {
		result["pip"] = rt
	}

	return result
}

// MeetsRequirement checks if a runtime version meets the requirement.
func MeetsRequirement(version, requirement string) bool {
	if requirement == "" {
		return true
	}

	// Parse requirement (e.g., ">=18", ">=3.10")
	re := regexp.MustCompile(`^(>=|<=|>|<|=)?(.+)$`)
	matches := re.FindStringSubmatch(requirement)
	if len(matches) != 3 {
		return false
	}

	operator := matches[1]
	if operator == "" {
		operator = ">="
	}
	requiredVersion := matches[2]

	return compareVersions(version, operator, requiredVersion)
}

func compareVersions(actual, operator, required string) bool {
	actualParts := parseVersion(actual)
	requiredParts := parseVersion(required)

	cmp := compareVersionParts(actualParts, requiredParts)

	switch operator {
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case "=", "==":
		return cmp == 0
	default:
		return false
	}
}

func parseVersion(v string) []int {
	parts := strings.Split(v, ".")
	result := make([]int, len(parts))
	for i, p := range parts {
		// Remove any non-numeric suffix
		numPart := regexp.MustCompile(`^\d+`).FindString(p)
		if numPart != "" {
			result[i], _ = strconv.Atoi(numPart)
		}
	}
	return result
}

func compareVersionParts(a, b []int) int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	for i := 0; i < maxLen; i++ {
		av, bv := 0, 0
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}

		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}

	return 0
}
