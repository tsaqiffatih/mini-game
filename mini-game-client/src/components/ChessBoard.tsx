// components/ChessBoardClient.tsx
"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Chess, Square } from "chess.js";
import { Chessboard } from "react-chessboard";
// import "@/styles/chessboard.css";
import { useRouter } from "next/navigation";
import { handleLeaveGameAlert, showAlert } from "@/utils/alerthelper";
import Waiting from "./Waiting";
import ChatOpened from "./ChatOpened";
import { useGameWebSocket } from "@/utils/gameWebsocket";
import { checkGameStatus, handleChessMove } from "@/utils/chessMoveHandlers";
import { PromotionPieceOption } from "react-chessboard/dist/chessboard/types";

interface ChessBoardProps {
  playerId: string;
  roomId: string;
  playerMark: string;
  initialState?: string;
}

const game = new Chess();

export default function ChessBoard({
  playerId,
  roomId,
  playerMark,
  initialState,
}: ChessBoardProps) {
  const [playerMarkState, setPlayerMarkState] = useState<string>(playerMark);
  const [fen, setFen] = useState<string>(initialState || game.fen());
  const [isGameActive, setIsGameActive] = useState<boolean>(false);
  const [winner, setWinner] = useState<string>("");
  const [chatMessages, setChatMessages] = useState<
    Array<{ sender: string; message: string; timestamp: string }>
  >([]);
  const [isChatOpen, setIsChatOpen] = useState<boolean>(false);
  const [hasNewMessage, setHasNewMessage] = useState<boolean>(false);
  const [resetRequest, setResetRequest] = useState<boolean>(false);
  const [lastMove, setLastMove] = useState<{ from: string; to: string } | null>(
    null
  );

  const router = useRouter();
  const lastMessageRef = useRef<string | null>(null);

  const { sendMessage, lastMessage } = useGameWebSocket(roomId, playerId);

  // reset function
  const resetGame = useCallback((): void => {
    game.reset();
    setFen(game.fen());
    setWinner("");
    setIsGameActive(true);
  }, []);

  const getSquareStyles = () => {
    const styles: { [key: string]: React.CSSProperties } = {};
    if (lastMove) {
      styles[lastMove.from] = { backgroundColor: "rgba(255, 255, 0, 0.5)" };
      styles[lastMove.to] = { backgroundColor: "rgba(255, 255, 0, 0.5)" };
    }
    return styles;
  };

  // onDrop: LOG first, then call handler
  const onDrop = useCallback(
    (sourceSquare: Square, targetSquare: Square): boolean => {
      // debug log MUST be before return

      console.log("DROP", { sourceSquare, targetSquare, playerMarkState });

      // quick check: is this a pawn promotion?
      const piece = game.get(sourceSquare);
      const isPawn = piece && piece.type === "p";
      const isWhitePromote =
        isPawn && piece!.color === "w" && targetSquare.endsWith("8");
      const isBlackPromote =
        isPawn && piece!.color === "b" && targetSquare.endsWith("1");

      if (isWhitePromote || isBlackPromote) {
        // Don't perform the move here — let the promotion modal trigger onPromotionPieceSelect.
        // Returning false avoids making a default promotion move.
        return false;
      }

      const ok = handleChessMove(
        game,
        sourceSquare,
        targetSquare,
        playerMarkState,
        playerId,
        sendMessage,
        setFen,
        setLastMove,
        setWinner,
        setIsGameActive
      );

      console.log("DROP RESULT", { ok, fen: game.fen() });

      return ok;
    },
    [playerId, playerMarkState, sendMessage]
  );

  const onPromotionPieceSelect = useCallback(
    (
      piece?: PromotionPieceOption,
      promoteFromSquare?: Square,
      promoteToSquare?: Square
    ): boolean => {
      if (!piece || !promoteFromSquare || !promoteToSquare) {
        return false;
      }

       // convert "wQ" | "bQ" -> "q"
    const normalizedPiece = piece.toLowerCase().replace(/^[wb]/, "") as
      | "q"
      | "r"
      | "b"
      | "n";

      // call handler with the user's chosen piece
      const ok = handleChessMove(
        game,
        promoteFromSquare,
        promoteToSquare,
        playerMarkState,
        playerId,
        sendMessage,
        setFen,
        setLastMove,
        setWinner,
        setIsGameActive,
        normalizedPiece // <-- pass promotion piece here
      );

      return ok;
    },
    [playerId, playerMarkState, sendMessage]
  );

  const handleResetRequest = () => {
    sendMessage(
      JSON.stringify({
        action: "REQUEST_RESET",
        sender: { player_id: playerId },
      })
    );
    setResetRequest(true);
  };

  useEffect(() => {
    const handlePopState = () => {
      handleLeaveGameAlert(router);
    };

    window.history.pushState(null, "", window.location.href);
    window.addEventListener("popstate", handlePopState);

    return () => {
      window.removeEventListener("popstate", handlePopState);
    };
  }, [router]);

  // useEffect untuk lastMessage
  useEffect(() => {
    if (!lastMessage || lastMessage.data === lastMessageRef.current) return;

    lastMessageRef.current = lastMessage.data;

    // parse message
    let parsed: any;
    try {
      parsed = JSON.parse(lastMessage.data);
    } catch (err) {
      console.log(err);

      console.error("Invalid ws message JSON", lastMessage.data);
      return;
    }

    const { action, message, sender, timestamp } = parsed;

    if (action === "CHESS_MOVE") {
      if (message) {
        console.log("WS CHESS_MOVE received", message);
        try {
          game.load(message.fen);
          setFen(game.fen());
          setLastMove(message.lastMove);
          checkGameStatus(game, sendMessage, setWinner, setIsGameActive);
        } catch (err) {
          console.error("Failed to apply CHESS_MOVE from WS", err, message);
        }
      }
    }

    if (action === "CHESS_GAME_STATE") {
      if (message) {
        try {
          game.load(message);
          setFen(game.fen());
        } catch (err) {
          console.error("Failed to load CHESS_GAME_STATE", err);
        }
      }
    }

    /* 🟢 1. USER_LEFT_ROOM */
    if (action === "USER_LEFT_ROOM") {
      showAlert({
        title: "Player Left",
        text: "The other player has left the room.",
        icon: "info",
        confirmButtonText: "Ok",
      });
    }

    /* 🟢 2. CONNECTED_ON_SERVER */
    if (action === "CONNECTED_ON_SERVER") {
      if (sender.player_id !== playerId) {
        showAlert({
          title: "Player Joined",
          text: "The other player has joined the room.",
          icon: "info",
          confirmButtonText: "Ok",
        });
      }
    }

    /* 🟢 3. START_GAME */
    if (action === "START_GAME") {
      setIsGameActive(true);
    }

    if (action === "CHAT_MESSAGE") {
      const newMessage = {
        sender: sender.player_id,
        message: message,
        timestamp: timestamp,
      };

      setChatMessages((prevMessages) => [...prevMessages, newMessage]);

      if (!isChatOpen) {
        setHasNewMessage(true);
        const audio = new Audio("/sounds/notification.mp3");
        audio.play();
      }
    }

    if (action === "GAME_CHECKMATE") {
      setWinner(message);
      setIsGameActive(false);
    }

    if (action === "GAME_DRAW") {
      setWinner(message);
      setIsGameActive(false);
    }

    if (action === "REQUEST_RESET") {
      showAlert({
        title: "Reset Game?",
        text: "The other player wants to reset the game. Do you agree?",
        icon: "warning",
        showCancelButton: true,
        confirmButtonText: "Yes, reset",
        cancelButtonText: "No, continue",
      }).then((result) => {
        if (result.isConfirmed) {
          sendMessage(
            JSON.stringify({
              action: "CONFIRM_RESET",
              sender: { player_id: playerId },
            })
          );
        }
      });
    }

    if (action === "CONFIRM_RESET") {
      resetGame();
    }

    if (action === "MARK_UPDATE") {
      const marks = message.marks;

      if (marks && marks[playerId]) {
        const newMark = marks[playerId];
        setPlayerMarkState(newMark);
        setIsGameActive(message.active);
        localStorage.setItem("playerMark", newMark);
      }
    }
  }, [lastMessage, resetGame, playerId, isChatOpen, sendMessage]);

  const boardPerspective = (): "white" | "black" => {
    return playerMarkState === "black" ? "black" : "white";
  };

  const handleSendMessage = (message: string) => {
    const chatMessage = {
      action: "CHAT_MESSAGE",
      message: message,
      sender: { player_id: playerId },
      time: new Date().toISOString(),
    };

    sendMessage(JSON.stringify(chatMessage));
  };

  const handleOpenChat = () => {
    setIsChatOpen(true);
    setHasNewMessage(false);
  };

  return (
    <div className="flex overflow-hidden">
      {roomId && !isGameActive && !winner && <Waiting roomId={roomId} />}
      {winner && winner !== "Draw" && !isGameActive && (
        <div className="flex flex-col h-full items-center justify-center">
          <h2 className="text-2xl font-bold text-green-600">
            Winner: {winner}
          </h2>
          <button
            className="btn btn-primary ml-4"
            onClick={handleResetRequest}
            disabled={resetRequest}
          >
            Request Reset
          </button>
        </div>
      )}
      {winner === "Draw" && (
        <div className="flex flex-col items-center justify-center space-y-4">
          <h2 className="text-2xl font-bold text-yellow-600 text-center">
            Game ended in a Draw!
          </h2>
          <p className="text-lg text-center">
            No more moves available, and no player has won.
          </p>
          <p className="text-lg text-center">Wait Until the game Restart</p>
        </div>
      )}
      {isGameActive && (
        <div className="flex flex-col md:flex-row">
          <h2 className="mb-2 md:mr-2">Room Id: &quot;{roomId}&quot;</h2>

          <div className="flex justify-around items-center mr-2 ">
            <Chessboard
              position={fen}
              onPieceDrop={onDrop}
              boardOrientation={boardPerspective()}
              customBoardStyle={{
                borderRadius: "5px",
                boxShadow: "0 5px 15px rgba(0, 0, 0, 0.5)",
              }}
              boardWidth={Math.min(window.innerWidth, window.innerHeight) * 0.9}
              customLightSquareStyle={{ backgroundColor: "AliceBlue" }}
              customDarkSquareStyle={{ backgroundColor: "#b3b3b3" }}
              customDropSquareStyle={{ boxShadow: "inset 0 0 1px 4px #ff0000" }}
              customSquareStyles={getSquareStyles()}
              promotionDialogVariant="modal"
              onPromotionPieceSelect={onPromotionPieceSelect}
            />
          </div>

          <div className=" hidden md:flex md:items-center ">
            <ChatOpened
              setOpenChatOpened={() => {}}
              userName={playerId}
              messages={chatMessages}
              onSendMessage={handleSendMessage}
            />
          </div>
          <div className="block md:hidden mx-auto mt-2 relative">
            <button
              className="btn btn-sm btn-primary btn-outline p-2"
              onClick={handleOpenChat}
            >
              Open Chat
              {hasNewMessage && (
                <span className="absolute top-0 right-0 badge badge-primary badge-xs transform translate-x-1/2 -translate-y-1/2"></span>
              )}
            </button>
          </div>
        </div>
      )}

      {isChatOpen && (
        <div className="fixed md:hidden inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
          <div className="bg-base-100 py-0 rounded-lg w-11/12">
            <ChatOpened
              setOpenChatOpened={() => setIsChatOpen(false)}
              userName={playerId}
              messages={chatMessages}
              onSendMessage={handleSendMessage}
            />
          </div>
        </div>
      )}
    </div>
  );
}
