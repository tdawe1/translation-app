/**
 * Watcher API endpoints
 */

import { client } from "./client";
import type {
  BrowserStateSyncRequest,
  WatcherConfig,
  WatcherEventsResponse,
  WatcherState,
} from "./types";

export const watcherApi = {
  getConfig: (): Promise<WatcherConfig> =>
    client.get<WatcherConfig>("/api/v1/watcher/config"),

  updateConfig: (data: Partial<WatcherConfig>): Promise<WatcherConfig> =>
    client.put<WatcherConfig>("/api/v1/watcher/config", data),

  getState: (): Promise<WatcherState> =>
    client.get<WatcherState>("/api/v1/watcher/state"),

  getEvents: (): Promise<WatcherEventsResponse> =>
    client.get<WatcherEventsResponse>("/api/v1/watcher/events"),

  syncBrowserState: (
    data: BrowserStateSyncRequest,
  ): Promise<{ status: string }> =>
    client.post<{ status: string }>("/api/v1/watcher/browser-state", data),

  restartBrowser: (): Promise<{ status: string }> =>
    client.post<{ status: string }>("/api/v1/watcher/browser/restart"),

  captureBrowserScreenshot: (): Promise<{
    status: string;
    url?: string;
    title?: string;
    screenshot_artifact_id?: string;
    screenshot_url?: string;
  }> => client.post("/api/v1/watcher/browser/screenshot"),

  start: (): Promise<{ status: string }> =>
    client.post<{ status: string }>("/api/v1/watcher/start"),

  stop: (): Promise<{ status: string }> =>
    client.post<{ status: string }>("/api/v1/watcher/stop"),
};
