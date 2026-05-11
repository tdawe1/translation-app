"use client";

import { BentoCard } from "@/components/ui/base/BentoCard";
import type { WatcherState } from "@/lib/api";

interface WatcherOperationsConsoleProps {
  state: WatcherState | null;
  loading: boolean;
  onRestartBrowser?: () => void;
  onCaptureScreenshot?: () => void;
  browserControlLoading?: boolean;
}

const STATUS_STYLES: Record<string, string> = {
  stopped: "text-neutral-500 border-neutral-300 bg-neutral-100",
  degraded: "text-amber-700 border-amber-300 bg-amber-50",
  blocked: "text-red-700 border-red-300 bg-red-50",
  failed: "text-red-700 border-red-300 bg-red-50",
  monitoring: "text-blue-700 border-blue-300 bg-blue-50",
  unconfigured: "text-yellow-700 border-yellow-300 bg-yellow-50",
  starting: "text-blue-700 border-blue-300 bg-blue-50",
  ready: "text-green-700 border-green-300 bg-green-50",
  busy: "text-blue-700 border-blue-300 bg-blue-50",
  idle: "text-neutral-700 border-neutral-300 bg-neutral-100",
  queued: "text-blue-700 border-blue-300 bg-blue-50",
  opening: "text-blue-700 border-blue-300 bg-blue-50",
  open: "text-cyan-700 border-cyan-300 bg-cyan-50",
  accepting: "text-blue-700 border-blue-300 bg-blue-50",
  accepted: "text-green-700 border-green-300 bg-green-50",
  warning: "text-amber-700 border-amber-300 bg-amber-50",
  critical: "text-red-700 border-red-300 bg-red-50",
  none: "text-green-700 border-green-300 bg-green-50",
  running: "text-green-700 border-green-300 bg-green-50",
  dashboard: "text-cyan-700 border-cyan-300 bg-cyan-50",
};

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:37181";

const RUNTIME_DOMAINS = [
  { label: "Overall", key: "overall_status" },
  { label: "Feeds", key: "feed_status" },
  { label: "Browser", key: "browser_status" },
  { label: "Action", key: "action_status" },
  { label: "Alerts", key: "alert_status" },
] as const;

function statusClass(status?: string) {
  if (!status) {
    return "text-neutral-500 border-neutral-300 bg-neutral-100";
  }
  return STATUS_STYLES[status] || "text-neutral-700 border-neutral-300 bg-neutral-100";
}

function formatDateTime(value?: string) {
  if (!value) return "Waiting";
  return new Date(value).toLocaleString();
}

