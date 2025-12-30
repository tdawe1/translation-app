/**
 * Toast Store - Global notification system
 *
 * Features:
 * - Auto-dismiss after 5 seconds
 * - Multiple toasts stack vertically
 * - Support for error, success, and info types
 */

import { create } from "zustand";

export type ToastType = "error" | "success" | "info";

export interface Toast {
  id: string;
  type: ToastType;
  message: string;
}

interface ToastStore {
  toasts: Toast[];
  addToast: (type: ToastType, message: string) => void;
  removeToast: (id: string) => void;
  clearToasts: () => void;
}

export const useToastStore = create<ToastStore>()((set, get) => ({
    toasts: [],

    addToast: (type, message) => {
      const id = Math.random().toString(36).substring(7);
      const toast: Toast = { id, type, message };

      set((state) => ({ toasts: [...state.toasts, toast] }));

      // Auto-dismiss after 5 seconds
      setTimeout(() => {
        get().removeToast(id);
      }, 5000);
    },

    removeToast: (id) =>
      set((state) => ({
        toasts: state.toasts.filter((t) => t.id !== id),
      })),

    clearToasts: () => set({ toasts: [] }),
  })
);

// Convenience functions
export const toast = {
  error: (message: string) => useToastStore.getState().addToast("error", message),
  success: (message: string) => useToastStore.getState().addToast("success", message),
  info: (message: string) => useToastStore.getState().addToast("info", message),
};
