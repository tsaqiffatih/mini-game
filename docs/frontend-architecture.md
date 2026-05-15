# Frontend Architecture

## Current Structure

The active frontend is `mini-game-client`, a Next.js App Router application.

Current layout:

```text
src/
  app/
    page.tsx
    chess/page.tsx
    tictactoe/page.tsx
    layout.tsx
    globals.css
  components/
    ChessBoard.tsx
    TicTacToeBoard.tsx
    Lobby.tsx
    RegisterUser.tsx
    ChatOpened.tsx
    ChessMoveHistory.tsx
    ...
  utils/
    gameWebsocket.ts
    handleGameChessUpdate.ts
    chessUtils.ts
    useChessSounds.ts
    ...
```

Dependencies:

- Next.js 15
- React 19
- Tailwind/DaisyUI
- Axios
- react-use-websocket
- chess.js
- chessground
- SweetAlert2

## Routing Strategy

Routes are simple:

- `/`: player registration and game selection.
- `/chess`: chess lobby or active chess board.
- `/tictactoe`: TicTacToe lobby or active board.

The game pages are client components that read `localStorage`. This is reasonable for the current prototype, but it couples room/session recovery directly to route rendering.

## State Management

There is no global state library. State is local React state plus `localStorage`.

Persistent browser keys:

- `playerId`
- `roomId`
- `playerMark`
- `aiLevel`

Current limitation: `roomId` and `playerMark` are shared across games. Chess and TicTacToe both read the same keys, which can create stale or cross-game room confusion.

## WebSocket Ownership

`src/utils/gameWebsocket.ts` owns physical websocket creation.

It:

- Builds URL from `NEXT_PUBLIC_WS_BACKEND_URL`, `roomId`, and `playerId`.
- Logs open/error in development.
- Reconnects unconditionally.
- On error, shows "Room expired", clears room data, and reloads.

This hook is intentionally small, but it currently mixes connection transport with product behavior. A generic websocket error is not always a room expiration.

Recommended future split:

```text
shared/websocket/useWebSocketConnection
features/room/useRoomSession
features/chess/useChessRealtime
features/tictactoe/useTicTacToeRealtime
features/chat/useChatRealtime
```

## Chess Component Coupling

`ChessBoard.tsx` currently owns many responsibilities:

- Board setup/destruction.
- Chessground config.
- Local `chess.js` state.
- WebSocket message parsing.
- Room state UI.
- Move submission.
- Chat.
- Move history.
- Captured pieces.
- Sound effects.
- Leave/back behavior.

This concentration is workable while the feature is young, but it will slow future additions like promotion UI, analysis arrows, undo buttons, AI identity, and real timers.

Recommended extraction:

- `useChessSnapshotReducer`
- `useChessBoardController`
- `useChessMoveSender`
- `ChessPlayerPanel`
- `ChessResultBanner`
- `ChessPromotionDialog`
- `ChessCapturedPieces`
- `ChessChatPanel`

## Chess State Flow

```text
backend game_update
  -> ChessBoard parses message
  -> handleGameChessUpdate
  -> reject stale state_version
  -> load FEN into chessRef
  -> update React state
  -> Chessground set()
  -> sound based on last_move flags
```

The backend snapshot is authoritative. The local `chess.js` instance is a mirror used for legal destinations, turn checks, and board UX.

## TicTacToe State Flow

TicTacToe is simpler:

```text
backend game_update
  -> TicTacToeBoard parses message
  -> set roomState/board/turn/winner
```

Unlike chess, TicTacToe does not currently reject stale snapshots with `state_version`.

## Shared vs Feature Separation

Current `components/` and `utils/` are global. As the project grows, this will increase accidental coupling.

Recommended target:

```text
src/
  features/
    player/
    lobby/
    room/
    chat/
    chess/
    tictactoe/
  shared/
    api/
    websocket/
    ui/
    alerts/
    sounds/
```

Feature-owned code should include feature-specific DTOs, reducers, hooks, and components. Shared code should be generic and stable.

## Current Weaknesses

- Room/session state is raw localStorage access spread across pages/components.
- WebSocket error handling is too aggressive.
- ChessBoard is large and feature-heavy.
- TicTacToe and Chess duplicate room/chat websocket handling.
- Frontend DTOs are mostly implicit and partially `any`.
- AI snapshot fields are not fully used by UI.
- Chess timers are placeholders.
- Promotion is auto-queen.
- Same `roomId` key is reused across games.

## Migration Plan

1. Add typed backend DTOs in frontend.
2. Introduce `RoomSessionStore` for localStorage.
3. Extract `useRoomSocket` with explicit socket states.
4. Move chess files into `features/chess`.
5. Move chat into `features/chat`.
6. Convert `ChessBoard` into composed components.
7. Add stale snapshot handling to TicTacToe.
8. Add UI for existing backend AI/undo/result fields.

## Scalability Problems

Frontend scalability is currently less about performance and more about maintainability:

- Large components make feature work risky.
- WebSocket event parsing is duplicated.
- Lack of typed contracts makes backend DTO changes risky.
- Local storage room model does not support multiple active games.

The right next move is not a new state library by default. First create typed feature hooks and reducers around the existing websocket contract.

