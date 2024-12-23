import { useState } from "react";

interface WaitingProps {
  roomId: string;
}

export default function Waiting({ roomId }: WaitingProps) {
  const [copySuccess, setCopySuccess] = useState("");

  const copyToClipboard = () => {
    navigator.clipboard.writeText(roomId).then(
      () => {
        setCopySuccess("Copied!");
        setTimeout(() => setCopySuccess(""), 2000); // Reset message after 2 seconds
      },
      () => {
        setCopySuccess("Failed to copy!");
      }
    );
  };

  return (
    <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 ring ring-primary rounded-lg w-80 shadow-lg">
      <div className="p-4">
        <div className="flex flex-col items-center text-primary py-6 text-center">
          <p className="text-primary mb-2">
            Share this Room ID with your friend:
          </p>
          <div className="flex mb-9">
            <div className="bg-primary-content p-4 rounded-lg">
              <p className=" font-bold text-primary tracking-widest">
                {roomId}
              </p>
            </div>
            <button onClick={copyToClipboard} className="text-primary ml-1 tooltip" data-tip="copy">
              <svg
                // fill="#000000"
                width="25px"
                height="25px"
                viewBox="-5 -2 24 24"
                xmlns="http://www.w3.org/2000/svg"
                preserveAspectRatio="xMinYMin"
                className="fill-current"
              >
                <path d="M5 2v2h4V2H5zm6 0h1a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h1a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2zm0 2a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2H2v14h10V4h-1zM4 8h6a1 1 0 0 1 0 2H4a1 1 0 1 1 0-2zm0 5h6a1 1 0 0 1 0 2H4a1 1 0 0 1 0-2z" />
              </svg>
            </button>
          </div>

          {copySuccess && <p className="text-green-500">{copySuccess}</p>}
          <div className="flex justify-center items-center">
            <span className="loading loading-spinner loading-lg"></span>
          </div>
          <p className="text-primary mt-4 mb-4">Waiting for players...</p>
        </div>
      </div>
    </div>
  );
}
