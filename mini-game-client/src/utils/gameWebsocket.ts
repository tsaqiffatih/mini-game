import useWebSocket from "react-use-websocket";
import { showErrorAlert } from "@/utils/alerthelper";

const backendUrl = process.env.NEXT_PUBLIC_WS_BACKEND_URL;

export const useGameWebSocket = (roomId: string, playerId: string) => {
  const { sendMessage, lastMessage } = useWebSocket(
    `${backendUrl}/ws?room_id=${roomId}&player_id=${playerId}`,
    {
      onOpen: () => {
        if (process.env.NODE_ENV === 'development') {
          console.log("WebSocket connected");
        }
      },
      onError: (event) => {
        if (process.env.NODE_ENV === 'development') {
          console.log("WebSocket error: ", event);
        }
        showErrorAlert(
          "Room expired or no longer available. Please create or join a new room."
        );
        localStorage.removeItem("roomId");
        localStorage.removeItem("playerMark");
        setTimeout(() => {
          window.location.reload();
        }, 1000);
      },
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      shouldReconnect: (_closeEvent) => true,
    }
  );

  return { sendMessage, lastMessage };
};