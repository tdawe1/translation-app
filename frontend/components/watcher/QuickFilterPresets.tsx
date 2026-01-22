import { cn } from "@/lib/utils";
import { DESIGN } from "@/lib/design/tokens";
import { QUICK_FILTERS } from "./utils/constants";
import type { QuickFilterPreset } from "./utils/types";

interface QuickFilterPresetsProps {
  activePresetId: string | null;
  onApplyPreset: (preset: QuickFilterPreset) => void;
}

export function QuickFilterPresets({ activePresetId, onApplyPreset }: QuickFilterPresetsProps) {
  return (
    <div className="mb-6">
      <p className={cn("text-xs mb-3", DESIGN.typography.label)}>
        Quick Presets
      </p>
      <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
        {QUICK_FILTERS.map((preset, index) => {
          const isActive = preset.id === activePresetId;

          return (
            <button
              key={preset.id}
              onClick={() => onApplyPreset(preset)}
              className={cn(
                "p-3 border text-left transition-colors duration-150",
                "focus:outline-none focus:border-blue-600",
                "hover:border-blue-600",
                isActive
                  ? `border-${preset.accentColor}-600 ${DESIGN.colors.accent[preset.accentColor]} bg-neutral-50`
                  : "border-neutral-200"
              )}
              style={{
                animationDelay: DESIGN.getStaggerDelay(index),
              }}
            >
              <p className={cn(
                "text-sm font-medium mb-0.5",
                isActive ? DESIGN.colors.accent[preset.accentColor] : "text-neutral-700"
              )}>
                {preset.label}
              </p>
              <p className="text-xs text-neutral-500">
                {preset.description}
              </p>
            </button>
          );
        })}
      </div>
    </div>
  );
}
