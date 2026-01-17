import { cn } from "@/lib/utils";
import type { SortBy } from "./utils/types";

interface SortByFilterProps {
  value: SortBy;
  onChange: (value: SortBy) => void;
}

const options = [
  { value: "newest" as const, label: "Newest First" },
  { value: "reward-high" as const, label: "Reward: High → Low" },
  { value: "reward-low" as const, label: "Reward: Low → High" },
] as const;

export function SortByFilter({ value, onChange }: SortByFilterProps) {
  return (
    <div className="flex flex-wrap gap-2">
      {options.map((option) => (
        <button
          key={option.value}
          onClick={() => onChange(option.value)}
          className={cn(
            "px-4 py-2 text-sm font-mono border transition-colors duration-150",
            "focus:outline-none focus:border-blue-600",
            value === option.value
              ? "border-blue-600 text-blue-600"
              : "border-neutral-200 text-neutral-600 hover:border-blue-600"
          )}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}
