# Testing Strategy & Implementation Analysis
**Translation-App (GengoWatcher SaaS)**
**Date**: 2025-12-28
**Analysis Scope**: Complete testing infrastructure, coverage, and quality assessment

---

## Executive Summary

**Current Test Coverage**: 0% effective (despite 16 tests present)

**Critical Finding**: All existing Python tests target non-existent `src.gengowatcher` module, while the actual backend is written in Go. This is a fundamental architectural mismatch that renders the entire test suite invalid.

**Test Pyramid Status**:
- Unit Tests: 0% (backend has 0 Go test files, frontend has 0 test files)
- Integration Tests: 0% (no API handler tests)
- E2E Tests: 0% (no end-to-end test scenarios)

**Lines of Code**: 2,909 (Go backend) + ~1,415 (TypeScript frontend) = 4,324 total LOC

**Recommendation**: Complete test infrastructure rebuild required. Existing Python tests should be deleted and rewritten in Go.

---

## Part 1: Current Test Architecture Analysis

### 1.1 Backend Testing Status (Go)

**Actual Backend**: Go (Fiber web framework, GORM, Redis)
**Current Test Count**: 0 files
**Test Framework Available**: `testify` (v1.11.1) - installed via dependencies but unused
**Additional Frameworks**: `ginkgo` (v2.12.0), `gomega` (v1.27.10) - installed via Redis but unused

**Backend Structure** (18 Go files):
```
backend/internal/
├── auth/
│   ├── token.go (85 lines) - JWT token service
│   └── user_service.go (191 lines) - User business logic
├── config/
│   └── config.go (98 lines) - Configuration management
├── database/
│   └── database.go (103 lines) - Database interface
├── errors/
│   └── errors.go (56 lines) - Typed error codes
├── handlers/
│   ├── auth.go (155 lines) - HTTP auth handlers
│   ├── lemonsqueezy.go (258 lines) - Billing handlers
│   ├── response.go (95 lines) - Response helpers
│   └── watcher.go (305 lines) - Watcher HTTP handlers
├── middleware/
│   └── jwt.go (174 lines) - JWT middleware
├── models/
│   ├── database.go (115 lines) - GORM models
│   └── user.go (153 lines) - User model
└── watcher/
    ├── constants.go (59 lines) - Constants
    ├── job_processor.go (131 lines) - Job processing
    ├── manager.go (236 lines) - Watcher orchestration
    ├── rss.go (307 lines) - RSS monitoring
    ├── state_manager.go (74 lines) - State DB operations
    └── websocket.go (314 lines) - WebSocket monitoring
```

**Testability Assessment**:
- Dependency Injection: ✅ Implemented (Database interface)
- Service Layer Separation: ✅ Implemented (TokenService, UserService)
- Test-Friendly Design: ✅ Post-refactoring SOLID principles applied
- Mock Support: ✅ Database interface enables mocking

### 1.2 Frontend Testing Status (TypeScript/React)

**Framework**: Next.js 16, React 19, TypeScript 5
**Current Test Count**: 0 files
**Test Framework**: None configured (no Jest, Vitest, or Playwright)
**Frontend Structure**: 5 TypeScript/TSX files

**Frontend Components**:
```
frontend/
├── app/
│   ├── auth/login/page.tsx
│   ├── auth/register/page.tsx
│   ├── dashboard/page.tsx
│   ├── layout.tsx
│   └── page.tsx
├── components/auth/
│   ├── provider.tsx
│   └── protected-route.tsx
├── lib/
│   └── api.ts (234 lines) - API client
└── store/
    └── auth.ts (57 lines) - Zustand state
```

### 1.3 Invalid Python Test Suite

**Files Present** (16 tests, 2 test files):
```
tests/
├── conftest.py (91 lines) - Pytest fixtures
├── auth/
│   ├── test_auth_service.py (129 lines, 8 tests)
│   └── test_auth_security.py (98 lines, 7 tests)
└── database/, watcher/ (empty)
```

