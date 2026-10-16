import { Chess } from "chess.js";
import type { Key } from "chessground/types";

import { showErrorAlert } from "@/utils/alerthelper";
import { CapturedPiece } from "@/components/ChessBoard";

type PlayerMark = "white" | "black";
type RoomState = "WAITING" | "PLAYING" | "FINISHED" | "RESETTING";

type RoomPlayerSnapshot = {
  player_id?: string;
  player_mark?: unknown;
};

type ChessMoveSnapshot = {
  id: string;
  actor?: { player_id?: string };
  flags?: {
    checkmate?: boolean;
    check?: boolean;
    castle?: boolean;
    capture?: boolean;
    promotion?: boolean;
  };
};

type ChessStateSnapshot = {
  fen?: unknown;
  winner?: unknown;
  pgn_moves?: unknown;
  last_move?: unknown;
  captured_pieces?: unknown;
};

type ChessRoomSnapshot = {
  state_version?: unknown;
  state?: unknown;
  players?: unknown;
  game?: { chess?: unknown };
  chess?: unknown;
};

type LastMove = {
  from: Key;
  to: Key;
};

interface HandleGameChessUpdateParams {
  state: unknown;

  playerId: string;

  chessRef: React.MutableRefObject<Chess>;

  latestStateVersionRef: React.MutableRefObject<number>;

  lastPlayedMoveIdRef: React.MutableRefObject<string | null>;

  setRoomState: (value: RoomState) => void;

  setPlayerMark: (value: PlayerMark) => void;

  setFen: (value: string) => void;

  setWinner: (value: string) => void;

  setPgnMoves: (value: string[]) => void;

  setLastMove: (value: LastMove | null) => void;

  getLastMoveFromSnapshot: (move: unknown) => LastMove | null;

  isPlayerMark: (value: unknown) => value is PlayerMark;

  playMoveSelf: () => void;

  playMoveOpponent: () => void;

  playCapture: () => void;

  playCastle: () => void;

  playCheck: () => void;

  playGameEnd: () => void;

  playPromote: () => void;

  setCapturedPieces: (value: {
    white: CapturedPiece[];
    black: CapturedPiece[];
  }) => void;
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object";

const isRoomState = (value: unknown): value is RoomState =>
  value === "WAITING" ||
  value === "PLAYING" ||
  value === "FINISHED" ||
  value === "RESETTING";

const isChessRoomSnapshot = (value: unknown): value is ChessRoomSnapshot =>
  isRecord(value);

const getRoomPlayers = (value: unknown): RoomPlayerSnapshot[] => {
  if (!Array.isArray(value)) return [];

  return value.flatMap((player) => {
    if (!isRecord(player)) return [];

    return [{
      player_id: typeof player.player_id === "string" ? player.player_id : undefined,
      player_mark: player.player_mark,
    }];
  });
};

const getChessState = (value: unknown): ChessStateSnapshot | null => {
  if (!isRecord(value)) return null;

  return {
    fen: value.fen,
    winner: value.winner,
    pgn_moves: value.pgn_moves,
    last_move: value.last_move,
    captured_pieces: value.captured_pieces,
  };
};

const isChessMoveSnapshot = (value: unknown): value is ChessMoveSnapshot => {
  if (!isRecord(value)) return false;

  const actor = isRecord(value.actor) ? value.actor : undefined;
  const flags = isRecord(value.flags) ? value.flags : undefined;
  const areBooleanFlags = [
    flags?.checkmate,
    flags?.check,
    flags?.castle,
    flags?.capture,
    flags?.promotion,
  ].every((flag) => flag === undefined || typeof flag === "boolean");

  return (
    typeof value.id === "string" &&
    (actor?.player_id === undefined || typeof actor.player_id === "string") &&
    areBooleanFlags
  );
};

const isCapturedPiece = (value: unknown): value is CapturedPiece =>
  isRecord(value) &&
  typeof value.type === "string" &&
  typeof value.color === "string";

const getCapturedPieces = (value: unknown): CapturedPiece[] =>
  Array.isArray(value) && value.every(isCapturedPiece) ? value : [];

export const handleGameChessUpdate = ({
  state,
  playerId,
  chessRef,
  latestStateVersionRef,
  lastPlayedMoveIdRef,
  setRoomState,
  setPlayerMark,
  setFen,
  setWinner,
  setPgnMoves,
  setLastMove,
  getLastMoveFromSnapshot,
  isPlayerMark,
  playMoveSelf,
  playMoveOpponent,
  playCapture,
  playCastle,
  playCheck,
  playGameEnd,
  playPromote,
  setCapturedPieces,
}: HandleGameChessUpdateParams) => {
  if (!isChessRoomSnapshot(state)) return;

  /** ignore stale snapshots */
  if (
    typeof state.state_version === "number" &&
    state.state_version <= latestStateVersionRef.current
  ) {
    return;
  }

  if (typeof state.state_version === "number") {
    latestStateVersionRef.current = state.state_version;
  }

  if (isRoomState(state.state)) setRoomState(state.state);

  /** PLAYER MARK */
  const players = getRoomPlayers(state.players);

  const me = players.find((p) => p.player_id === playerId);

  if (isPlayerMark(me?.player_mark)) {
    setPlayerMark(me.player_mark);
  }

  /** CHESS STATE */
  const chessState = getChessState(
    isRecord(state.game) ? state.game.chess : state.chess,
  );

  if (!chessState) return;

  if (typeof chessState.fen === "string") {
    try {
      chessRef.current.load(chessState.fen);

      setFen(chessState.fen);
    } catch {
      showErrorAlert("Received an invalid chess position");

      return;
    }
  }

  setWinner(typeof chessState.winner === "string" ? chessState.winner : "");

  setPgnMoves(
    Array.isArray(chessState.pgn_moves) &&
      chessState.pgn_moves.every((move) => typeof move === "string")
      ? chessState.pgn_moves
      : [],
  );

  const move = chessState.last_move;

  setLastMove(getLastMoveFromSnapshot(move));

  if (!isChessMoveSnapshot(move)) return;

  /** prevent replaying same move sound */
  if (move.id === lastPlayedMoveIdRef.current) {
    return;
  }

  const captured = chessState.captured_pieces;

  setCapturedPieces({
    white: isRecord(captured) ? getCapturedPieces(captured.white) : [],
    black: isRecord(captured) ? getCapturedPieces(captured.black) : [],
  });

  lastPlayedMoveIdRef.current = move.id;

  const isSelfMove = move?.actor?.player_id === playerId;

  if (move?.flags?.checkmate) {
    playGameEnd();
  } else if (move?.flags?.check) {
    playCheck();
  } else if (move?.flags?.castle) {
    playCastle();
  } else if (move?.flags?.capture) {
    playCapture();
  } else if (move?.flags?.promotion) {
    playPromote();
  } else {
    if (isSelfMove) {
      playMoveSelf();
    } else {
      playMoveOpponent();
    }
  }
};
