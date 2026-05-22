# Mini Game Project Checklist

Checklist gabungan dari:

* WebSocket lifecycle issues
* Roadmap
* Frontend architecture
* Backend architecture
* Deployment plan
* Chess features

---

# Priority Legend

* [ ] Critical → bug/architecture issue yang paling berdampak
* [ ] High → penting untuk stabilitas & portfolio quality
* [ ] Medium → improvement/polish
* [ ] Low → cleanup/refactor kecil

---

# Phase 1 — Stabilize Current Gameplay

## WebSocket Lifecycle

### Refresh/Reconnect Stability

* [✓] Pisahkan transient disconnect vs permanent leave
* [✓] Tambahkan event `player_disconnected`
* [✓] Broadcast `player_left` hanya setelah grace period habis
* [✓] Tambahkan reconnecting UI state di frontend
* [ ] Tambahkan socket states:
  * [ ] connecting
  * [ ] open
  * [ ] reconnecting
  * [ ] rejected
  * [ ] expired
* [✓] Hindari auto reload saat websocket error generic
* [✓] Gunakan `onClose` + close code aware handling
* [✓] Jangan clear localStorage pada generic websocket error
* [ ] Tambahkan room expired UI
* [ ] Tambahkan room recovery flow
* [✓] Tambahkan log websocket close code/reason
* [✓] Tambahkan log reconnect success/failure

### Delayed Disconnect Handling

* [ ] Audit `RemovePlayerAfterDelay`
* [ ] Tambahkan log skip removal karena reconnect
* [✓] Pastikan reconnect tidak memicu false leave alert
* [ ] Pastikan reconnect room-aware
* [ ] Tambahkan connection/session ID

### Client Registry

* [ ] Refactor registry dari:

  * [ ] `map[playerID]*Client`
* [ ] Menjadi:

  * [ ] `(roomID, playerID)` scoped registry
* [ ] Tambahkan generated connection ID
* [ ] Tambahkan registry logging
* [ ] Test multi-tab same player
* [ ] Test multi-room same player

### Stale Room State

* [✓] Ganti localStorage global `roomId`,`playerMark` Menjadi:
  * [✓] `chess.roomId`
  * [✓] `tictactoe.roomId`
* [ ] Simpan `gameType`
* [ ] Tambahkan room validation sebelum render board
* [ ] Tambahkan room recovery endpoint
* [ ] Buat `RoomSessionStore`

### Snapshot Consistency

* [ ] Tambahkan stale snapshot rejection ke TicTacToe
* [ ] Shared state_version handling
* [ ] Buat centralized snapshot reducer
* [ ] Tambahkan delayed websocket message testing

### Rate Limiter

* [ ] Audit websocket upgrade vs rate limiter
* [ ] Exempt `/ws` dari limiter HTTP biasa
* [ ] Tambahkan websocket-specific limits
* [ ] Tambahkan logging 429 websocket
* [ ] Gunakan trusted proxy IP handling

---

# Phase 2 — Chess UX & Gameplay Polish

## Chess Result & Status UI

* [ ] Tambahkan game result banner
* [ ] Tampilkan:

  * [ ] winner
  * [ ] checkmate
  * [ ] stalemate
  * [ ] draw reason
* [ ] Tambahkan finished state overlay/modal

## Promotion UI

* [ ] Hapus auto queen
* [ ] Tambahkan promotion modal
* [ ] Support:

  * [ ] queen
  * [ ] rook
  * [ ] bishop
  * [ ] knight
* [ ] Tambahkan cancel/reset behavior

## AI UX

* [ ] Tambahkan AI thinking indicator
* [ ] Tambahkan AI avatar
* [ ] Tambahkan AI identity/name
* [ ] Tambahkan AI difficulty display
* [ ] Tambahkan AI room metadata UI
* [ ] Tambahkan AI move error handling UI

## Undo System

* [ ] Tambahkan undo button vs AI
* [ ] Disable undo saat AI thinking
* [ ] Handle `CHESS_UNDO_REQUEST`
* [ ] Tambahkan undo tests

## Board Polish

* [ ] Checkmate highlight
* [ ] Highlight checked king
* [ ] Custom premove style
* [ ] Engine suggestion arrows
* [ ] Analysis/assist mode
* [ ] Improve move animations

