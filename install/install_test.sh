#!/usr/bin/env bash
# install_test.sh — end-to-end smoke test for both install methods.
# Requires network access and a published GitHub release.
# Brew cleanup is performed automatically on exit.

set -euo pipefail

PASS=0
FAIL=0

BREW_TAPPED=false
BREW_INSTALLED=false

CURL_INSTALL_PATH="/usr/local/bin/ops"

pass() { echo "  PASS: $1"; ((PASS++)) || true; }
fail() { echo "  FAIL: $1" >&2; ((FAIL++)) || true; }

cleanup_brew() {
  echo ""
  echo "=== cleanup ==="
  if [ "$BREW_INSTALLED" = true ]; then
    brew uninstall opsfile 2>/dev/null && echo "  brew uninstall: done" || echo "  brew uninstall: skipped"
  fi
  if [ "$BREW_TAPPED" = true ]; then
    brew untap seanseannery/opsfile 2>/dev/null && echo "  brew untap: done" || echo "  brew untap: skipped"
  fi
}
trap cleanup_brew EXIT

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── curl install test ─────────────────────────────────────────────────────────

echo ""
echo "=== curl install test ==="

if bash "$SCRIPT_DIR/install.sh"; then
  if ops --version > /dev/null 2>&1; then
    pass "ops binary installed and responds to --version"
  else
    fail "ops binary installed but --version failed"
  fi
else
  fail "curl install script exited non-zero"
fi

# Remove curl-installed binary before brew test to avoid path conflicts.
if [ -f "$CURL_INSTALL_PATH" ]; then
  rm -f "$CURL_INSTALL_PATH" 2>/dev/null || sudo rm -f "$CURL_INSTALL_PATH"
  pass "curl cleanup: removed $CURL_INSTALL_PATH"
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
