# Link Shortener Go Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the Rust link shortener in Go with Chi, SQLite, and html/template, preserving feature parity and deployment flow.

**Architecture:** Single Go binary. Chi router with middleware for auth. SQLite via pure-Go driver. Templates converted from Tera to html/template. Same container deployment with Podman.

**Tech Stack:** Go 1.22+, Chi v5, modernc.org/sqlite, html/template, golang.org/x/crypto/bcrypt

---

## File Structure

```
cmd/linkshortener/main.go
internal/config/config.go
internal/db/db.go
internal/db/links.go
internal/db/sessions.go
internal/db/settings.go
internal/auth/middleware.go
internal/auth/password.go
internal/handlers/dashboard.go
internal/handlers/login.go
internal/handlers/create.go
internal/handlers/redirect.go
internal/handlers/api.go
internal/handlers/templates.go
templates/base.html
templates/login.html
templates/dashboard.html
templates/create.html
templates/404.html
go.mod
go.sum
Containerfile
```

---

### Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialize module**

```bash
cd /home/pn/Projects/linkshortener
go mod init linkshortener
```

- [ ] **Step 2: Create directory structure**

```bash
mkdir -p cmd/linkshortener internal/config internal/db internal/auth internal/handlers
```

- [ ] **Step 3: Commit**

```bash
git add go.mod cmd/ internal/
git commit -m "chore: initialize Go module and project structure"
```

---

### Task 2: Config Module

**Files:**
- Create: `internal/config/config.go`
- Create: `.env.example` (update)

- [ ] **Step 1: Create config.go**

```go
package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL  string
	ListenAddr   string
	SessionSecret string
	TunnelToken  string
}

func Load() *Config {
	godotenv.Load()

	return &Config{
		DatabaseURL:  getEnv("DATABASE_URL", "sqlite:data/links.db"),
		ListenAddr:   getEnv("LISTEN_ADDR", "0.0.0.0:8081"),
		SessionSecret: getEnv("SESSION_SECRET", ""),
		TunnelToken:  getEnv("TUNNEL_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 2: Update .env.example**

```bash
DATABASE_URL=sqlite:data/links.db
LISTEN_ADDR=0.0.0.0:8081
SESSION_SECRET=change-me-to-random-string

# Cloudflare Tunnel
TUNNEL_TOKEN=your-tunnel-token-here
```

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go .env.example
git commit -m "feat: add config module with env loading"
```

---

### Task 3: Database Module - Connection and Migrations

**Files:**
- Create: `internal/db/db.go`
- Create: `internal/db/migrations.go` (embedded SQL)

- [ ] **Step 1: Create db.go with embedded migrations**

```go
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Init(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	slog.Info("database initialized", "url", databaseURL)
	return db, nil
}

func runMigrations(db *sql.DB) error {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		content, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("execute %s: %w", entry.Name(), err)
		}

		slog.Info("migration applied", "file", entry.Name())
	}

	return nil
}
```

