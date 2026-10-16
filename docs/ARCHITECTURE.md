Used tool: search_web

## Extensibility & Scalability Engineering Guide  
*Real‑Time Multi‑Game Platform (Next.js + Go + Caddy + Docker)*  

---  

### 1️⃣ Extensibility Guide – Adding a New Turn‑Based Game (e.g., **Connect Four**)

| Layer | What to do | Where (file / directory) | Key Concepts |
|-------|------------|--------------------------|--------------|
| **Backend – Core Engine** | Define a generic `GameEngine` interface that all games implement. | `server/game/engine.go` (new file) | `Init(roomID)`, `ApplyMove(move) (error)`, `State() GameStateDTO`, `IsGameOver() bool`, `AvailableMoves() []Move` |
| **Backend – Connect Four Implementation** | Implement the interface in a dedicated package. | `server/game/connectfour/engine.go` (new package) | Board as 6 × 7 int matrix, move validation, win detection, optional AI stub. |
| **Backend – Service Layer** | Add thin wrappers that expose the engine through the existing `GameService`. | `server/service/game_service.go` (add methods) | `CreateConnectFourRoom(ctx, req)`, `HandleConnectFourMoveWithContext(ctx, req)`, `RecoverConnectFourRoom(...)` – mirror the pattern of TicTacToe/Chess. |
| **Backend – DTOs** | Extend the public DTOs to expose Connect Four state to the client. | `server/api/dto/game.go` | ```go\ntype ConnectFourStateDTO struct { Board [][]int `json:\"board\"`; CurrentPlayer string `json:\"current_player\"`; Winner *string `json:\"winner,omitempty\"` }\n\n// add to GameStateDTO\ntype GameStateDTO struct {\n    Type string `json:\"type\"` // \"chess\", \"tictactoe\", \"connectfour\"\n    Chess *ChessStateDTO `json:\"chess,omitempty\"`\n    TicTacToe *TicTacToeStateDTO `json:\"tictactoe,omitempty\"`\n    ConnectFour *ConnectFourStateDTO `json:\"connectfour,omitempty\"`\n}\n``` |
| **Backend – WebSocket Message Types** | Introduce a new action constant and a processor. | `server/api/message.go` | Add `CONNECTFOUR_MOVE` constant; extend `handleMessageAction` switch with `processConnectFourMove`. |
| **Backend – Router** | (Optional) expose a REST endpoint for creating a Connect Four room. | `server/api/handlers.go` (RegisterRouter) | `POST /room/create/connectfour` → `createConnectFourRoom`. |
| **Frontend – TypeScript Types** | Mirror the DTOs so the UI can type‑check incoming snapshots. | `frontend/src/types/game.ts` (or similar) | ```ts\nexport interface ConnectFourStateDTO { board: number[][]; currentPlayer: string; winner?: string; }\nexport type GameStateDTO = { type: 'chess' | 'tictactoe' | 'connectfour'; chess?: ChessStateDTO; tictactoe?: TicTacToeStateDTO; connectfour?: ConnectFourStateDTO; };\n``` |
| **Frontend – UI Component** | Build a reusable board component. | `frontend/src/components/ConnectFourBoard.tsx` | Render a 7 × 6 grid, handle clicks → dispatch `CONNECTFOUR_MOVE` via the shared WebSocket client. |
| **Frontend – Redux / Context** | Add action creators, reducers, and selectors for Connect Four. | `frontend/src/store/gameSlice.ts` (or context file) | `connectFourMove(payload)`, `setConnectFourState(state)`. |
| **Frontend – Routing / Lobby** | Add a lobby entry and navigation to the new game. | `frontend/src/pages/lobby.tsx` (or component) | New button “Play Connect Four”, routes to `/game/connectfour/[roomId]`. |
| **Frontend – WebSocket Hook** | Extend the generic `useGameWebSocket` hook to listen for `CONNECTFOUR_MOVE` and `game_update` events. | `frontend/src/hooks/useGameWebSocket.ts` | Add case handling for `CONNECTFOUR_MOVE` → `dispatch(connectFourMove(...))`. |
| **Docker / CI** | No code changes needed, just ensure the new Go package is compiled (`go.mod` auto‑adds). | `docker-compose.yml` (unchanged) | The binary will automatically include the new package. |

#### Step‑by‑Step Walk‑through (Backend)

