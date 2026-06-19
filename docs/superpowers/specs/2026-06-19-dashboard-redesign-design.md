# Dashboard UI Redesign — Design Spec

**Date:** 2026-06-19
**Status:** Approved (brainstorm phase)
**Scope:** Full redesign of the link shortener dashboard and all shared pages (login, create, 404) with a dark-terminal theme, plus new interactive features.

## Context

The link shortener is a Go single-binary app (Chi + SQLite + html/template) deployed on a homeserver via podman-compose. The current dashboard (`templates/dashboard.html`, `internal/handlers/dashboard.go:10`) renders a plain table with Short Code, Original URL, Clicks, Created, and a Delete button. Styling is minimal inline CSS in `templates/base.html`. No search, no copy-to-clipboard, no edit, no QR, no bulk actions, no feedback on actions (delete does a full page reload).

This spec defines a full visual redesign and adds the highest-impact missing features while preserving the project's dependency-light, pure-Go, no-build-step ethos.

## Goals

1. Apply a cohesive dark-terminal visual theme across all pages (dashboard, login, create, 404).
2. Add: copy short URL to clipboard, search/filter, inline edit (code + URL), QR code per link, bulk delete, toast notifications.
3. Preserve the no-framework, no-build-step, pure-Go architecture. Only one new Go dependency (QR generation).
4. Work responsively from desktop to mobile.

## Non-Goals

