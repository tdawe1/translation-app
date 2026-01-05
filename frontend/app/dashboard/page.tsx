/**
 * Dashboard Page - Protected route with watcher controls
 */

"use client";

import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import { useWatcherStore } from "@/store/watcher";
import { useWatcherWebSocket } from "@/hooks/use-watcher-websocket";
import { ProtectedRoute } from "@/components/auth/protected-route";
import { Modal } from "@/components/ui/modal";
import { WatcherConfigForm } from "@/components/watcher/config-form";
import { JobList } from "@/components/watcher/job-list";
import { ErrorBoundary } from "@/components/error-boundary";
import { authApi } from "@/lib/api";
import { toast } from "@/store/toast";
import { useEffect, useState } from "react";

export default function DashboardPage() {
  const user = useAuthStore((state) => state.user);
  const [configModalOpen, setConfigModalOpen] = useState(false);
  const logout = async () => {
    await authApi.logout();
    sessionStorage.removeItem("access_token");
    window.location.href = "/";
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
  const { connected } = useWatcherWebSocket({
    enabled: !!user,
    onEvent: (event, data) => {
      // Refresh state when watcher starts/stops
      if (event === "watcher.started" || event === "watcher.stopped") {
        fetchState();
      }
    },
  });

  // Watcher control handlers
  const handleStart = async () => {
    try {
      await startWatcher();
      toast.success("Watcher started successfully");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to start watcher";
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
        {/* Header */}
        <header className="bg-white border-b border-neutral-200">
          <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
            <h1 className="text-xl font-light tracking-tighter">
              GengoWatcher
            </h1>
            <div className="flex items-center gap-4">
              <span className="font-mono text-xs text-neutral-500 uppercase tracking-widest">
                {user?.email}
              </span>
              <Link
                href="/settings"
                className="font-mono text-xs text-neutral-500 uppercase tracking-widest hover:text-blue-600"
              >
                Settings
              </Link>
              <button
                onClick={logout}
                className="font-mono text-xs text-neutral-900 uppercase tracking-widest hover:text-blue-600"
              >
                Sign Out
              </button>
            </div>
          </div>
        </header>

        {/* Dashboard Content */}
        <div className="max-w-6xl mx-auto px-6 py-12">
          <div className="mb-8">
            <h2 className="text-4xl font-light tracking-tighter mb-2">
              Dashboard
            </h2>
            <p className="text-neutral-500 font-mono text-xs uppercase tracking-widest">
              Welcome back
            </p>
          </div>

          {/* Bento Grid */}
          <div className="grid grid-cols-3 gap-6">
            {/* Watcher Status */}
            <div className="bento-card p-6">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-red-600 font-mono text-xs uppercase tracking-widest">
                  Watcher Status
                </h3>
                {connected && (
                  <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                )}
              </div>
              <p
                role="status"
                aria-live="polite"
                className={`text-3xl font-light ${statusDisplay.color}`}
              >
                {statusDisplay.text}
              </p>
            </div>

            {/* Jobs Found */}
            <div className="bento-card p-6">
              <h3 className="text-orange-600 font-mono text-xs uppercase tracking-widest mb-2">
                Jobs Found
              </h3>
              <p
                role="status"
                aria-live="polite"
                className="text-3xl font-light"
              >
                {state?.total_jobs_found ?? 0}
              </p>
            </div>

            {/* Earnings */}
            <div className="bento-card p-6">
              <h3 className="text-yellow-600 font-mono text-xs uppercase tracking-widest mb-2">
                Earnings
              </h3>
              <p
                role="status"
                aria-live="polite"
                className="text-3xl font-light"
              >
                ${state?.total_earnings?.toFixed(2) ?? "0.00"}
              </p>
            </div>

            {/* Watcher Configuration */}
            <div className="bento-card p-6 col-span-2">
              <h3 className="text-green-600 font-mono text-xs uppercase tracking-widest mb-4">
                Watcher Configuration
              </h3>
              {configLoading ? (
                <p className="text-sm text-neutral-400">Loading configuration...</p>
              ) : configError ? (
                <p className="text-sm text-red-500">{configError}</p>
              ) : config ? (
                <div className="space-y-3">
                  <div className="flex justify-between py-2 border-b border-neutral-100">
                    <span className="text-sm text-neutral-600">RSS Feed</span>
                    <span className="font-mono text-xs truncate max-w-[200px]">
                      {config.rss_feed_url}
                    </span>
                  </div>
                  <div className="flex justify-between py-2 border-b border-neutral-100">
                    <span className="text-sm text-neutral-600">Min Reward</span>
                    <span className="font-mono text-xs">
                      ${config.min_reward.toFixed(2)}
                    </span>
                  </div>
                  <div className="flex justify-between py-2 border-b border-neutral-100">
                    <span className="text-sm text-neutral-600">Max Reward</span>
                    <span className="font-mono text-xs">
                      ${config.max_reward.toFixed(2)}
                    </span>
                  </div>
                  <div className="flex justify-between py-2 border-b border-neutral-100">
                    <span className="text-sm text-neutral-600">WebSocket</span>
                    <span className="font-mono text-xs">
                      {config.websocket_enabled ? "Enabled" : "Disabled"}
                    </span>
                  </div>
                  <div className="flex justify-between py-2">
                    <span className="text-sm text-neutral-600">Auto Accept</span>
                    <span className="font-mono text-xs">
                      {config.auto_accept_enabled ? "Enabled" : "Disabled"}
                    </span>
                  </div>
                </div>
              ) : (
                <p className="text-sm text-neutral-400">No configuration found</p>
              )}
            </div>

            {/* Actions */}
            <div className="bento-card p-6">
              <h3 className="text-cyan-600 font-mono text-xs uppercase tracking-widest mb-4">
                Actions
              </h3>
              <div className="space-y-3">
                <button
                  onClick={handleStart}
                  disabled={isRunning || stateLoading}
                  className="w-full py-3 bg-neutral-900 text-white text-sm transition-colors duration-150 hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {stateLoading ? "Loading..." : "Start Watcher"}
                </button>
                <button
                  onClick={handleStop}
                  disabled={!isRunning || stateLoading}
                  className="w-full py-3 border border-neutral-300 text-sm transition-colors duration-150 hover:border-red-600 hover:text-red-600 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {stateLoading ? "Loading..." : "Stop Watcher"}
                </button>
                <button
                  onClick={() => setConfigModalOpen(true)}
                  className="w-full py-3 border border-neutral-300 text-sm transition-colors duration-150 hover:border-neutral-400 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                  title="Keyboard shortcut: Ctrl+K"
                >
                  Configure
                  <kbd className="font-mono text-[10px] px-1.5 py-0.5 bg-neutral-100 rounded text-neutral-500">
                    Ctrl+K
                  </kbd>
                </button>
              </div>
            </div>
          </div>

          {/* Last Activity */}
          {state?.last_activity && (
            <div className="mt-6 text-center">
              <p className="font-mono text-xs text-neutral-400">
                Last activity: {new Date(state.last_activity).toLocaleString()}
              </p>
            </div>
          )}

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
