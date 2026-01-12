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
import { authApi, oauthApi, type User } from "@/lib/api";
import { toast } from "@/store/toast";
import Link from "next/link";
import { navigateToOAuth } from "@/lib/navigation";

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

              <ConnectedAccounts user={user} />
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

/**
 * ProfileSection - Email display and password change
 */
function ProfileSection({
  user,
  isLoading,
  setIsLoading,
}: {
  user: User | null;
  isLoading: boolean;
  setIsLoading: (loading: boolean) => void;
}) {
  const [showPasswordForm, setShowPasswordForm] = useState(false);
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordError, setPasswordError] = useState("");
  const [passwordSuccess, setPasswordSuccess] = useState(false);

  const handleSubmitPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordError("");
    setPasswordSuccess(false);

    // Validation
    if (newPassword.length < 8) {
      setPasswordError("Password must be at least 8 characters");
      return;
    }
    if (newPassword !== confirmPassword) {
      setPasswordError("New passwords do not match");
      return;
    }

    setIsLoading(true);
    try {
      await authApi.changePassword({
        old_password: oldPassword,
        new_password: newPassword,
      });
      setPasswordSuccess(true);
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
      toast.success("Password updated successfully");
      setTimeout(() => setShowPasswordForm(false), 2000);
    } catch (err) {
      if (err instanceof Error) {
        setPasswordError(err.message);
      } else {
        setPasswordError("Failed to update password");
      }
    } finally {
      setIsLoading(false);
    }
  };

  const hasPassword = user?.provider === undefined || user?.provider === "";

  return (
    <div className="bento-card p-8">
      {/* Email */}
      <div className="mb-8">
        <label className="block font-mono text-xs uppercase tracking-widest text-neutral-600 mb-2 font-medium">
          Email
        </label>
        <p className="text-sm py-3 px-4 bg-neutral-50 border border-neutral-300 font-mono text-neutral-900">
          {user?.email}
        </p>
      </div>

      {/* Password Change */}
      {hasPassword ? (
        showPasswordForm ? (
          <form onSubmit={handleSubmitPassword} className="space-y-4">
            <div>
              <label
                htmlFor="old-password"
                className="block font-mono text-xs uppercase tracking-widest text-neutral-600 mb-2 font-medium"
              >
                Current Password
              </label>
              <input
                id="old-password"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                className="w-full px-4 py-3 bg-white border border-neutral-300 text-neutral-900 text-sm transition-colors duration-150 focus:outline-none focus:border-blue-500"
                required
                autoComplete="current-password"
              />
            </div>

            <div>
              <label
                htmlFor="new-password"
                className="block font-mono text-xs uppercase tracking-widest text-neutral-600 mb-2 font-medium"
              >
                New Password
              </label>
              <input
                id="new-password"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="w-full px-4 py-3 bg-white border border-neutral-300 text-neutral-900 text-sm transition-colors duration-150 focus:outline-none focus:border-blue-500"
                required
                autoComplete="new-password"
                minLength={8}
              />
            </div>

            <div>
              <label
                htmlFor="confirm-password"
                className="block font-mono text-xs uppercase tracking-widest text-neutral-600 mb-2 font-medium"
              >
                Confirm New Password
              </label>
              <input
                id="confirm-password"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="w-full px-4 py-3 bg-white border border-neutral-300 text-neutral-900 text-sm transition-colors duration-150 focus:outline-none focus:border-blue-500"
                required
                autoComplete="new-password"
                minLength={8}
              />
            </div>

            {passwordError && (
              <div className="p-3 bg-red-50 border border-red-200 text-red-800 text-sm">
                {passwordError}
              </div>
            )}

            {passwordSuccess && (
              <div className="p-3 bg-green-50 border border-green-200 text-green-800 text-sm">
                Password updated successfully
              </div>
            )}

            <div className="flex gap-3">
              <button
                type="submit"
                disabled={isLoading}
                className="px-6 py-3 bg-neutral-900 text-white text-sm transition-colors duration-150 hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isLoading ? "Updating..." : "Update Password"}
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowPasswordForm(false);
                  setPasswordError("");
                  setPasswordSuccess(false);
                  setOldPassword("");
                  setNewPassword("");
                  setConfirmPassword("");
                }}
                className="px-6 py-3 border border-neutral-300 text-neutral-900 text-sm transition-colors duration-150 hover:border-neutral-400"
              >
                Cancel
              </button>
            </div>
          </form>
        ) : (
          <div className="flex items-center justify-between py-3 border-t border-neutral-300">
            <div>
              <p className="font-medium text-sm text-neutral-900">Password</p>
              <p className="text-xs text-neutral-600 mt-1">Last changed recently</p>
            </div>
            <button
              onClick={() => setShowPasswordForm(true)}
              className="px-6 py-3 border border-neutral-300 text-neutral-900 text-sm transition-colors duration-150 hover:border-blue-500"
            >
              Change Password
            </button>
          </div>
        )
      ) : (
        <div className="py-3 border-t border-neutral-300">
          <p className="text-sm text-neutral-700 mb-2">
            You signed in with {user?.provider === "google" ? "Google" : "GitHub"}. Set a password to enable email/password sign-in.
          </p>
          <button
            className="px-6 py-3 border border-neutral-300 text-neutral-900 text-sm transition-colors duration-150 hover:border-blue-500 opacity-50 cursor-not-allowed"
            disabled
            title="Coming soon"
          >
            Set Password (Coming Soon)
          </button>
        </div>
      )}
    </div>
  );
}

