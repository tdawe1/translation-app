# Gengo Style Guide Integration Delta Plan

## Status

This plan supersedes the previous version.

The earlier plan was written against an outdated baseline and tried to recreate work that already exists. The parser, prompt builder, config hooks, style-checker rules, integration tests, and documentation are already in the repository.

What is still missing is the actual worker-side runtime wiring.

## Current State

Already implemented:

- `backend/cmd/translation-worker/style_guide/parser.py`
- `backend/cmd/translation-worker/style_guide/prompt_builder.py`
- `backend/cmd/translation-worker/audit/style_checker.py`
- `backend/cmd/translation-worker/tests/test_style_guide/*`
- `backend/cmd/translation-worker/tests/test_integration/test_gengo_integration.py`
- `backend/cmd/translation-worker/config.example.toml`
- `backend/cmd/translation-worker/docs/GENGO_STYLE_GUIDE.md`

Verified locally:

- `pytest tests/test_style_guide/test_parser.py tests/test_style_guide/test_prompt_builder.py tests/test_audit/test_style_checker.py tests/test_integration/test_gengo_integration.py`
- Result: passing

Known baseline problems:

1. `main.py` loads a style-guide-derived `system_prompt`, but does not pass it into the actual job-processing path.
2. `QueueConsumer` is created without a workflow, so queued jobs fall back to the stub path.
3. The queue worker does not build providers from config/env at startup.
4. The queue worker path does not apply the existing style checker as a review signal.
5. `config.example.toml` uses a machine-specific absolute path, which makes validation fail on other machines when enabled.

## Goal

Make the translation worker apply the Gengo style guide during real queued job execution, not just in isolated modules and CLI flows.

## Success Criteria

1. When `[style_guide].enabled = true`, the worker builds a prompt from the configured markdown file and uses it during queued translations.
2. The worker initializes a real `TranslationWorkflow` at startup and passes it to `QueueConsumer`.
3. The worker creates its translation provider from config and environment variables instead of running with no workflow.
4. Style-guide violations can influence review outcomes for queued jobs.
5. The example config and docs are portable and do not assume a developer-specific filesystem path.
6. The new behavior is covered by targeted tests.

## Non-Goals

This plan does not include:

- Rewriting the markdown parser
- Rewriting the prompt builder
- Replacing the current provider abstraction
- Adding new provider families
- Reworking all CLI translation flows

## Implementation Strategy

Prefer the smallest correct change set.

Use the existing provider `system_prompt` support as the canonical API-provider injection path for the queue worker. Do not inject the same Gengo prompt twice through both provider config and prompt-prefix composition in the same code path.

## Phase 1: Make Worker Startup Build a Real Runtime

### Scope

Modify:

- `backend/cmd/translation-worker/main.py`
- `backend/cmd/translation-worker/tests/test_main.py`

### Work

Add small runtime-construction helpers in `main.py` instead of introducing a new module unless the code becomes unwieldy:

1. Add a helper to load the optional style guide prompt from config.
2. Add a helper to build the primary translation provider from:
   - `translation.default_provider`
   - `translation.default_model`
   - required environment variables
3. Add a helper to build the judge provider.
   - Minimal first pass: use the same provider/model family as translation unless the config already supports separate judge settings.
4. Add a helper to build `TranslationWorkflow` with real dependencies.
5. Pass the resulting workflow into `QueueConsumer(...)`.

### Notes

- Use provider-native `system_prompt` injection for API providers in the worker path.
- Do not rely on `MultiModelTranslator.system_prompt` in the queue worker if the provider already receives the same prompt.
- If a required API key is missing, fail startup clearly instead of silently starting a stub worker.

### Acceptance Criteria

1. `QueueConsumer` no longer runs jobs with `self.workflow is None` in the normal configured path.
2. Worker startup logs whether the style guide is enabled and whether a workflow was built successfully.
3. Missing provider credentials produce an actionable startup error.

### Tests

Add or update tests in `tests/test_main.py` to cover:

1. Building a workflow when style guide is disabled.
2. Building a workflow when style guide is enabled.
3. Passing the loaded prompt into provider construction.
4. Failing cleanly when provider credentials are absent.

## Phase 2: Apply Style Checks in the Queue Workflow

### Scope

Modify:

- `backend/cmd/translation-worker/review/workflow.py`
- `backend/cmd/translation-worker/review/models.py`
- `backend/cmd/translation-worker/main.py`
- `backend/cmd/translation-worker/tests/test_review/test_workflow.py`
- `backend/cmd/translation-worker/tests/test_integration/test_gengo_integration.py`

