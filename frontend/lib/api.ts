/**
 * API Client for GengoWatcher SaaS
 * Refactored with proper separation of concerns and type safety
 */

// ============================================================
// Types and Interfaces
// ============================================================

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8000";

/** Shape of error response from the API */
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

// ============================================================
// Error Handling
// ============================================================

/**
 * ApiErrorClass represents a structured API error
 * Named with "Class" suffix to avoid shadowing the ApiErrorResponse interface
 */
export class ApiErrorClass extends Error {
  code: string;
  details?: Record<string, unknown>;

  constructor(message: string, code: string, details?: Record<string, unknown>) {
    super(message);
    this.name = "ApiErrorClass";
    this.code = code;
    this.details = details;
  }

  /**
   * Checks if this error is a specific error code
   */
  isCode(code: string): boolean {
    return this.code === code;
  }
}

// ============================================================
// HTTP Client
// ============================================================

/**
 * HTTP client with interceptors for auth
 */
class HttpClient {
  private baseUrl: string;
  private defaultHeaders: Record<string, string>;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
    this.defaultHeaders = {
      "Content-Type": "application/json",
    };
  }

  private async request<T>(
    path: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;

    // Add access token from sessionStorage if available
    const token = sessionStorage.getItem("access_token");
    // HeadersInit is a union type, so we use a plain object for manipulation
    const headers: Record<string, string> = {
      ...(this.defaultHeaders as Record<string, string>),
      ...(options.headers as Record<string, string>),
    };
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      ...options,
      headers,
      credentials: "include", // Send httpOnly cookies
    });

    // Handle 401 Unauthorized - redirect to login
    if (response.status === 401) {
      sessionStorage.removeItem("access_token");
      if (typeof window !== "undefined") {
        window.location.href = "/auth/login";
      }
      throw new ApiErrorClass("Unauthorized", "UNAUTHORIZED");
    }

    // Handle other errors
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new ApiErrorClass(
        (data.error as string) || response.statusText,
        (data.code as string) || "UNKNOWN_ERROR",
        data.details as Record<string, unknown> | undefined
      );
    }

    return response.json() as Promise<T>;
  }

  get<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: "GET" });
  }

  post<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>(path, {
      method: "POST",
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  put<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>(path, {
      method: "PUT",
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  patch<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>(path, {
      method: "PATCH",
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  delete<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: "DELETE" });
  }
}

// ============================================================
// API Client Singleton
// ============================================================

const client = new HttpClient(API_URL);

// ============================================================
// Auth API
// ============================================================

export const authApi = {
  register: (data: RegisterRequest): Promise<AuthResponse> =>
    client.post<AuthResponse>("/api/v1/auth/register", data),

  login: (data: LoginRequest): Promise<AuthResponse> =>
    client.post<AuthResponse>("/api/v1/auth/login", data),

  logout: (): Promise<void> =>
    client.post<void>("/api/v1/auth/logout"),

  me: (): Promise<User> => client.get<User>("/api/v1/me"),
};

// ============================================================
// Watcher API
// ============================================================

export const watcherApi = {
  getConfig: (): Promise<WatcherConfig> =>
    client.get<WatcherConfig>("/api/v1/watcher/config"),

  updateConfig: (data: Partial<WatcherConfig>): Promise<WatcherConfig> =>
    client.put<WatcherConfig>("/api/v1/watcher/config", data),

  getState: (): Promise<WatcherState> =>
    client.get<WatcherState>("/api/v1/watcher/state"),

  start: (): Promise<{ status: string }> =>
    client.post<{ status: string }>("/api/v1/watcher/start"),

  stop: (): Promise<{ status: string }> =>
    client.post<{ status: string }>("/api/v1/watcher/stop"),
};

// ============================================================
// Exports
// ============================================================

export { client, ApiErrorClass as ApiError };
