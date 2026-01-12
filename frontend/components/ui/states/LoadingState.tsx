/**
 * LoadingState - Skeleton loading states for async content
 *
 * Variants: card, list, table, stats
 */

import { cn } from "@/lib/utils";

export type LoadingVariant = "card" | "list" | "table" | "stats";

export interface LoadingStateProps {
  /** Visual variant matching the content being loaded */
  variant?: LoadingVariant;
  /** Number of skeleton items to show (for list/table variants) */
  count?: number;
  /** Custom className */
  className?: string;
}

// Base shimmer animation style
const shimmerClass =
  "animate-pulse bg-neutral-200 rounded-sm";

// Card skeleton - matches BentoCard structure
const CardSkeleton = ({ className }: { className?: string }) => (
  <div className={cn("bento-card p-6", className)}>
    <div className="h-4 w-24 bg-neutral-200 rounded-sm mb-4" />
    <div className="space-y-2">
      <div className="h-3 w-full bg-neutral-200 rounded-sm" />
      <div className="h-3 w-3/4 bg-neutral-200 rounded-sm" />
      <div className="h-3 w-1/2 bg-neutral-200 rounded-sm" />
    </div>
  </div>
);

// List skeleton - repeated items for job lists, etc.
const ListSkeleton = ({ count = 3 }: { count?: number }) => (
  <div className="bento-card p-6">
    <div className="space-y-3">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 py-2 border-b border-neutral-100 last:border-0">
          <div className="h-8 w-8 bg-neutral-200 rounded-sm flex-shrink-0" />
          <div className="flex-1 space-y-2">
            <div className="h-3 w-full bg-neutral-200 rounded-sm" />
            <div className="h-3 w-1/3 bg-neutral-200 rounded-sm" />
          </div>
          <div className="h-6 w-12 bg-neutral-200 rounded-sm" />
        </div>
      ))}
    </div>
  </div>
);

// Table skeleton - for data tables
const TableSkeleton = ({ count = 5 }: { count?: number }) => (
  <div className="bento-card p-6">
    <div className="w-full h-64">
      {/* Header row */}
      <div className="flex gap-4 mb-4 pb-2 border-b border-neutral-200">
        <div className="h-4 flex-1 bg-neutral-200 rounded-sm" />
        <div className="h-4 flex-1 bg-neutral-200 rounded-sm" />
        <div className="h-4 flex-1 bg-neutral-200 rounded-sm" />
        <div className="h-4 w-20 bg-neutral-200 rounded-sm" />
      </div>
      {/* Data rows */}
      <div className="space-y-3">
        {Array.from({ length: count }).map((_, i) => (
          <div key={i} className="flex gap-4">
            <div className="h-8 flex-1 bg-neutral-200 rounded-sm" />
            <div className="h-8 flex-1 bg-neutral-200 rounded-sm" />
            <div className="h-8 flex-1 bg-neutral-200 rounded-sm" />
            <div className="h-8 w-20 bg-neutral-200 rounded-sm" />
          </div>
        ))}
      </div>
    </div>
  </div>
);

// Stats skeleton - for stat cards with numbers
const StatsSkeleton = () => (
  <div className="grid grid-cols-3 gap-6">
    {Array.from({ length: 3 }).map((_, i) => (
      <div key={i} className="bento-card p-6">
        <div className="h-3 w-20 bg-neutral-200 rounded-sm mb-4" />
        <div className="h-8 w-16 bg-neutral-200 rounded-sm" />
      </div>
    ))}
  </div>
);

export const LoadingState = ({
  variant = "card",
  count = 3,
  className,
}: LoadingStateProps) => {
  const content = (() => {
    switch (variant) {
      case "card":
        return <CardSkeleton className={className} />;
      case "list":
        return <ListSkeleton count={count} />;
      case "table":
        return <TableSkeleton count={count} />;
      case "stats":
        return <StatsSkeleton />;
      default:
        return <CardSkeleton className={className} />;
    }
  })();

  return <div className="animate-fade-in" aria-busy="true" aria-label="Loading">
    {content}
  </div>
};
