"use client";

import Lobby from "@/components/Lobby";
import TicTacToeBoard from "@/components/TicTacToeBoard";
import { showAlert, showErrorAlert } from "@/utils/alerthelper";
import { clearGameSession } from "@/utils/gameStorage";
import { useRoomSession } from "@/utils/useRoomSession";
import { useRouter } from "next/navigation";
import { useEffect, useRef } from "react";

import ErrorBoundary from "@/components/ErrorBoundary";

const TicTacToePage: React.FC = () => {
  const router = useRouter();
  const missingPlayerAlertShownRef = useRef(false);
  const {
    status,
    isChecking,
    isValid,
    playerId,
    roomId,
    playerMark,
    acceptSession,
    error,
  } = useRoomSession("tictactoe");

  const handleRoomReady = (roomId: string, playerMark: string) => {
    acceptSession(roomId, playerMark);
  };

  useEffect(() => {
    if (
      status === "invalid" &&
      !playerId &&
      !missingPlayerAlertShownRef.current
    ) {
      missingPlayerAlertShownRef.current = true;
      showErrorAlert("Player not found. Please register first.");
      router.push("/");
    }
  }, [playerId, router, status]);

  return (
    <ErrorBoundary>
      <div className=" overflow-hidden sm:p-2 px-4 h-screen flex flex-col items-center justify-center">
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
                  clearGameSession("tictactoe");
                  router.push("/");
                }
              });
            } else {
              clearGameSession("tictactoe");
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
        {isChecking ? (
          <span className="loading loading-spinner loading-lg" />
        ) : !isValid ? (
          <>
            {error && playerId && (
              <p className="text-sm text-warning mb-4 text-center">{error}</p>
            )}
            <Lobby
              playerId={playerId}
              onRoomReady={handleRoomReady}
              gameType="tictactoe"
            />
          </>
        ) : (
          <TicTacToeBoard
            playerId={playerId}
            roomId={roomId}
            playerMark={playerMark}
          />
        )}
      </div>
    </ErrorBoundary>
  );
};

export default TicTacToePage;
