/**
 * Login Page - Data Factory Design
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import { authApi, oauthApi, ApiErrorClass } from "@/lib/api";
import { setToken } from "@/lib/auth/tokens";
import { AuthForm } from "@/components/auth/AuthForm";

export default function LoginPage() {
  const router = useRouter();
  const setUser = useAuthStore((state) => state.setUser);
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (data: { email: string; password: string }) => {
    setError("");
    setIsLoading(true);

    try {
      const response = await authApi.login({ email: data.email, password: data.password });

      // Store access token using TokenService (httpOnly cookie is also set by backend)
      setToken(response.access_token);
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

  const handleOAuthLogin = async (provider: "google" | "github") => {
    setError("");
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
              Sign In
            </h1>
            <p className="text-neutral-500 font-mono text-xs uppercase tracking-widest">
              Welcome back
            </p>
          </div>

          {/* Auth Form */}
          <AuthForm
            mode="login"
            onSubmit={handleSubmit}
            onOAuthLogin={handleOAuthLogin}
            errorMessage={error}
            isLoading={isLoading}
          />

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
