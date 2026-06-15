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

#### Cloudflare Tunnel

To expose the service publicly via Cloudflare Tunnel:

1. Install `cloudflared` and authenticate:
   ```bash
   cloudflared tunnel login
   ```

2. Create a tunnel:
   ```bash
   cloudflared tunnel create linkshortener
   ```

3. Add a DNS record pointing to the tunnel:
   ```bash
   cloudflared tunnel route dns linkshortener your-domain.com
   ```

4. Configure the tunnel with a `config.yml`:
   ```yaml
   tunnel: <TUNNEL_ID>
   credentials-file: /root/.cloudflared/<TUNNEL_ID>.json

   ingress:
     - hostname: your-domain.com
       service: http://localhost:8081
     - service: http_status:404
   ```

5. Run the tunnel:
   ```bash
   cloudflared tunnel run linkshortener
   ```

For persistent setup, create a systemd service for `cloudflared`.

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

On first use, enter your desired password on the login page — it will be stored for future logins. This password is used to access the dashboard and create links.

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
