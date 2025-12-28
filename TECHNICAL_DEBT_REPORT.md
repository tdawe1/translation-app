# Technical Debt Report: GengoWatcher SaaS

**Analysis Date**: 2025-12-28
**Project Stage**: Sprint 0 (Scaffolding) - ~80% complete
**Scope**: Actual technical debt only (excluding unfinished work)

---

## What This Report Covers

This report focuses on **actual technical debt** - design choices and implementation patterns that will create future maintenance burden. It **excludes** "not done yet" items like:
- Test coverage (tests written after features stabilize)
- CI/CD (setup when ready to deploy)
- Monitoring/observability (infrastructure comes later)
- Documentation (docs lag development, normal)

---

## Executive Summary

**12 actual technical debt items identified** across backend, frontend, and infrastructure.

| Severity | Count | Effort to Fix |
|----------|-------|---------------|
| Critical | 3 | ~2 hours |
| High | 5 | ~1 day |
| Medium | 4 | ~2 days |

**Total Remediation**: ~3-4 days for all actual debt items.

---

## Part 1: Backend Technical Debt

### 1. CRITICAL (Fix Before Continuing)

#### 1.1 Hardcoded JWT Secret Fallback
**Severity**: Critical | **File**: `backend/cmd/server/main.go:49-52`

```go
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    jwtSecret = "dev-secret-change-in-production"  // ❌ Silent failure
}
```

**Why It's Debt**: Silent fallback means production could start with weak defaults if env var is missing - a ticking time bomb.

**Fix** (2 min):
```go
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    log.Fatal("JWT_SECRET environment variable required")
}
```

---

#### 1.2 Global Database State
**Severity**: Critical | **File**: `backend/internal/models/database.go:14`

```go
var DB *gorm.DB  // ❌ Global variable
```

**Why It's Debt**: Global state makes testing impossible and creates implicit dependencies throughout the codebase. Harder to refactor later because everything depends on it.

**Fix** (2 hours): Inject DB as dependency instead of using global.

---

#### 1.3 Hardcoded CORS Origins
**Severity**: Critical | **File**: `backend/cmd/server/main.go:41-46`

```go
AllowOrigins: "http://localhost:3000,http://localhost:3001"
```

**Why It's Debt**: Will silently break production deployment. Frontend won't be able to authenticate.

**Fix** (10 min):
```go
allowOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
if allowOrigins == "" {
    allowOrigins = "http://localhost:3000"
}
```

---

### 2. HIGH SEVERITY

#### 2.1 No Request Timeout Context
**Severity**: High | **Files**: All database operations

**Why It's Debt**: Database queries don't use request context, so they won't timeout properly. Could accumulate hanging connections.

**Fix** (1 hour): Pass `c.Context()` to GORM queries.

---

#### 2.2 Inconsistent Error Responses
**Severity**: High | **Files**: Handler files

**Why It's Debt**: Some handlers return `fiber.Map{"error": ...}`, others return different structures. Inconsistent for API consumers.

**Fix** (2 hours): Create unified error response middleware.

---

## Part 2: Frontend Technical Debt

### 1. CRITICAL (Fix Before Continuing)

#### 1.1 Hard Navigation Anti-Pattern
**Severity**: Critical | **Files**:
- `frontend/lib/api.ts:89`
- `frontend/app/dashboard/page.tsx:16`

```typescript
window.location.href = "/auth/login"  // ❌ Breaks SPA behavior
```

**Why It's Debt**: Using Next.js wrong from the start. Breaks SPA navigation, loses React state, can't animate transitions. Harder to fix later because pattern is spread through codebase.

**Fix** (1 hour): Return errors from API and let components handle navigation with `router.push()`.

---

### 2. HIGH SEVERITY

#### 2.1 Duplicate Auth Form Markup (~100 lines duplicated)
**Severity**: High | **Files**:
- `frontend/app/auth/login/page.tsx`
- `frontend/app/auth/register/page.tsx`

**Why It's Debt**: Login and register pages share 90% identical markup (inputs, OAuth buttons, form layout). Every styling change requires updating two files.

**Fix** (3 hours): Extract shared components to `components/auth/`.

---

#### 2.2 sessionStorage Scattered Across 7 Files
**Severity**: High | **Files**: Multiple

**Why It's Debt**: Token storage logic duplicated. If you need to switch to httpOnly cookies (recommended for security), you'll need to update 7 files.

**Fix** (2 hours): Create `lib/auth/token-storage.ts` abstraction.

---

#### 2.3 Untyped API Error Details
**Severity**: High | **File**: `frontend/lib/api.ts:11,40`

```typescript
details?: Record<string, unknown>  // ❌ No type safety
```

**Why It's Debt**: Error handling code must typecast at runtime. Defeats purpose of TypeScript.

**Fix** (1 hour): Define proper error detail types.

---

### 3. MEDIUM SEVERITY

