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
  provider?: string; // 'google', 'github', or undefined
  oauth_accounts?: OAuthAccount[];
}

export interface OAuthAccount {
  provider: string; // 'google', 'github'
  created_at: string; // ISO 8601 timestamp
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
  worker_id: string;
  user_id: string;
  watcher_status: string;
  overall_status: string;
  feed_status: string;
  browser_status: string;
  action_status: string;
  alert_status: string;
  profile_status: string;
  current_job_id?: string;
  current_action_step?: string;
  current_url?: string;
  current_title?: string;
  frontend_url?: string;
  frontend_title?: string;
  frontend_last_seen_at?: string;
  logged_in_state?: string;
  browser_process_alive?: boolean;
  devtools_connected?: boolean;
  last_rss_poll_started_at?: string;
  last_rss_poll_ok_at?: string;
  rss_consecutive_failures?: number;
  last_ws_connect_at?: string;
  last_ws_message_at?: string;
  last_ws_pong_at?: string;
  last_ws_close_code?: number | null;
  last_ws_close_reason?: string;
  ws_reconnect_count?: number;
  last_browser_heartbeat_at?: string;
  last_error?: string;
  last_critical_alert?: string;
  latest_screenshot_artifact_id?: string;
  latest_screenshot_url?: string;
  total_jobs_found: number;
  total_jobs_accepted: number;
  total_earnings: number;
  last_activity: string;
  updated_at?: string;
}

export interface BrowserStateSyncRequest {
  current_url?: string;
  current_title?: string;
  current_action_step?: string;
  current_job_id?: string;
  frontend_url?: string;
  frontend_title?: string;
  logged_in_state?: "unknown" | "logged_in" | "logged_out";
  browser_process_alive?: boolean;
  devtools_connected?: boolean;
}

export interface WatcherEvent {
  id: string;
  worker_id: string;
  user_id: string;
  level: string;
  source: string;
  type: string;
  job_id?: string;
  message: string;
  data: Record<string, unknown>;
  occurred_at: string;
}

export interface WatcherEventsResponse {
  events: WatcherEvent[];
}

export type TranslationJobStatus =
  | "pending"
  | "processing"
  | "translating"
  | "pending_approval"
  | "approved"
  | "rejected"
  | "completed"
  | "failed"
  | "cancelled";

export interface TranslationJob {
  id: string;
  user_id: string;
  source_file: string;
  target_file?: string;
  source_lang: string;
  target_lang: string;
  status: TranslationJobStatus;
  project_type: "critical" | "routine";
  approval_mode: "blocking" | "async";
  overall_score: number;
  segment_count: number;
  flagged_count: number;
  judge_resolutions: number;
  progress: number;
  error?: string;
  worker_id?: string;
  redis_job_id?: string;
  completed_at?: string;
  approved_at?: string;
  approved_by?: string;
  created_at: string;
  updated_at: string;
  segments?: TranslationSegment[];
}

export interface TranslationSegment {
  id: string;
  job_id: string;
  segment_id: string;
  source: string;
  target: string;
  context?: string;
  judge_winner?: "model_a" | "model_b" | "edited" | "tie";
  judge_confidence: number;
  judge_reasoning?: string;
  is_flagged: boolean;
  flag_reason?: string;
  model_a_output?: string;
  model_b_output?: string;
  glossary_terms?: string;
  edited_by?: string;
  edited_at?: string;
  created_at: string;
  updated_at: string;
}

export interface JobSummary {
  id: string;
  source_file: string;
  status: TranslationJobStatus;
  overall_score: number;
  segment_count: number;
  flagged_count: number;
  progress: number;
  created_at: string;
  completed_at?: string;
}

export interface ListJobsResponse {
  jobs: JobSummary[];
  total_count: number;
  page: number;
  page_size: number;
}

export interface CreateJobRequest {
  source_file: string;
  source_lang?: string;
  target_lang?: string;
  project_type?: "critical" | "routine";
  approval_mode?: "blocking" | "async";
}

export interface UpdateSegmentRequest {
  target: string;
}

export interface RejectJobRequest {
  reason?: string;
}

export interface FlaggedSegmentsResponse {
  job_id: string;
  segments: TranslationSegment[];
  count: number;
}
