.PHONY: build clean test lint install release help

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
	-X 'github.com/mcpadapter/mcp-adapter/internal/cli.Version=$(VERSION)' \
	-X 'github.com/mcpadapter/mcp-adapter/internal/cli.Commit=$(COMMIT)' \
	-X 'github.com/mcpadapter/mcp-adapter/internal/cli.BuildDate=$(BUILD_DATE)'

# Binary name
BINARY := mcp-adapter

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOVET := $(GOCMD) vet

# Default target
all: build

## Build the binary
build:
	CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/mcp-adapter

## Build for all platforms
build-all: build-darwin-arm64 build-darwin-amd64 build-linux-amd64

build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 ./cmd/mcp-adapter

build-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 ./cmd/mcp-adapter

build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 ./cmd/mcp-adapter

## Run tests
test:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## Run tests with coverage report
coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## Run go vet
vet:
	$(GOVET) ./...

## Format code
fmt:
	gofmt -s -w .

## Tidy and verify dependencies
tidy:
	$(GOMOD) tidy
	$(GOMOD) verify

## Install the binary
install: build
	cp $(BINARY) /usr/local/bin/

## Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY)
	rm -rf dist/
	rm -f coverage.out coverage.html

## Create release artifacts
release: clean build-all
	@echo "Creating checksums..."
	cd dist && sha256sum * > checksums.txt

## Show help
help:
	@echo "mcp-adapter Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "Variables:"
	@echo "  VERSION    Current version (default: git tag or 'dev')"
	@echo "  COMMIT     Git commit hash"
	@echo "  BUILD_DATE Build timestamp"
