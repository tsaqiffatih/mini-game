"use client";

import ChessBoard from "@/components/ChessBoard";
import Lobby from "@/components/Lobby";
import { showAlert, showErrorAlert } from "@/utils/alerthelper";
import { clearGameSession, getGameSession } from "@/utils/gameStorage";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

const ChessPage: React.FC = () => {
  const [playerId, setPlayerId] = useState<string>("");
  const [roomId, setRoomId] = useState<string>("");
  const [playerMark, setPlayerMark] = useState<string>("");

  const router = useRouter();

  const handleRoomIdReady = (roomId: string, playerMark: string) => {
    setRoomId(roomId);
    setPlayerMark(playerMark);
  };

  useEffect(() => {
    const storedPlayerId = localStorage.getItem("playerId") || "";
    if (!storedPlayerId) {
      showErrorAlert("Player ID not found. Please register first.");
      router.push("/");
      return;
    }

    const session = getGameSession("chess");

    setPlayerMark(session.playerMark || "");
    setRoomId(session.roomId || "");

    setPlayerId(storedPlayerId);
  }, [router]);

  return (
    <div className="p2">
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
                clearGameSession("chess")
                router.push("/");
              }
            });
          } else {
            clearGameSession("chess")
            router.push("/");
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
        <div className=" overflow-hidden sm:p-2 px-5 h-screen flex flex-col items-center justify-center">
          <Lobby
            playerId={playerId}
            onRoomReady={handleRoomIdReady}
            gameType="chess"
          />
        </div>
      ) : (
        <ChessBoard
          roomId={roomId}
          playerId={playerId}
          // playerMark={playerMark}
          // initialState={initialState}
        />
      )}
    </div>
  );
};

export default ChessPage;
