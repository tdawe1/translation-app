# LLM Provider Configuration Guide

Complete guide for configuring and using LLM providers with the translation worker.

## Overview

The translation worker supports multiple LLM providers through a unified interface. **For development and internal use, leverage subscription-based endpoints** (Claude Code, GitHub Copilot, etc.) to avoid per-token API costs.

`★ Insight ─────────────────────────────────────`
**Cost Optimization Strategy**:
- **Development**: Use subscription-based endpoints (Claude Code, Copilot, etc.)
- **Production**: Use API keys with proper rate limiting and caching
- **Hybrid**: Cache translations to minimize repeated API calls
`─────────────────────────────────────────────────`

## Supported Providers

| Provider | Default Model | Strengths | Use Case |
|----------|---------------|-----------|----------|
| **Anthropic** | `claude-sonnet-4-5-20250929` | Accuracy, nuance | Professional translations |
| **OpenAI** | `gpt-4.1-2025-04-14` | Speed, consistency | Real-time translation |
| **Gemini** | `gemini-3.0-pro` | Cost-effective | Bulk processing |

### Subscription-Based Endpoints (Recommended for Development)

| Provider | Subscription Model | Endpoint | Notes |
|----------|-------------------|----------|-------|
| **Claude Code** | `$20/month` | `http://localhost:8000/v1/messages` | No per-token costs |
| **GitHub Copilot** | `$10/month` | Via VS Code extension | Code-focused |
| **Cursor** | `$20/month` | Built-in IDE integration | Includes Claude |
| **Gemini** | Free tier | `generativelanguage.googleapis.com` | Generous free quota |

## Configuration

### Environment Variables

All providers use environment variables for API credentials:

```bash
# Anthropic Claude
export ANTHROPIC_API_KEY="sk-ant-..."

# OpenAI GPT
export OPENAI_API_KEY="sk-..."

# Google Gemini
export GEMINI_API_KEY="..."
export GEMINI_PROJECT_ID="your-gcp-project-id"
export GEMINI_LOCATION="us-central1"  # Optional
```

### Config File

Set default provider in `config.toml`:

```toml
[translation]
default_provider = "anthropic"
default_model = "claude-sonnet-4-5-20250929"
```

## Anthropic Claude

### Models

| Model | ID | Use Case |
|-------|-----|----------|
| Claude Opus 4.5 | `claude-opus-4-5-20251101` | Highest quality, most expensive |
| Claude Sonnet 4.5 | `claude-sonnet-4-5-20250929` | Best balance (default) |
| Claude Haiku 4.5 | `claude-haiku-4-5-20250319` | Fastest, most cost-effective |

### Configuration

```python
from review.llm import get_provider

provider = get_provider(
    "anthropic",
    api_key="sk-ant-...",
    model="claude-sonnet-4-5-20250929"
)
```

### API Reference (2026)

```python
response = provider.generate(
    prompt="Translate to English: こんにちは",
    max_tokens=4096,
    temperature=0.0  # Lower = more deterministic
)

# Access response
print(response.text)        # Translated text
print(response.model)       # Model used
print(response.usage)       # Token usage
print(response.latency_ms)  # Request latency
```

### Rate Limits

- **Free Tier**: 5 requests per minute
- **Paid Tier**: 50 requests per minute (higher with enterprise)

Use retry logic (built-in with tenacity):

```python
@retry(
    stop=stop_after_attempt(3),
    wait=wait_exponential(multiplier=1, min=1, max=60)
)
def translate_with_retry(text):
    return provider.generate(prompt)
```

## OpenAI GPT

### Models

| Model | ID | Use Case |
|-------|-----|----------|
| GPT-4.1 | `gpt-4.1-2025-04-14` | High quality (default) |
| GPT-4.1 Mini | `gpt-4.1-mini-2025-04-14` | Cost-effective |
| GPT-5.2 | `gpt-5.2` | Latest model |

`★ Insight ─────────────────────────────────────`
**2026 API Change**: OpenAI deprecated `max_tokens` in favor of `max_completion_tokens`.
The translation worker uses `max_completion_tokens` automatically.
`─────────────────────────────────────────────────`

