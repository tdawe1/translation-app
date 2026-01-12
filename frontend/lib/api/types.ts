/**
 * Shared types for the API client
 */

export interface ApiErrorResponse {
  error: string;
  code: string;
  details?: Record<string, unknown>;
}

export interface User {
  id: string;
  email: string;
  email_verified: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  provider?: string;                    // 'google', 'github', or undefined
  oauth_accounts?: OAuthAccount[];
}

export interface OAuthAccount {
  provider: string;    // 'google', 'github'
  created_at: string;  // ISO 8601 timestamp
}

export interface AuthResponse {
  access_token: string;
  user: User;
}

export interface RegisterRequest {
  email: string;
  password: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface WatcherConfig {
  user_id: string;
  rss_feed_url: string;
  websocket_enabled: boolean;
  min_reward: number;
  max_reward: number;
  included_language_pairs: string[];
  enable_desktop_notifications: boolean;
  enable_sound_notifications: boolean;
  enable_email_notifications: boolean;
  auto_accept_enabled: boolean;
}

export interface WatcherState {
  user_id: string;
  watcher_status: string;
  total_jobs_found: number;
  total_jobs_accepted: number;
  total_earnings: number;
  last_activity: string;
}
