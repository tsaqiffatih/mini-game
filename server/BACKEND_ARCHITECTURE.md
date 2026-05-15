PROMPT:
You are a senior staff-level Go backend engineer and software architect.

Your task is to deeply analyze and document the ENTIRE backend codebase.

IMPORTANT:

* DO NOT modify any code
* DO NOT suggest refactors unless explicitly asked
* DO NOT invent behavior
* ONLY explain what ACTUALLY exists in the current codebase
* Your goal is to help another senior engineer fully understand the system architecture, runtime flow, ownership boundaries, realtime flow, concurrency model, and scaling characteristics.

I want a COMPLETE backend architecture analysis.

The project is a realtime multiplayer game server using:

* Go
* WebSocket
* Room-based architecture
* Chess and TicTacToe game engines

Analyze ALL relevant folders and files.

==================================================
PRIMARY GOAL
============

Generate a HIGH-LEVEL + LOW-LEVEL technical architecture report explaining:

* How the server works
* How requests/events flow
* How websocket events flow
* How game state is owned and mutated
* How rooms work
* How concurrency is handled
* How broadcasting works
* How snapshots work
* How services interact
* How handlers interact
* How game engines interact
* Which layers are coupled
* Which layers are isolated
* What architectural patterns are currently being used
* Potential scalability bottlenecks
* Current runtime lifecycle of the system

The explanation should assume the reader is an experienced backend engineer.

==================================================
ANALYSIS REQUIREMENTS
=====================

Please explain the following sections in depth.

# 1. Project Structure Overview

Explain:

* folder responsibilities
* layer responsibilities
* dependency direction
* ownership boundaries

Explain which folders are:

* transport layer
* application layer
* domain layer
* infrastructure layer
* game engine layer

Explain how the architecture is organized overall.

==================================================

# 2. Application Startup Flow

Explain:

* what happens from main.go startup
* initialization flow
* dependency construction
* websocket setup
* router setup
* service setup
* registry setup
* room manager setup
* observability setup

Generate a startup sequence explanation.

==================================================

# 3. WebSocket Architecture

Explain IN DETAIL:

* websocket connection lifecycle
* client registration
* client unregister/disconnect flow
* read loop
* write loop
* send channels
* broadcasting mechanism
* event parsing
* event dispatching
* context propagation
* error handling
* backpressure handling
* slow client handling

Show:

* how messages travel through the system
* who owns websocket writes
* whether writes are synchronized safely

IMPORTANT:
Identify whether architecture resembles:

* hub model
* actor model
* event bus
* direct room broadcast
* pub/sub
* shared memory concurrency

==================================================

# 4. Room Architecture

Explain:

* room lifecycle
* room creation
* room destruction
* room ownership
* room responsibilities
* room state ownership
* player management
* spectator support (if exists)
* reconnection handling (if exists)

Explain:

* who is allowed to mutate room state
* whether room acts as aggregate root
* whether room is authoritative

IMPORTANT:
Identify:

* single-owner patterns
* encapsulation quality
* coupling level

==================================================

# 5. Game State Architecture

Explain:

* how Chess state works
* how TicTacToe state works
* how game state is stored
* how game state is mutated
* whether mutations are centralized
* how snapshots are generated
* whether state exposure is safe

Explain:

* state ownership boundaries
* synchronization guarantees
* mutation guarantees

==================================================

# 6. Concurrency Model

VERY IMPORTANT.

Analyze:

* mutex usage
* lock ownership
* lock granularity
* race-condition prevention
* goroutine ownership
* shared memory patterns
* channels usage
* concurrent access patterns

Explain:

* whether the system uses actor-like patterns
* whether rooms behave like isolated state machines
* where contention may happen
* where deadlocks could theoretically happen
* where blocking operations may exist

IMPORTANT:
Explain the REAL concurrency architecture.

==================================================

# 7. Service Layer Architecture

Explain:

* service responsibilities
* orchestration responsibilities
* business logic ownership
* coordination responsibilities

Explain how services interact with:

* rooms
* websocket layer
* game engines
* snapshots

==================================================

# 8. Event Flow Analysis

For EACH major event:

* chess move
* tictactoe move
* chat message
* join room
* leave room
* reconnect
* game start
* game end

Explain COMPLETE FLOW:

Client
→ websocket
→ handler
→ service
→ room
→ game engine
→ snapshot
→ broadcast
→ client

Show ACTUAL runtime sequence.

==================================================

# 9. Snapshot System

Explain:

* how snapshots are created
* why snapshots exist
* snapshot ownership
* serialization flow
* websocket payload flow

Explain whether snapshots act as:

* DTOs
* read models
* projections

==================================================

# 10. Observability

Analyze:

* logging quality
* structured logging usage
* context propagation
* trace readiness
* metrics readiness
* OpenTelemetry readiness

Explain:

* what is already good
* what observability architecture already exists

DO NOT suggest improvements unless explicitly asked.

==================================================

# 11. Scaling Characteristics

Analyze current architecture and explain:

* expected bottlenecks
* websocket scaling characteristics
* room scaling characteristics
* memory scaling
* CPU hotspots
* mutex contention possibilities
* network bottlenecks

Explain:

* whether horizontal scaling is currently possible
* whether sticky sessions would be required
* whether rooms are process-local

