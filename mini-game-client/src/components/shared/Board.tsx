import Cell from "./Cell";

interface BoardProps {
  board: (string | null)[][];
  onCellClick: (rowIndex: number, colIndex: number) => void;
}

export default function Board({ board, onCellClick }: BoardProps) {
  return (
    <div className="grid grid-cols-3 gap-2 mt-3">
      {board.map((row, rowIndex) =>
        row.map((cell, colIndex) => (
          <Cell
            key={`${rowIndex}-${colIndex}`}
            value={cell}
            onClick={() => onCellClick(rowIndex, colIndex)}
          />
        ))
      )}
    </div>
  );
};

