/**
 * EventLog - Collapsible terminal-style event feed
 *
 * Displays realtime events in a collapsible, scrollable terminal view.
 */

import React, { useEffect, useRef, useState, useCallback } from "react";
import type { RealtimeEvent } from "@/store/realtime";
import { EventItem } from "./event-item";
import { useRealtimeStore } from "@/store/realtime";

interface EventLogProps {
  defaultCollapsed?: boolean;
  maxVisible?: number;
  testId?: string;
}

export const EventLog = React.memo<EventLogProps>(
  ({ defaultCollapsed = true, maxVisible = 50, testId }
) => {
  const [collapsed, setCollapsed] = useState(defaultCollapsed);
  const [autoScroll, setAutoScroll] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);
  const events = useRealtimeStore((state) => state.events);
  const clearEvents = useRealtimeStore((state) => state.clearEvents);

  // Track previous event count for auto-scroll
  const prevEventCountRef = useRef(events.length);

  // Auto-scroll to top when new events arrive and auto-scroll is enabled
  useEffect(() => {
    if (!collapsed && autoScroll && events.length > prevEventCountRef.current) {
      // Small delay to ensure DOM has updated
      const timeoutId = setTimeout(() => {
        scrollRef.current?.scrollTo({ top: 0, behavior: "smooth" });
      }, 50);
      return () => clearTimeout(timeoutId);
    }
    prevEventCountRef.current = events.length;
  }, [events.length, collapsed, autoScroll]);

  // Toggle collapsed state
  const toggleCollapsed = useCallback(() => {
    setCollapsed((prev) => !prev);
  }, []);

  // Toggle auto-scroll
  const toggleAutoScroll = useCallback(() => {
    setAutoScroll((prev) => !prev);
  }, []);

  // Clear all events
  const handleClear = useCallback(() => {
    clearEvents();
  }, [clearEvents]);

  const visibleEvents = events.slice(0, maxVisible);

  return (
    <div className="bento-card overflow-hidden" data-testid={testId}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-neutral-200">
        <h3 className="text-violet-600 font-mono text-xs uppercase tracking-widest flex items-center gap-2">
          <span>Event Log</span>
          <span className="text-neutral-400">({events.length})</span>
        </h3>
        <div className="flex items-center gap-3">
          {!collapsed && (
            <>
              {/* Auto-scroll toggle */}
              <label className="flex items-center gap-2 font-mono text-xs text-neutral-600 cursor-pointer hover:text-neutral-900">
                <input
                  type="checkbox"
                  checked={autoScroll}
                  onChange={toggleAutoScroll}
                  className="w-3 h-3 accent-violet-600"
                  aria-label="Toggle auto-scroll"
                />
                <span>Auto-scroll</span>
              </label>

              {/* Clear button */}
              <button
                onClick={handleClear}
                className="font-mono text-xs text-neutral-400 hover:text-red-600 transition-colors"
                aria-label="Clear all events"
              >
                Clear
              </button>
            </>
          )}

          {/* Collapse/Expand toggle */}
          <button
            onClick={toggleCollapsed}
            aria-label={collapsed ? "Expand event log" : "Collapse event log"}
            aria-expanded={!collapsed}
            className="p-1 hover:bg-neutral-100 rounded transition-colors"
            type="button"
          >
            <svg
              className={`w-4 h-4 text-neutral-500 transition-transform ${!collapsed ? "rotate-180" : ""}`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
            </svg>
          </button>
        </div>
      </div>

      {/* Terminal-style event list */}
      {!collapsed && (
        <div
          ref={scrollRef}
          className="border-t border-neutral-200 bg-neutral-100 text-neutral-800 font-mono text-xs p-4 max-h-[300px] overflow-y-auto terminal-scroll"
          aria-live="polite"
          aria-atomic="false"
        >
          {visibleEvents.length === 0 ? (
            <p className="text-neutral-500 italic text-center py-4">
              No events yet... Start the watcher to begin monitoring.
            </p>
          ) : (
            <div className="space-y-0.5">
              {visibleEvents.map((event) => (
                <EventItem key={event.id} event={event} showTimestamp />
              ))}
            </div>
          )}

          {/* Truncated indicator */}
          {events.length > maxVisible && (
            <div className="mt-2 pt-2 border-t border-neutral-300 text-center">
              <span className="text-neutral-500 text-xs">
                Showing {maxVisible} of {events.length} events
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
});

EventLog.displayName = "EventLog";
