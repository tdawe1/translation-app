# Translation Worker CLI Reference

Complete reference for the translation worker command-line interface.

## Overview

The translation worker provides CLI access through the `review` module for direct translation and evaluation without requiring a web server.

## Installation

```bash
cd backend/cmd/translation-worker
pip install -r requirements.txt
```

## Environment Setup

Set API keys for your preferred LLM provider:

```bash
# Anthropic Claude (recommended for accuracy)
export ANTHROPIC_API_KEY="sk-ant-..."

# OpenAI GPT
export OPENAI_API_KEY="sk-..."

# Google Gemini
export GEMINI_API_KEY="..."
export GEMINI_PROJECT_ID="your-project-id"
export GEMINI_LOCATION="us-central1"  # Optional
```

## Commands

### `translate` - Translate Text

Translates Japanese text to English using the specified LLM provider.

#### Syntax

```bash
python -m review translate TEXT --provider PROVIDER [OPTIONS]
```

#### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `TEXT` | string | Yes | Japanese text to translate |

#### Options

| Option | Short | Type | Required | Default | Description |
|--------|-------|------|----------|---------|-------------|
| `--provider` | `-p` | choice | Yes | - | LLM provider: `anthropic`, `openai`, `gemini` |
| `--model` | `-m` | string | No | provider default | Model identifier |
| `--format` | `-f` | choice | No | `text` | Output format: `text`, `json`, `csv` |
| `--output` | `-o` | path | No | stdout | Write output to file |
| `--parallel` | - | flag | No | `True` | Use parallel execution (multi-provider only) |

#### Examples

```bash
# Basic translation
python -m review translate "こんにちは" --provider anthropic

# With custom model
python -m review translate "こんにちは" --provider openai --model gpt-4.1

# JSON output with usage statistics
python -m review translate "こんにちは" --provider anthropic --format json

# Save to file
python -m review translate "こんにちは" --provider anthropic --output result.txt

# CSV output for batch processing
python -m review translate "こんにちは" --provider openai --format csv
```

#### Output Formats

**text** (default):
```
Hello
```

**json**:
```json
{
  "source": "こんにちは",
  "translation": "Hello",
  "provider": "anthropic",
  "model": "claude-sonnet-4-5-20250929",
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 5,
    "total_tokens": 25
  },
  "latency_ms": 450
}
```

**csv**:
```csv
source,translation,provider,model
"こんにちは","Hello","anthropic","claude-sonnet-4-5-20250929"
```

---

### `batch` - Batch Translate

Processes multiple texts from an input file, one translation per line.

#### Syntax

```bash
python -m review batch --input FILE --output FILE --provider PROVIDER [OPTIONS]
```

#### Options

| Option | Short | Type | Required | Default | Description |
|--------|-------|------|----------|---------|-------------|
| `--input` | `-i` | path | Yes | - | Input file with source texts (one per line) |
| `--output` | `-o` | path | Yes | - | Output file for translations |
| `--provider` | `-p` | choice | Yes | - | LLM provider |
| `--model` | `-m` | string | No | provider default | Model identifier |
| `--format` | `-f` | choice | No | `text` | Output format: `text`, `json`, `csv` |
| `--parallel` | - | flag | No | `True` | Execution mode |

#### Examples

```bash
# Basic batch processing
python -m review batch -i sources.txt -o translations.txt -p anthropic

# JSON output with metadata
python -m review batch -i sources.txt -o results.json -p openai -f json

# CSV for spreadsheet import
python -m review batch -i sources.txt -o output.csv -p anthropic -f csv
```

#### Input File Format

One Japanese text per line:

```text
こんにちは
おはようございます
ありがとう
```

#### Output Formats

**text** (default):
```text
Hello
Good morning
Thank you
```

**json**:
```json
[
  {
    "source": "こんにちは",
    "translation": "Hello",
    "usage": {"prompt_tokens": 20, "completion_tokens": 5, "total_tokens": 25}
  },
  ...
]
```

**csv**:
```csv
source,translation,prompt_tokens,completion_tokens
"こんにちは","Hello",20,5
"おはようございます","Good morning",25,8
```

---

### `judge` - Compare Translations

Compares two translation candidates and selects the better one using LLM evaluation.

#### Syntax

```bash
python -m review judge SOURCE CANDIDATE_A CANDIDATE_B [OPTIONS]
```

#### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `SOURCE` | path | Yes | Original source text file |
| `CANDIDATE_A` | path | Yes | First translation file |
| `CANDIDATE_B` | path | Yes | Second translation file |

#### Options

| Option | Short | Type | Required | Default | Description |
|--------|-------|------|----------|---------|-------------|
| `--provider` | `-p` | choice | No | `anthropic` | Judge LLM provider |
| `--model` | `-m` | string | No | `claude-4.5-sonnet` | Judge model |
| `--format` | `-f` | choice | No | `text` | Output format: `text`, `json` |

#### Examples

```bash
# Basic comparison
python -m review judge original.txt translation_a.txt translation_b.txt

# Using OpenAI as judge
python -m review judge original.txt trans_a.txt trans_b.txt -p openai

# JSON output
python -m review judge original.txt trans_a.txt trans_b.txt -f json
```

#### Output

**text** (default):
```
Winner: translation_a
Confidence: 0.85
Reasoning: Translation A captures the nuance better while maintaining accuracy.
```

**json**:
```json
{
  "winner": "translation_a",
  "confidence": 0.85,
  "reasoning": "Translation A captures the nuance better...",
  "concerns": []
}
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Missing API key |
| 3 | Invalid provider |
| 4 | File not found |

## Error Handling

The CLI provides detailed error messages for common issues:

```bash
# Missing API key
$ python -m review translate "test" -p anthropic
Error: Missing API key for anthropropic. Set ANTHROPIC_API_KEY environment variable.

# Invalid provider
$ python -m review translate "test" -p invalid
Error: Invalid value for '--provider': 'invalid'. Choose from anthropic, openai, gemini.

# Missing dependencies
$ python -m review translate "test" -p anthropic
Error: LLM integration not available. Install dependencies:
  pip install anthropic openai requests
```

---

## Version

```bash
python -m review --version
# Output: Translation Worker CLI, version 1.0.0
```

---

## Help

```bash
# General help
python -m review --help

# Command-specific help
python -m review translate --help
python -m review batch --help
python -m review judge --help
```
