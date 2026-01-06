/**
 * Register Page - Data Factory Design
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import { authApi, oauthApi, ApiErrorClass } from "@/lib/api";

export default function RegisterPage() {
  const router = useRouter();
  const setUser = useAuthStore((state) => state.setUser);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    // Validate passwords match
    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    // Validate password strength
    if (password.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }

    setIsLoading(true);

    try {
      const response = await authApi.register({ email, password });

      // Store access token in sessionStorage (httpOnly cookie is set by backend)
      sessionStorage.setItem("access_token", response.access_token);
      setUser(response.user);

      // Redirect to dashboard
      router.push("/dashboard");
    } catch (err) {
      if (err instanceof ApiErrorClass) {
        setError(err.message);
      } else {
        setError("An unexpected error occurred");
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleOAuthRegister = async (provider: "google" | "github") => {
    try {
      const response = await oauthApi.authorize(provider);
      // Redirect to OAuth provider's authorization page
      window.location.href = response.auth_url;
    } catch (err) {
      if (err instanceof ApiErrorClass) {
        setError(err.message);
      } else {
        setError("Failed to connect to OAuth provider");
      }
    }
  };

  return (
    <main className="min-h-screen bg-neutral-50 flex items-center justify-center p-6">
      <div className="w-full max-w-md">
        <div className="bento-card p-12">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-5xl font-light tracking-tighter mb-2">
              Create Account
            </h1>
            <p className="text-neutral-500 font-mono text-xs uppercase tracking-widest">
              Start monitoring jobs today
            </p>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label
                htmlFor="email"
                className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-2"
              >
                Email
              </label>
              <input
                id="email"
                type="email"
                placeholder="you@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full px-4 py-3 bg-white border border-neutral-200 text-sm transition-colors duration-150 focus:outline-none focus:border-blue-500"
                required
                autoComplete="email"
              />
            </div>

            <div>
              <label
                htmlFor="password"
                className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-2"
              >
                Password
              </label>
              <input
                id="password"
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-3 bg-white border border-neutral-200 text-sm transition-colors duration-150 focus:outline-none focus:border-blue-500"
                required
                autoComplete="new-password"
                minLength={8}
              />
            </div>

            <div>
              <label
                htmlFor="confirm-password"
                className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-2"
              >
                Confirm Password
              </label>
              <input
                id="confirm-password"
                type="password"
                placeholder="••••••••"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="w-full px-4 py-3 bg-white border border-neutral-200 text-sm transition-colors duration-150 focus:outline-none focus:border-blue-500"
                required
                autoComplete="new-password"
                minLength={8}
              />
            </div>

            {error && (
              <div className="p-3 bg-red-50 border border-red-100 text-red-700 text-sm">
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={isLoading}
              className="w-full py-3 bg-neutral-900 text-white text-sm transition-colors duration-150 hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? "Creating account..." : "Create Account"}
            </button>
          </form>

          {/* Divider */}
          <div className="flex items-center gap-4 my-8">
            <div className="flex-1 h-px bg-neutral-200" />
            <span className="font-mono text-xs text-neutral-400 uppercase tracking-widest">
              or
            </span>
            <div className="flex-1 h-px bg-neutral-200" />
          </div>

          {/* OAuth Options */}
          <div className="space-y-3">
            <button
              type="button"
              onClick={() => handleOAuthRegister("google")}
              className="w-full py-3 bg-white border border-neutral-200 text-sm text-neutral-600 transition-colors duration-150 hover:border-blue-500"
            >
              Continue with Google
            </button>
            <button
              type="button"
              onClick={() => handleOAuthRegister("github")}
              className="w-full py-3 bg-white border border-neutral-200 text-sm text-neutral-600 transition-colors duration-150 hover:border-blue-500"
            >
              Continue with GitHub
            </button>
          </div>

          {/* Terms notice */}
          <p className="mt-6 text-xs text-neutral-500 text-center">
            By creating an account, you agree to our Terms of Service and Privacy
            Policy.
          </p>

          {/* Footer */}
          <div className="mt-8 pt-8 border-t border-neutral-200 flex items-center justify-between">
            <span className="font-mono text-xs text-neutral-500 uppercase tracking-widest">
              Already have an account?
            </span>
            <Link
              href="/auth/login"
              className="text-sm font-medium transition-colors duration-150 hover:text-blue-600"
            >
              Sign in
            </Link>
          </div>
        </div>

        {/* Back to home */}
        <div className="text-center mt-6">
          <Link
            href="/"
            className="font-mono text-xs text-neutral-500 uppercase tracking-widest transition-colors duration-150 hover:text-neutral-900"
          >
            ← Back to home
          </Link>
        </div>
      </div>
    </main>
  );
}