==================================================

# 12. Architectural Pattern Identification

Identify which patterns CURRENTLY exist in the codebase.

Examples:

* aggregate root
* service layer
* repository pattern
* actor-like model
* command handling
* event-driven architecture
* CQRS-like separation
* domain isolation
* transport/domain separation

IMPORTANT:
Only identify patterns that ACTUALLY EXIST.

==================================================

# 13. Runtime Ownership Map

Generate a clear explanation of:

WHO OWNS:

* websocket connections
* room state
* game state
* broadcasts
* player lifecycle
* snapshots
* synchronization

This is VERY IMPORTANT.

==================================================

# 14. Critical Files & Important Components

Identify:

* most important files
* most important structs
* most important interfaces
* runtime-critical components

Explain WHY they are important.

==================================================

# 15. Final Architecture Summary

Generate:

* high-level architecture summary
* strengths of current architecture
* current maturity level
* overall backend style
* likely architectural direction

==================================================
OUTPUT FORMAT
=============

Use:

* clear sections
* runtime flow diagrams
* ownership diagrams
* sequence explanations
* dependency explanations
* concurrency explanations

The goal is:
Another senior engineer should be able to fully understand the backend architecture after reading the report.

Again:

* DO NOT MODIFY CODE
* DO NOT INVENT BEHAVIOR
* ONLY ANALYZE CURRENT IMPLEMENTATION
* BASE EVERYTHING ON REAL CODE



=============================================
HASIL
=============================================


**Backend Architecture Report**

Scope: read-only analysis of the current code in `/home/tsaqiffatih/programming/mini-game/server-copy`. I’m describing only behavior present in the implementation.

---

**1. Project Structure Overview**

The backend is organized as a small layered Go service:

```text
main.go
  wires runtime dependencies, HTTP server, middleware, cleanup goroutines

api/
  HTTP + WebSocket transport layer
  request parsing, response DTOs, WebSocket event parsing, client registry, broadcasting

service/
  application/service layer
  orchestration around players, rooms, repository access, room snapshots, move handling

game/
  domain layer for rooms, players, chat, lifecycle, room state transitions

chess/
  chess game engine wrapper around github.com/notnil/chess

tictactoe/
  TicTacToe game engine and AI helper

infrastructure/
  in-memory room repository

middleware/
  CORS and IP-based rate limiting

internal/observability/
  structured logging, request middleware, trace-id propagation helpers

actions/
  string constants for WebSocket event names
```

Dependency direction is mostly:

```text
main
 ├─ api
 ├─ service
 ├─ game
 ├─ infrastructure
 ├─ middleware
 └─ observability

api ──> service ──> game ──> chess/tictactoe
              └──> infrastructure via RoomRepository interface

api/dto ──> game snapshots
infrastructure ──> game.Room
```

The architectural center is `game.Room`. It owns room state, player membership, game engine instances, chat history, lifecycle transitions, reset scheduling, and per-room synchronization.

Layer classification:

| Folder | Layer | Responsibility |
|---|---|---|
| `api` | Transport | HTTP routes, WebSocket upgrade, event read/write, DTO mapping |
| `service` | Application | Use-case orchestration, room lookup, player lookup, move/chat delegation |
| `game` | Domain | Room aggregate, player lifecycle, chat, snapshots, room state machine |
| `chess` | Game engine | Chess rules/state via `notnil/chess` |
| `tictactoe` | Game engine | TicTacToe rules/state and standalone AI move computation |
| `infrastructure` | Infrastructure | Process-local in-memory room repository |
| `middleware` | Infrastructure/transport support | CORS and rate limiting |
| `internal/observability` | Infrastructure | JSON logging, request tracing helpers |

Ownership boundaries:

```text
WebSocket connection ownership: api.ClientRegistry / api.Client
Room ownership:               infrastructure.MemoryRoomRepository stores *game.Room
Room state ownership:          game.Room
Game engine ownership:         game.Room owns chess/tictactoe state pointers
Player registry ownership:     game.PlayerManager
Broadcast ownership:           api.NotifySnapshotToClients and helpers
Snapshot ownership:            game.Room creates domain snapshots; api/dto maps to JSON DTOs
```

---

**2. Application Startup Flow**

Startup begins in [main.go](/home/tsaqiffatih/programming/mini-game/server-copy/main.go):

```text
main()
 ├─ observability.Init("mini-game")
 ├─ optionally loads .env
 ├─ resolves PORT, default 8080
 ├─ creates PlayerManager
 ├─ creates MemoryRoomRepository
 ├─ creates GameService(repository, playerManager)
 ├─ creates ClientRegistry
 ├─ injects room notifier into GameService
 │    └─ notifier calls api.NotifyGameUpdateToClients(clients, snapshot)
 ├─ creates root signal context
 ├─ gives root context to GameService
 ├─ starts rate limiter cleanup goroutine
 ├─ creates gorilla/mux router
 ├─ registers root health-ish route
 ├─ api.RegisterRouter(...)
 ├─ configures CORS
 ├─ starts player cleanup goroutine
 ├─ starts room cleanup goroutine
 ├─ installs observability + rate limiter middleware
 ├─ starts http.Server.ListenAndServe in goroutine
 ├─ waits for signal context cancellation
 └─ shutdown:
      ├─ clients.CloseAll()
      ├─ server.Shutdown(10s timeout)
      ├─ gameService.CleanupRooms(..., inactiveFor=0)
      └─ logs shutdown complete
```

