# API Contract

Strictly based on current codebase.

## Common HTTP Response

All HTTP handlers write JSON in this shape:

```json
{
  "success": true,
  "data": {},
  "error": null
}
```

Error responses:

```json
{
  "success": false,
  "data": {},
  "error": "message"
}
```

## HTTP API

### `GET /`

Purpose: health/basic root response.

Request body: none.

Response:

```json
{
  "success": true,
  "data": null,
  "error": null
}
```

### `POST /create/user`

Purpose: create/register a player in memory.

Request body:

```json
{
  "player_id": "p1"
}
```

Success status: `201`

Success response `data`:

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

Error status:
- `400` for invalid JSON
- `400` if player already exists

Example error:

```json
{
  "success": false,
  "data": {},
  "error": "player already exists, choose another name"
}
```

### `POST /room/create`

Purpose: create a room and add the requesting player.

Request body:

```json
{
  "game_type": "tictactoe",
  "player_id": "p1"
}
```

Implemented `game_type` values are determined by room creation code:
- `"tictactoe"`
- `"chess"`

Success status: `201`

Success response `data`:

```json
{
  "player_id": "p1",
  "player_mark": "X",
  "room": {
    "id": "ABC1234",
    "room_id": "ABC1234"
  }
}
```

Error status:
- `400` for invalid JSON
- `400` if `game_type` is empty
- `400` for unknown game type
- `404` if player is not found

### `POST /room/create/ai`

Purpose: create an AI-enabled room and add the requesting player.

Request body:

```json
{
  "game_type": "tictactoe",
  "player_id": "p1"
}
```

Current code only supports AI for `"tictactoe"`.

Success status: `201`

Success response `data`:

```json
{
  "player_id": "p1",
  "player_mark": "X",
  "room": {
    "id": "ABC1234",
    "room_id": "ABC1234"
  }
}
```

Error status:
- `400` for invalid JSON
- `400` if `game_type` is empty
- `400` if AI is requested for unsupported game type
- `404` if player is not found

### `POST /room/join`

Purpose: join an existing room.

Request body:

```json
{
  "room_id": "ABC1234",
  "player_id": "p2",
  "game_type": "tictactoe"
}
```

Success status: `200`

Success response `data`:

```json
{
  "player_id": "p2",
  "player_mark": "O",
  "room": {
    "id": "ABC1234",
    "room_id": "ABC1234"
  }
}
```

Error status:
- `400` for invalid JSON
- `400` if game type does not match room game type
- `400` for invalid game state
- `404` if player is not found
- `404` for other join errors, including missing room

## WebSocket Contract

### Connection

Path:

```text
/ws?room_id=ABC1234&player_id=p1
```

Connection requirements:
- `room_id` query parameter is required.
- `player_id` query parameter is required.
- Player must already exist in the room.
- If validation fails before upgrade, the server returns the normal HTTP response envelope.

On successful connection:
- Client is attached by `player_id`.
- Player is marked connected.
- Server sends/broadcasts room events described below.
- Server starts a write pump for outbound events.
- Server reads inbound JSON WebSocket messages.

### Client-to-Server Message Format

```json
{
  "type": "EVENT_TYPE",
  "payload": {}
}
```

The server dispatches only these inbound message types:

## Inbound WebSocket Events

### `TICTACTOE_MOVE`

When used: submit a TicTacToe move.

Payload structure:

```json
{
  "room_id": "ABC1234",
  "player_id": "p1",
  "row": 0,
  "col": 0
}
```

Important current behavior:
- The handler uses the room and player from the WebSocket connection, not the payload `room_id` / `player_id`, when applying the move.
- `row` and `col` are used.

On success:
- Server sends `game_update`.

On failure:
- Server sends `chess_move_rejected`.

### `CHESS_UNDO_REQUEST`

When used: request authoritative rollback in an AI chess room.

Payload structure:

```json
{
  "mode": "last_turn"
}
```

