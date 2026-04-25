# Codebase index

Keep this file up to date after every code change. Update the relevant section
whenever a signature changes, a file is added or removed, or a responsibility
shifts. Do not let it drift from the actual code.

---

## Go packages

### `cmd/server` — process entry point
`main.go`

| Symbol | Signature | Purpose |
|--------|-----------|---------|
| `main` | `func main()` | Calls `run()`; exits non-zero on error |
| `run` | `func run() error` | Loads config, wires signal context, starts server |

**To change:** startup/shutdown sequence → `run()`. Process exit code → `main()`.

---

### `internal/config` — runtime configuration
`config.go`, `config_test.go`

| Symbol | Signature | Purpose |
|--------|-----------|---------|
| `Config` | `struct{ Domain string; FrontendOrigin string; Port int }` | All runtime config; parsed from env vars |
| `Load` | `func Load() (Config, error)` | Parses OS environment; call once at startup |
| `LoadFrom` | `func LoadFrom(vars map[string]string) (Config, error)` | Parses from an in-memory map; use in tests instead of `os.Setenv` |

Env vars: `PORT` (default `8080`), `DOMAIN` (default `localhost`), `FRONTEND_ORIGIN` (optional).

**To change:** add/remove a config variable → `Config` struct + this table.  
**Rule:** never call `os.Getenv` outside this package (enforced by golangci-lint `forbidigo`).

---

### `internal/server` — HTTP server
`server.go`, `static.go`, `server_test.go`

| Symbol | Signature | Purpose |
|--------|-----------|---------|
| `Server` | `struct{ echo *echo.Echo; cfg config.Config }` | Wraps Echo and its config |
| `New` | `func New(cfg config.Config) *Server` | Creates Echo instance, registers middleware and routes |
| `(*Server).Start` | `func (s *Server) Start(ctx context.Context) error` | Runs server until `ctx` is cancelled, then shuts down gracefully (10 s timeout) |
| `(*Server).Handler` | `func (s *Server) Handler() http.Handler` | Returns the Echo instance as `http.Handler`; use in tests with `httptest` |
| `healthHandler` | `func healthHandler(c *echo.Context) error` | `GET /api/v1/health` → `{"status":"ok"}` |
| `registerStatic` | `func registerStatic(e *echo.Echo)` | Serves embedded Vue SPA (Mode 1 only; delete this file to move to Mode 2) |

**To add a route:** `New()` in `server.go`.  
**To change graceful timeout:** `Start()` in `server.go`.  
**To migrate to decoupled deployment:** delete `static.go` and remove its call in `New()`.

---

### `internal/ui` — embedded frontend assets
`embed.go`

| Symbol | Signature | Purpose |
|--------|-----------|---------|
| `FS` | `var FS embed.FS` | Compiled Vue SPA embedded at build time via `//go:embed all:dist` |

**To populate:** `npm run build` in `frontend/`; output goes to `internal/ui/dist/`.

---

## Frontend (`frontend/src/`)

### `main.ts` — app entry point
Mounts the Vue app onto `#app`. No exports.

---

### `App.vue` — root component
Calls `checkHealth()` on mount; displays a coloured dot indicating API reachability.  
**To change the landing page:** edit this file.

---

### `api/client.ts` — typed API client
All `fetch` calls live here. No raw `fetch` elsewhere.

| Export | Signature | Purpose |
|--------|-----------|---------|
| `HealthResponse` | `interface{ status: string }` | Response shape for health endpoint |
| `checkHealth` | `function checkHealth(): Promise<HealthResponse>` | `GET /api/v1/health` |

**To add an API call:** add a function here, typed against the OpenAPI contract.

---

### `env.d.ts` — environment variable types
Declares `ImportMetaEnv` so `import.meta.env.VITE_*` variables are typed.  
**To add a frontend env var:** add a `readonly VITE_FOO?: string` entry here.

---

## Navigation guide

| I want to… | Go to… |
|------------|--------|
| Add a new API route | `internal/server/server.go` → `New()` |
| Add a config variable | `internal/config/config.go` → `Config` struct; update this index |
| Change startup / shutdown logic | `cmd/server/main.go` → `run()` |
| Change graceful shutdown timeout | `internal/server/server.go` → `Start()` |
| Test config parsing | Use `config.LoadFrom(map[string]string{...})` |
| Add a frontend API call | `frontend/src/api/client.ts` |
| Add a frontend env var | `frontend/src/env.d.ts` → `ImportMetaEnv` |
| Change the landing page UI | `frontend/src/App.vue` |
| Change CORS origin | Set `FRONTEND_ORIGIN` env var — no code change needed |
| Migrate to decoupled deployment | Delete `internal/server/static.go`; remove its call in `New()` |
| Add / change a golangci-lint rule | `.golangci.yml` |
| Add a CI job | `.github/workflows/ci.yml` |
| Add a security scan | `.github/workflows/security.yml` |
