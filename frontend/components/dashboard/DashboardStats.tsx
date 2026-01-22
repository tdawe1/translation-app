"use client";

import { WatcherStatusCard } from "./WatcherStatusCard";
import { JobsFoundCard } from "./JobsFoundCard";
import { EarningsCard } from "./EarningsCard";
import { WatcherConfigCard } from "./WatcherConfigCard";
import { ActionsCard } from "./ActionsCard";

interface DashboardStatsProps {
  connected: boolean;
  configLoading: boolean;
  configError: string | null;
  config: any;
  state: any;
  stateLoading: boolean;
  statusDisplay: { text: string; color: string };
  isRunning: boolean;
  onStartWatcher: () => void;
  onStopWatcher: () => void;
  onConfigure: () => void;
  startError: string | null;
  onDismissStartError: () => void;
  onLogout: () => void;
}

export function DashboardStats({
  connected,
  configLoading,
  configError,
  config,
  state,
  stateLoading,
  statusDisplay,
  isRunning,
  onStartWatcher,
  onStopWatcher,
  onConfigure,
  startError,
  onDismissStartError,
  onLogout,
}: DashboardStatsProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
      <WatcherStatusCard connected={connected} statusDisplay={statusDisplay} />
      <JobsFoundCard state={state} />
      <EarningsCard state={state} />
      <WatcherConfigCard configLoading={configLoading} configError={configError} config={config} />
      <ActionsCard
        isRunning={isRunning}
        stateLoading={stateLoading}
        onStartWatcher={onStartWatcher}
        onStopWatcher={onStopWatcher}
        onConfigure={onConfigure}
        startError={startError}
        onDismissStartError={onDismissStartError}
        onLogout={onLogout}
      />
    </div>
  );
}
