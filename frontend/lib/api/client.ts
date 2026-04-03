/**
 * HTTP Client with interceptors for auth and request deduplication
 */

import { getToken, clearToken as clearTokenStorage } from "../auth/tokens";

const API_URL = process.env.NEXT_PUBLIC_API_URL || process.env.REACT_APP_BACKEND_URL || "";

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

/**
 * HTTP client with interceptors for auth and request deduplication
 */
class HttpClient {
  private baseUrl: string;
  private defaultHeaders: Record<string, string>;
  // In-flight request deduplication: cache of pending requests
  private pendingRequests = new Map<string, Promise<unknown>>();

  constructor(baseUrl: string = API_URL) {
    this.baseUrl = baseUrl;
    this.defaultHeaders = {
      "Content-Type": "application/json",
    };
  }

  /**
   * Check if request deduplication is enabled via environment variable
   * Disabled in development by default to expose race conditions during testing
   */
  private get enableDeduplication(): boolean {
    // Default to true for safety (enable deduplication) unless explicitly disabled
    return process.env.NEXT_PUBLIC_ENABLE_REQUEST_DEDUP !== "false";
  }

  /**
   * Check if we have a token before making authenticated requests
   * Returns true if token exists, false otherwise
   */
  private hasToken(): boolean {
    return getToken() !== null;
  }

  /**
   * Generate a cache key for deduplication
   * Based on method, path, and request body
   */
  private getCacheKey(
    method: string,
    path: string,
    body?: string
  ): string {
    return `${method}:${path}${body ? `:${body}` : ""}`;
  }

  private async request<T>(
    path: string,
    options: RequestInit & { optional?: boolean } = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const method = options.method || "GET";
    const body = options.body as string | undefined;

    // Check for in-flight request (deduplication) - only if enabled
    if (this.enableDeduplication) {
      const cacheKey = this.getCacheKey(method, path, body);
      const existingRequest = this.pendingRequests.get(cacheKey) as Promise<T> | undefined;
      if (existingRequest) {
        return existingRequest;
      }
    }

    // Add access token from TokenService if available
    const token = getToken();
    // HeadersInit is a union type, so we use a plain object for manipulation
    const headers: Record<string, string> = {
      ...(this.defaultHeaders as Record<string, string>),
      ...(options.headers as Record<string, string>),
    };
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    // Create the request promise and store it for deduplication
    const requestPromise = (async () => {
      const response = await fetch(url, {
        ...options,
        headers,
        credentials: "include", // Send httpOnly cookies
      });

      // Handle 401 Unauthorized - try to read error message first
      if (response.status === 401) {
        // Try to read the actual error message from the response body
        const data = await response.json().catch((error) => {
          console.error("Failed to parse 401 error response:", error);
          return {};
        });
        clearTokenStorage(); // Use TokenService to clear token
        // If optional mode, return null instead of throwing
        if (options.optional) {
          return null as T;
        }
        // Don't redirect here - let the auth store handle routing
        // This prevents duplicate redirects and allows the auth store to manage state
        throw new ApiErrorClass(
          (data.error as string) || "Unauthorized",
          (data.code as string) || "UNAUTHORIZED",
          data.details as Record<string, unknown> | undefined
        );
      }

      // Handle other errors
      if (!response.ok) {
        const data = await response.json().catch((error) => {
          console.error("Failed to parse error response:", error);
          return {};
        });
        throw new ApiErrorClass(
          (data.error as string) || response.statusText,
          (data.code as string) || "UNKNOWN_ERROR",
          data.details as Record<string, unknown> | undefined
        );
      }

      return response.json() as Promise<T>;
    })();

    // Store the promise for deduplication and clean up after completion
    if (this.enableDeduplication) {
      const cacheKey = this.getCacheKey(method, path, body);
      this.pendingRequests.set(cacheKey, requestPromise);

      requestPromise
        .catch((error: unknown) => {
          console.error("[HttpClient] Request failed:", error);
        })
        .finally(() => {
          this.pendingRequests.delete(cacheKey);
        });
    }

    return requestPromise;
  }

  get<T>(path: string, options?: RequestInit & { optional?: boolean }): Promise<T> {
    return this.request<T>(path, { ...options, method: "GET" });
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

// Export singleton instance and class
const client = new HttpClient();
export { client, HttpClient };
