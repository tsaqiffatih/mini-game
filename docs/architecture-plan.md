# Architecture Plan

## Purpose

This document describes the active architecture of the `mini-game` monorepo as it exists now. It intentionally treats `server/` as the active backend and `archive/old-server/` as historical reference only.

The project is a realtime room-based multiplayer game platform with:

- Next.js frontend in `mini-game-client/`
- Go backend in `server/`
- WebSocket synchronization for room/game/chat state
- In-memory room and player state
- TicTacToe and Chess engines
- AI rooms, including Stockfish-backed chess AI

## Monorepo Structure

```text
mini-game/
  mini-game-client/     active frontend
  server/               active backend
  archive/old-server/   legacy backend reference only
  docs/                 architecture and planning documentation
```

The active runtime dependency flow is:

```text
Browser
  -> Next.js pages/components
  -> HTTP lobby/register calls
  -> WebSocket game/chat events
  -> Go api layer
  -> GameService
  -> Room aggregate
  -> Chess/TicTacToe engines
  -> RoomSnapshot DTO
  -> WebSocket broadcast
  -> Browser local UI reconciliation
```

## Frontend Architecture

The frontend is a Next.js App Router project using React 19, Tailwind/DaisyUI, `axios`, `react-use-websocket`, `chess.js`, and `chessground`.

Important folders:

- `src/app/`: route entries.
- `src/components/`: UI/game components. Current structure is mostly global component-based rather than feature-based.
- `src/utils/`: shared websocket, chess, sound, alert, and message helper functions.
- `public/sounds/`: chess sound assets.
- `public/chess/`: chess piece/avatar assets.

Current route ownership:

```text
/                  src/app/page.tsx
/tictactoe          src/app/tictactoe/page.tsx
/chess              src/app/chess/page.tsx
/example            src/app/example/page.tsx
```

The home page owns player registration state at a very light level by reading `localStorage.playerId`. Registration is performed by `RegisterUser`, which posts to `POST /create/user`.

The game pages own room bootstrap:

- Read `playerId`, `roomId`, and `playerMark` from `localStorage`.
- Redirect to `/` if no player exists.
- Render `Lobby` when no room is selected.
- Render the game board component once a room exists.

`Lobby` owns room creation/joining:

- `POST /room/create`
- `POST /room/create/ai`
- `POST /room/join`
- Writes room data back to `localStorage`.

`useGameWebSocket` in `src/utils/gameWebsocket.ts` owns the physical websocket connection:

```text
NEXT_PUBLIC_WS_BACKEND_URL/ws?room_id={roomId}&player_id={playerId}
```

It reconnects automatically and treats websocket errors as expired/missing rooms by clearing room keys and reloading the page.

### Chess Frontend

`src/components/ChessBoard.tsx` is currently the central chess feature component. It owns:

- WebSocket connection through `useGameWebSocket`.
- Local `chess.js` instance in `chessRef`.
- Chessground board lifecycle.
- Room state rendering: `WAITING`, `PLAYING`, `FINISHED`, `RESETTING`.
- Local turn checks.
- Move submission as `CHESS_MOVE`.
- Chat send/receive.
- Move history.
- Sound effects.
- Captured piece display.

Important helper ownership:

- `handleGameChessUpdate.ts`: reconciles backend snapshots into React and `chess.js` state.
- `chessUtils.ts`: FEN turn parsing, legal destination generation, move metadata parsing.
- `useChessSounds.ts`: sound loading/playback.
- `handleChatChessMessage.ts` and `handleChatChessHistory.ts`: chat normalization/deduplication.

The frontend performs local validation for UX using `chess.js`, but the backend remains authoritative. A frontend-accepted move can still be rejected by the backend with `chess_move_rejected`.

### TicTacToe Frontend

`src/components/TicTacToeBoard.tsx` owns:

- WebSocket connection through `useGameWebSocket`.
- Board/turn/winner state.
- Chat state.
- `TICTACTOE_MOVE` send path.
- Room state rendering.

Compared with chess, TicTacToe uses less helper modularization and does not use a local domain engine.

## Backend Architecture

The active backend is a Go application using Gorilla Mux, Gorilla WebSocket, in-memory repositories, and room aggregates.