### Configuration

```python
provider = get_provider(
    "openai",
    api_key="sk-...",
    model="gpt-4.1-2025-04-14"
)
```

### API Reference (2026)

```python
response = provider.generate(
    prompt="Translate to English: こんにちは",
    max_tokens=4096,  # Automatically uses max_completion_tokens
    temperature=0.0
)
```

### Rate Limits

- **Tier 1**: 3,000 RPM (requests per minute)
- **Tier 2**: 10,000 RPM
- **Tier 3**: 30,000 RPM

Check your tier at https://platform.openai.com/account/limits

## Google Gemini

### Models

| Model | ID | Use Case |
|-------|-----|----------|
| Gemini 3.0 Pro | `gemini-3.0-pro` | High quality (default) |
| Gemini 3.0 Flash | `gemini-3.0-flash` | Fast, cost-effective |

### Configuration

```python
provider = get_provider(
    "gemini",
    api_key="...",
    model="gemini-3.0-pro",
    project_id="your-gcp-project",
    location="us-central1"  # Optional
)
```

### API Reference (2026)

Gemini uses Vertex AI REST endpoint:

```python
response = provider.generate(
    prompt="Translate to English: こんにちは",
    max_tokens=4096,
    temperature=0.0
)

# Gemini-specific response parsing
text = response.text
usage = response.usage  # Contains promptTokenCount, candidatesTokenCount, totalTokenCount
```

### Environment Variables

```bash
export GEMINI_API_KEY="your-api-key"
export GEMINI_PROJECT_ID="my-project"
export GEMINI_LOCATION="us-central1"  # Default
```

## Multi-Provider Translation

For critical translations, use multiple providers and compare results:

```python
from review.multimodel import MultiModelTranslator

# Configure providers
translators = {
    "anthropic": get_provider("anthropic", api_key="..."),
    "openai": get_provider("openai", api_key="...")
}

multi = MultiModelTranslator(translators)

# Translate with all providers
results = multi.translate_parallel("こんにちは")

# Compare results
for provider, result in results.items():
    print(f"{provider}: {result.text}")
```

## Token Usage and Cost Estimation

### Estimating Tokens

Japanese text typically uses 2-3 tokens per character:

```python
def estimate_tokens(japanese_text: str) -> int:
    """Rough estimate: ~2.5 tokens per Japanese character"""
    return len(japanese_text) * 2.5
```

### Cost Comparison (per 1M tokens)

| Provider | Model | Input | Output |
|----------|-------|-------|--------|
| Anthropic | Sonnet 4.5 | $3.00 | $15.00 |
| OpenAI | GPT-4.1 | $2.50 | $10.00 |
| Gemini | Pro | $2.00 | $6.00 |

*Prices as of 2026 - check provider sites for current rates*

## Troubleshooting

### Common Errors

**ImportError: No module named 'anthropic'**
```bash
pip install anthropic openai requests redis
```

**ValueError: API key is required**
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

**Rate limit exceeded**
- Implement exponential backoff (built-in)
- Reduce concurrent requests
- Upgrade API tier

### Debug Mode

Enable detailed logging:

```python
import logging
logging.basicConfig(level=logging.DEBUG)
```

### Testing Connection

```python
provider = get_provider("anthropic", api_key="...")
if provider.is_available():
    print("Provider is ready")
else:
    print("Provider check failed")
```

## Best Practices

1. **Choose the right model**: Use Sonnet 4.5 for most cases, Opus 4.5 for critical content
2. **Set temperature to 0**: For consistent translations, use `temperature=0.0`
3. **Cache results**: Avoid re-translating identical content
4. **Batch requests**: Group similar translations to reduce overhead
5. **Monitor usage**: Track token consumption for cost management
6. **Use retries**: Built-in retry logic handles transient failures
7. **Compare providers**: Use `judge` command to compare translation quality

## Security

- Never commit API keys to version control
- Use environment variables or secret management
- Rotate API keys regularly
- Monitor usage for unusual activity
- Use separate keys for development and production
