#!/usr/bin/env bash
# One-time setup after creating a project from this template.
# Replaces placeholder strings with your actual project details.
#
# Usage: bash scripts/init.sh
# Note: uses BSD sed syntax (macOS). On Linux, replace `sed -i ''` with `sed -i`.

set -euo pipefail

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
sed -i '' "s|github.com/your-org/your-project|${MODULE}|g" go.mod
sed -i '' "s|github.com/your-org/your-project|${MODULE}|g" CLAUDE.md
sed -i '' "s|your-project-frontend|${PROJECT_NAME}-frontend|g" frontend/package.json

echo ""
echo "Done. Next steps:"
echo "  1. cd frontend && npm install   (generates a fresh package-lock.json)"
echo "  2. git add -A && git commit -m 'chore: initialise project from template'"