/**
 * ConnectedAccounts - Display linked OAuth accounts (read-only)
 */
function ConnectedAccounts({ user }: { user: User | null }) {
  const [isLinking, setIsLinking] = useState(false);

  const handleLinkProvider = async (provider: "google" | "github") => {
    setIsLinking(true);
    try {
      const response = await oauthApi.authorize(provider);
      navigateToOAuth(response.auth_url); // Full page redirect to OAuth provider
    } catch {
      // Silently fail - user can try again
    }
  };

  // Get display name for provider
  const getProviderDisplayName = (provider: string): string => {
    switch (provider) {
      case "google":
        return "Google";
      case "github":
        return "GitHub";
      default:
        return provider;
    }
  };

  // Get provider color
  const getProviderColor = (provider: string): string => {
    switch (provider) {
      case "google":
        return "text-blue-600";
      case "github":
        return "text-purple-600";
      default:
        return "text-neutral-600";
    }
  };

  // Get provider icon
  const getProviderIcon = (provider: string) => {
    if (provider === "google") {
      return (
        <svg className="w-5 h-5" viewBox="0 0 24 24">
          <path
            fill="currentColor"
            d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
          />
          <path
            fill="currentColor"
            d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
          />
          <path
            fill="currentColor"
            d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
          />
          <path
            fill="currentColor"
            d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
          />
        </svg>
      );
    }
    if (provider === "github") {
      return (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
          <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
        </svg>
      );
    }
    return null;
  };

  const accounts = user?.oauth_accounts || [];
  const hasGoogle = accounts.some((a) => a.provider === "google") || user?.provider === "google";
  const hasGitHub = accounts.some((a) => a.provider === "github") || user?.provider === "github";

  return (
    <div className="bento-card p-8">
      {accounts.length === 0 && !user?.provider ? (
        <p className="text-sm text-neutral-600 py-3">
          No connected accounts. Sign in with Google or GitHub to link your account.
        </p>
      ) : (
        <div className="space-y-4">
          {/* Primary Provider */}
          {user?.provider && (
            <div className="flex items-center justify-between py-3 border-b border-neutral-300">
              <div className="flex items-center gap-3">
                {getProviderIcon(user.provider)}
                <div>
                  <p className={`text-sm font-medium ${getProviderColor(user.provider)}`}>
                    {getProviderDisplayName(user.provider)}
                  </p>
                  <p className="text-xs text-neutral-600">Primary sign-in method</p>
                </div>
              </div>
              <span className="px-3 py-1 bg-green-50 text-green-800 text-xs font-mono uppercase border border-green-200">
                Active
              </span>
            </div>
          )}

          {/* Additional OAuth Accounts */}
          {accounts.map((account) => (
            <div
              key={`${account.provider}-${account.created_at}`}
              className="flex items-center justify-between py-3 border-b border-neutral-300 last:border-0"
            >
              <div className="flex items-center gap-3">
                {getProviderIcon(account.provider)}
                <div>
                  <p className={`text-sm font-medium ${getProviderColor(account.provider)}`}>
                    {getProviderDisplayName(account.provider)}
                  </p>
                  <p className="text-xs text-neutral-600">
                    Linked {new Date(account.created_at).toLocaleDateString()}
                  </p>
                </div>
              </div>
              <span className="px-3 py-1 bg-neutral-100 text-neutral-700 text-xs font-mono uppercase border border-neutral-300">
                Linked
              </span>
            </div>
          ))}

          {/* Available Providers (not linked) */}
          {!hasGoogle && (
            <button
              type="button"
              onClick={() => handleLinkProvider("google")}
              disabled={isLinking}
              className="w-full flex items-center justify-between py-3 border-t border-neutral-300 group hover:bg-neutral-50 disabled:opacity-50"
            >
              <div className="flex items-center gap-3">
                {getProviderIcon("google")}
                <p className="text-sm text-neutral-700 group-hover:text-blue-600">
                  Link Google account
                </p>
              </div>
              <span className="text-neutral-500 group-hover:text-blue-600">
                {isLinking ? "..." : "→"}
              </span>
            </button>
          )}

          {!hasGitHub && (
            <button
              type="button"
              onClick={() => handleLinkProvider("github")}
              disabled={isLinking}
              className="w-full flex items-center justify-between py-3 border-t border-neutral-300 group hover:bg-neutral-50 disabled:opacity-50"
            >
              <div className="flex items-center gap-3">
                {getProviderIcon("github")}
                <p className="text-sm text-neutral-700 group-hover:text-purple-600">
                  Link GitHub account
                </p>
              </div>
              <span className="text-neutral-500 group-hover:text-purple-600">
                {isLinking ? "..." : "→"}
              </span>
            </button>
          )}
        </div>
      )}
    </div>
  );
}
