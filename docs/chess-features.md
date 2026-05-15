# Chess Features

## Current Chess Architecture Baseline

Chess is the most advanced game in the project. The backend owns authoritative rules and room state, while the frontend owns board interaction and visual reconciliation.

Backend ownership:

- `server/chess/chess_state.go`: authoritative chess state, legal move application, metadata, PGN, captured pieces, check/checkmate/draw status, undo rollback primitives.
- `server/game/room.go`: room snapshot includes chess state and AI metadata.
- `server/game/room_actions.go`: validates player/room state, applies chess moves, schedules AI moves, handles undo.
- `server/game/stockfish.go`: Stockfish process wrapper.
- `server/api/game_handler.go`: websocket chess move and undo event handling.

Frontend ownership:

- `mini-game-client/src/components/ChessBoard.tsx`: board UI, websocket consumption, move sending, chat, captured pieces, move history, sounds.
- `mini-game-client/src/utils/handleGameChessUpdate.ts`: snapshot reconciliation.
- `mini-game-client/src/utils/chessUtils.ts`: local chess helper logic.
- `mini-game-client/src/utils/useChessSounds.ts`: sound playback.

## Current Features

### Multiplayer Chess

Purpose: two registered players can play in a room by sharing a room code.

UX impact: core realtime experience.

Current implementation:

- Room is created through `POST /room/create`.
- Second player joins through `POST /room/join`.
- Each player opens `/ws`.
- Backend transitions room from `WAITING` to `PLAYING` when two players exist.
- White/black marks are assigned in `Room.handleChessPlayerJoin`.

Complexity: medium.

Frontend ownership: lobby, room page, ChessBoard rendering and websocket connection.

Backend ownership: room membership, state transition, snapshots.

WebSocket requirements: `room_update`, `player_joined`, `player_left`, `game_update`.

Architecture impact: establishes room snapshot synchronization pattern used by all future multiplayer features.

### AI Chess

Purpose: allow one human to play against Stockfish.

UX impact: makes chess playable without matchmaking.

Current implementation:

- Lobby calls `POST /room/create/ai` with `game_type: "chess"` and `ai_level`.
- Backend creates AI player `"AI"` as black.
- Human joins as white.
- Stockfish process is created per AI room.
- After human move, backend schedules AI move and applies it through the same chess state mutation path.

Complexity: high.

Frontend ownership: AI difficulty modal and future AI status UI.

Backend ownership: AI room creation, Stockfish lifecycle, AI move scheduling, AI snapshot fields.

WebSocket requirements: `game_update` snapshots with `game.chess.ai`.

Architecture impact: introduces async room mutation after the original websocket event completes.

### Move Validation

Purpose: prevent illegal moves and keep all clients consistent.

UX impact: immediate local feedback plus authoritative server correctness.

Current implementation:

- Frontend uses `chess.js` to generate legal destinations and reject obvious illegal moves.
- Backend uses `notnil/chess` in `ChessGameState.findLegalMove`.
- Backend rejects invalid moves with `chess_move_rejected`.

Complexity: medium.

Frontend ownership: pre-validation, board reset after rejection, illegal sound.

Backend ownership: final validation.

WebSocket requirements: `CHESS_MOVE`, `chess_move_rejected`, `game_update`.

Architecture impact: creates dual validation, but backend remains source of truth.

### Promotion

Purpose: support pawn promotion.

Current implementation:

- Frontend detects promotion and auto-sends `"q"`.
- Backend requires promotion when a legal move needs it.

Complexity: currently low, future medium.

Frontend ownership: promotion UI is missing.

Backend ownership: promotion validation and metadata.

WebSocket requirements: `promotion` field in `CHESS_MOVE`.

Architecture impact: needs frontend UI state but no major backend change.

### Captured Pieces

Purpose: show material captured by each side.

Current implementation:

- Backend records captured pieces in `ChessGameState`.
- Snapshot exposes `captured_pieces`.
- Frontend renders piece images and material difference.

Complexity: low-medium.

Frontend ownership: display and scoring.

Backend ownership: metadata accuracy.

WebSocket requirements: included in `game_update`.

Architecture impact: stable snapshot field; useful for future analysis UI.

### Move History

Purpose: display PGN/SAN move sequence.

Current implementation:

- Backend appends SAN to `pgn_moves`.
- Frontend renders via `ChessMoveHistory`.

Complexity: low.

Frontend ownership: rendering.

Backend ownership: SAN generation.

WebSocket requirements: included in `game_update`.

Architecture impact: useful base for replay/export later.

### Sounds

Purpose: increase feedback for move types.

Current implementation:

- Backend move metadata includes flags and sound-like classification.
- Frontend chooses sounds from move flags: move, opponent move, capture, castle, check, game end, promotion, illegal.

Complexity: low.

Frontend ownership: playback and asset loading.

Backend ownership: move metadata flags.

WebSocket requirements: `last_move.flags`.

Architecture impact: mostly UI, but depends on accurate backend metadata.

### Premove

Purpose: allow a player to queue a move while it is not their turn.

Current implementation:

- Chessground premove is enabled.
- `playPremove` runs when room is playing and turn returns to the player.
- Custom premove styling is not implemented.

Complexity: medium.

Frontend ownership: almost entirely frontend.

Backend ownership: none beyond normal move validation.

WebSocket requirements: no new event required.

Architecture impact: requires careful local board state handling.

### Timers

