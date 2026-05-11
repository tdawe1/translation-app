/**
 * useWatcherWebSocket - Hook for real-time watcher updates via WebSocket
 *
 * Features:
 * - Automatic reconnection with exponential backoff
 * - Ticket-based authentication (short-lived UUID tickets)
 * - Handles job, event, and error messages
 * - Updates Zustand store with received data
 */

import { useEffect, useRef, useCallback, useState, useMemo } from "react";
import { useWatcherStore } from "@/store/watcher";
import { useJobsStore } from "@/store/jobs";
import { authApi } from "@/lib/api";
import type { WatcherState } from "@/lib/api";
import type { Job } from "@/store/jobs";

// WebSocket message types from backend
interface WSConnectedMessage {
  type: "connected";
  user_id: string;
  timestamp: string;
}

interface WSEventMessage {
  type: "event";
  event: string;
  data: unknown;
  timestamp: string;
}

interface WSErrorMessage {
  type: "error";
  message: string;
  timestamp: string;
}

interface WSJobDataMessage {
  type: "job";
  data: JobData;
  timestamp: string;
}

// Strict interface for job data from WebSocket
interface JobData {
  id: string;
  title: string;
  reward: number;
  url: string;
  source: string;
  currency?: string;
  timestamp?: string | number;
}

type WSMessage = WSConnectedMessage | WSEventMessage | WSErrorMessage | WSJobDataMessage;

function isWatcherStatePatch(data: unknown): data is Partial<WatcherState> {
  return typeof data === "object" && data !== null;
}

// Type guard for JobData
function isJobData(data: unknown): data is JobData {
  if (typeof data !== "object" || data === null) {
    return false;
  }
  const d = data as Record<string, unknown>;
  return (
    typeof d.id === "string" &&
    typeof d.title === "string" &&
    typeof d.reward === "number" &&
    typeof d.url === "string" &&
    typeof d.source === "string" &&
    d.source.trim().length > 0 &&
    (d.currency === undefined || typeof d.currency === "string") &&
    (d.timestamp === undefined || typeof d.timestamp === "string" || typeof d.timestamp === "number")
  );
}

function normalizeJobTimestamp(timestamp: JobData["timestamp"]): string {
  if (typeof timestamp === "number" && Number.isFinite(timestamp) && timestamp > 0) {
    return new Date(timestamp * 1000).toISOString();
  }
  if (typeof timestamp === "string" && timestamp.length > 0) {
    return timestamp;
  }
  return new Date().toISOString();
}

interface UseWatcherWebSocketOptions {
  enabled?: boolean;
  onJob?: (job: Job) => void;
  onEvent?: (event: string, data: unknown) => void;
  onError?: (error: string) => void;
}

export interface WebSocketMetrics {
  connected: boolean;
  reconnecting: boolean;
  reconnectCount: number;
  connectionStartTime: number | null;
  uptime: number; // seconds
  lastMessageTime: number | null;
  messagesReceived: number;
}

const RECONNECT_DELAYS = [1000, 2000, 4000, 8000, 16000]; // Exponential backoff
const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:37181/ws";

