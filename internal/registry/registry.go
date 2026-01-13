// Package registry provides MCP server registry management.
package registry

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mcpadapter/mcp-adapter/internal/manifest"
)

// Registry manages MCP server definitions.
type Registry struct {
	servers map[string]*manifest.Server
	parser  *manifest.Parser
}

// New creates a new Registry.
func New() *Registry {
	return &Registry{
		servers: make(map[string]*manifest.Server),
		parser:  manifest.NewParser(),
	}
}

// LoadFromFile loads servers from a manifest file.
func (r *Registry) LoadFromFile(path string) error {
	m, err := r.parser.ParseFile(path)
	if err != nil {
		return err
	}

	for i := range m.Servers {
		server := &m.Servers[i]
		r.servers[server.Name] = server
	}

	return nil
}

// LoadFromBytes loads servers from manifest bytes.
func (r *Registry) LoadFromBytes(data []byte) error {
	m, err := r.parser.ParseBytes(data)
	if err != nil {
		return err
	}

	for i := range m.Servers {
		server := &m.Servers[i]
		r.servers[server.Name] = server
	}

	return nil
}

// LoadFromEmbed loads servers from an embedded filesystem.
func (r *Registry) LoadFromEmbed(fsys embed.FS, pattern string) error {
	matches, err := fs.Glob(fsys, pattern)
	if err != nil {
		return fmt.Errorf("failed to glob embedded files: %w", err)
	}

	for _, match := range matches {
		data, err := fsys.ReadFile(match)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %q: %w", match, err)
		}

		if err := r.LoadFromBytes(data); err != nil {
			return fmt.Errorf("failed to load manifest from %q: %w", match, err)
		}
	}

	return nil
}

// LoadFromDirectory loads all manifest files from a directory.
func (r *Registry) LoadFromDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read directory %q: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		path := filepath.Join(dir, name)
		if err := r.LoadFromFile(path); err != nil {
			return fmt.Errorf("failed to load manifest %q: %w", path, err)
		}
	}

	return nil
}

// Get returns a server by name.
func (r *Registry) Get(name string) (*manifest.Server, bool) {
	server, ok := r.servers[name]
	return server, ok
}

// List returns all registered servers sorted by name.
func (r *Registry) List() []*manifest.Server {
	servers := make([]*manifest.Server, 0, len(r.servers))
	for _, server := range r.servers {
		servers = append(servers, server)
	}

	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	return servers
}

// Count returns the number of registered servers.
func (r *Registry) Count() int {
	return len(r.servers)
}
