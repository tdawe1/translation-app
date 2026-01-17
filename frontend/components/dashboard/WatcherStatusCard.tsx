"use client";

import { BentoCard } from "@/components/ui/base/BentoCard";

interface WatcherStatusCardProps {
  connected: boolean;
  statusDisplay: { text: string; color: string };
}

export function WatcherStatusCard({ connected, statusDisplay }: WatcherStatusCardProps) {
  return (
    <BentoCard
      accentColor="red"
      staggerIndex={0}
      testId="status-card"
      className="p-6"
    >
      <div className="flex items-center justify-between mb-2">
        <h3 className="font-mono text-xs uppercase tracking-widest text-red-600">
          Watcher Status
        </h3>
        {connected && (
          <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
        )}
      </div>
      <p
        data-testid="watcher-status"
        role="status"
        aria-live="polite"
        className={`text-3xl font-light ${statusDisplay.color}`}
      >
        {statusDisplay.text}
      </p>
    </BentoCard>
  );
}