Important runtime construction:

```text
PlayerManager
  └─ global process-local player map

MemoryRoomRepository
  └─ global process-local room map

GameService
  ├─ RoomRepository interface
  ├─ PlayerManager
  ├─ root context
  └─ roomNotifier callback

ClientRegistry
  └─ global process-local playerID -> WebSocket client map
```

HTTP routes registered in `api.RegisterRouter`:

```text
POST /create/user
POST /room/join
POST /room/create
POST /room/create/ai
GET  /ws
```

Observability setup consists of JSON `slog`, request middleware, trace-id extraction from `traceparent` or `X-Request-ID`, and span-like log helpers.

---

**3. WebSocket Architecture**

The WebSocket endpoint is [api/websocket.go](/home/tsaqiffatih/programming/mini-game/server-copy/api/websocket.go).

Connection lifecycle:

```text
Client connects to /ws?room_id=...&player_id=...
 ├─ validate query params
 ├─ service.GetPlayerInRoomWithContext(roomID, playerID)
 ├─ gorilla websocket upgrade
 ├─ clients.Attach(playerID, conn, pongWait)
 │    ├─ creates Client{PlayerID, Conn, Send buffered chan 256, done}
 │    ├─ installs pong handler / read deadline
 │    ├─ closes existing client for same playerID if present
 │    └─ stores new client in global registry
 ├─ service.MarkPlayerConnectedWithContext(...)
 ├─ starts client.WritePump(...)
 ├─ notifyRoomOnConnection(...)
 │    ├─ broadcasts player_joined to room players
 │    ├─ sends room snapshot to connecting client
 │    └─ sends chat history to connecting client
 ├─ starts readMessages goroutine
 ├─ waits on done
 └─ on read loop exit:
      ├─ clients.RemoveClient(client)
      ├─ service.MarkPlayerDisconnectedWithContext(...)
      ├─ broadcasts player_left
      └─ schedules RemovePlayerAfterDelay(30s)
```

Read loop:

```text
conn.ReadMessage()
 ├─ JSON unmarshal into WebSocketMessage{type,payload}
 ├─ log event
 ├─ gameService.UpdatePlayerActivityWithContext(...)
 └─ handleMessageAction(...)
```

Supported inbound WebSocket events in current code:

```text
TICTACTOE_MOVE
CHESS_MOVE
CHAT_SEND
CREATE_ROOM_WITH_AI
```

Unsupported event types receive an `"error"` event.

Write loop:

Each `api.Client` owns one `Send chan []byte` and one `WritePump`.

```text
broadcast/send helper
 └─ client.Enqueue(bytes)
      ├─ if done closed: false
      ├─ if Send has capacity: enqueue true
      └─ if Send full: client.Close(); false

WritePump
 ├─ reads from Send
 │    ├─ SetWriteDeadline(writeWait=10s)
 │    └─ Conn.WriteMessage(TextMessage, message)
 ├─ every pingPeriod=30s
 │    ├─ SetWriteDeadline
 │    └─ Conn.WriteMessage(PingMessage, nil)
 └─ exits on done/write error/ping error
```

WebSocket writes are serialized by design: application code does not write directly to `websocket.Conn` after connection setup. Messages go through `Client.Send`, and the single `WritePump` goroutine performs writes. That matches Gorilla’s single-writer expectation.

Broadcasting:

```text
NotifySnapshotToClients(clients, roomSnapshot, events...)
 ├─ marshal Event once per event
 ├─ iterate snapshot.Players
 ├─ clients.Get(player.ID)
 └─ client.Enqueue(messageBytes)
      └─ if enqueue fails, RemoveClient(client)
```

The broadcast target set comes from `RoomSnapshot.Players`, not from a room-scoped WebSocket registry. The actual client registry is global by `playerID`.

Backpressure and slow clients:

| Mechanism | Actual behavior |
|---|---|
| Per-client buffer | `Send` channel size 256 |
| Full queue | `Client.Close()` and enqueue returns false |
| Broadcast path | removes client from registry if enqueue fails |
| Direct send path | logs slow-client warning but does not remove from registry |
| Write timeout | `SetWriteDeadline(10s)` |
| Heartbeat | ping every 30s, pong extends read deadline 60s |

Architecture style:

```text
It is not a central hub goroutine.
It is not a full actor model.
It is not external pub/sub.
It is process-local shared-memory concurrency.

Closest description:
  room aggregate + global client registry + direct room snapshot broadcast.

There is an actor-like trait at the room boundary:
  room mutations are serialized by room.mu.

But rooms do not own an event mailbox/goroutine.
Callers directly invoke room methods under mutexes.
```

---

**4. Room Architecture**

`game.Room` is the domain aggregate-like object in [game/room.go](/home/tsaqiffatih/programming/mini-game/server-copy/game/room.go) and [game/room_actions.go](/home/tsaqiffatih/programming/mini-game/server-copy/game/room_actions.go).

Room fields include:

