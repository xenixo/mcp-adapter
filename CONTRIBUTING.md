# Contributing to mcp-adapter

Thank you for your interest in contributing to mcp-adapter! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions. We're all here to build something useful together.

## Getting Started

### Prerequisites

- Go 1.22 or later
- Make
- Git

### Setting Up Development Environment

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/mcp-adapter.git
   cd mcp-adapter
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/mcpadapter/mcp-adapter.git
   ```
4. Install dependencies:
   ```bash
   go mod download
   ```
5. Build:
   ```bash
   make build
   ```
6. Run tests:
   ```bash
   make test
   ```

## Development Workflow

### Creating a Branch

Create a feature branch from `main`:

```bash
git checkout main
git pull upstream main
git checkout -b feature/your-feature-name
```

### Making Changes

1. Write your code following Go best practices
2. Add tests for new functionality
3. Run tests: `make test`
4. Run linter: `make lint`
5. Format code: `make fmt`

### Commit Messages

Use clear, descriptive commit messages:

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters
- Reference issues and pull requests when relevant

Example:
```
Add support for custom manifest directories

- Allow users to specify additional manifest directories via config
- Update documentation with new configuration option
- Add tests for directory loading

Fixes #123
```

### Submitting a Pull Request

1. Push your branch to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```
2. Open a Pull Request on GitHub
3. Fill out the PR template with relevant information
4. Wait for review

## Adding New MCP Servers

To add a new MCP server to the default registry:

1. Edit `manifests/servers.yaml`
2. Follow the manifest schema (see README.md)
3. Test that the server can be installed and run:
   ```bash
   make build
   ./mcp-adapter install your-server
   ./mcp-adapter run your-server
   ```
4. Submit a PR

### Manifest Schema

```yaml
- name: server-name
  description: Human-readable description
  type: node|python|binary
  source:
    npm: "@scope/package-name"  # for node
    pypi: "package-name"        # for python
    url: "https://..."          # for binary
    version: "1.0.0"
    checksum: "sha256:..."      # required for binary
  entrypoint: command-name
  transport: stdio|http
  runtime:
    node: ">=18"               # optional version requirement
    python: ">=3.10"
  args: []                     # optional default args
  env: {}                      # optional environment variables
```

## Code Style

### Go Code

- Follow standard Go conventions
- Run `gofmt` and `goimports`
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and testable

### Documentation

- Keep README.md up to date
- Document new features and configuration options
- Include examples where helpful

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run specific tests
go test -v ./internal/manifest/...
```

### Writing Tests

- Write tests for all new functionality
- Use table-driven tests where appropriate
- Test error cases, not just happy paths
- Keep tests focused and readable

## Release Process

Releases are automated via GitHub Actions when a new tag is pushed:

1. Update version numbers if needed
2. Create and push a tag:
   ```bash
   git tag v1.0.0
   git push upstream v1.0.0
   ```
3. GitHub Actions will build and publish the release

## Getting Help

- Open an issue for bugs or feature requests
- Check existing issues before creating new ones
- Be specific and include reproduction steps for bugs

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
