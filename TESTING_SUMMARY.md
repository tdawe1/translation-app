# Testing Strategy - Executive Summary

**Project**: GengoWatcher SaaS (translation-app)
**Analysis Date**: 2025-12-28
**Status**: CRITICAL - No functional tests

---

## Critical Finding

**0% Effective Test Coverage** despite 16 test files present.

### Root Cause
- Tests written for **Python/FastAPI** backend
- Actual backend is **Go/Fiber** (2,909 lines)
- Tests import non-existent `src.gengowatcher` module
- All tests fail with `ModuleNotFoundError`

---

## Current State

| Component | LOC | Test Files | Test Count | Coverage |
|-----------|-----|------------|------------|----------|
| Go Backend | 2,909 | 0 | 0 | 0% |
| TS Frontend | ~1,415 | 0 | 0 | 0% |
| Python Tests | - | 2 | 16 | **BROKEN** |
| **TOTAL** | **4,324** | **2** | **16** | **0%** |

### Test Breakdown (All Broken)
```
tests/auth/test_auth_service.py    8 tests (imports fail)
tests/auth/test_auth_security.py   7 tests (imports fail)
tests/conftest.py                  1 fixture (wrong module)
```

---

## Required Testing

### Unit Tests (197 tests needed)

**Backend (138 tests)**:
- TokenService: 12 tests
- UserService: 18 tests
- AuthHandler: 15 tests
- JobProcessor: 12 tests
- StateManager: 8 tests
- WatcherManager: 20 tests
- RSSMonitor: 15 tests
- WebSocketMonitor: 18 tests
- JWT Middleware: 10 tests
- Config/Errors: 10 tests

**Frontend (59 tests)**:
- API Client: 12 tests
- Auth Store: 8 tests
- Login Page: 10 tests
- Register Page: 10 tests
- Dashboard: 6 tests
- Auth Provider: 8 tests
- Other Components: 5 tests

### Integration Tests (65 tests needed)

- HTTP Handlers: 30 tests
- Database Operations: 20 tests
- Redis Pub/Sub: 15 tests

### E2E Tests (10 scenarios needed)

1. Registration → Email → Dashboard
2. Login → Start Watcher → Jobs
3. Configure → Save → Restart
4. Logout → Login → Resume
5. Subscription → Features
6. Multi-User Isolation
7. WebSocket Reconnect
8. RSS Update → Filter → Notify
9. Payment Failed → Downgrade
10. API Key Auth

---

## Security Testing (47 tests)

### Authentication (20 tests)
- Password hashing (Argon2id/bcrypt)
- JWT expiry/validation
- httpOnly cookie security
- Rate limiting/lockout

### Authorization (12 tests)
- User isolation enforcement
- API key hashing
- Admin access control

### Input Validation (15 tests)
- SQL injection prevention
- XSS protection
- Command injection prevention

---

## Performance Testing

### Load Tests
- 1,000 logins/second
- 500 watcher operations/second
- 10,000 concurrent WebSocket connections
- 100 RSS feeds polled simultaneously

### Benchmarks
- Token generation: <1ms
- Password hashing: <100ms
- DB queries: <10ms

---

## Implementation Plan

### Week 1: Foundation (CRITICAL)
- [ ] Delete invalid Python tests
- [ ] Setup Go test infrastructure
- [ ] Write TokenService tests (12)
- [ ] Write UserService tests (18)
- [ ] Write AuthHandler tests (15)

**Deliverable**: 45 passing unit tests, auth package >80% coverage

### Week 2: Core Watcher Tests
- [ ] Write JobProcessor tests (12)
- [ ] Write StateManager tests (8)
- [ ] Write WatcherManager tests (20)
- [ ] Write RSSMonitor tests (15)
- [ ] Write WebSocketMonitor tests (18)

**Deliverable**: 73 passing unit tests, watcher package >80% coverage

### Week 3: Frontend Tests
- [ ] Setup Vitest + React Testing Library
- [ ] Write API client tests (12)
- [ ] Write auth store tests (8)
- [ ] Write login/register page tests (20)
- [ ] Write dashboard tests (6)

