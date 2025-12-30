/**
 * Watcher Store - Zustand state management for watcher status and configuration
 */

import { create } from "zustand";
import type { WatcherConfig, WatcherState } from "@/lib/api";
import { watcherApi } from "@/lib/api";
import { toast } from "./toast";

interface WatcherStoreState {
  // Configuration
  config: WatcherConfig | null;
  configLoading: boolean;
  configError: string | null;

  // Runtime state
  state: WatcherState | null;
  stateLoading: boolean;
  stateError: string | null;

  // Actions
  fetchConfig: () => Promise<void>;
  updateConfig: (data: Partial<WatcherConfig>) => Promise<void>;
  fetchState: () => Promise<void>;
  startWatcher: () => Promise<void>;
  stopWatcher: () => Promise<void>;
  setState: (state: WatcherState | null) => void;
  clear: () => void;
}

export const useWatcherStore = create<WatcherStoreState>()((set, get) => ({
  // Initial state
  config: null,
  configLoading: false,
  configError: null,

  state: null,
  stateLoading: false,
  stateError: null,

  // Fetch watcher configuration
  fetchConfig: async () => {
    set({ configLoading: true, configError: null });
    try {
      const config = await watcherApi.getConfig();
      set({ config, configLoading: false });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to fetch config";
      set({ configError: message, configLoading: false });
      toast.error(message);
    }
  },

  // Update watcher configuration
  updateConfig: async (data) => {
    set({ configLoading: true, configError: null });
    try {
      const config = await watcherApi.updateConfig(data);
      set({ config, configLoading: false });
      toast.success("Configuration saved");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to update config";
      set({ configError: message, configLoading: false });
      toast.error(message);
      throw error;
    }
  },

  // Fetch watcher runtime state
  fetchState: async () => {
    set({ stateLoading: true, stateError: null });
    try {
      const state = await watcherApi.getState();
      set({ state, stateLoading: false });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to fetch state";
      set({ stateError: message, stateLoading: false });
      toast.error(message);
    }
  },

  // Start the watcher
  startWatcher: async () => {
    set({ stateLoading: true, stateError: null });
    try {
      await watcherApi.start();
      // Refetch state after starting
      const state = await watcherApi.getState();
      set({ state, stateLoading: false });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to start watcher";
      set({ stateError: message, stateLoading: false });
      throw error;
    }
  },

  // Stop the watcher
  stopWatcher: async () => {
    set({ stateLoading: true, stateError: null });
    try {
      await watcherApi.stop();
      // Refetch state after stopping
      const state = await watcherApi.getState();
      set({ state, stateLoading: false });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to stop watcher";
      set({ stateError: message, stateLoading: false });
      throw error;
    }
  },

  // Direct state setter (for WebSocket updates)
  setState: (state) => set({ state }),

  // Clear all state
  clear: () =>
    set({
      config: null,
      configError: null,
      configLoading: false,
      state: null,
      stateError: null,
      stateLoading: false,
    }),
}));
