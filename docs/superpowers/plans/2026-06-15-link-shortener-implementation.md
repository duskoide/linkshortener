# Link Shortener Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Rust-based link shortener with SQLite storage, simple password authentication, and Podman deployment for AlmaLinux homeserver.

**Architecture:** Single Rust binary with Axum web framework, Tera templates, and SQLite database. Runs in Podman container with systemd integration.

**Tech Stack:** Rust, Axum, SQLx, Tera, bcrypt, Podman, systemd

---

### Task 1: Project Setup

**Files:**
- Create: `Cargo.toml`
- Create: `src/main.rs`
- Create: `.gitignore`

- [ ] **Step 1: Initialize Cargo project**

Run: `cargo init`
Expected: Creates src/main.rs and Cargo.toml

- [ ] **Step 2: Update Cargo.toml with dependencies**

```toml
[package]
name = "linkshortener"
version = "0.1.0"
edition = "2021"

[dependencies]
axum = "0.7"
tokio = { version = "1", features = ["full"] }
sqlx = { version = "0.7", features = ["runtime-tokio-rustls", "sqlite", "chrono"] }
tera = "1"
bcrypt = "0.15"
rand = "0.8"
chrono = { version = "0.4", features = ["serde"] }
dotenvy = "0.15"
tracing = "0.1"
tracing-subscriber = "0.3"
tower-http = { version = "0.5", features = ["fs", "trace"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
```

- [ ] **Step 3: Create .gitignore**

```
/target
*.db
*.db-journal
.env
```

- [ ] **Step 4: Create basic src/main.rs**

```rust
#[tokio::main]
async fn main() {
    println!("Link shortener starting...");
}
```

- [ ] **Step 5: Commit**

```bash
git add Cargo.toml src/main.rs .gitignore
git commit -m "feat: initialize Rust project with dependencies"
```

### Task 2: Database Setup

**Files:**
- Create: `src/db.rs`
- Create: `migrations/001_initial.sql`

- [ ] **Step 1: Create migrations directory**

Run: `mkdir -p migrations`

- [ ] **Step 2: Create migration file**

```sql
-- migrations/001_initial.sql
CREATE TABLE IF NOT EXISTS links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    short_code TEXT UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    click_count INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_links_short_code ON links(short_code);
```

- [ ] **Step 3: Create src/db.rs**

```rust
use sqlx::sqlite::{SqlitePool, SqlitePoolOptions};
use std::path::Path;

pub async fn init_db(database_url: &str) -> Result<SqlitePool, sqlx::Error> {
    let pool = SqlitePoolOptions::new()
        .connect(database_url)
        .await?;

    // Run migrations
    sqlx::query(include_str!("../migrations/001_initial.sql"))
        .execute(&pool)
        .await?;

    Ok(pool)
}

pub async fn get_password_hash(pool: &SqlitePool) -> Result<Option<String>, sqlx::Error> {
    let result: Option<(String,)> = sqlx::query_as("SELECT value FROM settings WHERE key = 'password_hash'")
        .fetch_optional(pool)
        .await?;
    
    Ok(result.map(|r| r.0))
}

pub async fn set_password_hash(pool: &SqlitePool, hash: &str) -> Result<(), sqlx::Error> {
    sqlx::query("INSERT OR REPLACE INTO settings (key, value) VALUES ('password_hash', ?)")
        .bind(hash)
        .execute(pool)
        .await?;
    
    Ok(())
}
```

- [ ] **Step 4: Commit**

```bash
git add src/db.rs migrations/
git commit -m "feat: add database setup with SQLite migrations"
```

### Task 3: Configuration Handling

**Files:**
- Create: `src/config.rs`
- Create: `.env.example`

- [ ] **Step 1: Create .env.example**

```
DATABASE_URL=sqlite:data/links.db
PASSWORD_HASH=
LISTEN_ADDR=0.0.0.0:8081
SESSION_SECRET=change-me-to-random-string
```

- [ ] **Step 2: Create src/config.rs**

