# System architecture

The system is split into two fully independent parts:

```
┌─────────────────────────────────┐        ┌──────────────────────────────────┐
│         Frontend (SPA)          │        │         Backend (Go/Echo)        │
│  Vue 3 · Reka UI · Tailwind v4  │◄──────►│  REST API · OpenAPI contract     │
│  Deployed: CDN / static host    │  HTTP  │  Deployed: independently         │
└─────────────────────────────────┘  JSON  └──────────────────────────────────┘
```

## Decoupling rules
- The backend is a pure JSON REST API. It has no knowledge of the frontend framework, renders no HTML, and serves no frontend assets in production.
- The frontend is a standalone SPA. It communicates with the backend exclusively over HTTP using the published API contract — no shared code, no server-side rendering, no backend templates.
- The API contract is the **only** interface between the two. Swapping Vue for React, Svelte, or any other framework requires zero backend changes.

## API contract
- All API endpoints are versioned under `/api/v1/`.
- The backend maintains an OpenAPI 3.x specification (e.g. `api/openapi.yaml`). This spec is the source of truth for the contract — not the implementation.
- Requests and responses are JSON. The backend always sets `Content-Type: application/json`.
- Errors follow a consistent envelope:
  ```json
  { "error": { "code": "not_found", "message": "resource not found" } }
  ```
- The backend sets CORS headers to allow the frontend origin. In development, allow `http://localhost:5173` (Vite default). In production, allow only the deployed frontend origin.
- Authentication tokens (JWT or opaque) are sent in the `Authorization: Bearer <token>` header — never in cookies, never in query strings.

## Backend responsibilities
- Expose REST endpoints under `/api/v1/`.
- Validate all input at the API boundary; return structured errors with appropriate HTTP status codes.
- Own all business logic, persistence, and external service integrations.
- Emit structured logs via `log/slog` for every request. The global slog logger is bridged to OpenTelemetry; traces, metrics, and logs are exported via OTLP gRPC in production (stdout in dev).

## Frontend responsibilities
- Fetch data from the backend API; own all rendering and UI state.
- Never embed business logic that belongs in the backend.
- All API calls go through a single typed API client layer (e.g. `src/api/`) — no raw `fetch` calls scattered across components.
- The API client layer is the only place that knows the backend base URL.

## Twelve-factor principles

Both the backend and frontend follow the [twelve-factor app](https://12factor.net/) methodology. The factors most directly relevant to day-to-day development are:

| Factor | Rule |
|--------|------|
| **III — Config** | All configuration that varies between environments lives in environment variables — never hardcoded. Backend: `PORT`, `FRONTEND_ORIGIN`, database URLs, external service credentials. Frontend: `VITE_API_URL`. No config files that must be edited per environment. |
| **IV — Backing services** | Databases, caches, queues, and external APIs are attached resources identified by a URL or connection string in the environment. The code makes no distinction between a local and a remote backing service. |
| **VI — Processes** | The application is stateless. No in-process session state, no sticky sessions, no local filesystem state that must survive a restart. Any state that needs to outlive a single request lives in a backing service. |
| **X — Dev/prod parity** | Keep development and production as similar as possible: same Go version, same Node version, same dependency versions, same environment variable names. Use the same code path for both; no `if isDev` branches that diverge behaviour. |
| **XI — Logs** | The application writes logs to stdout/stderr as a stream of events — it does not manage log files. The backend uses `log/slog` writing to stdout; log routing and storage is the infrastructure's responsibility. |

The remaining factors (codebase, dependencies, build/release/run, port binding, concurrency, disposability, admin processes) are enforced by the project structure and `build/Dockerfile` and do not require per-feature decisions.

## Development setup
- Backend and frontend are developed and run independently; they live in separate directories (e.g. `backend/` and `frontend/`).
- In development, Vite proxies `/api` requests to the running Go server to avoid CORS issues:
  ```ts
  // vite.config.ts
  server: { proxy: { '/api': 'http://localhost:8080' } }
  ```
- The frontend can also be run against a mock server (e.g. MSW) without a running backend.
