import { useEffect, useState } from "react";
import Lobby from "../components/shared/Lobby";
import { showErrorAlert } from "../utils/alerthelper";
import { useNavigate } from "react-router-dom";
import TicTacToeBoard from "../components/tictactoe/TicTacToeBoard";

const TicTacToePage: React.FC = () => {
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
    <div className="">
      {!roomId ? (
        <Lobby
          playerId={playerId}
          onRoomIdGenerated={handleRoomIdGenerated}
          gameType="tictactoe"
        />
      ) : (
        <TicTacToeBoard
          playerId={playerId}
          roomId={roomId}
          playerMark={playerMark}
        />
      )}
    </div>
  );
};

export default TicTacToePage;
