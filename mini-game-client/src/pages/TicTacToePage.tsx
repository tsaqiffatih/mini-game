import { useEffect, useState } from "react";
import Lobby from "../components/shared/Lobby";
import { showErrorAlert } from "../utils/alerthelper";
import { useNavigate } from "react-router-dom";
import TicTacToeBoard from "../components/tictactoe/TicTacToeBoard";

// 1. apakah roomId tidak ada? jika tidak ada maka tampilkan <Lobby />
// 2. apakah roomId ada dan isActive false ? jika iya maka tampilkan <Waiting/>
// 3. setelah nya tampilkan =>
//    3.1. apakah winner ada dan winner bukan "Draw" ? jika iya maka tampilkan pemenangnya
//    3.2. apakah winner ada dan winner "Draw" ? jika iya maka tampilkan Draw
//    3.3. apakah isActive true dan winner tidak ada ? jika iya maka tampilkan game board
//

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
