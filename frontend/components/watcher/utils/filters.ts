/**
 * Filtering logic for jobs
 */

import { Job } from "@/store/jobs";
import { JobFilters } from "./types";

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
