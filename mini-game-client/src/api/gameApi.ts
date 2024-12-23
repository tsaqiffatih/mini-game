import axios from "axios";

const API_BASE = "http://localhost:8080";

interface Move {
  roomId: string;
  playerId: string;
  row: number;
  col: number;
}

interface MoveCoba {
  playerId: string;
  row: number;
  col: number;
}

// API untuk menambahkan pemain
export const addPlayer = async (playerId: string) => {
  const response = await axios.post(`${API_BASE}/create/user`, {
    player_id: playerId,
  });
  return response.data;
};

// API untuk membuat room
export const createRoom = async (gameType: string, playerId: string) => {
  const response = await axios.post(`${API_BASE}/room/create`, {
    game_type: gameType,
    player_id: playerId,
  });
  return response.data;
};

// API untuk bergabung ke room
export const joinRoom = async (roomId: string, playerId: string) => {
  const response = await axios.post(`${API_BASE}/room/join`, {
    room_id: roomId,
    player_id: playerId,
  });
  return response.data;
};

// kebawah gak dipake
export const addPlayerCoba = async (playerId: string) => {
  const response = await axios.post(`${API_BASE}/players`, {
    player_id: playerId,
  });

  return response.data;
};

// API untuk mendapatkan state permainan
export const getGameState = async (roomId: string) => {
  const response = await axios.get(`${API_BASE}/game/state/${roomId}`);
  return response.data;
};

export const getGameStateCoba = async () => {
  const response = await axios.get(`${API_BASE}/game/state`);
  return response.data;
};

// // API untuk membuat gerakan
export const makeMove = async ({ roomId, playerId, row, col }: Move) => {
  const response = await axios.post(`${API_BASE}/game/move`, {
    room_id: roomId,
    player_id: playerId,
    row,
    col,
  });
  return response.data;
};

export const makeMoveCoba = async ({ playerId, row, col }: MoveCoba) => {
  const response = await axios.post(`${API_BASE}/game/move`, {
    player_id: playerId,
    row,
    col,
  });
  return response.data;
};
