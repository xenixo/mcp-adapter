#!/bin/bash
# Install script for mcp-adapter
# Usage: curl -sSL https://raw.githubusercontent.com/xenixo/mcp-adapter/main/scripts/install.sh | bash

set -euo pipefail

REPO="xenixo/mcp-adapter"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="mcp-adapter"

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin)
            OS="darwin"
            ;;
        linux)
            OS="linux"
            ;;
        *)
            echo "Unsupported OS: $OS"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            echo "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
}

# Get latest release version
get_latest_version() {
    curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install() {
    detect_platform

    echo "Detected platform: $PLATFORM"

    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        echo "Failed to get latest version"
        exit 1
    fi

    echo "Installing mcp-adapter $VERSION..."

    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-${PLATFORM}"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Download binary
    echo "Downloading from $DOWNLOAD_URL..."
    curl -sSL "$DOWNLOAD_URL" -o "$TMP_DIR/$BINARY_NAME"

    # Make executable
    chmod +x "$TMP_DIR/$BINARY_NAME"

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    else
        echo "Installing to $INSTALL_DIR requires sudo..."
        sudo mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    fi

    echo "Installed mcp-adapter to $INSTALL_DIR/$BINARY_NAME"
    echo ""
    echo "Run 'mcp-adapter doctor' to verify your installation."
}

install
