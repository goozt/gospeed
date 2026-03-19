#!/bin/sh
# gospeed installer — Linux, macOS, FreeBSD
# Usage: curl -fsSL https://gospeed.goozt.org/install.sh | bash
set -e

REPO="goozt/gospeed"
GOPATH="${GOPATH:-$(go env GOPATH 2>/dev/null || echo "$HOME/go")}"
INSTALL_DIR="${GOPATH}/bin"
BINARY="gospeed"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)  OS="linux" ;;
  Darwin*) OS="darwin" ;;
  FreeBSD*) OS="freebsd" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  armv7*|armhf)   ARCH="arm" ;;
  i386|i686)      ARCH="386" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Validate OS/arch combination
case "${OS}_${ARCH}" in
  darwin_386|darwin_arm|freebsd_arm|freebsd_386)
    echo "Unsupported combination: ${OS}/${ARCH}"
    exit 1 ;;
esac

# Get latest version
echo "Fetching latest version..."
VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/')"
if [ -z "$VERSION" ]; then
  echo "Failed to fetch latest version"
  exit 1
fi
echo "Latest version: v${VERSION}"

# Download
ARCHIVE="gospeed_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${ARCHIVE}..."
curl -fsSL "$URL" -o "${TMP}/${ARCHIVE}"

# Extract
tar -xzf "${TMP}/${ARCHIVE}" -C "$TMP"

# Install
mkdir -p "$INSTALL_DIR"
mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
chmod +x "${INSTALL_DIR}/${BINARY}"

# Add to PATH if not already there
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    SHELL_NAME="$(basename "$SHELL")"
    case "$SHELL_NAME" in
      zsh)  RC="$HOME/.zshrc" ;;
      bash) RC="$HOME/.bashrc" ;;
      *)    RC="$HOME/.profile" ;;
    esac
    echo "export PATH=\"\$PATH:${INSTALL_DIR}\"" >> "$RC"
    echo "Added ${INSTALL_DIR} to PATH in ${RC} (restart your terminal or run: source ${RC})"
    ;;
esac

echo "Installed gospeed v${VERSION} to ${INSTALL_DIR}/${BINARY}"

# Also install server if requested
if [ "${INSTALL_SERVER:-}" = "1" ] || [ "${1:-}" = "--server" ]; then
  SERVER_ARCHIVE="gospeed-server_${VERSION}_${OS}_${ARCH}.tar.gz"
  SERVER_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${SERVER_ARCHIVE}"
  echo "Downloading ${SERVER_ARCHIVE}..."
  curl -fsSL "$SERVER_URL" -o "${TMP}/${SERVER_ARCHIVE}"
  tar -xzf "${TMP}/${SERVER_ARCHIVE}" -C "$TMP"
  mv "${TMP}/gospeed-server" "${INSTALL_DIR}/gospeed-server"
  chmod +x "${INSTALL_DIR}/gospeed-server"
  echo "Installed gospeed-server v${VERSION} to ${INSTALL_DIR}/gospeed-server"
fi
