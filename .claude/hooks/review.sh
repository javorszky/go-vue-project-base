#!/usr/bin/env bash
# Runs after Claude stops. Guards on git changes, then feeds linter output
# back to Claude (via asyncRewake exit 2) for logical + security review.
# Only fires when the working tree has changed since the last review.

SENTINEL="${TMPDIR:-/tmp}/.hoplink-review-$(git rev-parse --show-toplevel 2>/dev/null | md5 -q)"

# Guard: nothing changed → silent exit
changes=$(git status --porcelain 2>/dev/null)
if [ -z "$changes" ]; then
  rm -f "$SENTINEL"
  exit 0
fi

# Guard: same diff as last review → silent exit
current_hash=$(echo "$changes" | md5 -q)
if [ -f "$SENTINEL" ] && [ "$(cat "$SENTINEL")" = "$current_hash" ]; then
  exit 0
fi

# Record this diff so we don't re-review it
echo "$current_hash" > "$SENTINEL"

echo "=== Changed files ==="
echo "$changes"
echo ""

# Go linters — only when .go files are present in the diff
if echo "$changes" | grep -q '\.go$'; then
  echo "=== golangci-lint ==="
  golangci-lint run ./... 2>&1 || true
  echo ""
  echo "=== govulncheck ==="
  govulncheck ./... 2>&1 || true
  echo ""
fi

echo "---"
echo "Review the changed files and any linter output above."
echo "Check for: logical correctness and bugs, simplification opportunities,"
echo "and security issues not already caught by the linters."
echo "Group findings by severity: critical / warning / suggestion."
echo "If nothing to add beyond what the linters already reported, say so briefly."

exit 2
