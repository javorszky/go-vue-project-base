# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Module: `github.com/your-org/your-project`  
Language: Go 1.26.2

> **New project?** Run `bash scripts/init.sh` once after cloning to replace placeholder names with your actual module path and project name.

This project is in its earliest stage — only `go.mod` exists. Architecture and conventions will grow here as the codebase develops.

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

## Domain guidelines

Load only the file(s) relevant to the task at hand.

- **Overall system design, API contract, decoupling rules**: see [`.ai/architecture/overview.md`](.ai/architecture/overview.md)
- **Backend (Go, Echo, OTel, coding style, context, shutdown)**: see [`.ai/backend/guidelines.md`](.ai/backend/guidelines.md)
- **Frontend (Vue 3, Reka UI, Tailwind CSS v4, Vite)**: see [`.ai/frontend/guidelines.md`](.ai/frontend/guidelines.md)
- **Orchestration — task sequencing and cross-layer workflows**: see [`.ai/workflows/common-tasks.md`](.ai/workflows/common-tasks.md)
