/**
 * HTTP Client with interceptors for auth and request deduplication
 */

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:37181";

/**
 * ApiErrorClass represents a structured API error
 * Named with "Class" suffix to avoid shadowing the ApiErrorResponse interface
 */
export class ApiErrorClass extends Error {
  code: string;
  details?: Record<string, unknown>;

  constructor(
    message: string,
    code: string,
    details?: Record<string, unknown>,
  ) {
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
  private refreshPromise: Promise<boolean> | null = null;

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
   * Generate a cache key for deduplication
   * Based on method, path, and request body
   */
  private getCacheKey(method: string, path: string, body?: string): string {
    return `${method}:${path}${body ? `:${body}` : ""}`;
  }

  private async readResponseData(response: Response): Promise<unknown> {
    if (typeof response.text !== "function") {
      if (typeof response.json === "function") {
        return response.json().catch(() => ({}));
      }
      return {};
    }

    const text = await response.text();
    if (text.trim() === "") {
      return {};
    }

    try {
      return JSON.parse(text);
    } catch {
      return {
        error: text,
      };
    }
  }

  private responseDataAsObject(data: unknown): Record<string, unknown> {
    return data && typeof data === "object" && !Array.isArray(data)
      ? (data as Record<string, unknown>)
      : {};
  }

  private async request<T>(
    path: string,
    options: RequestInit & { optional?: boolean } = {},
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const method = options.method || "GET";
    const body = options.body as string | undefined;

    // Check for in-flight request (deduplication) - only if enabled
    if (this.enableDeduplication) {
      const cacheKey = this.getCacheKey(method, path, body);
      const existingRequest = this.pendingRequests.get(cacheKey) as
        | Promise<T>
        | undefined;
      if (existingRequest) {
        return existingRequest;
      }
    }

    const headers: Record<string, string> = {
      ...(this.defaultHeaders as Record<string, string>),
      ...(options.headers as Record<string, string>),
    };

    // Create the request promise and store it for deduplication
    const requestPromise = (async () => {
      let response = await fetch(url, {
        ...options,
        headers,
        credentials: "include", // Send httpOnly cookies
      });

      if (response.status === 401 && this.shouldAttemptRefresh(path)) {
        const refreshed = await this.refreshSession();
        if (refreshed) {
          response = await fetch(url, {
            ...options,
            headers,
            credentials: "include",
          });
        }
      }

      // Handle 401 Unauthorized - try to read error message first
      if (response.status === 401) {
        // Try to read the actual error message from the response body.
        const data = this.responseDataAsObject(
          await this.readResponseData(response),
        );
        // If optional mode, return null instead of throwing
        if (options.optional) {
          return null as T;
        }
        // Don't redirect here - let the auth store handle routing
        // This prevents duplicate redirects and allows the auth store to manage state
        throw new ApiErrorClass(
          (data.error as string) || "Unauthorized",
          (data.code as string) || "UNAUTHORIZED",
          data.details as Record<string, unknown> | undefined,
        );
      }

      // Handle other errors
      if (!response.ok) {
        const data = this.responseDataAsObject(
          await this.readResponseData(response),
        );
        throw new ApiErrorClass(
          (data.error as string) || response.statusText,
          (data.code as string) || "UNKNOWN_ERROR",
          data.details as Record<string, unknown> | undefined,
        );
      }

      return (await this.readResponseData(response)) as T;
    })();

    // Store the promise for deduplication and clean up after completion
    if (this.enableDeduplication) {
      const cacheKey = this.getCacheKey(method, path, body);
      this.pendingRequests.set(cacheKey, requestPromise);

      void requestPromise
        .finally(() => {
          this.pendingRequests.delete(cacheKey);
        })
        .catch(() => {
          // Callers handle API errors; avoid turning expected 401s into dev-overlay noise.
        });
    }

    return requestPromise;
  }

  private shouldAttemptRefresh(path: string): boolean {
    return ![
      "/api/v1/auth/login",
      "/api/v1/auth/logout",
      "/api/v1/auth/register",
      "/api/v1/auth/refresh",
    ].includes(path);
  }

  private async refreshSession(): Promise<boolean> {
    if (!this.refreshPromise) {
      this.refreshPromise = (async () => {
        const response = await fetch(`${this.baseUrl}/api/v1/auth/refresh`, {
          method: "POST",
          headers: this.defaultHeaders,
          credentials: "include",
        });
        return response.ok;
      })()
        .catch(() => false)
        .finally(() => {
          this.refreshPromise = null;
        });
    }

    return this.refreshPromise;
  }

  get<T>(
    path: string,
    options?: RequestInit & { optional?: boolean },
  ): Promise<T> {
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
