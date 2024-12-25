import useWebSocket from "react-use-websocket";
import { showErrorAlert } from "../../utils/alerthelper";
import { useEffect, useState } from "react";
import { Chess } from "chess.js";

interface ChessBoardProps {
  playerId: string;
  roomId: string;
  playerMark: string;
}

const game = new Chess();

export default function ChessBoard({
  playerId,
  roomId,
  playerMark,
}: ChessBoardProps) {
    const [playerMarkState, setPlayerMarkState] = useState<string>(playerMark);


    const {sendMessage, lastMessage} = useWebSocket(
        `ws://localhost:8080/ws?room_id=${roomId}&player_id=${playerId}`,
        {
            onOpen: () => console.log("websocket connected"),
            onError: (event) => {
            console.log("WebSocket error: ", event);
            showErrorAlert(
                "Room expired or no longer available. Please create or join a new room."
            );
            localStorage.removeItem("roomId");
            localStorage.removeItem("playerMark");
            setTimeout(() => {
                window.location.reload();
            }, 1000);
            },
            onClose: () => {
            // localStorage.removeItem("roomId");
            // localStorage.removeItem("playerMark");
            },
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
            shouldReconnect: (_closeEvent) => true,
        }
    )

    useEffect(() => {
        if (lastMessage !== null) {
            console.log(lastMessage.data);
        }
    })

  return (
    <div>
      <h1>Chess Board</h1>
    </div>
  );
}
