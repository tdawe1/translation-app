/**
 * Auth Provider - Handles authentication state initialization
 */

"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/store/auth";
import { authApi } from "@/lib/api";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const setUser = useAuthStore((state) => state.setUser);
  const setLoading = useAuthStore((state) => state.setLoading);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  useEffect(() => {
    // Check if user is already logged in
    const checkAuth = async () => {
      const token = sessionStorage.getItem("access_token");

      if (!token) {
        setLoading(false);
        return;
      }

      try {
        const user = await authApi.me();
        setUser(user);
      } catch {
        // Token might be expired, clear it
        sessionStorage.removeItem("access_token");
        setUser(null);
      } finally {
        setLoading(false);
      }
    };

    checkAuth();
  }, [setUser, setLoading]);

  // Don't render children until we've checked auth
  // This prevents flash of unauthenticated content
  return <>{children}</>;
}
