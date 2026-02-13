/**
 * Auth Store - Zustand state management for authentication
 */

import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { User } from "@/lib/api";
import { authApi } from "@/lib/api";

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

  // Actions
  setUser: (user: User | null) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  clear: () => void;
  clearToken: () => void;
  fetchUser: () => Promise<void>;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: true,
      error: null,

      setUser: (user) =>
        set({
          user,
          isAuthenticated: !!user,
          error: null,
          isLoading: false,
        }),

      setLoading: (isLoading) => set({ isLoading }),

      setError: (error) => set({ error }),

      clear: () =>
        set({
          user: null,
          isAuthenticated: false,
          error: null,
        }),

      clearToken: () => {
        set({ user: null, isAuthenticated: false, error: null });
      },

      fetchUser: async () => {
        try {
          const user = await authApi.me();
          if (user) {
            set({ user, isAuthenticated: true, isLoading: false, error: null });
          } else {
            // No user session (optional request returned null)
            set({ user: null, isAuthenticated: false, isLoading: false, error: null });
          }
        } catch (err) {
          // User is not logged in or session expired
          set({ user: null, isAuthenticated: false, isLoading: false, error: null });
        }
      },
    }),
    {
      name: "gengowatcher-auth",
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);