export function useWatcherWebSocket(options: UseWatcherWebSocketOptions = {}) {
  const { enabled = true, onJob, onEvent, onError } = options;
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | undefined>(undefined);
  const reconnectAttemptsRef = useRef(0);
  // Explicit connection state ref to avoid race condition with readyState checks
  // This is set synchronously in onopen/onclose, ensuring consistent state
  const isConnectedRef = useRef(false);

  // Connection metrics state
  const [connectionStartTime, setConnectionStartTime] = useState<number | null>(null);
  const [messagesReceived, setMessagesReceived] = useState(0);
  const [lastMessageTime, setLastMessageTime] = useState<number | null>(null);

  const setState = useWatcherStore((state) => state.setState);
  const addJob = useJobsStore((state) => state.addJob);

  // Calculate uptime when realtime messages arrive instead of forcing a page-wide tick.
  const uptime = useMemo(() => {
    if (!connectionStartTime) return 0;
    return Math.floor((Date.now() - connectionStartTime) / 1000);
  }, [connectionStartTime, lastMessageTime]);

  // Clean up connection
  const cleanup = useCallback(() => {
    if (reconnectTimeoutRef.current !== undefined) {
      window.clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = undefined;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  // Connect to WebSocket
  const connect = useCallback(async () => {
    if (!enabled) return;


    // Fetch a one-time-use ticket for WebSocket authentication
    let wsUrl = WS_URL;
    try {
      const ticketResp = await authApi.getWSTicket();
      wsUrl = `${WS_URL}?ticket=${ticketResp.ticket}`;
      console.log("[WS] Got ticket, connecting...");
    } catch (err) {
      // Only log error if all retries are exhausted (prevents console spam during init)
      const isFinalRetry = reconnectAttemptsRef.current >= RECONNECT_DELAYS.length - 1;
      if (isFinalRetry) {
        console.error("[WS] Failed to get WebSocket ticket after all retries:", err);
      }
      // Schedule retry with exponential backoff
      if (enabled && reconnectAttemptsRef.current < RECONNECT_DELAYS.length) {
        const delay = RECONNECT_DELAYS[reconnectAttemptsRef.current];
        reconnectTimeoutRef.current = window.setTimeout(() => {
          reconnectAttemptsRef.current++;
          void connect();
        }, delay);
      }
      return;
    }

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log("[WS] Connected");
        reconnectAttemptsRef.current = 0;
        isConnectedRef.current = true; // Set explicit state
        setConnectionStartTime(Date.now());
        setMessagesReceived(0);
        setLastMessageTime(null);
      };

      ws.onmessage = (event) => {
        // Track message metrics
        setMessagesReceived((prev) => prev + 1);
        setLastMessageTime(Date.now());

        try {
          const message = JSON.parse(event.data) as WSMessage;

          switch (message.type) {
            case "connected":
              console.log("[WS] Server confirmed connection for user:", message.user_id);
              break;

            case "event":
              console.log("[WS] Event:", message.event, message.data);
              onEvent?.(message.event, message.data);

              if (message.event === "watcher.health" && isWatcherStatePatch(message.data)) {
                const currentState = useWatcherStore.getState().state;
                if (currentState) {
                  setState({
                    ...currentState,
                    ...message.data,
                  });
                }
              }

              // Update store for known events
              if (message.event === "worker.started" || message.event === "worker.stopped") {
                // Trigger state refresh
                const currentState = useWatcherStore.getState().state;
                if (currentState) {
                  const nextWatcherStatus = message.event === "worker.started" ? "running" : "stopped";
                  setState({
                    ...currentState,
                    watcher_status: nextWatcherStatus,
                    overall_status:
                      message.event === "worker.started"
                        ? "running"
                        : currentState.overall_status,
                  });
                }
              }
              break;

            case "error":
              console.error("[WS] Error:", message.message);
              onError?.(message.message);
              break;

            case "job":
              console.log("[WS] New job:", message.data);
              // Use type guard to safely validate job data
              if (isJobData(message.data)) {
                const job: Job = {
                  id: message.data.id,
                  title: message.data.title,
                  reward: message.data.reward,
                  url: message.data.url,
                  source: message.data.source,
                  currency: message.data.currency,
                  timestamp: normalizeJobTimestamp(message.data.timestamp),
                };
                addJob(job);
                onJob?.(job);
              } else {
                console.warn("[WS] Invalid job data received:", message.data);
              }
              break;

            default:
              console.warn("[WS] Unknown message type:", message);
          }
        } catch (err) {
          console.error("[WS] Failed to parse message:", err);
        }
      };

      ws.onclose = (event) => {
        console.log("[WS] Connection closed:", event.code, event.reason);
        wsRef.current = null;
        isConnectedRef.current = false; // Clear explicit state
        // Clear connection metrics on disconnect
        setConnectionStartTime(null);

        // Attempt reconnection with exponential backoff
        if (enabled && reconnectAttemptsRef.current < RECONNECT_DELAYS.length) {
          const delay = RECONNECT_DELAYS[reconnectAttemptsRef.current];
          console.log(`[WS] Reconnecting in ${delay}ms... (attempt ${reconnectAttemptsRef.current + 1})`);

          reconnectTimeoutRef.current = window.setTimeout(() => {
            reconnectAttemptsRef.current++;
            connect();
          }, delay);
        }
      };

      ws.onerror = (error) => {
        // WebSocket error events are often followed by onclose, so we log minimally here
        // The onclose handler will provide more actionable information
        console.log("[WS] WebSocket error event received");
      };

    } catch (err) {
      console.error("[WS] Failed to create WebSocket:", err);
    }
  }, [enabled, onEvent, onError, onJob, setState, addJob]);

  // Disconnect function
  const disconnect = useCallback(() => {
    cleanup();
    reconnectAttemptsRef.current = 0;
  }, [cleanup]);

  // Set up connection on mount and cleanup on unmount
  useEffect(() => {
    if (enabled) {
      connect();
    }

    return () => {
      cleanup();
    };
  }, [enabled, connect, cleanup]);

  // Use explicit state ref instead of deriving from readyState to avoid race conditions
  // The ref is set synchronously in onopen/onclose handlers for consistent state
  const connected = isConnectedRef.current;
  const reconnecting = reconnectAttemptsRef.current > 0 && !connected;

  return {
    connected,
    reconnecting,
    reconnectCount: reconnectAttemptsRef.current,
    connectionStartTime,
    uptime,
    lastMessageTime,
    messagesReceived,
    connect,
    disconnect,
  };
}
