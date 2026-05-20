

export const getGameStorageKeys = (gameType: string) => ({
  roomId: `${gameType}.roomId`,
  playerMark: `${gameType}.playerMark`,
  aiLevel: `${gameType}.aiLevel`,
});

export const saveGameSession = ({
  gameType,
  roomId,
  playerMark,
  aiLevel,
}: {
  gameType: string;
  roomId: string;
  playerMark: string;
  aiLevel?: number;
}) => {
  const keys = getGameStorageKeys(gameType);

  localStorage.setItem(keys.roomId, roomId);
  localStorage.setItem(keys.playerMark, playerMark);

  if (aiLevel !== undefined) {
    localStorage.setItem(keys.aiLevel, String(aiLevel));
  }
};

export const clearGameSession = (gameType: string) => {
  const keys = getGameStorageKeys(gameType);

  localStorage.removeItem(keys.roomId);
  localStorage.removeItem(keys.playerMark);
  localStorage.removeItem(keys.aiLevel);
};

export const getGameSession = (gameType: string) => {
  const keys = getGameStorageKeys(gameType);

  return {
    roomId: localStorage.getItem(keys.roomId),
    playerMark: localStorage.getItem(keys.playerMark),
    aiLevel: localStorage.getItem(keys.aiLevel),
  };
};