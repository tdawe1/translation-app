# Plan: Frontend WebSocket Ticket Authentication

## Overview

Update the frontend WebSocket hook to use the secure ticket-based authentication system instead of passing JWT tokens in query parameters. The backend `POST /api/v1/auth/ws-ticket` endpoint already exists and returns a short-lived (30s) UUID ticket.

**Current Problem**: JWT tokens are passed in WebSocket URL query parameters, exposing them in logs and browser history.

**Solution**: Fetch a one-time ticket from `/api/v1/auth/ws-ticket` (uses httpOnly cookie), then connect to WebSocket with `?ticket=<uuid>`.

---

## Files to Change

| File | Changes |
|------|----------|
| `frontend/lib/api.ts` | Add `getWSTicket()` function to `authApi` |
| `frontend/hooks/use-watcher-websocket.ts` | Replace JWT fetch with ticket fetch; update WebSocket URL |

---

## Implementation Plan

### 1. Add `getWSTicket()` to `api.ts`

Add to `authApi` object:

```typescript
getWSTicket: (): Promise<{ ticket: string; expires_at: number }> =>
  client.post<{ ticket: string; expires_at: number }>("/api/v1/auth/ws-ticket"),
```

This uses the existing `HttpClient` which automatically includes `credentials: "include"` to send httpOnly cookies.

---

### 2. Update `use-watcher-websocket.ts`

#### Remove unused code
- Delete line 106: `const token = sessionStorage.getItem("access_token");`
- Delete lines 107-110: token existence check

#### Add ticket fetch before connection
Insert before `const ws = new WebSocket(url);`:

```typescript
// Fetch a one-time-use ticket for WebSocket authentication
let wsUrl = WS_URL;
try {
  const ticketResp = await authApi.getWSTicket();
  wsUrl = `${WS_URL}?ticket=${ticketResp.ticket}`;
} catch (err) {
  console.error("[WS] Failed to get WebSocket ticket:", err);
  // Schedule retry with exponential backoff
  return;
}
```

Then use `wsUrl` instead of `url`:
- Change `const url = ...` to `let wsUrl = WS_URL;`
- Change `new WebSocket(url)` to `new WebSocket(wsUrl)`

#### Make `connect` async
- Change `const connect = useCallback(() => {` to `const connect = useCallback(async () => {`

---

## Functions

### `authApi.getWSTicket()`
Fetches a one-time-use WebSocket ticket from the backend. Uses httpOnly cookie for authentication. Returns `{ ticket, expires_at }`.

### `connect()` (modified)
Async function that fetches a ticket before establishing WebSocket connection. Falls back to retry logic if ticket fetch fails.

---

## Tests

### `use-watcher-websocket.test.ts`

```typescript
describe("useWatcherWebSocket", () => {
  describe("ticket authentication", () => {
    it("fetches ticket from /api/v1/auth/ws-ticket before connecting")
    // Verifies authApi.getWSTicket() is called

    it("uses ticket in WebSocket URL query parameter")
    // Verifies URL format: ws://host/ws?ticket=uuid

    it("retries connection when ticket fetch fails")
    // Verifies exponential backoff on API error

    it("removes sessionStorage token dependency")
    // Verifies no access_token from sessionStorage is used
  })
})
```

---

## Execution Order

1. Add `getWSTicket()` to `api.ts`
2. Update `use-watcher-websocket.ts` (fetch ticket, use it in URL)
3. Test manually: open dashboard, verify WebSocket connects
4. Build: `npm run build`
5. Commit and push
