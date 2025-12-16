import { Square } from "chess.js";

type SendMessageFn = (message: string) => void;

export type PromotionPiece = "q" | "r" | "b" | "n";

export function sendChessMove(
  sendMessage: SendMessageFn,
  playerId: string,
  from: Square,
  to: Square,
  promotion?: PromotionPiece
): boolean {
  const payload = {
    action: "CHESS_MOVE",
    message: {
      from,
      to,
      ...(promotion ? { promotion } : {}),
    },
    sender: { player_id: playerId },
  };

  try {
    sendMessage(JSON.stringify(payload));
    return true;
  } catch (err) {
    console.error("sendChessMove failed:", err);
    return false;
  }
}