```text
RoomID
players map[string]*Player
gameType
roomState
ticTacToe *tictactoe.TictactoeGameState
chess *chess.ChessGameState
isAIEnabled
reset scheduling fields
stateNotifier func(RoomSnapshot)
chatMessages []ChatMessage
mu sync.RWMutex
```

Room lifecycle:

```text
NewRoom(roomID, gameType)
 ├─ initializes players map
 ├─ roomState = WAITING
 ├─ creates tictactoe state if gameType == "tictactoe"
 ├─ creates chess state if gameType == "chess"
 └─ error for unknown game type

NewRoomWithAI(...)
 ├─ NewRoom(...)
 └─ EnableAI()
      └─ adds AI player with ID "AI", mark "O"
```

State machine:

```text
WAITING  -> PLAYING
PLAYING  -> FINISHED
FINISHED -> RESETTING
RESETTING -> PLAYING
```

Additional direct assignments exist in removal/reset paths:

```text
when player count drops below 2:
  roomState = WAITING

after reset with fewer than 2 players:
  roomState = WAITING
```

Player management:

```text
AddPlayer
 ├─ requires room not full
 ├─ requires roomState == WAITING
 ├─ rejects duplicate player in room
 ├─ assigns mark:
 │    ├─ TicTacToe: first human X, second O; AI is O
 │    └─ Chess: first white, second black
 ├─ when len(players)==2:
 │    ├─ transition WAITING -> PLAYING
 │    └─ activate TicTacToe status if applicable
 └─ returns JoinRoomResponse
```

Spectators:

There is no spectator model in the current code. Rooms cap `players` at 2 and broadcasts target players only.

Reconnection:

There is no explicit reconnect event. Reconnection exists implicitly:

```text
same player opens /ws again
 ├─ ClientRegistry.Attach closes existing connection for that playerID
 ├─ stores the new connection
 ├─ MarkPlayerConnected
 ├─ sends player_joined, room snapshot, chat history
 └─ delayed removal from prior disconnect checks clients.IsConnected(playerID)
```

Leave room:

There is no explicit inbound leave-room WebSocket action implemented. Leaving is represented by WebSocket read failure/close, then delayed domain removal after 30 seconds if the player did not reconnect.

Room destruction:

```text
RemovePlayerAfterDelay
 └─ room.HandlePlayerDisconnected(playerID)
      └─ if room becomes empty: repository.Delete(roomID)

CleanupRooms
 ├─ room.RemoveInactivePlayers(...)
 ├─ if room.IsEmpty() or roomInactive(...):
 │    ├─ room.Close()
 │    └─ repository.Delete(roomID)
```

Room authority:

`game.Room` is authoritative for room membership, room state, game state mutation, chat history, reset scheduling, and snapshots. The service layer does not mutate game internals directly.

---

**5. Game State Architecture**

TicTacToe:

`TictactoeGameState` in [tictactoe/tictactoe_state.go](/home/tsaqiffatih/programming/mini-game/server-copy/tictactoe/tictactoe_state.go) contains:

```text
Board [3][3]string
Turn string
Winner string
Status GameStatus
```

Statuses:

```text
waiting
active
ended
```

Mutation path:

```text
Room.HandleTicTacToeMove(...)
 ├─ room.mu.Lock()
 ├─ validate player exists
 ├─ validate ticTacToe state exists
 ├─ validate roomState == PLAYING
 ├─ ticTacToe.ApplyMove(player.Mark, row, col)
 │    ├─ requires StatusActive
 │    ├─ requires correct turn
 │    ├─ validates bounds and empty cell
 │    ├─ writes board cell
 │    ├─ computes winner/draw
 │    └─ switches turn if ongoing
 ├─ if ended:
 │    ├─ roomState PLAYING -> FINISHED
 │    └─ schedule reset
 └─ returns TicTacToeMoveResult
```

Chess:

`ChessGameState` in [chess/chess_state.go](/home/tsaqiffatih/programming/mini-game/server-copy/chess/chess_state.go) wraps `github.com/notnil/chess`.

State:

```text
game *notnil.Game
isActive bool
winner string
pgnMoves []string
```

Mutation path:

```text
Room.HandleChessMove(...)
 ├─ room.mu.Lock()
 ├─ validate player exists
 ├─ validate chess state exists
 ├─ validate roomState == PLAYING
 ├─ chess.UpdateState(player.Mark, from, to, promotion)
 │    ├─ requires active game
 │    ├─ requires player mark == current turn
 │    ├─ validates/apply UCI move through notnil/chess
 │    ├─ stores last move string
 │    └─ checks checkmate/stalemate/draw offer
 ├─ if result not ongoing:
 │    ├─ roomState PLAYING -> FINISHED
 │    └─ schedule reset
 └─ returns ChessMoveResult
```

Snapshots:

Room snapshots copy the room’s readable state while holding `room.mu.RLock()`.

```text
Room.Snapshot()
 ├─ copies room metadata
 ├─ copies players into []PlayerSnapshot
 ├─ copies TicTacToe board/turn/winner/status
 └─ copies Chess FEN/isActive/winner/PGNMoves
```

State exposure is mostly snapshot-based. The repository returns `*game.Room`, but external packages call exported room methods; game engine pointers themselves are not exposed from `Room`.

---

**6. Concurrency Model**

The real concurrency model is shared memory with mutexes and channels.

