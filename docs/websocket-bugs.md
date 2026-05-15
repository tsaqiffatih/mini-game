# WebSocket Lifecycle Risks and Bugs

This document lists probable websocket lifecycle issues based on current code. It does not claim that every item is a confirmed production bug; each item is tied to observed implementation paths.

## System Overview

Frontend websocket owner:

- `mini-game-client/src/utils/gameWebsocket.ts`

Backend websocket owners:

- `server/api/websocket.go`
- `server/api/client.go`
- `server/api/message.go`

State owners:

- `server/game/room.go`
- `server/game/room_actions.go`
- `server/infrastructure/memory_room_repository.go`

## Issue: Room Disappearing After Refresh or Leave

Suspected root causes:

- Rooms are in memory only.
- Empty rooms are deleted after delayed disconnect removal.
- Server restart deletes all rooms.
- If the only player disconnects and does not reconnect within 30 seconds, `RemovePlayerAfterDelay` can remove the player and delete the room.
- Frontend keeps stale `roomId` in `localStorage` until it sees an error.

Files involved:

- `server/service/game_service.go`
- `server/game/room_actions.go`
- `server/infrastructure/memory_room_repository.go`
- `mini-game-client/src/app/chess/page.tsx`
- `mini-game-client/src/app/tictactoe/page.tsx`
- `mini-game-client/src/utils/gameWebsocket.ts`

Probable lifecycle path:

```text
player has localStorage.roomId
  -> websocket closes
  -> backend schedules RemovePlayerAfterDelay(30s)
  -> no reconnect occurs
  -> Room.HandlePlayerDisconnected removes player
  -> room is empty
  -> repository Delete(roomID)
  -> later frontend reloads with stale roomId
  -> /ws validation fails because room not found
```

Debugging strategy:

- Log room ID, player ID, close reason, and delayed removal result.
- Add logs when `RemovePlayerAfterDelay` skips removal because player reconnected.
- Add frontend logging for websocket close/error codes in development.
- Confirm whether room loss correlates with server restarts or 30-second inactivity.

Recommended fixes:

- Add a `GET /room/{id}` or `POST /room/recover` endpoint so frontend can validate local room state before opening board UI.
- Store game type with room ID in localStorage.
- Show "room expired" UI instead of immediate reload.
- Consider longer grace period for active games.

Architecture improvements:

- Persist room metadata and completed games.
- Use explicit room expiration policy.
- Add room lifecycle events to logs/metrics.

## Issue: Refresh Causes False Player Left Alert

Suspected root cause:

- Backend broadcasts `player_left` immediately on websocket close.
- Actual player removal is delayed, but UX already receives a leave alert.

Files involved:

- `server/api/websocket.go`
- `mini-game-client/src/components/ChessBoard.tsx`
- `mini-game-client/src/components/TicTacToeBoard.tsx`

Probable lifecycle path:

```text
browser refresh
  -> old websocket closes
  -> backend handlePlayerDisconnection broadcasts player_left
  -> frontend opponent sees "Player Left"
  -> same player reconnects within seconds
  -> backend broadcasts player_joined
```

Debugging strategy:

- Reproduce with two browser windows and refresh one player.
- Compare timestamps of `player_left` and `player_joined`.
- Inspect whether `RemoveClient(oldClient)` returns true or false during replacement.

Recommended fixes:

- Change event semantics:
  - `player_disconnected` immediately
  - `player_left` only after 30-second grace removal
- Or include `transient: true` in immediate disconnect event.
- Frontend should show reconnecting presence, not a modal, for transient disconnects.

Architecture improvements:

- Separate connection presence from room membership.
- Add connection/session ID and reconnect grace status to snapshots.

## Issue: Reconnect Lifecycle Is Too Broad on Frontend

Suspected root cause:

- `shouldReconnect` always returns true.
- `onError` also clears room state and reloads after 1 second.
- This mixes transient network errors with actual room expiration.

Files involved:

- `mini-game-client/src/utils/gameWebsocket.ts`

Probable lifecycle path:

```text
temporary websocket error
  -> onError shows expired-room alert
  -> removes roomId/playerMark
  -> reloads page
  -> reconnect logic becomes irrelevant
```

Debugging strategy:

- Log websocket close code/reason and distinguish `onError` from `onClose`.
- Simulate backend restart, network offline, and invalid room separately.

Recommended fixes:

- Add `onClose` handling with code-aware behavior.
- Do not clear room state on generic error.
- Only clear room after explicit server rejection or failed room recovery endpoint.
- Add UI state for reconnecting.

Architecture improvements:

- Introduce a frontend `useRoomSession` hook that owns room recovery and websocket states.

## Issue: WebSocket Ownership Is Global by Player ID

Suspected root cause:

- `ClientRegistry.clients` is `map[string]*Client`.
- Key is only `playerID`, not room ID or connection ID.

Files involved:

- `server/api/client.go`

Probable lifecycle path:

```text
same player opens two rooms/tabs
  -> second websocket Attach(playerID)
  -> existing client closes
  -> first tab disconnects unexpectedly
  -> broadcasts/removal behavior depends on exact old/new client timing
```

Debugging strategy:

- Open chess and tictactoe with same registered player.
- Observe old websocket closure and room presence events.
- Add registry logs showing room ID, player ID, and connection ID.

Recommended fixes:

- Key registry by `(roomID, playerID)` for current game model.
- Add generated connection ID for exact close/remove semantics.
- Avoid global single connection unless product explicitly enforces one active game per player.

