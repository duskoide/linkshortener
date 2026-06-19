# Dashboard UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the link shortener dashboard (and all shared pages) with a dark-terminal theme, adding copy-to-clipboard, search/filter, inline edit, QR codes, bulk delete, and toast notifications.

**Architecture:** Vanilla JS + embedded CSS served via `//go:embed`. Server-side html/template rendering enhanced by client-side JS. New Go API endpoints for QR (SVG), inline edit (PATCH), and bulk delete. QR codes generated server-side from `github.com/skip2/go-qrcode`'s bitmap, rendered as SVG manually (pure Go, no CGO).

**Tech Stack:** Go 1.25, Chi v5, SQLite (modernc.org/sqlite), html/template, `github.com/skip2/go-qrcode`, vanilla JS, CSS custom properties.

**Spec ref:** `docs/superpowers/specs/2026-06-19-dashboard-redesign-design.md`

**Testing note:** Per the approved spec and AGENTS.md, this project has no test infrastructure and no CI. Verification is manual. Each task includes exact verification commands and expected outcomes. If the user requests tests later, they can be added as a follow-up.

**Spec refinement:** The spec named `github.com/yeqown/go-qrcode` for QR generation. This plan uses `github.com/skip2/go-qrcode` instead because it exposes a `Bitmap()` method (`[][]bool`) that makes manual SVG rendering trivial, whereas yeqown's library is oriented toward image-format output (PNG/JPEG) and doesn't clearly expose the raw matrix. The spec's intent — pure Go, no CGO, server-side SVG — is preserved.

**Homeserver note:** Code changes are made locally in the repo. Deployment/verification to the homeserver happens via `ssh pn@homeserver` (see Task 18).

---

## File Structure

**Created:**
- `static/app.css` — all CSS (dark theme, tokens, layout, responsive, components)
- `static/app.js` — all vanilla JS (toasts, clipboard, filter, inline edit, QR, bulk)
- `internal/handlers/qr.go` — `GET /api/links/{id}/qr` SVG handler
- `internal/handlers/edit.go` — `PATCH /api/links/{id}` inline edit handler
- `internal/handlers/static.go` — `//go:embed static/*` + file server handler
- `internal/handlers/notfound.go` — Chi NotFound handler rendering 404.html

**Modified:**
- `internal/config/config.go` — add `BaseURL` field
- `.env.example` — add `BASE_URL`
- `internal/db/links.go` — add `UpdateLink`, `DeleteLinks`
- `internal/handlers/api.go` — add `APIBulkDeleteLink`
- `internal/handlers/dashboard.go` — pass `BaseURL` into template data
- `cmd/linkshortener/main.go` — register new routes, static handler, NotFound
- `templates/base.html` — full restyle (dark theme, topbar, mobile nav, asset links)
- `templates/dashboard.html` — full restructure (grid table, toolbar, empty states)
- `templates/login.html` — restyle
- `templates/create.html` — restyle
- `templates/404.html` — restyle
- `go.mod` / `go.sum` — add `github.com/skip2/go-qrcode`

---

### Task 1: Add BASE_URL to config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `.env.example`

- [ ] **Step 1: Add BaseURL field to Config struct and Load function**

Edit `internal/config/config.go`. Add `BaseURL` to the `Config` struct and load it in `Load()`.

Replace the entire file content with:

```go
package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	ListenAddr    string
	SessionSecret string
	TunnelToken   string
	BaseURL       string
}

func Load() *Config {
	godotenv.Load()

	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "sqlite:data/links.db"),
		ListenAddr:    getEnv("LISTEN_ADDR", "0.0.0.0:8081"),
		SessionSecret: getEnv("SESSION_SECRET", ""),
		TunnelToken:   getEnv("TUNNEL_TOKEN", ""),
		BaseURL:       getEnv("BASE_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 2: Add BASE_URL to .env.example**

Edit `.env.example`. Add `BASE_URL` after `SESSION_SECRET`:

```
DATABASE_URL=sqlite:data/links.db
LISTEN_ADDR=0.0.0.0:8081
SESSION_SECRET=change-me-to-random-string
BASE_URL=https://s.duskoide.org

# Cloudflare Tunnel
TUNNEL_TOKEN=your-tunnel-token-here
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/config/config.go .env.example
git commit -m "feat: add BASE_URL config for short URL construction"
```

---

### Task 2: Add go-qrcode dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

Run: `go get github.com/skip2/go-qrcode`
Expected: `go.mod` updated with `github.com/skip2/go-qrcode` in the require block.

- [ ] **Step 2: Verify it downloads and compiles**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add github.com/skip2/go-qrcode for QR generation"
```

---

### Task 3: Add DB functions — UpdateLink, DeleteLinks

**Files:**
- Modify: `internal/db/links.go`

- [ ] **Step 1: Add UpdateLink and DeleteLinks functions**

Edit `internal/db/links.go`. Add these two functions at the end of the file, after the existing `DeleteLink` function:

```go
func UpdateLink(db *sql.DB, linkID int64, shortCode, originalURL string) error {
	_, err := db.Exec(
		"UPDATE links SET short_code = ?, original_url = ? WHERE id = ?",
		shortCode, originalURL, linkID,
	)
	return err
}

func DeleteLinks(db *sql.DB, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := ""
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, id)
	}

	query := "DELETE FROM links WHERE id IN (" + placeholders + ")"
	_, err := db.Exec(query, args...)
	return err
}
```

Note: `UpdateLink` will return an error on UNIQUE constraint violation for `short_code` (the column has `UNIQUE NOT NULL` per `migrations/001_initial.sql:4`). The handler will check for this to return 409.

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/db/links.go
git commit -m "feat: add UpdateLink and DeleteLinks db functions"
```

---

### Task 4: QR code SVG handler

**Files:**
- Create: `internal/handlers/qr.go`

- [ ] **Step 1: Create the QR handler**

Create `internal/handlers/qr.go`:

```go
package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"linkshortener/internal/db"

	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
)

func APIQRCode(database *sql.DB, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		link, err := db.GetLinkByID(database, id)
		if err != nil {
			http.Error(w, "Link not found", http.StatusNotFound)
			return
		}

		shortURL := baseURL + "/" + link.ShortCode
		if baseURL == "" {
			shortURL = r.Host + "/" + link.ShortCode
			if r.TLS == nil {
				shortURL = "http://" + shortURL
			} else {
				shortURL = "https://" + shortURL
			}
		}

		qr, err := qrcode.New(shortURL, qrcode.Medium)
		if err != nil {
			http.Error(w, "Failed to generate QR", http.StatusInternalServerError)
			return
		}

		bitmap := qr.Bitmap()
		size := len(bitmap)
		cellSize := 8
		margin := 2
		labelHeight := 20
		totalSize := (size+margin*2)*cellSize + labelHeight

		var sb strings.Builder
		sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="`)
		sb.WriteString(strconv.Itoa((size+margin*2)*cellSize))
		sb.WriteString(`" height="`)
		sb.WriteString(strconv.Itoa(totalSize))
		sb.WriteString(`" viewBox="0 0 `)
		sb.WriteString(strconv.Itoa((size+margin*2)*cellSize))
		sb.WriteString(" ")
		sb.WriteString(strconv.Itoa(totalSize))
		sb.WriteString(`">`)

		bgSize := (size + margin*2) * cellSize
		sb.WriteString(`<rect width="`)
		sb.WriteString(strconv.Itoa(bgSize))
		sb.WriteString(`" height="`)
		sb.WriteString(strconv.Itoa(bgSize))
		sb.WriteString(`" fill="#ffffff"/>`)

		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if bitmap[y][x] {
					px := (x + margin) * cellSize
					py := (y + margin) * cellSize
					sb.WriteString(`<rect x="`)
					sb.WriteString(strconv.Itoa(px))
					sb.WriteString(`" y="`)
					sb.WriteString(strconv.Itoa(py))
					sb.WriteString(`" width="`)
					sb.WriteString(strconv.Itoa(cellSize))
					sb.WriteString(`" height="`)
					sb.WriteString(strconv.Itoa(cellSize))
					sb.WriteString(`" fill="#020617"/>`)
				}
			}
		}

		labelY := bgSize + 14
		sb.WriteString(`<text x="`)
		sb.WriteString(strconv.Itoa(bgSize / 2))
		sb.WriteString(`" y="`)
		sb.WriteString(str strconv.Itoa(labelY))
		sb.WriteString(`" font-family="monospace" font-size="11" fill="#020617" text-anchor="middle">`)
		sb.WriteString(shortURL)
		sb.WriteString(`</text>`)
		sb.WriteString(`</svg>`)

		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(sb.String()))
	}
}
```

Wait — there's a typo in the code above (`str strconv.Itoa`). Fix it. The correct line is:

```go
		sb.WriteString(strconv.Itoa(labelY))