Main mutexes:

| Component | Lock | Protects |
|---|---|---|
| `game.Room` | `sync.RWMutex` | players, room state, game state, chat, reset fields, notifier |
| `game.PlayerManager` | `sync.RWMutex` | global player map |
| `infrastructure.MemoryRoomRepository` | `sync.RWMutex` | room map |
| `api.ClientRegistry` | `sync.RWMutex` | playerID -> client map |
| middleware limiter | `sync.Mutex` | IP limiter map |
| room code generator | `sync.Mutex` | shared `rand.Rand` |
| chat IDs | atomic uint64 | chat message sequence |

Goroutine ownership:

```text
main goroutine
 ├─ HTTP server goroutine
 ├─ PlayerManager.RemoveInactivePlayers cleanup goroutine
 ├─ GameService.StartRoomCleanup cleanup goroutine
 └─ rate limiter cleanup goroutine

per WebSocket connection
 ├─ WritePump goroutine
 └─ readMessages goroutine

per delayed disconnect
 └─ RemovePlayerAfterDelay goroutine

per finished game reset
 └─ Room.runScheduledReset goroutine
```

Channels:

```text
Client.Send chan []byte
  buffered outbound message queue

Client.done chan struct{}
  close signal for client lifecycle

readMessages done chan struct{}
  signals HandleWebSocket that read loop ended
```

Room mutation is serialized by `room.mu.Lock()`. Concurrent moves in the same room cannot mutate the board at the same time. Snapshots use `RLock`, so they wait behind writes and can run concurrently with other readers.

Repository locking protects only the map lookup/list/save/delete. Once `GetByID` returns `*game.Room`, room-level synchronization handles room internals.

Actor-like behavior:

Rooms behave like isolated synchronized state machines, but not actors in the strict sense. There is no room goroutine or mailbox. Multiple goroutines can call a room concurrently; serialization comes from `room.mu`.

Potential contention points from current design:

```text
Global ClientRegistry lock:
  every attach/get/remove/close-all uses global map lock.

Global RoomRepository lock:
  room lookup/list/save/delete all share one map lock.

Per-room lock:
  all moves, joins, chat writes, disconnect updates, resets, snapshots contend inside a room.

Broadcast fanout:
  broadcasting iterates all players in snapshot and does registry lookup per player.
  With current 2-player rooms this is small.

Cleanup:
  CleanupRooms lists all rooms, then touches each room.
```

Blocking operations:

```text
WebSocket writes can block inside WritePump until write deadline.
Client enqueue does not block if queue is full; it closes the client.
Room reset goroutines sleep using timers.
RemovePlayerAfterDelay sleeps for 30s before domain removal.
HTTP handlers block while waiting on room/service methods.
```

The code avoids holding `room.mu` while broadcasting in the notifier path: `notifyStateChanged` reads the notifier, releases the lock, then calls `r.Snapshot()` and invokes the notifier. That prevents broadcasting while holding the write lock.

---

**7. Service Layer Architecture**

`GameService` in [service/game_service.go](/home/tsaqiffatih/programming/mini-game/server-copy/service/game_service.go) is the application orchestration layer.

It owns references to:

```text
RoomRepository
PlayerManager
context.Context
roomNotifier func(game.RoomSnapshot)
```

Responsibilities:

```text
AddPlayer
CreateRoom
CreateRoomWithAI
JoinRoom
GetPlayerInRoom
UpdatePlayerActivity
RoomSnapshot
ChatHistory
HandleChatMessage
HandleTicTacToeMove
HandleChessMove
MarkPlayerConnected/Disconnected
RemovePlayerAfterDelay
CleanupRooms / StartRoomCleanup
```

The service does not implement chess or TicTacToe rules. It retrieves rooms and delegates mutations:

```text
service.HandleTicTacToeMove -> room.HandleTicTacToeMove -> tictactoe.ApplyMove
service.HandleChessMove     -> room.HandleChessMove     -> chess.UpdateState
service.HandleChatMessage   -> room.AddChatMessage
```

It also adapts errors into service-level errors for some flows, logs domain events, and starts span-like observability wrappers.

The service attaches the room notifier during room creation. That notifier is ultimately an API-layer callback wired in `main.go`, so the domain room has a callback field whose implementation points back to WebSocket broadcasting.

---

**8. Event Flow Analysis**

Chess move:

```text
Client sends {type:"CHESS_MOVE", payload:{from,to,promotion}}
 └─ WebSocket readMessages
    └─ handleMessageAction
       └─ processChessMove
          └─ parse dto.ChessMovePayload
             └─ gameService.HandleChessMoveWithContext
                └─ repository.GetByID
                   └─ room.HandleChessMove
                      └─ chess.UpdateState
                         └─ notnil/chess MoveStr
                      └─ maybe room FINISHED + schedule reset
                └─ service logs move
          └─ gameService.RoomSnapshotWithContext
             └─ room.Snapshot
          └─ NotifySnapshotToClients EventGameUpdate
             └─ client.Enqueue
                └─ WritePump writes JSON to WebSocket
```

TicTacToe move:

