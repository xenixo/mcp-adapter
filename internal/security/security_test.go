package security

import (
	"strings"
	"testing"
)

func TestValidatorValidatePackageName(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		pkg     string
		wantErr bool
	}{
		// Valid npm packages
		{"simple npm package", "express", false},
		{"scoped npm package", "@modelcontextprotocol/server-filesystem", false},
		{"npm package with numbers", "lodash4", false},
		{"npm package with hyphen", "left-pad", false},
		{"npm package with underscore", "node_modules", false},

		// Valid pypi packages
		{"simple pypi package", "requests", false},
		{"pypi package with hyphen", "scikit-learn", false},
		{"pypi package with underscore", "black_formatter", false},

		// Invalid packages
		{"package with semicolon", "express;rm -rf /", true},
		{"package with pipe", "express|cat /etc/passwd", true},
		{"package with backtick", "express`whoami`", true},
		{"empty package", "", true},
		{"package with spaces", "express package", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidatePackageName(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePackageName(%q) error = %v, wantErr %v", tt.pkg, err, tt.wantErr)
			}
		})
	}
}

func TestValidatorValidateVersion(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"simple version", "1.0.0", false},
		{"version with prerelease", "1.0.0-beta.1", false},
		{"version with build metadata", "1.0.0+build.123", false},
		{"version with both", "1.0.0-alpha+001", false},
		{"major only", "1", false},
		{"major.minor", "1.0", false},

		{"version with semicolon", "1.0.0;rm -rf /", true},
		{"version with pipe", "1.0.0|cat", true},
		{"version with spaces", "1.0.0 latest", true},
		{"empty version", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVersion(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

func TestValidatorValidateURL(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https url", "https://github.com/user/repo/releases/download/v1.0/binary", false},
		{"https with port", "https://example.com:8443/file", false},

		{"http url", "http://example.com/file", true},
		{"ftp url", "ftp://example.com/file", true},
		{"file url", "file:///etc/passwd", true},
		{"javascript url", "javascript:alert(1)", true},
		{"empty url", "", true},
		{"invalid url", "not-a-url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidatorValidateEntrypoint(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name       string
		entrypoint string
		wantErr    bool
	}{
		{"simple entrypoint", "mcp-server-filesystem", false},
		{"entrypoint with path", "dist/server.js", false},
		{"entrypoint with underscore", "mcp_server", false},

		{"entrypoint with semicolon", "server;rm -rf /", true},
		{"entrypoint with pipe", "server|cat", true},
		{"entrypoint with backtick", "server`whoami`", true},
		{"entrypoint with dollar", "server$PATH", true},
		{"entrypoint with path traversal", "../../../etc/passwd", true},
		{"entrypoint with redirect", "server>/etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateEntrypoint(tt.entrypoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEntrypoint(%q) error = %v, wantErr %v", tt.entrypoint, err, tt.wantErr)
			}
		})
	}
}

func TestVerifierVerify(t *testing.T) {
	v := NewVerifier()

	content := "hello world"
	// sha256 of "hello world"
	validSHA256 := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	tests := []struct {
		name         string
		content      string
		checksum     string
		checksumType ChecksumType
		wantErr      bool
	}{
		{"valid sha256", content, validSHA256, ChecksumSHA256, false},
		{"valid sha256 with empty type", content, validSHA256, "", false},
		{"invalid checksum", content, "invalidchecksum", ChecksumSHA256, true},
		{"wrong checksum", content, "a" + validSHA256[1:], ChecksumSHA256, true},
		{"unsupported type", content, validSHA256, "md5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Verify(strings.NewReader(tt.content), tt.checksum, tt.checksumType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