**Tests Present** (all broken):
1. `test_password_hashing()` - Tests Argon2id password hashing
2. `test_access_token_creation()` - Tests JWT generation
3. `test_access_token_expiry()` - Tests token expiration
4. `test_invalid_token()` - Tests invalid token handling
5. `test_refresh_token_generation()` - Tests refresh tokens
6. `test_api_key_generation()` - Tests API key creation
7. `test_api_key_verification()` - Tests API key validation
8. `test_register_user()` - Tests user registration
9. `test_register_duplicate_email()` - Tests duplicate handling
10. `test_authenticate_user()` - Tests login
11. `test_authenticate_wrong_password()` - Tests password validation
12. `test_authenticate_nonexistent_user()` - Tests non-existent user
13. `test_refresh_tokens()` - Tests token refresh flow
14. `test_get_user_by_id()` - Tests user lookup
15. `test_verify_email()` - Tests email verification
16. `test_change_password()` - Tests password change

**Import Errors**:
```python
# These imports fail - no src/gengowatcher module exists
from src.gengowatcher.auth.security import ...
from src.gengowatcher.auth.service import AuthService
from src.gengowatcher.database.models import Base
```

**Why These Tests Exist**:
The project README and documentation reference FastAPI/SQLAlchemy as the tech stack, suggesting a **plan change mid-development**. The tests were likely written before the backend was rewritten in Go.

**Evidence of Plan Change**:
- README.md (line 20-26): Claims "FastAPI, SQLAlchemy 2.0 async"
- pytest.ini: Configures `--cov=src/gengowatcher` (non-existent path)
- requirements.txt: Includes `fastapi`, `uvicorn`, `sqlalchemy`, `alembic`
- Actual backend: 2,909 lines of Go code using Fiber/GORM

---

## Part 2: Test Quality Metrics (Current State)

### 2.1 Assertion Density

**Python Tests (Invalid)**: 43 assertions / 2,277 lines of backend code = **0 assertions per 100 LOC**

**Go Tests (Non-existent)**: 0 assertions / 2,909 lines = **0 assertions per 100 LOC**

**Frontend Tests (Non-existent)**: 0 assertions / ~1,415 lines = **0 assertions per 100 LOC**

**Industry Benchmark**: 5-10 assertions per 100 LOC
**Current Deviation**: -100% from benchmark

### 2.2 Test Isolation

**Current State**: Not applicable (no runnable tests)

**Required Isolation Patterns**:
- Database: Mock `database.Database` interface
- Redis: Mock `redis.Client` or use testcontainers
- HTTP: Use httptest for Fiber handlers
- Time: Mock `time.Now()` for token expiry tests
- External APIs: Mock Gengo API calls

### 2.3 Test Maintenance Risk

**Current Risk Level**: CRITICAL

**Risk Factors**:
1. Architectural mismatch between tests and implementation
2. No test infrastructure in place
3. No test coverage reporting
4. No CI/CD integration for tests
5. Missing test fixtures and factories

### 2.4 Test Flakiness Potential

**Current Assessment**: Unknown (no tests running)

**Future Flakiness Risks**:
1. **WebSocket Tests**: Timing-dependent connection tests
2. **RSS Monitor Tests**: Network-dependent feed parsing
3. **Redis Tests**: Pub/sub race conditions
4. **Concurrent Watcher Tests**: Goroutine synchronization
5. **JWT Time Tests**: Clock-dependent expiry logic

**Mitigation Strategies**:
- Use `testify/assert` with timeouts
- Mock all network I/O
- Use deterministic time in tests
- Avoid real concurrency in unit tests

---

## Part 3: Testing Gap Analysis

### 3.1 Unit Test Gaps

**Coverage Target**: 80% (standard for production SaaS)
**Current Coverage**: 0%
**Gap**: 80% coverage needed across all modules

#### Backend Unit Tests Needed (Go)

| Module | Complexity | Priority | Test Cases Needed |
|--------|-----------|----------|-------------------|
| **TokenService** | Medium | HIGH | 12 tests |
| UserService | High | HIGH | 18 tests |
| AuthHandler | Medium | HIGH | 15 tests |
| JobProcessor | Medium | HIGH | 12 tests |
| StateManager | Low | MEDIUM | 8 tests |
| WatcherManager | High | MEDIUM | 20 tests |
| RSSMonitor | High | MEDIUM | 15 tests |
| WebSocketMonitor | High | MEDIUM | 18 tests |
| JWT Middleware | Medium | MEDIUM | 10 tests |
| Config | Low | LOW | 6 tests |
| Errors | Low | LOW | 4 tests |
| **Total** | | | **138 unit tests** |

