import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { getToken, setToken, clearToken, getTokenPayload, hasValidToken } from './tokens';

describe('TokenService', () => {
  beforeEach(() => {
    // Clear sessionStorage before each test
    sessionStorage.clear();
  });

  afterEach(() => {
    sessionStorage.clear();
  });

  describe('getToken', () => {
    it('should return null when no token exists', () => {
      const token = getToken();
      expect(token).toBeNull();
    });

    it('should return token when it exists', () => {
      sessionStorage.setItem('access_token', 'test-token-123');
      const token = getToken();
      expect(token).toBe('test-token-123');
    });

    it('should return null for empty string', () => {
      sessionStorage.setItem('access_token', '');
      const token = getToken();
      expect(token).toBeNull();
    });

    it('should trim whitespace and return null for whitespace-only strings', () => {
      sessionStorage.setItem('access_token', '   ');
      const token = getToken();
      expect(token).toBeNull();
    });
  });

  describe('setToken', () => {
    it('should store token in sessionStorage', () => {
      setToken('my-token');
      expect(sessionStorage.getItem('access_token')).toBe('my-token');
    });

    it('should clear existing token when setting new one', () => {
      setToken('old-token');
      setToken('new-token');
      expect(sessionStorage.getItem('access_token')).toBe('new-token');
    });

    it('should not store null or undefined', () => {
      setToken(null as any);
      expect(sessionStorage.getItem('access_token')).toBeNull();

      setToken(undefined as any);
      expect(sessionStorage.getItem('access_token')).toBeNull();
    });

    it('should not store empty string', () => {
      setToken('');
      expect(sessionStorage.getItem('access_token')).toBeNull();
    });
  });

  describe('clearToken', () => {
    it('should remove token from sessionStorage', () => {
      sessionStorage.setItem('access_token', 'test-token');
      clearToken();
      expect(sessionStorage.getItem('access_token')).toBeNull();
    });

    it('should be safe to call when no token exists', () => {
      expect(() => clearToken()).not.toThrow();
    });
  });

  describe('getTokenPayload', () => {
    it('should decode JWT payload', () => {
      // Format: header.payload.signature
      // Payload: {"id":"123","email":"test@example.com"}
      const mockToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjEyMyIsImVtYWlsIjoidGVzdEBleGFtcGxlLmNvbSJ9.signature';
      setToken(mockToken);
      const payload = getTokenPayload();
      expect(payload).toEqual({ id: '123', email: 'test@example.com' });
    });

    it('should decode JWT payload with exp and iat', () => {
      // Payload with exp and iat
      const mockToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjEyMyIsImVtYWlsIjoidGVzdEBleGFtcGxlLmNvbSIsImV4cCI6MTczNjY5MDYwMCwiaWF0IjoxNzM2Njg3MjAwfQ.signature';
      setToken(mockToken);
      const payload = getTokenPayload();
      expect(payload).toEqual({
        id: '123',
        email: 'test@example.com',
        exp: 1736690600,
        iat: 1736687200
      });
    });

    it('should return null for invalid token', () => {
      setToken('invalid-token');
      const payload = getTokenPayload();
      expect(payload).toBeNull();
    });

    it('should return null when no token exists', () => {
      const payload = getTokenPayload();
      expect(payload).toBeNull();
    });

    it('should return null for token with wrong number of parts', () => {
      setToken('only.one.part');
      const payload = getTokenPayload();
      expect(payload).toBeNull();
    });
  });

  describe('hasValidToken', () => {
    it('should return true when token has valid exp', () => {
      // Token with exp far in the future
      const futureExp = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now
      const header = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9';
      const payload = btoa(JSON.stringify({ id: '123', exp: futureExp }));
      const signature = 'signature';
      setToken(`${header}.${payload}.${signature}`);

      expect(hasValidToken()).toBe(true);
    });

    it('should return false when token is expired', () => {
      // Token with exp in the past
      const pastExp = Math.floor(Date.now() / 1000) - 3600; // 1 hour ago
      const header = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9';
      const payload = btoa(JSON.stringify({ id: '123', exp: pastExp }));
      const signature = 'signature';
      setToken(`${header}.${payload}.${signature}`);

      expect(hasValidToken()).toBe(false);
    });

    it('should return false when no token exists', () => {
      expect(hasValidToken()).toBe(false);
    });

    it('should return false when token has no exp claim', () => {
      const mockToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjEyMyIsImVtYWlsIjoidGVzdEBleGFtcGxlLmNvbSJ9.signature';
      setToken(mockToken);
      expect(hasValidToken()).toBe(false);
    });

    it('should return false when token is expired within buffer', () => {
      // Token expiring within 30 second buffer
      const nearExp = Math.floor(Date.now() / 1000) + 20; // 20 seconds from now
      const header = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9';
      const payload = btoa(JSON.stringify({ id: '123', exp: nearExp }));
      const signature = 'signature';
      setToken(`${header}.${payload}.${signature}`);

      expect(hasValidToken()).toBe(false);
    });
  });
});
