# Technical Debt Remediation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce technical debt score from 720 to ~500, eliminate critical security issues, and improve development velocity by ~35%

**Architecture:** Incremental refactoring with TDD approach - each change tested before committing, backward compatibility maintained, feature flags for user-facing changes

**Tech Stack:** Go 1.25, Next.js 16, React 19, Zustand, TypeScript, Fiber v2, PostgreSQL, Redis

**Estimated Effort:** 40-50 hours across 4 sprints

---

## Sprint 1: Critical Security & Quick Wins (Week 1)

### Task 1: Fix Docker Secrets Exposure

**Priority:** CRITICAL (Security)
**Effort:** 30 minutes

**Files:**
- Modify: `docker-compose.yml:9,22`
- Create: `.env` (if not exists)

**Why:** Hardcoded passwords in docker-compose.yml are a security risk if repository is public.

**Step 1: Create .env file for docker-compose**

```bash
# Create .env in project root
cat > .env << 'EOF'
# Database Credentials
POSTGRES_PASSWORD=change-me-in-production-32chars

# Redis Password
REDIS_PASSWORD=change-me-in-production-32chars
EOF
```

**Step 2: Update docker-compose.yml to use env vars**

Edit `docker-compose.yml` line 9:

```yaml
# BEFORE:
POSTGRES_PASSWORD: devpass

# AFTER:
POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-devpass}
```

Edit `docker-compose.yml` line 22:

```yaml
# BEFORE:
command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD:-devpass}

# AFTER (no change needed, already using env var):
# command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD:-devpass}
```

**Step 3: Update .gitignore to ensure .env is not committed**

```bash
# Ensure .env is in .gitignore
echo ".env" >> .gitignore
git status  # Verify .env is not tracked
```

**Step 4: Test docker-compose still works**

```bash
# Stop any running containers
./scripts/dev.sh down

# Start with new env vars
./scripts/dev.sh up

# Verify containers start
./scripts/dev.sh status
```

Expected: All services start successfully

**Step 5: Commit**

```bash
git add docker-compose.yml .gitignore .env
git commit -m "fix(security): use environment variables for docker secrets"
```

---

### Task 2: Centralize Token Management

**Priority:** HIGH (Velocity)
**Effort:** 4 hours

**Files:**
- Create: `frontend/lib/auth/tokens.ts`
- Create: `frontend/lib/auth/tokens.test.ts`
- Modify: `frontend/lib/api.ts:137-140,172,194`
- Modify: `frontend/app/auth/login/page.tsx:31`
- Modify: `frontend/app/auth/register/page.tsx:45`
- Modify: `frontend/app/dashboard/page.tsx`
- Modify: `frontend/app/settings/page.tsx`
- Modify: `frontend/hooks/use-watcher-websocket.ts`

**Why:** Session storage access is scattered across 7+ files. Centralizing makes security improvements easier and reduces maintenance burden.

**Step 1: Write failing test for token service**

Create `frontend/lib/auth/tokens.test.ts`:

```typescript
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { getToken, setToken, clearToken, getTokenPayload } from './tokens';

describe('TokenService', () => {
  beforeEach(() => {
    // Clear sessionStorage before each test
    sessionStorage.clear();
  });

  afterEach(() => {
    sessionStorage.clear();
  });

  describe('getToken', () => {
    it('should return null when no token exists', () => {
      const token = getToken();
      expect(token).toBeNull();
    });

    it('should return token when it exists', () => {
      sessionStorage.setItem('access_token', 'test-token-123');
      const token = getToken();
      expect(token).toBe('test-token-123');
    });

    it('should return null for empty string', () => {
      sessionStorage.setItem('access_token', '');
      const token = getToken();
      expect(token).toBeNull();
    });
  });

  describe('setToken', () => {
    it('should store token in sessionStorage', () => {
      setToken('my-token');
      expect(sessionStorage.getItem('access_token')).toBe('my-token');
    });

    it('should clear existing token when setting new one', () => {
      setToken('old-token');
      setToken('new-token');
      expect(sessionStorage.getItem('access_token')).toBe('new-token');
    });

    it('should not store null or undefined', () => {
      setToken(null as any);
      expect(sessionStorage.getItem('access_token')).toBeNull();
    });
  });

  describe('clearToken', () => {
    it('should remove token from sessionStorage', () => {
      sessionStorage.setItem('access_token', 'test-token');
      clearToken();
      expect(sessionStorage.getItem('access_token')).toBeNull();
    });

    it('should be safe to call when no token exists', () => {
      expect(() => clearToken()).not.toThrow();
    });
  });

  describe('getTokenPayload', () => {
    it('should decode JWT payload', () => {
      // Format: header.payload.signature
      const mockToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjEyMyIsImVtYWlsIjoidGVzdEBleGFtcGxlLmNvbSJ9.signature';
      setToken(mockToken);
      const payload = getTokenPayload();
      expect(payload).toEqual({ id: '123', email: 'test@example.com' });
    });

    it('should return null for invalid token', () => {
      setToken('invalid-token');
      const payload = getTokenPayload();
      expect(payload).toBeNull();
    });

    it('should return null when no token exists', () => {
      const payload = getTokenPayload();
      expect(payload).toBeNull();
    });
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd frontend
npm test tokens.test.ts
```