```text
Client sends {type:"TICTACTOE_MOVE", payload:{row,col}}
 └─ readMessages
    └─ processTicTacToeMove
       └─ gameService.HandleTicTacToeMoveWithContext
          └─ room.HandleTicTacToeMove
             └─ ticTacToe.ApplyMove
             └─ maybe room FINISHED + schedule reset
       └─ NotifyTicTacToeClients
          └─ gameService.RoomSnapshot
          └─ NotifySnapshotToClients EventGameUpdate
```

Chat message:

```text
Client sends {type:"CHAT_SEND", payload:{message}}
 └─ readMessages
    └─ processChatSend
       └─ gameService.HandleChatMessageWithContext
          └─ room.AddChatMessage
             ├─ validates player exists
             ├─ trims message
             ├─ validates non-empty and <= 300 runes
             ├─ creates atomic-sequence message ID
             └─ appends to bounded room chat history, max 50
       └─ gameService.RoomSnapshotWithContext
       └─ NotifySnapshotToClients EventChatMessage
          └─ payload is dto.FromChatMessageEvent(chatMessage)
```

Join room over HTTP:

```text
POST /room/join
 └─ api.joinRoom
    └─ decode room_id/player_id/game_type
       └─ gameService.JoinRoomWithContext
          ├─ playerManager.GetPlayer
          ├─ repository.GetByID
          ├─ room.GameType check
          └─ room.AddPlayer
             └─ maybe transition WAITING -> PLAYING
       └─ response dto.FromJoinRoomResponse
```

Join notification over WebSocket:

A joined player is not broadcast from the HTTP join handler. It is broadcast when the player connects to `/ws`:

```text
/ws connect
 └─ notifyRoomOnConnection
    └─ NotifyToClientsInRoom EventPlayerJoined
```

Leave room:

There is no explicit `USER_LEFT_ROOM` handler. Actual flow is WebSocket disconnect:

```text
ReadMessage error
 └─ readMessages closes done
    └─ HandleWebSocket resumes
       └─ handlePlayerDisconnection
          ├─ clients.RemoveClient
          ├─ gameService.MarkPlayerDisconnectedWithContext
          ├─ NotifyToClientsInRoom EventPlayerLeft
          └─ gameService.RemovePlayerAfterDelay(30s)
             └─ if not reconnected:
                └─ room.HandlePlayerDisconnected
                   ├─ remove player
                   ├─ if empty: repository.Delete
                   └─ if one remains: reset room/game to WAITING
```

Reconnect:

```text
Client connects again with same room_id/player_id
 ├─ service.GetPlayerInRoomWithContext succeeds if player not removed
 ├─ clients.Attach replaces old client for same playerID
 ├─ MarkPlayerConnected
 ├─ EventPlayerJoined broadcast
 ├─ room snapshot sent to that client
 └─ chat history sent to that client

Pending RemovePlayerAfterDelay checks clients.IsConnected(playerID).
If true, it does not remove the player.
```

Game start:

There is no implemented inbound `START_GAME` handler. Game start is automatic:

```text
second player successfully added to room
 └─ len(players)==2
    ├─ room.transitionLocked(WAITING -> PLAYING)
    └─ room.activateGameLocked()
       └─ for TicTacToe: StatusActive
```

For chess, the room transitions to `PLAYING`; the chess engine is active from construction.

Game end:

```text
TicTacToe ApplyMove detects win/draw
or Chess UpdateState detects non-ongoing result
 └─ Room.Handle...Move
    ├─ transition PLAYING -> FINISHED
    └─ scheduleResetLocked()
       └─ goroutine:
          ├─ wait finishedResetDelay, default 5s
          ├─ transition FINISHED -> RESETTING
          ├─ notifyStateChanged -> roomNotifier -> EventGameUpdate
          ├─ wait resettingDelay, default 3s
          ├─ reset game state
          ├─ if 2 players: transition RESETTING -> PLAYING
          └─ else: roomState = WAITING
          └─ notifyStateChanged -> EventGameUpdate
```

Create room with AI:

HTTP flow:

```text
POST /room/create/ai
 └─ gameService.CreateRoomWithAIWithContext
    ├─ NewRoomWithAI
    │  └─ EnableAI adds "AI" player
    ├─ repository.Save
    └─ room.AddPlayer(human)
       └─ len(players)==2 => PLAYING
```

WebSocket flow:

```text
CREATE_ROOM_WITH_AI with raw JSON string room ID
 └─ gameService.CreateRoomWithAIByIDWithContext(requestedRoomID, "tictactoe")
    └─ creates AI room by explicit ID
 └─ NotifyToClientsInRoom(EventRoomUpdate, dto.RoomDTO)
```

The TicTacToe AI engine `ComputeBestMove` exists, but current service/WebSocket move flow does not call it.

---

**9. Snapshot System**

Snapshots are domain read models created by `Room.Snapshot()`.

Purpose in current code:

```text
Provide lock-protected copies of mutable room/game/player state.
Serve as the broadcast target list.
Serve as the transport DTO source.
Avoid exposing room internals directly to JSON responses.
```

Snapshot flow:

```text
game.Room
 └─ Snapshot() -> game.RoomSnapshot
    └─ api/dto.FromRoomSnapshot(...) -> dto.RoomSnapshotDTO
       └─ json.Marshal(Event{Type, Payload})
          └─ Client.Send
             └─ WebSocket write
```

Snapshot categories:

