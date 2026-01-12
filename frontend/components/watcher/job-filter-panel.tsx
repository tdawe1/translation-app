/**
 * JobFilterPanel - Advanced filtering UI for detected jobs
 *
 * Enhanced with Data Factory base components and consistent styling.
 * Provides comprehensive filtering options with preset quick filters.
 */

"use client";

import { useState, useCallback, useMemo } from "react";
import { BentoCard } from "@/components/ui/base/BentoCard";
import { Button } from "@/components/ui/base/Button";
import { DESIGN } from "@/lib/design/tokens";
import { cn } from "@/lib/utils";

// ============================================================================
// TYPES
// ============================================================================

export type FilterSource = "all" | "rss" | "websocket";
export type SortBy = "newest" | "reward-high" | "reward-low";
export type TimeFilter = "all" | "today" | "hour" | "week";

export interface JobFilters {
  source: FilterSource;
  sortBy: SortBy;
  minReward: number | null;
  maxReward: number | null;
  timeFilter: TimeFilter;
  languagePairs: string[]; // For future use
}

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

interface QuickFilterPreset {
  id: string;
  label: string;
  description: string;
  accentColor: "red" | "orange" | "yellow" | "green" | "cyan" | "blue" | "indigo" | "violet";
  apply: (current: JobFilters) => JobFilters;
}

// ============================================================================
// QUICK FILTER PRESETS
// ============================================================================

