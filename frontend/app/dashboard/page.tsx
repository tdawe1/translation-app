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
import { WatcherOperationsConsole } from "@/components/dashboard/WatcherOperationsConsole";
import { authApi, watcherApi } from "@/lib/api";
import {
  alertForDetectedJob,
  formatJobAlertSummary,
  getSafeJobUrl,
  getPreparedJobAlertWindowState,
  playJobAlertSound,
  prepareJobAlertWindow,
  requestJobAlertPermission,
  showJobNotification,
  type JobAlertPayload,
  unlockJobAlertSound,
} from "@/lib/job-alerts";
import { toast } from "@/store/toast";
import { useCallback, useEffect, useRef, useState } from "react";
import { navigateToHome } from "@/lib/navigation";
import type { WatcherEvent } from "@/lib/api";

const LOCAL_FIREFOX_OPEN_KEY = "gengowatcher.openJobsInLocalFirefox";

type DashboardSection = "overview" | "operations" | "activity" | "jobs";

const DASHBOARD_SECTIONS: Array<{
  id: DashboardSection;
  label: string;
  eyebrow: string;
  title: string;
}> = [
  {
    id: "overview",
    label: "Overview",
    eyebrow: "CONTROL ROOM",
    title: "Watcher overview",
  },
  {
    id: "operations",
    label: "Operations",
    eyebrow: "RUNTIME",
    title: "Worker operations",
  },
  {
    id: "activity",
    label: "Activity",
    eyebrow: "LIVE FEED",
    title: "Realtime activity",
  },
  {
    id: "jobs",
    label: "Jobs",
    eyebrow: "QUEUE",
    title: "Detected jobs",
  },
];

function isFirefoxBrowser(): boolean {
  if (typeof navigator === "undefined") {
    return false;
  }
  return navigator.userAgent.toLowerCase().includes("firefox");
}

function deriveRealtimeMessage(event: string, data: unknown): string {
  if (data && typeof data === "object" && data !== null) {
    const payload = data as Record<string, unknown>;
    if (typeof payload.message === "string" && payload.message.length > 0) {
      return payload.message;
    }
    if (event === "job.detected" && typeof payload.title === "string") {
      return `Job detected: ${payload.title}`;
    }
  }

  switch (event) {
    case "worker.started":
      return "Watcher worker started";
    case "worker.stopped":
      return "Watcher worker stopped";
    case "worker.ready":
      return "Worker browser ready";
    case "worker.blocked":
      return "Worker blocked; manual inspection required";
    case "worker.restore_started":
      return "Restoring watcher after backend restart";
    case "worker.restore_succeeded":
      return "Watcher restored after backend restart";
    case "worker.restore_failed":
      return "Watcher restore failed";
    case "worker.shutdown_persisted":
      return "Backend shutdown persisted watcher for restore";
    case "browser.unconfigured":
      return "Browser worker is not configured";
    case "browser.dashboard_mode":
      return "Dashboard auto-open mode enabled";
    case "browser.started":
      return "Worker browser starting";
    case "browser.ready":
      return "Worker browser ready";
    case "browser.start_failed":
      return "Worker browser failed to start";
    case "browser.job_open_started":
      return "Opening job page in worker browser";
    case "browser.job_open_succeeded":
      return "Job page opened in worker browser";
    case "browser.screenshot_captured":
      return "Worker browser screenshot captured";
    case "browser.captcha_detected":
      return "Captcha or challenge detected";
    case "browser.suspicious_login_detected":
      return "Suspicious login prompt detected";
    case "action.accept_started":
      return "Accept click started";
    case "action.accept_succeeded":
      return "Job accepted in worker browser";
    case "action.accept_failed":
      return "Accept action failed";
    case "job.matched":
      return "Job matched watcher filters";
    case "rss.poll_started":
      return "RSS poll started";
    case "rss.poll_ok":
      return "RSS poll completed";
    case "websocket.connected":
      return "Gengo WSS connected";
    case "websocket.message":
      return "Gengo WSS message received";
    case "websocket.pong":
      return "Gengo WSS pong received";
    case "websocket.disconnected":
      return "Gengo WSS disconnected";
    default:
      return event;
  }
}