Expected: FAIL with "Cannot find module './tokens'"

**Step 3: Implement token service**

Create `frontend/lib/auth/tokens.ts`:

```typescript
/**
 * Token Management Service
 *
 * Centralized access to sessionStorage for auth tokens.
 * All token operations MUST go through this service.
 *
 * @example
 * import { getToken, setToken, clearToken } from '@/lib/auth/tokens';
 */

const TOKEN_KEY = 'access_token';

/**
 * JWT payload interface
 */
export interface TokenPayload {
  id: string;
  email: string;
  exp?: number;
  iat?: number;
}

/**
 * Get the current access token from sessionStorage
 * @returns Token string or null if not exists/empty
 */
export function getToken(): string | null {
  if (typeof sessionStorage === 'undefined') {
    return null;
  }
  const token = sessionStorage.getItem(TOKEN_KEY);
  // Treat empty string as no token
  return token && token.trim() !== '' ? token : null;
}

/**
 * Store access token in sessionStorage
 * @param token - Token to store (null/undefined/empty clears it)
 */
export function setToken(token: string | null): void {
  if (typeof sessionStorage === 'undefined') {
    return;
  }
  if (token && token.trim() !== '') {
    sessionStorage.setItem(TOKEN_KEY, token);
  } else {
    sessionStorage.removeItem(TOKEN_KEY);
  }
}

/**
 * Clear the access token from sessionStorage
 */
export function clearToken(): void {
  if (typeof sessionStorage === 'undefined') {
    return;
  }
  sessionStorage.removeItem(TOKEN_KEY);
}

/**
 * Decode JWT payload without verifying signature
 * WARNING: This does NOT verify the signature. Only use for display purposes.
 * @returns Decoded payload or null if invalid
 */
export function getTokenPayload(): TokenPayload | null {
  const token = getToken();
  if (!token) {
    return null;
  }

  try {
    // JWT format: header.payload.signature
    const parts = token.split('.');
    if (parts.length !== 3) {
      return null;
    }

    // Decode base64url payload
    const payload = parts[1];
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const decoded = atob(base64);
    return JSON.parse(decoded) as TokenPayload;
  } catch {
    return null;
  }
}

/**
 * Check if token exists and is not expired
 * @returns true if valid token exists
 */
export function hasValidToken(): boolean {
  const payload = getTokenPayload();
  if (!payload || !payload.exp) {
    return false;
  }
  // Check expiration (with 30s buffer)
  return payload.exp > Math.floor(Date.now() / 1000) + 30;
}
```

**Step 4: Run test to verify it passes**

```bash
cd frontend
npm test tokens.test.ts
```

Expected: PASS

**Step 5: Update HttpClient to use token service**

Edit `frontend/lib/api.ts`:

Add import at line ~4:
```typescript
import { getToken } from "@/lib/auth/tokens";
```

Replace `sessionStorage.getItem("access_token")` at lines 137-140 with:
```typescript
// OLD:
private hasToken(): boolean {
  return typeof sessionStorage !== "undefined" &&
         !!sessionStorage.getItem("access_token");
}
```

With:
```typescript
// NEW:
private hasToken(): boolean {
  return getToken() !== null;
}
```

Replace line 172:
```typescript
// OLD:
const token = sessionStorage.getItem("access_token");
```

With:
```typescript
// NEW:
const token = getToken();
```

Replace line 194:
```typescript
// OLD:
sessionStorage.removeItem("access_token");
```

With:
```typescript
// NEW: Use token service (will be handled by auth store)
```

**Step 6: Run API tests**

```bash
cd frontend
npm test api.test.ts
```

Expected: PASS (or fix any failing tests)

**Step 7: Update login page**

Edit `frontend/app/auth/login/page.tsx`:

