/**
 * Shared types for watcher components
 */

import { Job } from "@/store/jobs";

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

export interface QuickFilterPreset {
  id: string;
  label: string;
  description: string;
  accentColor: "red" | "orange" | "yellow" | "green" | "cyan" | "blue" | "indigo" | "violet";
  apply: (current: JobFilters) => JobFilters;
}