1. **Create the engine interface**  
   ```go
   // server/game/engine.go
   package game

   // Move is a generic payload; concrete games define their own struct that satisfies json.Marshaler.
   type Move interface{}

// GameEngine is the contract every game must fulfil.
type GameEngine interface {
    // Init prepares a new room (board, turn, etc.)
    Init(roomID string) error

    // ApplyMove validates and applies a player move.
    ApplyMove(move Move) error

    // State returns a JSON‑serialisable snapshot for the client.
    State() interface{} // will be cast to the appropriate DTO

    // IsGameOver reports if the game has finished.
    IsGameOver() bool

    // AvailableMoves returns the list of legal moves (optional, used for AI/help).
    AvailableMoves() []Move
}
   ```
2. **Implement Connect Four**  
   - `engine.go` in `server/game/connectfour` implements `GameEngine`.  
   - Use a 2‑D slice `board[6][7]int` (0 = empty, 1 = player 1, 2 = player 2).  
   - `ApplyMove` drops a disc in the chosen column, checks vertical/horizontal/diagonal four‑in‑a‑row, updates `winner` state.  
   - `State` returns a `ConnectFourStateDTO`.  

3. **Register the engine**  
   In `service.GameService` add a map `engines map[string]game.GameEngine` (`roomID → engine`).  
   - On room creation (`CreateConnectFourRoom`) instantiate `connectfour.NewEngine(roomID)` and store it.  
   - On each move handler (`HandleConnectFourMoveWithContext`) look up the engine, call `ApplyMove`, then broadcast the new snapshot using the existing `NotifySnapshotToClients`.  

4. **Expose WebSocket Action**  
   ```go
   // server/api/message.go
   const CONNECTFOUR_MOVE = "CONNECTFOUR_MOVE"
   ```
   Add a new case in `handleMessageAction`:
   ```go
   case CONNECTFOUR_MOVE:
       processConnectFourMove(s, c, m)
   ```

5. **Add DTO** (see table above).  
   Update `FromRoomSnapshot` to fill `GameStateDTO.ConnectFour` when the room’s game type is `"connectfour"`.

#### Step‑by‑Step Walk‑through (Frontend)

1. **Define Types** – add `ConnectFourStateDTO` and extend the union `GameStateDTO`.  
2. **Create UI** – a simple grid of clickable cells; on click, emit:
   ```ts
   ws.send(JSON.stringify({ type: 'CONNECTFOUR_MOVE', payload: { column: col } }))
   ```
3. **Hook into WebSocket** – extend the message dispatcher:
   ```ts
   case 'CONNECTFOUR_MOVE':
       dispatch(gameSlice.actions.connectFourMove(payload))
   case 'GAME_UPDATE':
       if (msg.gameState.type === 'connectfour') {
           dispatch(gameSlice.actions.setConnectFourState(msg.gameState.connectfour))
       }
   ```
4. **Add Lobby / Routing** – a new button/link routes to `/game/connectfour/[roomId]`.  
5. **Testing** – run `docker compose up --build` and verify that creating a Connect Four room shows the board, moves propagate in real time, and the UI flips to “PLAYING” once the second player joins.

---

### 2️⃣ Game‑Complexity## Extensibility & Scalability Engineering Guide  
*Real‑Time Multi‑Game Platform (Next.js + Go + Caddy + Docker)*  

---  

### 1️⃣ Extensibility Guide – Adding a New Turn‑Based Game (e.g., **Connect Four**)