Add import at line ~11:
```typescript
import { authApi, oauthApi, ApiErrorClass } from "@/lib/api";
import { setToken } from "@/lib/auth/tokens";
```

Replace line 31:
```typescript
// OLD:
sessionStorage.setItem("access_token", response.access_token);
```

With:
```typescript
// NEW:
setToken(response.access_token);
```

**Step 8: Update register page**

Edit `frontend/app/auth/register/page.tsx`:

Add import at line ~11:
```typescript
import { authApi, oauthApi, ApiErrorClass } from "@/lib/api";
import { setToken } from "@/lib/auth/tokens";
```

Replace line 45:
```typescript
// OLD:
sessionStorage.setItem("access_token", response.access_token);
```

With:
```typescript
// NEW:
setToken(response.access_token);
```

**Step 9: Update auth store**

Edit `frontend/store/auth.ts`:

Add import at line ~7:
```typescript
import { authApi } from "@/lib/api";
import { getToken, clearToken as clearTokenStorage } from "@/lib/auth/tokens";
```

Add `clearToken` action to interface at line ~21:
```typescript
clear: () => void;
clearToken: () => void;  // NEW
```

Update `clear` action at line ~44:
```typescript
clear: () =>
  set({
    user: null,
    isAuthenticated: false,
    error: null,
  }),

// NEW:
clearToken: () => {
  clearTokenStorage();
  set({ user: null, isAuthenticated: false, error: null });
},
```

**Step 10: Run frontend tests**

```bash
cd frontend
npm test
```

Expected: All tests pass

**Step 11: Commit**

```bash
git add frontend/lib/auth/tokens.ts frontend/lib/auth/tokens.test.ts
git add frontend/lib/api.ts frontend/app/auth/login/page.tsx
git add frontend/app/auth/register/page.tsx frontend/store/auth.ts
git commit -m "refactor(auth): centralize token management

- Add TokenService for all sessionStorage access
- Update login/register pages to use TokenService
- Add comprehensive tests for token operations

Reduces duplication from 7+ locations to single service.
Easier to add token refresh and security improvements."
```

---

### Task 3: Extract Shared Auth Form Component

**Priority:** HIGH (Velocity)
**Effort:** 3 hours

**Files:**
- Create: `frontend/components/auth/AuthForm.tsx`
- Create: `frontend/components/auth/AuthForm.test.tsx`
- Modify: `frontend/app/auth/login/page.tsx`
- Modify: `frontend/app/auth/register/page.tsx`

**Why:** Login and register pages share ~100 lines of duplicate code. Extracting reduces maintenance burden.

**Step 1: Write failing test for AuthForm component**

Create `frontend/components/auth/AuthForm.test.tsx`:

```typescript
import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthForm } from './AuthForm';

describe('AuthForm', () => {
  const mockOnSubmit = vi.fn();

  beforeEach(() => {
    mockOnSubmit.mockClear();
  });

  describe('login mode', () => {
    it('should render email and password fields', () => {
      render(<AuthForm mode="login" onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    });

    it('should not show confirm password field in login mode', () => {
      render(<AuthForm mode="login" onSubmit={mockOnSubmit} />);

      expect(screen.queryByLabelText(/confirm password/i)).not.toBeInTheDocument();
    });

    it('should submit email and password', async () => {
      const user = userEvent.setup();
      render(<AuthForm mode="login" onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith({
          email: 'test@example.com',
          password: 'password123',
        });
      });
    });
  });

  describe('register mode', () => {
    it('should show confirm password field', () => {
      render(<AuthForm mode="register" onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/confirm password/i)).toBeInTheDocument();
    });

    it('should validate password match', async () => {
      const user = userEvent.setup();
      render(<AuthForm mode="register" onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.type(screen.getByLabelText(/confirm password/i), 'different');
      await user.click(screen.getByRole('button', { name: /create account/i }));

      expect(screen.getByText(/passwords do not match/i)).toBeInTheDocument();
      expect(mockOnSubmit).not.toHaveBeenCalled();
    });

    it('should validate password length', async () => {
      const user = userEvent.setup();
      render(<AuthForm mode="register" onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/password/i), 'short');
      await user.type(screen.getByLabelText(/confirm password/i), 'short');
      await user.click(screen.getByRole('button', { name: /create account/i }));

      expect(screen.getByText(/at least 8 characters/i)).toBeInTheDocument();
      expect(mockOnSubmit).not.toHaveBeenCalled();
    });
  });

  describe('OAuth buttons', () => {
    it('should render OAuth buttons when provided', () => {
      const mockOAuth = vi.fn();
      render(
        <AuthForm
          mode="login"
          onSubmit={mockOnSubmit}
          onOAuthLogin={mockOAuth}
        />
      );

      expect(screen.getByRole('button', { name: /google/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /github/i })).toBeInTheDocument();
    });

    it('should not render OAuth buttons when not provided', () => {
      render(<AuthForm mode="login" onSubmit={mockOnSubmit} />);

      expect(screen.queryByRole('button', { name: /google/i })).not.toBeInTheDocument();
    });
  });

  describe('error handling', () => {
    it('should display error message', () => {
      render(
        <AuthForm
          mode="login"
          onSubmit={mockOnSubmit}
          errorMessage="Invalid credentials"
        />
      );

      expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument();
    });

    it('should clear error when user types', async () => {
      const user = userEvent.setup();
      render(
        <AuthForm
          mode="login"
          onSubmit={mockOnSubmit}
          errorMessage="Previous error"
        />
      );

      const error = screen.getByText(/previous error/i);
      expect(error).toBeInTheDocument();

      await user.type(screen.getByLabelText(/email/i), 'a');

      expect(error).not.toBeInTheDocument();
    });
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd frontend
npm test AuthForm.test.tsx
```

