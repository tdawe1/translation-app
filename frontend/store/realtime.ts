/**
 * Realtime Events Store - Zustand state management for live event feed
 *
 * Tracks watcher activity events in a rolling window for the realtime feed.
 */

import { create } from "zustand";

export type EventType =
  | "worker.started"
  | "worker.stopped"
  | "worker.restore_started"
  | "worker.restore_succeeded"
  | "worker.restore_failed"
  | "worker.shutdown_persisted"
  | "browser.unconfigured"
  | "job.detected"
  | "job.filtered"
  | "job.accepted"
  | "job.rejected"
  | "rss.fetched"
  | "rss.poll_started"
  | "rss.poll_ok"
  | "watcher.health"
  | "websocket.connected"
  | "websocket.message"
  | "websocket.pong"
  | "websocket.disconnected"
  | "error"
  | string;

export interface RealtimeEvent {
  id: string;
  type: EventType;
  message: string;
  level?: string;
  source?: string;
  data?: Record<string, unknown>;
  timestamp: string;
}

function dedupeEvents(events: RealtimeEvent[], limit: number): RealtimeEvent[] {
  const seen = new Set<string>();
  const deduped: RealtimeEvent[] = [];

  for (const event of events) {
    if (seen.has(event.id)) {
      continue;
    }
    seen.add(event.id);
    deduped.push(event);
    if (deduped.length >= limit) {
      break;
    }
  }

  return deduped;
}

// Session stats for summary cards
interface RealtimeSessionStats {
  jobsDetected: number;
  jobsAccepted: number;
  jobsFiltered: number;
  lastJobDetected: string | null;
}

interface RealtimeStoreState {
  events: RealtimeEvent[];
  maxEvents: number; // Maximum events to keep in rolling window

  // Session stats for summary cards
  stats: RealtimeSessionStats;

  // Actions
  addEvent: (type: EventType, message: string, data?: Record<string, unknown>) => void;
  setEvents: (events: RealtimeEvent[]) => void;
  clearEvents: () => void;
  incrementStat: (stat: keyof RealtimeSessionStats) => void;
  resetStats: () => void;
}

export const useRealtimeStore = create<RealtimeStoreState>()((set, get) => ({
  events: [],
  maxEvents: 50, // Keep last 50 events
  stats: {
    jobsDetected: 0,
    jobsAccepted: 0,
    jobsFiltered: 0,
    lastJobDetected: null,
  },

  addEvent: (type, message, data) =>
    set((state) => {
      const newEvent: RealtimeEvent = {
        id:
          typeof crypto !== "undefined" && "randomUUID" in crypto
            ? crypto.randomUUID()
            : `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
        type,
        message,
        data,
        timestamp: new Date().toISOString(),
      };
      // Add to front, dedupe, then trim to maxEvents.
      const newEvents = dedupeEvents([newEvent, ...state.events], state.maxEvents);

      // Auto-update stats based on event type
      const newStats = { ...state.stats };
      if (type === "job.detected") {
        newStats.jobsDetected += 1;
        newStats.lastJobDetected = newEvent.timestamp;
      } else if (type === "job.accepted" || type === "action.accept_succeeded") {
        newStats.jobsAccepted += 1;
      } else if (type === "job.filtered") {
        newStats.jobsFiltered += 1;
      }

      return { events: newEvents, stats: newStats };
    }),

  setEvents: (events) =>
    set((state) => {
      const nextEvents = dedupeEvents(events, state.maxEvents);
      const nextStats = nextEvents.reduce<RealtimeSessionStats>(
        (stats, event) => {
          if (event.type === "job.detected") {
            stats.jobsDetected += 1;
            stats.lastJobDetected ??= event.timestamp;
          } else if (
            event.type === "job.accepted" ||
            event.type === "action.accept_succeeded"
          ) {
            stats.jobsAccepted += 1;
          } else if (event.type === "job.filtered") {
            stats.jobsFiltered += 1;
          }
          return stats;
        },
        {
          jobsDetected: 0,
          jobsAccepted: 0,
          jobsFiltered: 0,
          lastJobDetected: null,
        },
      );

      return {
        events: nextEvents,
        stats: nextStats,
      };
    }),

  clearEvents: () => set({ events: [] }),

  incrementStat: (stat) =>
    set((state) => ({
      stats: {
        ...state.stats,
        [stat]: stat === "lastJobDetected" ? new Date().toISOString() : (state.stats[stat] as number) + 1,
      },
    })),

  resetStats: () =>
    set({
      stats: {
        jobsDetected: 0,
        jobsAccepted: 0,
        jobsFiltered: 0,
        lastJobDetected: null,
      },
    }),
}));
