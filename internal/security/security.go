// Package security provides security utilities for mcp-adapter.
package security

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"regexp"
)

// ChecksumType defines the type of checksum algorithm.
type ChecksumType string

const (
	ChecksumSHA256 ChecksumType = "sha256"
	ChecksumSHA512 ChecksumType = "sha512"
)

// Verifier handles checksum verification.
type Verifier struct{}

// NewVerifier creates a new checksum verifier.
func NewVerifier() *Verifier {
	return &Verifier{}
}

// VerifyFile verifies a file's checksum.
func (v *Verifier) VerifyFile(path string, expectedChecksum string, checksumType ChecksumType) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file for verification: %w", err)
	}
	defer file.Close()

	return v.Verify(file, expectedChecksum, checksumType)
}

// Verify verifies a reader's checksum.
func (v *Verifier) Verify(r io.Reader, expectedChecksum string, checksumType ChecksumType) error {
	var h hash.Hash

	switch checksumType {
	case ChecksumSHA256, "":
		h = sha256.New()
	case ChecksumSHA512:
		h = sha512.New()
	default:
		return fmt.Errorf("unsupported checksum type: %s", checksumType)
	}

	if _, err := io.Copy(h, r); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	actualChecksum := hex.EncodeToString(h.Sum(nil))

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// ComputeChecksum computes the checksum of a file.
func (v *Verifier) ComputeChecksum(path string, checksumType ChecksumType) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var h hash.Hash

	switch checksumType {
	case ChecksumSHA256, "":
		h = sha256.New()
	case ChecksumSHA512:
		h = sha512.New()
	default:
		return "", fmt.Errorf("unsupported checksum type: %s", checksumType)
	}

	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("failed to compute checksum: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Validator provides security validation utilities.
type Validator struct{}

// NewValidator creates a new security validator.
func NewValidator() *Validator {
	return &Validator{}
}

// ValidatePackageName validates that a package name is safe.
func (v *Validator) ValidatePackageName(name string) error {
	// npm package names: @scope/name or name
	npmPattern := regexp.MustCompile(`^(@[a-z0-9-~][a-z0-9-._~]*/)?[a-z0-9-~][a-z0-9-._~]*$`)
	// pypi package names
	pypiPattern := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$`)

	if npmPattern.MatchString(name) || pypiPattern.MatchString(name) {
		return nil
	}

	return fmt.Errorf("invalid package name: %q", name)
}

// ValidateVersion validates that a version string is safe.
func (v *Validator) ValidateVersion(version string) error {
	// Allow semver-like versions and calver (YYYY.MM.DD format)
	pattern := regexp.MustCompile(`^[0-9]+(\.[0-9]+)*(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`)
	if !pattern.MatchString(version) {
		return fmt.Errorf("invalid version format: %q", version)
	}
	return nil
}

// ValidateURL validates that a URL is safe for download.
func (v *Validator) ValidateURL(url string) error {
	// Only allow https URLs with optional port
	pattern := regexp.MustCompile(`^https://[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}(:[0-9]+)?/`)
	if !pattern.MatchString(url) {
		return fmt.Errorf("invalid or insecure URL: %q (only HTTPS is allowed)", url)
	}
	return nil
}

// ValidateEntrypoint validates that an entrypoint is safe.
func (v *Validator) ValidateEntrypoint(entrypoint string) error {
	// Disallow shell metacharacters
	dangerousChars := regexp.MustCompile(`[;&|<>$` + "`" + `\\]`)
	if dangerousChars.MatchString(entrypoint) {
		return fmt.Errorf("entrypoint contains dangerous characters: %q", entrypoint)
	}

	// Disallow path traversal
	if regexp.MustCompile(`\.\.`).MatchString(entrypoint) {
		return fmt.Errorf("entrypoint contains path traversal: %q", entrypoint)
	}

	return nil
}
