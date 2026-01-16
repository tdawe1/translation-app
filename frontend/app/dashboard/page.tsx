/**
 * Dashboard Page - Protected route with watcher controls
 *
 * Enhanced with Data Factory base components and improved stats cards.
 */

"use client";

import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import { useWatcherStore } from "@/store/watcher";
import { useWatcherWebSocket } from "@/hooks/use-watcher-websocket";
import { useRealtimeStore } from "@/store/realtime";
import { ProtectedRoute } from "@/components/auth/protected-route";
import { Modal } from "@/components/ui/modal";
import { WatcherConfigForm } from "@/components/watcher/config-form";
import { JobList } from "@/components/watcher/job-list";
import { RealtimeSection } from "@/components/realtime";
import { ErrorBoundary } from "@/components/error-boundary";
import { BentoCard } from "@/components/ui/base/BentoCard";
import { SectionHeader } from "@/components/ui/base/SectionHeader";
import { Button } from "@/components/ui/base/Button";
import { authApi } from "@/lib/api";
import { toast } from "@/store/toast";
import { useEffect, useState } from "react";
import { navigateToHome } from "@/lib/navigation";

export default function DashboardPage() {
  const user = useAuthStore((state) => state.user);
  const [configModalOpen, setConfigModalOpen] = useState(false);
  const [startError, setStartError] = useState<string | null>(null);
  const logout = async () => {
    try {
      await authApi.logout();
    } catch (err) {
      // Continue with logout even if API call fails
    } finally {
      sessionStorage.removeItem("access_token");
      navigateToHome(); // Full page redirect to clear state after logout
    }
  };

  // Watcher state and actions
  const {
    config,
    state,
    configLoading,
    stateLoading,
    configError,
    fetchConfig,
    fetchState,
    startWatcher,
    stopWatcher,
  } = useWatcherStore();

  // Load initial data
  useEffect(() => {
    fetchConfig();
    fetchState();
  }, [fetchConfig, fetchState]);

  // Keyboard shortcuts: Ctrl+K to open config, ESC to close modal
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Ctrl+K or Cmd+K to open config modal
      if ((e.ctrlKey || e.metaKey) && e.key === "k") {
        e.preventDefault();
        setConfigModalOpen(true);
        return;
      }
      // ESC to close modal (already handled by Modal component, but here for global context)
      if (e.key === "Escape" && configModalOpen) {
        setConfigModalOpen(false);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [configModalOpen]);

  // Set up WebSocket for real-time updates
  const { connected, uptime, lastMessageTime } = useWatcherWebSocket({
    enabled: !!user,
    onEvent: (event, data) => {
      // Refresh state when watcher starts/stops
      if (event === "watcher.started" || event === "watcher.stopped") {
        fetchState();
      }

      // Add events to realtime store for the feed
      const { addEvent } = useRealtimeStore.getState();

      switch (event) {
        case "watcher.started":
          addEvent("watcher.started", "Watcher monitoring started");
          break;
        case "watcher.stopped":
          addEvent("watcher.stopped", "Watcher monitoring stopped");
          break;
        case "job.detected":
          if (data && typeof data === "object" && "title" in data) {
            addEvent("job.detected", `Job detected: ${data.title}`, data);
          }
          break;
        case "job.accepted":
          if (data && typeof data === "object" && "title" in data) {
            addEvent("job.accepted", `Auto-accepted: ${data.title}`, data);
          }
          break;
        case "job.filtered":
          if (data && typeof data === "object" && "title" in data) {
            addEvent("job.filtered", `Filtered out: ${data.title}`, data);
          }
          break;
      }
    },
  });

  // Watcher control handlers
  const handleStart = async () => {
    setStartError(null);
    try {
      await startWatcher();
      toast.success("Watcher started successfully");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to start watcher";
      setStartError(message);
      toast.error(message);
    }
  };

  const handleStop = async () => {
    try {
      await stopWatcher();
      toast.success("Watcher stopped successfully");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to stop watcher";
      toast.error(message);
    }
  };

  // Helper to format status
  const getStatusDisplay = () => {
    if (stateLoading) return { text: "Loading...", color: "text-neutral-400" };
    if (!state) return { text: "Unknown", color: "text-neutral-400" };

    switch (state.watcher_status) {
      case "running":
        return { text: "Running", color: "text-green-600" };
      case "stopped":
        return { text: "Stopped", color: "text-red-600" };
      case "error":
        return { text: "Error", color: "text-red-600" };
      default:
        return { text: "Unknown", color: "text-neutral-400" };
    }
  };

  const statusDisplay = getStatusDisplay();
  const isRunning = state?.watcher_status === "running";

  return (
    <ProtectedRoute>
      <ErrorBoundary>
        <main id="main-content" className="min-h-screen bg-neutral-50">
          {/* Header - Enhanced with proper spacing */}
          <header className="bg-white border-b border-neutral-200">
            <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
              <Link href="/" className="text-xl font-light tracking-tighter text-neutral-900 hover:text-blue-600 transition-colors duration-150">
                GengoWatcher
              </Link>
              <div className="flex items-center gap-6">
                <span
                  data-testid="user-email"
                  className="hidden sm:block font-mono text-xs text-neutral-500 uppercase tracking-widest"
                >
                  {user?.email}
                </span>
                <Link
                  data-testid="settings-link"
                  href="/settings"
                  className="font-mono text-xs text-neutral-500 uppercase tracking-widest hover:text-blue-600 transition-colors duration-150"
                >
                  Settings
                </Link>
                <button
                  data-testid="sign-out-button"
                  onClick={logout}
                  className="font-mono text-xs text-neutral-900 uppercase tracking-widest hover:text-blue-600 transition-colors duration-150"
                >
                  Sign Out
                </button>
              </div>
            </div>
          </header>

          {/* Dashboard Content */}
          <div className="max-w-6xl mx-auto px-6 py-12">
            <SectionHeader
              title="Dashboard"
              meta="WELCOME BACK"
              accentColor="blue"
            />

            {/* Bento Grid - Enhanced with base components */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              {/* Watcher Status Card */}
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

              {/* Jobs Found Card */}
              <BentoCard
                accentColor="orange"
                staggerIndex={1}
                testId="jobs-found-card"
                className="p-6"
              >
                <h3 className="font-mono text-xs uppercase tracking-widest text-orange-600 mb-2">
                  Jobs Found
                </h3>
                <p
                  role="status"
                  aria-live="polite"
                  className="text-3xl font-light"
                >
                  {state?.total_jobs_found ?? 0}
                </p>
                {state?.total_jobs_accepted !== undefined && (
                  <p className="text-xs text-neutral-500 mt-1">
                    {state.total_jobs_accepted} accepted
                  </p>
                )}
              </BentoCard>

              {/* Earnings Card */}
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

              {/* Watcher Configuration Card - Spans 2 columns */}
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

              {/* Actions Card */}
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
                  <Button
                    data-testid="start-watcher-button"
                    onClick={handleStart}
                    disabled={isRunning || stateLoading}
                    loading={stateLoading}
                    loadingText="Starting..."
                    fullWidth
                    variant="primary"
                  >
                    Start Watcher
                  </Button>
                  <Button
                    data-testid="stop-watcher-button"
                    onClick={handleStop}
                    disabled={!isRunning || stateLoading}
                    loading={stateLoading}
                    loadingText="Stopping..."
                    fullWidth
                    variant="secondary"
                  >
                    Stop Watcher
                  </Button>
                  <button
                    data-testid="configure-button"
                    onClick={() => setConfigModalOpen(true)}
                    className="w-full py-3 border border-neutral-300 text-sm transition-colors duration-150 hover:border-blue-600 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                    title="Keyboard shortcut: Ctrl+K"
                  >
                    Configure
                    <kbd className="font-mono text-[10px] px-1.5 py-0.5 bg-neutral-100 rounded text-neutral-500">
                      Ctrl+K
                    </kbd>
                  </button>

                  {/* Inline error message */}
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
                              onClick={logout}
                              className="underline hover:text-red-800"
                            >
                              signing out and back in
                            </button>
                            .
                          </p>
                        )}
                      </div>
                      <button
                        onClick={() => setStartError(null)}
                        className="text-red-400 hover:text-red-600"
                        aria-label="Dismiss error"
                      >
                        ×
                      </button>
                    </div>
                  )}
                </div>
              </BentoCard>
            </div>

            {/* Last Activity */}
            {state?.last_activity && (
              <div className="mt-6 text-center">
                <p className="font-mono text-xs text-neutral-400">
                  Last activity: {new Date(state.last_activity).toLocaleString()}
                </p>
              </div>
            )}

            {/* Realtime Section */}
            <div className="mt-12">
              <RealtimeSection
                connected={connected}
                uptime={uptime}
                lastMessageTime={lastMessageTime}
              />
            </div>

            {/* Job List */}
            <div className="mt-8">
              <JobList />
            </div>
          </div>

          {/* Configuration Modal */}
          <Modal
            isOpen={configModalOpen}
            onClose={() => setConfigModalOpen(false)}
            title="Watcher Configuration"
          >
            <WatcherConfigForm onClose={() => setConfigModalOpen(false)} />
          </Modal>
        </main>
      </ErrorBoundary>
    </ProtectedRoute>
  );
}

// Helper component for configuration rows
function ConfigRow({
  label,
  value,
  truncate = false,
}: {
  label: string;
  value: string | number | boolean;
  truncate?: boolean;
}) {
  return (
    <div className="flex justify-between py-2 border-b border-neutral-100 last:border-0">
      <span className="text-sm text-neutral-600">{label}</span>
      <span
        className={`font-mono text-xs text-neutral-900 ${truncate ? "truncate max-w-[200px]" : ""}`}
      >
        {typeof value === "boolean" ? (value ? "Enabled" : "Disabled") : value}
      </span>
    </div>
  );
}
