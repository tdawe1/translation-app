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
  [key: string]: any;
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
  // Check expiration (with 30s buffer to account for clock skew)
  return payload.exp > Math.floor(Date.now() / 1000) + 30;
}