```

- [ ] **Step 2: Add GetLinkByID to db package**

The QR handler needs `db.GetLinkByID`. Edit `internal/db/links.go` and add this function after `GetLinkByShortCode`:

```go
func GetLinkByID(db *sql.DB, id int64) (*Link, error) {
	link := &Link{}
	err := db.QueryRow(
		"SELECT id, short_code, original_url, created_at, click_count FROM links WHERE id = ?",
		id,
	).Scan(&link.ID, &link.ShortCode, &link.OriginalURL, &link.CreatedAt, &link.ClickCount)
	if err != nil {
		return nil, err
	}
	return link, nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/qr.go internal/db/links.go
git commit -m "feat: add QR code SVG endpoint at GET /api/links/{id}/qr"
```

---

### Task 5: Inline edit handler

**Files:**
- Create: `internal/handlers/edit.go`

- [ ] **Step 1: Create the inline edit handler**

Create `internal/handlers/edit.go`:

```go
package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"linkshortener/internal/db"

	"github.com/go-chi/chi/v5"
)

var shortCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type editRequest struct {
	ShortCode   string `json:"short_code"`
	OriginalURL string `json:"original_url"`
}

func APIEditLink(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"Invalid ID"}`, http.StatusBadRequest)
			return
		}

		var req editRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
			return
		}

		req.ShortCode = strings.TrimSpace(req.ShortCode)
		req.OriginalURL = strings.TrimSpace(req.OriginalURL)

		if req.ShortCode == "" || len(req.ShortCode) > 64 || !shortCodePattern.MatchString(req.ShortCode) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Short code must be 1-64 chars, alphanumeric/underscore/hyphen only"}`))
			return
		}

		parsedURL, err := url.Parse(req.OriginalURL)
		if err != nil || parsedURL.Scheme == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"URL must be a valid http(s) URL"}`))
			return
		}

		err = db.UpdateLink(database, id, req.ShortCode, req.OriginalURL)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"error":"Short code already taken"}`))
				return
			}
			http.Error(w, `{"error":"Failed to update"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}
}
```

Note: SQLite's UNIQUE constraint violation error message contains "UNIQUE" — this is how we detect the 409 collision. The `errors` import is included for potential future use but can be removed if `go vet` complains about unused imports. Actually — remove the `"errors"` import since it's not used. The corrected import block should not include `"errors"`.

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: no errors. If `errors` import causes an "imported and not used" error, remove it from the import block.

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/edit.go
git commit -m "feat: add PATCH /api/links/{id} for inline editing"
```

---

### Task 6: Bulk delete handler

**Files:**
- Modify: `internal/handlers/api.go`

- [ ] **Step 1: Add APIBulkDeleteLink handler**

Edit `internal/handlers/api.go`. Add this function at the end of the file:

```go
func APIBulkDeleteLink(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IDs []int64 `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
			return
		}

		if len(req.IDs) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"deleted":0}`))
			return
		}

		if len(req.IDs) > 1000 {
			http.Error(w, `{"error":"Too many IDs (max 1000)"}`, http.StatusBadRequest)
			return
		}

		if err := db.DeleteLinks(database, req.IDs); err != nil {
			http.Error(w, `{"error":"Failed to delete"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"deleted":` + strconv.FormatInt(int64(len(req.IDs)), 10) + `}`))
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/api.go
git commit -m "feat: add DELETE /api/links/bulk for bulk deletion"
```

---

### Task 7: Static asset serving

**Files:**
- Create: `internal/handlers/static.go`
- Modify: `cmd/linkshortener/main.go`
- Create: `static/app.css` (placeholder — real CSS in Task 8)
- Create: `static/app.js` (placeholder — real JS in Tasks 11-15)

- [ ] **Step 1: Create the static directory with placeholder files**

Create `static/app.css` with a single comment placeholder:

```css
/* loaded in Task 8 */
```

Create `static/app.js` with a placeholder:

```javascript
/* loaded in Task 11 */
```

- [ ] **Step 2: Create static.go with embed**

Create `internal/handlers/static.go`:

```go
package handlers

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}
```

Wait — the `//go:embed static` directive looks for a `static/` directory relative to the `internal/handlers/` package directory. That won't work because `static/` is at the project root, not inside `internal/handlers/`.

There are two approaches:
1. Move the embed to `main.go` (project root) where `static/` is accessible.
2. Create the `static/` directory inside `internal/handlers/`.

Approach 1 is cleaner — the embed lives at the project root alongside `templates/`. Let me revise.

- [ ] **Step 2 (revised): Create static.go at project root level**

Actually, Go's `//go:embed` can only embed files in the same directory or subdirectories of the package. Since `static/` is at the project root, the embed must be in a file at the project root (e.g., `main.go` or a root-level package file).

The simplest approach: put the embed in `cmd/linkshortener/main.go` since that's at a subdirectory. Hmm, that won't work either — `cmd/linkshortener/` is not the project root.

Best approach: create a small package at the project root, e.g. `internal/static/static.go`, and put the `static/` directory there. Or, move `static/` inside `internal/handlers/`.

Let me move the static files inside `internal/handlers/static/`. This way `internal/handlers/static.go` can embed them:

Delete the project-root `static/` directory if created. Instead:

Create `internal/handlers/static/app.css`:

```css
/* loaded in Task 8 */
```

Create `internal/handlers/static/app.js`:

```javascript
/* loaded in Task 11 */
```

Create `internal/handlers/static.go`:

```go
package handlers

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}
```

- [ ] **Step 3: Register the static route in main.go**

Edit `cmd/linkshortener/main.go`. Add the static route in the public routes section (before the protected group). After the `r.Get("/{shortCode}", ...)` line, add:

```go
	r.Handle("/static/*", handlers.StaticHandler())
```

The public routes section should now look like:

```go
	// Public routes
	r.Get("/login", handlers.LoginPage)
	r.Post("/login", handlers.LoginPost(database))
	r.Post("/logout", handlers.Logout(database))
	r.Get("/{shortCode}", handlers.Redirect(database))
	r.Handle("/static/*", handlers.StaticHandler())
```

Note: `/{shortCode}` is registered before `/static/*`. Chi matches routes in order, and `/{shortCode}` is a catch-all parameter route. However, Chi's routing is radix-based, not order-based — it will match `/static/*` correctly because the literal `static` segment takes priority over the parameter. If there are issues, move `/static/*` before `/{shortCode}`.

Actually, to be safe, register `/static/*` **before** `/{shortCode}`:

```go
	// Public routes
	r.Get("/login", handlers.LoginPage)
	r.Post("/login", handlers.LoginPost(database))
	r.Post("/logout", handlers.Logout(database))
	r.Handle("/static/*", handlers.StaticHandler())
	r.Get("/{shortCode}", handlers.Redirect(database))
```

- [ ] **Step 4: Verify compilation and serving**

Run: `go build ./...`
Expected: no errors.

Run: `go run ./cmd/linkshortener &`
Then: `curl -s http://localhost:8081/static/app.css`
Expected: `/* loaded in Task 8 */`

Kill the server: `kill %1`

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/static.go internal/handlers/static/ cmd/linkshortener/main.go
git commit -m "feat: add embedded static asset serving at /static/*"
```

---

### Task 8: CSS — dark terminal theme

**Files:**
- Modify: `internal/handlers/static/app.css` (replace placeholder with full CSS)

- [ ] **Step 1: Write the complete CSS**

Replace the contents of `internal/handlers/static/app.css` with:

```css
:root {
	--bg-base: #0f172a;
	--bg-elevated: #1e293b;
	--bg-topbar: #020617;
	--border: #334155;
	--text-primary: #e2e8f0;
	--text-muted: #64748b;
	--accent: #22d3ee;
	--code-color: #34d399;
	--clicks-color: #fbbf24;
	--danger: #f87171;
	--danger-border: #7f1d1d;
	--success-bg: #022c22;
	--success-border: #34d399;
	--success-text: #6ee7b7;
	--radius: 5px;
	--font: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

* {
	margin: 0;
	padding: 0;
	box-sizing: border-box;
}

body {
	font-family: var(--font);
	background: var(--bg-base);
	color: var(--text-primary);
	line-height: 1.5;
}

a {
	color: var(--accent);
	text-decoration: none;
}

a:hover {
	text-decoration: underline;
}

.topbar {
	background: var(--bg-topbar);
	border-bottom: 1px solid var(--border);
	padding: 10px 16px;
	position: sticky;
	top: 0;
	z-index: 100;
}

.topbar-inner {
	max-width: 1100px;
	margin: 0 auto;
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.wordmark {
	color: var(--accent);
	font-weight: 700;
	font-size: 14px;
}

.wordmark:hover {
	text-decoration: none;
}

.breadcrumb {
	color: var(--text-muted);
	font-size: 11px;
}

.nav-desktop {
	display: flex;
	gap: 14px;
	font-size: 12px;
}

.nav-desktop a {
	color: var(--text-muted);
}

.nav-desktop a.active {
	color: var(--text-primary);
	border-bottom: 1px solid var(--accent);
	padding-bottom: 2px;
}

.nav-desktop a:hover {
	color: var(--text-primary);
	text-decoration: none;
}

.hamburger {
	display: none;
	cursor: pointer;
	color: var(--text-muted);
	font-size: 18px;
	user-select: none;
}

.nav-toggle {
	display: none;
}

.container {
	max-width: 1100px;
	margin: 0 auto;
	padding: 20px 16px;
}

.card {
	background: var(--bg-elevated);
	border-radius: var(--radius);
	padding: 1.5rem;
	margin-bottom: 1rem;
	border: 1px solid var(--border);
}

.btn {
	display: inline-block;
	background: var(--accent);
	color: var(--bg-topbar);
	padding: 6px 14px;
	border: none;
	border-radius: 4px;
	cursor: pointer;
	font-family: var(--font);
	font-size: 12px;
	font-weight: 700;
	text-decoration: none;
}

.btn:hover {
	opacity: 0.85;
	text-decoration: none;
}

.btn:disabled {
	opacity: 0.3;
	cursor: not-allowed;
}

.btn-outline {
	background: transparent;
	border: 1px solid var(--border);
	color: var(--text-muted);
}

.btn-outline:hover {
	border-color: var(--accent);
	color: var(--accent);
	opacity: 1;
}

.btn-danger {
	background: transparent;
	border: 1px solid var(--danger-border);
	color: var(--danger);
}

.btn-danger:hover {
	background: var(--danger);
	color: var(--bg-topbar);
	opacity: 1;
}

input,
textarea {
	width: 100%;
	padding: 8px 10px;
	background: var(--bg-topbar);
	border: 1px solid var(--border);
	border-radius: 4px;
	color: var(--text-primary);
	font-family: var(--font);
	font-size: 13px;
}

input:focus {
	outline: none;
	border-color: var(--accent);
}

input::placeholder {
	color: var(--text-muted);
}

.error {
	background: rgba(248, 113, 113, 0.1);
	border: 1px solid var(--danger);
	color: var(--danger);
	padding: 10px 14px;
	border-radius: 4px;
	margin-bottom: 1rem;
	font-size: 12px;
}

/* Dashboard toolbar */
.toolbar {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 12px;
	margin-bottom: 14px;
	flex-wrap: wrap;
}

.filter-group {
	display: flex;
	align-items: center;
	gap: 8px;
	flex: 1;
	min-width: 200px;
}

.filter-prefix {
	color: var(--text-muted);
	font-size: 12px;
}

.filter-input {
	flex: 1;
	padding: 5px 9px;
	font-size: 12px;
}

.toolbar-right {
	display: flex;
	gap: 8px;
	align-items: center;
}

.selection-count {
	color: var(--text-muted);
	font-size: 11px;
	min-width: 70px;
}

/* Table */
.table-wrap {
	background: var(--bg-elevated);
	border-radius: var(--radius);
	border: 1px solid var(--border);
	overflow: hidden;
}

.row {
	display: grid;
	grid-template-columns: 24px 70px 1fr 56px 64px 150px;
	grid-template-areas: "chk code url clk created actions";
	gap: 8px;
	padding: 8px 10px;
	align-items: center;
	font-size: 12px;
	border-bottom: 1px solid var(--border);
}

.row:last-child {
	border-bottom: none;
}

.row.header {
	background: var(--bg-topbar);
	text-transform: uppercase;
	letter-spacing: 0.05em;
	font-size: 9px;
	color: var(--text-muted);
}

.row.is-selected {
	background: rgba(34, 211, 238, 0.04);
}

.row.is-editing .cell-code,
.row.is-editing .cell-url {
	padding: 0;
}

.row.is-editing .cell-code input,
.row.is-editing .cell-url input {
	padding: 3px 5px;
	font-size: 12px;
}

.row.is-deleting {
	opacity: 0.5;
	pointer-events: none;
}

.cell-chk { grid-area: chk; }
.cell-code { grid-area: code; }
.cell-url { grid-area: url; }
.cell-clk { grid-area: clk; }
.cell-created { grid-area: created; }
.cell-actions { grid-area: actions; }

.cell-code a,
.cell-code span {
	color: var(--code-color);
	font-weight: 600;
}

.cell-url {
	color: var(--text-muted);
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.cell-clk {
	color: var(--clicks-color);
}

.cell-created {
	color: var(--text-muted);
	font-size: 11px;
}

.cell-actions {
	display: flex;
	gap: 6px;
	justify-content: flex-end;
}

.icon-btn {
	color: var(--text-muted);
	border: 1px solid var(--border);
	border-radius: 3px;
	padding: 1px 6px;
	font-size: 11px;
	cursor: pointer;
	background: transparent;
	font-family: var(--font);
	line-height: 1.4;
}

.icon-btn:hover {
	color: var(--accent);
	border-color: var(--accent);
}

.icon-btn.danger:hover {
	color: var(--danger);
	border-color: var(--danger);
}

.icon-btn.save {
	color: var(--code-color);
	border-color: var(--code-color);
}

/* Checkbox */
.row input[type="checkbox"] {
	width: 14px;
	height: 14px;
	accent-color: var(--accent);
	cursor: pointer;
}

/* Table footer */
.table-footer {
	color: var(--text-muted);
	font-size: 10px;
	padding: 10px 16px;
	border-top: 1px solid var(--border);
}

/* Empty states */
.empty-state {
	text-align: center;
	padding: 40px 20px;
	color: var(--text-muted);
	font-size: 13px;
}

.empty-state .btn {
	margin-top: 12px;
}

.no-match {
	color: var(--text-muted);
	font-size: 12px;
	padding: 20px;
	text-align: center;
}

/* QR popover */
.qr-popover {
	position: absolute;
	background: var(--bg-topbar);
	border: 1px solid var(--border);
	border-radius: var(--radius);
	padding: 10px;
	z-index: 200;
}

.qr-popover svg {
	display: block;
}

.qr-backdrop {
	position: fixed;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	z-index: 150;
}

/* Toasts */
.toast-container {
	position: fixed;
	bottom: 16px;
	right: 16px;
	z-index: 300;
	display: flex;
	flex-direction: column;
	gap: 8px;
}

.toast {
	display: flex;
	align-items: center;
	gap: 8px;
	padding: 8px 14px;
	border-radius: 4px;
	font-size: 12px;
	animation: toast-in 0.2s ease;
}

.toast.success {
	background: var(--success-bg);
	border: 1px solid var(--success-border);
	color: var(--success-text);
}

.toast.error {
	background: rgba(248, 113, 113, 0.1);
	border: 1px solid var(--danger);
	color: var(--danger);
}

.toast.info {
	background: rgba(34, 211, 238, 0.1);
	border: 1px solid var(--accent);
	color: var(--accent);
}

.toast-dismiss {
	cursor: pointer;
	opacity: 0.6;
	margin-left: 4px;
}

.toast-dismiss:hover {
	opacity: 1;
}

@keyframes toast-in {
	from { opacity: 0; transform: translateY(10px); }
	to { opacity: 1; transform: translateY(0); }
}

/* Centered card (login, create) */
.center-card {
	max-width: 360px;
	margin: 40px auto;
}

.center-card h2 {
	color: var(--text-primary);
	font-size: 14px;
	margin-bottom: 14px;
}

.center-card .hint {
	color: var(--text-muted);
	font-size: 11px;
	margin-top: 10px;
	text-align: center;
}

.center-card input {
	margin-bottom: 10px;
}

.center-card .btn {
	width: 100%;
	text-align: center;
}

/* 404 */
.notfound {
	text-align: center;
	padding: 60px 20px;
}

.notfound-code {
	color: var(--accent);
	font-size: 48px;
	font-weight: 700;
}

.notfound-msg {
	color: var(--text-muted);
	font-size: 13px;
	margin: 8px 0 20px;
}

/* Mobile nav sheet */
.nav-sheet {
	display: none;
	position: fixed;
	top: 0;
	left: 0;
	width: 58%;
	height: 100%;
	background: var(--bg-elevated);
	border-right: 1px solid var(--border);
	padding: 16px;
	z-index: 200;
	flex-direction: column;
	gap: 8px;
}

.nav-sheet-label {
	color: var(--text-muted);
	font-size: 9px;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	margin-bottom: 10px;
}

.nav-sheet a {
	color: var(--text-muted);
	font-size: 13px;
	padding-left: 10px;
}

.nav-sheet a.active {
	color: var(--text-primary);
	border-left: 2px solid var(--accent);
	padding-left: 8px;
}

.nav-backdrop {
	display: none;
	position: fixed;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	background: rgba(2, 6, 23, 0.6);
	z-index: 150;
}

.nav-toggle:checked ~ .nav-backdrop {
	display: block;
}

.nav-toggle:checked ~ .nav-sheet {
	display: flex;
}

/* Responsive */
@media (max-width: 640px) {
	.hamburger {
		display: block;
	}

	.nav-desktop {
		display: none;
	}

	.topbar-inner {
		gap: 10px;
	}

	.row {
		grid-template-columns: auto 1fr auto;
		grid-template-areas:
			"chk code clk"
			"url url url"
			"created actions actions";
		gap: 6px;
		padding: 10px;
		margin-bottom: 8px;
		border-radius: var(--radius);
		border: 1px solid var(--border);
	}

	.row.header {
		display: none;
	}

	.row.is-selected {
		border-color: var(--accent);
	}

	.table-wrap {
		background: transparent;
		border: none;
	}

	.toolbar {
		flex-direction: column;
		align-items: stretch;
	}

	.toolbar-right {
		justify-content: space-between;
	}

	.cell-url {
		font-size: 11px;
	}

	.cell-actions {
		justify-content: flex-end;
	}
}
```

- [ ] **Step 2: Verify it serves correctly**

Run: `go run ./cmd/linkshortener &`
Then: `curl -s http://localhost:8081/static/app.css | head -5`
Expected: the CSS custom properties block starting with `:root {`

Kill the server: `kill %1`

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/static/app.css
git commit -m "feat: add dark-terminal theme CSS"
```

---

### Task 9: base.html restyle

**Files:**
- Modify: `templates/base.html`

- [ ] **Step 1: Replace base.html with the dark-terminal layout**

Replace the entire contents of `templates/base.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>{{template "title" .}}</title>
	<link rel="stylesheet" href="/static/app.css">
</head>
<body>
	<input type="checkbox" id="nav-toggle" class="nav-toggle">
	<div class="nav-backdrop" onclick="document.getElementById('nav-toggle').checked=false"></div>
	<nav class="nav-sheet">
		<div class="nav-sheet-label">~ navigate</div>
		<a href="/dashboard">dashboard</a>
		<a href="/create">create</a>
		<a href="/logout" onclick="event.preventDefault();document.forms['logout-form'].submit()">logout</a>
		<form name="logout-form" method="POST" action="/logout" style="display:none"></form>
	</nav>

	<header class="topbar">
		<div class="topbar-inner">
			<div style="display:flex;align-items:center;gap:10px;">
				<label for="nav-toggle" class="hamburger">&#9776;</label>
				<a href="/" class="wordmark">&#9181; link-shortener</a>
				<span class="breadcrumb">{{template "breadcrumb" .}}</span>
			</div>
			<nav class="nav-desktop">
				{{template "nav" .}}
			</nav>
		</div>
	</header>

	<main class="container">
		{{template "content" .}}
	</main>

	<div class="toast-container" id="toast-container"></div>
	<script src="/static/app.js" defer></script>
</body>
</html>
```

Note: The base template now defines slots for `"breadcrumb"` and `"nav"` so each page can customize them. Every page template must define these blocks (or use a default empty definition).

- [ ] **Step 2: Add default empty blocks for breadcrumb and nav**

Since Go's html/template requires all referenced templates to be defined, add default definitions at the top of `base.html` (before the `<!DOCTYPE>` won't work — they must be in the same parse set). Actually, the simplest approach: each page template defines `"breadcrumb"` and `"nav"` blocks. Let me handle this in the individual template tasks (Tasks 10, 16).

For now, to verify base.html compiles, we need the blocks defined somewhere in the parse set. Each page template (dashboard.html, login.html, create.html, 404.html) must include:

```html
{{define "breadcrumb"}}{{end}}
{{define "nav"}}{{end}}
```

These will be added per-template in their respective tasks. For now, the base.html change is complete.

- [ ] **Step 3: Verify compilation**

Run: `go build ./...`
Expected: no errors (template parsing happens at runtime, not compile time).

- [ ] **Step 4: Commit**

```bash
git add templates/base.html
git commit -m "feat: restyle base.html with dark-terminal theme and mobile nav"
```

---

### Task 10: Dashboard template + handler restructure

**Files:**
- Modify: `templates/dashboard.html`
- Modify: `internal/handlers/dashboard.go`

- [ ] **Step 1: Replace dashboard.html with the new structure**

Replace the entire contents of `templates/dashboard.html`:

```html
{{define "title"}}Dashboard - Link Shortener{{end}}
{{define "breadcrumb"}}~ /dashboard{{end}}
{{define "nav"}}
	<a href="/dashboard" class="active">dashboard</a>
	<a href="/create">create</a>
	<a href="/logout" onclick="event.preventDefault();document.forms['logout-form'].submit()">logout</a>
{{end}}

{{define "content"}}
<div class="card">
	<div class="toolbar">
		<div class="filter-group">
			<span class="filter-prefix">$</span>
			<input type="text" class="filter-input" id="filter" placeholder="filter by code or url...">
		</div>
		<div class="toolbar-right">
			<span class="selection-count" id="selection-count">0 selected</span>
			<button class="btn btn-outline" id="bulk-delete" disabled>delete selected</button>
			<a href="/create" class="btn">+ new</a>
		</div>
	</div>

	{{if .Links}}
	<div class="table-wrap" data-base-url="{{.BaseURL}}">
		<div class="row header">
			<div class="cell-chk"><input type="checkbox" id="select-all"></div>
			<div class="cell-code">code</div>
			<div class="cell-url">destination</div>
			<div class="cell-clk">clicks</div>
			<div class="cell-created">created</div>
			<div class="cell-actions" style="text-align:right;">actions</div>
		</div>
		{{range .Links}}
		<div class="row" data-id="{{.ID}}" data-code="{{.ShortCode}}" data-url="{{.OriginalURL}}">
			<div class="cell-chk"><input type="checkbox" class="row-checkbox"></div>
			<div class="cell-code"><a href="/{{.ShortCode}}" target="_blank" rel="noopener noreferrer">/{{.ShortCode}}</a></div>
			<div class="cell-url">{{.OriginalURL}}</div>
			<div class="cell-clk">{{.ClickCount}}&#8599;</div>
			<div class="cell-created">{{.CreatedAt.Format "Jan 02"}}</div>
			<div class="cell-actions">
				<button class="icon-btn" data-action="copy">&#8853;</button>
				<button class="icon-btn" data-action="qr">&#9638;</button>
				<button class="icon-btn" data-action="edit">&#9998;</button>
				<button class="icon-btn danger" data-action="delete">&#10005;</button>
			</div>
		</div>
		{{end}}
	</div>
	<div class="table-footer" id="table-footer">showing {{len .Links}} of {{len .Links}}</div>
	{{else}}
	<div class="empty-state">
		no links yet
		<br><a href="/create" class="btn">create your first link</a>
	</div>
	{{end}}
</div>
{{end}}
```

Note: `{{.CreatedAt.Format "Jan 02"}}` uses Go's time formatting. The `Link.CreatedAt` field is `time.Time`, so this works.

- [ ] **Step 2: Update dashboard handler to pass BaseURL**

Edit `internal/handlers/dashboard.go`. Replace the entire file:

```go
package handlers

import (
	"database/sql"
	"net/http"

	"linkshortener/internal/db"
)

func Dashboard(database *sql.DB, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		links, err := db.GetAllLinks(database)
		if err != nil {
			http.Error(w, "Failed to fetch links", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderTemplate(w, "dashboard.html", map[string]any{
			"Links":   links,
			"BaseURL": baseURL,
		})
	}
}
```

- [ ] **Step 3: Update main.go to pass baseURL to Dashboard**

Edit `cmd/linkshortener/main.go`. In the protected routes section, change the Dashboard calls to pass `cfg.BaseURL`:

```go
		r.Get("/", handlers.Dashboard(database, cfg.BaseURL))
		r.Get("/dashboard", handlers.Dashboard(database, cfg.BaseURL))
```

Also register the new API routes in the protected group:

```go
		r.Get("/api/links", handlers.APILinks(database))
		r.Delete("/api/links/{id}", handlers.APIDeleteLink(database))
		r.Get("/api/links/{id}/qr", handlers.APIQRCode(database, cfg.BaseURL))
		r.Patch("/api/links/{id}", handlers.APIEditLink(database))
		r.Delete("/api/links/bulk", handlers.APIBulkDeleteLink(database))
```

- [ ] **Step 4: Verify it runs and renders**

Run: `go run ./cmd/linkshortener &`
Then: `curl -s http://localhost:8081/login | head -20`
Expected: HTML output with dark theme CSS link.

Kill the server: `kill %1`

- [ ] **Step 5: Commit**

```bash
git add templates/dashboard.html internal/handlers/dashboard.go cmd/linkshortener/main.go
git commit -m "feat: restructure dashboard template with grid table, toolbar, empty states"
```

---

### Task 11: JavaScript — toasts + clipboard

**Files:**
- Modify: `internal/handlers/static/app.js` (replace placeholder)

- [ ] **Step 1: Write the toast manager and clipboard functions**

Replace the entire contents of `internal/handlers/static/app.js`:

```javascript
(function () {
	var toastContainer = document.getElementById('toast-container');
	var toastTimers = {};

	function showToast(message, type) {
		type = type || 'info';
		while (toastContainer.children.length >= 3) {
			toastContainer.removeChild(toastContainer.firstChild);
		}

		var toast = document.createElement('div');
		toast.className = 'toast ' + type;

		var text = document.createElement('span');
		text.textContent = message;

		var dismiss = document.createElement('span');
		dismiss.className = 'toast-dismiss';
		dismiss.textContent = '\u00d7';
		dismiss.onclick = function () { removeToast(toast); };

		toast.appendChild(text);
		toast.appendChild(dismiss);
		toastContainer.appendChild(toast);

		toast.onmouseenter = function () {
			clearTimeout(toastTimers[toast._id]);
		};
		toast.onmouseleave = function () {
			startTimer(toast);
		};

		toast._id = 'toast-' + Date.now() + '-' + Math.random();
		startTimer(toast);
	}

	function startTimer(toast) {
		toastTimers[toast._id] = setTimeout(function () {
			removeToast(toast);
		}, 2500);
	}

	function removeToast(toast) {
		if (toast.parentNode) {
			toast.parentNode.removeChild(toast);
		}
		if (toast._id) {
			clearTimeout(toastTimers[toast._id]);
			delete toastTimers[toast._id];
		}
	}

	function copyToClipboard(text) {
		if (navigator.clipboard && navigator.clipboard.writeText) {
			navigator.clipboard.writeText(text).then(function () {
				showToast('copied ' + text, 'success');
			}).catch(function () {
				fallbackCopy(text);
			});
		} else {
			fallbackCopy(text);
		}
	}

	function fallbackCopy(text) {
		var input = document.createElement('input');
		input.style.position = 'fixed';
		input.style.opacity = '0';
		input.value = text;
		document.body.appendChild(input);
		input.select();
		try {
			document.execCommand('copy');
			showToast('copied ' + text, 'success');
		} catch (e) {
			showToast('failed to copy', 'error');
		}
		document.body.removeChild(input);
	}

	window.App = {
		showToast: showToast,
		copyToClipboard: copyToClipboard
	};

	document.addEventListener('DOMContentLoaded', function () {
		var table = document.querySelector('.table-wrap');
		if (!table) return;

		var baseURL = table.getAttribute('data-base-url') || '';
		var location = window.location.origin;

		function buildShortURL(code) {
			if (baseURL) {
				return baseURL + '/' + code;
			}
			return location + '/' + code;
		}

		table.addEventListener('click', function (e) {
			var btn = e.target.closest('[data-action="copy"]');
			if (!btn) return;
			var row = btn.closest('.row');
			var code = row.getAttribute('data-code');
			var url = buildShortURL(code);
			copyToClipboard(url);
		});
	});
})();
```

- [ ] **Step 2: Verify it serves**

Run: `go run ./cmd/linkshortener &`
Then: `curl -s http://localhost:8081/static/app.js | head -5`
Expected: `(function () {`

Kill the server: `kill %1`

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/static/app.js
git commit -m "feat: add toast manager and clipboard copy functionality"
```

---

### Task 12: JavaScript — filter

**Files:**
- Modify: `internal/handlers/static/app.js`

- [ ] **Step 1: Add filter logic inside the DOMContentLoaded callback**

Edit `internal/handlers/static/app.js`. Inside the `document.addEventListener('DOMContentLoaded', function () { ... })` callback, after the `table.addEventListener('click', ...)` block (but before the closing `});`), add:

```javascript
		var filterInput = document.getElementById('filter');
		if (filterInput) {
			filterInput.addEventListener('input', function () {
				var query = filterInput.value.toLowerCase();
				var rows = table.querySelectorAll('.row:not(.header)');
				var visible = 0;
				rows.forEach(function (row) {
					var code = (row.getAttribute('data-code') || '').toLowerCase();
					var url = (row.getAttribute('data-url') || '').toLowerCase();
					var match = code.indexOf(query) !== -1 || url.indexOf(query) !== -1;
					row.style.display = match ? '' : 'none';
					if (match) visible++;
				});

				var footer = document.getElementById('table-footer');
				if (footer) {
					footer.textContent = 'showing ' + visible + ' of ' + rows.length;
				}

				var noMatch = table.querySelector('.no-match');
				if (noMatch) noMatch.remove();

				if (visible === 0 && query !== '') {
					noMatch = document.createElement('div');
					noMatch.className = 'no-match';
					noMatch.textContent = 'no matches for "' + query + '"';
					table.appendChild(noMatch);
				}
			});
		}
```

- [ ] **Step 2: Verify it serves**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/static/app.js
git commit -m "feat: add client-side filter for links"
```

---

### Task 13: JavaScript — inline edit

**Files:**
- Modify: `internal/handlers/static/app.js`

- [ ] **Step 1: Add inline edit logic**

Edit `internal/handlers/static/app.js`. Inside the DOMContentLoaded callback, after the filter logic, add:

```javascript
		var editingRow = null;

		function enterEditMode(row) {
			if (editingRow && editingRow !== row) {
				exitEditMode(editingRow, false);
			}
			editingRow = row;
			row.classList.add('is-editing');

			var codeCell = row.querySelector('.cell-code');
			var urlCell = row.querySelector('.cell-url');
			var actionsCell = row.querySelector('.cell-actions');

			var code = row.getAttribute('data-code');
			var url = row.getAttribute('data-url');

			codeCell.innerHTML = '<input type="text" class="edit-code" value="' + code + '">';
			urlCell.innerHTML = '<input type="text" class="edit-url" value="' + url + '">';

			actionsCell.innerHTML = '<button class="icon-btn save" data-action="save">save</button><button class="icon-btn" data-action="cancel">esc</button>';

			var codeInput = codeCell.querySelector('input');
			codeInput.focus();
			codeInput.select();

			codeInput.addEventListener('keydown', function (e) {
				if (e.key === 'Enter') saveEdit(row);
				if (e.key === 'Escape') exitEditMode(row, false);
			});
			urlCell.querySelector('input').addEventListener('keydown', function (e) {
				if (e.key === 'Enter') saveEdit(row);
				if (e.key === 'Escape') exitEditMode(row, false);
			});
		}

		function exitEditMode(row, saved) {
			row.classList.remove('is-editing');
			if (editingRow === row) editingRow = null;

			if (!saved) {
				var code = row.getAttribute('data-code');
				var url = row.getAttribute('data-url');
				row.querySelector('.cell-code').innerHTML = '<a href="/' + code + '" target="_blank" rel="noopener noreferrer">/' + code + '</a>';
				row.querySelector('.cell-url').textContent = url;
				restoreActions(row);
			}
		}

		function restoreActions(row) {
			row.querySelector('.cell-actions').innerHTML =
				'<button class="icon-btn" data-action="copy">\u2295</button>' +
				'<button class="icon-btn" data-action="qr">\u25a6</button>' +
				'<button class="icon-btn" data-action="edit">\u270e</button>' +
				'<button class="icon-btn danger" data-action="delete">\u2715</button>';
		}

		function saveEdit(row) {
			var id = row.getAttribute('data-id');
			var codeInput = row.querySelector('.edit-code');
			var urlInput = row.querySelector('.edit-url');
			var newCode = codeInput.value.trim();
			var newUrl = urlInput.value.trim();

			fetch('/api/links/' + id, {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ short_code: newCode, original_url: newUrl })
			}).then(function (res) {
				if (res.ok) {
					row.setAttribute('data-code', newCode);
					row.setAttribute('data-url', newUrl);
					exitEditMode(row, true);
					row.querySelector('.cell-code').innerHTML = '<a href="/' + newCode + '" target="_blank" rel="noopener noreferrer">/' + newCode + '</a>';
					row.querySelector('.cell-url').textContent = newUrl;
					restoreActions(row);
					showToast('updated /' + newCode, 'success');
				} else if (res.status === 409) {
					showToast('code already taken', 'error');
				} else {
					return res.json().then(function (data) {
						showToast(data.error || 'update failed', 'error');
					});
				}
			}).catch(function () {
				showToast('network error', 'error');
			});
		}

		table.addEventListener('click', function (e) {
			var editBtn = e.target.closest('[data-action="edit"]');
			if (editBtn) {
				var row = editBtn.closest('.row');
				enterEditMode(row);
				return;
			}
			var saveBtn = e.target.closest('[data-action="save"]');
			if (saveBtn) {
				saveEdit(saveBtn.closest('.row'));
				return;
			}
			var cancelBtn = e.target.closest('[data-action="cancel"]');
			if (cancelBtn) {
				exitEditMode(cancelBtn.closest('.row'), false);
				return;
			}
		});
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/static/app.js
git commit -m "feat: add inline edit functionality for links"
```

---

### Task 14: JavaScript — QR popover

**Files:**
- Modify: `internal/handlers/static/app.js`

- [ ] **Step 1: Add QR popover logic**

Edit `internal/handlers/static/app.js`. Inside the DOMContentLoaded callback, after the inline edit logic, add:

```javascript
		var qrPopover = null;
		var qrBackdrop = null;

		function closeQRPopover() {
			if (qrPopover) { qrPopover.remove(); qrPopover = null; }
			if (qrBackdrop) { qrBackdrop.remove(); qrBackdrop = null; }
		}

		function openQRPopover(row, btn) {
			closeQRPopover();
			var id = row.getAttribute('data-id');

			qrBackdrop = document.createElement('div');
			qrBackdrop.className = 'qr-backdrop';
			qrBackdrop.onclick = closeQRPopover;
			document.body.appendChild(qrBackdrop);

			qrPopover = document.createElement('div');
			qrPopover.className = 'qr-popover';
			qrPopover.textContent = 'loading...';

			var rect = btn.getBoundingClientRect();
			qrPopover.style.top = (rect.bottom + window.scrollY + 4) + 'px';
			qrPopover.style.left = (rect.left + window.scrollX) + 'px';
			document.body.appendChild(qrPopover);

			fetch('/api/links/' + id + '/qr')
				.then(function (res) { return res.text(); })
				.then(function (svg) {
					if (qrPopover) qrPopover.innerHTML = svg;
				})
				.catch(function () {
					if (qrPopover) qrPopover.textContent = 'failed to load QR';
				});
		}

		table.addEventListener('click', function (e) {
			var qrBtn = e.target.closest('[data-action="qr"]');
			if (qrBtn) {
				var row = qrBtn.closest('.row');
				openQRPopover(row, qrBtn);
				return;
			}
		});

		document.addEventListener('keydown', function (e) {
			if (e.key === 'Escape') closeQRPopover();
		});
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/static/app.js
git commit -m "feat: add QR code popover for links"
```

---

### Task 15: JavaScript — bulk actions + single delete

**Files:**
- Modify: `internal/handlers/static/app.js`

- [ ] **Step 1: Add bulk actions and single delete logic**

Edit `internal/handlers/static/app.js`. Inside the DOMContentLoaded callback, after the QR popover logic, add:

```javascript
		var selectionCount = document.getElementById('selection-count');
		var bulkDeleteBtn = document.getElementById('bulk-delete');
		var selectAll = document.getElementById('select-all');

		function updateSelection() {
			var checked = table.querySelectorAll('.row-checkbox:checked');
			var count = checked.length;
			if (selectionCount) selectionCount.textContent = count + ' selected';
			if (bulkDeleteBtn) bulkDeleteBtn.disabled = count === 0;
		}

		function getVisibleRowIds() {
			var ids = [];
			table.querySelectorAll('.row:not(.header)').forEach(function (row) {
				if (row.style.display !== 'none') {
					ids.push(parseInt(row.getAttribute('data-id'), 10));
				}
			});
			return ids;
		}

		if (selectAll) {
			selectAll.addEventListener('change', function () {
				var checked = selectAll.checked;
				table.querySelectorAll('.row:not(.header)').forEach(function (row) {
					if (row.style.display !== 'none') {
						row.classList.toggle('is-selected', checked);
						var cb = row.querySelector('.row-checkbox');
						if (cb) cb.checked = checked;
					}
				});
				updateSelection();
			});
		}

		table.addEventListener('change', function (e) {
			if (e.target.classList.contains('row-checkbox')) {
				var row = e.target.closest('.row');
				row.classList.toggle('is-selected', e.target.checked);
				updateSelection();
			}
		});

		if (bulkDeleteBtn) {
			bulkDeleteBtn.addEventListener('click', function () {
				var ids = [];
				table.querySelectorAll('.row-checkbox:checked').forEach(function (cb) {
					ids.push(parseInt(cb.closest('.row').getAttribute('data-id'), 10));
				});
				if (ids.length === 0) return;
				if (!confirm('delete ' + ids.length + ' links?')) return;

				fetch('/api/links/bulk', {
					method: 'DELETE',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ ids: ids })
				}).then(function (res) {
					if (res.ok) {
						ids.forEach(function (id) {
							var row = table.querySelector('.row[data-id="' + id + '"]');
							if (row) row.remove();
						});
						showToast('deleted ' + ids.length + ' links', 'success');
						updateSelection();
						if (selectAll) selectAll.checked = false;

						var remaining = table.querySelectorAll('.row:not(.header)').length;
						var footer = document.getElementById('table-footer');
						if (footer) footer.textContent = 'showing ' + remaining + ' of ' + remaining;
					} else {
						showToast('delete failed', 'error');
					}
				}).catch(function () {
					showToast('network error', 'error');
				});
			});
		}

		table.addEventListener('click', function (e) {
			var deleteBtn = e.target.closest('[data-action="delete"]');
			if (!deleteBtn) return;
			var row = deleteBtn.closest('.row');
			var id = row.getAttribute('data-id');
			var code = row.getAttribute('data-code');

			if (!confirm('delete /' + code + '?')) return;

			row.classList.add('is-deleting');
			fetch('/api/links/' + id, { method: 'DELETE' })
				.then(function (res) {
					if (res.ok) {
						row.remove();
						showToast('deleted /' + code, 'success');
						var remaining = table.querySelectorAll('.row:not(.header)').length;
						var footer = document.getElementById('table-footer');
						if (footer) footer.textContent = 'showing ' + remaining + ' of ' + remaining;
					} else {
						row.classList.remove('is-deleting');
						showToast('delete failed', 'error');
					}
				})
				.catch(function () {
					row.classList.remove('is-deleting');
					showToast('network error', 'error');
				});
		});

		document.addEventListener('keydown', function (e) {
			if (e.key === 'Escape') {
				table.querySelectorAll('.row-checkbox:checked').forEach(function (cb) {
					cb.checked = false;
					cb.closest('.row').classList.remove('is-selected');
				});
				if (selectAll) selectAll.checked = false;
				updateSelection();
			}
		});
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/static/app.js
git commit -m "feat: add bulk select/delete and single delete with toast feedback"
```

---

### Task 16: Restyle login, create, 404 templates

**Files:**
- Modify: `templates/login.html`
- Modify: `templates/create.html`
- Modify: `templates/404.html`

- [ ] **Step 1: Replace login.html**

Replace the entire contents of `templates/login.html`:

```html
{{define "title"}}Login - Link Shortener{{end}}
{{define "breadcrumb"}}~ /login{{end}}
{{define "nav"}}{{end}}

{{define "content"}}
<div class="card center-card">
	<h2>~ login</h2>

	{{if .Error}}
	<div class="error">{{.Error}}</div>
	{{end}}

	<form method="POST" action="/login">
		<input type="password" name="password" placeholder="password" required autofocus>
		<button type="submit" class="btn">enter &rarr;</button>
	</form>
	<div class="hint">first login sets the password</div>
</div>
{{end}}
```

- [ ] **Step 2: Replace create.html**

Replace the entire contents of `templates/create.html`:

```html
{{define "title"}}Create Link - Link Shortener{{end}}
{{define "breadcrumb"}}~ /create{{end}}
{{define "nav"}}
	<a href="/dashboard">dashboard</a>
	<a href="/create" class="active">create</a>
	<a href="/logout" onclick="event.preventDefault();document.forms['logout-form'].submit()">logout</a>
{{end}}

{{define "content"}}
<div class="card center-card">
	<h2>~ create link</h2>

	{{if .Error}}
	<div class="error">{{.Error}}</div>
	{{end}}

	<form method="POST" action="/create">
		<input type="url" name="url" placeholder="https://example.com/your-very-long-url-here" required>
		<input type="text" name="custom_code" id="custom_code" placeholder="custom code (optional)" oninput="document.getElementById('preview').textContent = this.value || 'code'">
		<div class="hint" style="text-align:left;margin-bottom:10px;">&rarr; s.duskoide.org/<em id="preview" style="color:var(--code-color);font-style:normal;">code</em></div>
		<button type="submit" class="btn">shorten &rarr;</button>
	</form>
</div>
{{end}}
```

- [ ] **Step 3: Replace 404.html**

Replace the entire contents of `templates/404.html`:

```html
{{define "title"}}Not Found - Link Shortener{{end}}
{{define "breadcrumb"}}~ /404{{end}}
{{define "nav"}}{{end}}

{{define "content"}}
<div class="notfound">
	<div class="notfound-code">404</div>
	<div class="notfound-msg">~ link not found</div>
	<a href="/" class="btn btn-outline">&larr; back to dashboard</a>
</div>
{{end}}
```

- [ ] **Step 4: Verify it runs**

Run: `go run ./cmd/linkshortener &`
Then: `curl -s http://localhost:8081/login | grep -o 'center-card'`
Expected: `center-card`

Kill: `kill %1`

- [ ] **Step 5: Commit**

```bash
git add templates/login.html templates/create.html templates/404.html
git commit -m "feat: restyle login, create, and 404 templates with dark theme"
```

---

### Task 17: Register NotFound handler + finalize routes

**Files:**
- Create: `internal/handlers/notfound.go`
- Modify: `cmd/linkshortener/main.go`

- [ ] **Step 1: Create NotFound handler**

Create `internal/handlers/notfound.go`:

```go
package handlers

import "net/http"

func NotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	renderTemplate(w, "404.html", nil)
}
```

- [ ] **Step 2: Register NotFound and verify all routes in main.go**

Edit `cmd/linkshortener/main.go`. The complete file should look like:

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

	if err := handlers.InitTemplates("templates"); err != nil {
		slog.Error("failed to load templates", "error", err)
		panic(err)
	}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Public routes
	r.Get("/login", handlers.LoginPage)
	r.Post("/login", handlers.LoginPost(database))
	r.Post("/logout", handlers.Logout(database))
	r.Handle("/static/*", handlers.StaticHandler())
	r.Get("/{shortCode}", handlers.Redirect(database))

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(database))
		r.Get("/", handlers.Dashboard(database, cfg.BaseURL))
		r.Get("/dashboard", handlers.Dashboard(database, cfg.BaseURL))
		r.Get("/create", handlers.CreatePage)
		r.Post("/create", handlers.CreatePost(database))
		r.Get("/api/links", handlers.APILinks(database))
		r.Delete("/api/links/bulk", handlers.APIBulkDeleteLink(database))
		r.Get("/api/links/{id}/qr", handlers.APIQRCode(database, cfg.BaseURL))
		r.Patch("/api/links/{id}", handlers.APIEditLink(database))
		r.Delete("/api/links/{id}", handlers.APIDeleteLink(database))
	})

	r.NotFound(handlers.NotFound)

	slog.Info("link shortener listening", "addr", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, r); err != nil {
		slog.Error("server failed", "error", err)
	}
}
```

Note the route ordering: `/api/links/bulk` is registered **before** `/api/links/{id}` to ensure the literal `bulk` segment takes priority over the `{id}` parameter in Chi's radix tree.

- [ ] **Step 3: Verify compilation**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/notfound.go cmd/linkshortener/main.go
git commit -m "feat: add NotFound handler and finalize all routes"
```

