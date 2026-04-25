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
FROM golang:1.26-alpine AS go-builder
WORKDIR /app

# Cache module downloads separately from source.
# go.sum must be committed (run `go mod tidy` locally if it is missing).
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Overwrite frontend/dist with the freshly compiled SPA.
COPY --from=frontend-builder /frontend/dist ./frontend/dist

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
