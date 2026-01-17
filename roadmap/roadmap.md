---
feature: "Code Review Remediation"
spec: |
  Address all findings from comprehensive code review. Priority order: Critical security fixes (P0), High priority performance/testing/quality (P1), Medium priority architecture/CI-CD (P2). Target: Raise overall score from 62/100 to 80+/100. Constraints: No breaking changes to public APIs, maintain backwards compatibility, all changes must have tests.
---

## Task List

### Feature 1: Critical Security Fixes (P0)
Description: Immediate security vulnerabilities requiring urgent remediation
- [ ] 1.01 Remove .env files from git history and add to .gitignore
- [ ] 1.02 Remove hardcoded dev fallback secrets from backend/config, implement fail-fast on missing secrets
- [ ] 1.03 Fix Redis key patterns for multi-tenancy - change trans:queue:* to user:{user_id}:trans:queue:*
- [ ] 1.04 Implement JWT blocklist in Redis for token revocation
- [ ] 1.05 Add AES-GCM envelope encryption for OAuth tokens in database

### Feature 2: Performance Fixes (P1)
Description: Critical performance issues affecting scalability
- [ ] 2.01 Fix N+1 query in translation.go:604-609 using SQL aggregation with GROUP BY
- [ ] 2.02 Add channel drain with select/timeout to watcher/manager.go unbounded job channel
- [ ] 2.03 Add React.memo to job card components to prevent re-render thrashing
- [ ] 2.04 Fix WebSocket reconnection race condition with connection state machine

### Feature 3: Test Coverage (P1)
Description: Critical testing gaps requiring immediate attention
- [ ] 3.01 Add auth unit tests for Go backend (target 80% coverage on auth package)
- [ ] 3.02 Fix 11 failing Python MultiModelTranslator tests with proper mocks
- [ ] 3.03 Add watcher unit tests for Go backend
- [ ] 3.04 Add basic E2E tests for critical auth and translation flows

### Feature 4: Code Quality (P1)
Description: Code quality issues affecting maintainability
- [ ] 4.01 Fix empty catch blocks in frontend/lib/api/client.ts with proper error logging
- [ ] 4.02 Refactor QueueConsumer god object in job_queue/consumer.py into focused classes
- [ ] 4.03 Split 1068-line test file tests/test_review/test_cli.py by test category
- [ ] 4.04 Extract duplicated store logic into useAsyncAction hook

### Feature 5: CI/CD Pipeline (P2)
Description: DevOps infrastructure for automated quality gates
- [ ] 5.01 Create GitHub Actions CI workflow with test, lint, build stages
- [ ] 5.02 Add dependency scanning with Snyk or Trivy to CI pipeline
- [ ] 5.03 Add SAST with CodeQL to CI pipeline
- [ ] 5.04 Set up Sentry for error tracking integration

### Feature 6: Architecture Improvements (P2)
Description: Structural improvements for long-term maintainability
- [ ] 6.01 Add rate limiting middleware on auth endpoints
- [ ] 6.02 Standardize password policy to 12 chars minimum across all validators
- [ ] 6.03 Add OpenAPI/Swagger specification for REST API
- [ ] 6.04 Update AGENTS.md to reflect WebSocket implementation (fix SSE spec mismatch)