- [ ] **Step 2: Copy migration SQL files into internal/db/migrations/**

```bash
mkdir -p internal/db/migrations
cp migrations/001_initial.sql internal/db/migrations/
cp migrations/002_sessions.sql internal/db/migrations/
```

- [ ] **Step 3: Fetch dependencies**

```bash
cd /home/pn/Projects/linkshortener
go mod tidy
```

- [ ] **Step 4: Commit**

```bash
git add internal/db/ go.mod go.sum
git commit -m "feat: add database module with embedded migrations"
```

---

### Task 4: Links Queries

**Files:**
- Create: `internal/db/links.go`

- [ ] **Step 1: Create links.go**

```go
package db

import (
	"database/sql"
	"time"
)

type Link struct {
	ID          int64     `json:"id"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	ClickCount  int64     `json:"click_count"`
}

func GetLinkByShortCode(db *sql.DB, shortCode string) (*Link, error) {
	link := &Link{}
	err := db.QueryRow(
		"SELECT id, short_code, original_url, created_at, click_count FROM links WHERE short_code = ?",
		shortCode,
	).Scan(&link.ID, &link.ShortCode, &link.OriginalURL, &link.CreatedAt, &link.ClickCount)
	if err != nil {
		return nil, err
	}
	return link, nil
}

func GetAllLinks(db *sql.DB) ([]Link, error) {
	rows, err := db.Query(
		"SELECT id, short_code, original_url, created_at, click_count FROM links ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.ID, &l.ShortCode, &l.OriginalURL, &l.CreatedAt, &l.ClickCount); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func CreateLink(db *sql.DB, shortCode, originalURL string) error {
	_, err := db.Exec(
		"INSERT INTO links (short_code, original_url) VALUES (?, ?)",
		shortCode, originalURL,
	)
	return err
}

func IncrementClickCount(db *sql.DB, linkID int64) error {
	_, err := db.Exec("UPDATE links SET click_count = click_count + 1 WHERE id = ?", linkID)
	return err
}

func DeleteLink(db *sql.DB, linkID int64) error {
	_, err := db.Exec("DELETE FROM links WHERE id = ?", linkID)
	return err
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/db/links.go
git commit -m "feat: add link CRUD queries"
```

---

### Task 5: Sessions Queries

**Files:**
- Create: `internal/db/sessions.go`

- [ ] **Step 1: Create sessions.go**

```go
package db

import "database/sql"

func CreateSession(db *sql.DB, token string, expiresInHours int) error {
	_, err := db.Exec(
		"INSERT INTO sessions (token, expires_at) VALUES (?, datetime('now', '+' || ? || ' hours'))",
		token, expiresInHours,
	)
	return err
}

func ValidateSession(db *sql.DB, token string) (bool, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE token = ? AND expires_at > datetime('now')",
		token,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func DeleteSession(db *sql.DB, token string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func CleanupExpiredSessions(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM sessions WHERE expires_at <= datetime('now')")
	return err
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/db/sessions.go
git commit -m "feat: add session CRUD queries"
```

---

### Task 6: Settings Queries

**Files:**
- Create: `internal/db/settings.go`

- [ ] **Step 1: Create settings.go**

```go
package db

import "database/sql"

func GetPasswordHash(db *sql.DB) (string, error) {
	var hash string
	err := db.QueryRow("SELECT value FROM settings WHERE key = 'password_hash'").Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return hash, err
}

func SetPasswordHash(db *sql.DB, hash string) error {
	_, err := db.Exec(
		"INSERT OR REPLACE INTO settings (key, value) VALUES ('password_hash', ?)",
		hash,
	)
	return err
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/db/settings.go
git commit -m "feat: add settings queries for password hash"
```

---

### Task 7: Password Utilities

**Files:**
- Create: `internal/auth/password.go`

- [ ] **Step 1: Create password.go**

```go
package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/auth/password.go
git commit -m "feat: add bcrypt password utilities"
```

---

### Task 8: Auth Middleware

**Files:**
- Create: `internal/auth/middleware.go`

- [ ] **Step 1: Create middleware.go**

```go
package auth

import (
	"context"
	"database/sql"
	"net/http"

	"linkshortener/internal/db"

	"github.com/go-chi/chi/v5"
)

type contextKey string

const AuthKey contextKey = "auth"

type AuthSession struct {
	IsAuthenticated bool
}

func Middleware(database *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			valid, err := db.ValidateSession(database, cookie.Value)
			if err != nil || !valid {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			ctx := context.WithValue(r.Context(), AuthKey, &AuthSession{IsAuthenticated: true})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetAuth(r *http.Request) *AuthSession {
	session, _ := r.Context().Value(AuthKey).(*AuthSession)
	return session
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/auth/middleware.go
git commit -m "feat: add auth middleware for session validation"
```

---

### Task 9: Template Loader

**Files:**
- Create: `internal/handlers/templates.go`

- [ ] **Step 1: Create templates.go**

```go
package handlers

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed templates/*
var templateFS embed.FS

var templates *template.Template

func init() {
	sub, _ := fs.Sub(templateFS, "templates")
	templates = template.Must(template.ParseFS(sub, "*.html"))
}

func renderTemplate(w interface{ Write([]byte) (int, error) }, name string, data any) error {
	return templates.ExecuteTemplate(w, name, data)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/handlers/templates.go
git commit -m "feat: add template loader with embedded templates"
```

---

### Task 10: Dashboard Handler

**Files:**
- Create: `internal/handlers/dashboard.go`

- [ ] **Step 1: Create dashboard.go**

```go
package handlers

import (
	"database/sql"
	"net/http"

	"linkshortener/internal/db"
)

func Dashboard(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		links, err := db.GetAllLinks(database)
		if err != nil {
			http.Error(w, "Failed to fetch links", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.ExecuteTemplate(w, "dashboard.html", map[string]any{
			"Links": links,
		})
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/handlers/dashboard.go
git commit -m "feat: add dashboard handler"
```

---

### Task 11: Login Handlers

**Files:**
- Create: `internal/handlers/login.go`

- [ ] **Step 1: Create login.go**

```go
package handlers

import (
	"database/sql"
	"net/http"

	"linkshortener/internal/auth"
	"linkshortener/internal/db"
)

func LoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ExecuteTemplate(w, "login.html", nil)
}

func LoginPost(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		password := r.FormValue("password")

		storedHash, err := db.GetPasswordHash(database)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		// First login: no password set yet, store this one
		if storedHash == "" {
			hash, err := auth.HashPassword(password)
			if err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			if err := db.SetPasswordHash(database, hash); err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			storedHash = hash
		}

		if !auth.VerifyPassword(password, storedHash) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.ExecuteTemplate(w, "login.html", map[string]any{
				"Error": "Invalid password",
			})
			return
		}

		// Generate session token
		token := auth.GenerateToken()
		if err := db.CreateSession(database, token, 24); err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
		})

		http.Redirect(w, r, "/dashboard", http.StatusFound)
	}
}

func Logout(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err == nil {
			db.DeleteSession(database, cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})

		http.Redirect(w, r, "/login", http.StatusFound)
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/handlers/login.go
git commit -m "feat: add login, logout, and first-run password setup"
```

---

### Task 12: Token Generator

**Files:**
- Modify: `internal/auth/password.go` (add GenerateToken)

- [ ] **Step 1: Add GenerateToken to password.go**

```go
package auth

import (
	"crypto/rand"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateToken() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[n.Int64()]
	}
	return string(b)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/auth/password.go
git commit -m "feat: add cryptographically secure token generator"
```

---

### Task 13: Create Link Handler

**Files:**
- Create: `internal/handlers/create.go`

- [ ] **Step 1: Create create.go**

```go
package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"linkshortener/internal/auth"
	"linkshortener/internal/db"
)

func CreatePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ExecuteTemplate(w, "create.html", nil)
}

func CreatePost(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		url := r.FormValue("url")
		customCode := r.FormValue("custom_code")

		shortCode := customCode
		if shortCode == "" {
			shortCode = generateShortCode()
		}

		if len(shortCode) < 3 || len(shortCode) > 20 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.ExecuteTemplate(w, "create.html", map[string]any{
				"Error": "Short code must be 3-20 characters",
			})
			return
		}

		for _, c := range shortCode {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				templates.ExecuteTemplate(w, "create.html", map[string]any{
					"Error": "Short code must be alphanumeric",
				})
				return
			}
		}

		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.ExecuteTemplate(w, "create.html", map[string]any{
				"Error": "URL must start with http:// or https://",
			})
			return
		}

		if err := db.CreateLink(database, shortCode, url); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.ExecuteTemplate(w, "create.html", map[string]any{
				"Error": "Short code already exists",
			})
			return
		}

		http.Redirect(w, r, "/dashboard", http.StatusFound)
	}
}

func generateShortCode() string {
	token := auth.GenerateToken()
	return token[:6]
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/handlers/create.go
git commit -m "feat: add create link handler with validation"
```

---

### Task 14: Redirect Handler

**Files:**
- Create: `internal/handlers/redirect.go`

- [ ] **Step 1: Create redirect.go**

```go
package handlers

import (
	"database/sql"
	"net/http"

	"linkshortener/internal/db"
)

func Redirect(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortCode := chi.URLParam(r, "shortCode")

		link, err := db.GetLinkByShortCode(database, shortCode)
		if err == sql.ErrNoRows {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.ExecuteTemplate(w, "404.html", nil)
			return
		}
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		go db.IncrementClickCount(database, link.ID)

		http.Redirect(w, r, link.OriginalURL, http.StatusFound)
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/handlers/redirect.go
git commit -m "feat: add redirect handler with click tracking"
```

---

### Task 15: API Handlers

**Files:**
- Create: `internal/handlers/api.go`

- [ ] **Step 1: Create api.go**

```go
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"linkshortener/internal/db"
)

func APILinks(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		links, err := db.GetAllLinks(database)
		if err != nil {
			http.Error(w, `{"error":"Failed to fetch links"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(links)
	}
}

func APIDeleteLink(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		if err := db.DeleteLink(database, id); err != nil {
			http.Error(w, "Failed to delete", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/handlers/api.go
git commit -m "feat: add JSON API handlers"
```

---

### Task 16: Convert Templates to html/template

**Files:**
- Create: `templates/base.html` (rewrite)
- Create: `templates/login.html` (rewrite)
- Create: `templates/dashboard.html` (rewrite)
- Create: `templates/create.html` (rewrite)
- Create: `templates/404.html` (rewrite)

- [ ] **Step 1: Rewrite base.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{template "title" .}}</title>
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
        {{template "content" .}}
    </main>
</body>
</html>
```

- [ ] **Step 2: Rewrite login.html**

```html
{{define "title"}}Login - Link Shortener{{end}}

{{define "content"}}
<div class="card">
    <h2>Login</h2>
    <p>Enter the password to access the link shortener.</p>

    {{if .Error}}
    <div class="error">{{.Error}}</div>
    {{end}}

    <form method="POST" action="/login">
        <input type="password" name="password" placeholder="Password" required autofocus>
        <button type="submit" class="btn">Login</button>
    </form>
</div>
{{end}}
```

- [ ] **Step 3: Rewrite dashboard.html**

```html
{{define "title"}}Dashboard - Link Shortener{{end}}

{{define "content"}}
<div class="card">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
        <h2>Your Links</h2>
        <a href="/create" class="btn">Create New Link</a>
    </div>

    {{if .Links}}
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
            {{range .Links}}
            <tr>
                <td><a href="/{{.ShortCode}}" target="_blank" rel="noopener noreferrer">{{.ShortCode}}</a></td>
                <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{.OriginalURL}}</td>
                <td>{{.ClickCount}}</td>
                <td>{{.CreatedAt}}</td>
                <td class="actions">
                    <button class="btn btn-danger" onclick="deleteLink({{.ID}})">Delete</button>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{else}}
    <p>No links yet. <a href="/create">Create your first link</a>.</p>
    {{end}}
</div>

<script>
async function deleteLink(id) {
    if (confirm('Are you sure you want to delete this link?')) {
        await fetch(`/api/links/${id}`, { method: 'DELETE' });
        location.reload();
    }
}
</script>
{{end}}
```

- [ ] **Step 4: Rewrite create.html**

```html
{{define "title"}}Create Link - Link Shortener{{end}}

{{define "content"}}
<div class="card">
    <h2>Create New Link</h2>

    {{if .Error}}
    <div class="error">{{.Error}}</div>
    {{end}}

    {{if .Success}}
    <div class="success">
        Link created! Short URL: <a href="/{{.ShortCode}}">{{.ShortCode}}</a>
    </div>
    {{end}}

    <form method="POST" action="/create">
        <input type="url" name="url" placeholder="https://example.com/long-url" required>
        <input type="text" name="custom_code" placeholder="Custom short code (optional)">
        <button type="submit" class="btn">Create Link</button>
    </form>
</div>
{{end}}
```

- [ ] **Step 5: Rewrite 404.html**

```html
{{define "title"}}Not Found - Link Shortener{{end}}

{{define "content"}}
<div class="card">
    <h2>Link Not Found</h2>
    <p>The short link you're looking for doesn't exist or has been removed.</p>
    <p><a href="/" class="btn">Go to Dashboard</a></p>
</div>
{{end}}
```

- [ ] **Step 6: Commit**

```bash
git add templates/
git commit -m "feat: convert templates from Tera to html/template"
```

---

### Task 17: Main Entry Point

**Files:**
- Create: `cmd/linkshortener/main.go`

- [ ] **Step 1: Create main.go**

```go
package main

import (
	"log/slog"
	"net/http"

	"linkshortener/internal/auth"
	"linkshortener/internal/config"
	"linkshortener/internal/db"
	"linkshortener/internal/handlers"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.Load()

	database, err := db.Init(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		panic(err)
	}
	defer database.Close()

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Public routes
	r.Get("/login", handlers.LoginPage)
	r.Post("/login", handlers.LoginPost(database))
	r.Post("/logout", handlers.Logout(database))
	r.Get("/{shortCode}", handlers.Redirect(database))

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(database))
		r.Get("/", handlers.Dashboard(database))
		r.Get("/dashboard", handlers.Dashboard(database))
		r.Get("/create", handlers.CreatePage)
		r.Post("/create", handlers.CreatePost(database))
		r.Get("/api/links", handlers.APILinks(database))
		r.Delete("/api/links/{id}", handlers.APIDeleteLink(database))
	})

	slog.Info("link shortener listening", "addr", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, r); err != nil {
		slog.Error("server failed", "error", err)
	}
}
```

- [ ] **Step 2: Fetch all dependencies**

```bash
cd /home/pn/Projects/linkshortener
go mod tidy
```

- [ ] **Step 3: Build and verify**

```bash
go build -o linkshortener ./cmd/linkshortener
```

- [ ] **Step 4: Commit**

```bash
git add cmd/linkshortener/main.go go.mod go.sum
git commit -m "feat: add main entry point with Chi router"
```

---

### Task 18: Containerfile

**Files:**
- Modify: `Containerfile`

- [ ] **Step 1: Rewrite Containerfile**

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

- [ ] **Step 2: Commit**

```bash
git add Containerfile
git commit -m "feat: rewrite Containerfile for Go build"
```

---

### Task 19: Final Verification

**Files:**
- None (verification only)

- [ ] **Step 1: Build binary locally**

```bash
cd /home/pn/Projects/linkshortener
go build -o linkshortener ./cmd/linkshortener
```

- [ ] **Step 2: Run binary and verify it starts**

```bash
DATABASE_URL="sqlite::memory:" LISTEN_ADDR="127.0.0.1:8081" ./linkshortener &
sleep 2
curl -s http://127.0.0.1:8081/login | head -20
kill %1
```

- [ ] **Step 3: Build container image**

```bash
podman-compose build
```

- [ ] **Step 4: Commit final state**

```bash
git add -A
git commit -m "chore: Go rewrite complete - feature parity achieved"
```
