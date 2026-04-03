# Gengo Style Guide Integration — PRD

## Original Problem Statement
Make the translation worker apply the Gengo style guide during real queued job execution, not just in isolated modules and CLI flows.

## Architecture
- Python translation worker in `backend/cmd/translation-worker/`
- Uses LLM providers (OpenAI, Anthropic, Gemini) for translation + judging
- Redis job queue for horizontal scaling
- Style guide parsed from markdown → system prompt → injected into LLM provider
- Style checker validates translations post-hoc (regex-based rules)
- Structured JSON metrics emitted per-job for observability

## Core Requirements (Static)
1. Worker builds a real TranslationWorkflow at startup from config + env
2. Style guide prompt injected via provider's system_prompt
3. Style checker flags violations in queued job output
4. Structured metrics (violations, latency, flag rate) emitted per job
5. Config/docs are portable (no hardcoded paths)
6. Tests cover the worker path, not just isolated modules

## User Personas
- Translation team running the worker against Gengo-sourced content
- DevOps deploying/configuring the worker via config.toml + env vars
- Quality team monitoring translation quality via metrics dashboard

## What's Been Implemented

### Phase 1 — Worker Startup Runtime Construction (Jan 2026)
- Added `load_style_guide_prompt()`, `_resolve_api_key()`, `build_translation_provider()`, `build_judge_provider()`, `build_style_checker()`, `build_workflow()` to `main.py`
- `QueueConsumer` now receives a real workflow (no more stub path in normal operation)
- Missing API keys produce clear startup error
- Fixed `config.example.toml` and `config.toml` (removed hardcoded path)
- 19 new tests in `test_main.py`, 5 pre-existing failures fixed in `test_workflow.py`

### Phase 2 — Style Checker in Workflow (Jan 2026)
- `TranslationWorkflow` accepts optional `style_checker` parameter
- After winner selection, runs `StyleChecker.check()` on `segment.target`
- Style violations flag segments for review with reason `"Style: {category} - {message}"`
- Added `style_issues` field to `TranslationSegment` (list of dicts with severity/category/message)
- `ReviewWorkflowBuilder.with_style_checker()` for builder pattern support
- CSV exporter includes `style_issues` column

### Phase 2b — Structured Metrics (Jan 2026)
- New `review/metrics.py` with `JobMetrics` dataclass
- Tracks: violation count, violations by category, flag rate, processing duration
- JSON-serializable via `.to_dict()` / `.to_json()`
- Emitted to stdout as `[METRICS] {...}` after each job
- `QueueConsumer` also emits metrics in progress messages
- `workflow.last_metrics` available for programmatic access

### Phase 3 — Queue-Worker Integration Tests (Jan 2026)
- New `tests/test_queue/test_consumer_gengo.py` (7 tests)
- Tests: workflow vs stub path, style violations flagging, no-workflow fallback, empty segments, build_workflow wiring
- Updated `tests/test_integration/test_gengo_integration.py` (3 new tests)
- Tests: workflow flags violations, clean text passes, system_prompt reaches provider
- 6 new style checker workflow tests in `test_review/test_workflow.py`
- 12 new metrics tests in `test_review/test_metrics.py`

### Phase 4 — Docs/Config Cleanup (Jan 2026)
- `config.example.toml`: style_guide disabled by default, placeholder path
- `GENGO_STYLE_GUIDE.md`: rewritten with execution paths table, startup sequence, metrics docs, file manifest, dependency install instructions

### Phase 5 — Verification (Jan 2026)
- **187 tests passing, 0 failures** across full verification suite
- Pre-existing failures in unrelated test files (test_cli.py, test_review.py) not introduced by this work

## Prioritized Backlog

### P1 — Future Enhancements
- Separate judge provider/model config in config.toml
- Metrics output to file/Redis instead of just stdout
- Dashboard integration for quality trends
- Multiple style guide support (per language pair)

### P2 — Pre-existing Issues (Not introduced by this work)
- 11 failures in `test_cli.py` (CLI tool integration tests)
- 5 failures in `test_review.py` (factory function / no-provider tests)
- 1 failure in `test_manager.py` (constants test)
