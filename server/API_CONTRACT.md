# API Contract

_This contract is generated from the current Go source code in the `server/` package. It describes the public HTTP and WebSocket interfaces, request/response schemas, required headers, query parameters, and error handling._

---

## Common HTTP Response Envelope

All HTTP handlers return JSON wrapped in the following envelope:

```json
{
  "success": true | false,
  "data": <any>,
  "error": null | "error message"
}
```

- **`success`** – `true` for successful requests, `false` otherwise.
- **`data`** – The payload for successful requests (object, array, or `null`).
- **`error`** – `null` on success; a string with a human‑readable error message on failure.

### Standard HTTP Status Codes
| Code | Meaning | When used |
|------|---------|-----------|
| `200 OK` | Success for GET/POST that return data. | Normal operation |
| `201 Created` | Resource successfully created. | `POST /create/user`, `POST /room/create`, `POST /room/create/ai` |
| `400 Bad Request` | Invalid JSON, missing required fields, or request validation failure. | Body parsing errors, unknown `game_type`, invalid `ai_level`, etc. |
| `401 Unauthorized` | Missing or invalid `Authorization: Bearer <token>` header (REST) or invalid session token (WebSocket). |
| `404 Not Found` | Player, room, or game type not found. |
| `429 Too Many Requests` | Rate‑limiter triggered (middleware). |
| `500 Internal Server Error` | Unexpected server error.

---

## HTTP Endpoints

### `GET /`
- **Purpose**: Simple health / hello endpoint.
- **Headers**: none.
- **Response (200)**:
  ```json
  { "success": true, "data": null, "error": null }
  ```

### `GET /health` and `GET /healthz`
- **Purpose**: Liveness/Readiness health check.
- **Headers**: none.
- **Response (200):**
  ```json
  { "success": true, "data": { "status": "ok" }, "error": null }
  ```

---

### `POST /create/user`
- **Purpose**: Register a new player (in‑memory).
- **Headers**:
  - `Content-Type: application/json`
- **Request Body**:
  ```json
  { "player_id": "string" }
  ```
- **Success (201)** – `data` contains a `PlayerDTO`:
  ```json
  {
    "id": "p1",
    "player_id": "p1",
    "mark": "",
    "player_mark": "",
    "is_ai": false,
    "last_active": "2026-05-03T00:00:00Z",
    "session": "disconnected"
  }
  ```
- **Error (400)** – Invalid JSON or duplicate player.

---

### `POST /room/create`
- **Purpose**: Create a new room and add the requesting player.
- **Headers**:
  - `Content-Type: application/json`
  - `Authorization: Bearer <session_token>` – token obtained from `/create/user`.
- **Request Body**:
  ```json
  { "game_type": "tictactoe" | "chess", "player_id": "string" }
  ```
- **Success (201)** – `data` is a `JoinRoomResponseDTO`:
  ```json
  { "player_id": "p1", "player_mark": "X", "room": { "id": "ABC1234", "room_id": "ABC1234" } }
  ```
- **Error Mapping** (via `createRoomStatus`):
  - `404` – player not found (`service.ErrPlayerNotFound`).
  - `400` – missing/invalid `game_type` (`service.ErrGameTypeRequired`).
  - `503` – AI unavailable or max AI rooms reached (`game.ErrChessAIUnavailable`, `service.ErrMaxAIRoomsReached`).
  - `400` – any other validation error.

---

### `POST /room/create/ai`
- **Purpose**: Create a room with an AI opponent.
- **Headers**: same as `/room/create`.
- **Request Body**:
  ```json
  { "game_type": "tictactoe" | "chess", "player_id": "string", "ai_level": integer (optional) }
  ```
- **Success (201)** – Same payload as `/room/create`.
- **Error Mapping** (via `createRoomStatus`):
  - `404` – player not found.
  - `400` – missing/invalid `game_type` or unsupported AI for the requested game.
  - `503` – AI unavailable.

---

### `POST /room/join`
- **Purpose**: Join an existing room.
- **Headers**: same as `/room/create`.
- **Request Body**:
  ```json
  { "room_id": "ABC1234", "player_id": "string", "game_type": "tictactoe" | "chess" }
  ```
- **Success (200)** – `JoinRoomResponseDTO` (same shape as room creation).
- **Error Mapping** (via `joinRoomStatus`):
  - `400` – game type mismatch (`service.ErrGameTypeMismatch`).
  - `404` – player not found (`service.ErrPlayerNotFound`).
  - `400` – invalid room state (`game.ErrInvalidGameState`).
  - `404` – generic not‑found for other join failures.

---

### `POST /room/recover`
- **Purpose**: Re‑establish a session after a disconnect (client‑side reconnection).
- **Headers**: same as `/room/create`.
- **Request Body**:
  ```json
  { "room_id": "ABC1234", "player_id": "string", "game_type": "tictactoe" | "chess" }
  ```
- **Success (200)** – Returns the player snapshot and full room snapshot:
  ```json
  {
    "player_id": "p1",
    "player_mark": "X",
    "room": { ...RoomSnapshotDTO... }
  }
  ```
- **Error Mapping** (via `recoverRoomStatus`):
  - `400` – missing/invalid fields or game‑type mismatch.
  - `404` – player or room not found (`service.ErrPlayerNotFound`, `service.ErrRoomNotFound`).

---

## WebSocket API

