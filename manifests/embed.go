// Package manifests provides embedded manifest files.
package manifests

import "embed"

// FS contains the embedded manifest files.
//
//go:embed *.yaml
var FS embed.FS
