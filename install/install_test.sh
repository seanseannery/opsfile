#!/usr/bin/env bash
# install_test.sh — end-to-end smoke test for both install methods.
# Requires network access and a published GitHub release.
# Brew cleanup is performed automatically on exit.

set -euo pipefail

PASS=0
FAIL=0

BREW_TAPPED=false
BREW_INSTALLED=false

CURL_TMP_DIR=""

pass() { echo "  PASS: $1"; ((PASS++)) || true; }
fail() { echo "  FAIL: $1" >&2; ((FAIL++)) || true; }

cleanup() {
  echo ""
  echo "=== cleanup ==="
  if [ -n "$CURL_TMP_DIR" ] && [ -d "$CURL_TMP_DIR" ]; then
    rm -rf "$CURL_TMP_DIR"
    echo "  curl tmp dir removed"
  fi
  if [ "$BREW_INSTALLED" = true ]; then
    brew uninstall opsfile 2>/dev/null && echo "  brew uninstall: done" || echo "  brew uninstall: skipped"
  fi
  if [ "$BREW_TAPPED" = true ]; then
    brew untap seanseannery/opsfile 2>/dev/null && echo "  brew untap: done" || echo "  brew untap: skipped"
  fi
}
trap cleanup EXIT

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── curl install test ─────────────────────────────────────────────────────────

echo ""
echo "=== curl install test ==="

CURL_TMP_DIR="$(mktemp -d)"
if INSTALL_DIR="$CURL_TMP_DIR" bash "$SCRIPT_DIR/install.sh"; then
  if "$CURL_TMP_DIR/ops" --version > /dev/null 2>&1; then
    pass "ops binary installed and responds to --version"
  else
    fail "ops binary installed but --version failed"
  fi
else
  fail "curl install script exited non-zero"
fi

# ── brew install test ─────────────────────────────────────────────────────────

echo ""
echo "=== brew install test ==="

if ! command -v brew > /dev/null 2>&1; then
  echo "  SKIP: brew not found"
else
  if brew tap seanseannery/opsfile https://github.com/seanseannery/opsfile; then
    BREW_TAPPED=true
    if brew install seanseannery/opsfile; then
      BREW_INSTALLED=true
      BREW_OPS="$(brew --prefix)/bin/ops"
      if "$BREW_OPS" --version > /dev/null 2>&1; then
        pass "ops installed via brew and responds to --version"
      else
        fail "ops installed via brew but --version failed"
      fi
    else
      fail "brew install seanseannery/opsfile failed"
    fi
  else
    fail "brew tap seanseannery/opsfile failed"
  fi
fi

# ── summary ──────────────────────────────────────────────────────────────────

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