Expected: FAIL with "Cannot find module './AuthForm'"

**Step 3: Implement AuthForm component**

Create `frontend/components/auth/AuthForm.tsx`:

```typescript
/**
 * Shared Auth Form Component
 *
 * Used by both login and register pages to eliminate duplication.
 * Handles email/password input, validation, and OAuth buttons.
 */

"use client";

import { useState, useCallback } from "react";
import Link from "next/link";

export interface AuthFormProps {
  mode: "login" | "register";
  onSubmit: (data: { email: string; password: string; confirmPassword?: string }) => Promise<void>;
  onOAuthLogin?: (provider: "google" | "github") => Promise<void>;
  errorMessage?: string | null;
  isLoading?: boolean;
}

export function AuthForm({ mode, onSubmit, onOAuthLogin, errorMessage, isLoading = false }: AuthFormProps) {
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
        setError("Password must be at least 8 characters");
        return;
      }
      if (password !== confirmPassword) {
        setError("Passwords do not match");
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
        setError("An unexpected error occurred");
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  // Clear error when user starts typing
  const clearError = useCallback(() => setError(null), []);

  const isLogin = mode === "login";
  const submitText = isLogin ? "Sign In" : "Create Account";
  const loadingText = isLogin ? "Signing in..." : "Creating account...";

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* Email Input */}
      <div>
        <label
          htmlFor="email"
          className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-2"
        >
          Email
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
          className="w-full px-4 py-3 bg-white border border-neutral-200 text-sm transition-colors duration-150 focus:outline-none focus:border-blue-500"
          required
          autoComplete={isLogin ? "email" : "email"}
          disabled={isLoading || isSubmitting}
        />
      </div>

      {/* Password Input */}
      <div>
        <label
          htmlFor="password"
          className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-2"
        >
          Password
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
          className="w-full px-4 py-3 bg-white border border-neutral-200 text-sm transition-colors duration-150 focus:outline-none focus:border-blue-500"
          required
          autoComplete={isLogin ? "current-password" : "new-password"}
          minLength={8}
          disabled={isLoading || isSubmitting}
        />
      </div>

      {/* Confirm Password (Register only) */}
      {!isLogin && (
        <div>
          <label
            htmlFor="confirm-password"
            className="block font-mono text-xs uppercase tracking-widest text-neutral-500 mb-2"
          >
            Confirm Password
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
            className="w-full px-4 py-3 bg-white border border-neutral-200 text-sm transition-colors duration-150 focus:outline-none focus:border-blue-500"
            required
            autoComplete="new-password"
            minLength={8}
            disabled={isLoading || isSubmitting}
          />
        </div>
      )}

      {/* Error Display */}
      {error && (
        <div className="p-3 bg-red-50 border border-red-100 text-red-700 text-sm">
          {error}
        </div>
      )}

      {/* Submit Button */}
      <button
        type="submit"
        name="submit"
        disabled={isLoading || isSubmitting}
        className="w-full py-3 bg-neutral-900 text-white text-sm transition-colors duration-150 hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {isLoading || isSubmitting ? loadingText : submitText}
      </button>

      {/* Magic Link Link (Login only) */}
      {isLogin && (
        <div className="text-center">
          <Link
            href="/auth/magic-link"
            className="text-sm font-mono text-neutral-500 uppercase tracking-widest transition-colors duration-150 hover:text-blue-600"
          >
            Send magic link instead
          </Link>
        </div>
      )}
    </form>
  );
}

// OAuth Button Component (sub-component)
export interface OAuthButtonsProps {
  onOAuthLogin: (provider: "google" | "github") => Promise<void>;
  disabled?: boolean;
}

export function OAuthButtons({ onOAuthLogin, disabled = false }: OAuthButtonsProps) {
  const [error, setError] = useState<string | null>(null);

  const handleOAuth = async (provider: "google" | "github") => {
    setError(null);
    try {
      await onOAuthLogin(provider);
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("Failed to connect to OAuth provider");
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
        <div className="p-3 bg-red-50 border border-red-100 text-red-700 text-sm mb-3">
          {error}
        </div>
      )}

      {/* OAuth Buttons */}
      <div className="space-y-3">
        <button
          type="button"
          onClick={() => handleOAuth("google")}
          disabled={disabled}
          className="w-full py-3 bg-white border border-neutral-200 text-sm text-neutral-600 transition-colors duration-150 hover:border-blue-500 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
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
          Continue with Google
        </button>
        <button
          type="button"
          onClick={() => handleOAuth("github")}
          disabled={disabled}
          className="w-full py-3 bg-white border border-neutral-200 text-sm text-neutral-600 transition-colors duration-150 hover:border-blue-500 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
        >
          <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
          </svg>
          Continue with GitHub
        </button>
      </div>
    </>
  );
}

// Footer Component
export interface AuthFormFooterProps {
  mode: "login" | "register";
  alternateLink?: string;
  alternateText?: string;
  alternateLinkText?: string;
}

export function AuthFormFooter({ mode, alternateLink = "/", alternateText, alternateLinkText }: AuthFormFooterProps) {
  const isLogin = mode === "login";

  return (
    <div className="mt-8 pt-8 border-t border-neutral-200 flex items-center justify-between">
      <span className="font-mono text-xs text-neutral-500 uppercase tracking-widest">
        {isLogin ? "Don't have an account?" : "Already have an account?"}
      </span>
      <Link
        href={isLogin ? "/auth/register" : "/auth/login"}
        className="text-sm font-medium transition-colors duration-150 hover:text-blue-600"
      >
        {isLogin ? "Create account" : "Sign in"}
      </Link>
    </div>
  );
}
```

