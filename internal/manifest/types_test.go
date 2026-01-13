package manifest

import (
	"testing"
)

func TestServerValidate(t *testing.T) {
	tests := []struct {
		name    string
		server  Server
		wantErr bool
	}{
		{
			name: "valid node server",
			server: Server{
				Name:        "test-server",
				Description: "Test server",
				Type:        ServerTypeNode,
				Source: Source{
					NPM:     "@example/test-server",
					Version: "1.0.0",
				},
				Entrypoint: "test-server",
				Transport:  TransportStdio,
			},
			wantErr: false,
		},
		{
			name: "valid python server",
			server: Server{
				Name:        "test-server",
				Description: "Test server",
				Type:        ServerTypePython,
				Source: Source{
					PyPI:    "test-server",
					Version: "1.0.0",
				},
				Entrypoint: "test-server",
				Transport:  TransportStdio,
			},
			wantErr: false,
		},
		{
			name: "valid binary server",
			server: Server{
				Name:        "test-server",
				Description: "Test server",
				Type:        ServerTypeBinary,
				Source: Source{
					URL:      "https://example.com/test-server",
					Version:  "1.0.0",
					Checksum: "abc123",
				},
				Entrypoint: "test-server",
				Transport:  TransportHTTP,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			server: Server{
				Type: ServerTypeNode,
				Source: Source{
					NPM:     "@example/test-server",
					Version: "1.0.0",
				},
				Entrypoint: "test-server",
				Transport:  TransportStdio,
			},
			wantErr: true,
		},
		{
			name: "missing type",
			server: Server{
				Name: "test-server",
				Source: Source{
					NPM:     "@example/test-server",
					Version: "1.0.0",
				},
				Entrypoint: "test-server",
				Transport:  TransportStdio,
			},
			wantErr: true,
		},
		{
			name: "node server missing npm",
			server: Server{
				Name: "test-server",
				Type: ServerTypeNode,
				Source: Source{
					Version: "1.0.0",
				},
				Entrypoint: "test-server",
				Transport:  TransportStdio,
			},
			wantErr: true,
		},
		{
			name: "python server missing pypi",
			server: Server{
				Name: "test-server",
				Type: ServerTypePython,
				Source: Source{
					Version: "1.0.0",
				},
				Entrypoint: "test-server",
				Transport:  TransportStdio,
			},
			wantErr: true,
		},
		{
			name: "binary server missing url",
			server: Server{
				Name: "test-server",
				Type: ServerTypeBinary,
				Source: Source{
					Version:  "1.0.0",
					Checksum: "abc123",
				},
				Entrypoint: "test-server",
				Transport:  TransportStdio,
			},
			wantErr: true,
		},
		{
			name: "binary server missing checksum",
			server: Server{
				Name: "test-server",
				Type: ServerTypeBinary,
				Source: Source{
					URL:     "https://example.com/test-server",
					Version: "1.0.0",
				},
				Entrypoint: "test-server",
				Transport:  TransportStdio,
			},
			wantErr: true,
		},
		{
			name: "missing version",
			server: Server{
				Name: "test-server",
				Type: ServerTypeNode,
				Source: Source{
					NPM: "@example/test-server",
				},
				Entrypoint: "test-server",
				Transport:  TransportStdio,
			},
			wantErr: true,
		},
		{
			name: "missing entrypoint",
			server: Server{
				Name: "test-server",
				Type: ServerTypeNode,
				Source: Source{
					NPM:     "@example/test-server",
					Version: "1.0.0",
				},
				Transport: TransportStdio,
			},
			wantErr: true,
		},
		{
			name: "missing transport",
			server: Server{
				Name: "test-server",
				Type: ServerTypeNode,
				Source: Source{
					NPM:     "@example/test-server",
					Version: "1.0.0",
				},
				Entrypoint: "test-server",
			},
			wantErr: true,
		},
		{
			name: "invalid transport",
			server: Server{
				Name: "test-server",
				Type: ServerTypeNode,
				Source: Source{
					NPM:     "@example/test-server",
					Version: "1.0.0",
				},
				Entrypoint: "test-server",
				Transport:  "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			server: Server{
				Name: "test-server",
				Type: "invalid",
				Source: Source{
					Version: "1.0.0",
				},
				Entrypoint: "test-server",
				Transport:  TransportStdio,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.server.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Server.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManifestValidate(t *testing.T) {
	tests := []struct {
		name     string
		manifest Manifest
		wantErr  bool
	}{
		{
			name: "valid manifest",
			manifest: Manifest{
				Version: "1",
				Servers: []Server{
					{
						Name: "test-server",
						Type: ServerTypeNode,
						Source: Source{
							NPM:     "@example/test-server",
							Version: "1.0.0",
						},
						Entrypoint: "test-server",
						Transport:  TransportStdio,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty servers",
			manifest: Manifest{
				Version: "1",
				Servers: []Server{},
			},
			wantErr: true,
		},
		{
			name: "duplicate server names",
			manifest: Manifest{
				Version: "1",
				Servers: []Server{
					{
						Name: "test-server",
						Type: ServerTypeNode,
						Source: Source{
							NPM:     "@example/test-server",
							Version: "1.0.0",
						},
						Entrypoint: "test-server",
						Transport:  TransportStdio,
					},
					{
						Name: "test-server",
						Type: ServerTypeNode,
						Source: Source{
							NPM:     "@example/test-server-2",
							Version: "1.0.0",
						},
						Entrypoint: "test-server-2",
						Transport:  TransportStdio,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid server in manifest",
			manifest: Manifest{
				Version: "1",
				Servers: []Server{
					{
						Name: "", // missing name
						Type: ServerTypeNode,
						Source: Source{
							NPM:     "@example/test-server",
							Version: "1.0.0",
						},
						Entrypoint: "test-server",
						Transport:  TransportStdio,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Manifest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
