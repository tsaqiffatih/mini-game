'use client'

import ChessBoard from "@/components/ChessBoard";
import Lobby from "@/components/Lobby";
import { showErrorAlert } from "@/utils/alerthelper";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";


const ChessPage: React.FC = () => {
  const [playerId, setPlayerId] = useState<string>("");
  const [roomId, setRoomId] = useState<string>("");
  const [playerMark, setPlayerMark] = useState<string>("");
  const [initialState, setInitialState] = useState<string>("");

  const router = useRouter()

  const handleRoomIdGenerated = (roomId: string, playerMark: string, initialState: string) => {
    setRoomId(roomId);
    setPlayerMark(playerMark);
    setInitialState(initialState);
  };

  useEffect(() => {
    const storedPlayerId = localStorage.getItem("playerId") || "";
    if (storedPlayerId === "") {
      showErrorAlert("Player ID not found. Please register first.");
      router.push("/");
    }
    const storedRoomId = localStorage.getItem("roomId") || "";
    const storedPlayerMark = localStorage.getItem("playerMark") || "";
    setPlayerMark(storedPlayerMark);
    setRoomId(storedRoomId);
    setPlayerId(storedPlayerId);
  }, [router]);

  return (
    <div className=" overflow-hidden p-2 h-screen flex flex-col items-center justify-center">
      {!roomId ? (
        <Lobby
          playerId={playerId}
          onRoomIdGenerated={handleRoomIdGenerated}
          gameType="chess"
        />
      ) : (
        <ChessBoard
          roomId={roomId}
          playerId={playerId}
          playerMark={playerMark}
          initialState={initialState}
        />
      )}
    </div>
  );
};

export default ChessPage;