Important folders:

- `server/main.go`: application composition and lifecycle.
- `server/api/`: HTTP handlers, websocket handlers, event dispatch, DTO conversion.
- `server/service/`: application service layer.
- `server/game/`: room aggregate, player management, chat, AI scheduling, Stockfish wrapper.
- `server/chess/`: chess domain state and move metadata built on `github.com/notnil/chess`.
- `server/tictactoe/`: TicTacToe state and AI.
- `server/infrastructure/`: in-memory room repository.
- `server/internal/observability/`: structured logging and request middleware.
- `server/middleware/`: CORS and in-memory IP rate limiter.

Dependency direction:

```text
main
  -> api
  -> service
  -> game
  -> chess / tictactoe
  -> infrastructure implements service.RoomRepository
```

`api` depends on `service` and DTOs. `service` depends on `game`. `game` owns domain state and calls chess/tictactoe packages. `infrastructure` stores `*game.Room` objects behind the repository interface.

## Startup Flow

`server/main.go` performs:

1. Initialize JSON slog observability.
2. Load `.env` if present.
3. Resolve `PORT`, defaulting to `8080`.
4. Construct:
   - `game.NewPlayerManager()`
   - `infrastructure.NewMemoryRoomRepository()`
   - `service.NewGameService(roomRepository, playerManager)`
   - `api.NewClientRegistry()`
5. Register a `roomNotifier` so room-internal async changes can broadcast `game_update`.
6. Create a shutdown context from OS signals.
7. Start rate limiter cleanup.
8. Register HTTP and websocket routes.
9. Attach CORS, request logging, and rate limiting middleware.
10. Start periodic player and room cleanup.
11. Start HTTP server.
12. On shutdown, close websocket clients, stop HTTP server, and cleanup rooms.

## Backend HTTP Routes

Registered in `server/api/handlers.go`:

- `POST /create/user`
- `POST /room/create`
- `POST /room/create/ai`
- `POST /room/join`
- `GET /ws`
- `GET /`

HTTP is mostly used for identity and room bootstrap. Realtime gameplay is websocket-first.

## WebSocket Architecture

The websocket architecture is a direct registry/broadcast model, not a full actor system or external pub/sub system.

Core files:

- `server/api/websocket.go`
- `server/api/client.go`
- `server/api/message.go`
- `server/api/game_handler.go`

Connection lifecycle:

```text
Client opens /ws?room_id&player_id
  -> validate query params
  -> verify player exists in room
  -> upgrade HTTP connection
  -> ClientRegistry.Attach(playerID, conn)
  -> mark player connected in room
  -> start client WritePump
  -> broadcast player_joined room event
  -> send room_update snapshot to connecting client
  -> send chat_history to connecting client
  -> read inbound messages until socket closes
  -> remove exact client instance from registry
  -> mark player disconnected
  -> broadcast player_left
  -> schedule delayed room removal/player removal
```

`ClientRegistry` is keyed by `playerID`, not `(roomID, playerID)`. A new connection for the same player replaces and closes the old connection. This works for single-room play but becomes a limitation if the same player identity can open multiple rooms/tabs concurrently.

Outbound writes are serialized through a per-client buffered `Send` channel and `WritePump`. This avoids concurrent writes to one Gorilla websocket connection. If a client's queue fills, `Enqueue` closes that client.

Inbound messages are read in `readMessages` and dispatched by message `type`:

- `TICTACTOE_MOVE`
- `CHESS_MOVE`
- `CHESS_UNDO_REQUEST`
- `CHAT_SEND`
- `CREATE_ROOM_WITH_AI` legacy/limited path

Broadcasting is snapshot-based:

```text
Room mutation
  -> RoomSnapshot
  -> dto.FromRoomSnapshot
  -> Event JSON
  -> for each player in snapshot.Players
  -> clients.Get(player.ID)
  -> client.Enqueue
```

## Room Lifecycle

`server/game/room.go` and `server/game/room_actions.go` define `Room` as the aggregate root.

Room states:

```text
WAITING -> PLAYING -> FINISHED -> RESETTING -> PLAYING/WAITING
```

Creation:

