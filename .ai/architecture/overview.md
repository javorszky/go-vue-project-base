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
- Emit OTel traces, metrics, and logs for every request.

## Frontend responsibilities
- Fetch data from the backend API; own all rendering and UI state.
- Never embed business logic that belongs in the backend.
- All API calls go through a single typed API client layer (e.g. `src/api/`) — no raw `fetch` calls scattered across components.
- The API client layer is the only place that knows the backend base URL.

## Development setup
- Backend and frontend are developed and run independently; they live in separate directories (e.g. `backend/` and `frontend/`).
- In development, Vite proxies `/api` requests to the running Go server to avoid CORS issues:
  ```ts
  // vite.config.ts
  server: { proxy: { '/api': 'http://localhost:8080' } }
  ```
- The frontend can also be run against a mock server (e.g. MSW) without a running backend.
