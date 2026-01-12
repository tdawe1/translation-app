/**
 * API Client for GengoWatcher SaaS
 * Refactored with proper separation of concerns and type safety
 *
 * This barrel export provides backward compatibility for existing imports.
 */

// Export client and error handling
export { client, HttpClient, ApiErrorClass as ApiError } from "./client";

// Export all types
export type {
  ApiErrorResponse,
  User,
  OAuthAccount,
  AuthResponse,
  RegisterRequest,
  LoginRequest,
  ChangePasswordRequest,
  WatcherConfig,
  WatcherState,
} from "./types";

// Export API modules
export { authApi } from "./auth";
export { watcherApi } from "./watcher";
export { oauthApi } from "./oauth";
