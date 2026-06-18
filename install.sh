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

git clone --depth=1 --branch exp "$REPO" "$TMP/woossh"
cd "$TMP/woossh"

# Strip local-only plugin replace directives: fresh install has no ~/.local/woossh-plugins
go mod edit -dropreplace github.com/silveX89/woossh-plugins/tmux 2>/dev/null || true
go mod edit -droprequire github.com/silveX89/woossh-plugins/tmux 2>/dev/null || true
# Remove stale registry_gen.go — will be regenerated when user installs a plugin
rm -f registry_gen.go

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
