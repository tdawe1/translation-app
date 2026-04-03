# Gengo Style Guide Integration

This document describes the Gengo Japanese-to-English style guide integration for the translation worker.

## Overview

The style guide integration enables high-quality US English translations that follow Gengo's professional translation standards. The system:

1. Parses a Gengo-aligned style guide markdown file
2. Generates optimized system prompts for LLM providers
3. Injects prompts into the translation provider at worker startup
4. Validates translations against style rules post-translation
5. Flags style violations for review in the queue workflow
6. Emits structured JSON metrics (violations by category, flag rate, latency)

## Execution Paths

The style guide is applied in these execution paths:

| Path | Prompt Injection | Style Checker | Metrics |
|------|:---:|:---:|:---:|
| **Queued worker jobs** | Yes | Yes | Yes |
| **Document CLI flows** | Manual | Manual | No |
| **Provider-based translation** | Via `system_prompt` | No | No |

## Configuration

Enable the style guide in `config.toml`:

```toml
[style_guide]
enabled = true
path = "./path/to/gengo-style-guide.md"
```

**Important**: The `path` must point to a valid markdown file on the machine running the worker. When `enabled = true`, the worker will:
- Parse the markdown file at startup
- Build a system prompt from the parsed rules
- Inject the prompt into the translation provider via `system_prompt`
- Activate the `StyleChecker` with Gengo rules in the workflow

When `enabled = false` (the default), the worker operates normally without style guide injection or style checking.

## Style Guide Format

The parser expects a markdown file with `##` section headers. Key sections include:

- **Punctuation**: Standard English punctuation rules
- **Spelling**: US English requirements (color, organize, favor)
- **Grammar & Syntax**: Articles, agreement, idiomatic phrasing
- **Numbers**: Spell out 0-9, use numerals for 10+
- **Currency**: Symbol prefix format (US$1,000, ¥1,000)
- **Dates & Times**: US format (September 21, 2025), lowercase a.m./p.m.
- **Tone & Register**: Match source formality

## Style Checker Rules

When `gengo_rules_enabled=True` in the StyleChecker, the following checks are performed:

| Category | Rule | Example Violation | Suggestion |
|----------|------|-------------------|------------|
| `uk_spelling` | Use US English | "colour", "organise" | "color", "organize" |
| `oxford_comma` | Use Oxford comma in lists | "apples, oranges and bananas" | "apples, oranges, and bananas" |
| `currency_format` | Use symbol prefix | "1,000 dollars" | "US$1,000" |
| `date_format` | Use US format | "21 September 2025" | "September 21, 2025" |
| `time_format` | Use lowercase a.m./p.m. | "3:00 PM" | "3:00 p.m." |

## Worker Startup

The worker builds a real `TranslationWorkflow` at startup:

1. Loads style guide prompt from config (if enabled)
2. Builds translation provider from `translation.default_provider` + `translation.default_model` + env vars
3. Builds judge provider (same family, no style prompt)
4. Builds `StyleChecker` (if style guide enabled)
5. Assembles `TranslationWorkflow` and passes it to `QueueConsumer`

Required environment variables (depend on configured provider):
- `OPENAI_API_KEY` for OpenAI provider
- `ANTHROPIC_API_KEY` for Anthropic provider
- `GEMINI_API_KEY` for Gemini provider

Missing API keys produce a clear startup error.

## Structured Metrics

After each job, the workflow emits JSON metrics to stdout (prefixed with `[METRICS]`):

```json
{
  "job_id": "abc123",
  "segment_count": 15,
  "flagged_count": 3,
  "flag_rate": 0.2,
  "style_violation_count": 2,
  "style_violation_rate": 0.1333,
  "style_violations_by_category": {"uk_spelling": 1, "oxford_comma": 1},
  "flag_reasons": ["Style: uk_spelling - ...", "Low confidence"],
  "overall_score": 0.85,
  "provider_name": "openai",
  "style_guide_enabled": true,
  "processing_duration_ms": 4523
}
```

These can be piped to a logging aggregator for dashboards.

## Python API

### Parsing Style Guide

```python
from style_guide import parse_gengo_style_guide, ParsedStyleGuide

guide = parse_gengo_style_guide(Path("./path/to/guide.md"))

for section_name, section in guide.sections.items():
    print(f"Section: {section.name}")
    print(f"Rules: {len(section.rules)}")
```

### Building System Prompts

```python
from style_guide import build_system_prompt, SystemPromptConfig

config = SystemPromptConfig(
    include_examples=True,
    include_tone=True,
    include_formatting=True,
    max_section_length=500,
)

prompt = build_system_prompt(guide, config)
```

### Using with LLM Providers

```python
from review.llm import get_provider

provider = get_provider(
    "openai",
    api_key="sk-...",
    model="gpt-5.2",
    system_prompt=prompt,  # Injected into all translation requests
)
```

### Style Checking Translations

```python
from audit.style_checker import create_style_checker

checker = create_style_checker(gengo_rules_enabled=True)
issues = checker.check("The colour costs 1,000 dollars.")

for issue in issues:
    print(f"[{issue.severity}] {issue.category}: {issue.message}")
    if issue.suggestion:
        print(f"  Suggestion: {issue.suggestion}")
```

## Testing

Install dependencies first:

```bash
cd backend/cmd/translation-worker
pip install -r requirements.txt
```

Run the full verification suite:

```bash
cd backend/cmd/translation-worker

pytest tests/test_style_guide/test_parser.py \
  tests/test_style_guide/test_prompt_builder.py \
  tests/test_audit/test_style_checker.py \
  tests/test_review/test_llm_providers.py \
  tests/test_review/test_workflow.py \
  tests/test_review/test_metrics.py \
  tests/test_integration/test_gengo_integration.py \
  tests/test_main.py \
  tests/test_queue/test_consumer_gengo.py -v
```

## Files

| File | Purpose |
|------|---------|
| `style_guide/parser.py` | Markdown parser for style guide files |
| `style_guide/prompt_builder.py` | System prompt generator |
| `audit/style_checker.py` | Translation style validator |
| `review/workflow.py` | Workflow orchestrator (style checker + metrics) |
| `review/metrics.py` | Structured job metrics |
| `review/exporter.py` | CSV exporter (includes style_issues column) |
| `main.py` | Worker startup, runtime construction helpers |
| `tests/test_queue/test_consumer_gengo.py` | Queue consumer integration tests |

## Best Practices

1. **Keep style guide updated**: As Gengo updates their guidelines, update the markdown file
2. **Enable Gengo rules**: Set `gengo_rules_enabled=True` in StyleChecker for production
3. **Review flagged issues**: The checker provides suggestions, not automatic corrections
4. **Monitor metrics**: Use the JSON metrics output for quality dashboards and alerting
5. **Test with samples**: Use the integration tests as a template for validation