**Step 4: Run test to verify it passes**

```bash
cd frontend
npm test AuthForm.test.tsx
```

Expected: PASS

**Step 5: Refactor login page to use AuthForm**

Replace entire content of `frontend/app/auth/login/page.tsx` with:

```typescript
/**
 * Login Page - Data Factory Design
 */

"use client";

import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuthStore } from "@/store/auth";
import { authApi, oauthApi, ApiErrorClass } from "@/lib/api";
import { setToken } from "@/lib/auth/tokens";
import { AuthForm, OAuthButtons, AuthFormFooter } from "@/components/auth/AuthForm";

export default function LoginPage() {
  const router = useRouter();
  const setUser = useAuthStore((state) => state.setUser);

  const handleSubmit = async ({ email, password }: { email: string; password: string }) => {
    const response = await authApi.login({ email, password });
    setToken(response.access_token);
    setUser(response.user);
    router.push("/dashboard");
  };

  const handleOAuthLogin = async (provider: "google" | "github") => {
    const response = await oauthApi.authorize(provider);
    window.location.href = response.auth_url;
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
          <AuthForm mode="login" onSubmit={handleSubmit} />

          {/* OAuth Options */}
          <OAuthButtons onOAuthLogin={handleOAuthLogin} />

          {/* Footer */}
          <AuthFormFooter mode="login" />
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
```

**Step 6: Refactor register page to use AuthForm**

Replace entire content of `frontend/app/auth/register/page.tsx` with:

```typescript
/**
 * Register Page - Data Factory Design
 */

"use client";

import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { authApi, oauthApi, ApiErrorClass } from "@/lib/api";
import { setToken } from "@/lib/auth/tokens";
import { AuthForm, OAuthButtons, AuthFormFooter } from "@/components/auth/AuthForm";

export default function RegisterPage() {
  const router = useRouter();
  const setUser = useAuthStore((state) => state.setUser);

  const handleSubmit = async ({ email, password }: { email: string; password: string }) => {
    const response = await authApi.register({ email, password });
    setToken(response.access_token);
    setUser(response.user);
    router.push("/dashboard");
  };

  const handleOAuthRegister = async (provider: "google" | "github") => {
    const response = await oauthApi.authorize(provider);
    window.location.href = response.auth_url;
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
          <AuthForm mode="register" onSubmit={handleSubmit} />

          {/* OAuth Options */}
          <OAuthButtons onOAuthLogin={handleOAuthRegister} />

          {/* Terms notice */}
          <p className="mt-6 text-xs text-neutral-500 text-center">
            By creating an account, you agree to our Terms of Service and Privacy
            Policy.
          </p>

          {/* Footer */}
          <AuthFormFooter mode="register" />
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
```

