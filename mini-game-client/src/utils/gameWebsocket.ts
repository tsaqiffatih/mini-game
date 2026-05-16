import useWebSocket from "react-use-websocket";
import { showErrorAlert } from "@/utils/alerthelper";

const backendUrl = process.env.NEXT_PUBLIC_WS_BACKEND_URL;

const FATAL_CLOSE_CODES = new Set([
  4001, // room expired
  4002, // room full
  4003, // invalid room
  4004, // player not found
]);

export const useGameWebSocket = (roomId: string, playerId: string) => {
  const { sendMessage, lastMessage } = useWebSocket(
    `${backendUrl}/ws?room_id=${roomId}&player_id=${playerId}`,
    {
      onOpen: () => {
        if (process.env.NODE_ENV === "development") {
          console.log("WebSocket connected");
        }
      },
      onError: (error) => {
        if (process.env.NODE_ENV === "development") {
          console.log("WebSocket error: ", error);
        }
      },
      onClose: (event) => {
        if (process.env.NODE_ENV === "development") {
          console.log("WebSocket closed:", event.code, event.reason);
        }
        // fatal room/session errors only
        if (FATAL_CLOSE_CODES.has(event.code)) {
          showErrorAlert("Room expired or player session is no longer valid.");
          localStorage.removeItem("roomId");
          localStorage.removeItem("playerMark");
          setTimeout(() => {
            window.location.href = "/";
          }, 1000);
          console.log("WebSocket closed:", event.code, event.reason);
        }
      },

      shouldReconnect: (closeEvent) => {
        // stop reconnecting for fatal application errors
        if (FATAL_CLOSE_CODES.has(closeEvent.code)) {
          return false;
        }

        // normal close
        if (closeEvent.code === 1000) {
          return false;
        }

        // reconnect for:
        // - refresh disconnects
        // - temporary network failures
        // - abnormal closures
        // - backend restarts/redeploys
        return true;
      },

      reconnectAttempts: 10,
      reconnectInterval: (attemptNumber) =>
        Math.min(attemptNumber * 1000, 10000),
    },
  );

  return { sendMessage, lastMessage };
};
