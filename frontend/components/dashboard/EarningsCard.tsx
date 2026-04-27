"use client";

import { BentoCard } from "@/components/ui/base/BentoCard";

interface EarningsCardProps {
  state: any;
}

export function EarningsCard({ state }: EarningsCardProps) {
  return (
    <BentoCard
      accentColor="yellow"
      staggerIndex={2}
      testId="earnings-card"
      className="p-3"
    >
      <h3 className="mb-1 font-mono text-[10px] uppercase tracking-widest text-yellow-600">
        Earnings
      </h3>
      <p
        role="status"
        aria-live="polite"
        className="text-xl font-light"
      >
        ${state?.total_earnings?.toFixed(2) ?? "0.00"}
      </p>
      <p className="mt-0.5 text-xs text-neutral-500">
        Lifetime total
      </p>
    </BentoCard>
  );
}