#### Frontend Unit Tests Needed (TypeScript)

| Component | Complexity | Priority | Test Cases Needed |
|-----------|-----------|----------|-------------------|
| **API Client** | Medium | HIGH | 12 tests |
| Auth Store | Low | HIGH | 8 tests |
| Login Page | Medium | MEDIUM | 10 tests |
| Register Page | Medium | MEDIUM | 10 tests |
| Dashboard Page | Low | MEDIUM | 6 tests |
| Protected Route | Low | MEDIUM | 5 tests |
| Auth Provider | Medium | HIGH | 8 tests |
| **Total** | | | **59 unit tests** |

### 3.2 Integration Test Gaps

**Missing Test Categories**:

1. **HTTP Handler Integration Tests** (30 tests needed)
   - POST /api/v1/auth/register - success, validation, errors
   - POST /api/v1/auth/login - success, wrong password, not found
   - GET /api/v1/me - authenticated, not authenticated
   - POST /api/v1/watcher/start - start, already running
   - POST /api/v1/watcher/stop - stop, not running
   - GET /api/v1/watcher/config - get, update
   - POST /api/v1/watcher/config - validation, persistence
   - POST /api/v1/webhooks/lemonsqueezy - signature verification

2. **Database Integration Tests** (20 tests needed)
   - User creation with transactions
   - Watcher config/state relationships
   - Cascade delete operations
   - Concurrent user registration
   - Migration testing

3. **Redis Integration Tests** (15 tests needed)
   - Job deduplication via sets
   - Pub/sub event delivery
   - State persistence
   - Connection failure handling

### 3.3 End-to-End Test Gaps

**Critical User Journeys** (10 scenarios needed):

1. **Registration → Email Verification → Dashboard**
2. **Login → Start Watcher → Job Notification**
3. **Login → Configure Watcher → Save Settings**
4. **Logout → Login → Resume Session**
5. **Subscription Upgrade → Feature Unlock**
6. **Multiple Users → Isolated Watchers**
7. **WebSocket Reconnection → State Recovery**
8. **RSS Feed Update → Job Filter → Notification**
9. **Payment Failed → Subscription Downgrade**
10. **API Key Authentication → Watcher Control**

**Current E2E Tests**: 0

---

## Part 4: Test Infrastructure Requirements

### 4.1 Backend Test Infrastructure (Go)

**Required Setup**:

```go
// tests/setup.go
package tests

import (
    "testing"
    "github.com/stretchr/testify/suite"
    "github.com/tdawe1/translation-app/internal/database"
    "github.com/tdawe1/translation-app/internal/config"
)

type TestSuite struct {
    suite.Suite
    DB    database.Database
    Config *config.Config
    MockRedis *MockRedisClient
}

func (s *TestSuite) SetupTest() {
    // Initialize test database
    s.DB = setupTestDB()
    s.Config = config.LoadTestConfig()
    s.MockRedis = NewMockRedisClient()
}

func (s *TestSuite) TearDownTest() {
    // Clean up database
    cleanupTestDB(s.DB)
}
```

**Mock Implementations Needed**:

1. **MockDatabase** (implements `database.Database`)
```go
type MockDatabase struct {
    mock.Mock
    Users []models.User
    // ... other methods
}

func (m *MockDatabase) Create(value interface{}) *gorm.DB {
    args := m.Called(value)
    return args.Get(0).(*gorm.DB)
}
```

2. **MockRedisClient** (implements redis interface)
```go
type MockRedisClient struct {
    mock.Mock
    Sets map[string]map[string]struct{}
    Channels map[string]chan string
}
```

3. **Test Fixtures** (user, config, state factories)
```go
func CreateTestUser(email string) *models.User {
    return &models.User{
        Email: email,
        IsActive: true,
        EmailVerified: false,
    }
}
```

