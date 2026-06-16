# Design Spec: Link Shortener Go Rewrite

## Overview

Rewrite the existing Rust/Axum link shortener in Go using Chi router, SQLite, and stdlib templates. Deploy on AlmaLinux homeserver via Podman with Cloudflare Tunnel for public access at `s.duskoide.org`.

## Current State

- Rust + Axum + SQLite + Tera templates
- Single binary, cookie-based auth, session DB
- Podman-compose deployment with Cloudflare Tunnel sidecar
- systemd service managing podman-compose

## Target Stack

| Layer | Choice | Notes |
|-------|--------|-------|
| Language | Go 1.22+ | |
| Router | `go-chi/chi/v5` | Lightweight, stdlib-compatible |
| Database | SQLite via `modernc.org/sqlite` | Pure Go, no CGO required |
| DB driver | `database/sql` + `github.com/jmoiron/sqlx` | Named args, struct scanning |
| Templates | `html/template` (stdlib) | Compile-time type safety |
| Auth | `golang.org/x/crypto/bcrypt` | Same bcrypt as Rust version |
| Session | Cookie-based, DB-stored | 24h expiry, same as current |
| Container | Alpine-based, multi-stage build | ~10MB final image |

## Project Structure

```
.
├── cmd/
│   └── linkshortener/
│       └── main.go              # Entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Env loading (dotenvy equivalent)
│   ├── db/
│   │   ├── db.go                # Pool init, migration runner
│   │   ├── links.go             # Link CRUD queries
│   │   ├── sessions.go          # Session CRUD queries
│   │   └── settings.go          # Settings (password hash) queries
│   ├── auth/
│   │   ├── middleware.go         # AuthSession extractor middleware
│   │   └── password.go          # bcrypt hash/verify
│   └── handlers/
│       ├── dashboard.go         # GET /, GET /dashboard
│       ├── login.go             # GET/POST /login, POST /logout
│       ├── create.go            # GET/POST /create
│       ├── redirect.go          # GET /:short_code
│       └── api.go               # GET /api/links, DELETE /api/links/:id
├── templates/
│   ├── base.html                # Shared layout
│   ├── login.html
│   ├── dashboard.html
│   ├── create.html
│   └── 404.html
├── migrations/
│   ├── 001_initial.sql          # Reuse as-is from Rust version
│   └── 002_sessions.sql         # Reuse as-is from Rust version
├── go.mod
├── go.sum
├── Containerfile
├── docker-compose.yml           # Keep as-is
├── linkshortener.service        # Keep as-is
└── .env.example                 # Update with new vars if needed
```

## Database

### Driver Choice

`modernc.org/sqlite` — pure Go implementation. No CGO, no musl toolchain, no cross-compilation issues. Identical SQL compatibility.

### Migrations

Run at startup in `db.go`, same pattern as Rust version: `include_str!` equivalent via `embed` package to embed SQL files, execute sequentially.

```go
//go:embed migrations/001_initial.sql
//go:embed migrations/002_sessions.sql
var migrations embed.FS
```

### Schema

Reuse existing SQL files unchanged. Tables: `links`, `settings`, `sessions`.

## Auth & Sessions

### Password

- First login: store bcrypt hash in `settings` table (same flow as Rust version)
- Subsequent logins: verify against stored hash

### Sessions

- Generate random 32-char token (alphanumeric, same as current)
- Store in `sessions` table with 24h expiry
- Set as `HttpOnly` cookie named `session`, path `/`
- Logout: delete session from DB, remove cookie

### Middleware

`AuthSession` middleware extracts cookie, looks up session in DB, injects auth state into request context. Handlers that require auth use this middleware.

## Routes

Same as current Rust version:

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | `/` | dashboard | yes |
| GET | `/dashboard` | dashboard | yes |
| GET | `/login` | loginPage | no |
| POST | `/login` | loginPost | no |
| POST | `/logout` | logout | yes |
| GET | `/create` | createPage | yes |
| POST | `/create` | createPost | yes |
| GET | `/api/links` | apiLinks | yes |
| DELETE | `/api/links/{id}` | apiDeleteLink | yes |
| GET | `/{shortCode}` | redirect | no |

Catch-all route `/{shortCode}` must be registered last (Chi uses longest-match, but placing it last is clearer).

## Config

Environment variables (same as current):

- `DATABASE_URL` — SQLite path (default: `sqlite:data/links.db`)
- `LISTEN_ADDR` — listen address (default: `0.0.0.0:8081`)
- `SESSION_SECRET` — (kept for potential future use; cookie signing)
- `TUNNEL_TOKEN` — Cloudflare Tunnel token

Load `.env` file at startup using `github.com/joho/godotenv`.

## Cloudflare Tunnel

No changes to the tunnel setup. `docker-compose.yml` keeps both services:
- `linkshortener` — the Go app on port 8081
- `cloudflared` — Cloudflare Tunnel sidecar, routes `s.duskoide.org` → `linkshortener:8081`

Tunnel token loaded from `.env` via `TUNNEL_TOKEN` env var.

## Container

Multi-stage `Containerfile`:

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /linkshortener ./cmd/linkshortener

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /linkshortener .
COPY templates/ templates/
RUN mkdir -p /data
EXPOSE 8081
ENV DATABASE_URL=sqlite:/data/links.db
ENV LISTEN_ADDR=0.0.0.0:8081
CMD ["./linkshortener"]
```

No CGO needed (`CGO_ENABLED=0`). No musl toolchain. Final image is `alpine` + binary + templates.

## Error Handling

- Database errors: log with `log/slog`, return 500 with generic error page
- Template rendering errors: panic in dev (clear bug), graceful fallback in production
- Validation errors: re-render form with error message (same UX as current)

## Testing

Minimal integration test to start:
- Binary compiles and starts
- Can connect to in-memory SQLite
- Basic route smoke test (optional, depends on time)

## Deployment

No changes to deployment flow:
1. `podman-compose build`
2. `podman-compose up -d`
3. systemd service manages lifecycle

## Migration from Rust Version

1. Stop Rust version: `podman-compose down`
2. Back up SQLite database (`data/links.db`)
3. Deploy Go version (schema is identical, no migration needed)
4. Start: `podman-compose up -d`
5. Verify at `http://localhost:8081` and `https://s.duskoide.org`

## Out of Scope

- New features beyond feature parity with Rust version
- User management / multi-user support
- API rate limiting
- Link expiration
- Analytics beyond click count