---

### Task 18: Final verification

**Files:** None (verification only)

- [ ] **Step 1: Build the binary**

Run: `go build -o linkshortener ./cmd/linkshortener`
Expected: binary created, no errors.

- [ ] **Step 2: Start the server locally**

Run: `./linkshortener &`
Expected: log message `link shortener listening addr=0.0.0.0:8081`

- [ ] **Step 3: Test all pages render**

```bash
curl -s http://localhost:8081/login | grep -o 'center-card'        # login page
curl -s http://localhost:8081/login | grep -o 'app.css'            # CSS linked
curl -s -X POST -d 'password=test123' http://localhost:8081/login -c /tmp/cookies.txt -L -o /dev/null -w '%{http_code}'  # login (first login sets password)
curl -s -b /tmp/cookies.txt http://localhost:8081/dashboard | grep -o 'table-wrap'  # dashboard
curl -s -b /tmp/cookies.txt http://localhost:8081/create | grep -o 'center-card'    # create page
curl -s http://localhost:8081/nonexistent-page | grep -o 'notfound-code'            # 404
```

Expected: each grep outputs the matched class name.

- [ ] **Step 4: Test API endpoints**

```bash
# Create a link
curl -s -b /tmp/cookies.txt -X POST -d 'url=https://example.com&custom_code=test1' http://localhost:8081/create -o /dev/null -w '%{http_code}\n'
# Expected: 200 (redirects to dashboard, curl follows)

# List links
curl -s -b /tmp/cookies.txt http://localhost:8081/api/links | head -50

# Get QR code (replace 1 with actual ID from the list above)
curl -s -b /tmp/cookies.txt http://localhost:8081/api/links/1/qr -o /dev/null -w '%{content_type}\n'
# Expected: image/svg+xml

# Inline edit
curl -s -b /tmp/cookies.txt -X PATCH -H 'Content-Type: application/json' -d '{"short_code":"test2","original_url":"https://example.org"}' http://localhost:8081/api/links/1 -w '\n%{http_code}\n'
# Expected: {"ok":true} 200

# Bulk delete (create a few links first, then delete by IDs)
curl -s -b /tmp/cookies.txt -X DELETE -H 'Content-Type: application/json' -d '{"ids":[1]}' http://localhost:8081/api/links/bulk -w '\n%{http_code}\n'
# Expected: {"deleted":1} 200
```

