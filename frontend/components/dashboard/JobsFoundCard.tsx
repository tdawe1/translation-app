"use client";

import { BentoCard } from "@/components/ui/base/BentoCard";

interface JobsFoundCardProps {
  state: any;
}

export function JobsFoundCard({ state }: JobsFoundCardProps) {
  return (
    <BentoCard
      accentColor="orange"
      staggerIndex={1}
      testId="jobs-found-card"
      className="p-6"
    >
      <h3 className="font-mono text-xs uppercase tracking-widest text-orange-600 mb-2">
        Jobs Found
      </h3>
      <p
        role="status"
        aria-live="polite"
        className="text-3xl font-light"
      >
        {state?.total_jobs_found ?? 0}
      </p>
      {state?.total_jobs_accepted !== undefined && (
        <p className="text-xs text-neutral-500 mt-1">
          {state.total_jobs_accepted} accepted
        </p>
      )}
    </BentoCard>
  );
}
