# Technical Debt Remediation - Phase 2

**Date:** 2025-01-12
**Status:** Draft
**Previous:** Phase 1 reduced debt from 720 to ~500

---

## Goal

Reduce technical debt score from ~500 to ~400 by addressing remaining security bugs, removing deprecated code, and splitting large files.

---

## Remaining Issues from Phase 1

| Priority | Issue | File | Impact |
|----------|-------|------|--------|
| HIGH | Admin can change own role | `admin.go:146` | Security |
| HIGH | Admin can delete self | `admin.go` | Security |
| MED | Deprecated email.go | `handlers/email.go` (514 lines) | Maintenance |
| LOW | Large API client | `lib/api.ts` (335 lines) | Maintainability |
| LOW | Large settings page | `settings/page.tsx` (503 lines) | Maintainability |

---

## Sprint 1: Security Fixes (Week 1)

### Task 1: Fix Admin Self-Change Prevention

**Priority:** HIGH (Security)
**Effort:** 1 hour
**Files:** `backend/internal/handlers/admin.go`, `backend/tests/admin_test.go`

**Problem:** The comparison `requestingUserID == userID.String()` at line 146 fails due to type issues. Users can change their own role or delete themselves.

**Root Cause:** The JWT claim's `sub` is stored as a string in a map interface, and the direct comparison with `userID.String()` may not work as expected.

**Solution:**

```go
// Parse requestingUserID as UUID first
requestingUserUUID, err := uuid.Parse(requestingUserID)
if err != nil {
    return RespondWithError(c, fiber.StatusUnauthorized, apperrors.ErrNotAuthenticated, "Invalid user ID in token")
}

// Now compare UUIDs
if requestingUserUUID == userID {
    return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Cannot change your own role")
}
```

**Acceptance:**
- [ ] Admin cannot change their own role via API (returns 400)
- [ ] Admin cannot delete themselves via API (returns 400)
- [ ] Tests pass with expected behavior

---

## Sprint 2: Code Cleanup (Week 2)

### Task 2: Remove Deprecated email.go

**Priority:** MEDIUM (Maintenance)
**Effort:** 30 minutes
**Files:** `backend/internal/handlers/email.go`

**Action:** Delete the deprecated `email.go` file after verifying:
1. No imports reference it
2. `main.go` uses the new handlers (`EmailVerificationHandler`, `MagicLinkHandler`, `PasswordResetHandler`)
3. All tests pass

**Verification:**
```bash
grep -r "EmailHandler" backend/cmd/server/main.go
# Should return no results
```

---

### Task 3: Split API Client by Resource

**Priority:** LOW (Maintainability)
**Effort:** 2 hours
**Files:** `frontend/lib/api.ts` (335 lines)

**Target Structure:**

```
frontend/lib/api/
├── index.ts         # Main exports
├── client.ts        # Base fetch client, error handling
├── auth.ts          # Authentication endpoints
├── watcher.ts       # Watcher config/state endpoints
├── oauth.ts         # OAuth endpoints
└── types.ts         # Shared types
```

**Benefits:**
- Easier to locate endpoint definitions
- Smaller files (<100 lines each)
- Better testability per resource

---

## Sprint 3: Component Refactoring (Week 3)

### Task 4: Split Settings Page

**Priority:** LOW (Maintainability)
**Effort:** 2 hours
**Files:** `frontend/app/settings/page.tsx` (503 lines)

**Current Structure:**
- User profile section
- Email/Password change
- OAuth linked accounts (ConnectedAccounts)
- API keys section
- Danger zone

**Target Structure:**

```
frontend/app/settings/
├── page.tsx              # Main layout, tabs/navigation (~150 lines)
├── profile-section.tsx    # User profile form (~80 lines)
├── security-section.tsx   # Email/password change (~100 lines)
├── oauth-section.tsx      # Connected accounts (~120 lines)
├── apikeys-section.tsx    # API key management (~80 lines)
└── danger-section.tsx     # Account deletion (~50 lines)
```

---

## Success Metrics

| Metric | Before | Target |
|--------|--------|--------|
| Debt Score | ~500 | ~400 |
| Known Security Bugs | 2 | 0 |
| Deprecated Files | 1 | 0 |
| Files >300 lines | 4 | 2 |
| Largest File | 514 lines | <350 lines |

---

## Execution Order

1. **Week 1**: Task 1 (Security fixes) - HIGH priority, quick win
2. **Week 2**: Task 2 (Delete deprecated code) + Task 3 (API client)
3. **Week 3**: Task 4 (Settings page split)

---

**Plan created:** 2025-01-12
**Next review:** After Task 1 completion
