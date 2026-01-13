// Package manifest provides types and parsing for MCP server manifests.
package manifest

import "fmt"

// Transport defines the MCP transport type.
type Transport string

const (
	TransportStdio Transport = "stdio"
	TransportHTTP  Transport = "http"
)

// ServerType defines the type of MCP server.
type ServerType string

const (
	ServerTypeNode   ServerType = "node"
	ServerTypePython ServerType = "python"
	ServerTypeBinary ServerType = "binary"
)

// Source defines where an MCP server is obtained from.
type Source struct {
	// NPM package name (for node servers).
	NPM string `yaml:"npm,omitempty"`

	// PyPI package name (for python servers).
	PyPI string `yaml:"pypi,omitempty"`

	// URL for binary download.
	URL string `yaml:"url,omitempty"`

	// Version of the package.
	Version string `yaml:"version"`

	// Checksum for verification (optional for npm/pypi, required for binary).
	Checksum string `yaml:"checksum,omitempty"`

	// ChecksumType specifies the hash algorithm (sha256, sha512).
	ChecksumType string `yaml:"checksum_type,omitempty"`
}

// RuntimeRequirements defines the runtime version constraints.
type RuntimeRequirements struct {
	Node   string `yaml:"node,omitempty"`
	Python string `yaml:"python,omitempty"`
}

// Server represents an MCP server manifest entry.
type Server struct {
	// Name is the unique identifier for the server.
	Name string `yaml:"name"`

	// Description provides a human-readable description.
	Description string `yaml:"description"`

	// Type indicates the server type (node, python, binary).
	Type ServerType `yaml:"type"`

	// Source defines where to obtain the server.
	Source Source `yaml:"source"`

	// Entrypoint is the command or script to run.
	Entrypoint string `yaml:"entrypoint"`

	// Transport defines the MCP transport (stdio, http).
	Transport Transport `yaml:"transport"`

	// Runtime specifies version requirements.
	Runtime RuntimeRequirements `yaml:"runtime,omitempty"`

	// Args are default arguments to pass to the server.
	Args []string `yaml:"args,omitempty"`

	// Env defines environment variables for the server.
	Env map[string]string `yaml:"env,omitempty"`
}

// Manifest represents the complete manifest file.
type Manifest struct {
	// Version of the manifest schema.
	Version string `yaml:"version"`

	// Servers is the list of MCP server definitions.
	Servers []Server `yaml:"servers"`
}

// Validate checks if the server definition is valid.
func (s *Server) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("server name is required")
	}

	if s.Type == "" {
		return fmt.Errorf("server type is required for %q", s.Name)
	}

	switch s.Type {
	case ServerTypeNode:
		if s.Source.NPM == "" {
			return fmt.Errorf("npm source is required for node server %q", s.Name)
		}
	case ServerTypePython:
		if s.Source.PyPI == "" {
			return fmt.Errorf("pypi source is required for python server %q", s.Name)
		}
	case ServerTypeBinary:
		if s.Source.URL == "" {
			return fmt.Errorf("url source is required for binary server %q", s.Name)
		}
		if s.Source.Checksum == "" {
			return fmt.Errorf("checksum is required for binary server %q", s.Name)
		}
	default:
		return fmt.Errorf("invalid server type %q for %q", s.Type, s.Name)
	}

	if s.Source.Version == "" {
		return fmt.Errorf("version is required for server %q", s.Name)
	}

	if s.Entrypoint == "" {
		return fmt.Errorf("entrypoint is required for server %q", s.Name)
	}

	switch s.Transport {
	case TransportStdio, TransportHTTP:
		// valid
	case "":
		return fmt.Errorf("transport is required for server %q", s.Name)
	default:
		return fmt.Errorf("invalid transport %q for server %q", s.Transport, s.Name)
	}

	return nil
}

// Validate checks if the manifest is valid.
func (m *Manifest) Validate() error {
	if len(m.Servers) == 0 {
		return fmt.Errorf("manifest must contain at least one server")
	}

	seen := make(map[string]bool)
	for _, server := range m.Servers {
		if err := server.Validate(); err != nil {
			return err
		}
		if seen[server.Name] {
			return fmt.Errorf("duplicate server name: %q", server.Name)
		}
		seen[server.Name] = true
	}

	return nil
}