**Test Configuration**:
```go
// internal/config/test.go
func LoadTestConfig() *Config {
    return &Config{
        Env: "test",
        JWTSecret: "test-secret-key-32-bytes-long!",
        DatabaseURL: "sqlite::memory:",
        RedisURL: "", // Mocked
        Port: "0", // Random port for tests
    }
}
```

### 4.2 Frontend Test Infrastructure (TypeScript)

**Framework Selection**: Vitest (faster than Jest, native ESM)

**Setup Required**:

```typescript
// vitest.config.ts
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: ['node_modules/', 'src/test/'],
    },
  },
})
```

**Test Utilities**:

```typescript
// src/test/setup.ts
import { vi } from 'vitest'

// Mock fetch globally
global.fetch = vi.fn()

// Mock window.location
Object.defineProperty(window, 'location', {
  value: {
    href: '',
  },
  writable: true,
})
```

**Test Factories**:

```typescript
// src/test/factories/user.ts
export const createMockUser = (overrides?: Partial<User>): User => ({
  id: '00000000-0000-0000-0000-000000000001',
  email: 'test@example.com',
  email_verified: false,
  is_active: true,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
  ...overrides,
})
```

### 4.3 Integration Test Infrastructure

**Required Components**:

1. **Testcontainers for Go**
   - PostgreSQL container for real DB tests
   - Redis container for pub/sub tests
   - Network isolation between tests

2. **HTTP Test Server**
```go
func setupTestApp(handler *handlers.AuthHandler) *fiber.App {
    app := fiber.New(fiber.Config{
        DisableStartupMessage: true,
    })
    app.Post("/api/v1/auth/register", handler.Register)
    return app
}
```

3. **Test Data Management**
   - Database seeding helpers
   - Transaction rollback after each test
   - Unique ID generation for parallel tests

### 4.4 E2E Test Infrastructure

**Framework**: Playwright (supports multiple browsers, network mocking)

**Configuration**:
```typescript
// playwright.config.ts
import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  retries: process.env.CI ? 2 : 0,
  use: {
    baseURL: 'http://localhost:8000',
    trace: 'on-first-retry',
  },
  projects: [
    { name: 'chromium' },
    { name: 'firefox' },
    { name: 'webkit' },
  ],
  webServer: {
    command: 'go run cmd/server/main.go',
    port: 8000,
  },
})
```

---

## Part 5: Test Implementation Priority

### Phase 1: Critical Foundation (Week 1)

**Goal**: Enable basic unit testing for core services

**Tasks**:
1. ✅ Delete invalid Python tests (2 files, 16 tests)
2. ✅ Create Go test infrastructure (suite, mocks, fixtures)
3. ✅ Write TokenService tests (12 tests)
4. ✅ Write UserService tests (18 tests)
5. ✅ Write AuthHandler tests (15 tests)

**Deliverables**:
- `backend/internal/auth/token_test.go`
- `backend/internal/auth/user_service_test.go`
- `backend/internal/handlers/auth_test.go`
- `backend/tests/setup.go` (test suite, mocks)
- `backend/tests/fixtures.go` (test factories)

**Success Criteria**:
- All auth unit tests passing
- Coverage: auth package >80%
- CI pipeline running tests

### Phase 2: Core Watcher Tests (Week 2)

**Goal**: Test watcher business logic

**Tasks**:
1. ✅ Write JobProcessor tests (12 tests)
2. ✅ Write StateManager tests (8 tests)
3. ✅ Write WatcherManager tests (20 tests)
4. ✅ Write RSSMonitor tests (15 tests)
5. ✅ Write WebSocketMonitor tests (18 tests)

**Deliverables**:
- `backend/internal/watcher/job_processor_test.go`
- `backend/internal/watcher/state_manager_test.go`
- `backend/internal/watcher/manager_test.go`
- `backend/internal/watcher/rss_test.go`
- `backend/internal/watcher/websocket_test.go`

**Success Criteria**:
- All watcher unit tests passing
- Coverage: watcher package >80%
- Mocked RSS/WebSocket I/O

### Phase 3: Frontend Testing (Week 3)

**Goal**: Test React components and state management

**Tasks**:
1. ✅ Setup Vitest + React Testing Library
2. ✅ Write API client tests (12 tests)
3. ✅ Write auth store tests (8 tests)
4. ✅ Write login/register page tests (20 tests)
5. ✅ Write dashboard tests (6 tests)

