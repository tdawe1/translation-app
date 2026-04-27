/**
 * EventItem - Individual event display for the realtime feed
 *
 * Terminal-style monospace display with timestamp and event type badge.
 */

import React from "react";
import type { RealtimeEvent, EventType } from "@/store/realtime";

interface EventItemProps {
  event: RealtimeEvent;
  showTimestamp?: boolean;
}

// Event type badge styling
const EVENT_BADGE_STYLES: Partial<Record<EventType, string>> = {
  "worker.started": "bg-green-50 text-green-700 border-green-300",
  "worker.stopped": "bg-red-50 text-red-700 border-red-300",
  "worker.ready": "bg-green-50 text-green-700 border-green-300",
  "worker.blocked": "bg-red-50 text-red-700 border-red-300",
  "worker.restore_started": "bg-blue-50 text-blue-700 border-blue-300",
  "worker.restore_succeeded": "bg-green-50 text-green-700 border-green-300",
  "worker.restore_failed": "bg-red-50 text-red-700 border-red-300",
  "worker.shutdown_persisted": "bg-yellow-50 text-yellow-700 border-yellow-300",
  "browser.unconfigured": "bg-yellow-50 text-yellow-700 border-yellow-300",
  "browser.dashboard_mode": "bg-cyan-50 text-cyan-700 border-cyan-300",
  "browser.started": "bg-blue-50 text-blue-700 border-blue-300",
  "browser.ready": "bg-green-50 text-green-700 border-green-300",
  "browser.start_failed": "bg-red-50 text-red-700 border-red-300",
  "browser.job_open_started": "bg-blue-50 text-blue-700 border-blue-300",
  "browser.job_open_succeeded": "bg-green-50 text-green-700 border-green-300",
  "browser.screenshot_captured": "bg-violet-50 text-violet-700 border-violet-300",
  "browser.captcha_detected": "bg-red-50 text-red-700 border-red-300",
  "browser.suspicious_login_detected": "bg-red-50 text-red-700 border-red-300",
  "action.accept_started": "bg-blue-50 text-blue-700 border-blue-300",
  "action.accept_succeeded": "bg-green-50 text-green-700 border-green-300",
  "action.accept_failed": "bg-red-50 text-red-700 border-red-300",
  "job.detected": "bg-cyan-50 text-cyan-700 border-cyan-300",
  "job.matched": "bg-green-50 text-green-700 border-green-300",
  "job.accepted": "bg-green-50 text-green-700 border-green-300",
  "job.filtered": "bg-yellow-50 text-yellow-700 border-yellow-300",
  "job.rejected": "bg-orange-50 text-orange-700 border-orange-300",
  "rss.fetched": "bg-blue-50 text-blue-700 border-blue-300",
  "rss.poll_started": "bg-blue-50 text-blue-700 border-blue-300",
  "rss.poll_ok": "bg-green-50 text-green-700 border-green-300",
  "websocket.connected": "bg-indigo-50 text-indigo-700 border-indigo-300",
  "websocket.message": "bg-cyan-50 text-cyan-700 border-cyan-300",
  "websocket.pong": "bg-green-50 text-green-700 border-green-300",
  "websocket.disconnected": "bg-violet-50 text-violet-700 border-violet-300",
  "error": "bg-red-50 text-red-700 border-red-300",
};

// Short event type labels for badges
const EVENT_LABELS: Partial<Record<EventType, string>> = {
  "worker.started": "STARTED",
  "worker.stopped": "STOPPED",
  "worker.ready": "READY",
  "worker.blocked": "BLOCKED",
  "worker.restore_started": "RESTORE",
  "worker.restore_succeeded": "RESTORED",
  "worker.restore_failed": "RFAIL",
  "worker.shutdown_persisted": "SAVE",
  "browser.unconfigured": "BROWSER",
  "browser.dashboard_mode": "DASH",
  "browser.started": "BROWSER+",
  "browser.ready": "BREADY",
  "browser.start_failed": "BFAIL",
  "browser.job_open_started": "OPENING",
  "browser.job_open_succeeded": "OPENED",
  "browser.screenshot_captured": "SHOT",
  "browser.captcha_detected": "CAPTCHA",
  "browser.suspicious_login_detected": "LOGIN",
  "action.accept_started": "ACCEPT",
  "action.accept_succeeded": "ACCEPTED",
  "action.accept_failed": "AFAIL",
  "job.detected": "DETECTED",
  "job.matched": "MATCHED",
  "job.accepted": "ACCEPTED",
  "job.filtered": "FILTERED",
  "job.rejected": "REJECTED",
  "rss.fetched": "RSS",
  "rss.poll_started": "RSS+",
  "rss.poll_ok": "RSSOK",
  "websocket.connected": "WS+",
  "websocket.message": "WSS",
  "websocket.pong": "PONG",
  "websocket.disconnected": "WS-",
  "error": "ERROR",
};

// Format timestamp as HH:MM:SS local time
const formatTimestamp = (isoString: string): string => {
  const date = new Date(isoString);
  return date.toLocaleTimeString("en-US", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
};

export const EventItem = React.memo<EventItemProps>(({ event, showTimestamp = true }) => {
  const badgeClass =
    EVENT_BADGE_STYLES[event.type] ||
    (event.level === "critical"
      ? "bg-red-50 text-red-700 border-red-300"
      : "bg-white text-neutral-700 border-neutral-300");
  const badgeLabel = EVENT_LABELS[event.type] || event.type.toUpperCase();

  return (
    <div className="flex items-start gap-3 py-1 hover:bg-white rounded px-2 transition-colors group">
      {/* Timestamp */}
      {showTimestamp && (
        <span className="text-neutral-500 font-mono text-xs flex-shrink-0" aria-hidden="true">
          {formatTimestamp(event.timestamp)}
        </span>
      )}

      {/* Event type badge */}
      <span
        className={`px-1.5 py-0.5 text-[10px] font-mono uppercase rounded border ${badgeClass} flex-shrink-0`}
        aria-label={`Event type: ${event.type}`}
      >
        {badgeLabel}
      </span>

      {/* Message */}
      <span className="text-neutral-800 text-xs flex-1 truncate" title={event.message}>
        {event.message}
      </span>

      {/* Data hint on hover */}
      {event.data && (
        <span className="text-neutral-400 text-xs font-mono opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0" aria-hidden="true">
          {"{ }"}
        </span>
      )}
    </div>
  );
});

EventItem.displayName = "EventItem";
