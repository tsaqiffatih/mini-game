"use client";

import {
  handleLeaveGameAlert,
  showAlert,
  showErrorAlert,
} from "@/utils/alerthelper";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import Waiting from "./Waiting";
import Board from "./Board";
import ChatOpened from "./ChatOpened";
import { useGameWebSocket } from "@/utils/gameWebsocket";

interface TicTacToeBoardProps {
  playerId: string;
  roomId: string;
  playerMark: string;
}

export default function TicTacToeBoard({
  playerId,
  roomId,
  playerMark,
}: TicTacToeBoardProps) {
  const [board, setBoard] = useState<string[][]>([
    ["", "", ""],
    ["", "", ""],
    ["", "", ""],
  ]);
  const [turn, setTurn] = useState<string>("");
  const [winner, setWinner] = useState<string>("");

  const [roomState, setRoomState] = useState<
    "WAITING" | "PLAYING" | "FINISHED" | "RESETTING"
  >("WAITING");

  const [chatMessages, setChatMessages] = useState<
    Array<{
      id: string;
      sender: string;
      playerMark?: string;
      message: string;
      timestamp: string;
    }>
  >([]);
  const [playerMarkState, setPlayerMarkState] = useState<string>(playerMark);
  const [isChatOpen, setIsChatOpen] = useState<boolean>(false);
  const [hasNewMessage, setHasNewMessage] = useState<boolean>(false);

  const router = useRouter();

  const { sendMessage, lastMessage } = useGameWebSocket(roomId, playerId);

  const lastMessageRef = useRef<string | null>(null); // untuk melacak pesan terakhir
  const isChatOpenRef = useRef(false);

  // useEffect for message
  useEffect(() => {
    isChatOpenRef.current = isChatOpen;
  }, [isChatOpen]);

  useEffect(() => {
    const handlePopState = () => {
      handleLeaveGameAlert(router);
    };

    window.history.pushState(null, "", window.location.href);
    window.addEventListener("popstate", handlePopState);

    return () => {
      window.removeEventListener("popstate", handlePopState);
    };
  }, [router]);

  useEffect(() => {
    if (lastMessage !== null && lastMessage.data !== lastMessageRef.current) {
      lastMessageRef.current = lastMessage.data;
      const msg = JSON.parse(lastMessage.data);

      switch (msg.type) {
        case "room_update":
        case "game_update": {
          const state = msg.payload?.room ?? msg.payload;

          console.log(state);
          

          if (!state) break;

          setRoomState(state.state);

          const game = state.tictactoe;

          if (game) {
            setBoard(state.tictactoe.board);
            setTurn(state.tictactoe.turn);
            setWinner(state.tictactoe.winner);
          }

          break;
        }

        case "player_joined":
          const joinedPlayerId = msg.payload?.data?.player?.player_id;

          if (playerId !== joinedPlayerId) {
            console.log(msg);
            const state = msg.payload?.room ?? msg.payload;
            setRoomState(state.state);

            showAlert({
              title: "Player Joined",
              text: "The other player has joined the room.",
              icon: "info",
              confirmButtonText: "Ok",
            });
          }

          break;

        case "player_left":
          const leftedPlayerId = msg.payload?.data?.player?.player_id;

          if (leftedPlayerId != playerId) {
            showAlert({
              title: "Player Lefted",
              text: "The other player has left the room.",
              icon: "info",
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
                new Date(a.timestamp).getTime() -
                new Date(b.timestamp).getTime(),
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
    }
  }, [lastMessage, playerId, isChatOpen]);

  const handleCellClick = (row: number, col: number) => {
    if (roomState != "PLAYING") {
      showErrorAlert("The game is not active yet!");
      return;
    }

    if (turn !== playerMarkState) return;

    if (board[row][col] !== "") return

    sendMessage(
      JSON.stringify({
        type: "TICTACTOE_MOVE",
        payload: {
          room_id: roomId,
          player_id: playerId,
          row,
          col,
        },
      }),
    );
  };

  const handleSendMessage = (message: string) => {
    sendMessage(
      JSON.stringify({
        type: "CHAT_SEND",
        payload: {
          message,
        },
      }),
    );
  };

  const handleOpenChat = () => {
    setIsChatOpen(true);
    setHasNewMessage(false);
  };

  return (
    <div className="p-2 flex  items-center justify-center overflow-hidden">
      {/* WAITING */}
      {roomState === "WAITING" && <Waiting roomId={roomId} />}

      {/* FINISHED */}
      {roomState === "FINISHED" && (
        <div className="flex flex-col items-center justify-center">
          {winner === "Draw" ? (
            <h2 className="text-xl text-yellow-500">Game Draw!</h2>
          ) : (
            <h2 className="text-xl text-green-500">Winner: {winner}</h2>
          )}
        </div>
      )}

      {/* RESETTING */}
      {roomState === "RESETTING" && (
        <div className="text-center">
          <h2 className="text-lg">Resetting game...</h2>
        </div>
      )}

      {/* {winner && winner !== "Draw" && (
        <div className="flex h-full items-center justify-center">
          <h2 className="text-2xl font-bold text-green-600">
            Winner: {winner}
          </h2>
        </div>
      )}

      {winner === "Draw" && (
        <div className="flex flex-col items-center justify-center space-y-4">
          <h2 className="text-2xl font-bold text-yellow-600 text-center">
            Game ended in a Draw!
          </h2>
          <p className="text-lg text-center">
            No more moves available, and no player has won.
          </p>
          <p className="text-lg text-center">Wait Until the game Restart</p>
        </div>
      )} */}

      {roomState === "PLAYING" && (
        <div className="flex flex-col md:flex-row overflow-hidden p-2 items-center justify-center space-x-0 md:space-x-10">
          <div className="flex p-5">
            <div className="flex flex-col items-center justify-center space-y-4">
              <h2 className="text-lg md:text-3xl text-center">
                Enjoy the Game, &quot;{playerId}&quot;
              </h2>
              <div className="flex flex-wrap justify-center space-x-2 text-sm md:text-xl">
                <h2>Your Mark: &quot;{playerMarkState}&quot;</h2>
                <span className="text-primary-content">|</span>
                <h2>Turn: &quot;{turn}&quot;</h2>
                <span className="text-primary-content">|</span>
                <h2>Room Id: &quot;{roomId}&quot;</h2>
              </div>
              <Board board={board} onCellClick={handleCellClick} />
            </div>
          </div>
          <div className="hidden md:block">
            <ChatOpened
              setOpenChatOpened={() => {}}
              userName={playerId}
              messages={chatMessages}
              onSendMessage={handleSendMessage}
            />
          </div>
          <div className="block md:hidden mt-2 relative">
            <button
              className="btn btn-primary btn-outline p-2"
              onClick={handleOpenChat}
            >
              Open Chat
              {hasNewMessage && (
                <span className="absolute top-0 right-0 badge badge-primary badge-xs transform translate-x-1/2 -translate-y-1/2"></span>
              )}
            </button>
          </div>
        </div>
      )}

      {isChatOpen && (
        <div className="fixed md:hidden inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
          <div className="bg-base-100 py-0 rounded-lg w-11/12">
            <ChatOpened
              setOpenChatOpened={() => setIsChatOpen(false)}
              userName={playerId}
              messages={chatMessages}
              onSendMessage={handleSendMessage}
            />
          </div>
        </div>
      )}
    </div>
  );
}
