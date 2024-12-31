/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState } from "react";
import { showErrorAlert } from "../../utils/alerthelper";
import axios from "axios";

interface LobbyProps {
  gameType: string;
  playerId: string;
  onRoomIdGenerated: (
    roomId: string,
    playerMark: string,
    initialState?: any
  ) => void;
}

const backendUrl = import.meta.env.VITE_HTTP_BACKEND_URL

export default function Lobby({
  gameType,
  playerId,
  onRoomIdGenerated,
}: LobbyProps) {
  const [isLoadingNewGame, setIsLoadingNewGame] = useState(false);
  const [isLoadingJoinGame, setIsLoadingJoinGame] = useState(false);
  const [roomId, setRoomId] = useState("");

  const handleNewGame = async () => {
    setIsLoadingNewGame(true);
    try {
      const { data } = await axios.post(`${backendUrl}room/create`, {
        game_type: gameType,
        player_id: playerId,
      });

      if (data.success) {
        const newRoomId = data.data.room.room_id;
        const playerMark = data.data.player_mark;
        const initialState = data.data.room.game_state.data;

        localStorage.setItem("roomId", newRoomId);
        localStorage.setItem("playerMark", playerMark);

        setTimeout(() => {
          onRoomIdGenerated(newRoomId, playerMark, initialState);
          setIsLoadingNewGame(false);
        }, 2000);
      }
    } catch (error: any) {
      if (error.response.data.message === "Player not found") {
        showErrorAlert("Player not found. Please register first.");
        localStorage.removeItem("playerId");
        window.location.reload();
      } else if (error.response.data.message === "Invalid request") {
        showErrorAlert(
          "Invalid request. Please check your input and try again."
        );
      } else if (
        error.response.data.message === "RoomID and GameType are required"
      ) {
        showErrorAlert("GameType is required. Please select a game type.");
      } else {
        showErrorAlert(
          error.response.data.message ||
            "An error occurred while creating the room. Please try again."
        );
      }
      setIsLoadingNewGame(false);
    }
  };

  const handleJoinRoom = async () => {
    setIsLoadingJoinGame(true);
    try {
      const { data } = await axios.post(`${backendUrl}room/join`, {
        room_id: roomId,
        player_id: playerId,
        game_type: gameType,
      });

      if (data.success) {
        const playerMark = data.data.player_mark;
        localStorage.setItem("playerMark", playerMark);
        localStorage.setItem("roomId", roomId);
        setTimeout(() => {
          onRoomIdGenerated(roomId, playerMark);
          setIsLoadingJoinGame(false);
        }, 2000);
      }
    } catch (error: any) {
      if (error.response.data.message === "Player not found") {
        showErrorAlert("Player not found. Please register first.");
        localStorage.removeItem("playerId");
        window.location.reload();
        setIsLoadingJoinGame(false);
      } else if (error.response.data.message === "Room not found") {
        showErrorAlert("Room not found. Please check the room ID.");
        setIsLoadingJoinGame(false);
      } else if (error.response.data.message === "Game type not match") {
        showErrorAlert(
          "Game type does not match. Please choose the correct game."
        );
        setIsLoadingJoinGame(false);
      } else {
        setIsLoadingJoinGame(false);
        showErrorAlert(
          error.response.data.message ||
            "An error occurred while Joining the room. Please try again."
        );
      }
    }
  };

  return (
    <div className="flex justify-center items-center">
      <div className=" rounded-xl shadow-lg p-8 sm:w-96 sm:max-w-sm ring ring-primary outline outline-offset-4">
        <div className="flex flex-col items-center">
          <button
            onClick={handleNewGame}
            className="btn btn-primary btn-outline w-1/2 mb-3 sm:mb-4 shadow-xl textarea-xs sm:text-base outline outline-offset-4"
            disabled={isLoadingNewGame}
          >
            {!isLoadingNewGame ? (
              <>
                <span className="sm:mr-2 sm:w-6">
                  <svg
                    fill="currentColor"
                    width="25px"
                    height="25px"
                    viewBox="0 0 32 32"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <g
                      id="Group_26"
                      data-name="Group 26"
                      transform="translate(-526 -249.561)"
                    >
                      <path
                        id="Path_346"
                        data-name="Path 346"
                        d="M542,249.561a16,16,0,1,0,16,16A16,16,0,0,0,542,249.561Zm0,28a12,12,0,1,1,12-12A12,12,0,0,1,542,277.561Z"
                      />
                      <path
                        id="Path_348"
                        data-name="Path 348"
                        d="M540,271.561v-6h7Z"
                      />
                      <path
                        id="Path_349"
                        data-name="Path 349"
                        d="M540,259.561v6h7Z"
                      />
                    </g>
                  </svg>
                </span>
                New game
              </>
            ) : (
              <>
                <span className="loading loading-spinner loading-sm mr-2"></span>
                New game
              </>
            )}
          </button>

          <p className="text-base mt-4">-OR-</p>

          <label className="text-sm sm:text-base mt-2 mb-4">
            Enter Room Id
          </label>

          <div className="flex w-full items-center justify-center space-x-7">
            <input
              className="input input-bordered max-w-32 outline outline-offset-4 outline-primary text-primary text-center font-bold input-letter-spacing"
              maxLength={7}
              value={roomId.toUpperCase()}
              onChange={(e) => setRoomId(e.target.value.toUpperCase())}
            />
            <button
              onClick={handleJoinRoom}
              className="btn btn-primary ml-2 text-sm sm:text-base outline outline-offset-4"
              disabled={isLoadingJoinGame}
            >
              {!isLoadingJoinGame ? (
                <>
                  <span className="mr-1 sm:w-5">
                    <svg
                      fill="currentColor"
                      className="w-5 h-5 sm:w-6 sm:h-6"
                      viewBox="0 0 16 16"
                      version="1.1"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <rect
                        width="16"
                        height="16"
                        id="icon-bound"
                        fill="none"
                      />
                      <path d="M14,14l0,-12l-6,0l0,-2l8,0l0,16l-8,0l0,-2l6,0Zm-6.998,-0.998l4.998,-5.002l-5,-5l-1.416,1.416l2.588,2.584l-8.172,0l0,2l8.172,0l-2.586,2.586l1.416,1.416Z" />
                    </svg>
                  </span>
                  Join
                </>
              ) : (
                <>
                  <span className="loading loading-spinner loading-sm mr-2"></span>
                  Join
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