**Step 7: Run frontend tests**

```bash
cd frontend
npm test
```

Expected: All tests pass

**Step 8: Manual smoke test**

```bash
cd frontend
npm run dev

# In browser:
# 1. Visit http://localhost:3001/auth/login
# 2. Verify form renders correctly
# 3. Visit http://localhost:3001/auth/register
# 4. Verify form renders correctly
# 5. Try OAuth buttons (should redirect)
```

**Step 9: Commit**

```bash
git add frontend/components/auth/ frontend/app/auth/login/page.tsx
git add frontend/app/auth/register/page.tsx frontend/components/auth/AuthForm.test.tsx
git commit -m "refactor(auth): extract shared AuthForm component

- Create reusable AuthForm component for login/register
- Extract OAuthButtons and AuthFormFooter sub-components
- Add comprehensive tests for all components
- Reduce code duplication by ~100 lines

Pages now use shared components, making styling changes
apply to both pages automatically."
```

---

## Sprint 2: Complete OAuth & Backend Cleanup (Week 2)

### Task 4: Implement GetLinkedAccounts OAuth Endpoint

**Priority:** MEDIUM (Feature Completeness)
**Effort:** 2 hours

**Files:**
- Modify: `backend/internal/handlers/oauth.go:346-349`
- Modify: `backend/internal/oauth/service.go`
- Create: `backend/tests/oauth_linked_accounts_test.go`

**Why:** TODO comment exists, feature stub only. Users expect to see/manage their linked OAuth accounts.

**Step 1: Write failing test**

Create `backend/tests/oauth_linked_accounts_test.go`:

```go
package tests

import (
    "bytes"
    "encoding/json"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGetLinkedAccounts_Success(t *testing.T) {
    db := RequireDB(t)
    require.NotNil(t, db)

    // Create test user with OAuth account
    user := CreateTestUser(t, db, "oauth-test@example.com")

    // Setup test app and handler
    app := setupTestApp(db, nil)

    // Create session cookie
    token := createTestToken(t, user.ID.String())

    req := httptest.NewRequest("GET", "/api/v1/oauth/accounts", nil)
    req.Header.Set("Content-Type", "application/json")
    req.AddCookie(&http.Cookie{
        Name:  "session_token",
        Value: token,
    })

    resp, err := app.Test(req)
    require.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)

    var data map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&data)
    require.NoError(t, err)

    assert.Equal(t, "linked_accounts", data["linked_accounts"])
    accounts, ok := data["linked_accounts"].([]interface{})
    assert.True(t, ok)
    assert.Empty(t, accounts) // No accounts linked yet
}

func TestGetLinkedAccounts_Unauthorized(t *testing.T) {
    db := RequireDB(t)
    app := setupTestApp(db, nil)

    req := httptest.NewRequest("GET", "/api/v1/oauth/accounts", nil)
    req.Header.Set("Content-Type", "application/json")

    resp, err := app.Test(req)
    require.NoError(t, err)
    assert.Equal(t, 401, resp.StatusCode)
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend
go test -run TestGetLinkedAccounts ./tests/
```

Expected: FAIL (endpoint not implemented yet)

**Step 3: Implement GetLinkedAccounts handler**

Edit `backend/internal/handlers/oauth.go` at line ~337:

Replace the stub implementation with:

```go
// GetLinkedAccounts returns the user's linked OAuth accounts
func (h *OAuthHandler) GetLinkedAccounts(c *fiber.Ctx) error {
    userID := c.Locals("user_id")
    if userID == nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "unauthorized",
            "code":  "UNAUTHORIZED",
        })
    }

    // Fetch user with OAuth accounts
    var user models.User
    err := h.db.Preload("OAuthAccounts").Where("id = ?", userID).First(&user).Error
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to fetch accounts",
            "code":  "DATABASE_ERROR",
        })
    }

    // Build response
    linkedAccounts := make([]fiber.Map, len(user.OAuthAccounts))
    for i, oa := range user.OAuthAccounts {
        linkedAccounts[i] = fiber.Map{
            "provider":  oa.Provider,
            "created_at": oa.CreatedAt.Format(time.RFC3339),
        }
    }

    return c.JSON(fiber.Map{
        "linked_accounts": linkedAccounts,
    })
}
```

