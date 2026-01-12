/**
 * Settings Page - Data Factory Design
 *
 * User profile and account management
 */

"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { ProtectedRoute } from "@/components/auth/protected-route";
import { ErrorBoundary } from "@/components/error-boundary";
import { authApi, type User } from "@/lib/api";
import Link from "next/link";
import { ProfileSection } from "./profile-section";
import { OAuthSection } from "./oauth-section";

export default function SettingsPage() {
  const router = useRouter();
  const { user, setUser, clear } = useAuthStore();
  const [isLoading, setIsLoading] = useState(false);

  // Fetch fresh user data to get OAuth accounts
  useEffect(() => {
    const fetchUser = async () => {
      const freshUser = await authApi.me();
      if (freshUser) {
        setUser(freshUser);
      }
      // If freshUser is null, user was redirected to login or session expired
      // The ProtectedRoute component will handle the redirect
    };
    fetchUser();
  }, [setUser]);

  const handleLogout = async () => {
    try {
      await authApi.logout();
    } catch (err) {
      // Continue with logout even if API call fails
    } finally {
      sessionStorage.removeItem("access_token");
      clear();
      router.push("/");
    }
  };

  return (
    <ProtectedRoute>
      <ErrorBoundary>
        <main id="main-content" className="min-h-screen bg-neutral-50">
          {/* Header */}
          <header className="bg-white border-b border-neutral-300">
            <div className="max-w-4xl mx-auto px-6 py-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <Link
                  href="/dashboard"
                  className="font-mono text-xs text-neutral-600 uppercase tracking-widest hover:text-neutral-900 font-medium"
                >
                  ← Back
                </Link>
                <h1 className="text-xl font-light tracking-tighter text-neutral-900">
                  Settings
                </h1>
              </div>
              <div className="flex items-center gap-4">
                <span className="font-mono text-xs text-neutral-600 uppercase tracking-widest font-medium">
                  {user?.email}
                </span>
              </div>
            </div>
          </header>

          {/* Settings Content */}
          <div className="max-w-4xl mx-auto px-6 py-12 space-y-8">
            {/* Profile Section */}
            <section aria-labelledby="profile-heading">
              <div className="mb-6">
                <h2 id="profile-heading" className="text-4xl font-light tracking-tighter mb-2 text-neutral-900">
                  Profile
                </h2>
                <p className="text-neutral-600 font-mono text-xs uppercase tracking-widest font-medium">
                  Manage your account
                </p>
              </div>

              <ProfileSection user={user} isLoading={isLoading} setIsLoading={setIsLoading} />
            </section>

            {/* Connected Accounts Section */}
            <section aria-labelledby="accounts-heading">
              <div className="mb-6">
                <h2 id="accounts-heading" className="text-4xl font-light tracking-tighter mb-2 text-neutral-900">
                  Connected Accounts
                </h2>
                <p className="text-neutral-600 font-mono text-xs uppercase tracking-widest font-medium">
                  Linked sign-in methods
                </p>
              </div>

              <OAuthSection user={user} />
            </section>

            {/* Danger Zone */}
            <section aria-labelledby="danger-heading">
              <div className="mb-6">
                <h2 id="danger-heading" className="text-red-600 font-mono text-xs uppercase tracking-widest font-semibold">
                  Danger Zone
                </h2>
              </div>

              <div className="bento-card p-6">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-lg font-medium mb-1 text-neutral-900">Sign Out</h3>
                    <p className="text-sm text-neutral-600">Sign out of your account on this device</p>
                  </div>
                  <button
                    onClick={handleLogout}
                    className="px-6 py-3 border border-neutral-300 text-neutral-900 text-sm transition-colors duration-150 hover:border-red-600 hover:text-red-600"
                  >
                    Sign Out
                  </button>
                </div>
              </div>
            </section>
          </div>
        </main>
      </ErrorBoundary>
    </ProtectedRoute>
  );
}
