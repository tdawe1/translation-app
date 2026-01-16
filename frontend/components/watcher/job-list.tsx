/**
 * JobList - Displays detected jobs with filtering and detail modal
 *
 * Enhanced with Data Factory base components and consistent styling.
 * Includes integrated modal for job details and improved filtering UI.
 */

"use client";

import { useState, useMemo, useCallback } from "react";
import React from "react";
import { useJobsStore, type Job, type ExtendedJob } from "@/store/jobs";
import { BentoCard } from "@/components/ui/base/BentoCard";
import { Button } from "@/components/ui/base/Button";
import { JobDetailModal, JobDetailTrigger } from "./job-detail-modal";
import { DESIGN } from "@/lib/design/tokens";
import { cn } from "@/lib/utils";
import { filterJobs } from "./utils/filters";
import type { FilterSource, JobFilters, SortBy } from "./utils/types";
import { formatTimeAgo, getRewardColor, getSourceBadge } from "./utils/formatters";

// ============================================================================
// TYPES
// ============================================================================

interface JobListProps {
  /** Optional callback when job is accepted */
  onAcceptJob?: (job: ExtendedJob) => void;
  /** Loading state for accept action */
  isAccepting?: boolean;
}

// Re-export ExtendedJob for convenience
export type { ExtendedJob } from "@/store/jobs";

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export function JobList({ onAcceptJob, isAccepting = false }: JobListProps) {
  const jobs = useJobsStore((state) => state.jobs);
  const clearJobs = useJobsStore((state) => state.clearJobs);

  const [filters, setFilters] = useState<JobFilters>({
    source: "all",
    sortBy: "newest",
    minReward: null,
    maxReward: null,
    timeFilter: "all",
    languagePairs: [],
  });
  const [selectedJob, setSelectedJob] = useState<ExtendedJob | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  // Filter and sort jobs
  const filteredJobs = useMemo(() => filterJobs(jobs, filters), [jobs, filters]);

  // Calculate stats
  const stats = useMemo(() => {
    const totalReward = jobs.reduce((sum, job) => sum + job.reward, 0);
    const rssCount = jobs.filter((j) => j.source === "rss").length;
    const wsCount = jobs.filter((j) => j.source === "websocket").length;
    const avgReward = jobs.length > 0 ? totalReward / jobs.length : 0;
    return {
      totalReward,
      rssCount,
      wsCount,
      avgReward,
      count: jobs.length,
    };
  }, [jobs]);

  // Handle opening job detail
  const handleOpenJob = useCallback((job: ExtendedJob) => {
    setSelectedJob(job);
    setIsModalOpen(true);
  }, []);

  // Handle closing modal
  const handleCloseModal = useCallback(() => {
    setIsModalOpen(false);
    // Delay clearing selected job for animation
    setTimeout(() => setSelectedJob(null), 150);
  }, []);

  // Handle accepting job from modal
  const handleAcceptJob = useCallback((job: ExtendedJob) => {
    if (onAcceptJob) {
      onAcceptJob(job as ExtendedJob);
    }
  }, [onAcceptJob]);


  return (
    <>
      <div className="space-y-6">
        {/* Stats Grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <BentoCard
            accentColor="blue"
            staggerIndex={0}
            className="p-4"
            hoverDisabled
          >
            <p className="text-xs text-neutral-500 mb-1">Total Jobs</p>
            <p className="text-2xl font-light text-neutral-900">{stats.count}</p>
          </BentoCard>

          <BentoCard
            accentColor="green"
            staggerIndex={1}
            className="p-4"
            hoverDisabled
          >
            <p className="text-xs text-neutral-500 mb-1">Total Value</p>
            <p className="text-2xl font-light text-green-600">
              ${stats.totalReward.toFixed(2)}
            </p>
          </BentoCard>

          <BentoCard
            accentColor="orange"
            staggerIndex={2}
            className="p-4"
            hoverDisabled
          >
            <p className="text-xs text-neutral-500 mb-1">Avg Reward</p>
            <p className="text-2xl font-light text-orange-600">
              ${stats.avgReward.toFixed(2)}
            </p>
          </BentoCard>

          <BentoCard
            accentColor="cyan"
            staggerIndex={3}
            className="p-4"
            hoverDisabled
          >
            <p className="text-xs text-neutral-500 mb-1">RSS / WS</p>
            <p className="text-2xl font-light text-cyan-600">
              {stats.rssCount} / {stats.wsCount}
            </p>
          </BentoCard>
        </div>

        {/* Main Job List Card */}
        <BentoCard accentColor="blue" staggerIndex={0} className="p-6">
          {/* Header */}
          <div className="flex items-center justify-between mb-6">
            <h3 className={cn(
              "font-mono text-xs uppercase tracking-widest",
              DESIGN.colors.accent.blue
            )}>
              Detected Jobs
            </h3>
            <Button
              onClick={clearJobs}
              variant="secondary"
              size="sm"
            >
              Clear All
            </Button>
          </div>

          {/* Filters */}
          <div className="flex flex-wrap gap-3 mb-6 pb-6 border-b border-neutral-200">
            {/* Source Filter */}
            <select
              value={filters.source}
              onChange={(e) => setFilters({ ...filters, source: e.target.value as FilterSource })}
              className={cn(
                "px-3 py-2 text-sm font-mono border border-neutral-200",
                "focus:border-blue-600 focus:outline-none",
                "transition-colors duration-150"
              )}
            >
              <option value="all">All Sources</option>
              <option value="rss">RSS Only</option>
              <option value="websocket">WebSocket Only</option>
            </select>

            {/* Sort By */}
            <select
              value={filters.sortBy}
              onChange={(e) => setFilters({ ...filters, sortBy: e.target.value as SortBy })}
              className={cn(
                "px-3 py-2 text-sm font-mono border border-neutral-200",
                "focus:border-blue-600 focus:outline-none",
                "transition-colors duration-150"
              )}
            >
              <option value="newest">Newest First</option>
              <option value="reward-high">Reward: High to Low</option>
              <option value="reward-low">Reward: Low to High</option>
            </select>

            {/* Min Reward Filter */}
            <div className="flex items-center">
              <input
                type="number"
                placeholder="Min $"
                min="0"
                step="0.01"
                value={filters.minReward ?? ""}
                onChange={(e) =>
                  setFilters({
                    ...filters,
                    minReward: e.target.value ? parseFloat(e.target.value) : null,
                  })
                }
                className={cn(
                  "w-24 px-3 py-2 text-sm font-mono border border-neutral-200",
                  "focus:border-blue-600 focus:outline-none",
                  "transition-colors duration-150",
                  "placeholder:text-neutral-400"
                )}
              />
            </div>

            {/* Active Filters Display */}
            {(filters.source !== "all" || filters.minReward !== null) && (
              <div className="flex items-center gap-2 ml-auto">
                <span className="text-xs text-neutral-500">Active:</span>
                {filters.source !== "all" && (
                  <span className="px-2 py-1 text-xs font-mono bg-blue-50 text-blue-700 border border-blue-200">
                    {filters.source}
                  </span>
                )}
                {filters.minReward !== null && (
                  <span className="px-2 py-1 text-xs font-mono bg-green-50 text-green-700 border border-green-200">
                    ≥${filters.minReward}
                  </span>
                )}
              </div>
            )}
          </div>

          {/* Job List */}
          <div className="space-y-2">
            {filteredJobs.length === 0 ? (
              <div className="py-12 text-center">
                <p className="text-neutral-400 text-sm mb-2">
                  {jobs.length === 0
                    ? "No jobs detected yet. Start the watcher to begin monitoring."
                    : "No jobs match your current filters."}
                </p>
                {jobs.length > 0 && filteredJobs.length === 0 && (
                  <Button
                    onClick={() => {
                      setFilters({
                        ...filters,
                        source: "all",
                        minReward: null,
                      });
                    }}
                    variant="secondary"
                    size="sm"
                  >
                    Clear Filters
                  </Button>
                )}
              </div>
            ) : (
              filteredJobs.map((job, index) => (
                <JobListItem
                  key={job.id}
                  job={job}
                  staggerIndex={index % 4}
                  onOpen={() => handleOpenJob(job as ExtendedJob)}
                />
              ))
            )}
          </div>
        </BentoCard>
      </div>

      {/* Job Detail Modal */}
      {selectedJob && (
        <JobDetailModal
          job={selectedJob}
          isOpen={isModalOpen}
          onClose={handleCloseModal}
          onAccept={onAcceptJob ? handleAcceptJob : undefined}
          isAccepting={isAccepting}
        />
      )}
    </>
  );
}