**Step 4: Run test to verify it passes**

```bash
cd backend
go test -run TestGetLinkedAccounts ./tests/
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/handlers/oauth.go backend/tests/oauth_linked_accounts_test.go
git commit -m "feat(oauth): implement GetLinkedAccounts endpoint

- Fetch user's linked OAuth accounts from database
- Return provider and creation date for each account
- Add comprehensive test coverage

Resolves TODO at line 346."
```

---

### Task 5: Implement UnlinkAccount OAuth Endpoint

**Priority:** MEDIUM (Feature Completeness)
**Effort:** 2 hours

**Files:**
- Modify: `backend/internal/handlers/oauth.go:370-372`
- Modify: `backend/internal/oauth/service.go`
- Create: `backend/tests/oauth_unlink_test.go`

**Step 1: Write failing test**

Create `backend/tests/oauth_unlink_test.go`:

```go
package tests

import (
    "bytes"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUnlinkAccount_Success(t *testing.T) {
    db := RequireDB(t)
    require.NotNil(t, db)

    // Create test user with OAuth account
    user := CreateTestUser(t, db, "unlink-test@example.com")

    // Setup test app and handler
    app := setupTestApp(db, nil)

    // Create session cookie
    token := createTestToken(t, user.ID.String())

    req := httptest.NewRequest("DELETE", "/api/v1/oauth/github", nil)
    req.Header.Set("Content-Type", "application/json")
    req.AddCookie(&http.Cookie{
        Name:  "session_token",
        Value: token,
    })

    resp, err := app.Test(req)
    require.NoError(t, err)

    // Should return 204 No Content on success
    // or 404 if no account to unlink (expected behavior)
    assert.Contains(t, []int{204, 404}, resp.StatusCode)
}

func TestUnlinkAccount_Unauthorized(t *testing.T) {
    db := RequireDB(t)
    app := setupTestApp(db, nil)

    req := httptest.NewRequest("DELETE", "/api/v1/oauth/github", nil)
    req.Header.Set("Content-Type", "application/json")

    resp, err := app.Test(req)
    require.NoError(t, err)
    assert.Equal(t, 401, resp.StatusCode)
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend
go test -run TestUnlinkAccount ./tests/
```

Expected: FAIL (stub implementation)

**Step 3: Implement UnlinkAccount handler**

Edit `backend/internal/handlers/oauth.go` at line ~353:

Replace the stub implementation with:

```go
// UnlinkAccount unlinks an OAuth account
func (h *OAuthHandler) UnlinkAccount(c *fiber.Ctx) error {
    provider := c.Params("provider")
    if !ValidateProvider(provider) {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "invalid provider",
            "code":  "INVALID_PROVIDER",
        })
    }

    userID := c.Locals("user_id")
    if userID == nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "unauthorized",
            "code":  "UNAUTHORIZED",
        })
    }

    // Delete the OAuth account
    result := h.db.Where("user_id = ? AND provider = ?", userID, provider).Delete(&models.OAuthAccount{})

    // Return 204 regardless of whether account existed
    // This prevents account enumeration
    return c.SendStatus(fiber.StatusNoContent)
}
```

**Step 4: Run test to verify it passes**

```bash
cd backend
go test -run TestUnlinkAccount ./tests/
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/handlers/oauth.go backend/tests/oauth_unlink_test.go
git commit -m "feat(oauth): implement UnlinkAccount endpoint

- Delete OAuth account by provider for authenticated user
- Return 204 regardless of existence (prevents enumeration)
- Add comprehensive test coverage

Resolves TODO at line 370."
```

---

### Task 6: Complete Database Shim Migration

**Priority:** HIGH (Architecture)
**Effort:** 3 hours

**Files:**
- Modify: `backend/internal/models/database.go`
- Modify: `backend/cmd/server/main.go`
- Audit: All files importing `models.DB`

**Why:** Global DB state is deprecated. Completing migration to DI improves testability.

**Step 1: Find all usages of models.DB**

```bash
cd backend
grep -r "models\.DB" --include="*.go" . > /tmp/models_db_usage.txt
cat /tmp/models_db_usage.txt
```

**Step 2: For each file found, update to use dependency injection**

Pattern to fix:
```go
// OLD:
import "github.com/tdawe1/translation-app/internal/models"
models.DB.Where(...)

// NEW:
// Add db parameter to handler/struct
// Use injected db instead
```

**Step 3: Mark models/database.go for deletion**