- [ ] **Step 5: Manual browser test**

Open `http://localhost:8081/login` in a browser. Log in (first login sets password). Verify:
- [ ] Dark theme applied to all pages (login, dashboard, create)
- [ ] Desktop table: columns aligned (code, url, clicks, created, actions)
- [ ] Copy button (⨁) copies full URL, toast appears
- [ ] QR button (▦) opens SVG popover, closes on outside-click/Esc
- [ ] Edit button (✎) turns cells into inputs; save updates; esc cancels; duplicate code shows error toast
- [ ] Filter input narrows rows live; footer count updates; no-match state shows
- [ ] Checkbox selection highlights row; "delete selected" enables/disables; confirm dialog works
- [ ] Single delete (✕) shows confirm, removes row, toast appears
- [ ] Resize to ≤640px: card stack layout, hamburger on left opens nav sheet
- [ ] Visit nonexistent URL: 404 page with dark theme

- [ ] **Step 6: Kill the local server**

Run: `kill %1`

- [ ] **Step 7: Homeserver deployment (via SSH)**

```bash
ssh pn@homeserver
```

On the homeserver:
1. Pull the latest code: `cd /path/to/linkshortener && git pull`
2. Add `BASE_URL=https://s.duskoide.org` to the `.env` file
3. Rebuild the container: `podman-compose build`
4. Restart the service: `sudo systemctl restart linkshortener.service`
5. Smoke-test through the tunnel: open `https://s.duskoide.org/dashboard` in a browser

