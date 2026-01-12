# Translation Worker CLI

Command-line interface for Japanese-to-English translation with LLM provider support.

## Quick Start

```bash
# Using local CLI tools (no API costs!)
python -m review translate "こんにちは" --cli claude

# Using API providers (requires API key)
python -m review translate "こんにちは" --provider anthropic

# Batch processing
python -m review batch --input sources.txt --output translations.txt --cli claude

# Compare translations
python -m review judge source.txt trans_a.txt trans_b.txt --cli claude
```

## Installation

```bash
# For API providers
pip install anthropic openai

# For CLI tools (no API costs!)
npm install -g @anthropic-ai/claude-code
npm install -g @github-copilot/codex-cli
npm install -g @google/generative-ai-cli
```

## Commands

### translate

Translate Japanese text to English.

```bash
python -m review translate "こんにちは" --cli claude
python -m review translate "こんにちは" --cli codex
python -m review translate "こんにちは" --provider anthropic --format json
```

### batch

Batch translate from file (one line per source).

```bash
python -m review batch --input sources.txt --output translations.txt --cli claude --format csv
```

**File size limit:** Input files are limited to 10MB for security reasons.

### judge

Compare two translations and select winner.

```bash
python -m review judge source.txt translation_a.txt translation_b.txt --cli claude
```

## Options

| Option | Description |
|--------|-------------|
| `--provider` | Use API provider (anthropic, openai, gemini) |
| `--cli` | Use local CLI tool (claude, codex, gemini, ollama) |
| `--model` | Specify model identifier |
| `--format` | Output format: text, json, csv |
| `--output` | Write to file instead of stdout |
| `--dry-run` | Show command without executing (CLI only) |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `OPENAI_API_KEY` | OpenAI API key |
| `GEMINI_API_KEY` | Google Gemini API key |
| `GEMINI_PROJECT_ID` | Google Cloud project ID |
| `GEMINI_LOCATION` | Gemini region (default: us-central1) |
