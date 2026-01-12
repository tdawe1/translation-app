/**
 * JobDetailModal - Modal for displaying full job details
 *
 * Enhanced with Data Factory base components and consistent styling.
 * Shows comprehensive job information with accept action and external link.
 */

"use client";

import React, { useEffect, useCallback } from "react";
import type { ExtendedJob } from "@/store/jobs";
import { BentoCard } from "@/components/ui/base/BentoCard";
import { Button } from "@/components/ui/base/Button";
import { DESIGN } from "@/lib/design/tokens";
import { cn } from "@/lib/utils";

interface JobDetailModalProps {
  /** Job to display details for */
  job: ExtendedJob;
  /** Whether modal is open */
  isOpen: boolean;
  /** Called when modal should close */
  onClose: () => void;
  /** Called when user accepts the job */
  onAccept?: (job: ExtendedJob) => void;
  /** Loading state for accept action */
  isAccepting?: boolean;
}

// ============================================================================
// COMPONENT
// ============================================================================

export function JobDetailModal({
  job,
  isOpen,
  onClose,
  onAccept,
  isAccepting = false,
}: JobDetailModalProps) {
  // Handle escape key to close
  const handleEscape = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        onClose();
      }
    },
    [isOpen, onClose]
  );

  useEffect(() => {
    if (isOpen) {
      document.addEventListener("keydown", handleEscape);
      // Prevent body scroll when modal is open
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }

    return () => {
      document.removeEventListener("keydown", handleEscape);
      document.body.style.overflow = "";
    };
  }, [isOpen, handleEscape]);

  // Handle backdrop click to close
  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  // Format timestamp to relative time
  const formatTimeAgo = (timestamp?: string): string => {
    if (!timestamp) return "Just now";

    const date = new Date(timestamp);
    const now = new Date();
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

    if (seconds < 60) return "Just now";
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
  };

  // Get reward color based on value
  const getRewardColor = (reward: number): string => {
    if (reward >= 10) return "text-green-600";
    if (reward >= 5) return "text-yellow-600";
    return "text-neutral-600";
  };

  // Get source badge styles
  const getSourceBadge = (source: Job["source"]): string => {
    const styles = {
      rss: "bg-orange-50 border-orange-200 text-orange-700",
      websocket: "bg-blue-50 border-blue-200 text-blue-700",
    };
    return styles[source];
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-fade-in"
      onClick={handleBackdropClick}
      aria-labelledby="job-modal-title"
      role="dialog"
      aria-modal="true"
    >
      <BentoCard
        accentColor="blue"
        className="max-w-lg w-full max-h-[90vh] overflow-y-auto relative"
        hoverDisabled
      >
        {/* Close Button */}
        <button
          onClick={onClose}
          aria-label="Close modal"
          className={cn(
            "absolute top-4 right-4 p-2 text-neutral-400 hover:text-neutral-900",
            "transition-colors duration-150"
          )}
        >
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="square"
              strokeLinejoin="miter"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>

        {/* Header Section */}
        <div className="mb-6">
          {/* Title & Source Badge */}
          <div className="flex items-start gap-3 mb-4">
            <span
              className={cn(
                "px-2 py-0.5 text-[10px] font-mono uppercase border flex-shrink-0",
                getSourceBadge(job.source)
              )}
            >
              {job.source}
            </span>
            <h2
              id="job-modal-title"
              className="text-lg font-medium text-neutral-900 leading-snug"
            >
              {job.title}
            </h2>
          </div>

          {/* Reward - Prominent */}
          <div className="flex items-baseline gap-1 mb-2">
            <span className={cn("text-3xl font-light", getRewardColor(job.reward))}>
              ${job.reward.toFixed(2)}
            </span>
            {job.unitCount && (
              <span className="text-sm text-neutral-500">
                ({job.unitCount.toLocaleString()} {job.unitType === "words" ? "words" : "chars"})
              </span>
            )}
          </div>

          {/* Meta Info */}
          <div className="flex items-center gap-4 text-xs text-neutral-500">
            <span className="font-mono">
              {formatTimeAgo(job.timestamp)}
            </span>
            {job.deadline && (
              <>
                <span>•</span>
                <span>Due: {new Date(job.deadline).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
              </>
            )}
          </div>
        </div>

        {/* Divider */}
        <hr className="border-neutral-200 my-6" />

        {/* Language Pair */}
        {job.languagePair && (
          <div className="mb-6">
            <h3 className={cn("mb-2", DESIGN.typography.label)}>
              Language Pair
            </h3>
            <div className="inline-flex items-center gap-2 px-3 py-2 bg-neutral-50 border border-neutral-200">
              <span className="font-mono text-sm text-neutral-700">
                {job.languagePair}
              </span>
            </div>
          </div>
        )}

        {/* Description */}
        {job.description && (
          <div className="mb-6">
            <h3 className={cn("mb-2", DESIGN.typography.label)}>
              Description
            </h3>
            <p className="text-sm text-neutral-700 leading-relaxed whitespace-pre-wrap">
              {job.description}
            </p>
          </div>
        )}

        {/* Filter Reason (if applicable) */}
        {job.filterReason && (
          <div className={cn(
            "mb-6 p-3 border",
            DESIGN.semantic.warning
          )}>
            <h3 className={cn("mb-1", DESIGN.typography.label, "text-yellow-800")}>
              Filtered
            </h3>
            <p className="text-sm text-yellow-900">
              {job.filterReason}
            </p>
          </div>
        )}

        {/* Divider */}
        <hr className="border-neutral-200 my-6" />

        {/* Actions */}
        <div className="flex gap-3">
          {/* External Link */}
          <a
            href={job.url}
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              "inline-flex items-center justify-center gap-2 px-4 py-2",
              "border border-neutral-200 text-sm font-medium text-neutral-700",
              "transition-colors duration-150",
              "hover:border-blue-600 hover:text-blue-600",
              "focus:outline-none focus:border-blue-600"
            )}
          >
            <svg
              className="w-4 h-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
              />
            </svg>
            Open on Gengo
          </a>

          {/* Accept Button (if enabled) */}
          {onAccept && job.canAccept !== false && (
            <Button
              onClick={() => onAccept(job)}
              loading={isAccepting}
              loadingText="Accepting..."
              variant="primary"
              className="flex-1"
            >
              Accept Job
            </Button>
          )}
        </div>

        {/* Footer Note */}
        <p className="mt-4 text-xs text-neutral-400 text-center">
          Opening job in new tab. Close this modal when done.
        </p>
      </BentoCard>
    </div>
  );
}

// ============================================================================
// HELPER COMPONENTS
// ============================================================================

/**
 * Compact trigger button for opening the job detail modal.
 * Use this within list items to trigger the modal.
 */
export interface JobDetailTriggerProps {
  job: ExtendedJob;
  onOpen: (job: ExtendedJob) => void;
  className?: string;
}

export function JobDetailTrigger({
  job,
  onOpen,
  className,
}: JobDetailTriggerProps) {
  return (
    <button
      onClick={() => onOpen(job)}
      className={cn(
        "p-1.5 text-neutral-400 hover:text-blue-600",
        "transition-colors duration-150",
        "focus:outline-none focus:text-blue-600",
        className
      )}
      aria-label={`View details for "${job.title}"`}
    >
      <svg
        className="w-4 h-4"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
        />
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
        />
      </svg>
    </button>
  );
}
