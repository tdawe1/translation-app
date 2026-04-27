"use client";

import { BentoCard } from "@/components/ui/base/BentoCard";

interface ActionsCardProps {
  isRunning: boolean;
  stateLoading: boolean;
  onStartWatcher: () => void;
  onStopWatcher: () => void;
  onConfigure: () => void;
  startError: string | null;
  onDismissStartError: () => void;
  onLogout: () => void;
}

export function ActionsCard({
  isRunning,
  stateLoading,
  onStartWatcher,
  onStopWatcher,
  onConfigure,
  startError,
  onDismissStartError,
  onLogout,
}: ActionsCardProps) {
  return (
    <BentoCard
      accentColor="cyan"
      staggerIndex={4}
      testId="actions-card"
      className="p-3"
    >
      <h3 className="mb-3 font-mono text-[10px] uppercase tracking-widest text-cyan-600">
        Actions
      </h3>
      <div className="grid gap-2">
        <button
          data-testid="start-watcher-button"
          onClick={onStartWatcher}
          disabled={isRunning || stateLoading}
          className="w-full bg-blue-600 px-3 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {stateLoading ? "Starting..." : "Start Watcher"}
        </button>
        <button
          data-testid="stop-watcher-button"
          onClick={onStopWatcher}
          disabled={!isRunning || stateLoading}
          className="w-full bg-neutral-100 px-3 py-2 text-sm font-medium text-neutral-900 transition-colors duration-150 hover:bg-neutral-200 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {stateLoading ? "Stopping..." : "Stop Watcher"}
        </button>
        <button
          data-testid="configure-button"
          onClick={onConfigure}
          className="flex w-full items-center justify-center gap-2 border border-neutral-300 px-3 py-2 text-sm transition-colors duration-150 hover:border-blue-600 disabled:cursor-not-allowed disabled:opacity-50"
          title="Keyboard shortcut: Ctrl+K"
        >
          Configure
          <kbd className="font-mono text-[10px] px-1.5 py-0.5 bg-neutral-100 rounded text-neutral-500">
            Ctrl+K
          </kbd>
        </button>

        {startError && (
          <div className="mt-3 p-3 bg-red-50 border border-red-200 text-sm text-red-700 flex items-start gap-2">
            <span aria-hidden="true">✕</span>
            <div className="flex-1">
              <p className="font-medium">Error starting watcher</p>
              <p className="text-red-600 mt-1">{startError}</p>
              {startError.includes("Unauthorized") && (
                <p className="text-red-600 text-xs mt-2">
                  Your session may have expired. Try{" "}
                  <button
                    onClick={onLogout}
                    className="underline hover:text-red-800"
                  >
                    signing out and back in
                  </button>
                  .
                </p>
              )}
            </div>
            <button
              onClick={onDismissStartError}
              className="text-red-400 hover:text-red-600"
              aria-label="Dismiss error"
            >
              ×
            </button>
          </div>
        )}
      </div>
    </BentoCard>
  );
}