Purpose: display chess clocks.

Current implementation:

- UI shows hardcoded placeholder values.
- No authoritative backend clock exists.

Complexity: future high if real rated clocks are desired.

Frontend ownership: display.

Backend ownership: authoritative clock state, timeout detection, pause/resume rules.

WebSocket requirements: clock snapshots, tick/update policy, timeout event.

Architecture impact: significant because clocks must be server-authoritative.

## Future Planned Features

### Checkmate Visual Highlight

Purpose: make final checkmate state immediately understandable.

UX impact: high. It gives the game a polished finish.

Implementation complexity: medium.

Frontend ownership:

- Read `game.chess.last_move.flags.checkmate`.
- Read `game.chess.check.king_square`.
- Add board highlight or overlay.

Backend ownership:

- Already exposes check/checkmate metadata.
- Verify `king_square` is correct for checkmate states.

WebSocket requirements: no new event; use `game_update`.

Dependencies: current `CheckState`, Chessground highlighting/drawable API.

Architecture impact: low. Mostly visual.

### Custom Premove Style

Purpose: make queued premoves visually distinct and brand-consistent.

UX impact: medium.

Implementation complexity: low-medium.

Frontend ownership:

- Chessground CSS/custom classes.
- Possibly local premove state indicators.

Backend ownership: none.

WebSocket requirements: none.

Dependencies: Chessground premove config and CSS.

Architecture impact: low.

### Engine Suggestion Arrow

Purpose: show suggested move arrows for learning or assist mode.

UX impact: high for portfolio polish and AI/analysis identity.

Implementation complexity: high.

Frontend ownership:

- Toggle suggestion mode.
- Render arrow on board.
- Avoid showing suggestions in competitive multiplayer unless explicitly allowed.

Backend ownership:

- Provide a suggestion endpoint or websocket request using Stockfish.
- Rate-limit engine suggestions.
- Avoid blocking room lock while engine thinks.

WebSocket requirements:

- New request event such as `CHESS_ENGINE_SUGGESTION_REQUEST`.
- New response event such as `chess_engine_suggestion`.

Dependencies: Stockfish wrapper, async job handling.

Architecture impact: medium-high. Best implemented as separate AI analysis service/worker rather than piggybacking on room mutation.

### AI Thinking Indicator

Purpose: show when Stockfish is calculating.

UX impact: high. Current backend already has `ai.thinking`.

Implementation complexity: low.

Frontend ownership:

- Read `game.chess.ai.thinking`.
- Show subtle indicator near opponent/AI avatar.

Backend ownership:

- Already sets `aiThinking`.
- Ensure error paths notify clients when thinking ends.

WebSocket requirements: current `game_update` snapshot.

Dependencies: AI room snapshot.

Architecture impact: low, but exposes any missing notifications in AI error paths.

### Undo vs AI

Purpose: let human undo the latest AI turn pair.

UX impact: high for casual play.

Implementation complexity: medium.

Frontend ownership:

- Render undo button when `game.chess.undo.can_request`.
- Send `CHESS_UNDO_REQUEST`.
- Disable while AI is thinking or game is not active.

Backend ownership:

- Existing `Room.HandleChessUndo`.
- Cancel scheduled AI move.
- Roll back human+AI moves via `RollbackLastAITurn`.

WebSocket requirements:

- Existing inbound `CHESS_UNDO_REQUEST`.
- Existing outbound `game_update`.

Dependencies: backend undo snapshot fields.

Architecture impact: low-medium. Mostly UI, but should add tests for undo during AI thinking.

### AI Difficulty UI

Purpose: make selected AI strength clear before and during game.

UX impact: medium.

Implementation complexity: low.

Frontend ownership:

- Existing modal selects `aiLevel`.
- Show level in game header/opponent panel from snapshot.

Backend ownership:

- Existing `ai_level` request and normalized `AILevel`.

WebSocket requirements: current room/chess AI snapshot.

Dependencies: `AIDifficultyModal`, `Lobby`.

Architecture impact: low.

### Promotion UI Upgrade

Purpose: allow queen, rook, bishop, or knight promotion.

UX impact: high for correctness and chess quality.

Implementation complexity: medium.

Frontend ownership:

- Detect promotion.
- Pause move send until user chooses piece.
- Send chosen promotion.
- Handle cancellation/reset cleanly.

Backend ownership:

- Already supports promotion field.
- Continue rejecting missing promotion.

WebSocket requirements: current `CHESS_MOVE.promotion`.

Dependencies: board interaction state, modal/menu UI.

Architecture impact: low-medium.

### Chess Result Banner

Purpose: explain result: checkmate, stalemate, draw, winner.

UX impact: high.

Implementation complexity: low-medium.

Frontend ownership:

- Use `roomState === "FINISHED"`.
- Render `game.chess.status`, `result`, `winner`.

Backend ownership:

- Already exposes status/result/winner.

WebSocket requirements: current `game_update`.

Dependencies: snapshot typing.

Architecture impact: low.

### AI Avatar / Identity

Purpose: make AI opponent feel intentional instead of anonymous.

UX impact: medium-high.

Implementation complexity: low.

Frontend ownership:

- Detect AI player from `players[].is_ai`.
- Render name, avatar, difficulty badge.

Backend ownership:

- Existing AI player fields.
- Optional future: configurable AI names/personas.

WebSocket requirements: current player snapshot.

Dependencies: player panel refactor.

Architecture impact: low.

