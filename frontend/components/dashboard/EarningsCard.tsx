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
      className="p-6"
    >
      <h3 className="font-mono text-xs uppercase tracking-widest text-yellow-600 mb-2">
        Earnings
      </h3>
      <p
        role="status"
        aria-live="polite"
        className="text-3xl font-light"
      >
        ${state?.total_earnings?.toFixed(2) ?? "0.00"}
      </p>
      <p className="text-xs text-neutral-500 mt-1">
        Lifetime total
      </p>
    </BentoCard>
  );
}
