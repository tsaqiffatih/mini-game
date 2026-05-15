type ChatMessage = {
  id: string;
  sender: string;
  playerMark?: string;
  message: string;
  timestamp: string;
};

type ChatSnapshot = {
  id: string;
  player_id: string;
  player_mark?: string;
  message: string;
  created_at: string;
};

interface HandleChatChessHistoryParams {
  messages: ChatSnapshot[];

  setChatMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
}

export const handleChatChessHistory = ({
  messages,
  setChatMessages,
}: HandleChatChessHistoryParams) => {
  setChatMessages((prev) => {
    const map = new Map<string, ChatMessage>();

    [...prev, ...messages].forEach((m) => {
      const normalized: ChatMessage =
        "sender" in m
          ? m
          : {
              id: m.id,
              sender: m.player_id,
              playerMark: m.player_mark,
              message: m.message,
              timestamp: m.created_at,
            };

      map.set(m.id, {
        id: normalized.id,
        sender: normalized.sender,
        playerMark: normalized.playerMark,
        message: normalized.message,
        timestamp: normalized.timestamp,
      });
    });

    return Array.from(map.values()).sort(
      (a, b) =>
        new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
    );
  });
};
