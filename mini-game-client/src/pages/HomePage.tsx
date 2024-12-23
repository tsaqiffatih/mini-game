import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import RegisterUser from "../components/shared/RegisterUser";
import GitHubStarLink from "../components/shared/GitHubStarLink";

const HomePage = () => {
  const [playerId, setPlayerId] = useState<string>("");
  const getPlayerId = () => {
    const storedPlayerId = localStorage.getItem("playerId");
    if (storedPlayerId) {
      setPlayerId(storedPlayerId);
    }
  };

  const navigateToGame = useNavigate();

  useEffect(() => {
    getPlayerId();
  }, []);

  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-base-100 p-4 text-primary">
      <GitHubStarLink />
      <div className="rounded-lg ring ring-primary p-16 w-full max-w-2xl">
        <div className="text-4xl font-bold mb-4 text-center">
          <span className="">Welcome to</span>{" "}
          <span className="text-base-content font-serif text-4xl font-bold">Mini Game</span>
        </div>
        {playerId ? (
          <div className="w-full">
            <p className="text-lg mb-6 text-center">Select a game to play:</p>
            <div className="flex flex-col items-center justify-center sm:flex-row sm:space-x-4 space-y-4 sm:space-y-0 w-full">
              <div className="flex flex-col sm:flex-row sm:space-x-4 space-y-4 sm:space-y-0 w-full">
                <Link
                  to="/tictactoe"
                  onClick={() => navigateToGame("/tictactoe")}
                  className="btn btn-outline flex-1"
                >
                  Tic Tac Toe
                </Link>
                <Link
                  to="/snake"
                  onClick={() => navigateToGame("/snake")}
                  className="btn btn-outline flex-1"
                >
                  Snake
                </Link>
                <Link
                  to="/chess"
                  onClick={() => navigateToGame("/chess")}
                  className="btn btn-outline flex-1"
                >
                  Chess
                </Link>
              </div>
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center w-full">
            <p className="text-lg mb-6 text-center">
              Before Playing the game, let's register!!
            </p>
            <RegisterUser onRegister={getPlayerId} />
          </div>
        )}
      </div>
    </div>
  );
};

export default HomePage;