- [ ] **Step 8: Final commit (if any cleanup needed)**

```bash
git status
# If clean, done. If any stray changes, commit them.
```

---

## Self-Review

**Spec coverage check:**
- Dark-terminal theme across all pages: Tasks 8, 9, 16
- Copy to clipboard: Task 11
- Search/filter: Task 12
- Inline edit (code + URL): Tasks 3 (DB), 5 (handler), 13 (JS)
- QR code per link: Tasks 2 (dep), 3 (GetLinkByID), 4 (handler), 14 (JS)
- Bulk delete: Tasks 3 (DB), 6 (handler), 15 (JS)
- Toast notifications: Task 11
- Responsive (mobile card stack, left hamburger, left nav sheet): Task 8 (CSS)
- Column alignment (fixed grid): Task 8 (CSS grid-template-columns), Task 10 (HTML structure)
- Empty states: Task 10 (HTML), Task 12 (JS no-match)
- No full-page reloads: Tasks 11-15 (all fetch-based)
- BASE_URL config: Task 1
- Static embedding: Task 7
- All new API endpoints: Tasks 4, 5, 6, 17 (route registration)
- NotFound handler: Task 17
- Verification (local + homeserver): Task 18

**Gaps:** None found. All spec sections covered.

**Placeholder scan:** No TBD/TODO found in the plan. All code steps have complete code.

**Type consistency:** `Dashboard(database, baseURL)` signature matches across Tasks 10 and 17. `APIQRCode(database, baseURL)` matches across Tasks 4 and 17. `APIEditLink(database)` matches across Tasks 5 and 17. `APIBulkDeleteLink(database)` matches across Tasks 6 and 17. `db.UpdateLink(db, linkID, shortCode, originalURL)` matches across Tasks 3 and 5. `db.DeleteLinks(db, ids)` matches across Tasks 3 and 6. `db.GetLinkByID(db, id)` matches across Tasks 3 and 4.

**Note on Task 4 code:** The plan flagged a typo (`str strconv.Itoa`) in the QR handler code and provided the correction. The executing engineer should use the corrected version. Also, the `"errors"` import in Task 5 should be excluded.