Architecture improvements:

- Model presence as room-scoped.
- Prepare for spectators and multi-device sessions.

## Issue: Stale Room State in Frontend LocalStorage

Suspected root cause:

- `roomId` and `playerMark` are stored globally, not per game.
- Chess and TicTacToe pages both read the same keys.
- Room validity is not checked before rendering the board component.

Files involved:

- `mini-game-client/src/app/chess/page.tsx`
- `mini-game-client/src/app/tictactoe/page.tsx`
- `mini-game-client/src/components/Lobby.tsx`

Probable lifecycle path:

```text
player creates tictactoe room
  -> localStorage.roomId set
  -> navigates to chess
  -> chess page reads same roomId
  -> opens chess board with room from another game type
  -> websocket validation may fail or state shape may not match
```

Debugging strategy:

- Switch between game pages without clearing room state.
- Inspect stored values and websocket failure path.

Recommended fixes:

- Store per-game room keys:
  - `chess.roomId`
  - `tictactoe.roomId`
- Store `gameType` with room session.
- Validate room game type before opening websocket.

Architecture improvements:

- Create a `RoomSessionStore` abstraction rather than direct localStorage use in pages.

## Issue: Race Around Delayed Removal and Reconnect

Current behavior:

- Old websocket close schedules removal after 30 seconds.
- Timer checks `clients.IsConnected(playerID)`.
- If player reconnected, removal is skipped.

Why this mostly works:

- `ClientRegistry.Attach` replaces old clients by player ID.
- `RemoveClient(oldClient)` only removes if the old client is still the current registry value.

Remaining risk:

- Because registry is not room-scoped, reconnecting the same player in another room can cause removal skip for the old room.

Files involved:

- `server/api/client.go`
- `server/api/websocket.go`
- `server/service/game_service.go`

Probable lifecycle path:

```text
player leaves room A
  -> delayed removal scheduled for room A
  -> player connects to room B with same playerID
  -> clients.IsConnected(playerID) returns true
  -> room A removal is skipped
```

Debugging strategy:

- Test same player across two rooms.
- Log room ID in registry operations after registry becomes room-scoped.

Recommended fixes:

- Make `isConnected` room-aware.
- Pass `(roomID, playerID)` into connection registry checks.

Architecture improvements:

- Room-scoped presence is required before matchmaking/spectators.

## Issue: AI/Game State Desync Possibilities

Observed safeguards:

- AI move requests include `aiMoveVersion`.
- Scheduled AI move checks room state, current turn, and AI player.
- Cancelling scheduled moves increments version.
- Room mutations are protected by `Room.mu`.

Potential weak path:

- If Stockfish returns a move but applying it fails, `aiThinking` can be cleared and `stateVersion` bumped without broadcasting unless `changed` is true in `runScheduledChessAIMove`.
- Engine errors use `finishScheduledAIMove(version, true)` and do notify if thinking changed.

Files involved:

- `server/game/room_actions.go`
- `server/game/stockfish.go`

Probable lifecycle path:

```text
human move schedules AI
  -> aiThinking true
  -> Stockfish returns move
  -> backend fails to apply returned move
  -> aiThinking false and stateVersion bump
  -> no notify because changed=false
  -> frontend may keep showing thinking until next snapshot
```

Debugging strategy:

- Force invalid Stockfish response in tests with a fake engine abstraction.
- Add log on failed AI move application.
- Assert a snapshot is broadcast when `aiThinking` changes from true to false.

Recommended fixes:

- Notify whenever AI thinking state changes, even if move application fails.
- Consider emitting `ai_error` metadata in snapshot or websocket event.

Architecture improvements:

- Move Stockfish behind interface to test AI failure paths.

## Issue: Frontend Stale Snapshot Handling Is Chess-Only

Observed behavior:

- Chess ignores stale snapshots using `state_version`.
- TicTacToe does not appear to use `state_version` to reject stale snapshots.

Files involved:

- `mini-game-client/src/utils/handleGameChessUpdate.ts`
- `mini-game-client/src/components/TicTacToeBoard.tsx`

Probable lifecycle path:

```text
delayed websocket event arrives after newer state
  -> chess ignores it
  -> tictactoe may apply it
```

Debugging strategy:

- Inject delayed websocket messages in dev/test.
- Compare ChessBoard and TicTacToeBoard reconciliation behavior.

Recommended fixes:

- Use shared `state_version` handling for all room/game features.
- Centralize room snapshot reducer.

Architecture improvements:

- Introduce typed room snapshot store on frontend.

## Issue: Backend Rate Limiter May Affect WebSocket Upgrade

Suspected root cause:

- Rate limiter middleware wraps all routes including `/ws`.
- It keys by `r.RemoteAddr`, which includes IP and port.
- Behind proxies, behavior may not represent true client IP.

Files involved:

- `server/main.go`
- `server/middleware/limitter.go`

Probable lifecycle path:

```text
rapid reconnects or page reloads
  -> websocket upgrade requests hit rate limiter
  -> server returns 429
  -> frontend treats as room expired/error
```

Debugging strategy:

- Log 429s by path.
- Test rapid refresh and multiple tabs.

Recommended fixes:

- Exempt `/ws` from strict HTTP rate limiter or use websocket-specific limits.
- Use trusted proxy IP extraction.
- Return clear error payload for rate-limited websocket upgrades.

Architecture improvements:

- Separate API rate limiting from realtime connection limiting.

