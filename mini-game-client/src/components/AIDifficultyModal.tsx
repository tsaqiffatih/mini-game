"use client";

interface Props {
  isOpen: boolean;
  onClose: () => void;
  gameType: string;
  aiLevel: number;
  setAiLevel: (level: number) => void;
  onStart: () => void;
  isLoading: boolean;
}

const aiConfig = {
  tictactoe: {
    maxLevel: 10,
    label: "AI Difficulty",
  },
  chess: {
    maxLevel: 10,
    label: "Stockfish Level",
  },
};

export default function AIDifficultyModal({
  isOpen,
  onClose,
  gameType,
  aiLevel,
  setAiLevel,
  onStart,
  isLoading,
}: Props) {
  if (!isOpen) return null;

  const currentAIConfig = aiConfig[gameType as keyof typeof aiConfig];

  const getDifficultyLabel = () => {
    if (gameType === "chess") {
      if (aiLevel <= 2) return "Beginner";
      if (aiLevel <= 4) return "Easy";
      if (aiLevel <= 7) return "Intermediate";
      if (aiLevel <= 9) return "Hard";

      return "Master";
    }

    if (aiLevel <= 3) return "Easy";
    if (aiLevel <= 7) return "Medium";

    return "Impossible";
  };

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 px-4">
      <div className="bg-base-100 rounded-2xl p-6 w-full max-w-md border border-primary shadow-2xl">
        <h2 className="text-2xl font-bold text-center mb-6">Play With AI</h2>

        <div className="mb-6">
          <div className="flex items-center justify-between mb-2">
            <span className="font-semibold">{currentAIConfig.label}</span>

            <span className="badge badge-primary">Lv. {aiLevel}</span>
          </div>

          <input
            type="range"
            min={1}
            max={currentAIConfig.maxLevel}
            value={aiLevel}
            onChange={(e) => setAiLevel(Number(e.target.value))}
            className="range range-primary"
          />

          <div className="flex justify-between text-xs opacity-70 mt-1">
            <span>Easy</span>
            <span>Hard</span>
          </div>

          <p className="text-center text-sm mt-3 opacity-80">
            {getDifficultyLabel()}
          </p>
        </div>

        <div className="flex justify-end gap-3">
          <button
            className="btn btn-ghost"
            onClick={onClose}
            disabled={isLoading}
          >
            Cancel
          </button>

          <button
            className="btn btn-primary"
            onClick={onStart}
            disabled={isLoading}
          >
            {!isLoading ? (
              "Start Game"
            ) : (
              <>
                <span className="loading loading-spinner loading-sm mr-2"></span>
                Loading
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
