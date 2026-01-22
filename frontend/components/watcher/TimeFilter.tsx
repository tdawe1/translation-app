import { cn } from "@/lib/utils";
import type { TimeFilter } from "./utils/types";

interface TimeFilterProps {
  value: TimeFilter;
  onChange: (value: TimeFilter) => void;
}

const options = [
  { value: "all" as const, label: "All Time" },
  { value: "hour" as const, label: "Last Hour" },
  { value: "today" as const, label: "Today" },
  { value: "week" as const, label: "This Week" },
] as const;

export function TimeFilter({ value, onChange }: TimeFilterProps) {
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
