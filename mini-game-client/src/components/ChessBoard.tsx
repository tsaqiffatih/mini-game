"use client";

import { useEffect, useMemo, useRef, useState, useCallback } from "react";
import { Chess } from "chess.js";
import type { Square } from "chess.js";
import { Chessground } from "chessground";
import type { Api } from "chessground/api";
import type { Color as CgColor, Key, MoveMetadata } from "chessground/types";
import { useRouter } from "next/navigation";

import Waiting from "./Waiting";
import ChatOpened from "./ChatOpened";
import { useGameWebSocket } from "@/utils/gameWebsocket";
import {
  showAlert,
  handleLeaveGameAlert,
  showErrorAlert,
} from "@/utils/alerthelper";
import ChessMoveHistory from "./ChessMoveHistory";
import { useChessSounds } from "@/utils/useChessSounds";
import { handleGameChessUpdate } from "@/utils/handleGameChessUpdate";
import { handleChatChessMessage } from "@/utils/handleChatChessMessage";
import { handleChatChessHistory } from "@/utils/handleChatChessHistory";
import {
  getMoveDests,
  getPieceColor,
  getTurnFromFen,
  isChessSquare,
  LastMove,
  PlayerMark,
  isPlayerMark,
  getLastMoveFromSnapshot,
} from "@/utils/chessUtils";
import Image from "next/image";

interface ChessBoardProps {
  roomId: string;
  playerId: string;
}

type RoomState = "WAITING" | "PLAYING" | "FINISHED" | "RESETTING";

type ChatMessage = {
  id: string;
  sender: string;
  playerMark?: string;
  message: string;
  timestamp: string;
};

type ChatSnapshot = {
  id: string;
  player_id: string;
  player_mark?: string;
  message: string;
  created_at: string;
};

const PIECE_VALUES: Record<string, number> = {
  pawn: 1,
  knight: 3,
  bishop: 3,
  rook: 5,
  queen: 9,
  king: 0,
};

export type CapturedPiece = {
  type: string;
  color: string;
};

