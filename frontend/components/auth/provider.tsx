/**
 * Auth Provider - Handles authentication state initialization
 */

"use client";

import { useEffect, useRef } from "react";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const hasChecked = useRef(false);

  useEffect(() => {
    // Only check once on mount to avoid infinite loops
    if (hasChecked.current) return;
    hasChecked.current = true;

    // Import dynamically to avoid dependency issues
    import("@/store/auth").then(({ useAuthStore }) => {
      const fetchUser = useAuthStore.getState().fetchUser;
      fetchUser();
    });
  }, []);

  return <>{children}</>;
}