Current behavior:
- Supported for AI chess rooms.
- The server rolls back the last human turn. If the latest move is an AI response, the AI move and the preceding human move are both rolled back.
- Pending AI moves are cancelled before rollback.

On success:
- Server sends `game_update`.

On failure:
- Server sends `error`.

### `CHESS_MOVE`

When used: submit a chess move.

Payload structure:

```json
{
  "from": "e2",
  "to": "e4",
  "promotion": ""
}
```

`promotion` is omitted when empty.

On success:
- Server sends `game_update`.

On failure:
- Server sends `error`.

### `CREATE_ROOM_WITH_AI`

When used: create an AI TicTacToe room by explicit room ID.

Payload structure:

```json
"ABC1234"
```

The payload is a JSON string, not an object.

Current behavior:
- Game type is hardcoded to `"tictactoe"` in this WebSocket path.

On success:
- Server sends `room_update`.

On failure:
- Server sends `error`.

## Server-to-Client Message Format

```json
{
  "type": "EVENT_TYPE",
  "payload": {}
}
```

## Outbound WebSocket Events

### `room_update`

Sent when:
- A client connects, directly to that client with the current room snapshot.
- `CREATE_ROOM_WITH_AI` succeeds.

Canonical payload shape:

```json
{
  "id": "ABC1234",
  "room_id": "ABC1234",
  "state_version": 1,
  "game_type": "tictactoe",
  "state": "WAITING",
  "room_state": "WAITING",
  "is_active": false,
  "is_ai_enabled": false,
  "players": [],
  "game": {
    "type": "tictactoe",
    "tictactoe": {}
  },
  "tictactoe": {}
}
```

`room_update` always carries the direct `RoomSnapshotDTO`. It is not wrapped in `{ "room": ..., "data": ... }`.

### `game_update`

Sent when:
- A TicTacToe move succeeds.
- A chess move succeeds.

Payload structure:

```json
{
  "id": "ABC1234",
  "room_id": "ABC1234",
  "game_type": "tictactoe",
  "state": "PLAYING",
  "room_state": "PLAYING",
  "is_active": true,
  "is_ai_enabled": false,
  "players": [],
  "game": {
    "type": "tictactoe",
    "tictactoe": {}
  },
  "tictactoe": {}
}
```

For chess rooms, `game.chess` is the canonical chess state source.

The top-level `chess` field is a deprecated compatibility alias populated with the same state for older clients. New frontend code should read `payload.game.chess` and treat `payload.chess` as temporary migration support.

Chess rooms include backward-compatible fields plus schema v2 metadata:

```json
{
  "id": "ABC1234",
  "room_id": "ABC1234",
  "state_version": 7,
  "game_type": "chess",
  "state": "PLAYING",
  "room_state": "PLAYING",
  "is_active": true,
  "is_ai_enabled": true,
  "ai_level": 6,
  "players": [],
  "game": {
    "type": "chess",
    "chess": {
      "schema_version": 2,
      "fen": "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1",
      "is_active": true,
      "winner": "",
      "pgn_moves": ["e4"],
      "turn": "black",
      "status": "active",
      "ply": 1,
      "fullmove_number": 1,
      "last_move": {
        "id": "1",
        "ply": 1,
        "move_number": 1,
        "actor": {
          "player_id": "p1",
          "color": "white",
          "is_ai": false
        },
        "from": "e2",
        "to": "e4",
        "uci": "e2e4",
        "san": "e4",
        "piece": {
          "type": "pawn",
          "color": "white"
        },
        "flags": {
          "capture": false,
          "castle": false,
          "kingside_castle": false,
          "queenside_castle": false,
          "en_passant": false,
          "promotion": false,
          "check": false,
          "checkmate": false,
          "stalemate": false,
          "draw": false
        },
        "check": {
          "is_check": false
        },
        "sound": "move",
        "animation": {
          "from": "e2",
          "to": "e4"
        },
        "created_at": "2026-05-12T00:00:00Z"
      },
      "check": {
        "is_check": false
      },
      "captured_pieces": {
        "white": [],
        "black": []
      },
      "legal_moves": {
        "e7": ["e6", "e5"]
      },
      "ai": {
        "enabled": true,
        "thinking": true,
        "player_id": "AI",
        "color": "black",
        "level": 6
      },
      "undo": {
        "can_request": true,
        "can_undo_now": true,
        "last_undoable_ply": 1
      }
    }
  },
  "chess": {
    "schema_version": 2,
    "fen": "same state as game.chess",
    "is_active": true,
    "winner": "",
    "pgn_moves": ["e4"],
    "turn": "black",
    "status": "active",
    "ply": 1,
    "fullmove_number": 1,
    "check": {
      "is_check": false
    },
    "captured_pieces": {
      "white": [],
      "black": []
    },
    "legal_moves": {
      "e7": ["e6", "e5"]
    },
    "ai": {
      "enabled": true,
      "thinking": true,
      "player_id": "AI",
      "color": "black",
      "level": 6
    },
    "undo": {
      "can_request": true,
      "can_undo_now": true,
      "last_undoable_ply": 1
    }
  }
}
```

