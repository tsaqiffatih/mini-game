# Deployment Plan

## Goal

Deploy `mini-game` as a production-quality portfolio project with a clear single-node architecture first, while documenting the path to scalable realtime multiplayer.

## Recommended Initial Production Architecture

```text
Browser
  -> Frontend hosting
  -> HTTPS API/WebSocket domain
  -> Nginx or Caddy reverse proxy
  -> Go backend Docker container
  -> Stockfish binary inside container
  -> stdout JSON logs
```

This is enough for a strong portfolio deployment if the limitations are documented:

- Rooms are in memory.
- Server restart clears active rooms.
- Horizontal scaling is not yet supported.
- AI rooms consume backend CPU.

## Frontend Deployment

Recommended options:

- Vercel for easiest Next.js deployment.
- Static/container deployment if you want everything on one VPS.

Required environment variables:

```text
NEXT_PUBLIC_HTTP_BACKEND_URL=https://api.example.com
NEXT_PUBLIC_WS_BACKEND_URL=wss://api.example.com
```

If frontend and backend share one domain:

```text
NEXT_PUBLIC_HTTP_BACKEND_URL=https://example.com/api
NEXT_PUBLIC_WS_BACKEND_URL=wss://example.com/api
```

Current frontend uses public env vars directly in:

- `src/components/Lobby.tsx`
- `src/components/RegisterUser.tsx`
- `src/utils/gameWebsocket.ts`

Deployment checklist:

- Run `npm run build`.
- Verify `/`, `/chess`, and `/tictactoe`.
- Verify websocket URL uses `wss://` in production.
- Verify sound and chess piece assets load with correct casing.

## Backend Deployment

The backend already has:

- `server/Dockerfile`
- `server/docker-compose.yml`
- `server/Caddyfile`

Backend env vars:

```text
PORT=8080
ALLOWED_ORIGINS=https://example.com,https://www.example.com
STOCKFISH_PATH=/app/stockfish/stockfish
```

`STOCKFISH_PATH` is optional if the default path works from the backend working directory. In the provided container, the binary is copied to `/app/stockfish/stockfish`, which matches the default relative path when `WORKDIR /app`.

Recommended VPS size:

- Minimum demo: 1 vCPU, 1 GB RAM.
- Safer chess AI demo: 2 vCPU, 2 GB RAM.
- If many AI rooms are expected: 2-4 vCPU and CPU limits per container.

## WebSocket Deployment

Reverse proxy must support websocket upgrade.

Nginx example:

```nginx
server {
    server_name api.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 75s;
    }
}
```

Caddy example:

```caddyfile
api.example.com {
  reverse_proxy backend:8080
}
```

The current `Caddyfile` is minimal and suitable for local Docker smoke testing, not final public domain configuration.

## Docker Strategy

Current Dockerfile builds a static Go binary and copies Stockfish into a distroless image. This is a good production direction.

Recommended improvements:

- Add `.dockerignore` coverage for temp files, logs, test binaries, local env files.
- Confirm Stockfish binary has execute permission in image.
- Add healthcheck endpoint in backend, then Docker healthcheck.
- Keep CPU/memory limits in compose for portfolio stability.

Suggested production compose shape:

```yaml
services:
  backend:
    image: mini-game-backend:latest
    restart: unless-stopped
    environment:
      PORT: "8080"
      ALLOWED_ORIGINS: "https://example.com"
    expose:
      - "8080"

  caddy:
    image: caddy:2.8-alpine
    restart: unless-stopped
    depends_on:
      - backend
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
```

## SSL and Domain Setup

Recommended:

- `example.com` for frontend.
- `api.example.com` for backend HTTP and websocket.
- Use Caddy for automatic TLS or Nginx with Certbot.

DNS:

- `A example.com -> frontend host` if self-hosting.
- `CNAME api.example.com -> VPS` or `A api.example.com -> VPS IP`.

Frontend env must match final backend domain exactly.

## Environment Variable Strategy

Do not commit real `.env` files.

Create documented examples:

```text
mini-game-client/.env.example
server/.env.example
```

Server:

- `PORT`
- `ALLOWED_ORIGINS`
- `STOCKFISH_PATH`

Frontend:

- `NEXT_PUBLIC_HTTP_BACKEND_URL`
- `NEXT_PUBLIC_WS_BACKEND_URL`

Operational:

- Keep production secrets/env in VPS compose override, systemd environment, or hosting dashboard.

## Monitoring and Logging

Current backend logs JSON through `slog`.

Add:

- Request count by path/status.
- WebSocket connect/disconnect counts.
- Active rooms.
- Active clients.
- AI move duration and failures.
- Room deletion reasons.
- 429 rate-limit events.

Initial lightweight stack:

- Docker logs + log rotation.
- Uptime Kuma for HTTP health.
- VPS metrics from provider.

Future:

- Prometheus metrics endpoint.
- Grafana dashboard.
- Loki log aggregation.

## Scaling Considerations

Current hard limits:

- `MemoryRoomRepository` is process-local.
- `PlayerManager` is process-local.
- `ClientRegistry` is process-local.
- Chat history is process-local.
- Stockfish process is per AI chess room.
- No distributed room ownership.

This means horizontal scaling is unsafe unless all websocket and HTTP requests for a room land on the same process and room state is not lost.

Short-term scale:

- One backend instance.
- CPU/memory limits.
- Room cleanup.
- Conservative AI difficulty defaults.

Mid-term scale:

- Sticky sessions by room ID.
- Redis for presence and pub/sub.
- Postgres for durable users/game summaries.

Long-term scale:

- Room actor service.
- WebSocket gateway service.
- Distributed event bus.
- AI worker pool.

## Future Matchmaking Scaling

Current room model is invite-code based.

For matchmaking:

- Add queue by game type.
- Match players by rating/preference.
- Allocate/create room.
- Notify both players.
- Use room ownership metadata.

Storage:

- Redis sorted sets/lists for queues.
- Postgres for user profile/rating.

## CI/CD Suggestions

Backend pipeline:

- `go test ./...`
- `go build ./...`
- Docker build.
- Optional container smoke test.

Frontend pipeline:

- `npm ci`
- `npm run build`
- lint once script is aligned with Next 15/ESLint setup.

Deployment:

- GitHub Actions build backend image.
- Push to registry.
- SSH deploy compose pull/up on VPS.
- Vercel auto-deploy frontend from main branch.

## Recommended Folder Cleanup Before Production

Root:

- Keep `archive/old-server` clearly marked as legacy or move outside production repo if not needed.
- Ensure `/docs` is committed.
- Add root README explaining active paths.

Backend:

- Exclude `tmp/`, build logs, local binaries, `.env`.
- Keep `API_CONTRACT.md` updated or replace with generated docs.
- Review `note.md` and old architecture drafts for accuracy.

Frontend:

- Exclude `.next/`, local `.env`, `.codex`.
- Remove unused example route if not needed.
- Remove dead/deleted component references.

## Production Readiness Checklist

- Backend starts from clean container.
- Stockfish works inside container.
- Frontend connects over HTTPS/WSS.
- CORS allows only frontend domain.
- WebSocket origin policy is restricted.
- Room create/join works.
- Chess multiplayer works.
- Chess AI works.
- Refresh/reconnect behavior is acceptable.
- Logs show room lifecycle and websocket lifecycle.
- README documents known scaling limits.

