# go-vue-template

A GitHub template for production-ready full-stack applications: Go backend, Vue 3 frontend, strict tooling, and a clear path from single-binary to fully decoupled deployment.

---

## Quick start

### Prerequisites

| Tool | Version | Install |
|---|---|---|
| Go | 1.26+ | [go.dev/dl](https://go.dev/dl) |
| Node | 24+ | [nvm](https://github.com/nvm-sh/nvm) — `nvm install` reads `.nvmrc` |
| golangci-lint | latest | `brew install golangci-lint` |
| lefthook | 2.1+ | `brew install lefthook` |

### 1. Create a repo from this template

Click **Use this template → Create a new repository** on GitHub, then clone your new repo.

### 2. Initialise the project

```bash
bash scripts/init.sh
```

This replaces all placeholder strings (`your-org`, `your-project`) with your actual GitHub org and project name across `go.mod`, `CLAUDE.md`, and `frontend/package.json`.

### 3. Activate git hooks

```bash
lefthook install
```

Run this once per clone. From now on, linters run before every commit and tests run before every push.

### 4. Install frontend dependencies

```bash
cd frontend && npm install
```

### 5. Start developing

In two terminals:

```bash
# Terminal 1 — Go backend
go run ./cmd/server

# Terminal 2 — Vue frontend
cd frontend && npm run dev
```

The Vite dev server proxies `/api` requests to `http://localhost:8080`, so the frontend and backend talk to each other without any CORS configuration during development.

---

## Architecture

### System overview

The system is split into two fully independent halves:

```
Browser ◄──► Vue 3 SPA  ◄──HTTP/JSON──►  Go / Echo REST API
             (frontend/)                   (cmd/, internal/)
```

The backend is a pure JSON API — it renders no HTML and serves no frontend assets in production. The frontend is a standalone SPA that talks to the backend exclusively over the published API contract. Swapping one side requires zero changes to the other.

All endpoints are versioned under `/api/v1/`. Authentication uses `Authorization: Bearer <token>` — never cookies, never query strings.

---

### Backend — Go + Echo

- **Language:** Go 1.26, module path set by `scripts/init.sh`
- **Framework:** [Echo](https://echo.labstack.com/) — minimal, fast, request-lifecycle hooks
- **Observability:** OpenTelemetry traces, metrics, and structured logs on every request
- **Error envelope:** every API error follows `{ "error": { "code": "…", "message": "…" } }`
- **Entry point:** `cmd/server/main.go`; Echo setup and routes live in `internal/server/`

CORS is configured from day one, driven by a `FRONTEND_ORIGIN` environment variable. In the embedded deployment mode (see below) this is left empty; in the decoupled mode it is set to the frontend's origin. No code change is needed at migration time.

---

### Frontend — Vue 3 + Vite

- **Framework:** Vue 3 with `<script setup lang="ts">` — Options API and the `setup()` function form are both disallowed by ESLint
- **UI components:** [Reka UI](https://reka-ui.com/) — accessible, unstyled headless primitives
- **Styling:** Tailwind CSS v4 (Vite plugin, no PostCSS config required)
- **Build tool:** Vite 8; dev server proxies `/api` to the Go backend
- **Testing:** Vitest 4 with jsdom and `@vitest/coverage-v8` for coverage

All HTTP calls go through `src/api/` — no raw `fetch` calls elsewhere. This layer is the only place that reads `VITE_API_URL`. When empty (embedded mode) requests go to the same origin; when set (decoupled mode) requests go to the configured API URL. No component code changes are needed at migration time.

---

### Deployment — two modes

#### Mode 1: Embedded — single binary (current)

The compiled frontend is embedded into the Go binary via `//go:embed`. One container, one image, no inter-service networking.

```
Browser ──► scratch container
               └── Go binary
                    ├── /api/v1/* → business logic
                    └── /*        → embedded frontend (SPA fallback)
```

The multi-stage `build/Dockerfile` handles the full build:
1. **`frontend-deps`** — `npm ci`; cached until `package*.json` changes
2. **`frontend-builder`** — `npm run build`
3. **`go-builder`** — `golang:1.26-alpine`; embeds the SPA and compiles a static binary (`CGO_ENABLED=0 -trimpath -ldflags="-s -w"`)
4. **`scratch`** — copies only the binary and CA certificates; runs as UID 65534 (nobody)

The binary runs as UTC. If `time.LoadLocation()` for non-UTC zones is needed, see the comment in the `Dockerfile` scratch stage for the two-line addition required.

#### Mode 2: Decoupled — Caddy + scratch (future)

When traffic or team size justifies independent scaling, the frontend moves to its own container:

```
Browser ──► Caddy DHI container (dhi.io/caddy)
               ├── /api/* ──► scratch container (Go binary)
               └── /*      → /srv/ (SPA fallback → index.html)
```

The frontend image uses [Docker Hardened Images' Caddy](https://hub.docker.com/hardened-images/catalog/dhi/caddy) (Debian 13, zero-known CVEs). Create `deploy/Caddyfile` (template in `.ai/architecture/deployment.md`) to handle:
- SPA fallback routing (`try_files {path} /index.html`)
- API reverse proxy (`/api/*` → Go backend, internal network)
- Slowloris mitigation (read header/body timeouts)
- Request body and header size limits
- Security headers (HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy)
- Gzip/zstd compression

For volumetric DDoS, place Cloudflare or an equivalent CDN in front. Caddy's built-in limits are a backstop, not the primary defence. L7 rate limiting at the Caddy layer requires a custom `xcaddy` build with `caddy-ratelimit`; see `.ai/architecture/deployment.md` for details.

#### Migration from Mode 1 to Mode 2

The codebase is built with four constraints that make this migration mechanical:

| Constraint | Rule |
|---|---|
| `VITE_API_URL` | Always read from `import.meta.env.VITE_API_URL`; defaults to `""` (same-origin) |
| Centralised API client | All HTTP calls through `src/api/`; no raw `fetch` elsewhere |
| CORS always wired | Echo CORS middleware always active, reads `FRONTEND_ORIGIN` env var |
| `internal/server/static.go` isolated | All `//go:embed` code lives only in this file |

Migration is: delete `internal/server/static.go`, remove its one call-site, set `FRONTEND_ORIGIN` and `VITE_API_URL` in the environment. No other code changes. See `.ai/architecture/deployment.md` for the full playbook.

---

### Security tooling — three layers

Three tools cover distinct threat surfaces and do not duplicate each other:

| Tool | What it finds | Scope |
|---|---|---|
| **Trivy** | Known CVEs in dependencies | Go modules + npm packages |
| **govulncheck** | Known CVEs in Go dependencies, reachability-aware | Go modules only; no false positives from unused deps |
| **CodeQL** | Logic-level flaws: injection, unsafe data flows, path traversal, XSS | Go + JavaScript/TypeScript application code |

Trivy and govulncheck both cover Go dependency CVEs — they are complementary: Trivy is broader (includes npm), govulncheck is more actionable (reachability-filtered). CodeQL does not overlap with either; it works at the code-logic layer, not the dependency layer.

SARIF results from Trivy and CodeQL are uploaded to GitHub's **Security → Code scanning** tab. Workflow files are audited by [Zizmor](https://woodruffw.github.io/zizmor/).

---

### CI/CD — GitHub Actions

Two workflow files with a deliberate conceptual split:

**`ci.yml` — correctness** (runs on every push and PR):
- `go-lint`: golangci-lint + gofmt check
- `go-test`: `go test -race` with coverage artifact
- `frontend-lint`: ESLint + Prettier check + vue-tsc
- `frontend-test`: Vitest with coverage artifact

**`security.yml` — safety** (runs on push, PR, and weekly schedule):
- `trivy`: filesystem scan, SARIF upload to code scanning
- `govulncheck`: reachability-aware Go CVE scan
- `zizmor`: workflow file security audit, SARIF upload

**`codeql.yml` — semantic analysis** (runs on push, PR, and merge queue):
- `analyze (go)`: CodeQL for Go source
- `analyze (javascript-typescript)`: CodeQL for the Vue/TS frontend

All action versions are pinned to commit SHA digests and kept current by Renovate.

---

### Dependency management — Renovate

[Renovate](https://docs.renovatebot.com/) manages all dependency updates:

- **Go modules** (`go.mod`)
- **npm packages** (`frontend/package.json`)
- **GitHub Actions** (all `uses:` lines, SHA-pinned via `helpers:pinGitHubActionDigests`)

`stabilityDays` is set to `0` — updates land immediately on release with no waiting period. Renovate PRs go through the full CI pipeline before merge.

---

### Local tooling

#### Go linting — golangci-lint

`.golangci.yml` uses the `standard` default linter set plus:

`cyclop` `exhaustive` `gocritic` `gosec` `misspell` `nilerr` `nilnil` `revive` `unconvert` `unparam` `wrapcheck`

Imports are formatted by **gci** (three sections: stdlib / third-party / internal) and **gofumpt** (strict gofmt superset). Both run as formatters in the v2 config. `gosec`, `wrapcheck`, and `nilerr` are relaxed in `_test.go` files.

#### Frontend linting — ESLint

`eslint.config.ts` uses the flat config format with:
- `typescript-eslint` (type-aware rules, requires `tsconfig.json`)
- `eslint-plugin-vue` (Vue 3 `flat/recommended`)
- `eslint-plugin-vuejs-accessibility` (WCAG/ARIA rules)
- `eslint-config-prettier` (disables rules that conflict with Prettier)

Custom rules enforce `<script setup lang="ts">`, semantic self-closing tags, and `lang="ts"` on all script blocks.

#### Pre-commit and pre-push hooks — lefthook

`lefthook.yml` runs checks in parallel, triggered only when relevant files are staged:

| Hook | Trigger | Commands |
|---|---|---|
| `pre-commit` | `**/*.go` staged | golangci-lint, gofmt check |
| `pre-commit` | `frontend/**` staged | ESLint, Prettier check, vue-tsc |
| `pre-push` | always | `go test -race ./...`, Vitest |

Activate once after cloning with `lefthook install`.

---

### AI assistance — Claude Code

This repository is set up for productive AI-assisted development with [Claude Code](https://claude.ai/code).

The `.ai/` directory contains context documents loaded on demand. Populated so far:

| File | Content |
|---|---|
| `.ai/architecture/overview.md` | API contract, decoupling rules, CORS policy |
| `.ai/architecture/deployment.md` | Both deployment modes, Caddyfile, migration playbook |

Add these as the project grows (referenced in `CLAUDE.md` domain guidelines):

| File | Intended content |
|---|---|
| `.ai/backend/guidelines.md` | Go coding style, Echo patterns, OTel conventions |
| `.ai/frontend/guidelines.md` | Vue 3 conventions, component patterns, Tailwind usage |
| `.ai/workflows/common-tasks.md` | Cross-layer task sequencing |

Two MCP servers are required for full AI productivity — see `CLAUDE.md` for setup instructions.

---

## Licence

MIT — see [LICENSE](LICENSE).
