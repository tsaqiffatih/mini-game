# Backend Architecture

## Current Structure

The active backend is `server/`.

```text
server/
  main.go
  api/
    handlers.go
    websocket.go
    message.go
    game_handler.go
    client.go
    dto/
  service/
    game_service.go
  game/
    room.go
    room_actions.go
    player.go
    chat.go
    stockfish.go
  chess/
    chess_state.go
  tictactoe/
    tictactoe_state.go
    tictactoe_ai.go
  infrastructure/
    memory_room_repository.go
  middleware/
  internal/observability/
```

## Layer Responsibilities

Transport:

- `api/handlers.go`: HTTP routes.
- `api/websocket.go`: websocket connection lifecycle.
- `api/message.go`: websocket read loop and event dispatch.
- `api/game_handler.go`: event-specific handlers.
- `api/dto`: JSON contract conversion.

Application:

- `service/game_service.go`: orchestrates player lookup, room repository access, room mutation, cleanup, and logging.

Domain:

- `game.Room`: aggregate root for room lifecycle.
- `game.PlayerManager`: global in-memory player registry.
- `game.ChatMessage`: bounded room chat history.
- `chess.ChessGameState`: chess rules/state/metadata.
- `tictactoe.TictactoeGameState`: TicTacToe rules/state.

Infrastructure:

- `infrastructure.MemoryRoomRepository`: process-local room storage.
- `game.StockfishEngine`: process wrapper around Stockfish binary.

## DDD-Lite Direction

The backend already follows a useful DDD-lite shape:

- `Room` is the aggregate root.
- `GameService` is the application service.
- `RoomRepository` is an interface.
- `api/dto` prevents direct JSON exposure of domain objects.
- Game engines are domain modules.

The main inconsistency is that `game` currently also contains Stockfish infrastructure. Eventually Stockfish should move behind an infrastructure interface, but it does not need to block current development.

## Aggregate Ownership

`Room` owns:

- Players inside the room.
- Room state machine.
- Chess/TicTacToe state.
- Chat messages.
- AI player state.
- AI move scheduling.
- Reset scheduling.
- State versioning.
- Snapshot generation.

All important room mutations happen while holding `Room.mu`. This gives each room an isolated shared-memory state machine.

## Room Architecture

Room state:

```text
WAITING
  -> PLAYING
  -> FINISHED
  -> RESETTING
  -> PLAYING or WAITING
```

Rules:

- Max two players.
- AI counts as one player.
- First TicTacToe human is `X`; second is `O`.
- Chess AI is black; human is usually white in AI rooms.
- A room becomes active when two players exist.
- Removing down to one player resets the game and returns to `WAITING`.
- Removing all players deletes the room through repository cleanup path.

## WebSocket Architecture

`ClientRegistry` is a process-local map:

```go
map[playerID]*Client
```

Each `Client` has:

- websocket connection
- buffered send channel
- done channel
- close-once guard

Writes are safe because all outbound messages go through the `Send` channel and one `WritePump`.

Broadcasts are direct:

```text
RoomSnapshot.Players
  -> clients.Get(player.ID)
  -> client.Enqueue(event)
```

This is not external pub/sub. It is in-process direct fanout.

## Concurrency Ownership

Concurrency tools:

- `Room.mu`: protects room state.
- `PlayerManager.mu`: protects global players.
- `MemoryRoomRepository.mu`: protects room map.
- `ClientRegistry.mu`: protects client map.
- Client send channels: serialize websocket writes.
- Goroutines:
  - websocket write pump
  - websocket read loop
  - delayed disconnect removal
  - scheduled room reset
  - scheduled AI move
  - cleanup tickers

Strengths:

- Room state is not mutated without locks.
- Websocket writes are serialized.
- AI scheduled moves use version checks to avoid applying stale moves.
- Reset timers use version checks.

Risks:

- Process-local shared memory limits horizontal scaling.
- Stockfish call happens outside room lock, which is good, but AI lifecycle is still room-owned.
- Client registry keyed by player ID is too broad for multi-room.
- Async room mutations need reliable notifier behavior for every visible state change.

## AI Ownership

TicTacToe AI:

- Pure in-process minimax/random mix.
- Scheduled after human move.
- Uses same move application path.

Chess AI:

- Stockfish process per AI room.
- `Room` schedules AI move after human move.
- AI move applies through `handleChessMoveLocked`, same as human moves.
- `aiThinking` is included in snapshots.

Future direction:

- Extract `ChessEngine` interface.
- Add Stockfish process pool.
- Add AI job metrics and timeouts.
- Move engine infrastructure out of `game`.

## Service Boundaries

`GameService` should remain orchestration-only:

- Fetch room/player.
- Call room methods.
- Return snapshots/results.
- Start cleanup.
- Attach room notifier.

It should not own chess rules, move validation, or websocket serialization.

Current service is mostly aligned with this.

## Shared Infrastructure

Current infrastructure is intentionally minimal:

- In-memory room repository.
- In-memory player manager.
- In-memory client registry.
- JSON logs.
- In-memory rate limiter.

This is simple and fast, but all state disappears on restart.

## Future Multiplayer Game Scalability

Before adding many games, define a common game adapter boundary:

```text
GameModule
  ApplyMove(command)
  Snapshot()
  Reset()
  Status()
  MaybeScheduleAI()
```

Room should continue to own lifecycle, membership, and state versioning. Individual games should own rules.

Scaling path:

1. Keep single-process architecture but clean contracts.
2. Add persistence for user/game records.
3. Add Redis for room presence and event fanout.
4. Move to room actors or room ownership per process.
5. Add websocket gateway layer when horizontal scaling is required.

## Current Backend Weaknesses

- In-memory-only runtime state.
- Websocket registry keyed only by player ID.
- WebSocket origin allows all origins.
- HTTP rate limiter may affect websocket upgrades.
- Room code generation does not retry on collision.
- Stockfish infrastructure sits in domain package.
- No health/readiness endpoint.
- No durable game history.
- Immediate `player_left` event does not distinguish transient disconnect from final leave.

## Refactor Recommendations

Near-term:

- Add room-scoped client registry.
- Add typed websocket event constants shared in docs/tests.
- Add room recovery endpoint.
- Tighten websocket origin policy.
- Add health endpoint.
- Add tests for reconnect/disconnect grace.

Mid-term:

- Extract websocket package under `api/websocket`.
- Extract Stockfish behind interface.
- Add persistent repositories.
- Introduce frontend/backend contract tests.

Long-term:

- Split room actor/event architecture for horizontal scale.
- Add pub/sub fanout.
- Add matchmaking service.

