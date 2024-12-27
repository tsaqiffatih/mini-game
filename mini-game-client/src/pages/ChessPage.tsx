import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { showErrorAlert } from "../utils/alerthelper";
import Lobby from "../components/shared/Lobby";
import ChessBoard from "../components/chess/ChessBoard";

const ChessPage: React.FC = () => {
  const [playerId, setPlayerId] = useState<string>("");
  const [roomId, setRoomId] = useState<string>("");
  const [playerMark, setPlayerMark] = useState<string>("");

  const navigate = useNavigate();

  const handleRoomIdGenerated = (roomId: string, playerMark: string) => {
    setRoomId(roomId);
    setPlayerMark(playerMark);
  };

  useEffect(() => {
    const storedPlayerId = localStorage.getItem("playerId") || "";
    if (storedPlayerId === "") {
      showErrorAlert("Player ID not found. Please register first.");
      navigate("/");
    }
    const storedRoomId = localStorage.getItem("roomId") || "";
    const storedPlayerMark = localStorage.getItem("playerMark") || "";
    setPlayerMark(storedPlayerMark);
    setRoomId(storedRoomId);
    setPlayerId(storedPlayerId);
  }, [navigate]);

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
        />
      )}
    </div>
  );
};

export default ChessPage;
