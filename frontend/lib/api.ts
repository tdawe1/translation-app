/**
 * API Client for GengoWatcher SaaS
 * Communicates with the Go backend at /api/v1/*
 */

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8000";

export interface ApiError {
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

class ApiError extends Error {
  code: string;
  details?: Record<string, unknown>;

  constructor(message: string, code: string, details?: Record<string, unknown>) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.details = details;
  }
}

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
    const headers: HeadersInit = {
      ...this.defaultHeaders,
      ...options.headers,
    };
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      ...options,
      headers,
      credentials: "include", // Send httpOnly cookies
    });

    // Handle 401 Unauthorized - try to refresh or redirect
    if (response.status === 401) {
      sessionStorage.removeItem("access_token");
      window.location.href = "/auth/login";
      throw new ApiError("Unauthorized", "UNAUTHORIZED");
    }

    // Handle other errors
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      throw new ApiError(
        data.error || response.statusText,
        data.code || "UNKNOWN_ERROR",
        data.details
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

  delete<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: "DELETE" });
  }
}

const client = new HttpClient(API_URL);

/**
 * Auth API
 */
export const authApi = {
  register: (data: RegisterRequest): Promise<AuthResponse> =>
    client.post<AuthResponse>("/api/v1/auth/register", data),

  login: (data: LoginRequest): Promise<AuthResponse> =>
    client.post<AuthResponse>("/api/v1/auth/login", data),

  logout: (): Promise<{ message: string }> =>
    client.post<{ message: string }>("/api/v1/auth/logout"),

  me: (): Promise<User> => client.get<User>("/api/v1/me"),
};

/**
 * Watcher API
 */
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

export { client, ApiError };
