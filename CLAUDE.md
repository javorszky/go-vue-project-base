# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Module: `github.com/your-org/your-project`  
Language: Go 1.26.2

> **New project?** Run `bash scripts/init.sh` once after cloning to replace placeholder names with your actual module path and project name.

The project has a minimal working skeleton: Echo backend with a health endpoint and embedded Vue 3 frontend. Grow the guidelines below as conventions solidify.

## Required MCP servers

Both servers are configured user-scope via `claude mcp add -s user` and stored in `~/.claude.json`. They must be present for productive work on this project.

| Server | Purpose | Requirement |
|--------|---------|-------------|
| **gopls** | Go language server — definitions, references, `go test`, `go mod tidy`, `govulncheck` | `gopls` v0.20.0+ on `PATH` (`go install golang.org/x/tools/gopls@latest`) |
| **context7** | Version-aware docs for Vue, Tailwind, Echo, OTel, and the rest of the stack | Node.js / `npx` on `PATH`; package downloads automatically on first use |

If either server is missing, add it via the CLI (user-scoped so it applies to all projects):
```bash
# gopls — requires gopls v0.20.0+ on PATH
claude mcp add -s user -- gopls gopls mcp

# context7 — Node is typically via nvm; use the full path to npx and inject PATH
claude mcp add -s user \
  -e "PATH=/Users/<you>/.nvm/versions/node/<version>/bin:/usr/local/bin:/usr/bin:/bin" \
  -- context7 /Users/<you>/.nvm/versions/node/<version>/bin/npx -y @upstash/context7-mcp
```
Servers are stored in `~/.claude.json`. Verify with `claude mcp list`.

## Imperatives for adding dependencies and using tools

These rules apply every time a new library, package, or external tool is introduced, regardless of layer.

### 1. Always use the latest stable version — and verify compatibility

Before adding any dependency or tool, look up its current latest stable release:
- **Go**: check pkg.go.dev or run `go list -m -json` for the latest tagged version.
- **npm**: check npmjs.com or run `npm info <pkg> version`.
- **GitHub Actions** (workflow files): every `uses: owner/repo@version` line must use the latest stable tag. Use the **GitHub MCP server** (`get_latest_release` or `list_tags`) to look up the current release for each action before writing or updating a workflow file. Apply this to all actions in the file, not just the one being added.
- **CLI tools** (linters, scanners, etc.): check the project's GitHub releases or official install docs.

Do not assume the version in your training data is current — it is not. Use the version you looked up, not a guess.

After identifying the latest stable version, verify it is compatible with the existing stack before committing to it:
- If compatible: use it, and update any dependent code or config to match.
- If not compatible: find the latest version that is compatible, tell the user which version was chosen and why the latest could not be used, and ask for feedback before proceeding.

### 2. Read the official documentation before reading source code

Before inferring how a library or tool works from its source or from examples in this repo:
1. Fetch the official documentation using the **context7 MCP server** (preferred — version-aware).
2. If the tool is installed locally, run its `--help` or `help` subcommand and read the output.
3. Only fall back to reading source code or examples when docs are absent or incomplete.

This prevents using deprecated APIs, outdated configuration syntax, or patterns that were valid in an older version but have since changed.

## Codebase index

**[`.ai/index.md`](.ai/index.md)** — package-by-package map of every Go and frontend file: exported symbols, signatures, purposes, and a navigation guide for common tasks. Read it at the start of any session. **Keep it current:** after every code change that adds, removes, or renames a symbol or file, update the relevant section of `.ai/index.md` before finishing the task.

## Domain guidelines

Load only the file(s) relevant to the task at hand.

- **Overall system design, API contract, decoupling rules**: see [`.ai/architecture/overview.md`](.ai/architecture/overview.md)
- **Deployment modes, Docker images, Caddy config, migration playbook**: see [`.ai/architecture/deployment.md`](.ai/architecture/deployment.md)
- **Backend (Go, Echo, OTel, coding style, context, shutdown)**: see [`.ai/backend/guidelines.md`](.ai/backend/guidelines.md)
- **Frontend (Vue 3, Reka UI, Tailwind CSS v4, Vite)**: see [`.ai/frontend/guidelines.md`](.ai/frontend/guidelines.md)
- **Orchestration — task sequencing and cross-layer workflows**: see [`.ai/workflows/common-tasks.md`](.ai/workflows/common-tasks.md)
