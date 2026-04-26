# Codebase index

Keep this file up to date after every code change. Update the relevant section
whenever a signature changes, a file is added or removed, or a responsibility
shifts. Do not let it drift from the actual code.

---

## Go packages

### `cmd/server` — process entry point
`main.go`, `otel.go`

| Symbol | Signature | Purpose |
|--------|-----------|---------|
| `main` | `func main()` | Calls `run()`; exits non-zero on error |
| `run` | `func run() error` | Loads config, sets up OTel, wires signal context, starts server |
| `setupOTel` | `func setupOTel(ctx context.Context, cfg config.Config) (func(), error)` | Initialises trace/metric/log providers, registers globals, bridges slog; returns flush-shutdown func |
| `exporterSet` | `struct{ tracer SpanExporter; reader Reader; logger Exporter }` | Groups the three signal exporters for a single transport |
| `buildExporters` | `func buildExporters(ctx, cfg) (exporterSet, error)` | Single dispatch: stdout / grpc / http based on cfg |
| `buildTracerProvider` | `func buildTracerProvider(exporter, res, ratio) *TracerProvider` | Wraps exporter in a TracerProvider; no transport logic |
| `buildMeterProvider` | `func buildMeterProvider(reader, res) *MeterProvider` | Wraps pre-built reader in a MeterProvider |
| `buildLoggerProvider` | `func buildLoggerProvider(exporter, res) *LoggerProvider` | Wraps exporter in a LoggerProvider |
| `checkOTelConnectivity` | `func checkOTelConnectivity(endpoint, transport string) error` | Dispatches to gRPC or HTTP probe |
| `buildStdoutExporters` | `func buildStdoutExporters(cfg) (exporterSet, error)` | `otel_exporters_stdout.go` — stdout exporters for dev |
| `buildGRPCExporters` | `func buildGRPCExporters(ctx, cfg) (exporterSet, error)` | `otel_exporters_grpc.go` — OTLP gRPC exporters |
| `checkOTelGRPC` | `func checkOTelGRPC(endpoint string) error` | `otel_exporters_grpc.go` — gRPC protocol-level connectivity probe |
| `buildHTTPExporters` | `func buildHTTPExporters(ctx, cfg) (exporterSet, error)` | `otel_exporters_http.go` — OTLP HTTP exporters |
| `checkOTelHTTP` | `func checkOTelHTTP(endpoint string) error` | `otel_exporters_http.go` — HTTP HEAD connectivity probe |

**To change:** startup/shutdown sequence → `run()`. OTel provider config → `otel.go`. Exporter construction → `otel_exporters_*.go`. Process exit code → `main()`.

---

### `internal/config` — runtime configuration
`config.go`, `config_test.go`

| Symbol | Signature | Purpose |
|--------|-----------|---------|
| `Config` | `struct{ Domain string; FrontendOrigin string; OTelEndpoint string; OTelTransport string; ServiceName string; OTelExportInterval time.Duration; OTelSamplingRatio float64; Port int }` | All runtime config; parsed from env vars |
| `Load` | `func Load() (Config, error)` | Parses OS environment; call once at startup |
| `LoadFrom` | `func LoadFrom(vars map[string]string) (Config, error)` | Parses from an in-memory map; use in tests instead of `os.Setenv` |

Env vars: `PORT` (default `8080`), `DOMAIN` (default `localhost`), `FRONTEND_ORIGIN` (optional), `OTEL_EXPORTER_OTLP_ENDPOINT` (empty → stdout exporters), `OTEL_EXPORTER_OTLP_PROTOCOL` (`grpc` or `http`, default `grpc`), `OTEL_SERVICE_NAME` (default `hoplink`), `OTEL_SAMPLING_RATIO` (float 0–1, default `1.0`), `OTEL_METRIC_EXPORT_INTERVAL` (Go duration, default `15s`).

**To change:** add/remove a config variable → `Config` struct + this table.  
**Rule:** never call `os.Getenv` outside this package (enforced by golangci-lint `forbidigo`).

---

### `internal/server` — HTTP server
`server.go`, `middleware.go`, `status.go`, `static.go`, `server_test.go`

| Symbol | Signature | Purpose |
|--------|-----------|---------|
| `Server` | `struct{ echo *echo.Echo; addr string }` | Wraps Echo and the listen address |
| `New` | `func New(cfg config.Config, gitSHA, buildTime string) *Server` | Creates Echo instance, registers middleware and routes |
| `(*Server).Start` | `func (s *Server) Start(ctx context.Context) error` | Runs server until `ctx` is cancelled, then shuts down gracefully (10 s timeout) |
| `(*Server).Handler` | `func (s *Server) Handler() http.Handler` | Returns the Echo instance as `http.Handler`; use in tests with `httptest` |
| `otelMiddleware` | `func otelMiddleware(serviceName string) echo.MiddlewareFunc` | Custom Echo v5 OTel middleware: extracts W3C trace context, creates server span, records HTTP method/path/status |
| `healthHandler` | `func healthHandler(c *echo.Context) error` | `GET /api/v1/health` → `{"status":"ok"}` |
| `statusHandler` | `func statusHandler(gitSHA, buildTime string) echo.HandlerFunc` | `GET /api/v1/status` → `{"status":"ok","git_sha":"…","build_time":"…"}` |
| `registerStatic` | `func registerStatic(e *echo.Echo)` | Serves embedded Vue SPA (Mode 1 only; delete this file to move to Mode 2) |

Note: `otelecho` (the contrib package) targets Echo v4 and cannot be used here. `otelMiddleware` is the Echo v5 replacement.

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
| `HealthResponse` | `interface{ status: string }` | Response shape for `/api/v1/health` |
| `checkHealth` | `function checkHealth(): Promise<HealthResponse>` | `GET /api/v1/health` |
| `StatusResponse` | `interface{ status: string; git_sha: string; build_time: string }` | Response shape for `/api/v1/status` |
| `getStatus` | `function getStatus(): Promise<StatusResponse>` | `GET /api/v1/status` |

**To add an API call:** add a function here, typed against the OpenAPI contract.

---

### `env.d.ts` — environment variable types
Declares `ImportMetaEnv` so `import.meta.env.VITE_*` variables are typed.  
**To add a frontend env var:** add a `readonly VITE_FOO?: string` entry here.

---

## Navigation guide

| I want to… | Go to… |
|------------|--------|
| Read the API contract | `api/openapi.yaml` |
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