Notes:
- `payload.game.chess` is canonical for chess state.
- `payload.chess` is deprecated and kept temporarily for compatibility.
- Inside `ChessStateDTO`, `fen`, `is_active`, `winner`, and `pgn_moves` remain for backward compatibility.
- Frontends should use `last_move`, `check`, `captured_pieces`, `legal_moves`, `ai`, and `undo` for UI semantics instead of parsing SAN/PGN.
- `state_version` is monotonically increased by authoritative room state changes and can be used by clients to ignore stale updates.

### `chess_move_rejected`

Sent when:
- A chess move fails validation.
- A chess move payload is malformed or semantically invalid.

Payload structure:

```json
{
  "room_id": "ABC1234",
  "player_id": "p1",
  "attempted_move": {
    "from": "e2",
    "to": "e5",
    "promotion": ""
  },
  "code": "illegal_move",
  "message": "illegal move",
  "sound": "illegal"
}
```

Known `code` values:
- `invalid_payload`
- `not_your_turn`
- `promotion_required`
- `game_not_active`
- `player_not_in_room`
- `illegal_move`
- `invalid_move`

### `player_joined`

Sent when:
- A WebSocket client connects to a room.

Payload structure:

```json
{
  "room": {
    "id": "ABC1234",
    "room_id": "ABC1234",
    "game_type": "tictactoe",
    "state": "PLAYING",
    "room_state": "PLAYING",
    "is_active": true,
    "is_ai_enabled": false,
    "players": []
  },
  "data": {
    "message": "Player p1 connected to room ABC1234",
    "player": {
      "id": "p1",
      "player_id": "p1",
      "mark": "X",
      "player_mark": "X",
      "is_ai": false,
      "last_active": "2026-05-03T00:00:00Z",
      "session": "connected"
    },
    "timestamp": "2026-05-03T00:00:00Z"
  }
}
```

### `player_left`

Sent when:
- WebSocket read loop ends and the client is removed from the registry.

Payload structure:

```json
{
  "room": {
    "id": "ABC1234",
    "room_id": "ABC1234",
    "game_type": "tictactoe",
    "state": "WAITING",
    "room_state": "WAITING",
    "is_active": false,
    "is_ai_enabled": false,
    "players": []
  },
  "data": {
    "message": "Player p1 left the room",
    "player": {
      "id": "p1",
      "player_id": "p1",
      "mark": "X",
      "player_mark": "X",
      "is_ai": false,
      "last_active": "2026-05-03T00:00:00Z",
      "session": "connected"
    },
    "timestamp": "2026-05-03T00:00:00Z"
  }
}
```

### `error`

Sent when:
- WebSocket message JSON is invalid.
- Non-chess payloads cannot be decoded.
- Unsupported message type is received.
- A generic/internal/unexpected action failure occurs.

Illegal, invalid, or malformed chess move submissions use `chess_move_rejected` instead of `error`.

Payload structure:

```json
{
  "message": "Unsupported message type"
}
```

