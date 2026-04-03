# Gengo Style Guide Integration — PRD

## Original Problem Statement
Make the translation worker apply the Gengo style guide during real queued job execution, not just in isolated modules and CLI flows.

## Architecture
- Python translation worker in `backend/cmd/translation-worker/`
- Uses LLM providers (OpenAI, Anthropic, Gemini) for translation + judging
- Redis job queue for horizontal scaling
- Style guide parsed from markdown → system prompt → injected into LLM provider
- Style checker validates translations post-hoc (regex-based rules)

## Core Requirements (Static)
1. Worker builds a real TranslationWorkflow at startup from config + env
2. Style guide prompt injected via provider's system_prompt
3. Style checker flags violations in queued job output
4. Config/docs are portable (no hardcoded paths)
5. Tests cover the worker path, not just isolated modules

## User Personas
- Translation team running the worker against Gengo-sourced content
- DevOps deploying/configuring the worker via config.toml + env vars

## What's Been Implemented

### Phase 1 — Worker Startup Runtime Construction (Jan 2026)
- Added `load_style_guide_prompt()`, `_resolve_api_key()`, `build_translation_provider()`, `build_judge_provider()`, `build_style_checker()`, `build_workflow()` to `main.py`
- `QueueConsumer` now receives a real workflow (no more stub path in normal operation)
- Missing API keys produce clear startup error
- Fixed `config.example.toml` and `config.toml` (removed hardcoded path)
- 19 new tests in `test_main.py`, 5 pre-existing failures fixed in `test_workflow.py`
- **Result: 148 tests passing, 0 failures**

## Prioritized Backlog

### P0 — Phase 2: Style Checker in Workflow
- Add optional `style_checker` to `TranslationWorkflow`
- Run checker on `segment.target` after winner selection
- Convert findings to review flags
- Add `style_issues` field to `TranslationSegment`

### P0 — Phase 3: Queue-Worker Integration Tests
- Test `_process_job()` with mock provider
- Verify `system_prompt` reaches the provider
- Test style-violating translation gets flagged

### P1 — Phase 4: Documentation Cleanup
- Update README, GENGO_STYLE_GUIDE.md
- Document which execution paths are covered

### P1 — Phase 5: Full Verification
- Run complete test suite
- Manual startup verification

### P2 — Future
- Separate judge provider/model config
- Structured JSON metrics for quality dashboards
- Multiple style guide support
