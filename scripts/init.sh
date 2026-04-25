#!/usr/bin/env bash
# One-time setup after creating a project from this template.
# Replaces placeholder strings with your actual project details.
#
# Usage: bash scripts/init.sh

set -euo pipefail

# ── Portable in-place sed (GNU sed needs no suffix arg; BSD/macOS sed needs '') ──
if sed --version 2>/dev/null | grep -q GNU; then
  sedi() { sed -i "$@"; }
else
  sedi() { sed -i '' "$@"; }
fi

# ── Guards ───────────────────────────────────────────────────────────────────
[[ -f go.mod ]] || { echo "Error: run this script from the repo root."; exit 1; }

grep -q "your-org/your-project" go.mod || { echo "Already initialised — go.mod no longer contains the placeholder."; exit 0; }

# ── Prompt ──────────────────────────────────────────────────────────────────
read -rp "GitHub org or username (e.g. acme): " GH_ORG
read -rp "Repository / project name (e.g. my-app): " PROJECT_NAME

[[ "$GH_ORG"      =~ ^[a-zA-Z0-9_-]+$ ]] || { echo "Invalid org name — only letters, numbers, hyphens, underscores."; exit 1; }
[[ "$PROJECT_NAME" =~ ^[a-zA-Z0-9_-]+$ ]] || { echo "Invalid project name — only letters, numbers, hyphens, underscores."; exit 1; }

MODULE="github.com/${GH_ORG}/${PROJECT_NAME}"

echo ""
echo "Will set:"
echo "  Go module  : ${MODULE}"
echo "  npm name   : ${PROJECT_NAME}-frontend"
echo ""
read -rp "Looks good? [y/N] " CONFIRM
[[ "${CONFIRM}" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 1; }

# ── Replacements ─────────────────────────────────────────────────────────────
sedi "s|github.com/your-org/your-project|${MODULE}|g" go.mod CLAUDE.md
sedi "s|your-project-frontend|${PROJECT_NAME}-frontend|g" frontend/package.json

# Replace the module path in every Go source file.
while IFS= read -r -d '' f; do
  sedi "s|github.com/your-org/your-project|${MODULE}|g" "$f"
done < <(find . -type f -name '*.go' -print0)

echo ""
echo "Done. Next steps:"
echo "  1. cd frontend && npm install   (generates a fresh package-lock.json)"
echo "  2. git add -A && git commit -m 'chore: initialise project from template'"