### Work

The queue worker currently uses judge confidence and heuristic flagging, but not the existing `StyleChecker`.

Add optional style-checker support to the workflow:

1. Allow `TranslationWorkflow` to accept an optional style checker.
2. After the winning translation is selected, run the style checker on `segment.target`.
3. Convert style-checker findings into review signals.
4. Persist enough information for audit/review.

Minimal acceptable behavior:

- If Gengo rules are enabled and a translation triggers style issues, the segment is flagged for review.
- The flag reason includes the first relevant style issue or a concise summary.

Preferred behavior if the change remains small:

- Extend `TranslationSegment` with a lightweight `style_issues` field for audit/debug visibility.

### Acceptance Criteria

1. The queue workflow can flag translations for Oxford comma, UK spelling, currency, date, and time issues when Gengo rules are enabled.
2. When Gengo rules are disabled, queue workflow behavior remains unchanged.
3. Review/export output still works with the added metadata.

### Tests

Add or update tests to cover:

1. Style checker disabled: no additional flagging.
2. Style checker enabled: a violating translation is flagged.
3. Clean Gengo-compliant translation does not get flagged by style rules.

## Phase 3: Tighten Queue-Worker Integration Tests

### Scope

Modify:

- `backend/cmd/translation-worker/tests/test_main.py`
- `backend/cmd/translation-worker/tests/test_integration/test_gengo_integration.py`
- optionally add `backend/cmd/translation-worker/tests/test_queue/test_consumer_gengo.py`

### Work

Add tests around the actual worker path rather than only isolated helper modules.

Target scenarios:

1. Worker startup with style guide enabled builds a non-stub workflow.
2. `QueueConsumer._process_job()` runs a job through a configured workflow.
3. The configured provider receives the expected `system_prompt`.
4. A style-violating translation is surfaced as review-needed output.

### Testing Approach

Prefer mocks/fakes over real API calls.

- Mock provider creation at the `get_provider(...)` boundary.
- Use a temporary style guide fixture.
- Use a fake job manager or mock Redis-dependent boundaries where possible.

## Phase 4: Make Config and Docs Portable

### Scope

Modify:

- `backend/cmd/translation-worker/config.example.toml`
- `backend/cmd/translation-worker/README.md`
- `backend/cmd/translation-worker/docs/GENGO_STYLE_GUIDE.md`

### Work

Update docs and examples so they work on any machine:

1. Remove the hardcoded `/home/thomas/...` example path.
2. Change the example config so the style guide is disabled by default, or use a placeholder path that is clearly illustrative.
3. Document that enabling the style guide requires a real local markdown file.
4. Document which execution paths are covered:
   - queued worker jobs
   - document CLI flows
   - provider-based translation path
5. Document dependency installation before running provider tests.

### Acceptance Criteria

1. `config.example.toml` is safe to copy without immediately failing on another machine.
2. Documentation matches actual runtime behavior after Phases 1-3.

## Phase 5: Verification

### Environment

Install worker dependencies first:

```bash
cd backend/cmd/translation-worker
pip install -r requirements.txt
```

### Required Test Runs

```bash
cd backend/cmd/translation-worker

pytest tests/test_style_guide/test_parser.py \
  tests/test_style_guide/test_prompt_builder.py \
  tests/test_audit/test_style_checker.py \
  tests/test_review/test_llm_providers.py \
  tests/test_review/test_workflow.py \
  tests/test_integration/test_gengo_integration.py \
  tests/test_main.py -v
```

If a dedicated queue-consumer test file is added, include it in the verification run.

### Manual Verification

1. Start the worker with style guide disabled and confirm normal startup.
2. Start the worker with style guide enabled and a valid markdown path.
3. Confirm startup logs indicate:
   - style guide loaded
   - workflow initialized
   - provider/model selected
4. Enqueue a representative job and confirm it is processed by the workflow rather than the stub path.

## Delivery Order

Implement in this order:

1. Phase 1: real worker runtime construction
2. Phase 2: style-checker review integration
3. Phase 3: queue-worker integration tests
4. Phase 4: docs/config cleanup
5. Phase 5: verification

## Definition of Done

This work is done when:

1. The queue worker uses a real workflow in normal operation.
2. The style guide prompt is applied in queued translation execution.
3. Style-guide violations can affect queued review outcomes.
4. Tests cover the worker path, not just isolated utility modules.
5. Config and docs are portable and accurate.
