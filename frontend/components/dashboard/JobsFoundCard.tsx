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
      className="p-3"
    >
      <h3 className="mb-1 font-mono text-[10px] uppercase tracking-widest text-orange-600">
        Jobs Found
      </h3>
      <p
        role="status"
        aria-live="polite"
        className="text-xl font-light"
      >
        {state?.total_jobs_found ?? 0}
      </p>
      {state?.total_jobs_accepted !== undefined && (
        <p className="mt-0.5 text-xs text-neutral-500">
          {state.total_jobs_accepted} accepted
        </p>
      )}
    </BentoCard>
  );
}