// ============================================================================
// JOB LIST ITEM COMPONENT
// ============================================================================

interface JobListItemProps {
  job: Job;
  staggerIndex: number;
  onOpen: () => void;
}

const JobListItem = React.memo(function JobListItem({
  job,
  staggerIndex,
  onOpen,
}: JobListItemProps) {

  return (
    <div
      className={cn(
        "flex items-center gap-3 p-3 border border-neutral-200",
        "hover:border-blue-600",
        "transition-colors duration-150",
        "group"
      )}
      style={{
        animationDelay: DESIGN.getStaggerDelay(staggerIndex),
      }}
    >
      {/* Source Badge */}
      <span
        className={cn(
          "px-2 py-0.5 text-[10px] font-mono uppercase border flex-shrink-0",
          getSourceBadge(job.source)
        )}
      >
        {job.source === "rss" ? "RSS" : "WS"}
      </span>

      {/* Job Title - Truncated */}
      <div className="flex-1 min-w-0">
        <a
          href={job.url}
          target="_blank"
          rel="noopener noreferrer"
          className="block text-sm font-medium text-neutral-900 hover:text-blue-600 transition-colors truncate"
          onClick={(e) => {
            // Allow middle-click / ctrl+click for new tab
            if (!(e.button === 1 || e.ctrlKey || e.metaKey)) {
              e.preventDefault();
              onOpen();
            }
          }}
        >
          {job.title}
        </a>
        <p className="text-xs text-neutral-500 font-mono mt-0.5">
          {formatTimeAgo(job.timestamp)} ago
        </p>
      </div>

      {/* Reward */}
      <div className={cn("text-sm font-mono font-medium", getRewardColor(job.reward))}>
        ${job.reward.toFixed(2)}
      </div>

      {/* View Details Button */}
      <JobDetailTrigger job={job as ExtendedJob} onOpen={onOpen} />
    </div>
  );
});