## Sounds

* [ ] Audit sound filenames/casing
* [ ] Remove duplicate sound triggers
* [ ] Verify cross-platform asset loading

---

# Phase 3 — Frontend Architecture Refactor

## Feature-Based Structure

* [ ] Buat folder:

```text
src/
  features/
    chess/
    tictactoe/
    room/
    chat/
    player/
    lobby/
  shared/
    websocket/
    api/
    ui/
    alerts/
    sounds/
```

## ChessBoard Refactor

* [ ] Pecah `ChessBoard.tsx`
* [ ] Extract:

  * [ ] useChessRealtime
  * [ ] useChessBoardController
  * [ ] useChessSnapshotReducer
  * [ ] useChessMoveSender
  * [ ] ChessPlayerPanel
  * [ ] ChessResultBanner
  * [ ] ChessPromotionDialog
  * [ ] ChessCapturedPieces
  * [ ] ChessChatPanel

## Shared Room/WebSocket Layer

* [ ] Buat `useRoomSocket`
* [ ] Buat `useRoomSession`
* [ ] Buat websocket transport layer reusable
* [ ] Pisahkan transport vs business logic

## DTO Typing

* [ ] Hapus implicit `any`
* [ ] Tambahkan typed room snapshot DTO
* [ ] Tambahkan typed websocket event DTO
* [ ] Shared event constants
* [ ] Tambahkan frontend contract typing

## Chat Refactor

* [ ] Pisahkan chat dari game board
* [ ] Shared chat feature
* [ ] Shared websocket chat handling

## State Ownership

* [ ] Centralize room state
* [ ] Hindari direct localStorage access di pages/components
* [ ] Tambahkan reducer/store pattern sederhana

---

# Phase 4 — Backend Architecture Improvements

## Room Lifecycle

* [ ] Tambahkan room lifecycle logging
* [ ] Tambahkan explicit room expiration policy
* [ ] Tambahkan room recovery endpoint
* [ ] Tambahkan room health metrics

## WebSocket Infrastructure

* [ ] Restrict websocket origins
* [ ] Tambahkan connection/session abstraction
* [ ] Tambahkan room-scoped presence
* [ ] Pisahkan websocket package

## Stockfish Architecture

* [ ] Buat `ChessEngine` interface
* [ ] Pindahkan Stockfish keluar dari domain package
* [ ] Tambahkan AI timeout handling
* [ ] Tambahkan AI metrics
* [ ] Tambahkan fake engine untuk testing

## AI State Consistency

* [ ] Notify clients saat AI thinking berubah
* [ ] Tambahkan `ai_error` metadata/event
* [ ] Tambahkan AI failure tests

## Room/Game Abstraction

* [ ] Definisikan common game module interface
* [ ] Standardize:

  * [ ] ApplyMove
  * [ ] Snapshot
  * [ ] Reset
  * [ ] Status
  * [ ] MaybeScheduleAI

---

# Phase 5 — Persistence & Scalability

## Persistence

* [ ] Tambahkan Postgres
* [ ] Persist:

  * [ ] player/session
  * [ ] room metadata
  * [ ] completed games
  * [ ] optional chat history
* [ ] Tambahkan migrations
* [ ] Tambahkan room expiration persistence

## Realtime Scalability

* [ ] Tambahkan Redis presence/pubsub
* [ ] Tambahkan sticky sessions
* [ ] Tambahkan room ownership model
* [ ] Tambahkan distributed presence metadata
* [ ] Tambahkan graceful room draining

## Horizontal Scaling

* [ ] Externalize room state
* [ ] Externalize websocket fanout
* [ ] Tambahkan distributed locks/room actors
* [ ] Tambahkan websocket gateway layer
* [ ] Tambahkan AI worker pool

---

# Deployment Checklist

## Frontend Deployment

* [ ] Deploy frontend
* [ ] Verify:

  * [ ] `/`
  * [ ] `/chess`
  * [ ] `/tictactoe`
* [ ] Verify WSS connection
* [ ] Verify assets load correctly
* [ ] Setup production env vars

## Backend Deployment

