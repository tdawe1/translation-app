/**
 * JobFilterPanel - Advanced filtering UI for detected jobs
 *
 * Enhanced with Data Factory base components and consistent styling.
 * Provides comprehensive filtering options with preset quick filters.
 */

"use client";

import { useMemo } from "react";
import { BentoCard } from "@/components/ui/base/BentoCard";
import { Button } from "@/components/ui/base/Button";
import { DESIGN } from "@/lib/design/tokens";
import { cn } from "@/lib/utils";
import { QUICK_FILTERS } from "./utils/constants";
import { QuickFilterPresets } from "./QuickFilterPresets";
import { FilterSection } from "./FilterSection";
import { SourceFilter } from "./SourceFilter";
import { SortByFilter } from "./SortByFilter";
import { RewardRangeFilter } from "./RewardRangeFilter";
import { TimeFilter } from "./TimeFilter";
import { useRewardInput } from "@/lib/hooks/useRewardInput";
import { useActiveFiltersCount } from "@/lib/hooks/useActiveFiltersCount";
import type {
  JobFilters,
  QuickFilterPreset,
} from "./utils/types";

export type { JobFilters } from "./utils/types";

export interface JobFilterPanelProps {
  /** Current filter values */
  filters: JobFilters;
  /** Called when filters change */
  onFiltersChange: (filters: JobFilters) => void;
  /** Number of jobs matching current filters */
  matchingCount?: number;
  /** Total number of jobs */
  totalCount?: number;
  /** Whether to show compact version */
  compact?: boolean;
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export function JobFilterPanel({
  filters,
  onFiltersChange,
  matchingCount,
  totalCount,
  compact = false,
}: JobFilterPanelProps) {
  const activeFiltersCount = useActiveFiltersCount(filters);

  const rewardInput = useRewardInput({
    minReward: filters.minReward,
    maxReward: filters.maxReward,
    onRewardChange: (min, max) => {
      onFiltersChange({ ...filters, minReward: min, maxReward: max });
    },
  });

  const handleReset = () => {
    const resetFilters: JobFilters = {
      source: "all",
      sortBy: "newest",
      minReward: null,
      maxReward: null,
      timeFilter: "all",
      languagePairs: [],
    };
    onFiltersChange(resetFilters);
    rewardInput.setMinInput("");
    rewardInput.setMaxInput("");
  };

  const applyQuickFilter = (preset: QuickFilterPreset) => {
    const newFilters = preset.apply(filters);
    onFiltersChange(newFilters);
    rewardInput.setMinInput(newFilters.minReward?.toString() ?? "");
    rewardInput.setMaxInput(newFilters.maxReward?.toString() ?? "");
  };

  const updateFilter = <K extends keyof JobFilters>(key: K, value: JobFilters[K]) => {
    onFiltersChange({ ...filters, [key]: value });
  };

  const getActivePresetId = (): string | null => {
    if (activeFiltersCount === 0) return "all";
    if (filters.minReward === 10) return "high-value";
    if (filters.timeFilter === "today") return "new-today";
    if (filters.source === "rss") return "rss-only";
    if (filters.source === "websocket") return "websocket-only";
    if (filters.source === "external") return "external-only";
    return null;
  };

  const hasActiveFilters = activeFiltersCount > 0;
  const showStats = matchingCount !== undefined && totalCount !== undefined;

  if (compact) {
    return (
      <BentoCard
        accentColor="blue"
        className="p-4"
        hoverDisabled
      >
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2">
            <span className={cn("text-xs", DESIGN.typography.label)}>
              Filter:
            </span>
            {QUICK_FILTERS.slice(0, 4).map((preset) => (
              <button
                key={preset.id}
                onClick={() => applyQuickFilter(preset)}
                className={cn(
                  "px-3 py-1.5 text-xs font-mono border transition-colors duration-150",
                  "focus:outline-none focus:border-blue-600",
                  getActivePresetId() === preset.id
                    ? `border-${preset.accentColor}-600 ${DESIGN.colors.accent[preset.accentColor]}`
                    : "border-neutral-200 text-neutral-600 hover:border-blue-600"
                )}
              >
                {preset.label}
              </button>
            ))}
          </div>

          {hasActiveFilters && (
            <Button
              onClick={handleReset}
              variant="secondary"
              size="sm"
              className="ml-auto"
            >
              Reset ({activeFiltersCount})
            </Button>
          )}
        </div>
      </BentoCard>
    );
  }

  // Full version: comprehensive filter panel
  return (
    <BentoCard accentColor="blue" className="p-6" hoverDisabled>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h3 className={cn(
            "font-mono text-xs uppercase tracking-widest",
            DESIGN.colors.accent.blue
          )}>
            Filters
          </h3>
          {showStats && (
            <p className="text-xs text-neutral-500 mt-1">
              {matchingCount} of {totalCount} jobs match
            </p>
          )}
        </div>
        {hasActiveFilters && (
          <Button
            onClick={handleReset}
            variant="secondary"
            size="sm"
          >
            Reset ({activeFiltersCount})
          </Button>
        )}
      </div>

      <QuickFilterPresets
        activePresetId={getActivePresetId()}
        onApplyPreset={applyQuickFilter}
      />

      {/* Divider */}
      <hr className="border-neutral-200 my-6" />

      <div className="space-y-4">
        <FilterSection label="Source">
          <SourceFilter
            value={filters.source}
            onChange={(value) => updateFilter("source", value)}
          />
        </FilterSection>

        <FilterSection label="Sort By">
          <SortByFilter
            value={filters.sortBy}
            onChange={(value) => updateFilter("sortBy", value)}
          />
        </FilterSection>

        <FilterSection label="Reward Range">
          <RewardRangeFilter
            minInput={rewardInput.minInput}
            maxInput={rewardInput.maxInput}
            minReward={filters.minReward}
            onMinInputChange={rewardInput.setMinInput}
            onMaxInputChange={rewardInput.setMaxInput}
            onApplyReward={rewardInput.applyRewardRange}
            onQuickMinSelect={(min) => {
              rewardInput.setMinReward(min);
            }}
          />
        </FilterSection>

        <FilterSection label="Time Period">
          <TimeFilter
            value={filters.timeFilter}
            onChange={(value) => updateFilter("timeFilter", value)}
          />
        </FilterSection>
      </div>
    </BentoCard>
  );
}
