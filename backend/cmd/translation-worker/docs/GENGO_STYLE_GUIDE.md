# Gengo Style Guide Integration

This document describes the Gengo Japanese-to-English style guide integration for the translation worker.

## Overview

The style guide integration enables high-quality US English translations that follow Gengo's professional translation standards. The system:

1. Parses a Gengo-aligned style guide markdown file
2. Generates optimized system prompts for LLM providers
3. Validates translations against style rules

## Configuration

Enable the style guide in `config.toml`:

```toml
[style_guide]
enabled = true
path = "/path/to/gengo-style-guide.md"
```

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

## Python API

### Parsing Style Guide

```python
from style_guide import parse_gengo_style_guide, ParsedStyleGuide

guide = parse_gengo_style_guide(Path("/path/to/guide.md"))

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
    "anthropic",
    api_key="sk-ant-...",
    system_prompt=prompt,  # Injected into all translation requests
)

result = await provider.translate(source_text)
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

```bash
cd backend/cmd/translation-worker

# Run style guide tests
pytest tests/test_style_guide/ -v

# Run style checker tests
pytest tests/test_audit/test_style_checker.py -v

# Run integration tests
pytest tests/test_integration/test_gengo_integration.py -v
```

## Files

| File | Purpose |
|------|---------|
| `style_guide/parser.py` | Markdown parser for style guide files |
| `style_guide/prompt_builder.py` | System prompt generator |
| `audit/style_checker.py` | Translation style validator |

## Best Practices

1. **Keep style guide updated**: As Gengo updates their guidelines, update the markdown file
2. **Enable Gengo rules**: Set `gengo_rules_enabled=True` in StyleChecker for production
3. **Review flagged issues**: The checker provides suggestions, not automatic corrections
4. **Test with samples**: Use the integration tests as a template for validation
