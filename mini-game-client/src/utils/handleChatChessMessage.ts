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

interface HandleChatChessMessageParams {
  chat: unknown;

  setChatMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;

  isChatOpenRef: React.MutableRefObject<boolean>;

  setHasNewMessage: (value: boolean) => void;
}

const isChatSnapshot = (value: unknown): value is ChatSnapshot => {
  if (!value || typeof value !== "object") return false;

  const chat = value as Record<string, unknown>;
  return (
    typeof chat.id === "string" &&
    typeof chat.player_id === "string" &&
    (chat.player_mark === undefined || typeof chat.player_mark === "string") &&
    typeof chat.message === "string" &&
    typeof chat.created_at === "string"
  );
};

export const handleChatChessMessage = ({
  chat,
  setChatMessages,
  isChatOpenRef,
  setHasNewMessage,
}: HandleChatChessMessageParams) => {
  if (!isChatSnapshot(chat)) return;

  setChatMessages((prev) => {
    const exists = prev.some((m) => m.id === chat.id);

    if (exists) return prev;

    return [
      ...prev,
      {
        id: chat.id,
        sender: chat.player_id,
        playerMark: chat.player_mark,
        message: chat.message,
        timestamp: chat.created_at,
      },
    ];
  });

  if (!isChatOpenRef.current) {
    setHasNewMessage(true);
  }
};
