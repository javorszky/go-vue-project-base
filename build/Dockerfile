# ── Frontend ──────────────────────────────────────────────────────────────────
# Separate deps layer so node_modules is cached as long as package*.json is unchanged.
FROM node:24-alpine AS frontend-deps
WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm ci

FROM frontend-deps AS frontend-builder
COPY frontend/ ./
RUN npm run build

# ── Go binary ─────────────────────────────────────────────────────────────────
# Pinned to the exact Go patch release, not the floating 1.26-alpine tag: the go
# directive in go.mod is the enforced minimum Go version, so the builder must not
# silently drift away from it. Renovate keeps tag and digest in step (see renovate.json).
FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go-builder
WORKDIR /app

# Cache module downloads separately from source.
# go.sum must be committed (run `go mod tidy` locally if it is missing).
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY api/ ./api/
# Overwrite the placeholder with the freshly compiled SPA.
# Vite builds to ../internal/ui/dist (relative to /frontend workdir) = /internal/ui/dist.
COPY --from=frontend-builder /internal/ui/dist ./internal/ui/dist

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /server \
    ./cmd/server

# ── Runtime ───────────────────────────────────────────────────────────────────
FROM scratch
# CA bundle for outbound TLS; sourced from the Alpine-based builder.
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Timezone data is intentionally omitted — the binary runs in UTC.
# If time.LoadLocation() for non-UTC zones is needed, add to go-builder:
#   RUN apk add --no-cache tzdata
# and copy here:
#   COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=go-builder /server /server

EXPOSE 8080
# Run as nobody (UID 65534) — no /etc/passwd needed in scratch.
USER 65534:65534
ENTRYPOINT ["/server"]
