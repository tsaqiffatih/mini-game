import React from "react";

interface CustomButtonProps {
  onClick: () => void;
  disabled?: boolean;
  className?: string;
  children: React.ReactNode;
}

export default function CustomButton({
  onClick,
  disabled = false,
  className = "",
  children,
}: CustomButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`btn shadow-xl outline outline-offset-4 ${className}`}
      disabled={disabled}
    >
      {children}
    </button>
  );
}
