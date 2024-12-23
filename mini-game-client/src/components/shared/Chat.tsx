import React, { useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { id } from "date-fns/locale";

interface ChatProps {
  setOpenChat: (open: boolean) => void;
  userName: string;
}

function Chat({ setOpenChat, userName }: ChatProps) {
  const [openAlert, setOpen] = useState(true);
  const [messageToSend, setMessageToSend] = useState("");
  const maxLetterLimit = 30;

  const handleClose = () => {
    setOpen(false);
    setOpenChat(false);
  };

  // Data dummy untuk pesan
  const usersMessages = [
    { sender: "Alice", message: "Hello!", time: new Date().toISOString() },
    { sender: "Bob", message: "Hi there!", time: new Date().toISOString() },
    { sender: userName, message: "How are you?", time: new Date().toISOString() },
  ];

  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault();

    if (messageToSend.length < 1) {
      return;
    }

    // Tambahkan pesan baru ke data dummy
    usersMessages.push({
      sender: userName,
      message: messageToSend,
      time: new Date().toISOString(),
    });

    setMessageToSend("");
  };

  return (
    <div className={`modal ${openAlert ? "modal-open" : ""}`}>
      <div className="modal-box w-11/12 max-w-2xl">
        <div className="flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900">Chat</h3>
          <button
            type="button"
            className="btn btn-sm btn-circle btn-ghost"
            onClick={handleClose}
          >
            ✕
          </button>
        </div>

        <div className="flex flex-col flex-grow h-96 p-4 overflow-auto">
          {usersMessages.map((item, index) => (
            <div key={index}>
              {item.sender !== userName ? (
                <div className="flex w-full mt-2 mb-2 space-x-3 max-w-xs">
                  <div className="flex items-center justify-center h-8 w-10 rounded-full bg-gray-200">
                    <span className="text-gray-700">👤</span>
                  </div>
                  <div>
                    <div className="bg-gray-300 p-3 rounded-r-lg rounded-bl-lg">
                      <p className="text-sm">{item.message}</p>
                    </div>
                    <span className="text-xs text-gray-500 leading-none">
                      {formatDistanceToNow(new Date(item.time), {
                        addSuffix: true,
                        locale: id,
                      })}
                    </span>
                  </div>
                </div>
              ) : (
                <div className="flex w-full mt-2 mb-2 space-x-3 max-w-xs ml-auto justify-end">
                  <div>
                    <div className="bg-blue-600 text-white p-3 rounded-l-lg rounded-br-lg">
                      <p className="text-sm">{item.message}</p>
                    </div>
                    <span className="text-xs text-gray-500 leading-none">
                      {formatDistanceToNow(new Date(item.time), {
                        addSuffix: true,
                        locale: id,
                      })}
                    </span>
                  </div>
                  <div className="flex items-center justify-center h-8 w-12 rounded-full bg-gray-200">
                    <span className="text-gray-700">👤</span>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>

        <div className="bg-gray-200 p-4">
          <form className="flex" onSubmit={handleSendMessage}>
            <div className="flex flex-col w-full mr-4">
              <input
                className="input input-bordered w-full"
                type="text"
                placeholder="Type your message…"
                onChange={(e) => setMessageToSend(e.target.value)}
                value={messageToSend}
              />
              <span
                className={`text-sm mt-1 ${
                  messageToSend.length > maxLetterLimit
                    ? "text-red-500"
                    : "text-gray-400"
                }`}
              >
                {messageToSend.length}/{maxLetterLimit} letters
              </span>
            </div>
            <button
              disabled={messageToSend.length > maxLetterLimit}
              className={`btn btn-primary ${
                messageToSend.length <= maxLetterLimit ? "" : "btn-disabled"
              }`}
              type="submit"
            >
              Send
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}

export default Chat;