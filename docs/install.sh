#!/bin/sh
# gospeed installer — Linux, macOS, FreeBSD
# Usage: curl -fsSL https://gospeed.goozt.org/install.sh | bash
#        curl -fsSL https://gospeed.goozt.org/install.sh | bash -s -- --server
#
# Installs both gospeed (client) and gospeed-server when --server is passed;
# defaults to client-only.
set -eu

REPO="goozt/gospeed"
PROJECT_NAME="gospeed"
GOPATH="${GOPATH:-$(go env GOPATH 2>/dev/null || echo "$HOME/go")}"
INSTALL_DIR="${GOPATH}/bin"

BINARIES="gospeed"
if [ "${INSTALL_SERVER:-}" = "1" ] || [ "${1:-}" = "--server" ]; then
  BINARIES="gospeed gospeed-server"
fi

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
  aarch64|arm64) ARCH="arm64" ;;
  armv7*|armhf)  ARCH="arm" ;;
  i386|i686)     ARCH="386" ;;
  riscv64)       ARCH="riscv64" ;;
  s390x)         ARCH="s390x" ;;
  ppc64le)       ARCH="ppc64le" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Validate OS/arch combination (matches .goreleaser.yaml ignores)
case "${OS}_${ARCH}" in
  darwin_386|darwin_arm|darwin_riscv64|darwin_s390x|darwin_ppc64le|\
  freebsd_arm|freebsd_386|freebsd_riscv64|freebsd_s390x|freebsd_ppc64le)
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

# Download (single archive contains all binaries)
ARCHIVE="${PROJECT_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/v${VERSION}/checksums.txt"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${ARCHIVE}..."
curl -fsSL "$URL" -o "${TMP}/${ARCHIVE}"
curl -fsSL "$CHECKSUM_URL" -o "${TMP}/checksums.txt"

# Verify checksum
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$TMP" && grep " $ARCHIVE\$" checksums.txt | sha256sum -c -)
elif command -v shasum >/dev/null 2>&1; then
  (cd "$TMP" && grep " $ARCHIVE\$" checksums.txt | shasum -a 256 -c -)
else
  echo "warning: no sha256sum/shasum found, skipping checksum verification" >&2
fi

# Extract
tar -xzf "${TMP}/${ARCHIVE}" -C "$TMP"

# Install
mkdir -p "$INSTALL_DIR"
for bin in $BINARIES; do
  mv "${TMP}/${bin}" "${INSTALL_DIR}/${bin}"
  chmod +x "${INSTALL_DIR}/${bin}"
  echo "Installed ${bin} v${VERSION} to ${INSTALL_DIR}/${bin}"
done

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
