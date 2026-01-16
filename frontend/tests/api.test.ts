/**
 * HttpClient Request Deduplication Tests
 *
 * Tests for the request deduplication toggle feature:
 * - When enabled, concurrent identical requests return the same Promise
 * - When disabled, each call creates a separate fetch request
 * - Cache entries are cleaned up after completion (success or failure)
 * - Cache keys are unique per method, path, and body
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { HttpClient } from '@/lib/api';

// Mock global fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

// Mock sessionStorage
const sessionStorageMock = (() => {
  let store: Record<string, string> = {};

  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value.toString();
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
  };
})();

Object.defineProperty(window, 'sessionStorage', {
  value: sessionStorageMock,
});

// Helper to access private properties for testing
function getPendingRequests(client: HttpClient): Map<string, Promise<unknown>> {
  return (client as unknown as { pendingRequests: Map<string, Promise<unknown>> }).pendingRequests;
}

describe('HttpClient Request Deduplication', () => {
  let client: HttpClient;

  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorageMock.clear();

    // Reset environment variable
    delete process.env.NEXT_PUBLIC_ENABLE_REQUEST_DEDUP;

    // Create client with test base URL
    client = new HttpClient('http://test.local');

    // Default successful fetch response
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ data: 'test-response' }),
    } as Response);
  });

  describe('deduplication enabled (default)', () => {
    beforeEach(() => {
      delete process.env.NEXT_PUBLIC_ENABLE_REQUEST_DEDUP;
      client = new HttpClient('http://test.local');
    });

    it('returns same promise for concurrent identical requests', async () => {
      // Track fetch calls more carefully - use a counter
      let fetchCallCount = 0;
      mockFetch.mockImplementation(async () => {
        fetchCallCount++;
        return {
          ok: true,
          json: async () => ({ data: 'response-1', call: fetchCallCount }),
        };
      });

      // Make concurrent identical requests - the second should reuse the first's promise
      const promise1 = client.get('/api/test');

      // Second request before first completes should reuse the promise
      const promise2 = client.get('/api/test');

      // Both promises resolve to the same value (from the single fetch)
      const result1 = await promise1;
      const result2 = await promise2;

      // Only one fetch should have been made
      expect(fetchCallCount).toBe(1);

      // Both results should be identical
      expect(result1).toEqual({ data: 'response-1', call: 1 });
      expect(result2).toEqual({ data: 'response-1', call: 1 });
    });

    it('cleans up cache entry after successful request', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ success: true }),
      } as Response);

      const pendingRequests = getPendingRequests(client);

      // Before request: cache should be empty
      expect(pendingRequests.size).toBe(0);

      const promise = client.get('/api/test');
      const cacheKey = 'GET:/api/test';

      // During request: cache should have one entry
      expect(pendingRequests.size).toBe(1);
      expect(pendingRequests.has(cacheKey)).toBe(true);

      // Wait for completion
      await promise;

      // After request: cache should be empty again
      expect(pendingRequests.size).toBe(0);
      expect(pendingRequests.has(cacheKey)).toBe(false);
    });

    it('cleans up cache entry after failed request', async () => {
      // Use a special mock that tracks cleanup without throwing
      let fetchCompleted = false;
      mockFetch.mockImplementation(async () => {
        fetchCompleted = true;
        return {
          ok: false,
          status: 500,
          json: async () => ({ error: 'Server error', code: 'INTERNAL_ERROR' }),
        } as Response;
      });

      const pendingRequests = getPendingRequests(client);
      const cacheKey = 'GET:/api/test';

      // Make request - catch the error to prevent unhandled rejection
      const promise = client.get('/api/test').catch((err) => {
        // Expected error, return null to resolve
        return null;
      });

      // During request: cache should have entry
      expect(pendingRequests.size).toBe(1);
      expect(pendingRequests.has(cacheKey)).toBe(true);

      // Wait for completion
      await promise;

      // Verify fetch was called
      expect(fetchCompleted).toBe(true);

      // After failed request: cache should be empty (cleanup happened)
      expect(pendingRequests.size).toBe(0);
      expect(pendingRequests.has(cacheKey)).toBe(false);
    });

    it('generates unique cache keys for different HTTP methods', async () => {
      mockFetch.mockImplementation(async () => ({
        ok: true,
        json: async () => ({ data: 'response' }),
      }));

      const pendingRequests = getPendingRequests(client);

      // Make concurrent GET and POST to same path
      const getPromise = client.get('/api/test');
      const postPromise = client.post('/api/test', { body: 'data' });

      // Should be different promises
      expect(getPromise).not.toBe(postPromise);

      // Two fetch calls should be made
      expect(mockFetch).toHaveBeenCalledTimes(2);

      // Cache should have two entries with different keys
      expect(pendingRequests.size).toBe(2);
      expect(pendingRequests.has('GET:/api/test')).toBe(true);
      expect(pendingRequests.has('POST:/api/test:{"body":"data"}')).toBe(true);

      // Cleanup
      await Promise.all([getPromise, postPromise]);
    });

    it('generates unique cache keys for different request bodies', async () => {
      mockFetch.mockImplementation(async () => ({
        ok: true,
        json: async () => ({ data: 'response' }),
      }));

      const pendingRequests = getPendingRequests(client);

      // Make concurrent POST requests with different bodies
      const promise1 = client.post('/api/test', { id: 1 });
      const promise2 = client.post('/api/test', { id: 2 });

      // Should be different promises
      expect(promise1).not.toBe(promise2);

      // Two fetch calls should be made
      expect(mockFetch).toHaveBeenCalledTimes(2);

      // Cache should have two entries
      expect(pendingRequests.size).toBe(2);

      await Promise.all([promise1, promise2]);
    });

    it('handles requests with same path but different bodies as separate', async () => {
      mockFetch.mockImplementation(async () => ({
        ok: true,
        json: async () => ({ success: true }),
      }));

      const pendingRequests = getPendingRequests(client);

      const body1 = { username: 'user1' };
      const body2 = { username: 'user2' };

      const promise1 = client.post('/api/user', body1);
      const promise2 = client.post('/api/user', body2);

      // Should NOT deduplicate - different bodies
      expect(promise1).not.toBe(promise2);
      expect(mockFetch).toHaveBeenCalledTimes(2);

      // Two distinct cache entries
      expect(pendingRequests.size).toBe(2);
      expect(pendingRequests.has('POST:/api/user:' + JSON.stringify(body1))).toBe(true);
      expect(pendingRequests.has('POST:/api/user:' + JSON.stringify(body2))).toBe(true);

      await Promise.all([promise1, promise2]);
    });
  });

  describe('deduplication disabled (development mode)', () => {
    beforeEach(() => {
      process.env.NEXT_PUBLIC_ENABLE_REQUEST_DEDUP = 'false';
      client = new HttpClient('http://test.local');

      // Reset fetch to avoid state from previous tests
      mockFetch.mockReset();
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ data: 'test-response' }),
      } as Response);
    });

    it('creates separate requests for concurrent identical calls', async () => {
      mockFetch.mockImplementation(async () => ({
        ok: true,
        json: async () => ({ data: 'response' }),
      }));

      // Make concurrent identical requests
      const promise1 = client.get('/api/test');
      const promise2 = client.get('/api/test');

      // Should be different promises (no deduplication)
      expect(promise1).not.toBe(promise2);

      // Two fetch calls should be made
      expect(mockFetch).toHaveBeenCalledTimes(2);

      // Both resolve independently
      const result1 = await promise1;
      const result2 = await promise2;
      expect(result1).toEqual({ data: 'response' });
      expect(result2).toEqual({ data: 'response' });
    });

    it('does not cache any pending requests', async () => {
      // Use a delayed promise to check cache during request
      let resolveFetch: ((value: Response) => void) | undefined;
      mockFetch.mockImplementation(() => {
        return new Promise<Response>((resolve) => {
          resolveFetch = resolve;
        });
      });

      const pendingRequests = getPendingRequests(client);

      // Start a request (it will hang until we resolve)
      const promise = client.get('/api/test');

      // Cache should remain empty when deduplication is disabled
      expect(pendingRequests.size).toBe(0);

      // Resolve and clean up
      resolveFetch?.({
        ok: true,
        json: async () => ({ data: 'response' }),
      } as Response);
      await promise;
    });

    it('exposes race conditions by allowing parallel requests', async () => {
      // Track each request's execution order
      const executionOrder: number[] = [];
      let requestCounter = 0;

      mockFetch.mockImplementation(async () => {
        // Capture the counter value at fetch time
        const myCallNumber = ++requestCounter;
        executionOrder.push(myCallNumber);

        // Simulate a delay to allow parallel requests
        await new Promise((resolve) => setTimeout(resolve, 10));

        return {
          ok: true,
          json: async () => ({ request: myCallNumber }),
        } as Response;
      });

      // Fire multiple identical requests simultaneously
      const promises = [
        client.get('/api/race'),
        client.get('/api/race'),
        client.get('/api/race'),
      ];

      const results = await Promise.all(promises);

      // All three requests should have been made (no deduplication)
      expect(requestCounter).toBe(3);
      expect(executionOrder).toHaveLength(3);

      // Each result should have its unique request number
      const requestNumbers = results.map((r: { request: number }) => r.request).sort((a, b) => a - b);
      expect(requestNumbers).toEqual([1, 2, 3]);
    });
  });

  describe('cache key generation', () => {
    it('includes method, path, and body in cache key', async () => {
      process.env.NEXT_PUBLIC_ENABLE_REQUEST_DEDUP = 'true';
      client = new HttpClient('http://test.local');

      mockFetch.mockImplementation(async () => ({
        ok: true,
        json: async () => ({ data: 'ok' }),
      }));

      const pendingRequests = getPendingRequests(client);

      const promise = client.post('/api/test', { foo: 'bar' });

      const expectedKey = 'POST:/api/test:' + JSON.stringify({ foo: 'bar' });
      expect(pendingRequests.has(expectedKey)).toBe(true);

      await promise;
    });

    it('treats GET requests without body as having no body suffix', async () => {
      process.env.NEXT_PUBLIC_ENABLE_REQUEST_DEDUP = 'true';
      client = new HttpClient('http://test.local');

      mockFetch.mockImplementation(async () => ({
        ok: true,
        json: async () => ({ data: 'ok' }),
      }));

      const pendingRequests = getPendingRequests(client);

      const promise = client.get('/api/test');

      expect(pendingRequests.has('GET:/api/test')).toBe(true);

      await promise;
    });
  });
});
