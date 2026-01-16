/**
 * Translations Page - List of translation jobs
 *
 * Displays a table of translation jobs with filtering and pagination.
 */

"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTranslationStore } from "@/store/translationStore";
import type { JobSummary, TranslationJobStatus } from "@/lib/api/types";
import { ProtectedRoute } from "@/components/auth/protected-route";
import { SectionHeader } from "@/components/ui/base/SectionHeader";
import { BentoCard } from "@/components/ui/base/BentoCard";
import { Button } from "@/components/ui/base/Button";
import { cn } from "@/lib/utils";

// ============================================================================
// TYPES
// ============================================================================

type SortBy = "newest" | "oldest";

// ============================================================================
// COMPONENT
// ============================================================================

export default function TranslationsPage() {
  const router = useRouter();
  const {
    jobs,
    jobsLoading,
    jobsPageSize,
    jobsTotalCount,
    fetchJobs,
  } = useTranslationStore();

  const [statusFilter, setStatusFilter] = useState<TranslationJobStatus | "all">("all");
  const [sortBy, setSortBy] = useState<SortBy>("newest");
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = jobsPageSize || 10;
  const totalPages = Math.max(1, Math.ceil(jobsTotalCount / pageSize));

  useEffect(() => {
    fetchJobs({
      page: currentPage,
      pageSize,
      status: statusFilter,
      sort: sortBy,
    });
  }, [fetchJobs, currentPage, pageSize, statusFilter, sortBy]);

  useEffect(() => {
    setCurrentPage(1);
  }, [statusFilter, sortBy]);

  const handleRowClick = (jobId: string) => {
    router.push(`/translations/${jobId}`);
  };

  return (
    <ProtectedRoute>
      <main id="main-content" className="min-h-screen bg-neutral-50">
        <header className="bg-white border-b border-neutral-200">
          <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
            <Link href="/" className="text-xl font-light tracking-tighter text-neutral-900 hover:text-blue-600 transition-colors duration-150">
              GengoWatcher
            </Link>
            <div className="flex items-center gap-6">
              <Link
                href="/dashboard"
                className="font-mono text-xs text-neutral-500 uppercase tracking-widest hover:text-blue-600 transition-colors duration-150"
              >
                Dashboard
              </Link>
            </div>
          </div>
        </header>

        <div className="max-w-6xl mx-auto px-6 py-12">
          <SectionHeader
            title="Translations"
            meta="JOB MANAGEMENT"
            accentColor="blue"
          />

          <BentoCard accentColor="blue" staggerIndex={0} className="p-6">
            <div className="flex flex-wrap items-center justify-between gap-4 mb-6 pb-6 border-b border-neutral-200">
              <div className="flex items-center gap-4">
                <select
                  value={statusFilter}
                  onChange={(e) =>
                    setStatusFilter(e.target.value as TranslationJobStatus | "all")
                  }
                  className={cn(
                    "px-3 py-2 text-sm font-mono border border-neutral-200",
                    "focus:border-blue-600 focus:outline-none",
                    "transition-colors duration-150"
                  )}
                >
                  <option value="all">All Statuses</option>
                  <option value="pending">Pending</option>
                  <option value="processing">Processing</option>
                  <option value="translating">Translating</option>
                  <option value="pending_approval">Pending Approval</option>
                  <option value="approved">Approved</option>
                  <option value="rejected">Rejected</option>
                  <option value="completed">Completed</option>
                  <option value="failed">Failed</option>
                  <option value="cancelled">Cancelled</option>
                </select>

                <select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value as SortBy)}
                  className={cn(
                    "px-3 py-2 text-sm font-mono border border-neutral-200",
                    "focus:border-blue-600 focus:outline-none",
                    "transition-colors duration-150"
                  )}
                >
                  <option value="newest">Newest First</option>
                  <option value="oldest">Oldest First</option>
                </select>
              </div>

              <div className="text-xs font-mono text-neutral-500">
                {jobsTotalCount} jobs found
              </div>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-neutral-200">
                    <th className="py-3 px-4 font-mono text-xs uppercase tracking-widest text-neutral-500 font-normal">Source Text</th>
                    <th className="py-3 px-4 font-mono text-xs uppercase tracking-widest text-neutral-500 font-normal">Status</th>
                    <th className="py-3 px-4 font-mono text-xs uppercase tracking-widest text-neutral-500 font-normal">Progress</th>
                    <th className="py-3 px-4 font-mono text-xs uppercase tracking-widest text-neutral-500 font-normal">Score</th>
                    <th className="py-3 px-4 font-mono text-xs uppercase tracking-widest text-neutral-500 font-normal">Flagged</th>
                    <th className="py-3 px-4 font-mono text-xs uppercase tracking-widest text-neutral-500 font-normal">Date</th>
                  </tr>
                </thead>
                <tbody className="text-sm">
                  {jobsLoading ? (
                    <tr>
                      <td colSpan={6} className="py-8 text-center text-neutral-400">
                        Loading translations...
                      </td>
                    </tr>
                  ) : jobs.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="py-8 text-center text-neutral-400">
                        No translation jobs found.
                      </td>
                    </tr>
                  ) : (
                    jobs.map((job: JobSummary) => (
                      <tr
                        key={job.id}
                        onClick={() => handleRowClick(job.id)}
                        className="border-b border-neutral-100 last:border-0 hover:bg-blue-50/50 cursor-pointer transition-colors duration-150 group"
                      >
                        <td className="py-3 px-4 max-w-[200px] truncate text-neutral-900 font-medium">
                          {job.source_file}
                        </td>
                        <td className="py-3 px-4">
                          <StatusBadge status={job.status} />
                        </td>
                        <td className="py-3 px-4 font-mono text-xs text-neutral-600">
                          {Math.round(job.progress * 100)}%
                        </td>
                        <td className="py-3 px-4 font-mono text-xs text-neutral-600">
                          {job.overall_score > 0 ? job.overall_score.toFixed(2) : "-"}
                        </td>
                        <td className="py-3 px-4">
                          {job.flagged_count > 0 ? (
                            <span className="inline-flex items-center justify-center px-2 py-0.5 rounded text-xs font-mono bg-red-100 text-red-700">
                              {job.flagged_count}
                            </span>
                          ) : (
                            <span className="text-neutral-400">-</span>
                          )}
                        </td>
                        <td className="py-3 px-4 font-mono text-xs text-neutral-500">
                          {new Date(job.created_at).toLocaleDateString()}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>

            {totalPages > 1 && (
              <div className="flex items-center justify-between mt-6 pt-6 border-t border-neutral-200">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                >
                  Previous
                </Button>
                <span className="font-mono text-xs text-neutral-500">
                  Page {currentPage} of {totalPages}
                </span>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                  disabled={currentPage === totalPages}
                >
                  Next
                </Button>
              </div>
            )}
          </BentoCard>
        </div>
      </main>
    </ProtectedRoute>
  );
}

function StatusBadge({ status }: { status: TranslationJobStatus }) {
  const styles: Record<TranslationJobStatus, string> = {
    pending: "bg-neutral-100 text-neutral-600 border-neutral-200",
    processing: "bg-blue-50 text-blue-600 border-blue-200",
    translating: "bg-blue-100 text-blue-700 border-blue-300",
    pending_approval: "bg-orange-50 text-orange-600 border-orange-200",
    approved: "bg-green-100 text-green-700 border-green-300",
    rejected: "bg-red-50 text-red-600 border-red-200",
    completed: "bg-green-50 text-green-600 border-green-200",
    failed: "bg-red-100 text-red-700 border-red-300",
    cancelled: "bg-neutral-200 text-neutral-500 border-neutral-300",
  };

  return (
    <span
      className={cn(
        "px-2 py-0.5 text-[10px] font-mono uppercase border rounded-sm",
        styles[status]
      )}
    >
      {status.replace("_", " ")}
    </span>
  );
}
