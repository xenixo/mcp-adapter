# mcp-adapter

[![CI](https://github.com/mcpadapter/mcp-adapter/actions/workflows/ci.yml/badge.svg)](https://github.com/mcpadapter/mcp-adapter/actions/workflows/ci.yml)
[![Release](https://github.com/mcpadapter/mcp-adapter/actions/workflows/release.yml/badge.svg)](https://github.com/mcpadapter/mcp-adapter/actions/workflows/release.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

**mcp-adapter** is a production-grade CLI tool for discovering, installing, and running [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) servers through a unified interface.

## What is MCP?

The **Model Context Protocol (MCP)** is an open protocol that enables AI assistants to securely connect with external data sources and tools. MCP provides a standardized way for AI models to access files, databases, APIs, and other resources through "MCP servers" that implement the protocol.

## Why mcp-adapter?

MCP servers are written in different languages (Node.js, Python, Go) and use different package managers. Managing these servers manually is tedious and error-prone.

**mcp-adapter** solves this by providing:

- 🔍 **Discovery**: Browse available MCP servers from a curated registry
- 📦 **Installation**: One command to install any MCP server (npm, pip, or binary)
- 🚀 **Execution**: Launch servers with proper runtime detection and lifecycle management
- 🔒 **Security**: Checksum verification, safe package validation, no shell injection
- 🍺 **Homebrew**: Install via `brew install` for easy updates

## Installation

### Homebrew (Recommended)

```bash
brew tap mcpadapter/tap
brew install mcp-adapter
```

### Binary Download

Download the latest release from [GitHub Releases](https://github.com/mcpadapter/mcp-adapter/releases):

```bash
# macOS (Apple Silicon)
curl -Lo mcp-adapter https://github.com/mcpadapter/mcp-adapter/releases/latest/download/mcp-adapter-darwin-arm64
chmod +x mcp-adapter
sudo mv mcp-adapter /usr/local/bin/

# macOS (Intel)
curl -Lo mcp-adapter https://github.com/mcpadapter/mcp-adapter/releases/latest/download/mcp-adapter-darwin-amd64
chmod +x mcp-adapter
sudo mv mcp-adapter /usr/local/bin/

# Linux (amd64)
curl -Lo mcp-adapter https://github.com/mcpadapter/mcp-adapter/releases/latest/download/mcp-adapter-linux-amd64
chmod +x mcp-adapter
sudo mv mcp-adapter /usr/local/bin/
```

### From Source

```bash
go install github.com/mcpadapter/mcp-adapter/cmd/mcp-adapter@latest
```

Or build from source:

```bash
git clone https://github.com/mcpadapter/mcp-adapter.git
cd mcp-adapter
make build
```

## Quick Start

### 1. Check your environment

```bash
mcp-adapter doctor
```

This verifies that required runtimes (Node.js, Python) are installed.

### 2. List available servers

```bash
mcp-adapter list
```

Output:
```
NAME          TYPE  VERSION  TRANSPORT  INSTALLED  DESCRIPTION
----          ----  -------  ---------  ---------  -----------
filesystem    node  0.6.1    stdio      no         MCP server for filesystem operations
memory        node  0.6.1    stdio      no         MCP server for in-memory key-value storage
github        node  0.6.1    stdio      no         MCP server for GitHub API integration
...
```

### 3. Install a server

```bash
mcp-adapter install filesystem
```

### 4. Run a server

```bash
mcp-adapter run filesystem -- /path/to/allowed/directory
```

## Commands

### `mcp-adapter list`

List all available MCP servers from the registry.

```bash
# List all servers
mcp-adapter list

# List only installed servers
mcp-adapter list --installed

# Output as JSON
mcp-adapter list --json
```

### `mcp-adapter install <server>`

Install an MCP server.

```bash
# Install a server
mcp-adapter install filesystem

# Force reinstall
mcp-adapter install filesystem --force

# Set installation timeout
mcp-adapter install filesystem --timeout 15m
```

### `mcp-adapter run <server>`

Run an installed MCP server.

```bash
# Run with default settings
mcp-adapter run filesystem

# Pass arguments to the server
mcp-adapter run filesystem -- /allowed/path

# Set environment variables
mcp-adapter run github -e GITHUB_PERSONAL_ACCESS_TOKEN=ghp_xxx
```

### `mcp-adapter doctor`

Check system requirements and configuration.

```bash
mcp-adapter doctor
```

### `mcp-adapter uninstall <server>`

Remove an installed server.

```bash
# Uninstall a server
mcp-adapter uninstall filesystem

# Uninstall without confirmation
mcp-adapter uninstall filesystem --force

# Uninstall all servers
mcp-adapter uninstall --all
```

### `mcp-adapter version`

Print version information.

```bash
mcp-adapter version
mcp-adapter version --short
```

## Configuration

### Directory Structure

mcp-adapter uses `~/.mcp-adapter/` as its base directory:

```
~/.mcp-adapter/
├── servers/          # Installed MCP servers
│   ├── filesystem/
│   ├── github/
│   └── ...
└── manifests/        # Custom server manifests (optional)
    └── custom.yaml
```

### Custom Manifests

You can add custom MCP servers by creating YAML manifests in `~/.mcp-adapter/manifests/`:

```yaml
version: "1"
servers:
  - name: my-custom-server
    description: My custom MCP server
    type: node
    source:
      npm: "@myorg/mcp-server-custom"
      version: "1.0.0"
    entrypoint: mcp-server-custom
    transport: stdio
    runtime:
      node: ">=18"
```

### Manifest Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | Unique identifier for the server |
| `description` | string | ✓ | Human-readable description |
| `type` | enum | ✓ | Server type: `node`, `python`, or `binary` |
| `source.npm` | string | * | NPM package name (for node type) |
| `source.pypi` | string | * | PyPI package name (for python type) |
| `source.url` | string | * | Download URL (for binary type) |
| `source.version` | string | ✓ | Package version |
| `source.checksum` | string | ** | SHA256 checksum (required for binary) |
| `entrypoint` | string | ✓ | Command or script to run |
| `transport` | enum | ✓ | MCP transport: `stdio` or `http` |
| `runtime.node` | string | | Node.js version requirement (e.g., `>=18`) |
| `runtime.python` | string | | Python version requirement (e.g., `>=3.10`) |
| `args` | array | | Default arguments |
| `env` | object | | Default environment variables |

## Security Model

mcp-adapter is designed with security as a first-class concern:

### Package Validation
- Package names are validated against strict patterns before installation
- Version strings are validated to prevent injection attacks
- Binary downloads require HTTPS and checksum verification

### No Shell Execution
- All commands are executed directly without shell interpolation
- Entrypoints are validated to reject shell metacharacters
- No user-controlled strings are passed to shells

### Checksum Verification
- Binary downloads require SHA256 checksums
- Checksums are verified before making binaries executable
- Verification failures abort installation

### Sandboxing (Future)
- Planned: AppArmor/seccomp profiles for Linux
- Planned: Sandbox profiles for macOS
- Planned: Network isolation options

## Supported Runtimes

| Runtime | Minimum Version | Required For |
|---------|-----------------|--------------|
| Node.js | 18+ | Node MCP servers |
| npm | 8+ | Installing Node servers |
| Python | 3.10+ | Python MCP servers |
| pip | 22+ | Installing Python servers |

## Built-in Servers

The following MCP servers are included in the default registry:

| Server | Type | Description |
|--------|------|-------------|
| filesystem | node | File system operations |
| memory | node | In-memory key-value storage |
| github | node | GitHub API integration |
| gitlab | node | GitLab API integration |
| postgres | node | PostgreSQL database access |
| sqlite | node | SQLite database access |
| slack | node | Slack integration |
| brave-search | node | Brave Search API |
| fetch | node | Web content fetching |
| puppeteer | node | Browser automation |

## Integration with AI Assistants

### Claude Desktop

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "mcp-adapter",
      "args": ["run", "filesystem", "--", "/Users/you/Documents"]
    }
  }
}
```

### Cursor

Configure in Cursor settings to use mcp-adapter as the MCP server launcher.

## Development

### Prerequisites

- Go 1.22+
- Make

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run linter
make lint

# Create release artifacts
make release
```

### Project Structure

```
mcp-adapter/
├── cmd/mcp-adapter/       # CLI entrypoint
├── internal/
│   ├── cli/               # Cobra commands
│   ├── config/            # Configuration management
│   ├── installer/         # Package installers (npm, pip, binary)
│   ├── launcher/          # Process lifecycle management
│   ├── manifest/          # Manifest schema and parsing
│   ├── mcp/               # MCP protocol utilities
│   ├── registry/          # Server registry
│   ├── runtime/           # Runtime detection
│   └── security/          # Security utilities
├── manifests/             # Embedded server manifests
├── deploy/brew/           # Homebrew formula
└── .github/workflows/     # CI/CD pipelines
```

## Contributing

Contributions are welcome! Please read our contributing guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Adding New Servers

To add a new MCP server to the default registry:

1. Edit `manifests/servers.yaml`
2. Follow the manifest schema
3. Test installation and execution
4. Submit a PR

## Troubleshooting

### Server fails to start

1. Run `mcp-adapter doctor` to check runtimes
2. Check the server is installed: `mcp-adapter list --installed`
3. Try reinstalling: `mcp-adapter install <server> --force`

### Installation fails

1. Check network connectivity
2. Verify the package exists in npm/PyPI
3. Check for sufficient disk space
4. Review logs with `--verbose` flag

### Runtime not detected

Ensure the runtime is in your PATH:

```bash
which node
which python3
```

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [Model Context Protocol](https://modelcontextprotocol.io/) by Anthropic
- [Cobra](https://github.com/spf13/cobra) CLI framework
- [Zap](https://github.com/uber-go/zap) logging library

---

Made with ❤️ for the MCP ecosystem
