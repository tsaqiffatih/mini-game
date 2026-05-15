import { Square } from "chess.js";

type SendMessageFn = (message: string) => void;

export type PromotionPiece = "q" | "r" | "b" | "n";

export function sendChessMove(
  sendMessage: SendMessageFn,
  from: Square,
  to: Square,
  promotion?: PromotionPiece
): boolean {
  const payload = {
    type: "move",
    payload: {
      from,
      to,
      ...(promotion ? { promotion } : {}),
    },
  };

  try {
    sendMessage(JSON.stringify(payload));
    return true;
  } catch (err) {
    console.error("sendChessMove failed:", err);
    return false;
  }
}