#### 3.1 Unused Dependencies
**Severity**: Medium | **Files**: `frontend/package.json`

**Dependencies**: `better-auth`, `@tanstack/react-query` (installed but unused)

**Why It's Debt**: Bloats bundle size (~40KB), confusing to developers (which auth system do we use?).

**Fix** (10 min): Remove unused packages.

---

#### 3.2 Redundant Auth Store State
**Severity**: Medium | **File**: `frontend/store/auth.ts`

```typescript
isAuthenticated: boolean  // Redundant - can derive from `user`
```

**Why It's Debt**: Two sources of truth, requires manual sync. Can get out of sync.

**Fix** (30 min): Remove `isAuthenticated`, use selector `!!state.user`.

---

## Part 3: Infrastructure Debt

### 1. CRITICAL (Fix Before Production)

#### 1.1 Exposed Redis Without Authentication
**Severity**: Critical | **File**: `docker-compose.yml:25-26`

```yaml
ports:
  - "6379:6379"  # Exposed on all interfaces, no AUTH
```

**Why It's Debt**: Security vulnerability. Anyone with network access can read/write Redis data (session data, pub/sub messages).

**Fix** (15 min):
1. Remove port mapping or bind to 127.0.0.1
2. Enable `--requirepass` in Redis command

---

#### 1.2 Weak Defaults in docker-compose.yml
**Severity**: Critical | **File**: `docker-compose.yml`

```yaml
POSTGRES_PASSWORD: devpass
JWT_SECRET=dev-secret-change-in-production
```

**Why It's Debt**: If committed to git, credentials exposed in version control forever.

**Fix** (30 min):
1. Add `docker-compose.yml` to `.gitignore`
2. Create `docker-compose.example.yml` with placeholders
3. Document setup in README

---

### 2. HIGH SEVERITY

#### 2.1 Missing Resource Limits
**Severity**: High | **File**: `docker-compose.yml`

**Why It's Debt**: Services can consume unlimited host resources. One memory leak takes down entire machine.

**Fix** (30 min): Add CPU/memory limits to all services.

---

#### 2.2 No Backend Healthcheck
**Severity**: High | **File**: `docker-compose.yml`

**Why It's Debt**: Backend has `/health` endpoint but docker-compose doesn't use it. Container won't auto-restart on failure.

**Fix** (15 min): Add healthcheck configuration.

---

#### 2.3 Mixed Dev/Prod in Single Docker Compose
**Severity**: High | **File**: `docker-compose.yml`

**Why It's Debt**: MailHog (dev tool) in same file as production services. Risk of accidentally deploying dev tools to production.

**Fix** (1 hour): Split into `docker-compose.dev.yml` and `docker-compose.prod.yml`.

---

## Part 4: Prioritized Fix Order

### Fix Today (30 minutes)

```bash
# 1. JWT secret - fail fast if missing (2 min)
# backend/cmd/server/main.go:49-52

jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    log.Fatal("JWT_SECRET environment variable required")
}

# 2. CORS from env (10 min)
# backend/cmd/server/main.go:41-46

allowOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
if allowOrigins == "" {
    allowOrigins = "http://localhost:3000"
}

# 3. Redis AUTH (15 min)
# docker-compose.yml

command: redis-server --requirepass ${REDIS_PASSWORD:-devpass}
ports:
  - "127.0.0.1:6379:6379"  # Bind to localhost only
```

### Fix This Week (4 hours)

1. **Replace `window.location` with `router.push()`** - 1 hour
2. **Extract duplicate auth form components** - 3 hours

### Fix When Convenient (1-2 days)

1. Refactor global DB state to dependency injection - 2 hours
2. Centralize token storage abstraction - 2 hours
3. Split docker-compose into dev/prod - 1 hour
4. Add resource limits to docker-compose - 30 min
5. Add healthcheck to docker-compose - 15 min
6. Fix TypeScript error types - 1 hour
7. Remove unused dependencies - 10 min
8. Fix auth store redundancy - 30 min

---

## Part 5: Prevention Going Forward

### Before Merging Code

- [ ] No `window.location` navigation (use `router.push()`)
- [ ] No hardcoded secrets or fallback defaults
- [ ] No new global variables
- [ ] Environment variables required at startup (fail fast)
- [ ] Docker ports bound to 127.0.0.1 unless explicitly public

### Code Review Checklist

- [ ] Does this work with Docker networking (not localhost)?
- [ ] Are credentials NOT hardcoded?
- [ ] Will this work in production (not just dev)?
- [ ] Can this be tested without global state?

---

## Summary

**12 actual technical debt items**, **~3-4 days** to fix all.

The rest of what was identified in the full scan (tests, CI/CD, monitoring, docs) is normal for Sprint 0 and should be addressed per your existing plan timeline.

**Recommendation**: Fix the 3 critical items today (30 min), then continue with Sprint 0. Pick up the remaining items during appropriate phases of your plan.