| Layer | What to do | Where (file / directory) | Key Concepts |
|-------|------------|--------------------------|--------------|
| **Backend – Core Engine** | Define a generic `GameEngine` interface that all games implement. | `server/game/engine.go` (new file) | `Init(roomID)`, `ApplyMove(move) (error)`, `State() GameStateDTO`, `IsGameOver() bool`, `AvailableMoves() []Move` |
| **Backend – Connect Four Implementation** | Implement the interface in a dedicated package. | `server/game/connectfour/engine.go` (new package) | Board as 6 × 7 int matrix, move validation, win detection, optional AI stub. |
| **Backend – Service Layer** | Add thin wrappers that expose the engine through the existing `GameService`. | `server/service/game_service.go` (add methods) | `CreateConnectFourRoom(ctx, req)`, `HandleConnectFourMoveWithContext(ctx, req)`, `RecoverConnectFourRoom(...)` – mirror the pattern of TicTacToe/Chess. |
| **Backend – DTOs** | Extend the public DTOs to expose Connect Four state to the client. | `server/api/dto/game.go` | ```go\ntype ConnectFourStateDTO struct { Board [][]int `json:\"board\"`; CurrentPlayer string `json:\"current_player\"`; Winner *string `json:\"winner,omitempty\"` }\n\n// add to GameStateDTO\ntype GameStateDTO struct {\n    Type string `json:\"type\"` // \"chess\", \"tictactoe\", \"connectfour\"\n    Chess *ChessStateDTO `json:\"chess,omitempty\"`\n    TicTacToe *TicTacToeStateDTO `json:\"tictactoe,omitempty\"`\n    ConnectFour *ConnectFourStateDTO `json:\"connectfour,omitempty\"`\n}\n``` |
| **Backend – WebSocket Message Types** | Introduce a new action constant and a processor. | `server/api/message.go` | Add `CONNECTFOUR_MOVE` constant; extend `handleMessageAction` switch with `processConnectFourMove`. |
| **Backend – Router** | (Optional) expose a REST endpoint for creating a Connect Four room. | `server/api/handlers.go` (RegisterRouter) | `POST /room/create/connectfour` → `createConnectFourRoom`. |
| **Frontend – TypeScript Types** | Mirror the DTOs so the UI can type‑check incoming snapshots. | `frontend/src/types/game.ts` (or similar) | ```ts\nexport interface ConnectFourStateDTO { board: number[][]; currentPlayer: string; winner?: string; }\nexport type GameStateDTO = { type: 'chess' | 'tictactoe' | 'connectfour'; chess?: ChessStateDTO; tictactoe?: TicTacToeStateDTO; connectfour?: ConnectFourStateDTO; };\n``` |
| **Frontend – UI Component** | Build a reusable board component. | `frontend/src/components/ConnectFourBoard.tsx` | Render a 7 × 6 grid, handle clicks → dispatch `CONNECTFOUR_MOVE` via the shared WebSocket client. |
| **Frontend – Redux / Context** | Add action creators, reducers, and selectors for Connect Four. | `frontend/src/store/gameSlice.ts` (or context file) | `connectFourMove(payload)`, `setConnectFourState(state)`. |
| **Frontend – Routing / Lobby** | Add a lobby entry and navigation to the new game. | `frontend/src/pages/lobby.tsx` (or component) | New button “Play Connect Four”, routes to `/game/connectfour/[roomId]`. |
| **Frontend – WebSocket Hook** | Extend the generic `useGameWebSocket` hook to listen for `CONNECTFOUR_MOVE` and `game_update` events. | `frontend/src/hooks/useGameWebSocket.ts` | Add case handling for `CONNECTFOUR_MOVE` → `dispatch(connectFourMove(...))`. |
| **Docker / CI** | No code changes needed, just ensure the new Go package is compiled (`go.mod` auto‑adds). | `docker-compose.yml` (unchanged) | The binary will automatically include the new package. |

#### Step‑by‑Step Walk‑through (Backend)

1. **Create the engine interface**  
   ```go
   // server/game/engine.go
   package game

   // Move is a generic payload; concrete games define their own struct that satisfies json.Marshaler.
   type Move interface{}

// GameEngine is the contract every game must fulfil.
type GameEngine interface {
    // Init prepares a new room (board, turn, etc.)
    Init(roomID string) error

    // ApplyMove validates and applies a player move.
    ApplyMove(move Move) error

    // State returns a JSON‑serialisable snapshot for the client.
    State() interface{} // will be cast to the appropriate DTO

    // IsGameOver reports if the game has finished.
    IsGameOver() bool

    // AvailableMoves returns the list of legal moves (optional, used for AI/help).
    AvailableMoves() []Move
}
   ```
2. **Implement Connect Four**  
   - `engine.go` in `server/game/connectfour` implements `GameEngine`.  
   - Use a 2‑D slice `board[6][7]int` (0 = empty, 1 = player 1, 2 = player 2).  
   - `ApplyMove` drops a disc in the chosen column, checks vertical/horizontal/diagonal four‑in‑a‑row, updates `winner` state.  
   - `State` returns a `ConnectFourStateDTO`.  

