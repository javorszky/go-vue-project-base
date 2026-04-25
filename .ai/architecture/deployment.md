# Deployment architecture

This document covers the two deployment modes for this project and the constraints that make migrating between them mechanical. Read this before any work touching Docker, static file serving, CORS, or the `VITE_API_URL` / `FRONTEND_ORIGIN` environment variables.

---

## Phase 1: Embedded — go:embed, single binary (current)

The Go binary embeds the compiled frontend via `//go:embed`. One container, one image, zero inter-service networking.

```
Browser ──► [scratch container]
               └── Go binary (Echo)
                    ├── /api/v1/* → business logic
                    └── /*        → embedded dist/ files (SPA fallback)
```

**Docker image** — `FROM scratch`; copy the statically linked Go binary:

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o server ./cmd/server

FROM scratch
COPY --from=builder /app/server /server
ENTRYPOINT ["/server"]
```

**Environment variables at runtime:**

| Variable | Value | Purpose |
|---|---|---|
| `FRONTEND_ORIGIN` | _(empty)_ | No cross-origin CORS needed; frontend and API share the same origin |
| `PORT` | `8080` | HTTP listen port |

**Frontend build** — `VITE_API_URL` is not set (or empty). The `src/api/` client uses `""` as the base URL, producing same-origin requests (`/api/v1/…`).

---

## Phase 2: Decoupled — Caddy + scratch (future target)

Frontend static files are served by a dedicated Caddy container. The Go binary runs independently. Caddy proxies `/api/*` to the Go container.

```
Browser ──► [Caddy container: dhi.io/caddy]
               ├── /api/* ──► [scratch container: Go binary]
               └── /*      → /srv/dist/ (SPA fallback → index.html)
```

### Go backend Dockerfile

Same as Phase 1 — unchanged.

### Frontend Dockerfile

```dockerfile
FROM node:24-alpine AS builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
ARG VITE_API_URL
RUN npm run build

# Caddy DHI hardened image (Debian 13, zero-known CVEs)
# Digest is current as of 2026-04-25 — verify and update before deploying.
FROM dhi.io/caddy:2.11.x@sha256:7c337b4a5b2a5bb3f5d4d44387cd3a54b883e08a548c4c1756aee917e9064b74
COPY --from=builder /app/dist /srv
COPY deploy/Caddyfile /etc/caddy/Caddyfile
```

### Caddyfile (`deploy/Caddyfile`)

```caddyfile
{
    # Slowloris and header-flood mitigation — built into standard Caddy.
    servers {
        timeouts {
            read_header 5s
            read_body   30s
            write       30s
            idle        2m
        }
        max_header_size 64KB
    }
}

:80 {
    # Reject oversized request bodies before they reach the backend.
    request_body {
        max_size 10MB
    }

    # Bandwidth savings; also reduces response time under load.
    encode gzip zstd

    # Security headers applied to every response.
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Frame-Options            "DENY"
        X-Content-Type-Options     "nosniff"
        Referrer-Policy            "strict-origin-when-cross-origin"
        Permissions-Policy         "geolocation=(), microphone=(), camera=()"
        -Server
    }

    root * /srv

    # route enforces directive order: API proxy runs before the SPA fallback.
    route {
        # Caddy sets X-Forwarded-For automatically (appends to chain).
        # X-Real-IP carries the IP of whoever connected to Caddy — useful
        # when Caddy is the first hop, but is the CDN edge IP when behind
        # Cloudflare. In that case, read CF-Connecting-IP on the backend
        # instead, or configure trusted_proxies here for the CDN IP ranges.
        reverse_proxy /api/* {env.BACKEND_URL} {
            header_up Host              {upstream_hostport}
            header_up X-Real-IP         {remote_host}
            header_up X-Forwarded-Proto {scheme}
        }

        # SPA fallback: any path with no matching file serves index.html.
        try_files {path} /index.html
        file_server
    }
}
```

**Environment variables:**

| Variable | Container | Value | Purpose |
|---|---|---|---|
| `FRONTEND_ORIGIN` | Go backend | `https://app.example.com` | CORS allow-origin |
| `VITE_API_URL` | Frontend build arg | `https://api.example.com` | API base URL baked into JS bundle |
| `BACKEND_URL` | Caddy | `http://backend:8080` | Internal Docker network address of Go container |

### DDoS and availability — what Caddy provides vs what it does not

Standard Caddy (including the DHI image) provides:

- **Slowloris mitigation** — `read_header`/`read_body` timeouts cut off stalled clients.
- **Header-flood mitigation** — `max_header_size` rejects oversized headers before parsing.
- **Body-size enforcement** — `request_body max_size` returns 413 before the body is read into memory.
- **Compression** — reduces bandwidth under normal load, indirectly limiting amplification impact.

Standard Caddy does **not** include:

- **L7 rate limiting** — requires the `caddy-ratelimit` module (`github.com/mholt/caddy-ratelimit`), which is not in the standard binary. A custom Caddy build using `xcaddy` is needed, layered on top of the DHI image.
- **WAF / OWASP CRS** — requires the `coraza-caddy` module. Same build requirement.

**Recommended approach**: place Cloudflare (or equivalent CDN) in front of Caddy. Cloudflare's free tier handles volumetric DDoS, L7 rate limiting, and bot challenges before traffic reaches the container. Caddy's built-in limits are a last-resort backstop, not the primary defence.

If rate limiting at the Caddy layer is required without a CDN, build a custom image:

```dockerfile
FROM caddy:2.11-builder AS xcaddy
RUN xcaddy build \
    --with github.com/mholt/caddy-ratelimit

FROM dhi.io/caddy:2.11.x@sha256:7c337b4a5b2a5bb3f5d4d44387cd3a54b883e08a548c4c1756aee917e9064b74
COPY --from=xcaddy /usr/bin/caddy /usr/bin/caddy
```

This keeps the hardened image filesystem but replaces the DHI binary with one built from the official (non-hardened) `caddy:2.11-builder` image. The binary is no longer DHI-vetted; factor this into your threat model.

---

## Non-negotiable constraints

These constraints must be maintained throughout Phase 1. Each one is what makes the Phase 1 → Phase 2 migration a mechanical find-and-delete rather than a refactor.

| Constraint | Rule | Why it matters for migration |
|---|---|---|
| `VITE_API_URL` | Always read from `import.meta.env.VITE_API_URL`; default to `""` when absent | Phase 2: set this build arg to the Go API URL and the frontend requests the right host automatically |
| Centralised API client | All HTTP calls go through `src/api/`; no raw `fetch` elsewhere | Ensures `VITE_API_URL` is consumed in exactly one place; nothing to hunt down |
| CORS always configured | Echo CORS middleware always reads `FRONTEND_ORIGIN`; empty string = allow same-origin only | Phase 2: set `FRONTEND_ORIGIN` to the Caddy-served domain and CORS works; no code change |
| `internal/server/static.go` isolated | `//go:embed` and the route registration for static files live only in this file | Phase 2: delete this file and remove its one call-site; the rest of the server is untouched |
| SPA fallback last | The static catch-all (`/*`) is registered after all `/api/v1/*` routes | Prevents API routes from being swallowed by the SPA fallback; same requirement applies to the Caddyfile `route` block |

---

## Migration playbook: Phase 1 → Phase 2

### Backend (3 changes)

1. Delete `internal/server/static.go`.
2. Remove the `registerStaticHandler(e)` call in `internal/server/server.go` (or equivalent).
3. Set `FRONTEND_ORIGIN=https://app.example.com` in the backend's production environment.

### Frontend (1 change)

1. Set `VITE_API_URL=https://api.example.com` as a build argument in CI when building the frontend Docker image.

### Infrastructure (new files)

1. Create `deploy/Caddyfile` (see above).
2. Create `docker-compose.yml` (or Kubernetes manifests):

```yaml
services:
  backend:
    image: ghcr.io/your-org/your-project-backend:latest
    environment:
      FRONTEND_ORIGIN: https://app.example.com
    expose: ["8080"]

  frontend:
    image: ghcr.io/your-org/your-project-frontend:latest
    environment:
      BACKEND_URL: http://backend:8080
    ports: ["80:80"]
    depends_on: [backend]
```

3. Add CI jobs to build and push both images separately.

The backend and frontend images are built and deployed independently from this point forward. No Go code changes are required beyond the two deletions above.
