import useWebSocket, { ReadyState } from "react-use-websocket";
import { useEffect, useMemo, useRef, useState } from "react";
import { showErrorAlert } from "./alerthelper";
import { clearGameSession } from "./gameStorage";

const backendUrl = process.env.NEXT_PUBLIC_WS_BACKEND_URL;

const EXPIRED_CLOSE_CODES = new Set([
  4001, // room expired
]);

const REJECTED_CLOSE_CODES = new Set([
  4002, // room full
  4003, // invalid room
  4004, // player not found
]);

const FATAL_CLOSE_CODES = new Set([
  ...EXPIRED_CLOSE_CODES,
  ...REJECTED_CLOSE_CODES,
]);

export type SocketState =
  | "connecting"
  | "open"
  | "reconnecting"
  | "rejected"
  | "expired";

type GameWebSocketOptions = {
  onExpired?: (reason: string) => void;
  onRejected?: (reason: string) => void;
};

export const useGameWebSocket = (
  gameType: string,
  roomId: string,
  playerId: string,
  options: GameWebSocketOptions = {},
) => {
  const [isOffline, setIsOffline] = useState(() =>
    typeof navigator === "undefined" ? false : !navigator.onLine,
  );
  const [hasReconnectStopped, setHasReconnectStopped] = useState(false);
  const [fatalSocketState, setFatalSocketState] = useState<SocketState | null>(
    null,
  );
  const hasOpenedRef = useRef(false);

  const socketUrl = useMemo(() => {
    if (!backendUrl || !roomId || !playerId) return null;

    const params = new URLSearchParams({
      room_id: roomId,
      player_id: playerId,
    });

    return `${backendUrl}/ws?${params.toString()}`;
  }, [roomId, playerId]);

  const { sendMessage, lastMessage, readyState } = useWebSocket(
    socketUrl,
    {
      onOpen: () => {
        hasOpenedRef.current = true;
        setHasReconnectStopped(false);
        setFatalSocketState(null);

        if (process.env.NODE_ENV === "development") {
          console.log(`WebSocket connected for ${gameType}`);
        }
      },

      onError: (error) => {
        if (process.env.NODE_ENV === "development") {
          console.log("WebSocket error:", error);
        }
      },

      onClose: (event) => {
        if (process.env.NODE_ENV === "development") {
          console.log(
            `🔌 WebSocket closed | code=${event.code} | reason=${event.reason}`,
          );
        }

        if (EXPIRED_CLOSE_CODES.has(event.code)) {
          const reason = event.reason || "room expired";
          setFatalSocketState("expired");
          options.onExpired?.(reason);
          showErrorAlert("Room expired or player session is no longer valid.");

          clearGameSession(gameType);
          setTimeout(() => {
            window.location.href = "/";
          }, 1000);
          return;
        }

        if (REJECTED_CLOSE_CODES.has(event.code)) {
          const reason = event.reason || "websocket rejected";
          setFatalSocketState("rejected");
          clearGameSession(gameType);

          options.onRejected?.(reason);
          setTimeout(() => {
            window.location.href = "/";
          }, 1000);
        }
      },

      shouldReconnect: (closeEvent) => {
        if (FATAL_CLOSE_CODES.has(closeEvent.code)) {
          setHasReconnectStopped(true);
          return false;
        }

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
      onReconnectStop: () => {
        setHasReconnectStopped(true);
      },
    },
    Boolean(socketUrl),
  );

  const socketState: SocketState = useMemo(() => {
    if (fatalSocketState) return fatalSocketState;
    if (readyState === ReadyState.OPEN) return "open";
    if (isOffline) return "reconnecting";
    if (hasReconnectStopped) return "rejected";

    if (readyState === ReadyState.CONNECTING) {
      return hasOpenedRef.current ? "reconnecting" : "connecting";
    }

    if (readyState === ReadyState.CLOSING || readyState === ReadyState.CLOSED) {
      return hasOpenedRef.current ? "reconnecting" : "connecting";
    }

    return "connecting";
  }, [fatalSocketState, readyState, isOffline, hasReconnectStopped]);

  const isReconnecting = socketState === "reconnecting" || isOffline;

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
      if (process.env.NODE_ENV === "development") {
        console.log("Browser offline");
      }
      setIsOffline(true);
    };

    const handleOnline = () => {
      if (process.env.NODE_ENV === "development") {
        console.log("Browser online");
      }
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
      console.log(`WS STATUS: ${connectionStatus} (${socketState})`);
    }
  }, [connectionStatus, socketState]);

  return {
    sendMessage,
    lastMessage,
    readyState,
    connectionStatus,
    socketState,
    isReconnecting,
  };
};
