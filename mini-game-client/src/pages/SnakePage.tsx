import { Link, useNavigate } from "react-router-dom";

const SnakePage = () => {
  const navigateToGame = useNavigate();

  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-base-100">
      <h1 className="text-4xl font-bold mb-4">
        Ups Snake Game Still Creating By Provider
      </h1>
      <Link
        to="/"
        className="btn btn-outline"
        onClick={() => navigateToGame("/")}
      >
        Back To Home
      </Link>
    </div>
  );
};

export default SnakePage;