Edit `backend/internal/models/database.go`:

Add comment at top:
```go
// DEPRECATED: This file will be removed in v1.0
// All code should use dependency injection via database.Database interface
// See: backend/internal/database/database.go
```

**Step 4: Update main.go if needed**

Check `backend/cmd/server/main.go` for any InitDB() calls and remove.

**Step 5: Run all tests**

```bash
cd backend
make test
```

Expected: All tests pass

**Step 6: Commit**

```bash
git add backend/internal/models/database.go backend/cmd/server/main.go
git commit -m "refactor(db): complete migration from global DB to DI

- Mark models/database.go as deprecated
- Update all consumers to use dependency injection
- Remove InitDB calls from main.go

This improves testability and removes implicit dependencies."
```

---

## Sprint 3: Email Handler Refactor (Week 3)

### Task 7: Split Email Handler

**Priority:** MEDIUM (Code Quality)
**Effort:** 4 hours

**Files:**
- Create: `backend/internal/handlers/email/verification.go`
- Create: `backend/internal/handlers/email/magic_link.go`
- Create: `backend/internal/handlers/email/password_reset.go`
- Create: `backend/internal/service/token_service.go`
- Modify: `backend/internal/handlers/email.go` (deprecate)

**Why:** Email handler is 504 lines with 6 responsibilities. Splitting improves maintainability.

**Step 1: Create token service**

Create `backend/internal/service/token_service.go`:

```go
package service

import (
    "crypto/rand"
    "encoding/base64"
    "time"

    "gorm.io/gorm"
)

// TokenService handles token generation and validation
type TokenService struct {
    db *gorm.DB
}

func NewTokenService(db *gorm.DB) *TokenService {
    return &TokenService{db: db}
}

// GenerateSecureToken generates a cryptographically random token
func (s *TokenService) GenerateSecureToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}

// DeleteExistingTokens deletes unused tokens for a given email
func (s *TokenService) DeleteExistingTokens(email string, tokenModel interface{}) error {
    return s.db.Where("email = ? AND used_at IS NULL", email).Delete(tokenModel).Error
}
```

**Step 2: Split verification handler**

Create `backend/internal/handlers/email/verification.go`:

```go
package email

import (...)
// Extract verification-related logic from email.go
```

**Step 3: Run tests**

```bash
cd backend
go test ./internal/handlers/email/...
```

**Step 4: Commit**

```bash
git add backend/internal/handlers/email/ backend/internal/service/token_service.go
git commit -m "refactor(email): split EmailHandler into focused handlers

- Create TokenService for shared token operations
- Separate VerificationHandler, MagicLinkHandler, PasswordResetHandler
- Each handler is now <200 lines and single-responsibility

Reduces file from 504 to ~3 files of ~150 lines each."
```

---

## Sprint 4: Navigation & Logging (Week 4)

### Task 8: Fix SPA Navigation Pattern

**Priority:** MEDIUM (UX)
**Effort:** 2 hours

**Files:**
- Create: `frontend/lib/navigation.ts`
- Modify: 5 files using `window.location.href`

**Why:** Using `window.location.href` breaks SPA behavior and loses state.

**Step 1: Create navigation utility**

Create `frontend/lib/navigation.ts`:

```typescript
/**
 * Navigation utilities for SPA behavior
 *
 * Preserves React state and avoids full page reloads when possible.
 * OAuth redirects are an exception (must use window.location).
 */

import { useRouter } from "next/navigation";

/**
 * Navigate to OAuth provider (requires full page load)
 * Use this ONLY for OAuth redirects
 */
export function navigateToOAuth(url: string): void {
  window.location.href = url;
}

/**
 * Navigate to auth page when not authenticated
 * Preserves as much state as possible
 */
export function navigateToLogin(returnUrl?: string): void {
  const loginUrl = returnUrl
    ? `/auth/login?return=${encodeURIComponent(returnUrl)}`
    : "/auth/login";
  window.location.href = loginUrl; // Acceptable: auth flow needs redirect
}
```

**Step 2: Update OAuth calls**

**Step 3: Commit**

---

## Success Metrics

After completing this plan:

| Metric | Before | Target |
|--------|--------|--------|
| Debt Score | 720 | 500 |
| Code Duplication | 12% | 5% |
| TODO Comments | 2 | 0 |
| sessionStorage locations | 7 | 1 |
| Largest file | 543 lines | <300 |
| Auth form duplication | 100 lines | 0 |

---

**Plan complete and saved to `docs/plans/2025-01-12-technical-debt-remediation.md`.**

**Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
