# AGENTS.md - Python Translation Worker

> **Parent**: [../../AGENTS.md](../../AGENTS.md) (backend)

## Overview

Python-based translation service with multi-provider LLM support. Handles document parsing, translation, and quality assessment.

**Note**: This is a Python subsystem nested within the Go backend's `cmd/` directory.

## Multi-Tenancy (Redis Keys)

All Redis keys MUST be namespaced with user_id:

```python
# JobManager accepts user_id parameter
manager = JobManager(redis_host="localhost", user_id="uuid-here")

# Keys automatically namespaced:
# - user:{user_id}:trans:queue:*
# - user:{user_id}:trans:job:*
# - user:{user_id}:trans:state:*
# - user:{user_id}:translation:progress (pub/sub)

# Factory function also supports user_id
manager = create_job_manager(user_id="uuid-here")
```

**CRITICAL**: Never use global `trans:queue:*` keys - always pass user_id for tenant isolation.

## Quick Commands

```bash
# Install dependencies
pip install anthropic openai requests redis click

# Run worker
python main.py

# CLI usage
python -m review translate "こんにちは" --provider anthropic
python -m review batch -i sources.txt -o translations.txt
python -m review judge original.txt translation_a.txt translation_b.txt
```

## Directory Structure

```
translation-worker/
├── main.py                 # Worker entry point
├── config.toml             # Configuration file
├── review/                 # CLI + LLM integration
│   ├── cli.py              # Main CLI (636 lines - complexity hotspot)
│   ├── llm/                # LLM providers
│   │   └── cloud_providers.py  # Multiple providers (503 lines)
│   └── tests/              # CLI tests
├── parsers/                # Document parsers
│   ├── docx_parser.py      # Word docs (871 lines - largest file)
│   ├── pptx_parser.py      # PowerPoint (660 lines)
│   ├── pdf_parser.py       # PDF parsing
│   └── xlsx_parser.py      # Excel parsing
├── glossary/               # Term matching
│   └── matcher.py          # Fuzzy/POS matching (535 lines)
├── job_queue/              # Redis-backed queue
│   └── manager.py          # State machine (608 lines)
├── cache/                  # Translation caching
├── tests/                  # Integration tests
│   └── test_queue/         # Queue tests (1068 lines)
└── docs/                   # Documentation
    ├── CLI_REFERENCE.md
    ├── API_REFERENCE.md
    ├── LLM_PROVIDERS.md
    └── TROUBLESHOOTING.md
```

## Complexity Hotspots

| File | Lines | Issue | Recommendation |
|------|-------|-------|----------------|
| `docx_parser.py` | 871 | Layout preservation logic | Extract table helper class |
| `pptx_parser.py` | 660 | Slide extraction complexity | N/A (inherent) |
| `cli.py` | 636 | Multi-command CLI | Move error templates to config |
| `manager.py` | 608 | Redis state machine | N/A (fault-tolerance) |
| `matcher.py` | 535 | Dual matching paths | N/A (intentional fallback) |
| `cloud_providers.py` | 503 | Multiple providers in one file | Split per provider |

## Code Patterns

### Python Style
- **Line length**: 120 chars
- **Formatter**: Black
- **Testing**: pytest
- **Language**: US English only

### LLM Provider Pattern
```python
# Use max_completion_tokens, NOT max_tokens (deprecated)
response = client.messages.create(
    model="claude-sonnet-4-5-20250929",
    max_completion_tokens=4096,  # CORRECT
    # max_tokens=4096,  # WRONG - deprecated
    messages=[...]
)
```

### Job Queue Pattern
```python
# Priority levels: urgent, normal, bulk
await queue.enqueue(job, priority="normal")

# Checkpointing for fault tolerance
await queue.save_checkpoint(job_id, progress)
```

## Configuration (config.toml)

```toml
[worker]
id = "worker-1"
max_concurrent = 3

[translation]
default_provider = "anthropic"
default_model = "claude-sonnet-4-5-20250929"

[cache.redis]
host = "localhost"
port = 6379

[job_queue]
enabled = true
backend = "redis"
```

## MUST NOT

- Use `max_tokens` in LLM calls (use `max_completion_tokens`)
- Leave empty `except:` blocks
- Use mutable default arguments
- Skip glossary term checking in translations

## Environment Variables

```bash
ANTHROPIC_API_KEY=sk-ant-...    # Anthropic API key
OPENAI_API_KEY=sk-...           # OpenAI API key (optional)
```

---

*For full CLI reference, see docs/CLI_REFERENCE.md*
