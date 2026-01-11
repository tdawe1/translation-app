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
const EVENT_BADGE_STYLES: Record<EventType, string> = {
  "watcher.started": "bg-green-900 text-green-300 border-green-700",
  "watcher.stopped": "bg-red-900 text-red-300 border-red-700",
  "job.detected": "bg-cyan-900 text-cyan-300 border-cyan-700",
  "job.accepted": "bg-green-900 text-green-300 border-green-700",
  "job.filtered": "bg-yellow-900 text-yellow-300 border-yellow-700",
  "job.rejected": "bg-orange-900 text-orange-300 border-orange-700",
  "rss.fetched": "bg-blue-900 text-blue-300 border-blue-700",
  "websocket.connected": "bg-indigo-900 text-indigo-300 border-indigo-700",
  "websocket.disconnected": "bg-violet-900 text-violet-300 border-violet-700",
  "error": "bg-red-900 text-red-300 border-red-700",
};

// Short event type labels for badges
const EVENT_LABELS: Record<EventType, string> = {
  "watcher.started": "STARTED",
  "watcher.stopped": "STOPPED",
  "job.detected": "DETECTED",
  "job.accepted": "ACCEPTED",
  "job.filtered": "FILTERED",
  "job.rejected": "REJECTED",
  "rss.fetched": "RSS",
  "websocket.connected": "WS+",
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
  const badgeClass = EVENT_BADGE_STYLES[event.type] || "bg-neutral-800 text-neutral-300 border-neutral-600";
  const badgeLabel = EVENT_LABELS[event.type] || event.type.toUpperCase();

  return (
    <div className="flex items-start gap-3 py-1 hover:bg-neutral-800/50 rounded px-2 transition-colors group">
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
      <span className="text-neutral-100 text-xs flex-1 truncate" title={event.message}>
        {event.message}
      </span>

      {/* Data hint on hover */}
      {event.data && (
        <span className="text-neutral-600 text-xs font-mono opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0" aria-hidden="true">
          {"{ }"}
        </span>
      )}
    </div>
  );
});

EventItem.displayName = "EventItem";
