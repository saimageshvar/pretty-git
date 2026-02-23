#!/usr/bin/env bash
set -euo pipefail

# pretty-git installer
# Usage: curl -sSL https://raw.githubusercontent.com/saimageshvar/pretty-git/main/install.sh | sudo bash

REPO="saimageshvar/pretty-git"
BINARY="pgit"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── helpers ────────────────────────────────────────────────────────────────────

info()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
ok()    { printf '\033[1;32m  ✓\033[0m %s\n' "$*"; }
err()   { printf '\033[1;31mError:\033[0m %s\n' "$*" >&2; exit 1; }

# ── detect OS / arch ───────────────────────────────────────────────────────────

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) err "Unsupported architecture: $ARCH" ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) err "Unsupported OS: $OS" ;;
esac

# ── pick download tool ─────────────────────────────────────────────────────────

if command -v curl &>/dev/null; then
  download() { curl -sSfL "$1"; }
  download_to() { curl -sSfL -o "$2" "$1"; }
elif command -v wget &>/dev/null; then
  download() { wget -qO- "$1"; }
  download_to() { wget -qO "$2" "$1"; }
else
  err "curl or wget is required. Install one with: sudo apt install curl"
fi

# ── fetch latest version ───────────────────────────────────────────────────────

info "Fetching latest release..."
VERSION="$(download "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)"

[ -n "$VERSION" ] || err "Could not determine latest version. Check your internet connection."
ok "Latest version: $VERSION"

# ── download & install ─────────────────────────────────────────────────────────

ARCHIVE="pgit_${VERSION#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$ARCHIVE"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

info "Downloading $ARCHIVE..."
download_to "$TMP/$ARCHIVE" "$URL"

info "Installing to $INSTALL_DIR/$BINARY..."
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
install -m 755 "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"

ok "$BINARY $VERSION installed to $INSTALL_DIR/$BINARY"

# symlink pretty-git → pgit
ln -sf "$INSTALL_DIR/$BINARY" "$INSTALL_DIR/pretty-git"
ok "Symlinked pretty-git → pgit"

echo ""
echo "  Try it:  pgit log           (or: pretty-git log)"
echo "           pgit branch        (or: pretty-git branch)"
echo "           pgit checkout      (or: pretty-git checkout)"
echo ""
echo "  To update later, re-run this script."
