/**
 * Shared Auth Form Component
 *
 * Used by both login and register pages to eliminate duplication.
 * Handles email/password input, validation, and OAuth buttons.
 * Follows Data Factory design language.
 */

"use client";

import { useState, useCallback } from "react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/base/Button";
import { cn } from "@/lib/utils";

export interface AuthFormProps {
  mode: "login" | "register";
  onSubmit: (data: { email: string; password: string; confirmPassword?: string }) => Promise<void>;
  onOAuthLogin?: (provider: "google" | "github") => Promise<void>;
  errorMessage?: string | null;
  isLoading?: boolean;
}

const inputClass = cn(
  "w-full px-4 py-3 bg-white border border-neutral-200 text-sm",
  "transition-colors duration-150",
  "focus:outline-none focus:border-blue-600",
  "disabled:opacity-50 disabled:cursor-not-allowed"
);

export function AuthForm({ mode, onSubmit, onOAuthLogin, errorMessage, isLoading = false }: AuthFormProps) {
  const t = useTranslations('auth');
  const tCommon = useTranslations('common');
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(errorMessage ?? null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validation for register mode
    if (mode === "register") {
      if (password.length < 8) {
        setError(t('passwordTooShort'));
        return;
      }
      if (password !== confirmPassword) {
        setError(t('passwordMismatch'));
        return;
      }
    }

    setIsSubmitting(true);
    try {
      await onSubmit({ email, password, ...(mode === "register" && { confirmPassword }) });
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError(tCommon('api.error'));
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  // Clear error when user starts typing
  const clearError = useCallback(() => setError(null), []);

  const isLogin = mode === "login";
  const submitText = isLogin ? t('signIn') : t('signUp');
  const loadingText = isLogin ? t('signingIn') : t('creatingAccount');
  const disabled = isLoading || isSubmitting;

  return (
    <>
      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Email Input */}
        <div>
          <label
            htmlFor="email"
            className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-2"
          >
            {t('email')}
          </label>
          <input
            id="email"
            name="email"
            type="email"
            placeholder="you@example.com"
            value={email}
            onChange={(e) => {
              setEmail(e.target.value);
              clearError();
            }}
            className={inputClass}
            required
            autoComplete={isLogin ? "email" : "email"}
            disabled={disabled}
            suppressHydrationWarning
          />
        </div>

        {/* Password Input */}
        <div>
          <label
            htmlFor="password"
            className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-2"
          >
            {t('password')}
          </label>
          <input
            id="password"
            name="password"
            type="password"
            placeholder="•••••••••"
            value={password}
            onChange={(e) => {
              setPassword(e.target.value);
              clearError();
            }}
            className={inputClass}
            required
            autoComplete={isLogin ? "current-password" : "new-password"}
            disabled={disabled}
            suppressHydrationWarning
          />
        </div>

        {/* Confirm Password (Register only) */}
        {!isLogin && (
          <div>
            <label
              htmlFor="confirm-password"
              className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-2"
            >
              {tCommon('confirm')} {t('password')}
            </label>
            <input
              id="confirm-password"
              name="confirm-password"
              type="password"
              placeholder="•••••••••"
              value={confirmPassword}
              onChange={(e) => {
                setConfirmPassword(e.target.value);
                clearError();
              }}
              className={inputClass}
              required
              autoComplete="new-password"
              disabled={disabled}
              suppressHydrationWarning
            />
          </div>
        )}

        {/* Error Display */}
        {error && (
          <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-sm rounded-sm">
            {error}
          </div>
        )}

        {/* Submit Button */}
        <Button
          type="submit"
          variant="primary"
          fullWidth
          disabled={disabled}
          loading={disabled}
          loadingText={loadingText}
        >
          {submitText}
        </Button>

        {/* Magic Link Link (Login only) */}
        {isLogin && (
          <div className="text-center">
            <Link
              href="/auth/magic-link"
              className="text-sm font-mono text-neutral-500 uppercase tracking-widest transition-colors duration-150 hover:text-blue-600"
            >
              {t('magicLink')}
            </Link>
          </div>
        )}
      </form>

      {/* OAuth Buttons */}
      {onOAuthLogin && <OAuthButtons onOAuthLogin={onOAuthLogin} disabled={disabled} />}
    </>
  );
}

// OAuth Button Component (sub-component)
export interface OAuthButtonsProps {
  onOAuthLogin: (provider: "google" | "github") => Promise<void>;
  disabled?: boolean;
}

const oauthButtonClass = cn(
  "w-full py-3 bg-white border border-neutral-200 text-sm text-neutral-600",
  "transition-colors duration-150 hover:border-blue-600",
  "disabled:opacity-50 disabled:cursor-not-allowed",
  "flex items-center justify-center gap-2"
);

export function OAuthButtons({ onOAuthLogin, disabled = false }: OAuthButtonsProps) {
  const t = useTranslations('auth');
  const tCommon = useTranslations('common');
  const [error, setError] = useState<string | null>(null);

  const handleOAuth = async (provider: "google" | "github") => {
    setError(null);
    try {
      await onOAuthLogin(provider);
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError(tCommon('api.error'));
      }
    }
  };

  return (
    <>
      {/* Divider */}
      <div className="flex items-center gap-4 my-8">
        <div className="flex-1 h-px bg-neutral-200" />
        <span className="font-mono text-xs text-neutral-400 uppercase tracking-widest">
          or
        </span>
        <div className="flex-1 h-px bg-neutral-200" />
      </div>

      {/* Error Display */}
      {error && (
        <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-sm mb-3 rounded-sm">
          {error}
        </div>
      )}

       {/* OAuth Buttons */}
      <div className="space-y-3">
        <button
          type="button"
          onClick={() => handleOAuth("google")}
          disabled={disabled}
          className={oauthButtonClass}
        >
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
          {t('signInWithGoogle')}
        </button>
        <button
          type="button"
          onClick={() => handleOAuth("github")}
          disabled={disabled}
          className={oauthButtonClass}
        >
          <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
          </svg>
          {t('signInWithGitHub')}
        </button>
      </div>
    </>
  );
}
