"use client";

import { BentoCard } from "@/components/ui/base/BentoCard";
import { ConfigRow } from "./ConfigRow";

interface WatcherConfigCardProps {
  configLoading: boolean;
  configError: string | null;
  config: any;
}

export function WatcherConfigCard({
  configLoading,
  configError,
  config,
}: WatcherConfigCardProps) {
  return (
    <BentoCard
      accentColor="green"
      staggerIndex={3}
      testId="config-card"
      className="p-6 md:col-span-2"
    >
      <h3 className="font-mono text-xs uppercase tracking-widest text-green-600 mb-4">
        Watcher Configuration
      </h3>
      {configLoading ? (
        <p className="text-sm text-neutral-400">Loading configuration...</p>
      ) : configError ? (
        <p className="text-sm text-red-500">{configError}</p>
      ) : config ? (
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <ConfigRow label="Min Reward" value={`$${config.min_reward.toFixed(2)}`} />
            <ConfigRow label="Max Reward" value={`$${config.max_reward.toFixed(2)}`} />
          </div>
          <div className="space-y-2">
            <ConfigRow
              label="WebSocket"
              value={config.websocket_enabled ? "Enabled" : "Disabled"}
            />
            <ConfigRow
              label="Auto Accept"
              value={config.auto_accept_enabled ? "Enabled" : "Disabled"}
            />
          </div>
          <div className="col-span-2">
            <ConfigRow
              label="RSS Feed"
              value={config.rss_feed_url}
              truncate
            />
          </div>
        </div>
      ) : (
        <p className="text-sm text-neutral-400">No configuration found</p>
      )}
    </BentoCard>
  );
}
