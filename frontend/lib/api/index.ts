/**
 * API Client for GengoWatcher SaaS
 * Refactored with proper separation of concerns and type safety
 *
 * This barrel export provides backward compatibility for existing imports.
 */

// Export client and error handling
export { client, HttpClient, ApiErrorClass as ApiError, ApiErrorClass } from "./client";

// Export all types
export type {
  ApiErrorResponse,
  BillingCheckoutResponse,
  BillingPlan,
  BillingStatusResponse,
  User,
  OAuthAccount,
  AuthResponse,
  RegisterRequest,
  LoginRequest,
  ChangePasswordRequest,
  WatcherConfig,
  WatcherState,
  TranslationJob,
  TranslationSegment,
  JobSummary,
  ListJobsResponse,
  CreateJobRequest,
  UpdateSegmentRequest,
  RejectJobRequest,
  FlaggedSegmentsResponse,
  PaymentTransaction,
  TranslationJobStatus,
} from "./types";

// Export API modules
export { authApi } from "./auth";
export { billingApi } from "./billing";
export { watcherApi } from "./watcher";
export { oauthApi } from "./oauth";
export { translationApi } from "./translation";