**Deliverables**:
- `frontend/vitest.config.ts`
- `frontend/src/test/setup.ts`
- `frontend/lib/api.test.ts`
- `frontend/store/auth.test.ts`
- `frontend/app/auth/login/page.test.tsx`
- `frontend/app/auth/register/page.test.tsx`
- `frontend/app/dashboard/page.test.tsx`

**Success Criteria**:
- All frontend unit tests passing
- Coverage: >75% (excluding node_modules)
- Component mocking working

### Phase 4: Integration Tests (Week 4)

**Goal**: Test HTTP handlers with real database

**Tasks**:
1. ✅ Setup testcontainers
2. ✅ Write auth API integration tests (15 tests)
3. ✅ Write watcher API integration tests (15 tests)
4. ✅ Write database integration tests (20 tests)
5. ✅ Write Redis integration tests (15 tests)

**Deliverables**:
- `backend/tests/integration/auth_api_test.go`
- `backend/tests/integration/watcher_api_test.go`
- `backend/tests/integration/database_test.go`
- `backend/tests/integration/redis_test.go`

**Success Criteria**:
- All integration tests passing
- Tests run in parallel
- Database cleaned after each test

### Phase 5: E2E Tests (Week 5)

**Goal**: Test critical user journeys

**Tasks**:
1. ✅ Setup Playwright
2. ✅ Write registration flow test (3 scenarios)
3. ✅ Write login flow test (2 scenarios)
4. ✅ Write watcher control flow test (2 scenarios)
5. ✅ Write subscription flow test (2 scenarios)

**Deliverables**:
- `e2e/registration.spec.ts`
- `e2e/login.spec.ts`
- `e2e/watcher.spec.ts`
- `e2e/billing.spec.ts`

**Success Criteria**:
- All E2E tests passing on Chrome
- CI runs E2E tests before deployment
- Screenshots/videos on failure

---

## Part 6: Security Testing Requirements

### 6.1 Authentication Security Tests

**Required Tests** (20 tests):

1. **Password Security**
   - ✅ Password hashing uses Argon2id/bcrypt
   - ✅ Password verification timing attack resistant
   - ✅ Weak password rejected (<8 characters)
   - ✅ Password hash different from plaintext
   - ✅ Same password hashes differently (salt)

2. **JWT Security**
   - ✅ Token expiry enforced (15 minutes)
   - ✅ Invalid signature rejected
   - ✅ Expired token rejected
   - ✅ Token tampering detected
   - ✅ Secret key required (no weak defaults)

3. **Session Security**
   - ✅ httpOnly cookie prevents XSS access
   - ✅ Secure flag enabled in production
   - ✅ SameSite protection against CSRF
   - ✅ Cookie cleared on logout
   - ✅ Session invalidation after password change

4. **Authentication Failures**
   - ✅ Rate limiting on login attempts
   - ✅ Account lockout after N failures
   - ✅ Generic error messages (no user enumeration)
   - ✅ Email verification required before access
   - ✅ Inactive accounts cannot authenticate

### 6.2 Authorization Security Tests

**Required Tests** (12 tests):

1. **User Isolation**
   - ✅ Users cannot access other users' data
   - ✅ Watcher instances isolated per user
   - ✅ Redis keys namespaced by user_id
   - ✅ Database queries filtered by user_id

2. **API Key Security**
   - ✅ API key hashed before storage (SHA256)
   - ✅ Raw key never stored
   - ✅ API key shows prefix only (`gengo_sk_...`)
   - ✅ Revoked keys cannot authenticate

3. **Admin Privileges**
   - ✅ Regular users cannot access admin endpoints
   - ✅ Privilege escalation prevented
   - ✅ Admin actions logged

### 6.3 Input Validation Security Tests

**Required Tests** (15 tests):

1. **SQL Injection Prevention**
   - ✅ Email inputs sanitized
   - ✅ GORM parameterized queries used
   - ✅ Raw SQL avoided or escaped

2. **XSS Prevention**
   - ✅ User input escaped in responses
   - ✅ Content-Type headers prevent MIME sniffing
   - ✅ CSRF tokens on state-changing operations