- `GameService.CreateRoomWithContext`
- Generate 7-character room code.
- `game.NewRoom(roomID, gameType)`
- Save in `MemoryRoomRepository`.
- Add creator player.

AI creation:

- `GameService.CreateRoomWithAILevelWithContext`
- `game.NewRoomWithAILevel`
- `Room.EnableAILevel`
- For chess, creates a Stockfish process immediately.
- Human player joins after AI has already been inserted.

Joining:

- `GameService.JoinRoomWithContext`
- Fetch global player from `PlayerManager`.
- Fetch room from repository.
- Validate game type.
- `room.AddPlayer`.
- If second player joins, room transitions to `PLAYING`.

Disconnect:

- WebSocket close immediately marks player disconnected and broadcasts `player_left`.
- `RemovePlayerAfterDelay` waits 30 seconds.
- If the same player is connected again, removal is skipped.
- If still disconnected, `Room.HandlePlayerDisconnected` removes the player.
- If room becomes empty, repository deletes the room.
- If one player remains, room returns to `WAITING` and resets game state.

Game end:

- Domain engine marks game complete.
- Room transitions to `FINISHED`.
- Room schedules reset:
  - after 5 seconds: `RESETTING`
  - after 3 more seconds: reset board/chess state
  - if two players remain: `PLAYING`
  - otherwise: `WAITING`

## Multiplayer Synchronization Flow

The backend is authoritative. Clients submit intents; server validates and broadcasts full snapshots.

Chess move:

```text
ChessBoard.handleChessgroundMove
  -> local chess.js pre-check
  -> send { type: "CHESS_MOVE", payload: { from, to, promotion } }
  -> api.processChessMove
  -> GameService.HandleChessMoveWithContext
  -> Room.HandleChessMoveWithContext
  -> ChessGameState.UpdateState
  -> Room bumps stateVersion
  -> API fetches RoomSnapshot
  -> broadcast game_update
  -> frontend handleGameChessUpdate loads FEN into chess.js and updates UI
```

TicTacToe move:

```text
TicTacToeBoard.handleCellClick
  -> send TICTACTOE_MOVE
  -> api.processTicTacToeMove
  -> GameService.HandleTicTacToeMoveWithContext
  -> Room.HandleTicTacToeMove
  -> TictactoeGameState.ApplyMove
  -> Room bumps stateVersion
  -> broadcast game_update
```

Chat:

```text
ChatOpened
  -> CHAT_SEND
  -> GameService.HandleChatMessageWithContext
  -> Room.AddChatMessage
  -> EventChatMessage broadcast
```

Snapshots contain `state_version`. The chess frontend ignores snapshots whose `state_version` is lower than or equal to the latest applied version.

## AI Chess Architecture

AI chess is backend-owned.

Key files:

- `server/game/room_actions.go`
- `server/game/stockfish.go`
- `server/chess/chess_state.go`
- `mini-game-client/src/components/AIDifficultyModal.tsx`
- `mini-game-client/src/components/Lobby.tsx`

Flow:

```text
Lobby creates AI room with ai_level
  -> backend creates Room with AI player "AI"
  -> chess AI mark is black
  -> StockfishEngine starts per AI room
  -> human joins as white
  -> room becomes PLAYING
  -> human CHESS_MOVE is applied
  -> Room builds chessAIMoveRequest
  -> aiThinking=true, stateVersion bumps
  -> after aiMoveDelay, Stockfish BestMove(fen)
  -> AI move applied through same ChessGameState.UpdateState
  -> roomNotifier broadcasts game_update
```

The frontend already receives AI metadata in snapshots (`game.chess.ai.thinking`, `level`, `color`, `player_id`), but the current UI does not fully surface this as a polished thinking indicator or AI identity panel.

Undo vs AI exists on the backend via `CHESS_UNDO_REQUEST` and `Room.HandleChessUndo`, but the current chess UI does not expose a complete undo control.

## Ownership Boundaries

Authoritative ownership:

- Room lifecycle: `game.Room`.
- Player membership inside a room: `game.Room`.
- Global player registration: `game.PlayerManager`.
- Chess rules and metadata: `chess.ChessGameState`.
- TicTacToe rules: `tictactoe.TictactoeGameState`.
- Transport contract: `api` DTO/events.
- Frontend local UI representation: React components and local `chess.js` mirror.

