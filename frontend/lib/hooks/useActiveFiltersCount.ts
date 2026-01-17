import { useMemo } from "react";
import type { JobFilters } from "@/components/watcher/utils/types";

export function useActiveFiltersCount(filters: JobFilters) {
  return useMemo(() => {
    let count = 0;
    if (filters.source !== "all") count++;
    if (filters.minReward !== null) count++;
    if (filters.maxReward !== null) count++;
    if (filters.timeFilter !== "all") count++;
    if (filters.languagePairs.length > 0) count++;
    return count;
  }, [filters]);
}