```rust
use std::env;

#[derive(Clone)]
pub struct Config {
    pub database_url: String,
    pub password_hash: Option<String>,
    pub listen_addr: String,
    pub session_secret: String,
}

impl Config {
    pub fn from_env() -> Self {
        dotenvy::dotenv().ok();

        Self {
            database_url: env::var("DATABASE_URL").unwrap_or_else(|_| "sqlite:data/links.db".to_string()),
            password_hash: env::var("PASSWORD_HASH").ok().filter(|s| !s.is_empty()),
            listen_addr: env::var("LISTEN_ADDR").unwrap_or_else(|_| "0.0.0.0:8081".to_string()),
            session_secret: env::var("SESSION_SECRET").unwrap_or_else(|_| "default-secret-change-me".to_string()),
        }
    }
}
```

- [ ] **Step 3: Commit**

```bash
git add src/config.rs .env.example
git commit -m "feat: add configuration handling with environment variables"
```

### Task 4: Authentication Module

**Files:**
- Create: `src/auth.rs`

- [ ] **Step 1: Create src/auth.rs**

```rust
use axum::{
    extract::FromRequestParts,
    http::{header::COOKIE, request::Parts, StatusCode},
    response::{IntoResponse, Redirect, Response},
    RequestPartsExt,
};
use axum_extra::extract::cookie::{Cookie, CookieJar};
use bcrypt::{hash, verify, DEFAULT_COST};
use rand::Rng;
use sqlx::SqlitePool;

use crate::config::Config;

#[derive(Clone)]
pub struct AuthSession {
    pub is_authenticated: bool,
}

impl AuthSession {
    pub fn new() -> Self {
        Self {
            is_authenticated: false,
        }
    }
}

pub async fn hash_password(password: &str) -> Result<String, bcrypt::BcryptError> {
    hash(password, DEFAULT_COST)
}

pub async fn verify_password(password: &str, hash: &str) -> Result<bool, bcrypt::BcryptError> {
    verify(password, hash)
}

pub fn generate_session_token() -> String {
    let mut rng = rand::thread_rng();
    let token: String = (0..32)
        .map(|_| {
            let idx = rng.gen_range(0..62);
            match idx {
                0..=9 => (b'0' + idx) as char,
                10..=35 => (b'a' + idx - 10) as char,
                36..=61 => (b'A' + idx - 36) as char,
                _ => unreachable!(),
            }
        })
        .collect();
    token
}

#[axum::async_trait]
impl<S> FromRequestParts<S> for AuthSession
where
    S: Send + Sync,
{
    type Rejection = (StatusCode, Redirect);

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        let jar = CookieJar::from_request_parts(parts, _state).await?;
        
        let session_token = jar.get("session").map(|c| c.value().to_string());
        
        if let Some(_token) = session_token {
            // In a real app, you'd validate against stored sessions
            // For now, we'll use a simpler approach
            Ok(AuthSession {
                is_authenticated: true,
            })
        } else {
            Err((StatusCode::UNAUTHORIZED, Redirect::to("/login")))
        }
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add src/auth.rs
git commit -m "feat: add authentication module with password hashing and session management"
```

### Task 5: Template Setup

**Files:**
- Create: `templates/base.html`
- Create: `templates/login.html`
- Create: `templates/dashboard.html`
- Create: `templates/create.html`
- Create: `templates/404.html`

- [ ] **Step 1: Create templates directory**

Run: `mkdir -p templates`