```text
game.RoomSnapshot:
  domain-level read model / projection

api/dto.RoomSnapshotDTO:
  transport DTO

Event payload:
  serialized WebSocket message body
```

Snapshots are copied enough for current fields:

```text
players copied into new slice
TicTacToe board copied by value
Chess PGNMoves copied with append([]string(nil), ...)
chat history copied with append([]ChatMessage(nil), ...)
```

---

**10. Observability**

Observability exists in [internal/observability/observability.go](/home/tsaqiffatih/programming/mini-game/server-copy/internal/observability/observability.go).

Current observability components:

```text
JSON slog logger
global default logger with service name
HTTP request middleware
trace-id extraction from traceparent or X-Request-ID
context value storage for trace_id
StartSpan helper logging span start/end at debug level
structured fields:
  room_id
  player_id
  event_type
  game_type
  duration
  error
```

Logging is used across startup, shutdown, HTTP requests, WebSocket lifecycle, event receipt, move handling, chat handling, cleanup, room deletion, CORS config, and WebSocket write/read errors.

Trace readiness:

The code propagates request context into service methods and WebSocket connection setup. WebSocket read handling reuses the context captured from the upgrade request. There is no OpenTelemetry SDK/exporter in the code; `StartSpan` is a logging helper rather than a real distributed trace span.

Metrics readiness:

No metrics collector, counters, histograms, or exporter currently exist.

OpenTelemetry readiness:

The shape is partially trace-aware through context propagation and span-like helpers, but there is no actual OTel tracer/provider/exporter.

---

**11. Scaling Characteristics**

Rooms are process-local.

```text
MemoryRoomRepository
 └─ map[string]*game.Room in one Go process

ClientRegistry
 └─ map[string]*Client in one Go process

PlayerManager
 └─ map[string]*Player in one Go process
```

Horizontal scaling characteristics:

```text
A room exists only in the process that created it.
A WebSocket client exists only in the process it connected to.
Broadcasting only reaches clients connected to the same process.
```

Therefore, with multiple backend instances, current behavior would require routing a given room/player’s HTTP and WebSocket traffic to the same instance to see the same in-memory room and client state. There is no cross-process room store, distributed lock, external pub/sub, or shared WebSocket fanout.

Expected bottlenecks:

```text
Per-room mutex:
  hot rooms serialize all moves/chat/snapshot reads.
  Current game rooms max at 2 players, so this is naturally bounded.

Global repository mutex:
  all room lookups share the same repository map lock.
  Read locks allow concurrent reads, but cleanup/list/delete/save contend with writes.

Global client registry mutex:
  all broadcasts perform client lookups through one global registry.

WebSocket fanout:
  currently small because room players max at 2.
  If room player count expanded, broadcast is O(players).

Memory:
  grows with players, rooms, active connections, send queues, and chat histories.
  Chat per room is bounded to 50 messages.
  Send queue is bounded to 256 messages per client.

CPU:
  chess move validation is delegated to notnil/chess.
  TicTacToe move cost is constant.
  TicTacToe minimax AI exists but is not currently called by runtime move flow.

Network:
  one WebSocket per connected player.
  each broadcast serializes an event once, then enqueues to room players.
```

---

**12. Architectural Pattern Identification**

Patterns that actually exist:

```text
Service Layer
  GameService coordinates use cases between transport, repository, player manager, and rooms.

Repository Pattern
  service.RoomRepository interface with MemoryRoomRepository implementation.

Aggregate Root-like Room
  Room owns players, game state, room lifecycle, chat, and validates mutations.

DTO Mapping
  api/dto maps domain snapshots to transport shapes.

Command Handling
  WebSocket message type switch dispatches commands to handlers.

Shared-memory concurrency
  maps + mutexes + goroutines + channels.

Actor-like isolated state machine, partial
  Room serializes its own state transitions with a mutex,
  but does not own a goroutine/mailbox.

Projection/read-model snapshots
  RoomSnapshot and DTO snapshots are read-side representations of room state.

Transport/domain separation, partial
  api and dto are separate from service/game,
  but game.Room has a notifier callback wired to api broadcasting through service/main.
```

Patterns not present as implemented:

```text
No external event bus.
No distributed pub/sub.
No CQRS command/read storage split.
No persistent repository.
No explicit hub goroutine.
No room mailbox actor.
No spectator pattern.
No explicit reconnect protocol.
No explicit leave-room command handler.
```

---

**13. Runtime Ownership Map**

```text
WebSocket connections
  Owner: api.ClientRegistry + api.Client
  Writer: api.Client.WritePump

Outbound WebSocket writes
  Owner: per-client WritePump
  Input: Client.Send channel

Inbound WebSocket reads
  Owner: readMessages goroutine per connection

Client registration
  Owner: ClientRegistry global map keyed by playerID

Player global identity
  Owner: game.PlayerManager

Room storage
  Owner: infrastructure.MemoryRoomRepository

Room state
  Owner: game.Room

Game state
  Owner: game.Room owns either:
    - *tictactoe.TictactoeGameState
    - *chess.ChessGameState

Game mutation
  Owner: game.Room methods under room.mu

Snapshots
  Owner: game.Room creates game.RoomSnapshot
  Transport mapping: api/dto

Broadcast target selection
  Owner: api.NotifySnapshotToClients uses snapshot.Players

Broadcast transport delivery
  Owner: api.Client.Enqueue + Client.WritePump

Synchronization
  Room internals: room.mu
  Room map: repository.mu
  Client map: ClientRegistry.mu
  Player map: PlayerManager.mu
  Rate limiter map: middleware.mu
```

