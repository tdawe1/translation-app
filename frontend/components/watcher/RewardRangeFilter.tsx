import { cn } from "@/lib/utils";

interface RewardRangeFilterProps {
  minInput: string;
  maxInput: string;
  minReward: number | null;
  onMinInputChange: (value: string) => void;
  onMaxInputChange: (value: string) => void;
  onApplyReward: () => void;
  onQuickMinSelect: (min: number) => void;
}

const presets = [
  { label: "$5+", min: 5 },
  { label: "$10+", min: 10 },
  { label: "$20+", min: 20 },
] as const;

export function RewardRangeFilter({
  minInput,
  maxInput,
  minReward,
  onMinInputChange,
  onMaxInputChange,
  onApplyReward,
  onQuickMinSelect,
}: RewardRangeFilterProps) {
  return (
    <div>
      <div className="flex items-center gap-3">
        <div className="flex-1">
          <input
            type="number"
            placeholder="Min $"
            min="0"
            step="0.01"
            value={minInput}
            onChange={(e) => onMinInputChange(e.target.value)}
            onBlur={onApplyReward}
            onKeyDown={(e) => e.key === "Enter" && onApplyReward()}
            className={cn(
              "w-full px-3 py-2 text-sm font-mono border border-neutral-200",
              "focus:border-blue-600 focus:outline-none",
              "transition-colors duration-150"
            )}
          />
        </div>
        <span className="text-neutral-400">—</span>
        <div className="flex-1">
          <input
            type="number"
            placeholder="Max $"
            min="0"
            step="0.01"
            value={maxInput}
            onChange={(e) => onMaxInputChange(e.target.value)}
            onBlur={onApplyReward}
            onKeyDown={(e) => e.key === "Enter" && onApplyReward()}
            className={cn(
              "w-full px-3 py-2 text-sm font-mono border border-neutral-200",
              "focus:border-blue-600 focus:outline-none",
              "transition-colors duration-150"
            )}
          />
        </div>
      </div>
      <div className="flex gap-2 mt-2">
        {presets.map((preset) => (
          <button
            key={preset.label}
            onClick={() => onQuickMinSelect(preset.min)}
            className={cn(
              "px-2 py-1 text-xs font-mono border transition-colors duration-150",
              "focus:outline-none focus:border-green-600",
              minReward === preset.min
                ? "border-green-600 text-green-600"
                : "border-neutral-200 text-neutral-500 hover:border-green-600"
            )}
          >
            {preset.label}
          </button>
        ))}
      </div>
    </div>
  );
}