const QUICK_FILTERS: QuickFilterPreset[] = [
  {
    id: "all",
    label: "All Jobs",
    description: "Show all detected jobs",
    accentColor: "blue",
    apply: () => ({
      source: "all",
      sortBy: "newest",
      minReward: null,
      maxReward: null,
      timeFilter: "all",
      languagePairs: [],
    }),
  },
  {
    id: "high-value",
    label: "High Value",
    description: "$10+ reward only",
    accentColor: "green",
    apply: (current) => ({
      ...current,
      minReward: 10,
      maxReward: null,
    }),
  },
  {
    id: "new-today",
    label: "New Today",
    description: "Last 24 hours",
    accentColor: "cyan",
    apply: (current) => ({
      ...current,
      timeFilter: "today",
    }),
  },
  {
    id: "rss-only",
    label: "RSS Feed",
    description: "RSS source only",
    accentColor: "orange",
    apply: (current) => ({
      ...current,
      source: "rss",
    }),
  },
  {
    id: "websocket-only",
    label: "WebSocket",
    description: "WebSocket source only",
    accentColor: "indigo",
    apply: (current) => ({
      ...current,
      source: "websocket",
    }),
  },
];

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
  // Local state for reward range inputs
  const [minRewardInput, setMinRewardInput] = useState(
    filters.minReward?.toString() ?? ""
  );
  const [maxRewardInput, setMaxRewardInput] = useState(
    filters.maxReward?.toString() ?? ""
  );

  // Calculate active filters count
  const activeFiltersCount = useMemo(() => {
    let count = 0;
    if (filters.source !== "all") count++;
    if (filters.minReward !== null) count++;
    if (filters.maxReward !== null) count++;
    if (filters.timeFilter !== "all") count++;
    if (filters.languagePairs.length > 0) count++;
    return count;
  }, [filters]);

  // Reset all filters
  const handleReset = useCallback(() => {
    const resetFilters: JobFilters = {
      source: "all",
      sortBy: "newest",
      minReward: null,
      maxReward: null,
      timeFilter: "all",
      languagePairs: [],
    };
    onFiltersChange(resetFilters);
    setMinRewardInput("");
    setMaxRewardInput("");
  }, [onFiltersChange]);

  // Apply quick filter preset
  const applyQuickFilter = useCallback(
    (preset: QuickFilterPreset) => {
      const newFilters = preset.apply(filters);
      onFiltersChange(newFilters);
      setMinRewardInput(newFilters.minReward?.toString() ?? "");
      setMaxRewardInput(newFilters.maxReward?.toString() ?? "");
    },
    [filters, onFiltersChange]
  );

  // Update single filter
  const updateFilter = useCallback(
    <K extends keyof JobFilters>(key: K, value: JobFilters[K]) => {
      onFiltersChange({ ...filters, [key]: value });
    },
    [filters, onFiltersChange]
  );

  // Apply reward range
  const applyRewardRange = useCallback(() => {
    const min = minRewardInput ? parseFloat(minRewardInput) : null;
    const max = maxRewardInput ? parseFloat(maxRewardInput) : null;
    onFiltersChange({
      ...filters,
      minReward: min,
      maxReward: max,
    });
  }, [filters, minRewardInput, maxRewardInput, onFiltersChange]);

  const hasActiveFilters = activeFiltersCount > 0;
  const showStats = matchingCount !== undefined && totalCount !== undefined;

  // Compact version: horizontal filter bar
  if (compact) {
    return (
      <BentoCard
        accentColor="blue"
        className="p-4"
        hoverDisabled
      >
        <div className="flex flex-wrap items-center gap-3">
          {/* Quick Filters */}
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
                  filters.source === preset.id ||
                    (preset.id === "high-value" && filters.minReward === 10) ||
                    (preset.id === "new-today" && filters.timeFilter === "today") ||
                    (preset.id === "rss-only" && filters.source === "rss") ||
                    (preset.id === "websocket-only" && filters.source === "websocket")
                    ? `border-${preset.accentColor}-600 ${DESIGN.colors.accent[preset.accentColor]}`
                    : "border-neutral-200 text-neutral-600 hover:border-blue-600"
                )}
              >
                {preset.label}
              </button>
            ))}
          </div>

          {/* Reset Button */}
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

      {/* Quick Filter Presets */}
      <div className="mb-6">
        <p className={cn("text-xs mb-3", DESIGN.typography.label)}>
          Quick Presets
        </p>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
          {QUICK_FILTERS.map((preset, index) => {
            const isActive =
              (preset.id === "all" && activeFiltersCount === 0) ||
              (preset.id === "high-value" && filters.minReward === 10) ||
              (preset.id === "new-today" && filters.timeFilter === "today") ||
              (preset.id === "rss-only" && filters.source === "rss") ||
              (preset.id === "websocket-only" && filters.source === "websocket");

            return (
              <button
                key={preset.id}
                onClick={() => applyQuickFilter(preset)}
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

      {/* Divider */}
      <hr className="border-neutral-200 my-6" />

      {/* Detailed Filters */}
      <div className="space-y-4">
        {/* Source Filter */}
        <div>
          <label className={cn("block mb-2", DESIGN.typography.label)}>
            Source
          </label>
          <div className="flex flex-wrap gap-2">
            {[
              { value: "all" as const, label: "All Sources" },
              { value: "rss" as const, label: "RSS Feed" },
              { value: "websocket" as const, label: "WebSocket" },
            ].map((option) => (
              <button
                key={option.value}
                onClick={() => updateFilter("source", option.value)}
                className={cn(
                  "px-4 py-2 text-sm font-mono border transition-colors duration-150",
                  "focus:outline-none focus:border-blue-600",
                  filters.source === option.value
                    ? "border-blue-600 text-blue-600"
                    : "border-neutral-200 text-neutral-600 hover:border-blue-600"
                )}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>

        {/* Sort By */}
        <div>
          <label className={cn("block mb-2", DESIGN.typography.label)}>
            Sort By
          </label>
          <div className="flex flex-wrap gap-2">
            {[
              { value: "newest" as const, label: "Newest First" },
              { value: "reward-high" as const, label: "Reward: High → Low" },
              { value: "reward-low" as const, label: "Reward: Low → High" },
            ].map((option) => (
              <button
                key={option.value}
                onClick={() => updateFilter("sortBy", option.value)}
                className={cn(
                  "px-4 py-2 text-sm font-mono border transition-colors duration-150",
                  "focus:outline-none focus:border-blue-600",
                  filters.sortBy === option.value
                    ? "border-blue-600 text-blue-600"
                    : "border-neutral-200 text-neutral-600 hover:border-blue-600"
                )}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>

        {/* Reward Range */}
        <div>
          <label className={cn("block mb-2", DESIGN.typography.label)}>
            Reward Range
          </label>
          <div className="flex items-center gap-3">
            <div className="flex-1">
              <input
                type="number"
                placeholder="Min $"
                min="0"
                step="0.01"
                value={minRewardInput}
                onChange={(e) => setMinRewardInput(e.target.value)}
                onBlur={applyRewardRange}
                onKeyDown={(e) => e.key === "Enter" && applyRewardRange()}
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
                value={maxRewardInput}
                onChange={(e) => setMaxRewardInput(e.target.value)}
                onBlur={applyRewardRange}
                onKeyDown={(e) => e.key === "Enter" && applyRewardRange()}
                className={cn(
                  "w-full px-3 py-2 text-sm font-mono border border-neutral-200",
                  "focus:border-blue-600 focus:outline-none",
                  "transition-colors duration-150"
                )}
              />
            </div>
          </div>
          {/* Quick reward presets */}
          <div className="flex gap-2 mt-2">
            {[
              { label: "$5+", min: 5 },
              { label: "$10+", min: 10 },
              { label: "$20+", min: 20 },
            ].map((preset) => (
              <button
                key={preset.label}
                onClick={() => {
                  setMinRewardInput(preset.min.toString());
                  onFiltersChange({ ...filters, minReward: preset.min });
                }}
                className={cn(
                  "px-2 py-1 text-xs font-mono border transition-colors duration-150",
                  "focus:outline-none focus:border-green-600",
                  filters.minReward === preset.min
                    ? "border-green-600 text-green-600"
                    : "border-neutral-200 text-neutral-500 hover:border-green-600"
                )}
              >
                {preset.label}
              </button>
            ))}
          </div>
        </div>

        {/* Time Filter */}
        <div>
          <label className={cn("block mb-2", DESIGN.typography.label)}>
            Time Period
          </label>
          <div className="flex flex-wrap gap-2">
            {[
              { value: "all" as const, label: "All Time" },
              { value: "hour" as const, label: "Last Hour" },
              { value: "today" as const, label: "Today" },
              { value: "week" as const, label: "This Week" },
            ].map((option) => (
              <button
                key={option.value}
                onClick={() => updateFilter("timeFilter", option.value)}
                className={cn(
                  "px-4 py-2 text-sm font-mono border transition-colors duration-150",
                  "focus:outline-none focus:border-blue-600",
                  filters.timeFilter === option.value
                    ? "border-blue-600 text-blue-600"
                    : "border-neutral-200 text-neutral-600 hover:border-blue-600"
                )}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>
      </div>
    </BentoCard>
  );
}

// ============================================================================
// HELPER: Apply filters to job list
// ============================================================================

/**
 * Filter jobs based on the provided filter criteria.
 * This is a pure function that can be used independently of the UI.
 */
export interface Job {
  id: string;
  title: string;
  reward: number;
  url: string;
  source: "rss" | "websocket";
  timestamp?: string;
}

export function filterJobs(jobs: Job[], filters: JobFilters): Job[] {
  let result = [...jobs];

  // Filter by source
  if (filters.source !== "all") {
    result = result.filter((job) => job.source === filters.source);
  }

  // Filter by reward range
  if (filters.minReward !== null) {
    result = result.filter((job) => job.reward >= filters.minReward!);
  }
  if (filters.maxReward !== null) {
    result = result.filter((job) => job.reward <= filters.maxReward!);
  }

  // Filter by time
  if (filters.timeFilter !== "all") {
    const now = new Date();
    const cutoff = new Date();

    switch (filters.timeFilter) {
      case "hour":
        cutoff.setHours(now.getHours() - 1);
        break;
      case "today":
        cutoff.setHours(0, 0, 0, 0);
        break;
      case "week":
        cutoff.setDate(now.getDate() - 7);
        break;
    }

    result = result.filter((job) => {
      if (!job.timestamp) return true;
      const jobDate = new Date(job.timestamp);
      return jobDate >= cutoff;
    });
  }

  // Sort
  switch (filters.sortBy) {
    case "reward-high":
      result.sort((a, b) => b.reward - a.reward);
      break;
    case "reward-low":
      result.sort((a, b) => a.reward - b.reward);
      break;
    case "newest":
    default:
      // Jobs are assumed to be in newest-first order
      break;
  }

  return result;
}
