/**
 * ProfileSection - Email display and password change
 *
 * Enhanced with Data Factory base components and consistent styling.
 */

"use client";

import { useState } from "react";
import { authApi } from "@/lib/api";
import type { User } from "@/lib/api";
import { toast } from "@/store/toast";
import { Button } from "@/components/ui/base/Button";
import { cn } from "@/lib/utils";

interface ProfileSectionProps {
  user: User | null;
  isLoading: boolean;
  setIsLoading: (loading: boolean) => void;
}

export function ProfileSection({
  user,
  isLoading,
  setIsLoading,
}: ProfileSectionProps) {
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

  const inputClass = cn(
    "w-full px-4 py-3 bg-white border border-neutral-200 text-sm",
    "transition-colors duration-150",
    "focus:outline-none focus:border-blue-600",
    "disabled:opacity-50 disabled:cursor-not-allowed"
  );

  return (
    <div className="bento-card p-8">
      {/* Email */}
      <div className="mb-8">
        <label className="block font-mono text-xs uppercase tracking-widest text-neutral-600 mb-2 font-medium">
          Email
        </label>
        <p className="text-sm py-3 px-4 bg-neutral-50 border border-neutral-200 font-mono text-neutral-900">
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
                className={inputClass}
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
                className={inputClass}
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
                className={inputClass}
                required
                autoComplete="new-password"
                minLength={8}
              />
            </div>

            {passwordError && (
              <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-sm rounded-sm">
                {passwordError}
              </div>
            )}

            {passwordSuccess && (
              <div className="p-3 bg-green-50 border border-green-200 text-green-700 text-sm rounded-sm">
                Password updated successfully
              </div>
            )}

            <div className="flex gap-3">
              <Button
                type="submit"
                variant="primary"
                disabled={isLoading}
                loading={isLoading}
                loadingText="Updating..."
              >
                Update Password
              </Button>
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  setShowPasswordForm(false);
                  setPasswordError("");
                  setPasswordSuccess(false);
                  setOldPassword("");
                  setNewPassword("");
                  setConfirmPassword("");
                }}
              >
                Cancel
              </Button>
            </div>
          </form>
        ) : (
          <div className="flex items-center justify-between py-3 border-t border-neutral-200">
            <div>
              <p className="font-medium text-sm text-neutral-900">Password</p>
              <p className="text-xs text-neutral-600 mt-1">Last changed recently</p>
            </div>
            <Button
              onClick={() => setShowPasswordForm(true)}
              variant="secondary"
            >
              Change Password
            </Button>
          </div>
        )
      ) : (
        <div className="py-3 border-t border-neutral-200">
          <p className="text-sm text-neutral-700 mb-2">
            You signed in with {user?.provider === "google" ? "Google" : "GitHub"}. Set a password to enable email/password sign-in.
          </p>
          <Button
            variant="secondary"
            disabled
            title="Coming soon"
          >
            Set Password (Coming Soon)
          </Button>
        </div>
      )}
    </div>
  );
}
