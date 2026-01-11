/**
 * Realtime Events Store - Zustand state management for live event feed
 *
 * Tracks watcher activity events in a rolling window for the realtime feed.
 */

import { create } from "zustand";

export type EventType =
  | "watcher.started"
  | "watcher.stopped"
  | "job.detected"
  | "job.filtered"
  | "job.accepted"
  | "job.rejected"
  | "rss.fetched"
  | "websocket.connected"
  | "websocket.disconnected"
  | "error";

export interface RealtimeEvent {
  id: string;
  type: EventType;
  message: string;
  data?: Record<string, unknown>;
  timestamp: string;
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
        id: `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
        type,
        message,
        data,
        timestamp: new Date().toISOString(),
      };
      // Add to front, trim to maxEvents
      const newEvents = [newEvent, ...state.events].slice(0, state.maxEvents);

      // Auto-update stats based on event type
      const newStats = { ...state.stats };
      if (type === "job.detected") {
        newStats.jobsDetected += 1;
        newStats.lastJobDetected = newEvent.timestamp;
      } else if (type === "job.accepted") {
        newStats.jobsAccepted += 1;
      } else if (type === "job.filtered") {
        newStats.jobsFiltered += 1;
      }

      return { events: newEvents, stats: newStats };
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