export function WatcherOperationsConsole({
  state,
  loading,
  onRestartBrowser,
  onCaptureScreenshot,
  browserControlLoading = false,
}: WatcherOperationsConsoleProps) {
  if (loading || !state) {
    return (
      <BentoCard accentColor="indigo" staggerIndex={0} className="p-6">
        <h3 className="font-mono text-xs uppercase tracking-widest text-indigo-600 mb-4">
          Operations Console
        </h3>
        <p className="text-neutral-500">Loading watcher runtime...</p>
      </BentoCard>
    );
  }

  const actionLabel = state.current_action_step || "Idle, monitoring feeds";
  const wsActivity =
    state.last_ws_message_at || state.last_ws_pong_at || state.last_ws_connect_at;
  const screenshotSrc = state.latest_screenshot_url
    ? `${API_URL}${state.latest_screenshot_url}`
    : null;

  return (
    <div className="grid grid-cols-1 xl:grid-cols-[1.4fr_0.9fr_1fr] gap-6">
      <BentoCard accentColor="indigo" staggerIndex={0} className="p-6">
        <div className="flex items-start justify-between gap-4 mb-5">
          <div>
            <h3 className="font-mono text-xs uppercase tracking-widest text-indigo-600 mb-2">
              Operations Console
            </h3>
            <p className="text-2xl font-light tracking-tight text-neutral-900">
              Worker {state.worker_id.slice(0, 8)}
            </p>
          </div>
          <div className="text-right">
            <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">Profile</p>
            <p className="text-sm text-neutral-900">{state.profile_status}</p>
          </div>
        </div>

        <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
          {RUNTIME_DOMAINS.map((domain) => {
            const value = state[domain.key];
            return (
              <div key={domain.key} className="border border-neutral-200 p-3">
                <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500 mb-2">
                  {domain.label}
                </p>
                <span
                  className={`inline-flex items-center border px-2 py-1 text-xs font-mono uppercase tracking-widest ${statusClass(value)}`}
                >
                  {value}
                </span>
              </div>
            );
          })}
        </div>
      </BentoCard>

      <BentoCard accentColor="cyan" staggerIndex={1} className="p-6">
        <h3 className="font-mono text-xs uppercase tracking-widest text-cyan-600 mb-4">
          Current Action
        </h3>
        <p className="text-xl font-light tracking-tight text-neutral-900 mb-3">{actionLabel}</p>
        <div className="space-y-3 text-sm text-neutral-600">
          <div className="flex items-center justify-between gap-4">
            <span className="font-mono text-[11px] uppercase tracking-widest">Job</span>
            <span className="text-right text-neutral-900">{state.current_job_id || "None queued"}</span>
          </div>
          <div className="flex items-center justify-between gap-4">
            <span className="font-mono text-[11px] uppercase tracking-widest">Last Error</span>
            <span className="text-right text-neutral-900">{state.last_error || "Clear"}</span>
          </div>
          <div className="flex items-center justify-between gap-4">
            <span className="font-mono text-[11px] uppercase tracking-widest">Last Activity</span>
            <span className="text-right text-neutral-900">{formatDateTime(state.last_activity)}</span>
          </div>
        </div>
      </BentoCard>

      <BentoCard accentColor="blue" staggerIndex={2} className="p-6">
        <h3 className="font-mono text-xs uppercase tracking-widest text-blue-600 mb-4">
          Feed Health
        </h3>
        <div className="space-y-4">
          <div>
            <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500 mb-1">RSS</p>
            <p className="text-sm text-neutral-900">Last poll: {formatDateTime(state.last_rss_poll_ok_at)}</p>
            <p className="text-sm text-neutral-600">
              Consecutive failures: {state.rss_consecutive_failures ?? 0}
            </p>
          </div>
          <div>
            <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500 mb-1">
              Realtime WebSocket
            </p>
            <p className="text-sm text-neutral-900">
              Last activity: {formatDateTime(wsActivity)}
            </p>
            <p className="text-sm text-neutral-600">
              Reconnects: {state.ws_reconnect_count ?? 0}
            </p>
          </div>
        </div>
      </BentoCard>

      <BentoCard accentColor="violet" staggerIndex={3} className="p-6 xl:col-span-2">
        <h3 className="font-mono text-xs uppercase tracking-widest text-violet-600 mb-4">
          Worker Browser
        </h3>
        <div className="mb-4 flex flex-wrap gap-3">
          <button
            type="button"
            disabled={browserControlLoading || !onCaptureScreenshot}
            onClick={onCaptureScreenshot}
            className="border border-violet-600 px-3 py-2 font-mono text-[11px] uppercase tracking-widest text-violet-700 disabled:cursor-not-allowed disabled:border-neutral-300 disabled:text-neutral-400"
          >
            Force Screenshot
          </button>
          <button
            type="button"
            disabled={browserControlLoading || !onRestartBrowser}
            onClick={onRestartBrowser}
            className="border border-red-300 px-3 py-2 font-mono text-[11px] uppercase tracking-widest text-red-700 disabled:cursor-not-allowed disabled:border-neutral-300 disabled:text-neutral-400"
          >
            Restart Browser
          </button>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-[220px_1fr] gap-5">
          <div className="border border-neutral-200 bg-neutral-100 aspect-video flex items-center justify-center overflow-hidden">
            {screenshotSrc ? (
              <img
                src={screenshotSrc}
                alt="Latest worker browser screenshot"
                className="h-full w-full object-cover"
              />
            ) : (
              <span className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
                No screenshot
              </span>
            )}
          </div>
          <div className="space-y-3 text-sm text-neutral-600">
            <div>
              <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
                Current URL
              </p>
              <p className="break-all text-neutral-900">{state.current_url || "Waiting"}</p>
            </div>
          <div>
            <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
              Page Title
            </p>
            <p className="text-neutral-900">{state.current_title || "Waiting"}</p>
          </div>
          <div>
            <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
              Frontend Location
            </p>
            <p className="break-all text-neutral-900">{state.frontend_url || "Not synced"}</p>
            <p className="text-neutral-500">
              {state.frontend_title || "Waiting for dashboard heartbeat"}
            </p>
          </div>
          <div className="grid grid-cols-2 gap-3">
              <div>
                <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
                  Process
                </p>
                <p className="text-neutral-900">
                  {state.browser_process_alive ? "Alive" : "Not running"}
                </p>
              </div>
              <div>
                <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
                  DevTools
                </p>
                <p className="text-neutral-900">
                  {state.devtools_connected ? "Connected" : "Disconnected"}
                </p>
              </div>
            </div>
            <div>
              <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
                Browser Heartbeat
              </p>
              <p className="text-neutral-900">{formatDateTime(state.last_browser_heartbeat_at)}</p>
            </div>
            <div>
              <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
                Latest Artifact
              </p>
              <p className="break-all text-neutral-900">
                {state.latest_screenshot_artifact_id || "None captured"}
              </p>
            </div>
            {state.last_critical_alert ? (
              <p className="border border-red-200 bg-red-50 p-3 text-sm text-red-700">
                {state.last_critical_alert}
              </p>
            ) : null}
          </div>
        </div>
      </BentoCard>
    </div>
  );
}
