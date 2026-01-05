# GengoWatcher SaaS - Sprint 3 Plan (REVISED)

**Status**: Planning | **Est. Duration**: 3 days (down from 6)
**Start Date**: 2025-12-30
**Tech Stack**: Go 1.23 + Fiber 3.x + GORM + Next.js 16 + React 19

---

## What Changed (Review Feedback)

After peer review, the original 6-day plan was **overengineered**. Key findings:

1. **Security tasks (3.1)** - Already complete in codebase
   - JWT validation: ✅ Already implemented (`jwt.go:34-46`)
   - Origin validation: ✅ Already implemented (`websocket.go:139-156`)
   - Rate limiting: ✅ Already implemented (`ratelimit.go`)

2. **API Key Management (3.3)** - Premature for MVP
   - No users yet, no integration use cases
   - Move to Sprint 5+ when customers request it

3. **OAuth Integration (3.2)** - Overcomplicated
   - No need to store OAuth tokens (not calling Google APIs)
   - No need for separate `OAuthAccount` table
   - Use simple `users.provider` column instead

4. **Email Job Notifications** - Duplicate functionality
   - WebSocket real-time updates already implemented
   - Email is redundant fallback

---

## Sprint 3 Goals (Simplified)

1. **OAuth Integration** - Google + GitHub login (simplified)
2. **Magic Link Authentication** - Passwordless option
3. **Basic Settings Page** - Profile management

---

## Task Breakdown (3 Days)

### Task 3.1: OAuth Integration (1 day)

**Priority**: P2 - Core Feature

**Simplified Approach**:
- Add `provider` and `provider_id` columns to `users` table
- Use Redis for OAuth state (10-minute TTL)
- Fetch email, discard token immediately
- No token storage, no encryption needed

**Migration**:
```sql
ALTER TABLE users ADD COLUMN provider VARCHAR(20);
ALTER TABLE users ADD COLUMN provider_id VARCHAR(255);
CREATE INDEX idx_users_provider ON users(provider, provider_id);
```

**Backend Files**:
- `backend/internal/handlers/oauth.go` (NEW)
  - `GET /api/v1/auth/oauth/:provider` - Start OAuth flow
  - `GET /api/v1/auth/oauth/:provider/callback` - OAuth callback

**Frontend Files**:
- `frontend/app/auth/login/page.tsx` (MODIFY)
  - Add "Continue with Google" button
  - Add "Continue with GitHub" button

---

### Task 3.2: Magic Link Authentication (0.5 day)

**Priority**: P2 - UX Enhancement

**Implementation**:
- Store token in Redis with 15-minute expiry
- Use `GETDEL` for atomic consume (prevent reuse)
- Frontend success page with auto-redirect

**Backend Files**:
- `backend/internal/handlers/auth.go` (MODIFY)
  - `POST /api/v1/auth/magic-link` - Request magic link
  - `GET /api/v1/auth/verify` - Verify token

**Email Templates**:
- Magic link email (Resend)

---

### Task 3.3: Basic Settings Page (0.5 day)

**Priority**: P3 - Basic Feature

**Sections** (keep it minimal):
1. **Profile** - Email, password change
2. **Connected Accounts** - Read-only display of OAuth provider
3. **Sign Out** - Logout button

**NOT in Sprint 3**:
- API Keys (deferred to Sprint 5)
- Notification preferences (only one type exists)
- Delete account (deferred to Sprint 4)

**Frontend Files**:
- `frontend/app/settings/page.tsx` (NEW)
- `frontend/components/settings/profile-section.tsx` (NEW)
- `frontend/components/settings/connected-accounts.tsx` (NEW)

---

### Task 3.4: Testing & Polish (1 day)

**Priority**: P1 - Quality

**Tests**:
- OAuth flow integration tests (mock providers)
- Magic link atomic consume test
- Settings page E2E tests

**Files**:
- `backend/tests/oauth_test.go` (NEW)
- `backend/tests/auth_magic_link_test.go` (NEW)
- `frontend/tests/settings.test.tsx` (NEW)

---

## Dependencies

```
OAuth (3.1) ──> Settings (3.3)
Magic Link (3.2) ──> Settings (3.3)
All ──> Testing (3.4)
```

---

## Files to Create

### Backend
- `backend/internal/handlers/oauth.go` - OAuth flow endpoints
- `backend/internal/email/service.go` - Resend integration
- `backend/tests/oauth_test.go`
- `backend/tests/auth_magic_link_test.go`
- `backend/migrations/*_add_provider_columns.go`

### Frontend
- `frontend/app/settings/page.tsx` - Settings page
- `frontend/components/settings/profile-section.tsx`
- `frontend/components/settings/connected-accounts.tsx`
- `frontend/tests/settings.test.tsx`

---

## Files to Modify

### Backend
- `backend/internal/handlers/auth.go` - Add magic link endpoints
- `backend/internal/models/user.go` - Add Provider, ProviderID fields

### Frontend
- `frontend/app/auth/login/page.tsx` - Add OAuth buttons

---

## Security Considerations

### OAuth State (Redis)
```go
// Store with auto-expiration
key := fmt.Sprintf("oauth:state:%s", state)
redis.Set(ctx, key, provider, 10*time.Minute)

// Atomic validate and consume
val, err := redis.GetDel(ctx, key).Result()
```

### Magic Link (Atomic Consume)
```go
// GETDEL prevents concurrent reuse
email, err := redis.GetDel(ctx, "magiclink:"+token).Result()
if err == redis.Nil {
    return c.Status(400).JSON(fiber.Map{"error": "Invalid or expired token"})
}
```

### No Token Storage
OAuth tokens are discarded immediately after fetching email. No encryption needed since we're not storing them.

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| OAuth provider downtime | Low | Email/password still works |
| Redis downtime | Medium | Fallback to database for state |
| Email delivery failure | Medium | Show "resend" button |
| Magic link phishing | Low | Short TTL (15 min), single-use |

---

## Definition of Done

- [ ] OAuth (Google + GitHub) login working
- [ ] Magic link authentication working
- [ ] Settings page with Profile + Connected Accounts
- [ ] All new code has tests
- [ ] Production build successful
- [ ] No regression in existing tests

---

## Removed from Original Plan

| Task | Reason |
|------|--------|
| JWT validation | Already implemented |
| Origin validation | Already implemented |
| Rate limiting | Already implemented |
| API Key Management | Premature for MVP |
| Job notification emails | WebSocket already handles this |
| API key scopes | No permission system designed |
| Settings: Notification preferences | Only one notification type |
| Settings: Delete account | Defer to later sprint |

---

## What's Next (Sprint 4)

After Sprint 3 completes:
1. LemonSqueezy billing integration
2. Job history/persistence
3. Account deletion
4. Deployment preparation

---

## Notes

**Why 3 days instead of 6?**
- Removed 1 day of already-complete security work
- Removed 1.5 days of premature API key management
- Removed 0.5 day of duplicate email notifications
- Simplified OAuth (no token storage, no separate table)

**Philosophy**: Ship the MVP. Add features when users ask for them.
