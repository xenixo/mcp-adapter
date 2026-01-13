package manifest

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Parser handles parsing of manifest files.
type Parser struct{}

// NewParser creates a new manifest parser.
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile parses a manifest from a file path.
func (p *Parser) ParseFile(path string) (*Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest file %q: %w", path, err)
	}
	defer file.Close()

	return p.Parse(file)
}

// Parse parses a manifest from a reader.
func (p *Parser) Parse(r io.Reader) (*Manifest, error) {
	var manifest Manifest

	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)

	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	return &manifest, nil
}

// ParseBytes parses a manifest from bytes.
func (p *Parser) ParseBytes(data []byte) (*Manifest, error) {
	var manifest Manifest

	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	return &manifest, nil
}
