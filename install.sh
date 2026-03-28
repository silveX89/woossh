#!/usr/bin/env bash
set -e

REPO="https://github.com/silveX89/woossh"
INSTALL_DIR="/usr/local/bin"
BINARY="woossh"

# Check for Go
if ! command -v go &>/dev/null; then
    echo "Error: Go is not installed. Install it from https://go.dev/dl and re-run this script."
    exit 1
fi

echo "Installing woossh..."

# Clone to a temp dir, build, install
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

git clone --depth=1 "$REPO" "$TMP/woossh"
cd "$TMP/woossh"
go build -o "$BINARY" .

# Install binary
if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY" "$INSTALL_DIR/$BINARY"
else
    echo "Needs sudo to write to $INSTALL_DIR"
    sudo mv "$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo "woossh installed to $INSTALL_DIR/$BINARY"
echo "Run: woossh"

# Optional: shell completion hint
echo ""
echo "Shell completion (bash) — add to ~/.bashrc:"
echo '  complete -C "woossh --list-hosts" woossh'
