package registry

import (
	"testing"

	"github.com/xenixo/mcp-adapter/internal/manifest"
)

func TestRegistryLoadFromBytes(t *testing.T) {
	reg := New()

	yamlData := []byte(`
version: "1"
servers:
  - name: test-server
    description: Test server
    type: node
    source:
      npm: "@example/test-server"
      version: "1.0.0"
    entrypoint: test-server
    transport: stdio
`)

	if err := reg.LoadFromBytes(yamlData); err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}

	if reg.Count() != 1 {
		t.Errorf("Count() = %d, want 1", reg.Count())
	}

	server, ok := reg.Get("test-server")
	if !ok {
		t.Fatal("Get(test-server) returned false")
	}

	if server.Name != "test-server" {
		t.Errorf("server.Name = %q, want %q", server.Name, "test-server")
	}

	if server.Type != manifest.ServerTypeNode {
		t.Errorf("server.Type = %v, want %v", server.Type, manifest.ServerTypeNode)
	}
}

func TestRegistryLoadFromBytesInvalid(t *testing.T) {
	reg := New()

	// Invalid YAML
	yamlData := []byte(`
version: "1"
servers:
  - name: test-server
    # missing required fields
`)

	if err := reg.LoadFromBytes(yamlData); err == nil {
		t.Error("LoadFromBytes() should fail for invalid manifest")
	}
}

func TestRegistryList(t *testing.T) {
	reg := New()

	yamlData := []byte(`
version: "1"
servers:
  - name: server-b
    description: Server B
    type: node
    source:
      npm: "@example/server-b"
      version: "1.0.0"
    entrypoint: server-b
    transport: stdio
  - name: server-a
    description: Server A
    type: node
    source:
      npm: "@example/server-a"
      version: "1.0.0"
    entrypoint: server-a
    transport: stdio
`)

	if err := reg.LoadFromBytes(yamlData); err != nil {
		t.Fatalf("LoadFromBytes() error = %v", err)
	}

	servers := reg.List()

	if len(servers) != 2 {
		t.Fatalf("List() returned %d servers, want 2", len(servers))
	}

	// Should be sorted alphabetically
	if servers[0].Name != "server-a" {
		t.Errorf("servers[0].Name = %q, want %q", servers[0].Name, "server-a")
	}
	if servers[1].Name != "server-b" {
		t.Errorf("servers[1].Name = %q, want %q", servers[1].Name, "server-b")
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	reg := New()

	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("Get(nonexistent) should return false")
	}
}

func TestRegistryMultipleLoads(t *testing.T) {
	reg := New()

	yaml1 := []byte(`
version: "1"
servers:
  - name: server-1
    description: Server 1
    type: node
    source:
      npm: "@example/server-1"
      version: "1.0.0"
    entrypoint: server-1
    transport: stdio
`)

	yaml2 := []byte(`
version: "1"
servers:
  - name: server-2
    description: Server 2
    type: python
    source:
      pypi: "server-2"
      version: "1.0.0"
    entrypoint: server-2
    transport: stdio
`)

	if err := reg.LoadFromBytes(yaml1); err != nil {
		t.Fatalf("LoadFromBytes(yaml1) error = %v", err)
	}

	if err := reg.LoadFromBytes(yaml2); err != nil {
		t.Fatalf("LoadFromBytes(yaml2) error = %v", err)
	}

	if reg.Count() != 2 {
		t.Errorf("Count() = %d, want 2", reg.Count())
	}

	if _, ok := reg.Get("server-1"); !ok {
		t.Error("server-1 not found")
	}

	if _, ok := reg.Get("server-2"); !ok {
		t.Error("server-2 not found")
	}
}
