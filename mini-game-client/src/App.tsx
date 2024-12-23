import { createBrowserRouter, RouterProvider } from "react-router-dom";
import MainLayout from "./components/shared/MainLayout";
import HomePage from "./pages/HomePage";
import TicTacToePage from "./pages/TicTacToePage";
import SnakePage from "./pages/SnakePage";
import NotFoundPage from "./pages/NotFoundPage";
import ChessPage from "./pages/ChessPage";
import Coba from "./pages/Coba";

export default function App() {
  const router = createBrowserRouter([
    {
      element: <MainLayout />,
      children: [
        {
          path: "/",
          element: <HomePage />,
        },
        {
          path: "/tictactoe",
          element: <TicTacToePage />,
        },
        {
          path: "/snake",
          element: <SnakePage />,
        },
        {
          path: "/chess",
          element: <ChessPage />,
        },
        {
          path: "/coba",
          element: <Coba />,
        },
        {
          path: "*",
          element: <NotFoundPage />,
        },
      ],
    },
  ]);

  return <RouterProvider router={router} />;
}
