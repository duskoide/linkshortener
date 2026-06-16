# Link Shortener

Go link shortener: Chi + SQLite + html/template. Designed for single-team/homeserver use.

## Quick Reference

```bash
go run ./cmd/linkshortener          # local dev (serves on :8081)
go build -o linkshortener ./cmd/linkshortener  # build binary
go build ./...                      # verify compilation
```

## Architecture

- **Single module**, no workspace. Entry: `cmd/linkshortener/main.go`.
- **Packages**: `internal/config` (env loading), `internal/db` (sqlite queries), `internal/handlers` (route handlers + templates), `internal/auth` (middleware + password utils).
- **Templates**: html/template, loaded per-page with `base.html` as layout. Binary must run from project root.
- **Database**: SQLite via `modernc.org/sqlite` (pure Go, no CGO). Migrations in `internal/db/migrations/` embedded via `//go:embed`.
- **Auth**: Cookie-based sessions stored in DB. Password bcrypt-hashed, stored in `settings` table. First login sets the password.
- **Router**: Chi v5 with Logger + Recoverer middleware. Protected routes use auth middleware group.

## Deployment

- **Container**: `Containerfile`. Multi-stage build: golang alpine → alpine runtime. No CGO.
- **Runtime**: Uses `podman-compose`, not `docker-compose`. The systemd service (`linkshortener.service`) calls `podman-compose`.
- **Compose services**: `linkshortener` (app) + `cloudflared` (Cloudflare Tunnel).
- **Env**: Copy `.env.example` to `.env`. Key vars: `DATABASE_URL`, `LISTEN_ADDR`, `SESSION_SECRET`, `TUNNEL_TOKEN`.

## Gotchas

- Templates path is hardcoded relative to CWD — always run from project root.
- `renderTemplate()` in `internal/handlers/templates.go` parses base.html + page as a set. Don't use `template.ParseGlob` — it causes `{{define}}` conflicts.
- `go.mod` uses `go 1.22`. Keep in sync with `Containerfile` golang version.
- `SESSION_SECRET` and `TUNNEL_TOKEN` are loaded but not yet used — placeholders for future cookie signing / tunnel config.
- `CleanupExpiredSessions` exists but is never called. Sessions accumulate until manually cleaned.
- No tests exist. No CI, no pre-commit hooks.