---

**14. Critical Files & Components**

Critical files:

| File | Why it matters |
|---|---|
| [main.go](/home/tsaqiffatih/programming/mini-game/server-copy/main.go) | Runtime composition, server lifecycle, cleanup goroutines, notifier wiring |
| [api/websocket.go](/home/tsaqiffatih/programming/mini-game/server-copy/api/websocket.go) | WebSocket lifecycle, connection validation, connect/disconnect flow |
| [api/client.go](/home/tsaqiffatih/programming/mini-game/server-copy/api/client.go) | Client registry, outbound queues, write pump, slow-client behavior |
| [api/message.go](/home/tsaqiffatih/programming/mini-game/server-copy/api/message.go) | WebSocket read dispatch, chat flow, broadcast helpers |
| [api/game_handler.go](/home/tsaqiffatih/programming/mini-game/server-copy/api/game_handler.go) | Chess/TicTacToe WebSocket move handlers |
| [api/handlers.go](/home/tsaqiffatih/programming/mini-game/server-copy/api/handlers.go) | HTTP API routes for users and rooms |
| [api/dto/game.go](/home/tsaqiffatih/programming/mini-game/server-copy/api/dto/game.go) | Transport DTOs and snapshot serialization shape |
| [service/game_service.go](/home/tsaqiffatih/programming/mini-game/server-copy/service/game_service.go) | Application orchestration and repository boundary |
| [game/room.go](/home/tsaqiffatih/programming/mini-game/server-copy/game/room.go) | Room aggregate state, snapshots, lifecycle transitions, scheduled reset |
| [game/room_actions.go](/home/tsaqiffatih/programming/mini-game/server-copy/game/room_actions.go) | Room mutations: join, move, disconnect, chat, AI enablement |
| [game/player.go](/home/tsaqiffatih/programming/mini-game/server-copy/game/player.go) | Global player registry and session status |
| [game/chat.go](/home/tsaqiffatih/programming/mini-game/server-copy/game/chat.go) | Chat validation and message ID generation |
| [chess/chess_state.go](/home/tsaqiffatih/programming/mini-game/server-copy/chess/chess_state.go) | Chess engine wrapper and move validation |
| [tictactoe/tictactoe_state.go](/home/tsaqiffatih/programming/mini-game/server-copy/tictactoe/tictactoe_state.go) | TicTacToe rules and board mutation |
| [infrastructure/memory_room_repository.go](/home/tsaqiffatih/programming/mini-game/server-copy/infrastructure/memory_room_repository.go) | Process-local room persistence |
| [internal/observability/observability.go](/home/tsaqiffatih/programming/mini-game/server-copy/internal/observability/observability.go) | Logging, request middleware, trace-id helpers |

Most important structs/interfaces:

```text
api.Client
api.ClientRegistry
service.GameService
service.RoomRepository
game.Room
game.PlayerManager
game.Player
game.RoomSnapshot
game.TicTacToeMoveResult
game.ChessMoveResult
tictactoe.TictactoeGameState
chess.ChessGameState
infrastructure.MemoryRoomRepository
```

---

**15. Final Architecture Summary**

This backend is a process-local realtime game server. HTTP APIs create users and rooms; WebSocket connections attach players to existing rooms; game commands arrive over WebSocket; the service layer resolves the room; the room serializes mutation under its mutex; snapshots are generated from authoritative room state; API broadcast helpers enqueue JSON events to each connected player’s write pump.

High-level runtime shape:

```text
HTTP / WebSocket transport
        ↓
api handlers / WS dispatch
        ↓
GameService orchestration
        ↓
MemoryRoomRepository -> *game.Room
        ↓
Room mutex-protected domain mutation
        ↓
Chess/TicTacToe engine
        ↓
RoomSnapshot
        ↓
DTO/Event JSON
        ↓
ClientRegistry -> Client.Send -> WritePump -> WebSocket
```

Current strengths:

```text
Clear central room authority.
Move mutation is centralized through room methods.
Per-room locking gives straightforward race prevention.
WebSocket writes are funneled through one writer goroutine per client.
Snapshots provide a clean read model for transport serialization.
Chat history and send queues are bounded.
Shutdown closes clients and cleans rooms.
```

Current maturity level:

The backend is a compact, in-memory, single-process realtime architecture. It is suitable for local or single-instance deployment semantics. The architecture has clear domain boundaries around rooms and game engines, but runtime state, WebSocket clients, and broadcasts are not distributed.

Overall backend style:

```text
Go service-layer architecture
with aggregate-like room domain objects,
in-memory repository,
direct WebSocket fanout,
mutex-based concurrency,
and snapshot DTO broadcasting.
```

Likely architectural direction implied by the current code:

The code is structured around preserving room authority and emitting snapshots after mutations. If it grows, the natural direction is to keep `Room` as the state owner while changing storage/fanout mechanics around it. As implemented today, though, all authoritative room state and realtime delivery are local to one Go process.