* [ ] Dockerize backend
* [ ] Verify Stockfish inside container
* [ ] Setup reverse proxy
* [ ] Setup SSL
* [ ] Restrict CORS origins
* [ ] Add health endpoint
* [ ] Add Docker healthcheck

## Environment Setup

* [ ] Buat `.env.example`
* [ ] Jangan commit `.env`
* [ ] Document env vars

## Monitoring

* [ ] Log websocket lifecycle
* [ ] Log room lifecycle
* [ ] Log AI failures
* [ ] Log active rooms
* [ ] Log active clients
* [ ] Tambahkan uptime monitoring
* [ ] Tambahkan metrics endpoint

## CI/CD

### Backend

* [ ] `go test ./...`
* [ ] `go build ./...`
* [ ] Docker build pipeline

### Frontend

* [ ] `npm ci`
* [ ] `npm run build`
* [ ] Setup linting

### Deployment Automation

* [ ] GitHub Actions
* [ ] Auto deploy frontend
* [ ] SSH deploy VPS

---

# Cleanup & Technical Debt

## Frontend Cleanup

* [ ] Remove stray console logs
* [ ] Remove dead components
* [ ] Improve responsive layout
* [ ] Improve accessibility labels
* [ ] Consolidate alert helpers

## Backend Cleanup

* [ ] Cleanup legacy comments
* [ ] Cleanup temp/log files
* [ ] Review old architecture notes
* [ ] Update API docs

## Repository Cleanup

* [ ] Add proper README
* [ ] Mark legacy folders clearly
* [ ] Audit `.gitignore`
* [ ] Add docs folder structure

---

# Additional Architecture Plan Tasks

## Monorepo & Documentation

* [ ] Pastikan `server/` dianggap active backend
* [ ] Pastikan `archive/old-server/` hanya historical reference
* [ ] Rapikan struktur `/docs`
* [ ] Dokumentasikan active runtime flow
* [ ] Dokumentasikan websocket event flow
* [ ] Dokumentasikan authoritative ownership boundaries

## Frontend Runtime Improvements

* [ ] Audit seluruh penggunaan `localStorage`
* [ ] Hindari route-level business logic berlebihan
* [ ] Refactor pages agar hanya compose features
* [ ] Audit websocket reconnect semantics
* [ ] Pisahkan websocket transport dari room lifecycle logic
* [ ] Tambahkan explicit room bootstrap flow

## Backend Runtime Improvements

* [ ] Tambahkan explicit websocket lifecycle documentation
* [ ] Pisahkan websocket registry vs websocket dispatcher
* [ ] Audit shutdown cleanup flow
* [ ] Tambahkan graceful websocket disconnect handling
* [ ] Tambahkan health/readiness endpoints
* [ ] Tambahkan room lifecycle metrics

## Ownership & Contracts

* [ ] Dokumentasikan snapshot contract versioning
* [ ] Standardize websocket event schema
* [ ] Buat typed transport contracts frontend/backend
* [ ] Audit deprecated DTO compatibility (`state.game?.chess ?? state.chess`)
* [ ] Plan deprecation removal untuk legacy DTO shape

## AI Architecture

* [ ] Audit Stockfish process lifecycle
* [ ] Tambahkan timeout/cancellation monitoring
* [ ] Tambahkan AI queue/process metrics
* [ ] Persiapkan future Stockfish worker pool

## Production Hardening

* [ ] Restrict websocket CORS origin
* [ ] Audit reverse proxy websocket handling
* [ ] Tambahkan graceful restart strategy
* [ ] Tambahkan structured websocket metrics/logging
* [ ] Audit cleanup ticker behavior

# Suggested Execution Order

## First Priority

* [ ] Websocket reconnect lifecycle
* [ ] Room recovery handling
* [ ] LocalStorage per-game session
* [ ] Socket state UI
* [ ] Client registry room-scope

## Second Priority

* [ ] Promotion UI
* [ ] Result banner
* [ ] AI thinking UI
* [ ] DTO typing
* [ ] ChessBoard refactor

## Third Priority

* [ ] Deployment production setup
* [ ] Monitoring/logging
* [ ] Persistence layer
* [ ] Redis/pubsub exploration

## Fourth Priority

* [ ] Matchmaking
* [ ] Spectators
* [ ] Analysis mode
* [ ] Horizontal scaling architecture
