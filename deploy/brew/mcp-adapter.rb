# frozen_string_literal: true

# Homebrew formula for mcp-adapter
# To install: brew install mcpadapter/tap/mcp-adapter
class McpAdapter < Formula
  desc "CLI tool for discovering, installing, and running MCP servers"
  homepage "https://github.com/mcpadapter/mcp-adapter"
  version "0.1.0"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/mcpadapter/mcp-adapter/releases/download/v#{version}/mcp-adapter-darwin-arm64"
      sha256 "PLACEHOLDER_SHA256_DARWIN_ARM64"

      def install
        bin.install "mcp-adapter-darwin-arm64" => "mcp-adapter"
      end
    end

    on_intel do
      url "https://github.com/mcpadapter/mcp-adapter/releases/download/v#{version}/mcp-adapter-darwin-amd64"
      sha256 "PLACEHOLDER_SHA256_DARWIN_AMD64"

      def install
        bin.install "mcp-adapter-darwin-amd64" => "mcp-adapter"
      end
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/mcpadapter/mcp-adapter/releases/download/v#{version}/mcp-adapter-linux-amd64"
      sha256 "PLACEHOLDER_SHA256_LINUX_AMD64"

      def install
        bin.install "mcp-adapter-linux-amd64" => "mcp-adapter"
      end
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/mcp-adapter version")
  end
end