## DTO Structures

### `PlayerDTO`

```json
{
  "id": "p1",
  "player_id": "p1",
  "mark": "X",
  "player_mark": "X",
  "is_ai": false,
  "last_active": "2026-05-03T00:00:00Z",
  "session": "connected"
}
```

Session values in current code:
- `"connected"`
- `"disconnected"`
- `"removed"`

### `RoomDTO`

```json
{
  "id": "ABC1234",
  "room_id": "ABC1234"
}
```

### `JoinRoomResponseDTO`

```json
{
  "player_id": "p1",
  "player_mark": "X",
  "room": {
    "id": "ABC1234",
    "room_id": "ABC1234"
  }
}
```

### `RoomSnapshotDTO`

```json
{
  "id": "ABC1234",
  "room_id": "ABC1234",
  "state_version": 7,
  "game_type": "tictactoe",
  "state": "PLAYING",
  "room_state": "PLAYING",
  "is_active": true,
  "is_ai_enabled": false,
  "players": [],
  "game": {
    "type": "tictactoe",
    "tictactoe": {}
  },
  "tictactoe": {}
}
```

Room state values in current code:
- `"WAITING"`
- `"PLAYING"`
- `"FINISHED"`
- `"RESETTING"`

### `GameStateDTO`

```json
{
  "type": "chess",
  "chess": {}
}
```

Only the matching game state is populated for the room game type.
For chess rooms, `game.chess` is canonical.

### `TicTacToeStateDTO`

```json
{
  "board": [
    ["X", "", ""],
    ["", "O", ""],
    ["", "", ""]
  ],
  "turn": "X",
  "winner": "",
  "status": "active",
  "is_active": true
}
```

TicTacToe status values in current code:
- `"waiting"`
- `"active"`
- `"ended"`

Winner values:
- Player mark, e.g. `"X"` or `"O"`
- `"Draw"`
- Empty string when no winner exists

### `ChessStateDTO`

```json
{
  "schema_version": 2,
  "fen": "current FEN string",
  "is_active": true,
  "winner": "",
  "pgn_moves": [],
  "turn": "white",
  "status": "active",
  "result": "",
  "ply": 0,
  "fullmove_number": 1,
  "last_move": null,
  "check": {
    "is_check": false
  },
  "captured_pieces": {
    "white": [],
    "black": []
  },
  "legal_moves": {},
  "ai": {
    "enabled": false,
    "thinking": false,
    "level": 10
  },
  "undo": {
    "can_request": false,
    "can_undo_now": false,
    "last_undoable_ply": 0
  }
}
```

Chess state fields are taken from the current chess game state. `schema_version`, move metadata, check state, captured pieces, legal moves, AI state, and undo state are part of the current websocket contract.

## State Version and Stale Updates

`state_version` is the authoritative monotonic version for a room snapshot.

It increments after authoritative state mutations clients should reconcile:
- players joining or being removed
- room lifecycle changes such as playing, finished, resetting, or waiting after reset
- successful TicTacToe moves
- successful chess moves
- chess AI thinking state changes
- chess AI moves
- successful chess undo
- scheduled resets

It does not increment for rejected/no-op actions, invalid moves, chat messages, read activity updates, or player activity timestamps.

Clients should track the latest `state_version` per `room_id` and ignore snapshots with a lower or equal version than the latest applied snapshot for that room. This is stale-update protection for delayed websocket delivery; it does not change the server-authoritative snapshot model.

## Explicitly Unclear or Missing

- There is no HTTP endpoint in the current router for fetching a room snapshot.
- There is no HTTP endpoint in the current router for submitting a move.
- WebSocket `TICTACTOE_MOVE` payload defines `room_id` and `player_id`, but the handler applies moves using the WebSocket connection’s room/player values.
- WebSocket `CREATE_ROOM_WITH_AI` expects a raw JSON string payload; no object format is implemented.
- Some constants exist in `actions/index.go` but are not handled by the current WebSocket dispatch logic. Only documented events above are actually dispatched or sent by the current code.
