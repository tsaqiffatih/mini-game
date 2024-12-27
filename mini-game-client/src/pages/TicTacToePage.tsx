import { useEffect, useState } from "react";
import Lobby from "../components/shared/Lobby";
import { showAlert, showErrorAlert } from "../utils/alerthelper";
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
    <div className=" overflow-hidden p-2 h-screen flex flex-col items-center justify-center">
      <button
        className="btn btn-primary text-sm btn-sm sm:btn-md sm:text-base btn-outline top-0 left-0 m-2 sm:m-4 absolute"
        onClick={() => {
          if (roomId) {
            showAlert({
              title: "Leave Game?",
              text: "Are you sure you want to leave the game? Your progress will be lost.",
              icon: "warning",
              showCancelButton: true,
              confirmButtonText: "Yes, leave",
              cancelButtonText: "No, stay",
            }).then((result) => {
              if (result.isConfirmed) {
                localStorage.removeItem("roomId");
                localStorage.removeItem("playerMark");
                navigate("/");
              }
            });
          } else {
            localStorage.removeItem("roomId");
            localStorage.removeItem("playerMark");
            navigate("/");
          }
        }}
      >
        <svg
          viewBox="0 0 1024 1024"
          className="icon w-3 h-3 mr-0 sm:w-5 sm:h-5"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            fill="currentColor"
            d="M224 480h640a32 32 0 110 64H224a32 32 0 010-64z"
          />
          <path
            fill="currentColor"
            d="M237.248 512l265.408 265.344a32 32 0 01-45.312 45.312l-288-288a32 32 0 010-45.312l288-288a32 32 0 1145.312 45.312L237.248 512z"
          />
        </svg>
        Home
      </button>
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
