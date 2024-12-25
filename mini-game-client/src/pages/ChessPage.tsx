import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { showErrorAlert } from "../utils/alerthelper";
import Lobby from "../components/shared/Lobby";

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
    <div>
      {!roomId ? (
        <Lobby
          playerId={playerId}
          onRoomIdGenerated={handleRoomIdGenerated}
          gameType="chess"
        />
      ) : (
        <div>
          <h1>Chess Board</h1>
          <h2>Player ID: {playerId}</h2>
          <h2>Room ID: {roomId}</h2>
          <h2>Player Mark: {playerMark}</h2>
        </div>
      )}
    </div>
  );
};

export default ChessPage;
