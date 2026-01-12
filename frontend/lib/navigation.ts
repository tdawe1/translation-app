/**
 * Navigation utilities for SPA behavior
 *
 * Preserves React state and avoids full page reloads when possible.
 * OAuth redirects are an exception (must use window.location).
 */

import { useRouter } from "next/navigation";

/**
 * Navigate to OAuth provider (requires full page load)
 * Use this ONLY for OAuth redirects to external providers like Google/GitHub
 */
export function navigateToOAuth(url: string): void {
  window.location.href = url;
}

/**
 * Navigate to auth page when not authenticated
 * Preserves as much state as possible with return URL
 */
export function navigateToLogin(returnUrl?: string): void {
  const loginUrl = returnUrl
    ? `/auth/login?return=${encodeURIComponent(returnUrl)}`
    : "/auth/login";
  window.location.href = loginUrl; // Acceptable: auth flow needs redirect
}

/**
 * Navigate to home page
 * Use this for post-logout navigation
 */
export function navigateToHome(): void {
  window.location.href = "/";
}

/**
 * Navigate with router (preserves SPA state)
 * Use this for internal navigation when authenticated
 */
export function useAppRouter() {
  const router = useRouter();

  return {
    /**
     * Navigate to a path while preserving SPA state
     */
    push: (href: string) => router.push(href),

    /**
     * Replace current route without adding to history
     */
    replace: (href: string) => router.replace(href),

    /**
     * Go back in history
     */
    back: () => router.back(),

    /**
     * Refresh current route
     */
    refresh: () => router.refresh(),
  };
}
