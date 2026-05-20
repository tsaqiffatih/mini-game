import useWebSocket, { ReadyState } from "react-use-websocket";
import { showErrorAlert } from "@/utils/alerthelper";
import { useEffect, useMemo, useState } from "react";

const backendUrl = process.env.NEXT_PUBLIC_WS_BACKEND_URL;

const FATAL_CLOSE_CODES = new Set([
  4001, // room expired
  4002, // room full
  4003, // invalid room
  4004, // player not found
]);

export const useGameWebSocket = (
  gameType: string,
  roomId: string,
  playerId: string,
) => {
  const [isOffline, setIsOffline] = useState(!navigator.onLine);
  const [hasReconnectStopped, setHasReconnectStopped] = useState(false);

  const { sendMessage, lastMessage, readyState } = useWebSocket(
    `${backendUrl}/ws?room_id=${roomId}&player_id=${playerId}`,
    {
      onOpen: () => {
        if (process.env.NODE_ENV === "development") {
          console.log("✅ WebSocket connected");
        }
      },

      onError: (error) => {
        if (process.env.NODE_ENV === "development") {
          console.log("❌ WebSocket error:", error);
        }
      },

      onClose: (event) => {
        if (process.env.NODE_ENV === "development") {
          console.log(
            `🔌 WebSocket closed | code=${event.code} | reason=${event.reason}`,
          );
        }

        // fatal errors → stop reconnect
        if (FATAL_CLOSE_CODES.has(event.code)) {
          showErrorAlert("Room expired or player session is no longer valid.");

          localStorage.removeItem("roomId");
          localStorage.removeItem("playerMark");

          setTimeout(() => {
            window.location.href = "/";
          }, 1000);
        }
      },

      shouldReconnect: (closeEvent) => {
        // fatal errors → do not reconnect
        if (FATAL_CLOSE_CODES.has(closeEvent.code)) {
          setHasReconnectStopped(true);
          return false;
        }

        // normal close
        if (closeEvent.code === 1000) {
          setHasReconnectStopped(true);
          return false;
        }

        return true;
      },

      reconnectAttempts: 10,

      reconnectInterval: (attemptNumber) => {
        const delay = Math.min(attemptNumber * 1000, 10000);

        if (process.env.NODE_ENV === "development") {
          console.log(
            `🔄 Reconnect attempt ${attemptNumber} in ${delay / 1000}s`,
          );
        }

        return delay;
      },
    },
  );

  // derived reconnect state
  const isReconnecting = useMemo(() => {
    if (hasReconnectStopped) return false;

    return (
      isOffline ||
      readyState === ReadyState.CONNECTING ||
      readyState === ReadyState.CLOSING ||
      readyState === ReadyState.CLOSED
    );
  }, [readyState, isOffline, hasReconnectStopped]);

  // readable status
  const connectionStatus = useMemo(() => {
    return {
      [ReadyState.CONNECTING]: "CONNECTING",
      [ReadyState.OPEN]: "OPEN",
      [ReadyState.CLOSING]: "CLOSING",
      [ReadyState.CLOSED]: "CLOSED",
      [ReadyState.UNINSTANTIATED]: "UNINSTANTIATED",
    }[readyState];
  }, [readyState]);

  useEffect(() => {
    const handleOffline = () => {
      console.log("📴 Browser offline");
      setIsOffline(true);
    };

    const handleOnline = () => {
      console.log("🌐 Browser online");
      setIsOffline(false);
    };

    window.addEventListener("offline", handleOffline);
    window.addEventListener("online", handleOnline);

    return () => {
      window.removeEventListener("offline", handleOffline);
      window.removeEventListener("online", handleOnline);
    };
  }, []);

  // debug logger
  useEffect(() => {
    if (process.env.NODE_ENV === "development") {
      console.log(`📡 WS STATUS: ${connectionStatus}`);
    }
  }, [connectionStatus]);

  useEffect(() => {
    console.log("READY STATE:", readyState);
  }, [readyState]);

  return {
    sendMessage,
    lastMessage,
    readyState,
    connectionStatus,
    isReconnecting,
  };
};
