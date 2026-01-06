/**
 * Auth Provider - Handles authentication state initialization
 */

"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/store/auth";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const fetchUser = useAuthStore((state) => state.fetchUser);
  const isLoading = useAuthStore((state) => state.isLoading);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  useEffect(() => {
    // Check if user is already logged in via httpOnly cookie
    // We try to fetch the user even without a sessionStorage token
    // because OAuth uses httpOnly cookies for security
    const checkAuth = async () => {
      // Only fetch if we don't already have a user from persisted storage
      if (!isAuthenticated) {
        await fetchUser();
      }
    };

    checkAuth();
  }, [fetchUser, isAuthenticated]);

  // Don't render children until we've checked auth
  // This prevents flash of unauthenticated content
  if (isLoading) {
    return null;
  }

  return <>{children}</>;
}
