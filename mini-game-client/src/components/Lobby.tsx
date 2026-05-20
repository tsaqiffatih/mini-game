"use client";

/* eslint-disable @typescript-eslint/no-explicit-any */
import { useState } from "react";
import axios from "axios";
import { showErrorAlert } from "@/utils/alerthelper";
import { useRouter } from "next/navigation";
import AIDifficultyModal from "./AIDifficultyModal";
import { saveGameSession } from "@/utils/gameStorage";

type GameType = "chess" | "tictactoe";
interface LobbyProps {
  gameType: GameType;
  playerId: string;
  onRoomReady: (roomId: string, playerMark: string) => void;
}

const backendUrl = process.env.NEXT_PUBLIC_HTTP_BACKEND_URL;

export default function Lobby({ gameType, playerId, onRoomReady }: LobbyProps) {
  const [isLoadingNewGame, setIsLoadingNewGame] = useState(false);
  const [isLoadingJoinGame, setIsLoadingJoinGame] = useState(false);

  const [roomId, setRoomId] = useState("");

  // AI MODAL
  const [isAIModalOpen, setIsAIModalOpen] = useState(false);

  // AI LEVEL
  const [aiLevel, setAiLevel] = useState(5);

  const router = useRouter();

  const handlePlayWithAI = async () => {
    setIsLoadingNewGame(true);

    try {
      const { data } = await axios.post(`${backendUrl}/room/create/ai`, {
        game_type: gameType,
        player_id: playerId,
        ai_level: aiLevel,
      });

      const newRoomId = data?.data?.room?.room_id;
      const playerMark = data?.data?.player_mark;

      if (!newRoomId) {
        throw new Error("roomId missing");
      }

      saveGameSession({
        gameType,
        roomId: newRoomId,
        playerMark,
        aiLevel,
      });

      setIsAIModalOpen(false);
      onRoomReady(newRoomId, playerMark);
    } catch (error: any) {
      handleHttpError(error);
    } finally {
      setIsLoadingNewGame(false);
    }
  };

  const handleCreateRoom = async () => {
    setIsLoadingNewGame(true);

    try {
      const { data } = await axios.post(`${backendUrl}/room/create`, {
        game_type: gameType,
        player_id: playerId,
      });

      const newRoomId = data?.data?.room?.room_id;
      const playerMark = data?.data?.player_mark;

      if (!newRoomId) {
        throw new Error("roomId missing");
      }

      saveGameSession({
        gameType,
        roomId: newRoomId,
        playerMark,
      });

      onRoomReady(newRoomId, playerMark);
    } catch (error: any) {
      console.log(error.response.data.error, "<<<<<");

      handleHttpError(error);
    } finally {
      setIsLoadingNewGame(false);
    }
  };

  const handleJoinRoom = async () => {
    setIsLoadingJoinGame(true);

    try {
      const { data } = await axios.post(`${backendUrl}/room/join`, {
        room_id: roomId,
        player_id: playerId,
        game_type: gameType,
      });

      const newRoomId = data?.data?.room?.room_id;
      const playerMark = data?.data?.player_mark;

      saveGameSession({
        gameType,
        roomId: newRoomId,
        playerMark,
      });

      onRoomReady(newRoomId, playerMark);
    } catch (error: any) {
      handleHttpError(error);
    } finally {
      setIsLoadingJoinGame(false);
    }
  };

  const handleHttpError = (error: any) => {
    const msg = error?.response?.data?.error;

    switch (msg) {
      case "Player not found":
        // showErrorAlert("Player not found. Please register first.");

        localStorage.removeItem("playerId");

        window.location.reload();

        break;

      case "Room not found":
        showErrorAlert("Room not found. Please check the room ID.");

        break;

      case "Game type not match":
        showErrorAlert("Game type does not match.").then(() =>
          router.push("/"),
        );

        break;

      case "failed to create room":
        showErrorAlert("An error occurred. Please try again.");

        break;

      default:
        showErrorAlert(msg || "An error occurred. Please try again.");
    }
  };

  return (
    <>
      {/* AI MODAL */}
      <AIDifficultyModal
        isOpen={isAIModalOpen}
        onClose={() => setIsAIModalOpen(false)}
        gameType={gameType}
        aiLevel={aiLevel}
        setAiLevel={setAiLevel}
        onStart={handlePlayWithAI}
        isLoading={isLoadingNewGame}
      />

      {/* MAIN LOBBY */}
      <div className="flex flex-col items-center">
        <h1 className="text-2xl sm:text-3xl font-extrabold mb-5 text-center">
          Welcome in {gameType} Game
        </h1>

        <div className="rounded-xl shadow-lg p-5 w-full sm:w-96 sm:max-w-sm ring ring-primary outline outline-offset-4 flex flex-col items-center border border-primary">
          {/* CREATE BUTTONS */}
          <div className="flex space-x-5 mb-3 px-3 sm:mb-4 w-full items-center justify-center">
            {/* NEW GAME */}
            <button
              onClick={handleCreateRoom}
              className="btn btn-primary btn-outline w-1/2 mb-3 sm:mb-4 shadow-xl text-base outline outline-offset-4"
              disabled={isLoadingNewGame}
            >
              {!isLoadingNewGame ? (
                <div className="flex items-center justify-center space-x-3">
                  <span className="mr-2 w-6">
                    <svg
                      fill="currentColor"
                      width="25px"
                      height="25px"
                      viewBox="0 0 32 32"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <g transform="translate(-526 -249.561)">
                        <path d="M542,249.561a16,16,0,1,0,16,16A16,16,0,0,0,542,249.561Zm0,28a12,12,0,1,1,12-12A12,12,0,0,1,542,277.561Z" />
                        <path d="M540,271.561v-6h7Z" />
                        <path d="M540,259.561v6h7Z" />
                      </g>
                    </svg>
                  </span>
                  New game
                </div>
              ) : (
                <>
                  <span className="loading loading-spinner loading-sm mr-2"></span>
                  New game
                </>
              )}
            </button>

            {/* AI GAME */}
            <button
              onClick={() => setIsAIModalOpen(true)}
              className="btn btn-primary btn-outline w-1/2 mb-3 sm:mb-4 shadow-xl text-base outline outline-offset-4"
              disabled={isLoadingNewGame}
            >
              {!isLoadingNewGame ? (
                <div className="flex items-center justify-center space-x-3">
                  <span className="mr-2 w-6">
                    <svg
                      version="1.1"
                      xmlns="http://www.w3.org/2000/svg"
                      fill="currentColor"
                      width="25px"
                      height="25px"
                      viewBox="0 0 45.342 45.342"
                    >
                      <g>
                        <path d="M40.462,19.193H39.13v-1.872c0-3.021-2.476-5.458-5.496-5.458h-8.975v-4.49c1.18-0.683,1.973-1.959,1.973-3.423c0-2.182-1.771-3.95-3.951-3.95c-2.183,0-3.963,1.769-3.963,3.95c0,1.464,0.785,2.74,1.965,3.423v4.49h-8.961c-3.021,0-5.448,2.437-5.448,5.458v1.872H4.893c-1.701,0-3.091,1.407-3.091,3.108v6.653c0,1.7,1.39,3.095,3.091,3.095h1.381v1.887c0,3.021,2.427,5.442,5.448,5.442h2.564v2.884c0,1.701,1.393,3.08,3.094,3.08h10.596c1.701,0,3.08-1.379,3.08-3.08v-2.883h2.578c3.021,0,5.496-2.422,5.496-5.443V32.05h1.332c1.701,0,3.078-1.394,3.078-3.095v-6.653C43.54,20.601,42.165,19.193,40.462,19.193z" />
                      </g>
                    </svg>
                  </span>
                  Play With AI
                </div>
              ) : (
                <>
                  <span className="loading loading-spinner loading-sm mr-2"></span>
                  Loading
                </>
              )}
            </button>
          </div>

          <p className="text-base mt-4">-OR-</p>

          <label className="text-sm sm:text-base mt-2 mb-4">
            Enter Room Id
          </label>

          {/* JOIN ROOM */}
          <div className="flex w-full items-center justify-center space-x-7">
            <input
              className="input input-bordered max-w-32 outline outline-offset-4 outline-primary text-primary text-center font-bold input-letter-spacing"
              maxLength={7}
              value={roomId.toUpperCase()}
              onChange={(e) => setRoomId(e.target.value.toUpperCase())}
            />

            <button
              onClick={handleJoinRoom}
              className="btn btn-primary ml-2 text-base outline outline-offset-4"
              disabled={isLoadingJoinGame}
            >
              {!isLoadingJoinGame ? (
                <>
                  <span className="mr-1 sm:w-5">
                    <svg
                      fill="currentColor"
                      className="w-5 h-5 sm:w-6 sm:h-6"
                      viewBox="0 0 16 16"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <rect width="16" height="16" fill="none" />

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
    </>
  );
}
