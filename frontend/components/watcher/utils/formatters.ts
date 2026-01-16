/**
 * Formatting helpers for watcher components
 */

import { Job } from "@/store/jobs";

// Get reward color based on value
export const getRewardColor = (reward: number): string => {
  if (reward >= 10) return "text-green-600";
  if (reward >= 5) return "text-yellow-600";
  return "text-neutral-600";
};

// Get source badge styles
export const getSourceBadge = (source: Job["source"]): string => {
  const styles = {
    rss: "bg-orange-50 border-orange-200 text-orange-700",
    websocket: "bg-blue-50 border-blue-200 text-blue-700",
  };
  return styles[source];
};

// Format timestamp to relative time (short version for lists)
export const formatTimeAgo = (timestamp?: string): string => {
  if (!timestamp) return "now";
  
  const date = new Date(timestamp);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (seconds < 60) return "now";
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  return `${Math.floor(seconds / 86400)}d`;
};

// Detailed format for modal
export const formatTimeAgoDetailed = (timestamp?: string): string => {
    if (!timestamp) return "Just now";

    const date = new Date(timestamp);
    const now = new Date();
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

    if (seconds < 60) return "Just now";
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
};