3. **Command Injection**
   - ✅ RSS feed URLs validated
   - ✅ Gengo credentials not executed
   - ✅ File paths sanitized

### 6.4 Dependency Security Tests

**Required Tests**:
- ✅ Run `go list -json -m all | nancy sleuth` for Go CVEs
- ✅ Run `npm audit` for frontend vulnerabilities
- ✅ Check `git ls-remote --heads https://github.com/advisories` for advisories

### 6.5 Configuration Security Tests

**Required Tests**:
- ✅ JWT_SECRET required in production (fail-fast)
- ✅ Database credentials not hardcoded
- ✅ CORS origins validated
- ✅ Redis AUTH enabled
- ✅ No debug logging in production

---

## Part 7: Performance Testing Requirements

### 7.1 Load Testing Scenarios

**Required Tests** (using k6 or vegeta):

1. **Authentication Endpoints**
   - Target: 1000 logins/second
   - Duration: 5 minutes
   - Max latency: P95 <200ms, P99 <500ms
   - Error rate: <0.1%

2. **Watcher Control Endpoints**
   - Target: 500 start/stop operations/second
   - Duration: 10 minutes
   - Max latency: P95 <100ms
   - No race conditions on concurrent starts

3. **WebSocket Connections**
   - Target: 10,000 concurrent connections
   - Memory: <1GB for 10K connections
   - Message latency: P95 <50ms

4. **RSS Feed Processing**
   - Target: 100 feeds polled simultaneously
   - Processing time: <5s per feed
   - No duplicate job notifications

### 7.2 Stress Testing

**Required Tests**:
- ✅ Database connection pool exhaustion
- ✅ Redis connection limits
- ✅ Memory leaks in watcher goroutines
- ✅ Goroutine leaks under load
- ✅ File descriptor limits

### 7.3 Performance Regression Tests

**Required Benchmarks** (Go):
```go
func BenchmarkTokenGeneration(b *testing.B) {
    svc := auth.NewTokenService("test-secret")
    for i := 0; i < b.N; i++ {
        _, _ = svc.GenerateAccessToken(uuid.New())
    }
}

func BenchmarkPasswordHashing(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _, _ = bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
    }
}
```

**Regression Detection**:
- Token generation: <1ms per token
- Password hashing: <100ms per hash (Argon2id/bcrypt)
- Database queries: <10ms per query (indexed)

---

## Part 8: Mock Strategy

### 8.1 Database Mocking

**Approach**: Mock `database.Database` interface for unit tests, use testcontainers for integration tests

**Implementation**:
```go
// mocks/database.go
type MockDatabase struct {
    mock.Mock
    Users map[uuid.UUID]*models.User
    mu    sync.RWMutex
}

func (m *MockDatabase) Create(value interface{}) *gorm.DB {
    m.mu.Lock()
    defer m.mu.Unlock()

    if user, ok := value.(*models.User); ok {
        m.Users[user.ID] = user
        return &gorm.DB{Error: nil}
    }
    return &gorm.DB{Error: errors.New("not a user")}
}

func (m *MockDatabase) Where(query interface{}, args ...interface{}) *gorm.DB {
    argsCalled := m.Called(query, args)
    return argsCalled.Get(0).(*gorm.DB)
}
```

### 8.2 Redis Mocking

**Approach**: Mock Redis client for pub/sub and set operations

**Implementation**:
```go
// mocks/redis.go
type MockRedisClient struct {
    mock.Mock
    Sets     map[string]map[string]struct{}
    Channels map[string]chan string
    mu       sync.RWMutex
}

func (m *MockRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.Sets[key] == nil {
        m.Sets[key] = make(map[string]struct{})
    }
    for _, member := range members {
        m.Sets[key][fmt.Sprintf("%v", member)] = struct{}{}
    }

    cmd := redis.NewIntCmd(ctx)
    cmd.SetVal(int64(len(members)))
    return cmd
}

func (m *MockRedisClient) Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
    m.mu.Lock()
    defer m.mu.Unlock()

    if ch, ok := m.Channels[channel]; ok {
        select {
        case ch <- message.(string):
        default:
            // Channel full, drop message
        }
    }

    cmd := redis.NewIntCmd(ctx)
    cmd.SetVal(1)
    return cmd
}
```

