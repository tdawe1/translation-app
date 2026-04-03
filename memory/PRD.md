# Gengo Style Guide Integration — PRD

## Original Problem Statement
Make the translation worker apply the Gengo style guide during real queued job execution, not just in isolated modules and CLI flows.

## Architecture
- Python translation worker in `backend/cmd/translation-worker/`
- Uses LLM providers (OpenAI, Anthropic, Gemini) for translation + judging
- Redis job queue for horizontal scaling
- Style guide: markdown → parser → system prompt → provider injection
- Style checker: regex-based post-translation validation → segment flagging
- Metrics: JobMetrics dataclass → JSON stdout + Prometheus counters/histograms/gauges
- Prometheus: HTTP `/metrics` endpoint on configurable port (default 9090)

## What's Been Implemented

### Phase 1 — Worker Startup Runtime Construction
- `load_style_guide_prompt()`, `_resolve_api_key()`, `build_translation_provider()`, `build_judge_provider()`, `build_style_checker()`, `build_workflow()` in `main.py`
- `QueueConsumer` receives real workflow; missing API keys fail startup clearly
- Fixed `config.example.toml` and `config.toml` (removed hardcoded path, style_guide disabled by default)

### Phase 2 — Style Checker in Workflow
- `TranslationWorkflow` accepts optional `style_checker`
- Runs `StyleChecker.check()` on `segment.target` after winner selection
- Violations flag segments for review with `"Style: {category} - {message}"`
- `style_issues` field on `TranslationSegment`, included in CSV export

### Phase 2b — Structured Metrics
- `review/metrics.py`: `JobMetrics` dataclass (violations by category, flag rate, duration)
- JSON emitted to stdout as `[METRICS] {...}` after each job
- `workflow.last_metrics` for programmatic access

### Phase 3 — Integration Tests
- `tests/test_queue/test_consumer_gengo.py`: 7 queue consumer tests
- `tests/test_integration/test_gengo_integration.py`: 3 workflow-level tests
- `tests/test_review/test_workflow.py`: 6 style checker + 5 metrics tests
- `tests/test_review/test_metrics.py`: 12 JobMetrics unit tests

### Phase 4 — Docs/Config Cleanup
- `GENGO_STYLE_GUIDE.md`: rewritten with execution paths, startup sequence, metrics format
- `config.example.toml`: portable, style guide disabled by default

### Phase 5 — Verification
- 197 tests passing, 0 failures

### Prometheus Metrics
- `review/prometheus.py`: Counters (jobs, segments, violations by category), Histograms (duration, quality score), Gauges (flag rate, violation rate, active jobs, style_guide enabled), Info (worker metadata)
- HTTP server on configurable port via `[metrics]` config section
- Wired into workflow (auto-update after each job) and consumer (active jobs, failures)
- 10 Prometheus-specific tests

### Code Review
- Formal review conducted, documented in `docs/CODE_REVIEW.md`
- Fixed: unused imports, return type mismatch, ACTIVE_JOBS gauge wiring
- All design decisions documented with rationale

### Test Deployment
- `scripts/test_deploy.sh`: automated verification (deps → tests → metrics endpoint)
- Verified: 197 tests pass, `/metrics` endpoint responds with all expected metrics

## Test Suite Summary
- **197 tests, 0 failures**
- Coverage: parser, prompt builder, style checker, providers, workflow, metrics, prometheus, integration, main helpers, queue consumer

## Prioritized Backlog

### P1 — Production Readiness
- Grafana dashboard template for Prometheus metrics
- `prometheus_client.multiprocess` mode for multi-worker scaling
- Histogram bucket tuning after production observation

### P2 — Feature Enhancement
- Separate judge provider/model config in config.toml
- Multiple style guide support (per language pair)
- Metrics output to Redis pub/sub for real-time dashboards
- Webhook sink for alerting on high violation rates
