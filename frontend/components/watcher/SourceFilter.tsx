import { cn } from "@/lib/utils";
import type { FilterSource } from "./utils/types";

interface SourceFilterProps {
  value: FilterSource;
  onChange: (value: FilterSource) => void;
}

const options = [
  { value: "all" as const, label: "All Sources" },
  { value: "rss" as const, label: "RSS Feed" },
  { value: "websocket" as const, label: "WebSocket" },
  { value: "external" as const, label: "External Bridge" },
] as const;

export function SourceFilter({ value, onChange }: SourceFilterProps) {
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
