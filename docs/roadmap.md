# Roadmap

## Project Vision

`mini-game` should become a polished realtime multiplayer game platform that demonstrates full-stack architecture, websocket synchronization, authoritative game servers, AI game modes, and production deployment discipline.

The near-term product identity is:

- Browser-based games with room codes.
- Realtime two-player multiplayer.
- AI-supported solo play.
- Chess as the flagship game.
- Clear engineering documentation and production-ready deployment.

## Current State

Frontend:

- Next.js App Router client.
- Home registration flow stores `playerId` in `localStorage`.
- Lobby creates, joins, and creates AI rooms through HTTP.
- Shared websocket hook connects to `/ws`.
- Chess uses Chessground for board interaction and `chess.js` for local validation.
- Chess receives authoritative backend snapshots and updates FEN, move history, captures, sounds, room state, and chat.
- TicTacToe uses simpler React board state from snapshots.

Backend:

- Go websocket/HTTP server.
- In-memory `PlayerManager`, `MemoryRoomRepository`, and `ClientRegistry`.
- `Room` aggregate owns room lifecycle, player membership, game state, reset scheduling, chat, AI scheduling, and snapshots.
- Chess rules are owned by `chess.ChessGameState` using `notnil/chess`.
- Chess AI uses one Stockfish process per AI room.
- Room cleanup and inactive player cleanup run periodically.
- Dockerfile, compose, Caddyfile, tests, and structured logging exist.

## Phase 1: Stabilize Current Gameplay

Goal: make current rooms reliable enough for portfolio demos.

Priorities:

- Document and standardize websocket event contracts.
- Fix frontend websocket error handling so refresh/reconnect does not aggressively clear room state.
- Surface reconnect states in UI.
- Avoid immediate scary `player_left` UX on transient refresh.
- Finish AI thinking UI using existing snapshot fields.
- Add chess result banner driven by `game.chess.status`, `winner`, and `result`.
- Add a real promotion UI instead of auto-queen.
- Remove stray console logs from gameplay paths.
- Confirm sound file names/casing are stable across platforms.

Technical debt:

- Centralize frontend websocket message types.
- Replace `any` in chess update helpers with typed room snapshot DTOs.
- Keep `game.chess` as canonical and plan deprecation of top-level `chess`.
- Add frontend handling for `CHESS_UNDO_REQUEST` if undo is exposed.

Deployment milestone:

- Single backend container on VPS.
- Frontend deployed to Vercel or static Next hosting.
- Caddy/Nginx reverse proxy with SSL.
- Environment variables documented.

## Phase 2: Chess Polish and AI UX

Goal: make chess feel intentional rather than prototype-like.

Features:

- Checkmate visual highlight.
- AI thinking indicator.
- AI avatar and identity.
- AI difficulty display and room metadata.
- Undo vs AI button.
- Promotion selection modal.
- Result banner with win/draw reason.
- Engine suggestion arrow for analysis/assist mode.
- Custom premove style.

Engineering:

- Move chess UI into `features/chess`.
- Create `useChessRealtime` and `useChessBoardController`.
- Use backend `legal_moves` where useful, while keeping client-side `chess.js` for board UX.
- Add frontend tests for snapshot reconciliation.
- Add backend tests for AI thinking state and error paths.

## Phase 3: Realtime Robustness

Goal: make websocket lifecycle predictable.

Backend:

- Key client registry by `(roomID, playerID)` rather than only `playerID`.
- Introduce explicit connection/session IDs.
- Separate `player_disconnected_pending` from final `player_left`.
- Broadcast final leave only after reconnect grace expires, or mark event as transient.
- Add health and readiness endpoints.
- Normalize client IP for rate limiting behind reverse proxy.
- Restrict websocket origins in production.

Frontend:

- Replace reload-on-websocket-error with reconnect UI and room recovery.
- Add route-level room validation when localStorage has stale room data.
- Track socket states: connecting, open, reconnecting, rejected, expired.
- Avoid storing cross-game `roomId` without game type context.

## Phase 4: Persistence and Game History

Goal: survive process restarts and enable user-facing history.

Milestones:

- Add persistent player/session store.
- Store room metadata and lifecycle status.
- Persist completed game summaries.
- Persist chat only if intentionally part of the product.
- Add migration tooling if using Postgres.
- Add room expiration policy visible to users.

Recommended storage:

- Postgres for users, game summaries, durable room metadata.
- Redis for ephemeral room/session state if moving toward distributed realtime.

## Phase 5: Multiplayer Expansion

Goal: support more games and matchmaking without duplicating transport logic.

Work:

- Define a common game module interface for snapshots, moves, validation, reset, and AI hooks.
- Move TicTacToe and Chess behind consistent room-game adapters.
- Add spectator/read-only room mode only after player lifecycle is stable.
- Add matchmaking queues by game type and AI preference.
- Add rematch flow.
- Add invite/share room flow.

Scalability:

- Introduce room ownership model.
- Use sticky sessions for the first horizontal step.
- Later use Redis/NATS for event fanout.

## Phase 6: Horizontal Scaling

Goal: support multiple backend instances.

Required changes:

- Externalize room state or assign rooms to process owners.
- Externalize websocket fanout.
- Use distributed locks or single-room actors for mutation ownership.
- Replace process-local client registry with distributed presence metadata.
- Add graceful room draining on deploy.
- Move Stockfish to worker pool or AI service.

Possible architecture:

```text
Frontend
  -> Load balancer
  -> WebSocket gateway instances
  -> Room service / room actors
  -> Redis or NATS event bus
  -> Postgres for durable records
  -> AI worker pool
```

## Technical Debt Priorities

High:

- WebSocket lifecycle semantics.
- In-memory-only rooms and players.
- Player ID as global connection key.
- Frontend monolithic game components.
- Missing typed DTOs on frontend.
- Production CORS/origin restrictions.

Medium:

- Deprecated top-level chess DTO alias.
- Timer placeholders.
- Chat duplicated across game components.
- AI errors not surfaced clearly to users.
- Room code collision handling.
- Rate limiter behavior behind proxy.

Low:

- Folder cleanup.
- Remove legacy comments/logs.
- Consolidate alert helpers.
- Improve visual consistency across games.

## Deployment Milestones

1. Local Docker smoke test.
2. VPS deploy with backend + reverse proxy.
3. Frontend deploy with production env vars.
4. HTTPS domain setup.
5. Basic monitoring/logging.
6. Public demo checklist.
7. CI/CD with tests and image build.

## Polish Tasks

- Responsive chess layout pass.
- Replace placeholder timers with either hidden timers or real backend clocks.
- Improve waiting room copy and room code sharing.
- Add loading states for AI room creation.
- Add clear stale room recovery.
- Add game result modal/banner.
- Add accessible labels for icon buttons.

## Scalability Milestones

- 1 process, documented limits.
- 1 VPS, Dockerized, monitored.
- Durable user/game records.
- Redis presence/fanout prototype.
- Multi-instance websocket deployment with sticky sessions.
- Distributed room/event architecture.

