interface CellProps {
  value: string | null | number;
  onClick: () => void;
}

export default function Cell({ value, onClick }: CellProps) {
  return (
    <button
      onClick={onClick}
      className="w-16 h-16 border-4 border-base-content text-2xl hover:bg-secondary font-bold"
    >
      {value}
    </button>
  );
}
