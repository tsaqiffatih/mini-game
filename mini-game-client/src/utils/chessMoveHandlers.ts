import { Chess } from "chess.js";
import playCaptureSound from "./capturedSound";
import playMoveSound from "./moveSound";

export const checkGameStatus = (
  game: Chess,
  sendMessage: (message: string) => void,
  setWinner: (winner: string) => void,
  setIsGameActive: (isActive: boolean) => void
) => {
  if (game.isCheckmate()) {
    const winnerMessage =
      game.turn() === "w"
        ? "Black pieces win by checkmate!"
        : "White pieces win by checkmate!";
    sendMessage(
      JSON.stringify({
        action: "GAME_CHECKMATE",
        message: winnerMessage,
      })
    );
    setWinner(winnerMessage);
    setIsGameActive(false);
  } else if (
    game.isDraw() ||
    game.isThreefoldRepetition() ||
    game.isStalemate()
  ) {
    sendMessage(
      JSON.stringify({
        action: "GAME_DRAW",
        message: "The match ended in a draw.",
      })
    );
    setWinner("Draw");
    setIsGameActive(false);
  }
};

export const handleChessMove = (
  game: Chess,
  sourceSquare: string,
  targetSquare: string,
  playerMarkState: string,
  playerId: string,
  sendMessage: (message: string) => void,
  setFen: (fen: string) => void,
  setLastMove: (move: { from: string; to: string }) => void,
  setWinner: (winner: string) => void,
  setIsGameActive: (isActive: boolean) => void
): boolean => {
  if (
    (game.turn() === "w" && playerMarkState !== "white") ||
    (game.turn() === "b" && playerMarkState !== "black")
  )
    return false;

  const move = game.move({
    from: sourceSquare,
    to: targetSquare,
    promotion: "q",
  });

  if (!move) return false;

  if (move.captured) playCaptureSound();
  else playMoveSound();

  setFen(game.fen());
  setLastMove({ from: sourceSquare, to: targetSquare });

  const moveMessage = {
    action: "CHESS_MOVE",
    message: {
      fen: game.fen(),
      lastMove: { from: sourceSquare, to: targetSquare },
    },
    sender: { player_id: playerId },
  };

  sendMessage(JSON.stringify(moveMessage));

  checkGameStatus(game, sendMessage, setWinner, setIsGameActive);

  return true;
};