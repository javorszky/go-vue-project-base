# OpenTelemetry implementation TODO

This project currently uses `log/slog` for structured request logging.
Full OpenTelemetry support is planned after the initial punch-list fixes.

## What needs to be wired up

### 1. SDK and exporter dependencies

Add to `go.mod`:
- `go.opentelemetry.io/otel`
- `go.opentelemetry.io/otel/sdk`
- `go.opentelemetry.io/otel/sdk/trace`
- `go.opentelemetry.io/otel/sdk/metric`
- `go.opentelemetry.io/otel/exporters/stdout/stdouttrace` (dev)
- `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` (prod)
- `go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho`

### 2. Provider initialisation in `cmd/server/main.go`

- Create a `TracerProvider` and `MeterProvider` in `run()`.
- Register them as global providers.
- Shut them down (with flush) on context cancellation, before the server shuts down.
- Pass a `propagator` (W3C TraceContext + Baggage) via `otel.SetTextMapPropagator`.

### 3. Echo middleware in `internal/server/server.go`

- Add `otelecho.Middleware("service-name")` after `Recover` to create a span per request.
- The span automatically reads incoming W3C trace headers from upstream services.

### 4. slog → OTel log bridge

- Use `go.opentelemetry.io/contrib/bridges/otelslog` to route `slog` records into
  the OTel log pipeline, so the current `slog.LogAttrs` calls in the request logger
  appear in traces without a code change.

### 5. Configuration

Add to `internal/config/config.go`:
- `OTelEndpoint string` (env: `OTEL_EXPORTER_OTLP_ENDPOINT`) — empty means stdout exporter.
- `ServiceName string` (env: `OTEL_SERVICE_NAME`, default: binary name).

### 6. Docs to update when done

- `README.md` — replace the TODO line with the real description.
- `.ai/backend/guidelines.md` — same.
- `.ai/architecture/overview.md` — same.
- Remove this file.