Important boundary: frontend `chess.js` is not authoritative. It improves interaction quality but server snapshots win.

## Current Weaknesses

- All active state is in memory. Server restart deletes rooms, players, chat history, websocket registry, and active Stockfish processes.
- `ClientRegistry` is keyed only by `playerID`, limiting multi-room/multi-tab behavior.
- WebSocket reconnect policy is broad. The frontend always reconnects, but `onError` also clears room state and reloads.
- The backend broadcasts `player_left` immediately on transient disconnect, before the 30-second reconnect grace period finishes.
- HTTP join/create updates are not enough by themselves to notify already connected clients; meaningful room sync happens when websockets connect or snapshots broadcast.
- Chess frontend consumes `state.game?.chess ?? state.chess`, which supports current canonical and deprecated DTO shapes but keeps compatibility complexity alive.
- Frontend feature ownership is concentrated in large components.
- Timers are currently UI placeholders in chess (`10:00`, `09:42`), not authoritative clock state.
- Rate limiting uses `r.RemoteAddr`, which includes port and can behave poorly behind proxies unless normalized with trusted forwarded headers.
- CORS `CheckOrigin` for websocket currently allows all origins.

## Future DDD-Lite Direction

The backend is already close to DDD-lite:

- `Room` is the aggregate root.
- `GameService` is an application service.
- `MemoryRoomRepository` is infrastructure.
- `api/dto` is a transport boundary.
- `chess` and `tictactoe` are domain submodules.

Recommended direction:

```text
server/
  api/
    http/
    websocket/
    dto/
  application/
    game_service.go
    commands.go
  domain/
    room/
    player/
    chat/
    chess/
    tictactoe/
  infrastructure/
    memory/
    redis/
    stockfish/
```

The next step should not be a large rewrite. First stabilize contracts and persistence seams:

- Keep `Room` authoritative.
- Introduce repository interfaces for players and chat.
- Split websocket registry from websocket event dispatch.
- Make snapshots/events versioned and documented as API contracts.
- Use explicit room-scoped connection identity.

## Future Feature-Based Frontend Direction

Recommended frontend shape:

```text
src/
  app/
  features/
    lobby/
    chess/
      components/
      hooks/
      utils/
      types/
      api/
    tictactoe/
      components/
      hooks/
      types/
    chat/
    player/
  shared/
    api/
    websocket/
    ui/
    sounds/
    alerts/
```

Move websocket message parsing into feature hooks:

- `useRoomSocket`
- `useChessRealtime`
- `useTicTacToeRealtime`
- `useChatRealtime`

The goal is to make pages compose features rather than owning behavior directly.

## Scalability Strategy

Current production ceiling is one backend process. Horizontal scaling will not work correctly without sticky sessions and shared state because rooms and clients are process-local.

Milestones:

1. Production single-node hardening:
   - Docker image.
   - Reverse proxy with websocket upgrade.
   - SSL.
   - Structured logs.
   - Health endpoint.
   - Safer CORS/origin config.

2. Durable single-node:
   - Persistent player/session store.
   - Persistent room metadata.
   - Optional room replay/event log.
   - Graceful restart strategy.

3. Multi-process realtime:
   - Redis/NATS pub-sub for room events.
   - Redis room registry or database-backed room ownership.
   - Room affinity/sticky sessions or room actor ownership.
   - Distributed connection registry.

4. Matchmaking:
   - Queue service.
   - Rating/difficulty metadata.
   - Room allocation service.

5. AI scaling:
   - Stockfish process pool.
   - Per-room AI job queue.
   - Timeout/cancellation monitoring.
   - CPU quotas per AI request.

## Recommended Production Architecture

```text
Browser
  -> CDN / Vercel / static Next frontend
  -> api.example.com
  -> Nginx/Caddy reverse proxy
  -> Go backend container
  -> in-memory rooms initially
  -> logs/metrics

Future:
  -> Redis for websocket fanout, room metadata, queues
  -> Postgres for users/game history
```

The project can be portfolio-ready on a single VPS first, as long as limitations are explicit and the deployment is clean.

