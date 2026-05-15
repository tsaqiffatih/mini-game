// components/MoveHistory.tsx

interface ChessMoveHistoryProps {
  pgnMoves: string[];
  containerClassName?: string;
  scrollClassName?: string;
}

export default function ChessMoveHistory({
  pgnMoves,
  containerClassName = "",
  scrollClassName = "",
}: ChessMoveHistoryProps) {
  return (
    <div
      className={`border rounded p-2 text-sm md:text-lg ${containerClassName}`}
    >
      <h3 className="font-bold mb-2 border-b">Move History</h3>

      <div className={`overflow-y-auto ${scrollClassName}`}>
        <div className="text-sm md:text-base font-mono">
          {Array.from({ length: Math.ceil(pgnMoves.length / 2) }, (_, i) => {
            const whiteMove = pgnMoves[i * 2];
            const blackMove = pgnMoves[i * 2 + 1];

            return (
              <div
                key={i}
                className={`grid grid-cols-[30px_1fr_auto_1fr] gap-2 border-b border-base-300 ${
                  i % 2 === 0 ? "bg-black/20" : "bg-neutral/40"
                }`}
              >
                <span className="font-bold">{i + 1}.</span>

                <span>{whiteMove}</span>

                <span>|</span>

                <span>{blackMove ?? "-"}</span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
