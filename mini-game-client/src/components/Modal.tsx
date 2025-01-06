'use client'

import React, { useEffect, useState } from "react";

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}

export default function Modal({
  isOpen,
  onClose,
  children,
  title,
}: ModalProps) {
  // Prevent rendering on the server
  const [isClient, setIsClient] = useState(false);

  useEffect(() => {
    setIsClient(true);
  }, []);

  if (!isClient || !isOpen) return null;

  return (
    <dialog
      className="modal"
      open
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="modal-box bg-base-100">
        <h3 className="font-bold text-lg mb-4 text-base-content">{title}</h3>
        {children}
      </div>
    </dialog>
  );
}
