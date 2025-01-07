# Mini Game Project

The Mini Game Project is a collection of simple board games currently featuring **Chess** and **Tic-Tac-Toe**. Designed for two players in **multiplayer mode**, the project offers a **realtime** experience for enjoyable gameplay. Additionally, it includes a chat feature that allows players to communicate during matches.

---

## Key Features

- **Multiplayer**: Two-player games with realtime connection.
- **Realtime Updates**: Live updates for moves and chats.
- **Chat Functionality**: Players can communicate while playing.
- **Cross-Device Compatibility**: Playable on both desktop and mobile devices.

---

## Technologies Used

### Frontend
- **[Next.js](https://nextjs.org/)**: Framework for building user interfaces.
- **[Chess.js](https://github.com/jhlywa/chess.js)**: Handles the logic for chess gameplay.
- **[react-chessboard](https://github.com/Clariity/react-chessboard)**: Interactive chessboard component for React.js.
- **[react-use-websocket](https://github.com/robtaussig/react-use-websocket)**: WebSocket implementation for the frontend.
- **[date-fns](https://date-fns.org/)**: Time management, such as timestamps for chats.
- **[sweetalert2](https://sweetalert2.github.io/)**: Interactive alerts and confirmation dialogs.

### Backend
- **[Golang](https://golang.org/)**: Main backend for game and communication management.
- **[gorilla/websocket](https://github.com/gorilla/websocket)**: WebSocket implementation for the backend.
- **[golang.org/x/time](https://pkg.go.dev/golang.org/x/time)**: Library for managing time-based features like rate limiters.
- **[gorilla/handlers](https://github.com/gorilla/handlers)**: Middleware for CORS and other requirements.

---

## How to Run the Project

### Backend Setup

#### Using Air (Development with Live Reload)
`Air` is a tool for live reloading during Go development. To use Air:
1. Install Air globally:
   ```bash
   go install github.com/cosmtrek/air@latest
   ```
   For more details on installation, visit the official [Air GitHub Repository](https://github.com/cosmtrek/air).

2. Navigate to the server directory:
   ```bash
   cd server
   ```
3. Initialize Air (if not already initialized):
   ```bash
   air init
   ```
4. Run the server with Air:
   ```bash
   air
   ```

#### Without Air
To run the server manually:
1. Navigate to the server directory:
   ```bash
   cd server
   ```
2. Run the server directly:
   ```bash
   go run main.go
   ```

---

### Frontend Setup

1. Navigate to the client directory:
   ```bash
   cd mini-game-client
   ```
2. Install dependencies:
   ```bash
   npm install
   ```
3. Start the development server:
   ```bash
   npm run dev
   ```

---

## License

This project is licensed under the MIT license. See the [LICENSE](./LICENSE) file for more details.

---

## Screenshots
![HomePage](https://i.imgur.com/VVJGRiL.png)
<br></br>
![Tictactoe](https://i.imgur.com/sVKH6Gx.png)
---

## Demo
The project is deployed and can be accessed via the following link:

[Mini Game Project Demo](https://mini-game-tau.vercel.app/)

---

## Contact the Developer

If you have any questions, feedback, or suggestions, feel free to reach out through one of the following:

- Open an issue on [GitHub Issues](https://github.com/tsaqiffatih/mini-game/issues).
- Send an email to [tsaqiffatih@gmail.com](mailto:tsaqiffatih@gmail.com).

Your feedback is highly appreciated, and I'm happy to collaborate or assist with any related queries!

---

## Additional Notes

- The game is still in its early stages of development; additional features may be added in the future.
- Feedback and contributions are highly appreciated.

---

Thank you for trying the Mini Game Project! Happy playing! 🎮

