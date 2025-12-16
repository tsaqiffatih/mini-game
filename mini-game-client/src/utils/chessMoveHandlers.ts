// utils/chessMoveHandlers.ts
import { Chess, Square, Piece, Move } from "chess.js";
import playCaptureSound from "./capturedSound";
import playMoveSound from "./moveSound";

type SendMessageFn = (message: string) => void;
type SetFenFn = (fen: string) => void;
type SetLastMoveFn = (move: { from: string; to: string } | null) => void;
type SetWinnerFn = (winner: string) => void;
type SetIsGameActiveFn = (isActive: boolean) => void;
type PromotionPiece = "q" | "r" | "b" | "n";

export const checkGameStatus = (
  game: Chess,
  sendMessage: SendMessageFn,
  setWinner: SetWinnerFn,
  setIsGameActive: SetIsGameActiveFn
) => {
  try {
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
      return;
    }

    if (game.isDraw() || game.isThreefoldRepetition() || game.isStalemate()) {
      sendMessage(
        JSON.stringify({
          action: "GAME_DRAW",
          message: "The match ended in a draw.",
        })
      );
      setWinner("Draw");
      setIsGameActive(false);
    }
  } catch (err) {
    // non-fatal, but help debugging
  
    console.error("checkGameStatus error:", err);
  }
};

export const handleChessMove = (
  game: Chess,
  sourceSquare: Square,
  targetSquare: Square,
  playerMarkState: string,
  playerId: string,
  sendMessage: SendMessageFn,
  setFen: SetFenFn,
  setLastMove: SetLastMoveFn,
  setWinner: SetWinnerFn,
  setIsGameActive: SetIsGameActiveFn,
  promotionPiece?: PromotionPiece,
): boolean => {
  // Validate turn ownership quickly
  if (
    (game.turn() === "w" && playerMarkState !== "white") ||
    (game.turn() === "b" && playerMarkState !== "black")
  ) {
    // Not this player's turn
  
    console.log("handleChessMove: not player's turn", {
      turn: game.turn(),
      playerMarkState,
    });
    return false;
  }

  // Defensive: cast squares to chess.js Square type
  const from = sourceSquare as Square;
  const to = targetSquare as Square;

  try {
    // Inspect piece at source
    const piece = game.get(from) as Piece | null;

    // Build move input (include promotion conditionally)
    const moveInput: Record<string, unknown> = { from, to };

    // If piece is pawn and reaching last rank -> add promotion
    if (piece && piece.type === "p") {
      const isWhitePawnPromote = piece.color === "w" && to.endsWith("8");
      const isBlackPawnPromote = piece.color === "b" && to.endsWith("1");
      if (isWhitePawnPromote || isBlackPawnPromote) {
        if (!promotionPiece) {
          // indicate to caller that move wasn't completed because promotion choice missing
          return false;
        }
        // use user-selected promotion piece
        moveInput.promotion = promotionPiece;
      }
    }

    // Try to make the move
    const moveResult = game.move(moveInput as any) as Move | null;

    if (!moveResult) {
      // Move illegal according to chess.js
    
      console.log("handleChessMove: illegal move", { from, to, moveInput });
      return false;
    }

    // Play sounds (side-effect)
    if (moveResult.captured) {
      // lazy import or call external sound util
      try {
        // require dynamic import because file might be client-only
        
        // const { default: playCaptureSound } = require("./capturedSound");
        playCaptureSound();
      } catch {
        // ignore if sound util not available
      }
    } else {
      try {
        
        // const { default: playMoveSound } = require("./moveSound");
        playMoveSound();
      } catch {
        // ignore
      }
    }

    // Update local UI state
    setFen(game.fen());
    setLastMove({ from: sourceSquare, to: targetSquare });

    // Build message and send to server (wrapped in try/catch)
    const moveMessage = {
      action: "CHESS_MOVE",
      message: {
        fen: game.fen(),
        lastMove: { from: sourceSquare, to: targetSquare },
      },
      sender: { player_id: playerId },
    };

    try {
      sendMessage(JSON.stringify(moveMessage));
    
      console.log("handleChessMove: sent CHESS_MOVE", moveMessage);
    } catch (err) {
    
      console.error("handleChessMove: sendMessage failed", err, moveMessage);
    }

    // Check game status
    checkGameStatus(game, sendMessage, setWinner, setIsGameActive);

    return true;
  } catch (err) {
    // Any unexpected runtime error
    
    console.error("handleChessMove error:", err, { sourceSquare, targetSquare });
    return false;
  }
};
