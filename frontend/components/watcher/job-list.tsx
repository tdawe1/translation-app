/**
 * JobList - Displays detected jobs with filtering
 */

import { useState, useMemo } from "react";
import { useJobsStore } from "@/store/jobs";
import type { Job } from "@/store/jobs";

type FilterSource = "all" | "rss" | "websocket";
type SortBy = "newest" | "reward-high" | "reward-low";

export function JobList() {
  const jobs = useJobsStore((state) => state.jobs);
  const clearJobs = useJobsStore((state) => state.clearJobs);

  const [filterSource, setFilterSource] = useState<FilterSource>("all");
  const [sortBy, setSortBy] = useState<SortBy>("newest");
  const [minRewardFilter, setMinRewardFilter] = useState<number | null>(null);

  // Filter and sort jobs
  const filteredJobs = useMemo(() => {
    let result = [...jobs];

    // Filter by source
    if (filterSource !== "all") {
      result = result.filter((job) => job.source === filterSource);
    }

    // Filter by minimum reward
    if (minRewardFilter !== null) {
      result = result.filter((job) => job.reward >= minRewardFilter);
    }

    // Sort
    switch (sortBy) {
      case "reward-high":
        result.sort((a, b) => b.reward - a.reward);
        break;
      case "reward-low":
        result.sort((a, b) => a.reward - b.reward);
        break;
      case "newest":
      default:
        // Jobs are already in newest-first order from the store
        break;
    }

    return result;
  }, [jobs, filterSource, sortBy, minRewardFilter]);

  const stats = useMemo(() => {
    const totalReward = jobs.reduce((sum, job) => sum + job.reward, 0);
    const rssCount = jobs.filter((j) => j.source === "rss").length;
    const wsCount = jobs.filter((j) => j.source === "websocket").length;
    return { totalReward, rssCount, wsCount };
  }, [jobs]);

  return (
    <div className="bento-card p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-blue-600 font-mono text-xs uppercase tracking-widest">
          Detected Jobs
        </h3>
        <button
          onClick={clearJobs}
          aria-label="Clear all jobs"
          className="font-mono text-xs text-neutral-400 hover:text-red-600 transition-colors"
        >
          Clear
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-4 pb-4 border-b border-neutral-100">
        {/* Source Filter */}
        <select
          value={filterSource}
          onChange={(e) => setFilterSource(e.target.value as FilterSource)}
          className="px-3 py-1 text-xs font-mono border border-neutral-200 focus:border-blue-600 focus:outline-none"
        >
          <option value="all">All Sources</option>
          <option value="rss">RSS Only</option>
          <option value="websocket">WebSocket Only</option>
        </select>

        {/* Sort By */}
        <select
          value={sortBy}
          onChange={(e) => setSortBy(e.target.value as SortBy)}
          className="px-3 py-1 text-xs font-mono border border-neutral-200 focus:border-blue-600 focus:outline-none"
        >
          <option value="newest">Newest First</option>
          <option value="reward-high">Reward: High to Low</option>
          <option value="reward-low">Reward: Low to High</option>
        </select>

        {/* Min Reward Filter */}
        <div className="flex items-center gap-2">
          <input
            type="number"
            placeholder="Min $"
            min="0"
            step="0.01"
            value={minRewardFilter ?? ""}
            onChange={(e) =>
              setMinRewardFilter(
                e.target.value ? parseFloat(e.target.value) : null
              )
            }
            className="w-20 px-2 py-1 text-xs font-mono border border-neutral-200 focus:border-blue-600 focus:outline-none"
          />
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-3 mb-4 pb-4 border-b border-neutral-100">
        <div className="text-center">
          <p className="text-xs text-neutral-500">Total</p>
          <p className="text-lg font-light">{jobs.length}</p>
        </div>
        <div className="text-center">
          <p className="text-xs text-neutral-500">Value</p>
          <p className="text-lg font-light">${stats.totalReward.toFixed(2)}</p>
        </div>
        <div className="text-center">
          <p className="text-xs text-neutral-500">RSS / WS</p>
          <p className="text-lg font-light">
            {stats.rssCount} / {stats.wsCount}
          </p>
        </div>
      </div>

      {/* Job List */}
      <div className="space-y-2 max-h-[400px] overflow-y-auto">
        {filteredJobs.length === 0 ? (
          <p className="text-center text-neutral-400 py-8 text-sm">
            No jobs detected yet. Start the watcher to begin monitoring.
          </p>
        ) : (
          filteredJobs.map((job) => (
            <JobListItem key={job.id} job={job} />
          ))
        )}
      </div>
    </div>
  );
}

function JobListItem({ job }: { job: Job }) {
  const getRewardColor = (reward: number) => {
    if (reward >= 10) return "text-green-600";
    if (reward >= 5) return "text-yellow-600";
    return "text-neutral-600";
  };

  const getSourceBadge = (source: Job["source"]) => {
    const styles = {
      rss: "bg-orange-100 text-orange-700",
      websocket: "bg-blue-100 text-blue-700",
    };
    return styles[source];
  };

  return (
    <div className="flex items-start gap-3 p-3 border border-neutral-100 hover:border-blue-600 transition-colors duration-150">
      {/* Source Badge */}
      <span
        className={`px-2 py-0.5 text-[10px] font-mono uppercase rounded ${getSourceBadge(
          job.source
        )} flex-shrink-0`}
      >
        {job.source}
      </span>

      {/* Job Details */}
      <div className="flex-1 min-w-0">
        <a
          href={job.url}
          target="_blank"
          rel="noopener noreferrer"
          className="block text-sm font-medium hover:text-blue-600 transition-colors"
        >
          {job.title}
        </a>
      </div>

      {/* Reward */}
      <div className={`text-sm font-mono ${getRewardColor(job.reward)}`}>
        ${job.reward.toFixed(2)}
      </div>
    </div>
  );
}
