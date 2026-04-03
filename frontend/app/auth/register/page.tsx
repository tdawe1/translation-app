/**
 * Register Page - Data Factory Design
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import { authApi, ApiErrorClass } from "@/lib/api";
import { setToken } from "@/lib/auth/tokens";
import { AuthForm } from "@/components/auth/AuthForm";

export default function RegisterPage() {
  const router = useRouter();
  const setUser = useAuthStore((state) => state.setUser);
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (data: { email: string; password: string }) => {
    setError("");
    setIsLoading(true);

    try {
      const response = await authApi.register({ email: data.email, password: data.password });

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

          {/* Auth Form */}
          <AuthForm
            mode="register"
            onSubmit={handleSubmit}
            errorMessage={error}
            isLoading={isLoading}
          />

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
