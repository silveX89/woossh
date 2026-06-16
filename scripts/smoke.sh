#!/usr/bin/env bash
# woossh Smoke Test
# Validates the binary builds, reports version, lists hosts, and checks flags.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BINARY="$ROOT/woossh"
PASS=0
FAIL=0

green() { printf "\033[32m✓\033[0m %s\n" "$1"; }
red()   { printf "\033[31m✗\033[0m %s\n" "$1"; }

# ---- Clean build ----
echo "── Building woossh ──"
cd "$ROOT"
go build -o "$BINARY" . 2>&1
echo ""

# ---- Test 1: --version ----
echo "── Test 1: --version ──"
VERSION_OUTPUT=$("$BINARY" --version 2>&1)
if echo "$VERSION_OUTPUT" | grep -q "^woossh v"; then
    green "--version: $VERSION_OUTPUT"
    PASS=$((PASS + 1))
else
    red "--version: expected 'woossh v...', got '$VERSION_OUTPUT'"
    FAIL=$((FAIL + 1))
fi

# ---- Test 2: -v (short flag) ----
echo "── Test 2: -v ──"
V_OUTPUT=$("$BINARY" -v 2>&1)
if echo "$V_OUTPUT" | grep -q "^woossh v"; then
    green "-v: $V_OUTPUT"
    PASS=$((PASS + 1))
else
    red "-v: expected 'woossh v...', got '$V_OUTPUT'"
    FAIL=$((FAIL + 1))
fi

# ---- Test 3: --list-hosts ----
echo "── Test 3: --list-hosts ──"
HOSTS_OUTPUT=$("$BINARY" --list-hosts 2>&1)
if [ -n "$HOSTS_OUTPUT" ]; then
    COUNT=$(echo "$HOSTS_OUTPUT" | wc -l)
    green "--list-hosts: $COUNT host(s) found"
    PASS=$((PASS + 1))
else
    red "--list-hosts: empty output"
    FAIL=$((FAIL + 1))
fi

# ---- Test 4: Dry-run direct mode ----
echo "── Test 4: Direct mode dry-run (/d) ──"
FIRST_HOST=$(echo "$HOSTS_OUTPUT" | head -1)
if [ -n "$FIRST_HOST" ]; then
    DRY_OUTPUT=$("$BINARY" /d "$FIRST_HOST" 2>&1)
    if echo "$DRY_OUTPUT" | grep -q "^ssh "; then
        green "/d $FIRST_HOST: $DRY_OUTPUT"
        PASS=$((PASS + 1))
    else
        red "/d $FIRST_HOST: expected 'ssh ...', got '$DRY_OUTPUT'"
        FAIL=$((FAIL + 1))
    fi
fi

# ---- Test 5: go vet ----
echo "── Test 5: go vet ──"
VET_OUTPUT=$(cd "$ROOT" && go vet ./... 2>&1)
if [ -z "$VET_OUTPUT" ]; then
    green "go vet: clean"
    PASS=$((PASS + 1))
else
    red "go vet has issues:" "$VET_OUTPUT"
    FAIL=$((FAIL + 1))
fi

# ---- Summary ----
echo ""
echo "══ Results: $PASS passed, $FAIL failed ══"
if [ "$FAIL" -gt 0 ]; then
    exit 1
fi