- SPA rewrite, build tooling, npm/node dependencies.
- Pagination/virtualization (deferred — homeserver scale doesn't currently require it).
- Stats detail view / click-over-time analytics (deferred — would need new DB columns).
- Authentication or authorization changes.
- Database schema changes beyond what inline edit requires (an UPDATE on existing columns).

## Approach

**Approach A — Vanilla JS + embedded CSS** (chosen over HTMX/Alpine and SPA alternatives).

- Static assets (CSS, JS) live in a `static/` directory, embedded into the binary via `//go:embed`, served by Chi.
- Plain vanilla JS (no framework) handles fetch, clipboard, toasts, filter, bulk-select, inline edit.
- QR codes generated server-side by `github.com/yeqown/go-qrcode` (pure Go, no CGO), served as SVG at `GET /api/links/{id}/qr`.
- Initial page render stays server-side (html/template); JS enhances from there. No full-page reloads for mutations.

## Section 1 — Visual System

### Theme: Dark / Terminal

A dev-tool aesthetic fitting the homeserver audience. Single monospace font stack across the entire UI.

### Design Tokens

| Token | Value | Usage |
|---|---|---|
| bg base | `#0f172a` (slate-900) | page background |
| bg elevated | `#1e293b` (slate-800) | cards, table rows, nav sheet |
| bg topbar | `#020617` (slate-950) | top header bar |
| border | `#334155` (slate-700) | all borders, never heavier than 1px |
| text primary | `#e2e8f0` (slate-200) | body text |
| text muted | `#64748b` (slate-500) | secondary text, labels |
| accent | `#22d3ee` (cyan-400) | primary action, focus, active nav |
| short code | `#34d399` (emerald-400) | short code values |
| clicks | `#fbbf24` (amber-400) | click counts |
| danger | `#f87171` (red-400) | destructive actions, delete borders |
| success bg | `#022c22` (green-950) | success toast background |
| font | `ui-monospace, SFMono-Regular, Menlo, Consolas, monospace` | entire UI |
| radius | 4–6px | cards, inputs, buttons |
| row padding | 8–10px | table rows for density without crowding |

### Layout (Desktop)

- Sticky topbar: `⌁ link-shortener` wordmark left, nav links right (dashboard / create / logout).
- Max content width raised from 800px to **1100px**.
- Toolbar row pinned below topbar: `$ filter` input (left), selection count + "delete selected" + "+ new" (right).
- Table fills remaining width.

### Critical: Column Alignment

The table uses a **fixed grid template** applied identically to the header row and every body row (including edit-mode rows):

```
grid-template-columns: 24px 70px 1fr 56px 64px 150px;
                       chk  code  url  clk  created  actions
```

- The `1fr` (URL) column absorbs width changes; all other columns are fixed-width.
- Edit-mode inputs use `width: 100%` within their grid cell — they do not set their own width.
- This guarantees clicks / created / actions stay aligned across default, selected, and editing rows.

### Shared Across All Pages

`base.html` carries the topbar + token set. Login, create, 404 render different content cards but inherit the same theme. Login is the only page that hides nav links (user is not yet authed). 404 uses a minimal topbar (wordmark only, no nav).

## Section 2 — Interaction States & JS Behavior

### Row States (CSS classes on the row element)

- **default**: checkboxes empty, action icons always visible (not hover-only — better for touch).
- **selected** (`.is-selected`): cyan-tinted background `rgba(34,211,238,0.04)`, checkbox fills cyan. Selection count in toolbar updates live.
- **editing** (`.is-editing`): code + destination cells swap to `<input>` elements (width:100%), actions cell swaps to `save` / `esc` buttons. Only one row editable at a time; entering edit on another row exits the current one and discards unsaved changes.
- **deleting** (`.is-deleting`): row dims to 50% opacity, its actions disabled while DELETE fetch is in flight; row element removed on 200 OK.

### Copy to Clipboard

Click ⧉ → `navigator.clipboard.writeText(fullShortURL)` → success toast `copied s.duskoide.org/go` (green). On failure (clipboard API unavailable, e.g. non-HTTPS local context) → fallback to hidden `<input>` + `document.execCommand('copy')` + same toast. Toast auto-dismisses after 2.5s.

The full short URL is constructed client-side from a base URL injected into the page (e.g. `data-base-url` attribute on the table) + the short code.

### QR Popover

Click ▦ → `GET /api/links/{id}/qr` (server returns `image/svg+xml`) → inject SVG into a small popover anchored below the action cell. Click anywhere outside (or Esc) closes it. Only one popover open at a time. The short URL is rendered as a `<text>` element inside the SVG itself (no separate label needed).

### Inline Edit

Click ✎ → row enters `.is-editing`. `save` → `PATCH /api/links/{id}` with JSON body `{short_code, original_url}`:
- 200: row returns to default state with updated values.
- 409 (short code collision): error toast `code already taken`, stay in edit mode.
- 400 (validation): error toast with the validation message.

`esc` or clicking outside the row → discard, return to default. Enter key in either input triggers save.

### Filter

`$` input — on every `input` event, client-side filter rows by case-insensitive substring match against code + destination. No debounce needed at homeserver scale. Footer count updates (`showing X of Y`). If filter matches nothing → show a muted `no matches for "query"` row. This is distinct from the empty state (no links exist at all).

### Bulk Actions

- Checkbox per row + "select all" checkbox in the header.
- "delete selected" button disabled (greyed) when count = 0.
- Click "delete selected" → confirm dialog (`delete N links?`) → `DELETE /api/links/bulk` with JSON body `{ids: [...]}` (new endpoint) → remove all matched rows, success toast `deleted N links`, clear selection.
- Esc clears all selection.
- Select-all checkbox selects/deselects all currently-visible (post-filter) rows.

### Toasts

- Single fixed container, bottom-right.
- Queue: max 3 visible; oldest dismissed when a 4th arrives.
- Types: `success` (green bg `#022c22`, emerald text/border), `error` (red), `info` (cyan).
- Auto-dismiss after 2.5s; hover pauses dismiss; click X dismisses immediately.

### Empty States

- **No links at all**: centered card — `no links yet` + a cyan `create your first link` button.
- **Filter no match**: muted `no matches for "query"` row inside the table.

### No Full-Page Reloads

All mutations (copy, delete, edit save, bulk delete) use fetch + DOM updates. Initial page load is server-rendered HTML; JS enhances from there.

## Section 3 — Responsive & Other Pages

### Breakpoints

- `≥1024px`: table view, max-width 1100px, all columns visible.
- `641–1023px`: table view, max-width 100%, action icons always visible.
- `≤640px`: card stack — each link becomes a vertical card.

### Mobile Card Layout (≤640px)

Each link renders as a card (not a table row):
- Top line: checkbox + `/code` (left), clicks (right).
- Middle: destination URL, ellipsized.
- Bottom line: created date (left), action icons (right).
- Selected card: cyan border + tint instead of row background.

### Mobile Topbar

Order: `≡ · ⌁ wordmark · ~ breadcrumb (right-aligned)`.

Hamburger is on the **left** of the wordmark. Tapping it slides a **left-anchored nav sheet** out (~58% width) with nav links, dimmed backdrop behind it. Active page gets the cyan left-border accent. Closes on: tapping backdrop, tapping hamburger again, or selecting a link.

### Breakpoint Implementation

Pure CSS via a single `@media (max-width: 640px)` query that toggles `display` between the table grid and the card stack. Server renders one HTML structure; CSS swaps presentation. No JS needed for the swap.

### Nav Sheet Toggle

CSS-only approach using a hidden checkbox + `:checked` selector to keep JS minimal. The hamburger is a `<label>` for the hidden checkbox.

### Other Pages (Shared Theme)

- **Login**: centered card, `~ login` heading, password input, cyan `enter →` button, muted hint "first login sets the password". No nav links (unauthed).
- **Create**: `~ create link` heading, URL input, custom code input, live preview `→ s.duskoide.org/code` (emerald code), cyan `shorten →` button.
- **404**: big cyan `404`, `~ link not found`, muted explanation, cyan-outline `← back to dashboard` link. Minimal topbar (wordmark only).

## Section 4 — Backend Changes

### New API Endpoints

| Method | Path | Body | Response | Purpose |
|---|---|---|---|---|
| `GET` | `/api/links/{id}/qr` | — | `image/svg+xml` | Server-rendered QR code for the link's short URL |
| `PATCH` | `/api/links/{id}` | `{short_code, original_url}` | `200` / `409` / `400` | Inline edit |
| `DELETE` | `/api/links/bulk` | `{ids: [int64,...]}` | `200` | Bulk delete |

All endpoints are auth-protected (inside the existing `r.Group` with `auth.Middleware`).

### QR Generation

- Dependency: `github.com/yeqown/go-qrcode` (pure Go, no CGO — matches the project's `modernc.org/sqlite` pure-Go ethos).
- Handler builds the full short URL from a configurable base (env var `BASE_URL`, e.g. `https://s.duskoide.org`) + the link's short code.
- Generates an SVG with the QR encoding the full URL, plus a `<text>` element beneath rendering the short URL string.
- Content-Type: `image/svg+xml`.

### Inline Edit — DB & Validation

- New `db.UpdateLink(db, linkID, shortCode, originalURL)` function: `UPDATE links SET short_code = ?, original_url = ? WHERE id = ?`.
- Validation in the handler:
  - `short_code` non-empty, matches `^[a-zA-Z0-9_-]+$`, length 1–64.
  - `original_url` non-empty, valid URL (parseable by `net/url`), scheme http/https.
  - Short code uniqueness check before update; return `409` on collision.
- No DB schema migration needed — updates existing columns.

### Bulk Delete

- New `db.DeleteLinks(db, ids []int64)` function: `DELETE FROM links WHERE id IN (?, ?, ...)`.
- Handler parses JSON `{ids: [...]}`, validates each is a valid int64, caps at a reasonable max (e.g. 1000) to avoid query bloat.
- Returns `200` on success.

### BASE_URL Configuration

- New env var `BASE_URL` (e.g. `https://s.duskoide.org`), loaded in `internal/config/config.go`.
- Used by: QR handler (server-side), and injected into the dashboard page as a `data-base-url` attribute for client-side copy-to-clipboard URL construction.
- Falls back to deriving from the request host if unset (best-effort).

### Static Asset Embedding

- New `static/` directory at project root containing `app.css` and `app.js` (the JS may be split into modules: `clipboard.js`, `toasts.js`, `filter.js`, `bulk.js`, `inline-edit.js`, `qr.js` — or kept as a single `app.js`; implementation plan will decide).
- `//go:embed static/*` in a new `internal/handlers/static.go` (or in `main.go`).
- Chi serves `/static/*` via `http.FileServer(http.FS(...))` or `http.StripPrefix`.
- Templates reference `/static/app.css` and `/static/app.js` instead of inline `<style>`/`<script>`.

### Template Changes

- `base.html`: replace inline `<style>` with `<link rel="stylesheet" href="/static/app.css">` + `<script src="/static/app.js" defer></script>`. Apply dark-terminal tokens. Topbar markup for desktop + mobile (hamburger + hidden checkbox for nav sheet).
- `dashboard.html`: new table structure with fixed grid columns, checkbox column, action icons, edit-mode markup, empty states, toolbar with filter + bulk actions.
- `login.html`, `create.html`, `404.html`: restyle to match dark theme (content cards, monospace, cyan accents).
- `templates.go`: register `data-base-url` injection — the dashboard handler passes `BaseURL` into the template data map.

## Section 5 — Verification & Deployment

### Local Verification

```bash
go build ./...                       # verify compilation
go run ./cmd/linkshortener           # local dev on :8081
```

Manual checks against `http://localhost:8081`:
- All four pages render with dark theme, topbar correct per page.
- Desktop table: columns aligned across default/selected/editing rows.
- Copy button copies full short URL; toast appears.
- QR popover opens, shows SVG, closes on outside-click/Esc.
- Inline edit: save updates row, 409 on duplicate code, esc discards.
- Filter narrows rows live; footer count updates; no-match state shows.
- Bulk select: checkbox toggles, select-all works, delete N with confirm.
- Mobile (resize to ≤640px): card stack, left hamburger, left nav sheet.

### Homeserver Deployment

The app runs on the homeserver via podman-compose + systemd (`linkshortener.service`). Deployment/verification steps happen over SSH:

```bash
ssh pn@homeserver   # to build/deploy/verify on the server
```

- `BASE_URL` must be added to the homeserver's `.env` (e.g. `BASE_URL=https://s.duskoide.org`).
- Rebuild the container image, restart the service, smoke-test the dashboard through the Cloudflare Tunnel.

### No Tests / No CI

Per AGENTS.md, the project has no tests and no CI. Verification is manual (local + homeserver smoke test). If the user requests tests later, the `writing-plans` phase can include a testing strategy.

## Open Questions

None. All design decisions were resolved during brainstorming.

## References

- Current dashboard: `templates/dashboard.html`, `internal/handlers/dashboard.go:10`
- Current layout: `templates/base.html`
- Data model: `internal/db/links.go:8` (`Link` struct)
- Routes: `cmd/linkshortener/main.go:42` (protected group)
- Existing API: `internal/handlers/api.go` (`APILinks`, `APIDeleteLink`)
- Config loading: `internal/config/config.go`
