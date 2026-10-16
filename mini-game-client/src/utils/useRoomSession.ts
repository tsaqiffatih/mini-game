"use client";

import axios from "axios";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  clearPlayerSession,
  clearGameSession,
  getAuthorizationHeaders,
  getGameSession,
  saveGameSession,
} from "./gameStorage";
import { getHttpBackendUrl } from "./apiConfig";

export type RoomSessionStatus =
  | "checking"
  | "valid"
  | "expired"
  | "unavailable"
  | "invalid";

type GameType = "chess" | "tictactoe";

type RoomSession = {
  roomId: string;
  playerMark: string;
  room: unknown;
};

type RecoveryFailure = {
  status: RoomSessionStatus;
  message: string;
};

type RecoveryResponse = {
  data?: {
    player_mark?: string;
    room?: {
      room_id?: string;
    };
  };
};

const backendUrl = getHttpBackendUrl();
const recoveryRequests = new Map<string, Promise<RecoveryResponse>>();

const recoverRoomSession = (recoveryKey: string, payload: unknown) => {
  const existingRequest = recoveryRequests.get(recoveryKey);
  if (existingRequest) {
    return existingRequest;
  }

  const request = axios
    .post(`${backendUrl}/room/recover`, payload, {
      headers: getAuthorizationHeaders(),
    })
    .then((response) => response.data)
    .finally(() => {
      recoveryRequests.delete(recoveryKey);
    });

  recoveryRequests.set(recoveryKey, request);
  return request;
};

const classifyRecoveryError = (error: unknown): RecoveryFailure => {
  if (!axios.isAxiosError(error)) {
    return { status: "unavailable", message: "Unable to recover room session" };
  }

  const responseStatus = error.response?.status;
  const message =
    error.response?.data?.error || "Unable to recover room session";

  if (responseStatus === 401) {
    return { status: "invalid", message: "Session expired. Please register again." };
  }

  if (responseStatus === 404 && message === "Room not found") {
    return { status: "expired", message };
  }

  if (
    responseStatus === 400 ||
    message === "Player not found" ||
    message === "Game type not match"
  ) {
    return { status: "invalid", message };
  }

  return { status: "unavailable", message };
};

export const useRoomSession = (gameType: GameType) => {
  const [status, setStatus] = useState<RoomSessionStatus>("checking");
  const [playerId, setPlayerId] = useState("");
  const [session, setSession] = useState<RoomSession | null>(null);
  const [error, setError] = useState<string | null>(null);
  const recoveryKeyRef = useRef<string | null>(null);

  const acceptSession = useCallback(
    (roomId: string, playerMark: string, aiLevel?: number) => {
      saveGameSession({ gameType, roomId, playerMark, aiLevel });
      setSession({ roomId, playerMark, room: null });
      setStatus("valid");
      setError(null);
    },
    [gameType],
  );

  useEffect(() => {
    let cancelled = false;

    const storedPlayerId = localStorage.getItem("playerId") || "";
    setPlayerId(storedPlayerId);

    if (!storedPlayerId || !getAuthorizationHeaders().Authorization) {
      clearPlayerSession();
      clearGameSession(gameType);
      setStatus("invalid");
      setError("Player session not found");
      return;
    }

    const storedSession = getGameSession(gameType);

    if (!storedSession.roomId) {
      setStatus("invalid");
      setSession(null);
      setError(null);
      return;
    }

    const recoveryKey = `${gameType}:${storedPlayerId}:${storedSession.roomId}`;
    if (recoveryKeyRef.current === recoveryKey) {
      return;
    }
    recoveryKeyRef.current = recoveryKey;

    const recover = async () => {
      setStatus("checking");
      setError(null);

      try {
        const data = await recoverRoomSession(recoveryKey, {
          game_type: gameType,
          player_id: storedPlayerId,
          room_id: storedSession.roomId,
        });

        if (cancelled) return;

        const recovered = data?.data;
        const recoveredRoomId = recovered?.room?.room_id;
        const recoveredPlayerMark = recovered?.player_mark;

        if (!recoveredRoomId || !recoveredPlayerMark) {
          clearGameSession(gameType);
          setSession(null);
          setStatus("invalid");
          setError("Invalid room recovery response");
          return;
        }

        saveGameSession({
          gameType,
          roomId: recoveredRoomId,
          playerMark: recoveredPlayerMark,
          aiLevel: storedSession.aiLevel
            ? Number(storedSession.aiLevel)
            : undefined,
        });
        setSession({
          roomId: recoveredRoomId,
          playerMark: recoveredPlayerMark,
          room: recovered.room,
        });
        setStatus("valid");
      } catch (error) {
        if (cancelled) return;

        const failure = classifyRecoveryError(error);

        if (failure.status === "expired" || failure.status === "invalid") {
          if (failure.status === "invalid") clearPlayerSession();
          clearGameSession(gameType);
        }

        setSession(null);
        setStatus(failure.status);
        setError(failure.message);
      }
    };

    recover();

    return () => {
      cancelled = true;
    };
  }, [gameType]);

  return {
    status,
    isChecking: status === "checking",
    isValid: status === "valid",
    playerId,
    roomId: session?.roomId || "",
    playerMark: session?.playerMark || "",
    room: session?.room,
    error,
    acceptSession,
  };
};
