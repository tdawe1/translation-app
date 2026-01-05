/**
 * Magic Link Request Page - Data Factory Design
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ApiErrorClass } from "@/lib/api";

export default function MagicLinkPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setSuccess(false);
    setIsLoading(true);

    try {
      const response = await fetch("/api/v1/auth/magic-link", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || "Failed to send magic link");
      }

      setSuccess(true);
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("An unexpected error occurred");
      }
    } finally {
      setIsLoading(false);
    }
  };

  if (success) {
    return (
      <main className="min-h-screen bg-neutral-50 flex items-center justify-center p-6">
        <div className="w-full max-w-md">
          <div className="bento-card p-12 text-center">
            {/* Success Icon */}
            <div className="mb-6 flex justify-center">
              <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center">
                <svg
                  className="w-8 h-8 text-green-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M5 13l4 4L19 7"
                  />
                </svg>
              </div>
            </div>

            <h1 className="text-3xl font-light tracking-tighter mb-4">
              Check your email
            </h1>
            <p className="text-neutral-600 mb-8">
              We sent a magic link to <strong>{email}</strong>. Click the link in
              the email to sign in. The link expires in 15 minutes.
            </p>

            <div className="space-y-4">
              <button
                onClick={() => setSuccess(false)}
                className="w-full py-3 bg-neutral-900 text-white text-sm transition-colors duration-150 hover:bg-blue-600"
              >
                Try another email
              </button>

              <Link
                href="/auth/login"
                className="block w-full py-3 border border-neutral-200 text-sm text-center transition-colors duration-150 hover:border-blue-500"
              >
                Back to sign in
              </Link>
            </div>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-neutral-50 flex items-center justify-center p-6">
      <div className="w-full max-w-md">
        <div className="bento-card p-12">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-5xl font-light tracking-tighter mb-2">
              Magic Link
            </h1>
            <p className="text-neutral-500 font-mono text-xs uppercase tracking-widest">
              Passwordless sign in
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

            {error && (
              <div className="p-3 bg-red-50 border border-red-100 text-red-700 text-sm">
                {error}
              </div>
            )}

            {success && (
              <div className="p-3 bg-green-50 border border-green-100 text-green-700 text-sm">
                If an account exists, a magic link has been sent.
              </div>
            )}

            <button
              type="submit"
              disabled={isLoading}
              className="w-full py-3 bg-neutral-900 text-white text-sm transition-colors duration-150 hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? "Sending..." : "Send Magic Link"}
            </button>
          </form>

          {/* Footer */}
          <div className="mt-8 pt-8 border-t border-neutral-200 flex items-center justify-between">
            <span className="font-mono text-xs text-neutral-500 uppercase tracking-widest">
              Remember your password?
            </span>
            <Link
              href="/auth/login"
              className="text-sm font-medium transition-colors duration-150 hover:text-blue-600"
            >
              Sign in with password
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
