/**
 * Constants for watcher components
 */

import { QuickFilterPreset } from "./types";
import { DESIGN } from "@/lib/design/tokens";

export const QUICK_FILTERS: QuickFilterPreset[] = [
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
  {
    id: "external-only",
    label: "External",
    description: "External bridge source only",
    accentColor: "violet",
    apply: (current) => ({
      ...current,
      source: "external",
    }),
  },
];