### 8.3 External API Mocking

**Approach**: Mock Gengo API calls in tests

**Implementation**:
```go
// mocks/gengo_client.go
type MockGengoClient struct {
    mock.Mock
}

func (m *MockGengoClient) GetJobs(ctx context.Context) ([]Job, error) {
    args := m.Called(ctx)
    if args.Get(0) == nil {
        return []Job{}, errors.New("API error")
    }
    return args.Get(0).([]Job), nil
}
```

### 8.4 Time Mocking

**Approach**: Use deterministic time in tests

**Implementation**:
```go
// Use fixed time for tests
func TestTokenExpiry(t *testing.T) {
    svc := auth.NewTokenService("test-secret")
    svc.Now = func() time.Time {
        return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
    }

    token, _ := svc.GenerateAccessToken(uuid.New())
    claims, _ := svc.ValidateToken(token)

    expectedExpiry := time.Date(2025, 1, 1, 0, 15, 0, 0, time.UTC)
    assert.Equal(t, expectedExpiry.Unix(), claims.ExpiresAt.Unix())
}
```

---

## Part 9: Test Data Management

### 9.1 Test Factories

**User Factory**:
```go
// fixtures/user.go
func CreateTestUser(overrides ...func(*models.User)) *models.User {
    user := &models.User{
        ID:           uuid.New(),
        Email:        "test@example.com",
        PasswordHash: "$2a$12$hashedpassword",
        IsActive:     true,
        EmailVerified: false,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }

    for _, override := range overrides {
        override(user)
    }

    return user
}

// Usage:
user := CreateTestUser(func(u *models.User) {
    u.Email = "custom@example.com"
})
```

**Watcher Config Factory**:
```go
func CreateTestConfig(overrides ...func(*models.WatcherConfig)) *models.WatcherConfig {
    config := &models.WatcherConfig{
        UserID:                  uuid.New(),
        RSSFeedURL:              "https://example.com/feed",
        WebSocketEnabled:        true,
        MinReward:               2.0,
        MaxReward:               100.0,
        IncludedLanguagePairs:   []string{"en→ja", "en→ko"},
        EnableDesktopNotifications: true,
        EnableSoundNotifications: false,
        EnableEmailNotifications: true,
        AutoAcceptEnabled: false,
    }

    for _, override := range overrides {
        override(config)
    }

    return config
}
```

### 9.2 Database Seeding

**Seed Script**:
```go
// tests/seeds.go
func SeedTestData(db database.Database) error {
    users := []*models.User{
        CreateTestUser(func(u *models.User) { u.Email = "user1@example.com" }),
        CreateTestUser(func(u *models.User) { u.Email = "user2@example.com" }),
    }

    for _, user := range users {
        if err := db.Create(user).Error; err != nil {
            return err
        }
    }

    return nil
}
```

### 9.3 Test Data Cleanup

**Transaction Rollback**:
```go
func (s *TestSuite) SetupTest() {
    s.DB = setupTestDB()
    s.Tx = s.DB.Begin()
}

func (s *TestSuite) TearDownTest() {
    s.Tx.Rollback()
}
```

---

## Part 10: CI/CD Integration

### 10.1 GitHub Actions Configuration

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  backend-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:14
        env:
          POSTGRES_PASSWORD: testpass
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"

    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.25'
    - name: Run tests
      run: |
        cd backend
        go test -v -race -coverprofile=coverage.out ./...
    - name: Upload coverage
      uses: codecov/codecov-action@v3

  frontend-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-node@v4
      with:
        node-version: '20'
    - name: Install dependencies
      run: cd frontend && npm ci
    - name: Run tests
      run: cd frontend && npm test
    - name: Upload coverage
      uses: codecov/codecov-action@v3

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Setup Go
      uses: actions/setup-go@v5
    - name: Setup Node
      uses: actions/setup-node@v4
    - name: Install Playwright
      run: npx playwright install --with-deps
    - name: Run E2E tests
      run: npx playwright test
    - uses: actions/upload-artifact@v3
      if: always()
      with:
        name: playwright-report
        path: playwright-report/
