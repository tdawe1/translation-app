/**
 * Dashboard Page - Protected route with watcher controls
 *
 * Enhanced with Data Factory base components and improved stats cards.
 */

"use client";

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
import { DashboardHeader } from "@/components/dashboard/DashboardHeader";
import { DashboardStats } from "@/components/dashboard/DashboardStats";
import { DashboardContentHeader } from "@/components/dashboard/DashboardContentHeader";
import { LastActivity } from "@/components/dashboard/LastActivity";
import { authApi } from "@/lib/api";
import { toast } from "@/store/toast";
import { useEffect, useState } from "react";
import { navigateToHome } from "@/lib/navigation";

export default function DashboardPage() {
  const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clear);
  const [configModalOpen, setConfigModalOpen] = useState(false);
  const [startError, setStartError] = useState<string | null>(null);

  const handleLogout = async () => {
    try {
      await authApi.logout();
    } catch (err) {
      // Continue with logout even if API call fails
    } finally {
      clearAuth();
      navigateToHome();
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
          <DashboardHeader user={user} onLogout={handleLogout} />

          <div className="max-w-6xl mx-auto px-6 py-12">
            <DashboardContentHeader title="Dashboard" meta="WELCOME BACK" accentColor="blue" />

            <DashboardStats
              connected={connected}
              configLoading={configLoading}
              configError={configError}
              config={config}
              state={state}
              stateLoading={stateLoading}
              statusDisplay={statusDisplay}
              isRunning={isRunning}
              onStartWatcher={handleStart}
              onStopWatcher={handleStop}
              onConfigure={() => setConfigModalOpen(true)}
              startError={startError}
              onDismissStartError={() => setStartError(null)}
              onLogout={handleLogout}
            />

            <LastActivity lastActivity={state?.last_activity} />

            <div className="mt-12">
              <RealtimeSection
                connected={connected}
                uptime={uptime}
                lastMessageTime={lastMessageTime}
              />
            </div>

            <div className="mt-8">
              <JobList />
            </div>
          </div>

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
