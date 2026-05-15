"use client";

import { useEffect, useMemo, useRef, useState, useCallback } from "react";
import { Chess, Square } from "chess.js";
import { Chessboard } from "react-chessboard";
import { PromotionPieceOption } from "react-chessboard/dist/chessboard/types";
import { useRouter } from "next/navigation";

import { useGameWebSocket } from "@/utils/gameWebsocket";
import {
  showAlert,
  handleLeaveGameAlert,
  showErrorAlert,
} from "@/utils/alerthelper";
import ChatOpened from "@/components/ChatOpened";
import Waiting from "@/components/Waiting";

interface ChessBoardProps {
  roomId: string;
  playerId: string;
}

type RoomState = "WAITING" | "PLAYING" | "FINISHED" | "RESETTING";

export default function ChessBoard({ roomId, playerId }: ChessBoardProps) {
  const router = useRouter();
  const { sendMessage, lastMessage } = useGameWebSocket(roomId, playerId);

  /** =========================
   * AUTHORITATIVE GAME STATE
   * ========================= */

  const chessRef = useRef(new Chess());

  const [roomState, setRoomState] = useState<RoomState>("WAITING");

  const [fen, setFen] = useState(chessRef.current.fen());

  const [winner, setWinner] = useState("");

  const [turn, setTurn] = useState<"white" | "black">("white");

  const [playerMark, setPlayerMark] = useState<"white" | "black">("white");

  const [pgnMoves, setPgnMoves] = useState<string[]>([]);

  /** =========================
   * UI STATE
   * ========================= */

  const [chatMessages, setChatMessages] = useState<
    Array<{
      id: string;
      sender: string;
      playerMark?: string;
      message: string;
      timestamp: string;
    }>
  >([]);
  const [isChatOpen, setIsChatOpen] = useState(false);
  const [hasNewMessage, setHasNewMessage] = useState(false);

  const [lastMove, setLastMove] = useState<{
    from: string;
    to: string;
  } | null>(null);

  // const lastWSRef = useRef<string | null>(null);

  const latestStateVersionRef = useRef(0);

  const lastPlayedMoveIdRef = useRef<string | null>(null);

  const isChatOpenRef = useRef(false);

  /** ====== SOUND  ====== */

  const [audioUnlocked, setAudioUnlocked] = useState(false);

  const soundsRef = useRef({
    moveSelf: new Audio("/sounds/move-self.mp3"),
    moveOpponent: new Audio("/sounds/move-opponent.mp3"),
    capture: new Audio("/sounds/capture.mp3"),
    castle: new Audio("/sounds/castle.mp3"),
    check: new Audio("/sounds/check.mp3"),
    gameEnd: new Audio("/sounds/game-end.mp3"),
    illegal: new Audio("/sounds/illegal.mp3"),
    promote: new Audio("/sounds/promote.mp3"),
  });

  const playSound = useCallback(
    async (audio: HTMLAudioElement) => {
      console.log(audioUnlocked, "<<< audioUnlocked");
      if (!audioUnlocked) return;

      try {
        audio.currentTime = 0;
        await audio.play();
      } catch {}
    },
    [audioUnlocked],
  );

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

        console.log(msg);

        if (!state) break;

        /** ignore stale snapshots */
        if (
          typeof state.state_version === "number" &&
          state.state_version <= latestStateVersionRef.current
        ) {
          break;
        }

        latestStateVersionRef.current = state.state_version;

        setRoomState(state.state);

        /** PLAYER MARK */
        const me = state.players?.find((p: any) => p.player_id === playerId);

        if (me?.player_mark === "white" || me?.player_mark === "black") {
          setPlayerMark(me.player_mark);
        }

        /** CHESS STATE */
        const chessState = state.game?.chess ?? state.chess;

        if (chessState) {
          setFen(chessState.fen);

          setWinner(chessState.winner || "");

          setPgnMoves(chessState.pgn_moves ?? []);

          /** sync local parser */
          chessRef.current.load(chessState.fen);

          /** derive turn from FEN */
          setTurn(chessState.turn);

          const move = chessState.last_move;

          if (!move) break;

          /** prevent replaying same move sound */
          if (move.id === lastPlayedMoveIdRef.current) {
            break;
          }

          lastPlayedMoveIdRef.current = move.id;

          const isSelfMove = move?.actor?.player_id === playerId;
          console.log(move);

          if (move?.flags?.checkmate) {
            playSound(soundsRef.current.gameEnd);
          } else if (move?.flags?.check) {
            playSound(soundsRef.current.check);
          } else if (move?.flags?.castle) {
            playSound(soundsRef.current.castle);
          } else if (move?.flags?.capture) {
            playSound(soundsRef.current.capture);
          } else if (move?.flags?.promotion) {
            playSound(soundsRef.current.promote);
          } else {
            console.log("masuk else");

            playSound(
              isSelfMove
                ? soundsRef.current.moveSelf
                : soundsRef.current.moveOpponent,
            );
          }

          /** derive last move from PGN */
          const history = chessRef.current.history({
            verbose: true,
          });

          const latestMove = history[history.length - 1];

          if (latestMove) {
            setLastMove({
              from: latestMove.from,
              to: latestMove.to,
            });
          } else {
            setLastMove(null);
          }
        }

        break;

      case "chess_move_rejected":
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

      case "player_left":
        const leftPlayerId = msg.payload?.data?.player?.player_id;

        if (leftPlayerId && leftPlayerId !== playerId) {
          showAlert({
            title: "Player Left",
            text: `${leftPlayerId} left the room.`,
            icon: "warning",
            confirmButtonText: "Ok",
          });
        }

        break;

      case "chat_history":
        const messages = msg.payload?.messages ?? [];

        setChatMessages((prev) => {
          const map = new Map();

          [...prev, ...messages].forEach((m: any) => {
            map.set(m.id, {
              id: m.id,
              sender: m.player_id,
              playerMark: m.player_mark,
              message: m.message,
              timestamp: m.created_at,
            });
          });

          return Array.from(map.values()).sort(
            (a, b) =>
              new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
          );
        });

        break;

      case "chat_message": {
        const chat = msg.payload;

        if (!chat) break;

        setChatMessages((prev) => {
          const exists = prev.some((m) => m.id === chat.id);

          if (exists) return prev;

          return [
            ...prev,
            {
              id: chat.id,
              sender: chat.player_id,
              playerMark: chat.player_mark,
              message: chat.message,
              timestamp: chat.created_at,
            },
          ];
        });

        if (!isChatOpenRef.current) {
          setHasNewMessage(true);
        }

        break;
      }

      case "error":
        showErrorAlert(msg.payload?.message || "Something went wrong");
        break;

      default:
        break;
    }
  }, [lastMessage, playerId, playSound]);

  /** =========================
   * SEND MOVE
   * ========================= */
  const sendMove = useCallback(
    (from: Square, to: Square, promotion?: string) => {
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
    [sendMessage, roomState, turn, playerMark],
  );

  /** =========================
   * INIT SOUND
   * ========================= */

  useEffect(() => {
    const unlockAudio = () => {
      setAudioUnlocked(true);

      window.removeEventListener("pointerdown", unlockAudio);
    };

    window.addEventListener("pointerdown", unlockAudio);

    return () => {
      window.removeEventListener("pointerdown", unlockAudio);
    };
  }, []);

  /** =========================
   * DROP HANDLER
   * ========================= */
  const onDrop = useCallback(
    (sourceSquare: Square, targetSquare: Square) => {
      const piece = chessRef.current.get(sourceSquare);

      if (!piece) return false;

      /** prevent moving opponent piece */
      if (
        (piece.color === "w" && playerMark !== "white") ||
        (piece.color === "b" && playerMark !== "black")
      ) {
        return false;
      }

      /** promotion detection */
      const needsPromotion =
        piece.type === "p" &&
        ((piece.color === "w" && targetSquare.endsWith("8")) ||
          (piece.color === "b" && targetSquare.endsWith("1")));

      if (needsPromotion) {
        return false;
      }

      return sendMove(sourceSquare, targetSquare);
    },
    [playerMark, sendMove],
  );

  /** =========================
   * PROMOTION
   * ========================= */
  const onPromotionPieceSelect = useCallback(
    (piece?: PromotionPieceOption, from?: Square, to?: Square) => {
      if (!piece || !from || !to) {
        return false;
      }

      const promotion = piece.toLowerCase().replace(/^[wb]/, "");

      return sendMove(from, to, promotion);
    },
    [sendMove],
  );

  /** =========================
   * BOARD ORIENTATION
   * ========================= */
  const boardOrientation = useMemo(() => {
    return playerMark === "black" ? "black" : "white";
  }, [playerMark]);

  /** =========================
   * LAST MOVE HIGHLIGHT
   * ========================= */
  const squareStyles = useMemo(() => {
    if (!lastMove) return {};

    return {
      [lastMove.from]: {
        backgroundColor: "rgba(255,255,0,0.5)",
      },
      [lastMove.to]: {
        backgroundColor: "rgba(255,255,0,0.5)",
      },
    };
  }, [lastMove]);

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

  /** =========================
   * WAITING
   * ========================= */
  if (roomState === "WAITING") {
    return <Waiting roomId={roomId} />;
  }

  /** =========================
   * FINISHED
   * ========================= */
  if (roomState === "FINISHED") {
    return (
      <div className="flex flex-col items-center justify-center gap-4 min-h-dvh">
        <div className="text-center">
          {winner.toLowerCase() === "draw" ? (
            <h2 className="text-2xl font-bold text-yellow-500">Draw Game</h2>
          ) : (
            <h2 className="text-2xl font-bold text-green-500">
              Winner: {winner}
            </h2>
          )}

          <p className="mt-2 text-sm opacity-70">
            Waiting for automatic reset...
          </p>
        </div>
      </div>
    );
  }

  /** =========================
   * RESETTING
   * ========================= */
  if (roomState === "RESETTING") {
    return (
      <div className="flex flex-col items-center justify-center gap-4 min-h-dvh">
        <div className="text-center">
          <h2 className="text-xl font-bold">Resetting Game...</h2>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full flex flex-col gap-4 items-center justify-center pt-28 md:pt-2 md:overflow-hidden md:h-screen">
      <div className="text-center space-y-1">
        <h2 className="text-xl md:text-2xl font-bold">Chess Match</h2>

        <div className="flex flex-wrap items-center justify-center gap-2 text-sm md:text-base">
          <span>You: &quot;{playerId}&quot;</span>

          <span>|</span>

          <span>Mark: &quot;{playerMark}&quot;</span>

          <span>|</span>

          <span>Turn: &quot;{turn}&quot;</span>

          <span>|</span>

          <span>Room: &quot;{roomId}&quot;</span>
        </div>
      </div>
      <div className="flex flex-col md:flex-row gap-4 items-start justify-center">
        <div className="hidden md:block max-h-[600px] border rounded p-2 text-sm md:text-lg md:h-[600px]">
          <h3 className="font-bold mb-2 border-b ">Move History</h3>

          <div className="max-h-[600px] overflow-y-auto w-52 md:h-[600px]">
            <div className=" text-sm md:text-base font-mono">
              {Array.from(
                { length: Math.ceil(pgnMoves.length / 2) },
                (_, i) => {
                  const whiteMove = pgnMoves[i * 2];
                  const blackMove = pgnMoves[i * 2 + 1];

                  return (
                    <div
                      key={i}
                      className={`grid grid-cols-[30px_1fr_auto_1fr] gap-2 border-b border-base-300 ${i % 2 === 0 ? "bg-black/20" : "bg-neutral/40"}`}
                    >
                      <span className="font-bold">{i + 1}.</span>

                      <span>{whiteMove}</span>

                      <span>|</span>

                      <span>{blackMove ?? "-"}</span>
                    </div>
                  );
                },
              )}
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-4 lg:flex-row ">
          <div className="w-full max-w-[600px] mx-0 p-0 md:mx-auto">
            <Chessboard
              position={fen}
              boardOrientation={boardOrientation}
              onPieceDrop={onDrop}
              onPromotionPieceSelect={onPromotionPieceSelect}
              customSquareStyles={squareStyles}
              promotionDialogVariant="modal"
              boardWidth={
                typeof window !== "undefined"
                  ? Math.min(window.innerWidth - 32, 600)
                  : 600
              }
            />
          </div>
        </div>

        {/* DESKTOP CHAT */}
        <div className="hidden md:block w-72">
          <ChatOpened
            setOpenChatOpened={() => setIsChatOpen(false)}
            userName={playerId}
            messages={chatMessages}
            onSendMessage={sendChat}
          />
        </div>

        {/* MOBILE CHAT BUTTON */}
        <div className="md:hidden relative">
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

        {/* MOBILE MOVE HISTORY */}
        <div className="md:hidden max-h-[600px] border rounded p-2 text-sm md:text-lg md:h-[600px]">
          <h3 className="font-bold mb-2 border-b ">Move History</h3>

          <div className="max-h-[400px] overflow-y-auto w-80">
            <div className=" text-sm md:text-base font-mono">
              {Array.from(
                { length: Math.ceil(pgnMoves.length / 2) },
                (_, i) => {
                  const whiteMove = pgnMoves[i * 2];
                  const blackMove = pgnMoves[i * 2 + 1];

                  return (
                    <div
                      key={i}
                      className={`grid grid-cols-[30px_1fr_auto_1fr] gap-2 border-b border-base-300 ${i % 2 === 0 ? "bg-black/20" : "bg-neutral/40"}`}
                    >
                      <span className="font-bold">{i + 1}.</span>

                      <span>{whiteMove}</span>

                      <span>|</span>

                      <span>{blackMove ?? "-"}</span>
                    </div>
                  );
                },
              )}
            </div>
          </div>
        </div>

        {/* MOBILE CHAT MODAL */}
        {isChatOpen && (
          <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center">
            <div className="bg-base-100 rounded-lg w-11/12">
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
    </div>
  );
}
