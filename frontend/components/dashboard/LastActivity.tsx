"use client";

import { useEffect, useState } from "react";
import { useRealtimeStore, type RealtimeEvent } from "@/store/realtime";

interface LastActivityProps {
  lastActivity?: string;
  connected?: boolean;
  lastMessageTime?: number | null;
}

function formatDateTime(value?: string | number | Date | null) {
  if (!value) return "Waiting for activity";
  return new Date(value).toLocaleString();
}

function formatTime(value?: string | number | Date | null) {
  if (!value) return "--:--:--";
  return new Date(value).toLocaleTimeString();
}

function formatEventLine(event: RealtimeEvent) {
  const source = event.source ? `${event.source} ` : "";
  return `${source}${event.type}: ${event.message}`;
}

export function LastActivity({
  lastActivity,
  connected = false,
  lastMessageTime = null,
}: LastActivityProps) {
  const [now, setNow] = useState(() => new Date());
  const events = useRealtimeStore((state) => state.events);
  const recentEvents = events.slice(0, 5);
  const latestEvent = recentEvents[0];

  useEffect(() => {
    const interval = window.setInterval(() => setNow(new Date()), 1_000);
    return () => window.clearInterval(interval);
  }, []);

  return (
    <div className="border border-neutral-200 bg-white">
      <div className="grid grid-cols-1 border-b border-neutral-200 md:grid-cols-3">
        <div className="border-b border-neutral-200 p-4 md:border-b-0 md:border-r">
          <p className="font-mono text-[11px] uppercase tracking-widest text-blue-600">
            Live Clock
          </p>
          <p
            className="mt-2 font-mono text-2xl tracking-tight text-neutral-900"
            data-testid="live-clock"
          >
            {formatTime(now)}
          </p>
          <p className="mt-1 text-sm text-neutral-500">
            {connected ? "Realtime connected" : "Realtime disconnected"}
          </p>
        </div>

        <div className="border-b border-neutral-200 p-4 md:border-b-0 md:border-r">
          <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
            Last State Activity
          </p>
          <p className="mt-2 text-base text-neutral-900">
            {formatDateTime(lastActivity)}
          </p>
          <p className="mt-1 text-sm text-neutral-500">
            WebSocket message: {formatTime(lastMessageTime)}
          </p>
        </div>

        <div className="p-4">
          <p className="font-mono text-[11px] uppercase tracking-widest text-neutral-500">
            Latest Output
          </p>
          <p className="mt-2 line-clamp-2 text-base text-neutral-900">
            {latestEvent ? latestEvent.message : "No watcher output received yet"}
          </p>
          <p className="mt-1 font-mono text-[11px] uppercase tracking-widest text-neutral-400">
            {latestEvent ? latestEvent.type : "idle"}
          </p>
        </div>
      </div>

      <div
        className="border-t border-neutral-200 bg-neutral-100 p-4 font-mono text-xs text-neutral-800"
        data-testid="overview-terminal-output"
      >
        <div className="mb-3 flex items-center justify-between gap-4">
          <p className="uppercase tracking-widest text-neutral-500">
            Recent terminal output
          </p>
          <span
            className={`h-2 w-2 rounded-full ${
              connected ? "bg-green-500" : "bg-neutral-400"
            }`}
            aria-hidden="true"
          />
        </div>

        {recentEvents.length > 0 ? (
          <ol className="space-y-2" aria-live="polite">
            {recentEvents.map((event) => (
              <li key={event.id} className="grid gap-1 sm:grid-cols-[90px_1fr]">
                <time className="text-neutral-500" dateTime={event.timestamp}>
                  {formatTime(event.timestamp)}
                </time>
                <span className="break-words text-neutral-800">
                  {formatEventLine(event)}
                </span>
              </li>
            ))}
          </ol>
        ) : (
          <p className="py-4 text-center italic text-neutral-500">
            No events yet. Start the watcher or wait for the next heartbeat.
          </p>
        )}
      </div>
    </div>
  );
}