3. **Register the engine**  
   In `service.GameService` add a map `engines map[string]game.GameEngine` (`roomID → engine`).  
   - On room creation (`CreateConnectFourRoom`) instantiate `connectfour.NewEngine(roomID)` and store it.  
   - On each move handler (`HandleConnectFourMoveWithContext`) look up the engine, call `ApplyMove`, then broadcast the new snapshot using the existing `NotifySnapshotToClients`.  

4. **Expose WebSocket Action**  
   ```go
   // server/api/message.go
   const CONNECTFOUR_MOVE = "CONNECTFOUR_MOVE"
   ```
   Add a new case in `handleMessageAction`:
   ```go
   case CONNECTFOUR_MOVE:
       processConnectFourMove(s, c, m)
   ```

5. **Add DTO** (see table above).  
   Update `FromRoomSnapshot` to fill `GameStateDTO.ConnectFour` when the room’s game type is `"connectfour"`.

#### Step‑by‑Step Walk‑through (Frontend)

1. **Define Types** – add `ConnectFourStateDTO` and extend the union `GameStateDTO`.  
2. **Create UI** – a simple grid of clickable cells; on click, emit:
   ```ts
   ws.send(JSON.stringify({ type: 'CONNECTFOUR_MOVE', payload: { column: col } }))
   ```
3. **Hook into WebSocket** – extend the message dispatcher:
   ```ts
   case 'CONNECTFOUR_MOVE':
       dispatch(gameSlice.actions.connectFourMove(payload))
   case 'GAME_UPDATE':
       if (msg.gameState.type === 'connectfour') {
           dispatch(gameSlice.actions.setConnectFourState(msg.gameState.connectfour))
       }
   ```
4. **Add Lobby / Routing** – a new button/link routes to `/game/connectfour/[roomId]`.  
5. **Testing** – run `docker compose up --build` and verify that creating a Connect Four room shows the board, moves propagate in real time, and the UI flips to “PLAYING” once the second player joins.

---

### 2️⃣ Game‑Complexity Analysis  

| Complexity | Example Games | Why It Fits This Architecture |
|------------|----------------|--------------------------------|
| **Easy** | Connect Four, Gomoku, Othello, Checkers (basic), Tic‑Tac‑Toe (already present) | Simple deterministic board, turn‑based, low‑CPU AI (minimax depth ≤ 3). Minimal extra state, easy to express as a `GameEngine`. |
| **Medium** | Chess (already present), Ludo, Backgammon, Monopoly | Larger state spaces, richer rule sets, optional dice/randomness or deeper AI (e.g., Stockfish). Still turn‑based, but may need more sophisticated move generation and time‑bounded AI. |
| **Hard / Out‑of‑Scope** | Real‑Time Strategy (StarCraft), First‑Person Shooter, MOBA, Racing games | Require high‑frequency state sync (30‑60 Hz), physics simulation, large number of entities, latency‑critical networking → not suitable for the current single‑process WebSocket manager. Would need authoritative server tick loops, UDP, and massive scaling infra. |

*Bottom line*: The platform comfortably supports any **deterministic, turn‑based** game with modest AI requirements. Adding a new game is essentially a matter of implementing `GameEngine` and wiring DTOs.

---

### 3️⃣ Scalability Roadmap  

#### 3.1 Vertical Scaling (single instance)  
| Concern | Current State | Scaling Tips |
|---------|---------------|--------------|
| **CPU** – Stockfish AI is the most expensive component. | Runs synchronously inside the Go process, using the default number of threads (`stockfish -threads 1`). | Pass `-threads N` and `-hash <size>` via environment variables (`STOCKFISH_THREADS`, `STOCKFISH_HASH`). Tune `GOMAXPROCS` to match physical cores (`export GOMAXPROCS=$(nproc)`). |
| **Memory** – Large board histories, many concurrent rooms. | Each room holds a snapshot (`RoomSnapshot`) and a per‑room engine instance. | Increase container memory limits (`mem_limit` in `docker-compose.yml`). Enable Go GC tuning (`GODEBUG=gctrace=1` for diagnostics). |
| **File Descriptors** – WebSocket connections + DB / Redis handles. | Default Docker limits (1024). | Raise `ulimit -n` in the compose file (`ulimits: nofile: soft: 4096 hard: 8192`). |
| **Profiling** – Identify hot paths. | No profiler attached. | Add `pprof` endpoint (`/debug/pprof`) behind a protected route; use `go tool pprof` to visualise CPU/memory hot spots. |

