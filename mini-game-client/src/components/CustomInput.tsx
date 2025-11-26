import React from "react";

interface CustomInputProps {
  value: string;
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  maxLength?: number;
  className?: string;
}

export default function CustomInput({
  value,
  onChange,
  maxLength = 255,
  className = "",
}: CustomInputProps) {
  return (
    <input
      className={`input input-bordered outline outline-offset-4 ${className}`}
      value={value}
      onChange={onChange}
      maxLength={maxLength}
    />
  );
}
