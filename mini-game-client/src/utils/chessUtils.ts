// features/chess/utils/chess.utils.ts

import { Chess, Square } from "chess.js";
import type { Dests, Key } from "chessground/types";

export type PlayerMark = "white" | "black";

export type LastMove = {
  from: Key;
  to: Key;
};

export const isPlayerMark = (value: unknown): value is PlayerMark => {
  return value === "white" || value === "black";
};

export const isChessSquare = (value: string): value is Square => {
  return /^[a-h][1-8]$/.test(value);
};

export const getTurnFromFen = (fen: string): PlayerMark => {
  return fen.split(" ")[1] === "b" ? "black" : "white";
};

export const getPieceColor = (color: "w" | "b"): PlayerMark => {
  return color === "w" ? "white" : "black";
};

export const getMoveDests = (chess: Chess): Dests => {
  const dests: Dests = new Map();

  for (const move of chess.moves({ verbose: true })) {
    const from = move.from as Key;
    const to = move.to as Key;

    const existing = dests.get(from);

    if (existing) {
      if (!existing.includes(to)) {
        existing.push(to);
      }
    } else {
      dests.set(from, [to]);
    }
  }

  return dests;
};

export const getLastMoveFromSnapshot = (move: unknown): LastMove | null => {
  if (!move || typeof move !== "object") {
    return null;
  }

  const data = move as Record<string, unknown>;

  const from =
    typeof data.from === "string"
      ? data.from
      : typeof data.from_square === "string"
        ? data.from_square
        : null;

  const to =
    typeof data.to === "string"
      ? data.to
      : typeof data.to_square === "string"
        ? data.to_square
        : null;

  if (from && to && isChessSquare(from) && isChessSquare(to)) {
    return {
      from: from as Key,
      to: to as Key,
    };
  }

  const uci =
    typeof data.uci === "string"
      ? data.uci
      : typeof data.lan === "string"
        ? data.lan
        : null;

  if (uci && uci.length >= 4) {
    const uciFrom = uci.slice(0, 2);
    const uciTo = uci.slice(2, 4);

    if (isChessSquare(uciFrom) && isChessSquare(uciTo)) {
      return {
        from: uciFrom as Key,
        to: uciTo as Key,
      };
    }
  }

  return null;
};
