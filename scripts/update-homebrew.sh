#!/bin/bash
# Update Homebrew formula after release

set -e

VERSION=${1:-}
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.10.0"
    exit 1
fi

VERSION_NUM=${VERSION#v}
FORMULA_PATH="Formula/godl.rb"
TAP_REPO="forest6511/homebrew-tap"

# Calculate SHA256 for macOS binaries
DARWIN_AMD64_URL="https://github.com/forest6511/godl/releases/download/${VERSION}/godl-darwin-amd64"
DARWIN_ARM64_URL="https://github.com/forest6511/godl/releases/download/${VERSION}/godl-darwin-arm64"

echo "Downloading binaries to calculate SHA256..."
wget -q "${DARWIN_AMD64_URL}" -O /tmp/godl-darwin-amd64
wget -q "${DARWIN_ARM64_URL}" -O /tmp/godl-darwin-arm64

SHA256_AMD64=$(sha256sum /tmp/godl-darwin-amd64 | cut -d' ' -f1)
SHA256_ARM64=$(sha256sum /tmp/godl-darwin-arm64 | cut -d' ' -f1)

# Create Homebrew formula
cat > godl.rb << EOF
class Godl < Formula
  desc "Fast, concurrent file downloader for Go"
  homepage "https://github.com/forest6511/godl"
  version "${VERSION_NUM}"
  license "MIT"

  on_macos do
    if Hardware::CPU.intel?
      url "${DARWIN_AMD64_URL}"
      sha256 "${SHA256_AMD64}"
    else
      url "${DARWIN_ARM64_URL}"
      sha256 "${SHA256_ARM64}"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/forest6511/godl/releases/download/${VERSION}/godl-linux-amd64"
      sha256 "LINUX_SHA256_HERE"
    end
  end

  def install
    bin.install "godl"
  end

  test do
    system "#{bin}/godl", "--version"
  end
end
EOF

echo "Homebrew formula created for version ${VERSION}"
echo "To publish:"
echo "1. Clone or update ${TAP_REPO}"
echo "2. Copy godl.rb to ${FORMULA_PATH}"
echo "3. Commit and push"