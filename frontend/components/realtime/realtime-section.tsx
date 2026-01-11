/**
 * RealtimeSection - Container for realtime dashboard section
 *
 * Displays live summary stats and collapsible event log.
 * Uses the Hybrid approach: summary cards always visible,
 * detailed log available on demand.
 */

import React from "react";
import { StatCard } from "./stat-card";
import { EventLog } from "./event-log";
import { useRealtimeStore } from "@/store/realtime";

interface RealtimeSectionProps {
  connected: boolean;
  uptime: number;
  lastMessageTime: number | null;
}

// Format uptime in human-readable format
const formatUptime = (seconds: number): string => {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}m ${secs}s`;
  }
  const hours = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${mins}m`;
};

export const RealtimeSection = React.memo<RealtimeSectionProps>(
  ({ connected, uptime, lastMessageTime }
) => {
    const { stats } = useRealtimeStore();

    // Derive status display from connection state
    const statusValue = connected ? "Live" : "Disconnected";
    const statusColor = connected ? ("green" as const) : ("red" as const);

    return (
      <div className="space-y-6">
        {/* Section Header */}
        <div>
          <h2 className="text-3xl font-light tracking-tighter mb-1">
            Realtime
          </h2>
          <p className="text-neutral-500 font-mono text-xs uppercase tracking-widest flex items-center gap-3">
            <span>Live activity feed</span>
            {connected && (
              <span className="flex items-center gap-1.5">
                <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" aria-hidden="true" />
                <span>Connected</span>
              </span>
            )}
            {uptime > 0 && (
              <span className="text-neutral-400">
                Uptime: {formatUptime(uptime)}
              </span>
            )}
          </p>
        </div>

        {/* Summary Stat Cards Grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatCard
            label="Status"
            value={statusValue}
            color={statusColor}
            testId="realtime-status"
          />
          <StatCard
            label="Detected"
            value={stats.jobsDetected}
            color="cyan"
            testId="realtime-detected"
          />
          <StatCard
            label="Accepted"
            value={stats.jobsAccepted}
            color="green"
            testId="realtime-accepted"
          />
          <StatCard
            label="Filtered"
            value={stats.jobsFiltered}
            color="yellow"
            testId="realtime-filtered"
          />
        </div>

        {/* Event Log */}
        <EventLog
          defaultCollapsed={true}
          maxVisible={50}
          testId="realtime-event-log"
        />
      </div>
    );
  }
);

RealtimeSection.displayName = "RealtimeSection";
