import useWebSocket from "react-use-websocket";
import { showAlert, showErrorAlert } from "../../utils/alerthelper";
import { useCallback, useEffect, useState } from "react";
import { Chess } from "chess.js";
import playCaptureSound from "../../utils/capturedSound";
import playMoveSound from "../../utils/moveSound";
import { Chessboard } from "react-chessboard";
import { useNavigate } from "react-router-dom";
import Waiting from "../shared/Waiting";
import ChatOpened from "../shared/ChatOpened";
import "./chessboard.css";

interface ChessBoardProps {
  playerId: string;
  roomId: string;
  playerMark: string;
  initialState?: string;
}

const backendUrl = import.meta.env.VITE_WS_BACKEND_URL;

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

  const navigate = useNavigate();

  const { sendMessage, lastMessage } = useWebSocket(
    `${backendUrl}/ws?room_id=${roomId}&player_id=${playerId}`,
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
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      shouldReconnect: (_closeEvent) => true,
    }
  );

  // function for checking turn
  const isMyTurn = useCallback((): boolean => {
    return (
      (game.turn() === "w" && playerMarkState === "white") ||
      (game.turn() === "b" && playerMarkState === "black")
    );
  }, [playerMarkState]);

  // function for reset the game
  const resetGame = useCallback((): void => {
    game.reset();

    setFen(game.fen());
    setWinner("");
    setIsGameActive(true);
  }, []);

  const checkGameStatus = useCallback(() => {
    if (game.isCheckmate()) {
      const winnerMessage =
        game.turn() === "w"
          ? "Black pieces win by checkmate!"
          : "White pieces win by checkmate!";
      sendMessage(
        JSON.stringify({
          action: "GAME_CHECKMATE",
          message: winnerMessage,
        })
      );
      setWinner(winnerMessage);
      setIsGameActive(false);
    } else if (
      game.isDraw() ||
      game.isThreefoldRepetition() ||
      game.isStalemate()
    ) {
      sendMessage(
        JSON.stringify({
          action: "GAME_DRAW",
          message: "The match ended in a draw.",
        })
      );
      setWinner("Draw");
      setIsGameActive(false);
    }
  }, [sendMessage]);

  const getSquareStyles = () => {
    const styles: { [key: string]: React.CSSProperties } = {};
    if (lastMove) {
      styles[lastMove.from] = { backgroundColor: "rgba(255, 255, 0, 0.5)" };
      styles[lastMove.to] = { backgroundColor: "rgba(255, 255, 0, 0.5)" };
    }
    return styles;
  };

  // function for moving the pieces
  const onDrop = useCallback(
    (sourceSquare: string, targetSquare: string): boolean => {
      if (!isMyTurn()) return false;

      const move = game.move({
        from: sourceSquare,
        to: targetSquare,
        promotion: "q",
      });

      if (!move) return false;

      if (move.captured) playCaptureSound();
      else playMoveSound();

      setFen(game.fen());
      setLastMove({ from: sourceSquare, to: targetSquare });

      const moveMessage = {
        action: "CHESS_MOVE",
        message: {
          fen: game.fen(),
          lastMove: { from: sourceSquare, to: targetSquare },
        },
        sender: { player_id: playerId },
      };

      sendMessage(JSON.stringify(moveMessage));

      checkGameStatus();

      return true;
    },
    [isMyTurn, sendMessage, checkGameStatus, playerId]
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
      showAlert({
        title: "Leave Game?",
        text: "Are you sure you want to leave the game? Your progress will be lost.",
        icon: "warning",
        showCancelButton: true,
        confirmButtonText: "Yes, leave",
        cancelButtonText: "No, stay",
      }).then((result) => {
        if (result.isConfirmed) {
          localStorage.removeItem("roomId");
          localStorage.removeItem("playerMark");
          navigate("/");
        } else {
          window.history.pushState(null, "", window.location.href);
        }
      });
    };

    window.addEventListener("popstate", handlePopState);

    return () => {
      window.removeEventListener("popstate", handlePopState);
    };
  }, [playerId, sendMessage, navigate]);

  useEffect(() => {
    if (!lastMessage) return;

    const { action, message, sender, timestamp } = JSON.parse(lastMessage.data);

    if (action === "CHESS_MOVE") {
      if (message) {
        game.load(message.fen);
        setFen(game.fen());
        setLastMove(message.lastMove);
        console.log("lastMove =>", message.lastMove);
        
        checkGameStatus();
        // console.log("message =>", message);
        
      } else {
        console.error("Invalid FEN received:", message);
      }
    }

    if (action === "CHESS_GAME_STATE") {
      if (message) {
        game.load(message);
        setFen(game.fen());
      } else {
        console.error("Invalid game state received:", message);
      }
    }

    if (action === "USER_LEFT_ROOM") {
      showAlert({
        title: "Player Left",
        text: "The other player has left the room.",
        icon: "info",
        confirmButtonText: "Ok",
      });
    }

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
  }, [
    checkGameStatus,
    lastMessage,
    resetGame,
    playerId,
    isChatOpen,
    sendMessage,
  ]);

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
          <h2 className="mb-2 md:mr-2">Room Id: "{roomId}"</h2>

          <div className="flex justify-around items-center mr-2">
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
              // customDropSquareStyle={{ boxShadow: "inset 0 0 1px 4px #ff0000" }}
              customSquareStyles={getSquareStyles()}
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
