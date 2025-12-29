---
status: resolved
priority: p1
issue_id: "002"
tags:
  - security
  - websocket
  - cors
  - code-review
dependencies: []
---

# P1: WebSocket Missing Security Validations

## Problem Statement

The WebSocket handler has multiple security vulnerabilities:
1. JWT token passed via query parameter (leaks in logs/referers)
2. No Origin header validation (CSRF risk)
3. Timing attack vulnerability in signature verification

**Files**:
- `backend/internal/middleware/jwt.go:138-149` (token in query)
- `backend/internal/handlers/websocket.go:28-101` (no origin check)
- `backend/internal/handlers/lemonsqueezy.go:116-127` (timing attack)

## Findings

### Issue 1: JWT in Query Parameter
**Location**: `internal/middleware/jwt.go:138-149`

```go
func extractTokenFromQuery(c *fiber.Ctx, config *JWTConfig) (string, error) {
    token := c.Query("token", "")
    // ...
}
```

**OWASP**: A01:2021 - Broken Access Control
**CWE**: CWE-598 (Use of GET Request Method With Sensitive Query Strings)

Token appears in:
- Browser history
- Server access logs
- Referer headers
- ISP logs

### Issue 2: Missing Origin Validation
**Location**: `internal/handlers/websocket.go:28-101`

No validation of `Origin` header on WebSocket upgrade. Any site can open a WebSocket connection (though JWT still required).

### Issue 3: Webhook Timing Attack
**Location**: `internal/handlers/lemonsqueezy.go:116-127`

Signature format is `<timestamp>.<hex_signature>` but code ignores timestamp component, allowing replay attacks.

## Proposed Solutions

### Option 1: WebSocket Ticket System (Recommended)

Replace query parameter token with short-lived ticket exchange:

```go
// POST /api/v1/ws-ticket - exchanges JWT for short-lived WebSocket ticket
func (h *Handler) GetWSTicket(c *fiber.Ctx) error {
    userID := c.Locals("user").(jwt.MapClaims)["sub"].(string)

    ticket := uuid.New().String()
    // Store in Redis with 30s expiration
    h.redis.Set(ctx, "ws-ticket:"+ticket, userID, 30*time.Second)

    return c.JSON(fiber.Map{"ticket": ticket})
}

// WebSocket handler validates ticket
func (h *WebSocketHandler) HandleWebSocket() fiber.Handler {
    return websocket.New(func(c *websocket.Conn) {
        ticket := c.Query("ticket")
        userID, err := h.validateTicket(ticket)
        if err != nil {
            c.Close()
            return
        }
        // ... continue with userID
    })
}
```

**Pros**:
- Token never exposed in URL
- Tickets expire quickly (30 seconds)
- Captured tickets can't be reused
- No JWT in logs

**Cons**:
- Requires additional API endpoint
- Slightly more complex handshake
- Two round-trips for connection

**Effort**: Medium
**Risk**: Low

### Option 2: Add Origin Validation Only (Simpler)

```go
func (h *WebSocketHandler) isValidOrigin(origin string) bool {
    allowedOrigins := []string{
        "https://yourdomain.com",
        "http://localhost:3000",
    }
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
}

func (h *WebSocketHandler) HandleWebSocket() fiber.Handler {
    return websocket.New(func(c *websocket.Conn) {
        origin := c.Headers("Origin")
        if !h.isValidOrigin(origin) {
            log.Printf("[WS] Rejected: invalid origin %s", origin)
            c.Close()
            return
        }
        // ... rest of handler
    })
}
```

**Pros**:
- Simple implementation
- Prevents CSRF from malicious sites
- No API changes needed

**Cons**:
- JWT still in query parameter (logs/referers)
- Doesn't fully address token leakage

**Effort**: Small
**Risk**: Low

### Option 3: Subprotocol Negotiation

Use WebSocket subprotocol to pass token:

```go
// Client
const ws = new WebSocket("wss://host/ws", ["jwt-token", token]);

// Server - validate subprotocol
if c.Subprotocols()[0] != "jwt-token" {
    // reject
}
token := c.Subprotocols()[1]
```

**Pros**:
- Token not in URL
- Standard WebSocket feature

**Cons**:
- Browser WebSocket API has limited subprotocol support
- Token still visible in WebSocket handshake
- More complex client-side code

**Effort**: Medium
**Risk**: Medium (browser compatibility)

## Recommended Action

**Implement Option 1** (WebSocket Ticket System) for production deployments.

## Technical Details

### Affected Files
- `backend/internal/middleware/jwt.go:138-149`
- `backend/internal/handlers/websocket.go:28-101`
- `backend/internal/handlers/lemonsqueezy.go:116-127`
- `frontend/hooks/use-watcher-websocket.ts` (client-side)

### Components
- WebSocket connection handler
- JWT middleware
- Webhook signature verification

### Database Changes
None (uses Redis for ticket storage)

## Acceptance Criteria

- [ ] JWT token no longer passed in WebSocket URL
- [ ] `/api/v1/ws-ticket` endpoint implemented
- [ ] Tickets expire after 30 seconds
- [ ] Origin validation implemented
- [ ] Webhook timestamp validation implemented
- [ ] Replay attack prevention working
- [ ] All WebSocket connections require valid ticket
- [ ] Frontend updated to use ticket exchange
- [ ] Tests for ticket validation
- [ ] Documentation updated

## Work Log

### 2025-12-29
- **Finding**: Security audit identified multiple WebSocket security issues
- **Analysis**: Confirmed vulnerabilities in token handling and origin validation
- **Decision**: Selected ticket system approach
- **Status**: Pending implementation

## Resources

- [OWASP WebSocket Security](https://cheatsheetseries.owasp.org/cheatsheets/WebSocket_Security_Cheat_Sheet.html)
- [RFC 6455 - WebSocket Protocol](https://tools.ietf.org/html/rfc6455)
- [CWE-598](https://cwe.mitre.org/data/definitions/598.html)