**Deliverable**: 46 passing unit tests, frontend >75% coverage

### Week 4: Integration Tests
- [ ] Setup testcontainers
- [ ] Write auth API tests (15)
- [ ] Write watcher API tests (15)
- [ ] Write database tests (20)
- [ ] Write Redis tests (15)

**Deliverable**: 65 passing integration tests

### Week 5: E2E Tests
- [ ] Setup Playwright
- [ ] Write registration flow (3)
- [ ] Write login flow (2)
- [ ] Write watcher flow (2)
- [ ] Write billing flow (2)

**Deliverable**: 10 passing E2E scenarios

---

## Effort Estimate

| Phase | Duration | Tests | Deliverable |
|-------|----------|-------|-------------|
| **Foundation** | 1 week | 45 | Auth tests passing |
| **Watcher** | 1 week | 73 | Watcher tests passing |
| **Frontend** | 1 week | 46 | Frontend tests passing |
| **Integration** | 1 week | 65 | API integration tests |
| **E2E** | 1 week | 10 scenarios | Critical flows covered |
| **TOTAL** | **5 weeks** | **239** | Production-ready coverage |

---

## Immediate Actions

### Today (1 hour)

```bash
# 1. Delete broken Python tests
rm -rf tests/
rm pytest.ini
rm requirements-dev.txt
```

### This Week (5 days)

```bash
# 2. Create test infrastructure
mkdir -p backend/tests
mkdir -p backend/tests/mocks
mkdir -p backend/tests/fixtures

# 3. Install Go testing dependencies (testify already installed)
# 4. Write first test file
touch backend/internal/auth/token_test.go

# 5. Run tests
cd backend
go test -v ./internal/auth/...
```

---

## Coverage Targets

| Metric | Current | Week 3 | Week 5 |
|--------|---------|--------|--------|
| Backend Unit | 0% | 80% | 85% |
| Frontend Unit | 0% | 0% | 75% |
| Integration | 0% | 0% | 60% |
| E2E Critical Paths | 0% | 0% | 100% |

---

## Quality Gates

### Before Merge
- All tests passing
- Coverage not decreased
- No new security vulnerabilities
- Performance benchmarks not regressed

### CI/CD Pipeline
- Unit tests: <5 minutes
- Integration tests: <10 minutes
- E2E tests: <15 minutes
- Total: <30 minutes

---

## Success Metrics

| Metric | Target | Status |
|--------|--------|--------|
| Tests Passing | 239 | 0/239 (0%) |
| Backend Coverage | 80% | 0% |
| Frontend Coverage | 75% | 0% |
| E2E Scenarios | 10 | 0 |
| Security Tests | 47 | 0 |
| Performance Benchmarks | 5 | 0 |

---

## Why Python Tests Exist

**Evidence of Plan Change**:

1. **README.md** claims:
   ```
   Tech Stack: FastAPI, SQLAlchemy 2.0 async
   ```

2. **requirements.txt** includes:
   ```
   fastapi, uvicorn, sqlalchemy, alembic, redis
   ```

3. **Actual backend**:
   ```
   2,909 lines of Go code using Fiber/GORM
   ```

**Conclusion**: Tests written for FastAPI backend before rewrite to Go. Tests not updated after architectural change.

---

## Recommended Testing Stack

### Backend (Go)
- **Framework**: `testify` (already installed)
- **Mocking**: `testify/mock`
- **Integration**: `testcontainers-go`
- **Coverage**: `go test -coverprofile`

### Frontend (TypeScript)
- **Framework**: Vitest (faster than Jest)
- **Components**: React Testing Library
- **E2E**: Playwright (cross-browser)

### CI/CD
- **Platform**: GitHub Actions
- **Coverage**: Codecov
- **Artifacts**: Test reports, screenshots

---

## Next Steps

1. **Review this report** with team
2. **Approve testing strategy** and timeline
3. **Delete Python tests** to prevent confusion
4. **Start Week 1 tasks** (foundation + auth tests)
5. **Setup CI/CD pipeline** after first tests pass

---

**Report Version**: 1.0
**Last Updated**: 2025-12-28
**Author**: Test Engineering Analysis
