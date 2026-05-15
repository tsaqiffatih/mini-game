type ChatMessage = {
  id: string;
  sender: string;
  playerMark?: string;
  message: string;
  timestamp: string;
};

interface HandleChatChessMessageParams {
  chat: any;

  setChatMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;

  isChatOpenRef: React.MutableRefObject<boolean>;

  setHasNewMessage: (value: boolean) => void;
}

export const handleChatChessMessage = ({
  chat,
  setChatMessages,
  isChatOpenRef,
  setHasNewMessage,
}: HandleChatChessMessageParams) => {
  if (!chat) return;

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