#### 3.2 Horizontal Scaling (multiple instances)  

| Goal | Recommended Approach |
|------|-----------------------|
| **Stateless HTTP** | Keep all HTTP endpoints stateless; they already just forward to the backend service. Deploy multiple Go containers behind Caddy (or an external load balancer). |
| **WebSocket Broadcast Across Instances** | Replace the in‑process `BroadcastToRoom` with a **pub/sub** system (Redis, NATS, or Google Cloud Pub/Sub). <br>*Implementation sketch*: <br>1. Each instance publishes a `room:<id>:event` message when it wants to push a snapshot. <br>2. All instances subscribe to `room:*` and forward received events to their local `ClientRegistry`. |
| **Session / Room Persistence** | Store room state in an external store (Redis or Cloud SQL) instead of in‑memory map. On server start, load missing rooms on‑demand. This makes any instance able to serve any player. |
| **Sticky Sessions (optional)** | If you keep in‑memory rooms, configure Caddy to **sticky‑session** based on `room_id` so a client always reaches the same backend instance. |
| **Scaling the AI** | Offload Stockfish to a dedicated service (Docker container per core) and communicate via gRPC. Then the Go backend becomes thin and can be horizontally scaled without CPU contention. |
| **Observability** | Export Prometheus metrics (`/metrics`) for active connections, rooms, CPU usage; Grafanaana dashboards for capacity planning. |

**Migration Path**  
1. Add Redis to `docker-compose.yml`.  
2. Refactor `ClientRegistry` to use a `pubsub.Client` for broadcasting.  
3. Move `GameService.rooms` map to a Redis hash (or persist snapshots in a DB).  
4. Deploy two Go containers and verify that a player can reconnect to a different instance after a restart.  

---

### 4️⃣ v2.0.0 Feature Roadmap (Non‑Game Enhancements)

| Category | Feature | Rationale / Benefit |
|----------|---------|----------------------|
| **Auth & Identity** | JWT‑based authentication (with refresh tokens). | Secure, stateless auth for API & WebSocket; easy integration with third‑party SSO. |
| **Player Profiles** | Persistent user accounts, Elo/MMR rating, win‑loss stats. | Enables skill‑based matchmaking and leaderboards. |
| **Matchmaking Service** | Queue system that pairs players of similar rating, optionally with auto‑requeue. | Improves user experience, reduces wait times. |
| **Spectator Mode** | Read‑only WebSocket client that receives `game_update` events without sending moves. | Allows broadcasting of tournaments and replays. |
| **Analytics Dashboard** | Real‑time metrics (active rooms, latency, AI load) + historical reports. | Ops visibility, capacity planning, SLA monitoring. |
| **Internationalisation (i18n)** | Next.js i18n routing, locale JSON files, dynamic language switch. | Wider audience reach. |
| **Server‑Side Event Store** | Append‑only log of game events (Kafka/Google Pub/Sub) for replay, auditing, and cheat detection. | Guarantees deterministic replays, compliance. |
| **CI/CD Enhancements** | GitHub Actions workflow that builds, tests, and pushes Docker images to a registry; automated semantic version bump. | Faster, reliable releases. |
| **Testing Suite** | End‑to‑end integration tests using Playwright for UI and `go test` with `testcontainers` for the backend. | Prevent regressions when adding new games. |
| **Config‑Driven Game Registry** | JSON/YAML file that declares supported games, their WebSocket action names, and engine factories. Allows adding a new game by merely dropping a plugin package. | Reduces code churn, encourages modular plugins. |

---

## 📌 Quick Recap

1. **Add a game** → implement `GameEngine`, expose DTO & WS action, plug into `GameService` and UI.  
2. **Complexity** – the platform shines for deterministic turn‑based games; anything requiring high‑frequency sync is out‑of‑scope.  
3. **Scale vertically** by tuning Stockfish and Go runtime; **scale horizontally** by moving room state & WS broadcast to Redis/NATS and adding sticky routing.  
4. **v2.0.0** will focus on auth, rating, matchmaking, spectators, observability, i18n, and a plugin‑friendly architecture.

Feel free to ask for deeper details on any of the sections (e.g., sample code for the Redis pub/sub integration or the exact `GameEngine` method signatures). Happy extending!