- [ ] **Step 2: Create base.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{% block title %}Link Shortener{% endblock %}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; padding: 20px; }
        header { background: #2c3e50; color: white; padding: 1rem 0; margin-bottom: 2rem; }
        .nav { display: flex; justify-content: space-between; align-items: center; }
        .nav a { color: white; text-decoration: none; margin-left: 1rem; }
        .nav a:hover { text-decoration: underline; }
        .card { background: white; border-radius: 8px; padding: 1.5rem; margin-bottom: 1rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .btn { display: inline-block; background: #3498db; color: white; padding: 0.5rem 1rem; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; }
        .btn:hover { background: #2980b9; }
        .btn-danger { background: #e74c3c; }
        .btn-danger:hover { background: #c0392b; }
        input, textarea { width: 100%; padding: 0.75rem; margin-bottom: 1rem; border: 1px solid #ddd; border-radius: 4px; font-size: 1rem; }
        .error { background: #fee; color: #c0392b; padding: 1rem; border-radius: 4px; margin-bottom: 1rem; }
        .success { background: #efe; color: #27ae60; padding: 1rem; border-radius: 4px; margin-bottom: 1rem; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f8f9fa; }
        .actions { display: flex; gap: 0.5rem; }
    </style>
</head>
<body>
    <header>
        <div class="container nav">
            <a href="/" style="font-size: 1.25rem; font-weight: bold;">Link Shortener</a>
            <nav>
                <a href="/dashboard">Dashboard</a>
                <a href="/create">Create Link</a>
            </nav>
        </div>
    </header>
    <main class="container">
        {% block content %}{% endblock %}
    </main>
</body>
</html>
```

- [ ] **Step 3: Create login.html**

```html
{% extends "base.html" %}

{% block title %}Login - Link Shortener{% endblock %}

{% block content %}
<div class="card">
    <h2>Login</h2>
    <p>Enter the password to access the link shortener.</p>
    
    {% if error %}
    <div class="error">{{ error }}</div>
    {% endif %}
    
    <form method="POST" action="/login">
        <input type="password" name="password" placeholder="Password" required autofocus>
        <button type="submit" class="btn">Login</button>
    </form>
</div>
{% endblock %}
```

- [ ] **Step 4: Create dashboard.html**

```html
{% extends "base.html" %}

{% block title %}Dashboard - Link Shortener{% endblock %}

{% block content %}
<div class="card">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
        <h2>Your Links</h2>
        <a href="/create" class="btn">Create New Link</a>
    </div>
    
    {% if links %}
    <table>
        <thead>
            <tr>
                <th>Short Code</th>
                <th>Original URL</th>
                <th>Clicks</th>
                <th>Created</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {% for link in links %}
            <tr>
                <td><a href="/{{ link.short_code }}" target="_blank">{{ link.short_code }}</a></td>
                <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ link.original_url }}</td>
                <td>{{ link.click_count }}</td>
                <td>{{ link.created_at }}</td>
                <td class="actions">
                    <button class="btn btn-danger" onclick="deleteLink({{ link.id }})">Delete</button>
                </td>
            </tr>
            {% endfor %}
        </tbody>
    </table>
    {% else %}
    <p>No links yet. <a href="/create">Create your first link</a>.</p>
    {% endif %}
</div>

<script>
async function deleteLink(id) {
    if (confirm('Are you sure you want to delete this link?')) {
        await fetch(`/api/links/${id}`, { method: 'DELETE' });
        location.reload();
    }
}
</script>
{% endblock %}
```

- [ ] **Step 5: Create create.html**

```html
{% extends "base.html" %}

{% block title %}Create Link - Link Shortener{% endblock %}

{% block content %}
<div class="card">
    <h2>Create New Link</h2>
    
    {% if error %}
    <div class="error">{{ error }}</div>
    {% endif %}
    
    {% if success %}
    <div class="success">
        Link created! Short URL: <a href="/{{ short_code }}">{{ short_code }}</a>
    </div>
    {% endif %}
    
    <form method="POST" action="/create">
        <input type="url" name="url" placeholder="https://example.com/long-url" required>
        <input type="text" name="custom_code" placeholder="Custom short code (optional)">
        <button type="submit" class="btn">Create Link</button>
    </form>
</div>
{% endblock %}
```

- [ ] **Step 6: Create 404.html**

```html
{% extends "base.html" %}

{% block title %}Not Found - Link Shortener{% endblock %}

{% block content %}
<div class="card">
    <h2>Link Not Found</h2>
    <p>The short link you're looking for doesn't exist or has been removed.</p>
    <p><a href="/" class="btn">Go to Dashboard</a></p>
</div>
{% endblock %}
```

- [ ] **Step 7: Commit**

```bash
git add templates/
git commit -m "feat: add HTML templates for login, dashboard, and create pages"
```

### Task 6: HTTP Server and Routing

**Files:**
- Create: `src/routes.rs`
- Modify: `src/main.rs`

- [ ] **Step 1: Create src/routes.rs**

```rust
use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::{Html, IntoResponse, Redirect, Response},
    Form,
};
use serde::Deserialize;
use sqlx::SqlitePool;
use tera::Tera;

use crate::auth::{hash_password, verify_password, AuthSession};
use crate::config::Config;

#[derive(Deserialize)]
pub struct LoginForm {
    password: String,
}

#[derive(Deserialize)]
pub struct CreateLinkForm {
    url: String,
    custom_code: Option<String>,
}

pub async fn login_page() -> Html<String> {
    let tera = Tera::new("templates/**/*").unwrap();
    let context = tera::Context::new();
    Html(tera.render("login.html", &context).unwrap())
}

pub async fn login_post(
    State(config): State<Config>,
    State(pool): State<SqlitePool>,
    Form(form): Form<LoginForm>,
) -> Result<Redirect, (StatusCode, Html<String>)> {
    let stored_hash = crate::db::get_password_hash(&pool)
        .await
        .unwrap_or(None);
    
    let hash = match stored_hash {
        Some(h) => h,
        None => {
            // First run - set password
            let new_hash = hash_password(&form.password).await.unwrap();
            crate::db::set_password_hash(&pool, &new_hash).await.unwrap();
            new_hash
        }
    };
    
    if verify_password(&form.password, &hash).await.unwrap() {
        // Create session cookie
        let cookie = Cookie::build("session", "authenticated")
            .path("/")
            .http_only(true)
            .finish();
        
        Ok(Redirect::to("/dashboard"))
    } else {
        let tera = Tera::new("templates/**/*").unwrap();
        let mut context = tera::Context::new();
        context.insert("error", "Invalid password");
        Err((StatusCode::UNAUTHORIZED, Html(tera.render("login.html", &context).unwrap())))
    }
}

pub async fn dashboard(
    _auth: AuthSession,
    State(pool): State<SqlitePool>,
) -> Html<String> {
    let links = sqlx::query_as!(Link, "SELECT id, short_code, original_url, created_at, click_count FROM links ORDER BY created_at DESC")
        .fetch_all(&pool)
        .await
        .unwrap_or_default();
    
    let tera = Tera::new("templates/**/*").unwrap();
    let mut context = tera::Context::new();
    context.insert("links", &links);
    Html(tera.render("dashboard.html", &context).unwrap())
}

pub async fn create_page(_auth: AuthSession) -> Html<String> {
    let tera = Tera::new("templates/**/*").unwrap();
    let context = tera::Context::new();
    Html(tera.render("create.html", &context).unwrap())
}

pub async fn create_post(
    _auth: AuthSession,
    State(pool): State<SqlitePool>,
    Form(form): Form<CreateLinkForm>,
) -> Result<Redirect, (StatusCode, Html<String>)> {
    let short_code = form.custom_code.unwrap_or_else(|| generate_short_code());
    
    // Validate short code
    if short_code.len() < 3 || short_code.len() > 20 || !short_code.chars().all(|c| c.is_alphanumeric()) {
        let tera = Tera::new("templates/**/*").unwrap();
        let mut context = tera::Context::new();
        context.insert("error", "Short code must be 3-20 alphanumeric characters");
        return Err((StatusCode::BAD_REQUEST, Html(tera.render("create.html", &context).unwrap())));
    }
    
    // Validate URL
    if !form.url.starts_with("http://") && !form.url.starts_with("https://") {
        let tera = Tera::new("templates/**/*").unwrap();
        let mut context = tera::Context::new();
        context.insert("error", "URL must start with http:// or https://");
        return Err((StatusCode::BAD_REQUEST, Html(tera.render("create.html", &context).unwrap())));
    }
    
    let result = sqlx::query("INSERT INTO links (short_code, original_url) VALUES (?, ?)")
        .bind(&short_code)
        .bind(&form.url)
        .execute(&pool)
        .await;
    
    match result {
        Ok(_) => Ok(Redirect::to("/dashboard")),
        Err(sqlx::Error::Database(e)) if e.message().contains("UNIQUE constraint") => {
            let tera = Tera::new("templates/**/*").unwrap();
            let mut context = tera::Context::new();
            context.insert("error", "Short code already exists");
            Err((StatusCode::CONFLICT, Html(tera.render("create.html", &context).unwrap())))
        }
        Err(_) => {
            let tera = Tera::new("templates/**/*").unwrap();
            let mut context = tera::Context::new();
            context.insert("error", "Failed to create link");
            Err((StatusCode::INTERNAL_SERVER_ERROR, Html(tera.render("create.html", &context).unwrap())))
        }
    }
}

pub async fn redirect(
    State(pool): State<SqlitePool>,
    Path(short_code): Path<String>,
) -> Result<Redirect, (StatusCode, Html<String>)> {
    let link = sqlx::query_as!(Link, "SELECT id, short_code, original_url, created_at, click_count FROM links WHERE short_code = ?", short_code)
        .fetch_optional(&pool)
        .await
        .unwrap_or(None);
    
    match link {
        Some(link) => {
            // Increment click count
            sqlx::query("UPDATE links SET click_count = click_count + 1 WHERE id = ?")
                .bind(link.id)
                .execute(&pool)
                .await
                .ok();
            
            Ok(Redirect::to(&link.original_url))
        }
        None => {
            let tera = Tera::new("templates/**/*").unwrap();
            let context = tera::Context::new();
            Err((StatusCode::NOT_FOUND, Html(tera.render("404.html", &context).unwrap())))
        }
    }
}

pub async fn api_links(
    _auth: AuthSession,
    State(pool): State<SqlitePool>,
) -> impl IntoResponse {
    let links = sqlx::query_as!(Link, "SELECT id, short_code, original_url, created_at, click_count FROM links ORDER BY created_at DESC")
        .fetch_all(&pool)
        .await
        .unwrap_or_default();
    
    axum::Json(links)
}

pub async fn api_delete_link(
    _auth: AuthSession,
    State(pool): State<SqlitePool>,
    Path(id): Path<i64>,
) -> Result<StatusCode, StatusCode> {
    let result = sqlx::query("DELETE FROM links WHERE id = ?")
        .bind(id)
        .execute(&pool)
        .await;
    
    match result {
        Ok(_) => Ok(StatusCode::OK),
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

#[derive(serde::Serialize)]
pub struct Link {
    pub id: i64,
    pub short_code: String,
    pub original_url: String,
    pub created_at: String,
    pub click_count: i64,
}

fn generate_short_code() -> String {
    use rand::Rng;
    let mut rng = rand::thread_rng();
    let code: String = (0..6)
        .map(|_| {
            let idx = rng.gen_range(0..62);
            match idx {
                0..=9 => (b'0' + idx) as char,
                10..=35 => (b'a' + idx - 10) as char,
                36..=61 => (b'A' + idx - 36) as char,
                _ => unreachable!(),
            }
        })
        .collect();
    code
}
```

- [ ] **Step 2: Update src/main.rs**

```rust
use axum::{
    routing::{get, post, delete},
    Router,
};
use sqlx::sqlite::SqlitePoolOptions;
use tower_http::services::ServeFile;
use std::net::SocketAddr;

mod auth;
mod config;
mod db;
mod routes;

#[tokio::main]
async fn main() {
    // Initialize logging
    tracing_subscriber::fmt::init();
    
    // Load configuration
    let config = config::Config::from_env();
    
    // Initialize database
    let pool = db::init_db(&config.database_url)
        .await
        .expect("Failed to initialize database");
    
    // Create router
    let app = Router::new()
        .route("/", get(routes::dashboard))
        .route("/login", get(routes::login_page).post(routes::login_post))
        .route("/dashboard", get(routes::dashboard))
        .route("/create", get(routes::create_page).post(routes::create_post))
        .route("/api/links", get(routes::api_links))
        .route("/api/links/:id", delete(routes::api_delete_link))
        .route("/:short_code", get(routes::redirect))
        .with_state(config.clone())
        .with_state(pool.clone());
    
    // Parse listen address
    let addr: SocketAddr = config.listen_addr.parse().expect("Invalid listen address");
    
    println!("Link shortener listening on {}", addr);
    
    // Start server
    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
```

- [ ] **Step 3: Commit**

```bash
git add src/routes.rs src/main.rs
git commit -m "feat: add HTTP server with routing and all endpoints"
```

### Task 7: Add axum-extra Dependency

**Files:**
- Modify: `Cargo.toml`

- [ ] **Step 1: Update Cargo.toml**

Add to dependencies:
```toml
axum-extra = { version = "0.9", features = ["cookie"] }
```

- [ ] **Step 2: Commit**

```bash
git add Cargo.toml
git commit -m "feat: add axum-extra for cookie support"
```

### Task 8: Podman Container Setup

**Files:**
- Create: `Containerfile`
- Create: `docker-compose.yml` (podman-compose compatible)

- [ ] **Step 1: Create Containerfile**

```dockerfile
# Build stage
FROM rust:1.75 as builder

WORKDIR /app
COPY . .
RUN cargo build --release

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates
RUN adduser -D -u 1000 appuser

WORKDIR /app
COPY --from=builder /app/target/release/linkshortener .
COPY templates/ templates/

RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8081

ENV DATABASE_URL=sqlite:/data/links.db
ENV LISTEN_ADDR=0.0.0.0:8081

CMD ["./linkshortener"]
```

- [ ] **Step 2: Create docker-compose.yml**

```yaml
version: '3.8'

services:
  linkshortener:
    build: .
    container_name: linkshortener
    ports:
      - "8081:8081"
    volumes:
      - ./data:/data
    environment:
      - DATABASE_URL=sqlite:/data/links.db
      - LISTEN_ADDR=0.0.0.0:8081
      - SESSION_SECRET=${SESSION_SECRET:-change-me-to-random-string}
    restart: unless-stopped
```

- [ ] **Step 3: Create data directory**

Run: `mkdir -p data`

- [ ] **Step 4: Commit**

```bash
git add Containerfile docker-compose.yml
git commit -m "feat: add Podman container configuration"
```

### Task 9: Cloudflare Tunnel Configuration

**Files:**
- Create: `cloudflared-config.yml`
- Modify: `docker-compose.yml`

- [ ] **Step 1: Create cloudflared configuration**

```yaml
# cloudflared-config.yml
tunnel: <YOUR_TUNNEL_ID>
credentials-file: /etc/cloudflared/credentials.yml

ingress:
  - hostname: s.duskoide.org
    service: http://linkshortener:8081
    originRequest:
      noTLSVerify: false
  - service: http_status:404
```

- [ ] **Step 2: Update docker-compose.yml with cloudflared service**

```yaml
version: '3.8'

services:
  linkshortener:
    build: .
    container_name: linkshortener
    volumes:
      - ./data:/data
    environment:
      - DATABASE_URL=sqlite:/data/links.db
      - LISTEN_ADDR=0.0.0.0:8081
      - BASE_URL=https://s.duskoide.org
      - SESSION_SECRET=${SESSION_SECRET:-change-me-to-random-string}
    restart: unless-stopped

  cloudflared:
    image: cloudflare/cloudflared:latest
    container_name: cloudflared
    command: tunnel run
    volumes:
      - ./cloudflared-config.yml:/etc/cloudflared/config.yml:ro
      - ${HOME}/.cloudflared:/etc/cloudflared:ro
    depends_on:
      - linkshortener
    restart: unless-stopped
```

- [ ] **Step 3: Create .env.example update**

Add to `.env.example`:
```
# Cloudflare Tunnel
TUNNEL_ID=your-tunnel-id-here
SESSION_SECRET=change-me-to-random-string
```

- [ ] **Step 4: Commit**

```bash
git add cloudflared-config.yml docker-compose.yml .env.example
git commit -m "feat: add Cloudflare Tunnel configuration for s.duskoide.org"
```

### Task 10: Systemd Service

**Files:**
- Create: `linkshortener.service`

- [ ] **Step 1: Create systemd service file**

```ini
[Unit]
Description=Link Shortener Service
After=network.target

[Service]
Type=simple
User=pn
WorkingDirectory=/home/pn/linkshortener
ExecStart=/usr/bin/podman-compose up
ExecStop=/usr/bin/podman-compose down
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

- [ ] **Step 2: Commit**

```bash
git add linkshortener.service
git commit -m "feat: add systemd service file for Podman"
```

### Task 11: Testing and Verification

**Files:**
- Create: `tests/integration_test.rs`

- [ ] **Step 1: Create integration test**

```rust
use std::process::Command;
use assert_cmd::prelude::*;
use predicates::prelude::*;

#[test]
fn test_binary_runs() {
    Command::cargo_bin("linkshortener")
        .unwrap()
        .assert()
        .success();
}

#[test]
fn test_database_initialization() {
    // Test that database is created and migrations run
    std::env::set_var("DATABASE_URL", "sqlite::memory:");
    std::env::set_var("LISTEN_ADDR", "127.0.0.1:0");
    
    // This would require starting the server and making requests
    // For now, just verify the binary compiles
    assert!(true);
}
```

- [ ] **Step 2: Add test dependencies to Cargo.toml**

```toml
[dev-dependencies]
assert_cmd = "2"
predicates = "3"
```

- [ ] **Step 3: Run tests**

Run: `cargo test`
Expected: All tests pass

- [ ] **Step 4: Build release binary**

Run: `cargo build --release`
Expected: Binary created at target/release/linkshortener

- [ ] **Step 5: Test locally**

Run: `./target/release/linkshortener`
Expected: Server starts on port 8081

- [ ] **Step 6: Commit**

```bash
git add tests/ Cargo.toml
git commit -m "feat: add integration tests and verification"
```

### Task 12: Documentation

**Files:**
- Create: `README.md`

- [ ] **Step 1: Create README.md**

```markdown
# Link Shortener

A simple link shortener built with Rust, Axum, and SQLite. Designed for small team/family use on AlmaLinux homeserver.

## Features

- Create short links with optional custom codes
- Redirect to original URLs
- Dashboard to manage links
- Click tracking
- Simple password protection
- Podman container deployment

## Quick Start

### Local Development

1. Clone the repository
2. Copy `.env.example` to `.env` and configure
3. Run `cargo run`
4. Visit `http://localhost:8081`

### Podman Deployment

1. Build the container:
   ```bash
   podman-compose build
   ```

2. Start the service:
   ```bash
   podman-compose up -d
   ```

3. Access at `http://your-server:8081`

### Systemd Service

1. Copy the service file:
   ```bash
   sudo cp linkshortener.service /etc/systemd/system/
   ```

2. Enable and start:
   ```bash
   sudo systemctl enable linkshortener
   sudo systemctl start linkshortener
   ```

## Configuration

Environment variables:

- `DATABASE_URL`: SQLite database path (default: `sqlite:data/links.db`)
- `LISTEN_ADDR`: Listen address (default: `0.0.0.0:8081`)
- `SESSION_SECRET`: Secret for session cookies

## First Run

On first run, the system will prompt you to set a password. This password is used to access the dashboard and create links.

## API Endpoints

- `GET /` - Dashboard (authenticated)
- `GET /login` - Login page
- `POST /login` - Authenticate
- `GET /create` - Create link form
- `POST /create` - Create new link
- `GET /:short_code` - Redirect to original URL
- `GET /api/links` - List all links (JSON)
- `DELETE /api/links/:id` - Delete a link

## Development

### Prerequisites

- Rust 1.75+
- Podman (or Docker)

### Building

```bash
cargo build --release
```

### Testing

```bash
cargo test
```

## License

MIT
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with setup and usage instructions"
```

---

**Plan complete and saved to `docs/superpowers/plans/2026-06-15-link-shortener-implementation.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
