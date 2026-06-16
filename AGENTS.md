# Link Shortener

Rust link shortener: Axum + SQLite + Tera templates. Designed for single-team/homeserver use.

## Quick Reference

```bash
cargo run                          # local dev (serves on :8081)
cargo build --release              # release build
cargo test                         # run tests
cargo clippy                       # lint (no custom config)
cargo fmt                          # format
```

## Architecture

- **Single crate**, no workspace. Entry: `src/main.rs`.
- **Modules**: `config` (env loading), `db` (sqlite/sqlx), `routes` (handlers), `auth` (session extraction).
- **Templates**: Tera, loaded from `templates/` relative to CWD. Binary must run from project root.
- **Database**: SQLite via sqlx. Migrations in `migrations/` are run in code (`db.rs`), not via `sqlx migrate`.
- **Auth**: Cookie-based sessions stored in DB. Password bcrypt-hashed, stored in `settings` table. First login sets the password.

## Deployment

- **Container**: `Containerfile` (not Dockerfile). Builds musl binary for Alpine.
- **Runtime**: Uses `podman-compose`, not `docker-compose`. The systemd service (`linkshortener.service`) calls `podman-compose`.
- **Compose services**: `linkshortener` (app) + `cloudflared` (Cloudflare Tunnel).
- **Env**: Copy `.env.example` to `.env`. Key vars: `DATABASE_URL`, `LISTEN_ADDR`, `SESSION_SECRET`, `TUNNEL_TOKEN`.

## Gotchas

- No `sqlx` compile-time query checking. Queries are runtime; schema changes won't cause compile errors.
- Templates path is hardcoded relative to CWD — always run from project root.
- `AuthSession` extractor pulls `SqlitePool` from request extensions (`Extension(pool)` layer in main.rs), not from `AppState`. Both exist; `AppState` is for handlers that also need Tera.
- Integration tests are minimal (`tests/integration_test.rs`). No unit tests exist.
- No CI, no pre-commit hooks, no linting beyond `clippy`.