### Connection URL
```
/ws?room_id=<ROOM_ID>&player_id=<PLAYER_ID>&session_token=<TOKEN>
```
- **`room_id`** – Required, must match the pattern `^[A-Z0-9]{7}$` (validated by `validRoomID`).
- **`player_id`** – Required, must correspond to a player that already exists in the room.
- **`session_token`** – The session token obtained when the player was created (`/create/user`). It is verified by `middleware.VerifySessionToken`.
- **Headers**: `Authorization: Bearer <session_token>` is optional for the WebSocket upgrade; the token is provided via query param.
- **Upgrade Errors** – If validation fails, the server returns a normal HTTP envelope with the appropriate status (e.g., `400` or `401`). On successful upgrade the client is attached and receives the initial `room_update` plus chat history.

### Message Envelope (both inbound and outbound)
```json
{ "type": "EVENT_TYPE", "payload": { ... } }
```
- `type` – String identifying the event.
- `payload` – Event‑specific JSON object (see tables below).

### Inbound (Client → Server) Events
| Type | Description | Payload Schema |
|------|-------------|----------------|
| `TICTACTOE_MOVE` | Submit a Tic‑Tac‑Toe move. | `TictactoeMovePayload` – `{ "room_id":"...", "player_id":"...", "row":0, "col":0 }` (room_id/player_id fields are ignored; the server uses the connection context). |
| `CHESS_MOVE` | Submit a chess move. | `ChessMovePayload` – `{ "from":"e2", "to":"e4", "promotion":"" }` |
| `CHESS_UNDO_REQUEST` | Request an undo of the last human turn in an AI chess room. | `ChessUndoPayload` – `{ "mode":"last_turn" }` (currently only `last_turn` supported). |
| `CHAT_SEND` | Send a chat message to the room. | `ChatSendPayload` – `{ "message":"Hello" }` |
| `CREATE_ROOM_WITH_AI` | Create an AI‑enabled Tic‑Tac‑Toe room with a specific ID (used by the client for deterministic room IDs). Payload is a raw JSON string containing the desired `room_id`. | `"ABC1234"` |

### Outbound (Server → Client) Events
| Type | Description | Payload Schema |
|------|-------------|----------------|
| `room_update` | Sent on connection, after successful `CREATE_ROOM_WITH_AI`, and when a player joins/leaves. Contains a full `RoomSnapshotDTO`. |
| `game_update` | Broadcast after a successful move (Tic‑Tac‑Toe or chess) or after an undo. Contains a full `RoomSnapshotDTO`. |
| `player_joined` | A player has connected to the room. Payload: `RoomEventPayload` with `room` (snapshot) and `data` containing a message and player info. |
| `player_reconnecting` / `player_reconnected` | Client is reconnecting or has reconnected. Same payload shape as `player_joined`. |
| `player_left` | Player has permanently left (after disconnect grace period). Payload similar to `player_joined`. |
| `chat_message` | A new chat message has been posted. Payload: `ChatMessageDTO`. |
| `chat_history` | Sent once after connection, contains all historic messages. Payload: `ChatHistoryDTO`. |
| `error` | Generic error for malformed JSON, unsupported message type, etc. Payload: `{ "message": "..." }`. |
| `chess_move_rejected` | Chess move validation failed. Payload: `ChessMoveRejectedDTO` – includes `room_id`, `player_id`, attempted move, error `code`, human‑readable `message`, and a `sound` hint. |

### Error Codes for `chess_move_rejected`
| Code | Meaning |
|------|---------|
| `invalid_payload` | JSON could not be parsed. |
| `not_your_turn` | Player attempted to move out of turn. |
| `promotion_required` | Pawn promotion required but not supplied. |
| `game_not_active` | Move attempted on a finished or inactive game. |
| `player_not_in_room` | Player not part of the room. |
| `illegal_move` | Move violates chess rules. |
| `invalid_move` | Generic fallback for other validation failures. |

---

## Rate Limiting
The server applies a token‑bucket rate limiter (see `middleware/limitter.go`). Exceeding the limit returns **`429 Too Many Requests`** with the standard response envelope.

---

## DTO Reference (generated from `api/dto/*.go`)
- **`PlayerDTO`** – fields: `id`, `player_id`, `mark`, `player_mark`, `is_ai`, `last_active`, `session`.
- **`RoomDTO`** – `id`, `room_id`.
- **`JoinRoomResponseDTO`** – `player_id`, `player_mark`, `room` (RoomDTO).
- **`RoomSnapshotDTO`** – comprehensive snapshot including `id`, `room_id`, `state_version`, `game_type`, `state`, `room_state`, `is_active`, `is_ai_enabled`, `ai_level`, `players[]`, `game` (GameStateDTO), `tictactoe`, `chess` (deprecated alias).
- **`GameStateDTO`** – `type` (`"tictactoe"` or `"chess"`), embeds the concrete game state DTO.
- **`TicTacToeStateDTO`** – `board`, `turn`, `winner`, `status`, `is_active`.
- **`ChessStateDTO`** – `schema_version`, `fen`, `is_active`, `winner`, `pgn_moves`, `turn`, `status`, `result`, `ply`, `fullmove_number`, `last_move`, `check`, `captured_pieces`, `legal_moves`, `ai` (ChessAIDTO), `undo` (ChessUndoDTO).
- **`ChatMessageDTO`**, **`ChatHistoryDTO`**, **`ChessMoveRejectedDTO`**, etc.

---

*This contract should be kept in sync with the source code. Any change to request bodies, response structures, or event names must be reflected here.*