export default function ChessBoard({ roomId, playerId }: ChessBoardProps) {
  const router = useRouter();
  const { sendMessage, lastMessage, isReconnecting } = useGameWebSocket(
    "chess",
    roomId,
    playerId,
  );

  const {
    playMoveSelf,
    playMoveOpponent,
    playCapture,
    playCastle,
    playCheck,
    playGameEnd,
    playIllegal,
    playPromote,
  } = useChessSounds();

  /** =========================
   * AUTHORITATIVE GAME STATE
   * ========================= */

  const chessRef = useRef(new Chess());

  const boardContainerRef = useRef<HTMLDivElement | null>(null);

  const chessgroundRef = useRef<Api | null>(null);

  const [roomState, setRoomState] = useState<RoomState>("WAITING");

  const [fen, setFen] = useState(chessRef.current.fen());

  const [winner, setWinner] = useState("");

  const [playerMark, setPlayerMark] = useState<PlayerMark>("white");

  const [pgnMoves, setPgnMoves] = useState<string[]>([]);

  /** =========================
   * UI STATE
   * ========================= */

  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [isChatOpen, setIsChatOpen] = useState(false);
  const [hasNewMessage, setHasNewMessage] = useState(false);

  const [opponentReconnecting, setOpponentReconnecting] = useState(false);

  const [lastMove, setLastMove] = useState<LastMove | null>(null);

  const latestStateVersionRef = useRef(0);

  const lastPlayedMoveIdRef = useRef<string | null>(null);

  const isChatOpenRef = useRef(false);

  const [capturedPieces, setCapturedPieces] = useState<{
    white: CapturedPiece[];
    black: CapturedPiece[];
  }>({
    white: [],
    black: [],
  });

  const PIECE_TYPE_MAP: Record<string, string> = {
    pawn: "P",
    rook: "R",
    knight: "N",
    bishop: "B",
    queen: "Q",
    king: "K",
  };

  const getCapturedPieceImage = (piece: { type: string; color: string }) => {
    const colorPrefix = piece.color === "white" ? "w" : "b";

    const type = PIECE_TYPE_MAP[piece.type];

    return `/chess/cburnett/${colorPrefix}${type}.svg`;
  };

  const turn = useMemo(() => getTurnFromFen(fen), [fen]);

  /** ====== LEAVE GAME ALERT ====== */
  useEffect(() => {
    const handler = () => handleLeaveGameAlert(router);
    window.history.pushState(null, "", window.location.href);
    window.addEventListener("popstate", handler);
    return () => window.removeEventListener("popstate", handler);
  }, [router]);

  /** ====== WS HANDLER ====== */

  useEffect(() => {
    isChatOpenRef.current = isChatOpen;
  }, [isChatOpen]);

  useEffect(() => {
    if (roomState === "RESETTING") {
      setCapturedPieces({
        white: [],
        black: [],
      });
    }
  }, [roomState]);

  useEffect(() => {
    if (!lastMessage) return;

    let msg;

    try {
      msg = JSON.parse(lastMessage.data);
    } catch {
      return;
    }

    switch (msg.type) {
      case "room_update":
      case "game_update":
        const state = msg.payload?.room ?? msg.payload;

        handleGameChessUpdate({
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
        });

        break;

      case "chess_move_rejected":
        chessgroundRef.current?.set({
          fen: chessRef.current.fen(),
          turnColor: getTurnFromFen(chessRef.current.fen()),
          movable: {
            dests: getMoveDests(chessRef.current),
          },
        });
        playIllegal();
        showErrorAlert(msg.payload?.message || "Illegal move");
        break;

      case "player_joined":
        const joinedPlayerId = msg.payload?.data?.player?.player_id;

        if (joinedPlayerId && joinedPlayerId !== playerId) {
          const state = msg.payload?.room ?? msg.payload;
          setRoomState(state.state);
          showAlert({
            title: "Player Joined",
            text: `${joinedPlayerId} joined the room.`,
            icon: "info",
            confirmButtonText: "Ok",
          });
        }

        break;

      case "player_reconnecting":
        const reconnectingPlayerId = msg.payload?.data?.player?.player_id;

        if (reconnectingPlayerId !== playerId) {
          setOpponentReconnecting(true);
        }
        break;

      case "player_reconnected":
        const reconnectedPlayerId = msg.payload?.data?.player?.player_id;

        if (reconnectedPlayerId !== playerId) {
          setOpponentReconnecting(false);
        }
        break;

      case "player_left":
        const leftPlayerId = msg.payload?.data?.player?.player_id;

        if (leftPlayerId && leftPlayerId !== playerId) {
          setOpponentReconnecting(false);

          showAlert({
            title: "Player Left",
            text: `${leftPlayerId} left the room.`,
            icon: "warning",
            confirmButtonText: "Ok",
          });
        }

        break;

      case "chat_history":
        const messages: ChatSnapshot[] = Array.isArray(msg.payload?.messages)
          ? msg.payload.messages
          : [];

        handleChatChessHistory({
          messages,
          setChatMessages,
        });

        break;

      case "chat_message": {
        handleChatChessMessage({
          chat: msg.payload,
          setChatMessages,
          isChatOpenRef,
          setHasNewMessage,
        });

        break;
      }

      case "error":
        showErrorAlert(msg.payload?.message || "Something went wrong");
        break;

      default:
        break;
    }
  }, [
    lastMessage,
    playerId,
    playCapture,
    playCastle,
    playCheck,
    playGameEnd,
    playIllegal,
    playMoveOpponent,
    playMoveSelf,
    playPromote,
  ]);

  /** =========================
   * SEND MOVE
   * ========================= */
  const sendMove = useCallback(
    (from: Square, to: Square, promotion?: string): boolean => {
      if (opponentReconnecting) {
        showErrorAlert("Opponent reconnecting...");
        return false;
      }

      if (roomState !== "PLAYING") {
        showErrorAlert("Game is not active");
        return false;
      }

      /** enforce local turn */
      if (turn !== playerMark) {
        showErrorAlert("It's not your turn");
        return false;
      }

      sendMessage(
        JSON.stringify({
          type: "CHESS_MOVE",
          payload: {
            from,
            to,
            promotion,
          },
        }),
      );

      return true;
    },
    [sendMessage, roomState, turn, playerMark, opponentReconnecting],
  );

  const handleMove = useCallback(
    (from: Key, to: Key): boolean => {
      if (!isChessSquare(from) || !isChessSquare(to)) {
        return false;
      }

      const piece = chessRef.current.get(from);

      if (!piece) return false;

      if (getPieceColor(piece.color) !== playerMark) {
        return false;
      }

      if (turn !== playerMark) {
        showErrorAlert("It's not your turn");
        return false;
      }

      const needsPromotion =
        piece.type === "p" &&
        ((piece.color === "w" && to.endsWith("8")) ||
          (piece.color === "b" && to.endsWith("1")));

      const legalMoves = chessRef.current.moves({ verbose: true });
      const isLegalMove = legalMoves.some((move) => {
        return (
          move.from === from &&
          move.to === to &&
          (!needsPromotion || move.promotion === "q")
        );
      });

      if (!isLegalMove) {
        playIllegal();
        return false;
      }

      if (needsPromotion) {
        return sendMove(from, to, "q");
      }

      return sendMove(from, to);
    },
    [playerMark, sendMove, turn, playIllegal],
  );

  const handleChessgroundMove = useCallback(
    (from: Key, to: Key, metadata: MoveMetadata): void => {
      if (metadata.premove && turn !== playerMark) {
        return;
      }

      const moveAccepted = handleMove(from, to);

      if (!moveAccepted) {
        chessgroundRef.current?.set({
          fen: chessRef.current.fen(),
          turnColor: turn,
          movable: {
            dests: getMoveDests(chessRef.current),
          },
        });
      }
    },
    [handleMove, playerMark, turn],
  );

  /** =========================
   * BOARD ORIENTATION
   * ========================= */
  const boardOrientation = useMemo<CgColor>(() => {
    return playerMark === "black" ? "black" : "white";
  }, [playerMark]);

  /** =========================
   * INIT CHESSGROUND
   * ========================= */

  useEffect(() => {
    if (!boardContainerRef.current) return;

    if (chessgroundRef.current) return;

    chessgroundRef.current = Chessground(boardContainerRef.current, {
      fen,
      coordinates: true,
      orientation: boardOrientation,
      turnColor: turn,

      highlight: {
        lastMove: true,
        check: true,
      },

      animation: {
        enabled: true,
        duration: 250,
      },

      autoCastle: true,
      blockTouchScroll: true,
      disableContextMenu: true,

      movable: {
        free: false,
        color: roomState === "PLAYING" ? playerMark : undefined,
        dests: getMoveDests(chessRef.current),
        showDests: true,
        rookCastle: true,

        events: {
          after: handleChessgroundMove,
        },
      },

      premovable: {
        enabled: true,
        showDests: true,
        castle: true,
      },

      drawable: {
        enabled: true,
        visible: true,
        defaultSnapToValidMove: true,
        eraseOnClick: false,
      },

      draggable: {
        enabled: false,
        showGhost: false,
      },

      selectable: {
        enabled: true,
      },
    });

    return () => {
      chessgroundRef.current?.destroy();
      chessgroundRef.current = null;
    };
  }, [
    boardOrientation,
    fen,
    handleChessgroundMove,
    playerMark,
    roomState,
    turn,
  ]);

  /** ====== CHAT SEND ====== */
  const sendChat = (text: string) => {
    sendMessage(
      JSON.stringify({
        type: "CHAT_SEND",
        payload: {
          message: text,
        },
      }),
    );
  };

  useEffect(() => {
    if (!chessgroundRef.current) return;

    const legalDests = getMoveDests(chessRef.current);

    chessgroundRef.current.set({
      fen,

      orientation: boardOrientation,
      turnColor: turn,
      check: chessRef.current.isCheck() ? turn : false,

      lastMove: lastMove
        ? [lastMove.from as Key, lastMove.to as Key]
        : undefined,

      movable: {
        free: false,
        color:
          roomState === "PLAYING" && !opponentReconnecting
            ? playerMark
            : undefined,
        dests: legalDests,
        showDests: true,
        rookCastle: true,
        events: {
          after: handleChessgroundMove,
        },
      },

      premovable: {
        enabled: roomState === "PLAYING",
        showDests: true,
        castle: true,
      },

      drawable: {
        enabled: true,
        visible: true,
        defaultSnapToValidMove: true,
        eraseOnClick: false,
      },

      draggable: {
        enabled: roomState === "PLAYING",
        showGhost: true,
      },

      selectable: {
        enabled: roomState === "PLAYING",
      },
    });

    if (roomState === "PLAYING" && turn === playerMark) {
      chessgroundRef.current.playPremove();
    }
  }, [
    boardOrientation,
    fen,
    handleChessgroundMove,
    lastMove,
    playerMark,
    roomState,
    turn,
    opponentReconnecting,
  ]);

  const whiteScore = useMemo(() => {
    return capturedPieces.white.reduce((total, piece) => {
      return total + PIECE_VALUES[piece.type];
    }, 0);
  }, [capturedPieces.white]);

  const blackScore = useMemo(() => {
    return capturedPieces.black.reduce((total, piece) => {
      return total + PIECE_VALUES[piece.type];
    }, 0);
  }, [capturedPieces.black]);

  const whiteAdvantage = whiteScore - blackScore;
  const blackAdvantage = blackScore - whiteScore;

  /** =========================
   * WAITING
   * ========================= */
  if (roomState === "WAITING") {
    return <Waiting roomId={roomId} />;
  }

  return (
    <div className="w-full flex flex-col gap-4 items-center justify-start pb-2 lg:pb-0 pt-16 lg:pt-2 overflow-x-hidden lg:overflow-hidden lg:h-screen">
      {/* HEADER */}
      <div className="text-center space-y-1">
        <h2 className="text-xl lg:text-2xl font-bold">Chess Match</h2>

        <div className="flex flex-wrap items-center justify-center gap-2 text-sm lg:text-base">
          <span>You: &quot;{playerId}&quot;</span>

          <span>|</span>

          <span>Mark: &quot;{playerMark}&quot;</span>

          <span>|</span>

          <span>Turn: &quot;{turn}&quot;</span>

          <span>|</span>

          <span>Room: &quot;{roomId}&quot;</span>
        </div>
      </div>

      {/* MAIN LAYOUT */}
      <div className="flex flex-col min-[960px]:flex-row gap-4 items-center lg:items-start justify-center w-full">
        {/* DESKTOP MOVE HISTORY */}
        <ChessMoveHistory
          pgnMoves={pgnMoves}
          containerClassName="hidden min-[960px]:block max-h-[600px] lg:h-[600px]"
          scrollClassName="max-h-[600px] w-52 lg:h-[540px]"
        />

        {/* BOARD SECTION */}
        <div className="aspect-square w-[98vw] sm:w-[92vw] md:w-[80vw] lg:w-[58vw] xl:w-[520px] max-w-[900px]">
          {/* TOP PLAYER */}
          <div className="flex items-center justify-between px-2 py-2 bg-base-200 rounded-t-sm border-b">
            <div className="flex items-center gap-2">
              {/* AVATAR */}
              <div className="w-10 h-10 rounded overflow-hidden bg-base-300 flex items-center justify-center">
                <Image
                  src="chess/user-image/user-image.svg"
                  alt="avatar"
                  width={40}
                  height={40}
                />
              </div>

              {/* USER INFO */}
              <div className="flex flex-col">
                <span className="font-semibold text-sm lg:text-base">
                  Opponent
                </span>

                {/* CAPTURED PIECES */}
                <div className="flex items-center gap-[2px] min-h-[20px]">
                  {capturedPieces.white.map((piece, index) => (
                    <Image
                      key={`${piece.type}-${index}`}
                      src={getCapturedPieceImage(piece)}
                      alt={piece.type}
                      width={18}
                      height={18}
                    />
                  ))}

                  {/* SCORE */}
                  {whiteAdvantage > 0 && (
                    <span className="text-xs font-bold ml-1 text-success">
                      +{whiteAdvantage}
                    </span>
                  )}
                </div>
              </div>
            </div>
            {/* CONNECTION STATUS */}
            <div className="flex items-center gap-2">
              <div
                className={`w-2 h-2 rounded-full transition-all duration-300
                  ${
                    opponentReconnecting
                      ? "bg-warning animate-pulse"
                      : "bg-success"
                  }`}
              />

              <span
                className={`text-xs lg:text-sm font-medium transition-colors${
                  opponentReconnecting ? "text-warning" : "text-success"
                }`}
              >
                {opponentReconnecting ? "Reconnecting..." : "Online"}
              </span>
            </div>
          </div>

          {/* BOARD */}
          <div className="aspect-square w-full">
            <div ref={boardContainerRef} className="cg-wrap w-full h-full" />
          </div>

          {/* BOTTOM PLAYER */}
          <div className="flex items-center justify-between px-2 py-2 bg-base-200 rounded-b-md border-t">
            <div className="flex items-center gap-2">
              {/* AVATAR */}
              <div className="w-10 h-10 rounded overflow-hidden bg-base-300 flex items-center justify-center">
                <Image
                  src="chess/user-image/user-image.svg"
                  alt="avatar"
                  width={40}
                  height={40}
                />
              </div>

              {/* USER INFO */}
              <div className="flex flex-col">
                <span className="font-semibold text-sm lg:text-base">
                  {playerId}
                </span>

                {/* CAPTURED PIECES */}
                <div className="flex items-center gap-[2px] min-h-[20px]">
                  {capturedPieces.black.map((piece, index) => (
                    <Image
                      key={`${piece.type}-${index}`}
                      src={getCapturedPieceImage(piece)}
                      alt={piece.type}
                      width={18}
                      height={18}
                    />
                  ))}

                  {blackAdvantage > 0 && (
                    <span className="text-xs font-bold ml-1 text-success">
                      +{blackAdvantage}
                    </span>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* DESKTOP CHAT */}
        <div className="hidden min-[960px]:block w-72">
          <ChatOpened
            setOpenChatOpened={() => setIsChatOpen(false)}
            userName={playerId}
            messages={chatMessages}
            onSendMessage={sendChat}
          />
        </div>

        {/* MOBILE/TABLET CHAT BUTTON */}
        <div className="min-[960px]:hidden relative">
          <button
            className="btn btn-outline"
            onClick={() => {
              setIsChatOpen(true);
              setHasNewMessage(false);
            }}
          >
            Open Chat
            {hasNewMessage && (
              <span className="absolute top-0 right-0 badge badge-primary badge-xs transform translate-x-1/2 -translate-y-1/2"></span>
            )}
          </button>
        </div>

        {/* MOBILE/TABLET MOVE HISTORY */}
        <ChessMoveHistory
          pgnMoves={pgnMoves}
          containerClassName="min-[960px]:hidden max-h-[600px]"
          scrollClassName="max-h-[400px] w-[92vw] w-[98vw] sm:w-[92vw] md:w-[80vw] lg:w-[58vw] h-[400px]"
        />

        {/* MOBILE/TABLET CHAT MODAL */}
        {isChatOpen && (
          <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center">
            <div className="bg-base-100 rounded-lg w-11/12 md:w-[700px]">
              <ChatOpened
                userName={playerId}
                messages={chatMessages}
                onSendMessage={sendChat}
                setOpenChatOpened={() => setIsChatOpen(false)}
              />
            </div>
          </div>
        )}
      </div>
      {/* RECONNECTING MODAL */}
      {isReconnecting && (
        <div className="fixed inset-0 z-[999] flex items-center justify-center bg-black/70">
          <div className="bg-transparent rounded-xl p-6 shadow-xl flex flex-col items-center space-y-4">
            <span className="loading loading-spinner loading-lg"></span>

            <div className="text-center">
              <h2 className="text-xl font-bold">Reconnecting...</h2>
              <p className="text-sm opacity-70">
                Trying to reconnect to the game server
              </p>
            </div>
          </div>
        </div>
      )}

      {/* FINISHED MODAL */}
      {roomState === "FINISHED" && (
        <div className="fixed inset-0 z-[997] flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <div className="bg-base-100 rounded-2xl p-8 shadow-2xl flex flex-col items-center space-y-4">
            {winner.toLowerCase() === "draw" ? (
              <>
                <h2 className="text-3xl font-bold text-yellow-500">
                  Draw Game
                </h2>

                <p className="text-sm opacity-70 text-center">
                  Nobody wins this match
                </p>
              </>
            ) : (
              <>
                <h2 className="text-3xl font-bold text-green-500">
                  Winner: {winner}
                </h2>

                <p className="text-sm opacity-70 text-center">
                  Waiting for automatic reset...
                </p>
              </>
            )}
          </div>
        </div>
      )}

      {/* RESETTING MODAL */}
      {roomState === "RESETTING" && (
        <div className="fixed inset-0 z-[997] flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <div className="bg-base-100 rounded-2xl p-8 shadow-2xl flex flex-col items-center space-y-4">
            <span className="loading loading-spinner loading-lg"></span>

            <div className="text-center">
              <h2 className="text-2xl font-bold">Resetting Game...</h2>

              <p className="text-sm opacity-70">Preparing next round</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
