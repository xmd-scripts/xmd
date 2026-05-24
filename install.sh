#!/bin/sh
# xmd installer
# Usage: curl -sSL https://raw.githubusercontent.com/xmd-scripts/xmd/main/install.sh | sh
set -e

REPO="xmd-scripts/xmd"
BINARY="xmd"

# Detect platform
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Darwin) OS_NAME="darwin" ;;
  Linux)  OS_NAME="linux" ;;
  *)
    echo "Unsupported OS: $OS"
    echo "See https://github.com/$REPO for manual installation."
    exit 1
esac

case "$ARCH" in
  x86_64)  ARCH_NAME="amd64" ;;
  arm64|aarch64) ARCH_NAME="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    echo "See https://github.com/$REPO for manual installation."
    exit 1
esac

PLATFORM="${OS_NAME}-${ARCH_NAME}"
ASSET_NAME="${BINARY}-${PLATFORM}"

echo "xmd installer"
printf "Detecting platform... %s\n" "$PLATFORM"

# Determine latest release
LATEST_URL="https://api.github.com/repos/$REPO/releases/latest"
VERSION="$(curl -sSf "$LATEST_URL" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
if [ -z "$VERSION" ]; then
  echo "Failed to determine latest version."
  exit 1
fi
printf "Fetching latest release... %s\n" "$VERSION"

# Download URL
BASE_URL="https://github.com/$REPO/releases/download/$VERSION"
DOWNLOAD_URL="$BASE_URL/$ASSET_NAME"
CHECKSUM_URL="$BASE_URL/checksums.txt"

# Determine install directory
if [ -w /usr/local/bin ]; then
  INSTALL_DIR="/usr/local/bin"
elif [ -d "$HOME/.local/bin" ] && [ -w "$HOME/.local/bin" ]; then
  INSTALL_DIR="$HOME/.local/bin"
else
  # Try to create ~/.local/bin
  mkdir -p "$HOME/.local/bin" 2>/dev/null && INSTALL_DIR="$HOME/.local/bin" || {
    echo "Cannot find a writable install directory."
    echo "Try: sudo mkdir -p /usr/local/bin && sudo chown \$USER /usr/local/bin"
    exit 1
  }
fi

# Download binary
TMPDIR_INST="$(mktemp -d)"
TMPBIN="$TMPDIR_INST/$ASSET_NAME"
TMPCHK="$TMPDIR_INST/checksums.txt"
trap 'rm -rf "$TMPDIR_INST"' EXIT

SIZE_MB=""
printf "Downloading %s%s...\n" "$ASSET_NAME" "$SIZE_MB"
curl -sSfL "$DOWNLOAD_URL" -o "$TMPBIN"

# Verify checksum
printf "Verifying checksum... "
curl -sSfL "$CHECKSUM_URL" -o "$TMPCHK"
if command -v sha256sum >/dev/null 2>&1; then
  EXPECTED="$(grep "$ASSET_NAME" "$TMPCHK" | awk '{print $1}')"
  ACTUAL="$(sha256sum "$TMPBIN" | awk '{print $1}')"
else
  # macOS
  EXPECTED="$(grep "$ASSET_NAME" "$TMPCHK" | awk '{print $1}')"
  ACTUAL="$(shasum -a 256 "$TMPBIN" | awk '{print $1}')"
fi
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "FAILED"
  echo "Checksum mismatch. Expected: $EXPECTED, got: $ACTUAL"
  exit 1
fi
echo "ok"

# Install
chmod +x "$TMPBIN"
printf "Installing to %s/%s... " "$INSTALL_DIR" "$BINARY"
mv "$TMPBIN" "$INSTALL_DIR/$BINARY"
echo "ok"

echo ""
printf "xmd %s installed.\n" "$VERSION"
echo ""

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo "Warning: $INSTALL_DIR is not in your PATH."
    echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo ""
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    echo ""
    ;;
esac
echo "xmd talks to an OpenAI-compatible completion endpoint by default at"
echo "http://localhost:11434/v1/chat/completions. Start any compatible server"
echo "on that port (or set XMD_COMPLETION_URL to point elsewhere) and"
echo "run your first script:"
echo ""
echo "  curl -sSL https://raw.githubusercontent.com/xmd-scripts/xmd-examples/main/hello/hello.md > hello.md"
echo "  chmod +x hello.md"
echo "  ./hello.md name=world"
echo ""
echo "Documentation: https://github.com/$REPO"