```

### 10.2 Coverage Requirements

**Gatekeepers**:
- Backend: 80% coverage (enforced via `.codecov.yml`)
- Frontend: 75% coverage
- E2E: 100% of critical paths covered

**Patch Coverage**:
- Pull requests must maintain coverage percentage
- New code must have >90% coverage

---

## Part 11: Recommendations Summary

### Immediate Actions (Week 1)

1. **Delete Invalid Python Tests**
   ```bash
   rm -rf tests/
   rm pytest.ini
   rm requirements-dev.txt
   ```

2. **Create Go Test Infrastructure**
   - Install testify (already available)
   - Create `backend/tests/` directory
   - Implement test suite and mocks

3. **Write Critical Auth Tests**
   - TokenService: 12 tests
   - UserService: 18 tests
   - AuthHandler: 15 tests

### Short-Term Goals (Month 1)

1. **Achieve 80% Unit Test Coverage** (backend + frontend)
2. **Implement Integration Tests** for all HTTP endpoints
3. **Setup CI/CD Pipeline** with automated testing

### Medium-Term Goals (Quarter 1)

1. **Implement E2E Tests** for critical user journeys
2. **Performance Testing** infrastructure and benchmarks
3. **Security Testing** automation in CI/CD

### Long-Term Goals (Ongoing)

1. **Mutation Testing** (using Go-mutesting)
2. **Chaos Engineering** (fault injection testing)
3. **Contract Testing** (for API compatibility)

---

## Part 12: Estimated Effort

| Task | Effort | Priority |
|------|--------|----------|
| **Delete Python tests** | 1 hour | CRITICAL |
| **Setup Go test infrastructure** | 1 day | CRITICAL |
| **Write backend unit tests** (138 tests) | 5 days | HIGH |
| **Write frontend unit tests** (59 tests) | 3 days | HIGH |
| **Setup integration test infrastructure** | 2 days | MEDIUM |
| **Write integration tests** (65 tests) | 5 days | MEDIUM |
| **Setup E2E test infrastructure** | 1 day | MEDIUM |
| **Write E2E tests** (10 scenarios) | 3 days | MEDIUM |
| **Security testing automation** | 2 days | HIGH |
| **Performance testing setup** | 2 days | LOW |
| **CI/CD integration** | 1 day | MEDIUM |
| **Documentation** | 1 day | LOW |
| **TOTAL** | **27 days** (~5 weeks)** | |

---

## Part 13: Success Metrics

### Coverage Targets

| Metric | Current | Target | Timeline |
|--------|---------|--------|----------|
| Backend Unit Coverage | 0% | 80% | Week 3 |
| Frontend Unit Coverage | 0% | 75% | Week 4 |
| Integration Test Coverage | 0% | 60% | Week 5 |
| E2E Critical Path Coverage | 0% | 100% | Week 6 |

### Quality Metrics

| Metric | Target |
|--------|--------|
| Test Run Time | <5 minutes (unit), <15 minutes (integration) |
| Flaky Test Rate | <0.1% |
| Test Maintenance Burden | <2 hours/week |
| Time to Debug Failure | <10 minutes (with good error messages) |

---

## Conclusion

The translation-app codebase has **zero effective test coverage** due to a fundamental architectural mismatch between the existing Python test suite (which tests a FastAPI backend that doesn't exist) and the actual Go backend implementation.

**Key Findings**:
1. 16 Python tests exist but cannot run (wrong module imports)
2. Backend has 2,909 lines of Go code with 0 tests
3. Frontend has ~1,415 lines of TypeScript with 0 tests
4. No test infrastructure (frameworks, mocks, fixtures) exists
5. Testify framework available in Go but unused
6. No frontend testing framework configured

**Recommended Approach**:
1. Delete all Python tests immediately
2. Build Go test infrastructure from scratch
3. Prioritize auth service tests (critical for security)
4. Add frontend testing with Vitest
5. Implement integration tests with testcontainers
6. Add E2E tests with Playwright for critical flows

**Estimated Effort**: 5 weeks to reach production-ready coverage (80% backend, 75% frontend, 100% critical paths E2E).

---

**Report Generated**: 2025-12-28
**Next Review**: After Phase 1 completion (Week 1)
