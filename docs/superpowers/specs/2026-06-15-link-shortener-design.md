# Link Shortener Design Spec

**Date:** 2026-06-15
**Purpose:** Personal link shortener for small team/family use on AlmaLinux homeserver

## Tech Stack

- **Backend:** Rust with Axum web framework
- **Database:** SQLite with SQLx
- **Templates:** Tera (server-side rendering)
- **Deployment:** Podman container on AlmaLinux
- **Authentication:** Simple shared password

## Architecture Overview

Single binary that embeds HTML templates, uses SQLite for storage, serves both API and web interface. Runs in Podman container on AlmaLinux with simple password authentication.

## Components

1. **HTTP Server** (Axum)
   - Handles all incoming requests
   - Routes to appropriate handlers
   - Manages sessions/cookies

2. **Template Engine** (Tera)
   - Renders HTML pages
   - Dashboard, create link, manage links
   - Login page

3. **Database Layer** (SQLx)
   - SQLite connection pool
   - CRUD operations for links
   - Link lookup for redirects

4. **Authentication**
   - Simple password verification
   - Session management via cookies
   - Password stored as bcrypt hash

5. **Redirect Handler**
   - Looks up short code
   - Returns 302 redirect to original URL
   - Increments click counter

## Data Flow

### Creating a Short Link
1. User visits `/create` (authenticated)
2. Enters original URL and optional custom code
3. Server generates or uses custom code
4. Saves to database with timestamp
5. Returns success with short link

### Using a Short Link
1. User visits `/{short_code}`
2. Server looks up code in database
3. If found: increments click counter, returns 302 redirect
4. If not found: returns 404 page

### Dashboard
1. User visits `/dashboard` (authenticated)
2. Server fetches all links from database
3. Renders template with link list, click counts
4. Shows create/edit/delete options

### Authentication
1. User visits `/login`
2. Enters password
3. Server verifies against stored bcrypt hash
4. Sets session cookie
5. Redirects to dashboard

## Database Schema

### links table
```sql
CREATE TABLE links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    short_code TEXT UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    click_count INTEGER DEFAULT 0
);
```

### settings table
```sql
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

Settings stores:
- `password_hash`: bcrypt hash of the shared password
- `created_at`: when the system was first set up

## Authentication Design

### Setup
- On first run, prompts for password (or uses environment variable)
- Stores bcrypt hash in settings table
- No user accounts - one shared password

### Login Flow
1. User visits any page
2. If not authenticated, redirects to `/login`
3. User enters password
4. Server verifies against stored hash
5. Sets secure session cookie (httpOnly, sameSite)
6. Redirects to requested page

### Session Management
- Cookie-based sessions
- Session expires after 24 hours
- No logout endpoint (clear cookie manually or wait)

### Security
- Passwords hashed with bcrypt (cost factor 12)
- Session cookies are httpOnly and secure
- Rate limiting on login attempts (optional, can add later)

## Podman Deployment

### Containerfile
- Multi-stage build for small image
- Rust builder stage
- Alpine runtime stage
- Non-root user for security

### Container Configuration
- Single container
- Volume mount for SQLite database
- Port mapping (8081:8081)
- Environment variables for password

### Cloudflare Tunnel Integration
- App runs on localhost:8080 inside container
- Cloudflare Tunnel routes traffic from s.duskoide.org to container
- No exposed ports needed (tunnel handles external access)
- Uses cloudflared container or host-based cloudflared

### Data Persistence
- SQLite database stored in mounted volume
- Survives container restarts
- Easy backup (copy single file)

### Systemd Integration
- Podman container runs as systemd service
- Auto-restart on failure
- Logs to journald

## Error Handling

### HTTP Errors
- 404: Custom "link not found" page
- 401: Redirect to login page
- 500: Generic error page with request ID
- 400: Validation errors (invalid URL, etc.)

### Database Errors
- Connection failures: Retry with backoff
- Constraint violations: User-friendly messages
- Corruption: Log error, suggest backup restore

### Validation
- URL validation (must be valid HTTP/HTTPS)
- Short code validation (alphanumeric, 3-20 chars)
- Duplicate short code handling

### Logging
- Structured logging (tracing crate)
- Request/response logging
- Error logging with context

### Graceful Degradation
- If database is down, show maintenance page
- If template fails, return JSON error
- Always return meaningful error messages

## API Endpoints

- `GET /` - Dashboard (authenticated)
- `GET /login` - Login page
- `POST /login` - Authenticate
- `GET /create` - Create link form
- `POST /create` - Create new link
- `GET /:short_code` - Redirect to original URL
- `GET /api/links` - List all links (JSON)
- `DELETE /api/links/:id` - Delete a link

## Pages

- Login page
- Dashboard with link list
- Create link form
- 404 page
- Error pages

## Configuration

### Environment Variables
- `DATABASE_URL`: SQLite database path (default: `/data/links.db`)
- `PASSWORD_HASH`: bcrypt hash (alternative to interactive setup)
- `LISTEN_ADDR`: Listen address (default: `0.0.0.0:8081`)
- `BASE_URL`: Public URL for generating short links (default: `http://localhost:8080`)
- `SESSION_SECRET`: Secret for session cookies

### Cloudflare Tunnel Configuration
- Domain: `s.duskoide.org`
- Tunnel routes traffic from `https://s.duskoide.org` to `http://localhost:8081`
- TLS termination handled by Cloudflare
- No SSL certificates needed in container

### First Run
1. Podman container starts
2. Checks if database exists
3. If not, creates schema
4. If no password hash, prompts for password (or uses env var)
5. Starts server

## Future Enhancements (Out of Scope)

- User accounts and permissions
- Link expiration
- QR code generation
- Tags and categories
- API rate limiting
- Link preview/metadata extraction
- Bulk import/export
