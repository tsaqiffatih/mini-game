import useWebSocket from "react-use-websocket";
import { WebSocketContext } from "./useWebsocketContext";

export const WebSocketProvider: React.FC<{ url: string; children: React.ReactNode }> = ({
  url,
  children,
}) => {
  const { sendMessage, lastMessage, readyState } = useWebSocket(url, {
    shouldReconnect: () => true,
  });

  return (
    <WebSocketContext.Provider value={{ sendMessage, lastMessage, readyState }}>
      {children}
    </WebSocketContext.Provider>
  );
};
