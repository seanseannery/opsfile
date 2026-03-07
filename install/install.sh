#!/usr/bin/env bash
set -euo pipefail

REPO="seanseannery/opsfile"
BINARY="ops"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── OS detection ────────────────────────────────────────────────────────────

OS="$(uname -s)"
case "$OS" in
  Linux*)   ASSET="ops_unix_v" ;;
  Darwin*)  ASSET="ops_darwin_v" ;;
  *)
    echo "Error: unsupported operating system: $OS" >&2
    echo "Windows users: download ops_v<version>.exe from https://github.com/$REPO/releases/latest" >&2
    exit 1
    ;;
esac

# ── Resolve latest release URL ───────────────────────────────────────────────

echo "Fetching latest release from github.com/$REPO ..."

DOWNLOAD_URL="$(
  curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep "browser_download_url" \
  | grep "$ASSET" \
  | sed 's/.*"browser_download_url": "\(.*\)"/\1/'
)"

if [ -z "$DOWNLOAD_URL" ]; then
  echo "Error: could not find a release asset matching '$ASSET'" >&2
  exit 1
fi

VERSION="$(echo "$DOWNLOAD_URL" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+')"
echo "Downloading $BINARY $VERSION for $OS ..."

# ── Download and install ─────────────────────────────────────────────────────

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

curl -fsSL "$DOWNLOAD_URL" -o "$TMP"
chmod +x "$TMP"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "$INSTALL_DIR/$BINARY"
else
  echo "Installing to $INSTALL_DIR (sudo required) ..."
  sudo mv "$TMP" "$INSTALL_DIR/$BINARY"
fi

echo "Installed: $(command -v $BINARY)"
echo "Version:   $($BINARY --version)"
