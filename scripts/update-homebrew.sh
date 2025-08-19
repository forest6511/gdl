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
FORMULA_PATH="Formula/gdl.rb"
TAP_REPO="forest6511/homebrew-tap"

# Calculate SHA256 for macOS binaries
DARWIN_AMD64_URL="https://github.com/forest6511/gdl/releases/download/${VERSION}/gdl-darwin-amd64"
DARWIN_ARM64_URL="https://github.com/forest6511/gdl/releases/download/${VERSION}/gdl-darwin-arm64"

echo "Downloading binaries to calculate SHA256..."
wget -q "${DARWIN_AMD64_URL}" -O /tmp/gdl-darwin-amd64
wget -q "${DARWIN_ARM64_URL}" -O /tmp/gdl-darwin-arm64

SHA256_AMD64=$(sha256sum /tmp/gdl-darwin-amd64 | cut -d' ' -f1)
SHA256_ARM64=$(sha256sum /tmp/gdl-darwin-arm64 | cut -d' ' -f1)

# Create Homebrew formula
cat > gdl.rb << EOF
class Gdl < Formula
  desc "Fast, concurrent file downloader for Go"
  homepage "https://github.com/forest6511/gdl"
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
      url "https://github.com/forest6511/gdl/releases/download/${VERSION}/gdl-linux-amd64"
      sha256 "LINUX_SHA256_HERE"
    end
  end

  def install
    bin.install "gdl"
  end

  test do
    system "#{bin}/gdl", "--version"
  end
end
EOF

echo "Homebrew formula created for version ${VERSION}"
echo "To publish:"
echo "1. Clone or update ${TAP_REPO}"
echo "2. Copy gdl.rb to ${FORMULA_PATH}"
echo "3. Commit and push"