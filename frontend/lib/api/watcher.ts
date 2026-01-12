/**
 * Watcher API endpoints
 */

import { client } from "./client";
import type { WatcherConfig, WatcherState } from "./types";

export const watcherApi = {
  getConfig: (): Promise<WatcherConfig> =>
    client.get<WatcherConfig>("/api/v1/watcher/config"),

  updateConfig: (data: Partial<WatcherConfig>): Promise<WatcherConfig> =>
    client.put<WatcherConfig>("/api/v1/watcher/config", data),

  getState: (): Promise<WatcherState> =>
    client.get<WatcherState>("/api/v1/watcher/state"),

  start: (): Promise<{ status: string }> =>
    client.post<{ status: string }>("/api/v1/watcher/start"),

  stop: (): Promise<{ status: string }> =>
    client.post<{ status: string }>("/api/v1/watcher/stop"),
};
