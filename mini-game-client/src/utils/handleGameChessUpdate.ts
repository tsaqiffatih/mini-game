import { Chess } from "chess.js";
import type { Key } from "chessground/types";

import { showErrorAlert } from "@/utils/alerthelper";
import { CapturedPiece } from "@/components/ChessBoard";

type PlayerMark = "white" | "black";

type RoomPlayerSnapshot = {
  player_id?: string;
  player_mark?: unknown;
};

type LastMove = {
  from: Key;
  to: Key;
};

interface HandleGameChessUpdateParams {
  state: any;

  playerId: string;

  chessRef: React.MutableRefObject<Chess>;

  latestStateVersionRef: React.MutableRefObject<number>;

  lastPlayedMoveIdRef: React.MutableRefObject<string | null>;

  setRoomState: (value: any) => void;

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
  if (!state) return;

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

  setRoomState(state.state);

  /** PLAYER MARK */
  const players = Array.isArray(state.players)
    ? (state.players as RoomPlayerSnapshot[])
    : [];

  const me = players.find((p) => p.player_id === playerId);

  if (isPlayerMark(me?.player_mark)) {
    setPlayerMark(me.player_mark);
  }

  /** CHESS STATE */
  const chessState = state.game?.chess ?? state.chess;

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

  setWinner(chessState.winner || "");

  setPgnMoves(chessState.pgn_moves ?? []);

  const move = chessState.last_move;

  setLastMove(getLastMoveFromSnapshot(move));

  if (!move) return;

  /** prevent replaying same move sound */
  if (move.id === lastPlayedMoveIdRef.current) {
    return;
  }

  console.log(chessState.captured_pieces);

  const captured = chessState.captured_pieces;

  setCapturedPieces({
    white: Array.isArray(captured?.white) ? captured.white : [],
    black: Array.isArray(captured?.black) ? captured.black : [],
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
