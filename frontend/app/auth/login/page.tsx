/**
 * Login Page - Data Factory Design
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import { authApi, oauthApi, ApiErrorClass } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const setUser = useAuthStore((state) => state.setUser);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setIsLoading(true);

    try {
      const response = await authApi.login({ email, password });

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

  const handleOAuthLogin = (provider: "google" | "github") => {
    oauthApi.authorize(provider);
  };

  return (
    <main className="min-h-screen bg-neutral-50 flex items-center justify-center p-6">
      <div className="w-full max-w-md">
        <div className="bento-card p-12">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-5xl font-light tracking-tighter mb-2">
              Sign In
            </h1>
            <p className="text-neutral-500 font-mono text-xs uppercase tracking-widest">
              Welcome back
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
                autoComplete="current-password"
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
              {isLoading ? "Signing in..." : "Sign In"}
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
              onClick={() => handleOAuthLogin("google")}
              className="w-full py-3 bg-white border border-neutral-200 text-sm text-neutral-600 transition-colors duration-150 hover:border-blue-500"
            >
              Continue with Google
            </button>
            <button
              type="button"
              onClick={() => handleOAuthLogin("github")}
              className="w-full py-3 bg-white border border-neutral-200 text-sm text-neutral-600 transition-colors duration-150 hover:border-blue-500"
            >
              Continue with GitHub
            </button>
          </div>

          {/* Footer */}
          <div className="mt-8 pt-8 border-t border-neutral-200 flex items-center justify-between">
            <span className="font-mono text-xs text-neutral-500 uppercase tracking-widest">
              Don't have an account?
            </span>
            <Link
              href="/auth/register"
              className="text-sm font-medium transition-colors duration-150 hover:text-blue-600"
            >
              Create account
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
