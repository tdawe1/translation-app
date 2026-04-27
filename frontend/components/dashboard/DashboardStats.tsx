"use client";

interface DashboardStatsProps {
  connected: boolean;
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
    <div className="border border-neutral-200 bg-white">
      <div className="grid grid-cols-2 divide-y divide-neutral-200 md:grid-cols-[1fr_0.8fr_0.8fr_1.4fr] md:divide-x md:divide-y-0">
        <div className="p-3">
          <div className="mb-1 flex items-center gap-2">
            <p className="font-mono text-[10px] uppercase tracking-widest text-neutral-500">
              Watcher
            </p>
            {connected ? (
              <span className="h-2 w-2 rounded-full bg-green-500 animate-pulse" />
            ) : null}
          </div>
          <p
            data-testid="watcher-status"
            role="status"
            aria-live="polite"
            className={`text-xl font-light ${statusDisplay.color}`}
          >
            {statusDisplay.text}
          </p>
        </div>

        <div className="p-3">
          <p className="font-mono text-[10px] uppercase tracking-widest text-neutral-500">
            Jobs Found
          </p>
          <p role="status" aria-live="polite" className="mt-1 text-xl font-light">
            {state?.total_jobs_found ?? 0}
          </p>
        </div>

        <div className="p-3">
          <p className="font-mono text-[10px] uppercase tracking-widest text-neutral-500">
            Earnings
          </p>
          <p role="status" aria-live="polite" className="mt-1 text-xl font-light">
            ${state?.total_earnings?.toFixed(2) ?? "0.00"}
          </p>
        </div>

        <div className="col-span-2 p-3 md:col-span-1">
          <p className="mb-2 font-mono text-[10px] uppercase tracking-widest text-neutral-500">
            Controls
          </p>
          <div className="grid gap-2 sm:grid-cols-3 md:grid-cols-1 xl:grid-cols-3">
            <button
              data-testid="start-watcher-button"
              aria-label="Start Watcher"
              onClick={onStartWatcher}
              disabled={isRunning || stateLoading}
              className="bg-blue-600 px-3 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {stateLoading ? "Starting..." : "Start"}
            </button>
            <button
              data-testid="stop-watcher-button"
              aria-label="Stop Watcher"
              onClick={onStopWatcher}
              disabled={!isRunning || stateLoading}
              className="bg-neutral-100 px-3 py-2 text-sm font-medium text-neutral-900 transition-colors duration-150 hover:bg-neutral-200 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {stateLoading ? "Stopping..." : "Stop"}
            </button>
            <button
              data-testid="configure-button"
              onClick={onConfigure}
              className="border border-neutral-300 px-3 py-2 text-sm transition-colors duration-150 hover:border-blue-600 disabled:cursor-not-allowed disabled:opacity-50"
              title="Keyboard shortcut: Ctrl+K"
            >
              Configure
            </button>
          </div>
        </div>
      </div>

      {startError ? (
        <div className="border-t border-red-200 bg-red-50 p-3 text-sm text-red-700">
          <div className="flex items-start gap-2">
            <span aria-hidden="true">x</span>
            <div className="flex-1">
              <p className="font-medium">Error starting watcher</p>
              <p className="mt-1 text-red-600">{startError}</p>
              {startError.includes("Unauthorized") ? (
                <p className="mt-2 text-xs text-red-600">
                  Your session may have expired. Try{" "}
                  <button
                    onClick={onLogout}
                    className="underline hover:text-red-800"
                  >
                    signing out and back in
                  </button>
                  .
                </p>
              ) : null}
            </div>
            <button
              onClick={onDismissStartError}
              className="text-red-400 hover:text-red-600"
              aria-label="Dismiss error"
            >
              x
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
