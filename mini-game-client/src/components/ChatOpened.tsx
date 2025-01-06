import React, { useEffect, useRef, useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { id } from "date-fns/locale";

interface ChatOpenedProps {
  setOpenChatOpened: (open: boolean) => void;
  userName: string;
  messages: Array<{
    sender: string;
    message: string;
    timestamp: string;
  }>;
  onSendMessage: (message: string) => void;
}

function ChatOpened({
  setOpenChatOpened,
  userName,
  messages,
  onSendMessage,
}: ChatOpenedProps) {
  const [messageToSend, setMessageToSend] = useState("");
  const maxLetterLimit = 30;
  const [usersMessages, setUsersMessages] = useState(messages);

  // Ref untuk elemen container pesan
  const messagesEndRef = useRef<HTMLDivElement | null>(null);

  const isMessageTooLong = messageToSend.length > maxLetterLimit;

  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault();

    if (messageToSend.trim()) {
      const newMessage = {
        sender: userName,
        message: messageToSend.trim(),
        timestamp: new Date().toISOString(),
      };
      setUsersMessages((prevMessages) => [...prevMessages, newMessage]);
      onSendMessage(messageToSend.trim());
      setMessageToSend("");
    }
  };

  useEffect(() => {
    setUsersMessages(messages);
  }, [messages]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [usersMessages]);

  return (
    <div className="flex sm:pt-10 flex-col max-h-screen max-w-screen-sm rounded-lg overflow-hidden">
      {/* Header */}
      <div className="bg-gray-200 p-3 border-x border-t rounded-t-lg border-black flex justify-between items-center">
        <h3 className="text-lg font-semibold text-gray-900">Chat</h3>
        <button
          type="button"
          className="btn btn-sm btn-circle sm:hidden btn-ghost"
          onClick={() => setOpenChatOpened(false)}
          aria-label="Close chat"
        >
          <svg
            width="25px"
            height="25px"
            xmlns="http://www.w3.org/2000/svg"
            viewBox="-6 -6 24 24"
            fill="currentColor"
          >
            <path d="M7.314 5.9l3.535-3.536A1 1 0 1 0 9.435.95L5.899 4.485 2.364.95A1 1 0 1 0 .95 2.364l3.535 3.535L.95 9.435a1 1 0 1 0 1.414 1.414l3.535-3.535 3.536 3.535a1 1 0 1 0 1.414-1.414L7.314 5.899z" />
          </svg>
        </button>
      </div>

      {/* Messages */}
      <div className="flex flex-col border-x border-black p-4 h-5/6 min-h-96 max-h-96 overflow-auto scroll-smooth">
        {usersMessages.map((item, index) => (
          <div
            key={index}
            className={`chat ${
              item.sender == userName ? "chat-end" : "chat-start"
            }`}
          >
            <div className="chat-image avatar">
              <div className="w-9 rounded-full border bg-gray-300">
                <svg
                  width="36"
                  height="36"
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 100 100"
                  fill="currentColor"
                >
                  <g>
                    <ellipse cx="50" cy="36.5" rx="14.9" ry="16.5" />
                    <path d="M80,71.2V74c0,3.3-2.7,6-6,6H26c-3.3,0-6-2.7-6-6v-2.8c0-7.3,8.5-11.7,16.5-15.2c0.3-0.1,0.5-0.2,0.8-0.4 c0.6-0.3,1.3-0.3,1.9,0.1C42.4,57.8,46.1,59,50,59c3.9,0,7.6-1.2,10.8-3.2c0.6-0.4,1.3-0.4,1.9-0.1c0.3,0.1,0.5,0.2,0.8,0.4 C71.5,59.5,80,63.9,80,71.2z" />
                  </g>
                </svg>
              </div>
            </div>
            <div
              className={`chat-bubble ${
                item.sender === userName ? "chat-bubble-info" : ""
              }`}
            >
              {item.message}
            </div>
            <div className="chat-footer text-xs opacity-50">
              {formatDistanceToNow(new Date(item.timestamp), {
                addSuffix: true,
                locale: id,
              })}
            </div>
            <div ref={messagesEndRef}></div>
          </div>
        ))}
      </div>

      {/* Input */}
      <form
        className="flex p-3 border-x border-black border-b rounded-b-lg"
        onSubmit={handleSendMessage}
      >
        <div className="flex flex-col w-full mr-4">
          <input
            className={`input input-bordered input-primary w-full ${
              isMessageTooLong ? "border-red-500" : ""
            }`}
            type="text"
            placeholder="Type your message…"
            onChange={(e) => setMessageToSend(e.target.value)}
            value={messageToSend}
            maxLength={maxLetterLimit + 1} // Allow typing until limit
          />
          <span
            className={`text-sm mt-1 ${
              isMessageTooLong ? "text-red-500" : "text-gray-500"
            }`}
          >
            {messageToSend.length}/{maxLetterLimit} characters
          </span>
        </div>
        <button
          disabled={isMessageTooLong || messageToSend.trim() === ""}
          className={`btn btn-primary btn-outline ${
            isMessageTooLong ? "btn-disabled" : ""
          }`}
          type="submit"
        >
          <svg
            width="25px"
            height="25px"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
          >
            <path
              fill="currentColor"
              fillRule="evenodd"
              d="M2.345 2.245a1 1 0 0 1 1.102-.14l18 9a1 1 0 0 1 0 1.79l-18 9a1 1 0 0 1-1.396-1.211L4.613 13H10a1 1 0 1 0 0-2H4.613L2.05 3.316a1 1 0 0 1 .294-1.071z"
              clipRule="evenodd"
            />
          </svg>
        </button>
      </form>
    </div>
  );
}

export default ChatOpened;