function mapWatcherEventToRealtime(event: WatcherEvent) {
  return {
    id: event.id,
    type: event.type,
    level: event.level,
    source: event.source,
    message: event.message,
    data: event.data,
    timestamp: event.occurred_at,
  };
}

export default function DashboardPage() {
  const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clear);
  const [configModalOpen, setConfigModalOpen] = useState(false);
  const [activeSection, setActiveSection] =
    useState<DashboardSection>("overview");
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [startError, setStartError] = useState<string | null>(null);
  const [openJobsInLocalFirefox, setOpenJobsInLocalFirefox] = useState(false);
  const [dashboardIsFirefox, setDashboardIsFirefox] = useState(false);
  const [browserControlLoading, setBrowserControlLoading] = useState(false);
  const openedJobIds = useRef<Set<string>>(new Set());
  const setRealtimeEvents = useRealtimeStore((store) => store.setEvents);

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
    stateLoading,
    fetchConfig,
    fetchState,
    startWatcher,
    stopWatcher,
  } = useWatcherStore();

  // Load initial data
  useEffect(() => {
    let cancelled = false;

    void Promise.all([fetchConfig(), fetchState()]);

    void watcherApi
      .getEvents()
      .then((response) => {
        if (!cancelled) {
          setRealtimeEvents(response.events.map(mapWatcherEventToRealtime));
        }
      })
      .catch((error) => {
        console.error("Failed to load watcher events", error);
      });

    return () => {
      cancelled = true;
    };
  }, [fetchConfig, fetchState, setRealtimeEvents]);

  useEffect(() => {
    if (!user) {
      return;
    }

    let cancelled = false;

    const syncFrontendState = () => {
      void watcherApi
        .syncBrowserState({
          frontend_url: window.location.href,
          frontend_title: document.title || "GengoWatcher Dashboard",
        })
        .then(() => {
          if (!cancelled) {
            void fetchState();
          }
        })
        .catch((error) => {
          console.error("Failed to sync frontend browser state", error);
        });
    };

    syncFrontendState();
    const interval = window.setInterval(syncFrontendState, 30_000);
    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        syncFrontendState();
      }
    };
    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      cancelled = true;
      window.clearInterval(interval);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [fetchState, user]);

  useEffect(() => {
    setDashboardIsFirefox(isFirefoxBrowser());
    setOpenJobsInLocalFirefox(
      window.localStorage.getItem(LOCAL_FIREFOX_OPEN_KEY) === "true",
    );
  }, []);

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

  useEffect(() => {
    if (
      !config?.enable_sound_notifications &&
      !config?.enable_desktop_notifications
    ) {
      return;
    }

    const armAlerts = () => {
      if (config.enable_sound_notifications) {
        unlockJobAlertSound();
      }
      if (config.enable_desktop_notifications) {
        void requestJobAlertPermission();
      }
      window.removeEventListener("pointerdown", armAlerts);
      window.removeEventListener("keydown", armAlerts);
      window.removeEventListener("touchstart", armAlerts);
    };

    window.addEventListener("pointerdown", armAlerts, { once: true });
    window.addEventListener("keydown", armAlerts, { once: true });
    window.addEventListener("touchstart", armAlerts, { once: true });

    return () => {
      window.removeEventListener("pointerdown", armAlerts);
      window.removeEventListener("keydown", armAlerts);
      window.removeEventListener("touchstart", armAlerts);
    };
  }, [
    config?.enable_desktop_notifications,
    config?.enable_sound_notifications,
  ]);

  const setLocalFirefoxOpenPreference = useCallback((enabled: boolean) => {
    setOpenJobsInLocalFirefox(enabled);
    window.localStorage.setItem(LOCAL_FIREFOX_OPEN_KEY, String(enabled));
  }, []);

  const handleArmLocalFirefoxWindow = useCallback(() => {
    if (!dashboardIsFirefox) {
      toast.error("Open this dashboard in Firefox to use Firefox local job tabs.");
      return;
    }
    if (prepareJobAlertWindow()) {
      toast.success("Firefox job tab armed for local opens.");
      void watcherApi.syncBrowserState({
        frontend_url: window.location.href,
        frontend_title: "Firefox local job tab armed",
      });
    } else {
      toast.error("Firefox blocked the job tab. Click the dashboard and try again.");
    }
  }, [dashboardIsFirefox]);

  const handleDetectedJob = useCallback(
    (payload: JobAlertPayload) => {
      if (openedJobIds.current.has(payload.id)) {
        return;
      }

      openedJobIds.current.add(payload.id);
      const summary = formatJobAlertSummary(payload);
      let soundPlayed = false;

      if (config?.enable_sound_notifications) {
        soundPlayed = playJobAlertSound();
      }

      if (config?.enable_desktop_notifications === false) {
        toast.info(`Job detected: ${summary}`);
        return;
      }

      const safeUrl = getSafeJobUrl(payload.url);
      if (!safeUrl) {
        toast.error("Job detected, but the job URL was not a safe public URL.");
        return;
      }

      if (openJobsInLocalFirefox) {
        if (!dashboardIsFirefox) {
          toast.error("Local job open skipped: this dashboard is not running in Firefox.");
          return;
        }

        const localOpen = alertForDetectedJob(payload);
        void watcherApi.syncBrowserState({
          frontend_url: window.location.href,
          frontend_title: localOpen.opened
            ? `Firefox opened job ${payload.id}`
            : `Firefox local open blocked for ${payload.id}`,
          ...getPreparedJobAlertWindowState(),
        });
        if (localOpen.opened) {
          toast.success(`Opened locally in Firefox: ${summary}`);
        } else {
          toast.error(localOpen.reason || "Firefox blocked the local job tab.");
        }
        return;
      }

      const notified = showJobNotification(
        payload,
        safeUrl,
        "worker browser is handling it",
      );
      const suffix =
        config?.enable_sound_notifications && !soundPlayed
          ? " (click the dashboard once to arm sound)"
          : "";
      toast.info(
        `Job detected: ${summary}. Worker browser is handling it${suffix}`,
      );
      if (!notified && config?.enable_desktop_notifications) {
        void requestJobAlertPermission();
      }
    },
    [
      config?.enable_desktop_notifications,
      config?.enable_sound_notifications,
      dashboardIsFirefox,
      openJobsInLocalFirefox,
    ],
  );

  const handleRealtimeEvent = useCallback(
    (event: string, data: unknown) => {
      if (event === "watcher.health") {
        return;
      }

      // Refresh state when watcher starts/stops or job counters change.
      if (
        event === "worker.started" ||
        event === "worker.stopped" ||
        event === "worker.ready" ||
        event === "worker.blocked" ||
        event === "worker.restore_started" ||
        event === "worker.restore_succeeded" ||
        event === "worker.restore_failed" ||
        event === "worker.shutdown_persisted" ||
        event === "browser.unconfigured" ||
        event === "browser.started" ||
        event === "browser.ready" ||
        event === "browser.start_failed" ||
        event === "browser.job_open_started" ||
        event === "browser.job_open_succeeded" ||
        event === "browser.screenshot_captured" ||
        event === "browser.captcha_detected" ||
        event === "browser.suspicious_login_detected" ||
        event === "action.accept_started" ||
        event === "action.accept_succeeded" ||
        event === "action.accept_failed" ||
        event === "job.matched" ||
        event === "rss.poll_started" ||
        event === "rss.poll_ok" ||
        event === "websocket.connected" ||
        event === "websocket.message" ||
        event === "websocket.pong" ||
        event === "websocket.disconnected" ||
        event === "job.detected"
      ) {
        fetchState();
      }

      if (
        event === "worker.blocked" ||
        event === "browser.captcha_detected" ||
        event === "browser.suspicious_login_detected"
      ) {
        toast.error(deriveRealtimeMessage(event, data));
      }

      // Add events to realtime store for the feed.
      const { addEvent } = useRealtimeStore.getState();
      addEvent(
        event,
        deriveRealtimeMessage(event, data),
        data as Record<string, unknown> | undefined,
      );
    },
    [fetchState],
  );

  // Set up WebSocket for real-time updates
  const { connected, uptime, lastMessageTime } = useWatcherWebSocket({
    enabled: !!user,
    onJob: handleDetectedJob,
    onEvent: handleRealtimeEvent,
  });

  const handleStart = async () => {
    setStartError(null);
    if (config?.enable_sound_notifications) {
      unlockJobAlertSound();
    }
    if (config?.enable_desktop_notifications) {
      void requestJobAlertPermission();
    }

    try {
      await startWatcher();
      toast.success("Watcher started successfully");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to start watcher";
      setStartError(message);
      toast.error(message);
    }
  };

  const handleStop = async () => {
    try {
      await stopWatcher();
      toast.success("Watcher stopped successfully");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to stop watcher";
      toast.error(message);
    }
  };

  const handleRestartBrowser = async () => {
    setBrowserControlLoading(true);
    try {
      await watcherApi.restartBrowser();
      await fetchState();
      toast.success("Worker browser restarted");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to restart worker browser";
      toast.error(message);
    } finally {
      setBrowserControlLoading(false);
    }
  };

  const handleCaptureScreenshot = async () => {
    setBrowserControlLoading(true);
    try {
      await watcherApi.captureBrowserScreenshot();
      await fetchState();
      toast.success("Worker browser screenshot captured");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to capture worker browser screenshot";
      toast.error(message);
    } finally {
      setBrowserControlLoading(false);
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
  const activeSectionMeta =
    DASHBOARD_SECTIONS.find((section) => section.id === activeSection) ??
    DASHBOARD_SECTIONS[0];

  return (
    <ProtectedRoute>
      <ErrorBoundary>
        <main id="main-content" className="min-h-screen bg-neutral-50">
          <DashboardHeader user={user} onLogout={handleLogout} />

          <div
            className={`grid w-full grid-cols-1 gap-5 px-4 py-5 sm:px-6 lg:px-6 lg:py-6 ${
              sidebarCollapsed
                ? "lg:grid-cols-[56px_minmax(0,1fr)]"
                : "lg:grid-cols-[220px_minmax(0,1fr)]"
            }`}
          >
            <aside className="lg:sticky lg:top-6 lg:self-start">
              <nav
                aria-label="Dashboard sections"
                className="border border-neutral-200 bg-white"
              >
                <div className="flex items-center justify-between gap-2 border-b border-neutral-200 p-3">
                  {!sidebarCollapsed ? (
                    <div>
                      <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
                        Workspace
                      </p>
                      <p className="mt-1 text-lg font-light tracking-tight text-neutral-900">
                        Dashboard
                      </p>
                    </div>
                  ) : (
                    <span className="font-mono text-sm uppercase tracking-widest text-neutral-900">
                      GW
                    </span>
                  )}
                  <button
                    type="button"
                    onClick={() => setSidebarCollapsed((collapsed) => !collapsed)}
                    className="hidden border border-neutral-300 px-2 py-1 font-mono text-xs text-neutral-600 transition-colors duration-150 hover:border-blue-600 hover:text-blue-700 lg:block"
                    aria-label={sidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
                  >
                    {sidebarCollapsed ? ">" : "<"}
                  </button>
                </div>
                <div className="hidden p-2 lg:block">
                  {DASHBOARD_SECTIONS.map((section) => {
                    const isActive = activeSection === section.id;

                    return (
                      <button
                        key={section.id}
                        type="button"
                        onClick={() => setActiveSection(section.id)}
                        aria-current={isActive ? "page" : undefined}
                        title={sidebarCollapsed ? section.label : undefined}
                        className={`flex w-full items-center justify-between border-l-2 px-3 py-3 text-left transition-colors duration-150 ${
                          isActive
                            ? "border-blue-600 bg-neutral-100 text-blue-700"
                            : "border-transparent text-neutral-700 hover:border-neutral-300 hover:bg-neutral-50"
                        }`}
                      >
                        <span className="font-mono text-xs uppercase tracking-widest">
                          {sidebarCollapsed ? section.label.slice(0, 1) : section.label}
                        </span>
                      </button>
                    );
                  })}
                </div>
                <div className="grid grid-cols-4 gap-px bg-neutral-200 lg:hidden">
                  {DASHBOARD_SECTIONS.map((section) => {
                    const isActive = activeSection === section.id;

                    return (
                      <button
                        key={section.id}
                        type="button"
                        onClick={() => setActiveSection(section.id)}
                        aria-current={isActive ? "page" : undefined}
                        className={`bg-white px-2 py-3 font-mono text-[11px] uppercase tracking-widest transition-colors duration-150 ${
                          isActive
                            ? "text-blue-700"
                            : "text-neutral-600 hover:text-neutral-900"
                        }`}
                      >
                        {section.label}
                      </button>
                    );
                  })}
                </div>
              </nav>
            </aside>

            <section className="min-w-0">
              {activeSection !== "overview" ? (
                <DashboardContentHeader
                  title={activeSectionMeta.title}
                  meta={activeSectionMeta.eyebrow}
                  accentColor="blue"
                />
              ) : null}

              {activeSection === "overview" ? (
                <div className="space-y-4">
                  <DashboardStats
                    connected={connected}
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

                  <LastActivity
                    lastActivity={state?.last_activity}
                    connected={connected}
                    lastMessageTime={lastMessageTime}
                  />
                </div>
              ) : null}

              {activeSection === "operations" ? (
                <div className="space-y-8">
                  <div className="border border-neutral-200 bg-white p-5">
                    <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
                      <div>
                        <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
                          Local Firefox Open
                        </p>
                        <h2 className="mt-1 text-xl font-light tracking-tight text-neutral-900">
                          Open matched jobs on this frontend PC
                        </h2>
                        <p className="mt-2 max-w-2xl text-sm text-neutral-600">
                          This only opens in Firefox when this dashboard itself is running in Firefox.
                          Browsers do not allow a web page to choose a different installed browser.
                        </p>
                        {!dashboardIsFirefox ? (
                          <p className="mt-2 text-sm text-amber-700">
                            Current dashboard browser is not Firefox; local auto-open is disabled.
                          </p>
                        ) : null}
                      </div>
                      <div className="flex flex-wrap gap-3">
                        <button
                          type="button"
                          disabled={!dashboardIsFirefox}
                          onClick={() =>
                            setLocalFirefoxOpenPreference(!openJobsInLocalFirefox)
                          }
                          className="border border-neutral-900 px-4 py-2 font-mono text-xs uppercase tracking-widest text-neutral-900 disabled:cursor-not-allowed disabled:border-neutral-300 disabled:text-neutral-400"
                        >
                          {openJobsInLocalFirefox ? "Disable Firefox Open" : "Enable Firefox Open"}
                        </button>
                        <button
                          type="button"
                          disabled={!dashboardIsFirefox}
                          onClick={handleArmLocalFirefoxWindow}
                          className="border border-blue-600 px-4 py-2 font-mono text-xs uppercase tracking-widest text-blue-700 disabled:cursor-not-allowed disabled:border-neutral-300 disabled:text-neutral-400"
                        >
                          Arm Firefox Tab
                        </button>
                      </div>
                    </div>
                  </div>

                  <WatcherOperationsConsole
                    state={state}
                    loading={stateLoading}
                    onRestartBrowser={handleRestartBrowser}
                    onCaptureScreenshot={handleCaptureScreenshot}
                    browserControlLoading={browserControlLoading}
                  />
                </div>
              ) : null}

              {activeSection === "activity" ? (
                <RealtimeSection
                  connected={connected}
                  uptime={uptime}
                  lastMessageTime={lastMessageTime}
                />
              ) : null}

              {activeSection === "jobs" ? <JobList /> : null}
            </section>
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
