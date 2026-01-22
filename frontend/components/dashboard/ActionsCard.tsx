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
      className="p-6"
    >
      <h3 className="font-mono text-xs uppercase tracking-widest text-cyan-600 mb-4">
        Actions
      </h3>
      <div className="space-y-3">
        <button
          data-testid="start-watcher-button"
          onClick={onStartWatcher}
          disabled={isRunning || stateLoading}
          className="w-full py-3 bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors duration-150"
        >
          {stateLoading ? "Starting..." : "Start Watcher"}
        </button>
        <button
          data-testid="stop-watcher-button"
          onClick={onStopWatcher}
          disabled={!isRunning || stateLoading}
          className="w-full py-3 bg-neutral-100 text-neutral-900 text-sm font-medium hover:bg-neutral-200 disabled:opacity-50 disabled:cursor-not-allowed transition-colors duration-150"
        >
          {stateLoading ? "Stopping..." : "Stop Watcher"}
        </button>
        <button
          data-testid="configure-button"
          onClick={onConfigure}
          className="w-full py-3 border border-neutral-300 text-sm transition-colors duration-150 hover:border-blue-600 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
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
