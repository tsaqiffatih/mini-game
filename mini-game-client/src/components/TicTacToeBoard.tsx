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
  const [isActive, setIsActive] = useState<boolean>(false);
  const [chatMessages, setChatMessages] = useState<
    Array<{ sender: string; message: string; timestamp: string }>
  >([]);
  const [playerMarkState, setPlayerMarkState] = useState<string>(playerMark);
  const [isChatOpen, setIsChatOpen] = useState<boolean>(false);
  const [hasNewMessage, setHasNewMessage] = useState<boolean>(false);

  const router = useRouter();

  const { sendMessage, lastMessage } = useGameWebSocket(roomId, playerId);

  const lastMessageRef = useRef<string | null>(null); // untuk melacak pesan terakhir

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
      const messageFromServer = JSON.parse(lastMessage.data);
      console.log(messageFromServer);
      

      if (messageFromServer.action === "TICTACTOE_GAME_STATE") {
        setBoard(messageFromServer.message.board);

        setTurn(messageFromServer.message.turn);
        setWinner(messageFromServer.message.winner);
        setIsActive(messageFromServer.message.is_active);
      }

      if (messageFromServer.action === "USER_LEFT_ROOM") {
        showAlert({
          title: "Player Left",
          text: "The other player has left the room.",
          icon: "info",
          confirmButtonText: "Ok",
        });
      }

      if (messageFromServer.action === "CONNECTED_ON_SERVER") {
        if (messageFromServer.sender.player_id !== playerId) {
          showAlert({
            title: "Player Joined",
            text: "The other player has joined the room.",
            icon: "info",
            confirmButtonText: "Ok",
          });
        }
      }

      if (messageFromServer.action === "MARK_UPDATE") {
        const marks = messageFromServer.message.marks;

        if (marks && marks[playerId]) {
          const newMark = marks[playerId];
          setPlayerMarkState(newMark);
          localStorage.setItem("playerMark", newMark);
        }
      }

      if (messageFromServer.action === "CHAT_MESSAGE") {
        const newMessage = {
          sender: messageFromServer.sender.player_id,
          message: messageFromServer.message,
          timestamp: messageFromServer.timestamp,
        };

        setChatMessages((prevMessages) => [...prevMessages, newMessage]);

        if (!isChatOpen) {
          setHasNewMessage(true);
          const audio = new Audio("/sounds/notification.mp3");
          audio.play();
        }
      }
    }
  }, [lastMessage, setIsActive, playerId, isChatOpen]);

  const handleCellClick = (row: number, col: number) => {
    if (!isActive) {
      alert("The game is not active yet!");
      return;
    }

    if (turn !== playerMarkState) {
      showErrorAlert("It's not your turn!");
      return;
    }

    const message = {
      action: "TICTACTOE_MOVE",
      message: { room_id: roomId, player_id: playerId, row, col },
      sender: { id: playerId },
    };
    sendMessage(JSON.stringify(message));
  };

  const handleSendMessage = (message: string) => {
    const chatMessage = {
      action: "CHAT_MESSAGE",
      message: message,
      sender: { player_id: playerId },
      time: new Date().toISOString(),
    };

    sendMessage(JSON.stringify(chatMessage));
  };

  const handleOpenChat = () => {
    setIsChatOpen(true);
    setHasNewMessage(false);
  };

  return (
    <div className="p-2 flex  items-center justify-center overflow-hidden">
      {roomId && !isActive && !winner && <Waiting roomId={roomId} />}
      {winner && winner !== "Draw" && (
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
      )}
      {isActive && (
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
