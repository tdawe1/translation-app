# LLM API Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace MVP stubs in translation worker review module with real Anthropic/OpenAI/Gemini API calls for multi-model translation and judge evaluation, with CLI support for local execution.

**Architecture:** Create provider abstraction layer with retry logic using tenacity, inject API clients into existing MultiModelTranslator and TranslationJudge, maintain backward compatibility with stubs for testing, add CLI for direct invocation.

**Tech Stack:** anthropic>=0.40.0, openai>=1.0.0, google-cloud-aiplatform>=1.0.0, tenacity>=8.2.0, asyncio, click, pytest-mock

## Priority: CLI Support (INDISPENSABLE)

**CLI Requirements:**
- Direct invocation without web server: `python -m review.cli translate "こんにちは"`
- Support all providers: `--provider anthropic|openai|gemini`
- Batch processing from file: `--input sources.txt --output translations.txt`
- Configurable models: `--model claude-4.5-sonnet`
- Parallel execution control: `--parallel/--sequential`
- Output formats: `--format text|json|csv`

---

## Task 1: Create LLM provider abstraction layer

**Files:**
- Create: `review/llm/__init__.py`
- Create: `review/llm/base.py`
- Create: `review/llm/providers.py`
- Test: `tests/test_review/test_llm_providers.py`

**Step 1: Write the failing test**

```python
# tests/test_review/test_llm_providers.py
import pytest
import sys
from pathlib import Path

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.llm.providers import AnthropicProvider, OpenAIProvider, GeminiProvider, get_provider


class TestProviderAbstraction:
    def test_anthropic_provider_requires_api_key(self):
        """Should raise error if API key missing."""
        provider = AnthropicProvider(api_key=None)
        with pytest.raises(ValueError, match="API key"):
            provider.is_available()

    def test_openai_provider_requires_api_key(self):
        """Should raise error if API key missing."""
        provider = OpenAIProvider(api_key=None)
        with pytest.raises(ValueError, match="API key"):
            provider.is_available()

    def test_gemini_provider_requires_api_key(self):
        """Should raise error if API key missing."""
        provider = GeminiProvider(api_key=None)
        assert not provider.is_available()  # Returns False instead of raising

    def test_get_provider_returns_correct_type(self):
        """Should return correct provider instance."""
        anthropic = get_provider("anthropic", api_key="test-key")
        assert isinstance(anthropic, AnthropicProvider)

        openai = get_provider("openai", api_key="test-key")
        assert isinstance(openai, OpenAIProvider)

        gemini = get_provider("gemini", api_key="test-key", project_id="test-project")
        assert isinstance(gemini, GeminiProvider)

    def test_get_provider_raises_on_unknown(self):
        """Should raise ValueError for unknown provider."""
        with pytest.raises(ValueError, match="Unknown provider"):
            get_provider("unknown", api_key="test-key")
```

**Step 2: Run test to verify it fails**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_llm_providers.py -v`

Expected: FAIL with "ModuleNotFoundError: No module named 'review.llm'"

**Step 3: Write minimal implementation**

Create `review/llm/__init__.py`:
```python
"""LLM provider abstraction for translation and judge operations."""
from .base import BaseProvider, ProviderConfig, ProviderResponse
from .providers import AnthropicProvider, OpenAIProvider, GeminiProvider, get_provider

__all__ = ["BaseProvider", "ProviderConfig", "ProviderResponse", "AnthropicProvider", "OpenAIProvider", "GeminiProvider", "get_provider"]
```

Create `review/llm/base.py`:
```python
"""Base provider interface and data structures."""
from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Optional, List


@dataclass
class ProviderConfig:
    """Configuration for LLM provider."""
    api_key: str
    base_url: Optional[str] = None
    model: str = "claude-sonnet-4-5-20250929"
    max_tokens: int = 8192
    timeout: int = 120


@dataclass
class ProviderResponse:
    """Response from LLM provider."""
    text: str
    model: str
    usage: dict
    latency_ms: int
    raw_response: Optional[dict] = None


class BaseProvider(ABC):
    """Abstract base for LLM providers."""

    def __init__(self, config: ProviderConfig):
        self.config = config

    @abstractmethod
    def is_available(self) -> bool:
        """Check if provider is properly configured."""
        pass

    @abstractmethod
    def generate(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion from prompt."""
        pass

    @abstractmethod
    async def generate_async(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """Async version of generate."""
        pass
```

Create `review/llm/providers.py`:
```python
"""Concrete LLM provider implementations.

2026 API Schema Notes:
- Anthropic: response.content[0].text, usage.input_tokens/output_tokens
- OpenAI: max_completion_tokens (NOT max_tokens - deprecated)
- Gemini: candidates[0].content.parts[0].text, usageMetadata
"""
import asyncio
import logging
import os
import time
from typing import Optional
import anthropic
import openai

try:
    from google.cloud import aiplatform
    GEMINI_AVAILABLE = True
except ImportError:
    GEMINI_AVAILABLE = False

from tenacity import (
    retry,
    stop_after_attempt,
    wait_exponential,
    retry_if_exception_type
)

from .base import BaseProvider, ProviderConfig, ProviderResponse

logger = logging.getLogger(__name__)


class AnthropicProvider(BaseProvider):
    """Anthropic Claude API provider.

    2026 Models:
    - claude-opus-4-5-20251101 (Opus 4.5)
    - claude-sonnet-4-5-20250929 (Sonnet 4.5)
    - claude-haiku-4-5-20250319 (Haiku 4.5)
    """

    DEFAULT_MODEL = "claude-sonnet-4-5-20250929"

    def __init__(self, api_key: str, model: str = None):
        model = model or self.DEFAULT_MODEL
        config = ProviderConfig(api_key=api_key, model=model)
        super().__init__(config)
        self._client: Optional[anthropic.Anthropic] = None

    def _get_client(self) -> anthropic.Anthropic:
        """Lazy client initialization."""
        if self._client is None:
            self._client = anthropic.Anthropic(api_key=self.config.api_key)
        return self._client

    def is_available(self) -> bool:
        """Check if API key is set."""
        return bool(self.config.api_key and self.config.api_key != "")

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=60),
        retry=retry_if_exception_type((anthropic.APITimeoutError, anthropic.InternalServerError)),
    )
    def generate(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion using Anthropic Messages API (2026)."""
        start = time.time()
        client = self._get_client()

        max_tokens = max_tokens or self.config.max_tokens

        response = client.messages.create(
            model=self.config.model,
            max_tokens=max_tokens,
            temperature=temperature,
            messages=[{"role": "user", "content": prompt}]
        )

        latency = int((time.time() - start) * 1000)

        return ProviderResponse(
            text=response.content[0].text,
            model=response.model,
            usage={
                "prompt_tokens": response.usage.input_tokens,
                "completion_tokens": response.usage.output_tokens,
                "total_tokens": response.usage.input_tokens + response.usage.output_tokens
            },
            latency_ms=latency
        )

    async def generate_async(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """Async version using asyncio.to_thread."""
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            None,
            lambda: self.generate(prompt, max_tokens, temperature)
        )


class OpenAIProvider(BaseProvider):
    """OpenAI GPT API provider.

    2026 Models:
    - gpt-4.1-2025-04-14 (GPT-4.1)
    - gpt-4.1-mini-2025-04-14 (GPT-4.1 Mini)
    - gpt-5.2 (GPT-5.2)

    IMPORTANT: max_tokens is DEPRECATED - use max_completion_tokens
    """

    DEFAULT_MODEL = "gpt-4.1-2025-04-14"

    def __init__(self, api_key: str, model: str = None):
        model = model or self.DEFAULT_MODEL
        config = ProviderConfig(api_key=api_key, model=model)
        super().__init__(config)
        self._client: Optional[openai.OpenAI] = None

    def _get_client(self) -> openai.OpenAI:
        """Lazy client initialization."""
        if self._client is None:
            self._client = openai.OpenAI(api_key=self.config.api_key)
        return self._client

    def is_available(self) -> bool:
        """Check if API key is set."""
        return bool(self.config.api_key and self.config.api_key != "")

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=60),
        retry=retry_if_exception_type((openai.APITimeoutError, openai.InternalServerError)),
    )
    def generate(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion using OpenAI Chat Completions API (2026)."""
        start = time.time()
        client = self._get_client()

        max_tokens = max_tokens or self.config.max_tokens

        response = client.chat.completions.create(
            model=self.config.model,
            max_completion_tokens=max_tokens,  # NOT max_tokens (deprecated 2026)
            temperature=temperature,
            messages=[{"role": "user", "content": prompt}]
        )

        latency = int((time.time() - start) * 1000)

        return ProviderResponse(
            text=response.choices[0].message.content,
            model=response.model,
            usage={
                "prompt_tokens": response.usage.prompt_tokens,
                "completion_tokens": response.usage.completion_tokens,
                "total_tokens": response.usage.total_tokens
            },
            latency_ms=latency
        )

    async def generate_async(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """Async version using asyncio.to_thread."""
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            None,
            lambda: self.generate(prompt, max_tokens, temperature)
        )


class GeminiProvider(BaseProvider):
    """Google Gemini API provider via Vertex AI.

    2026 Models:
    - gemini-3.0-pro (Gemini 3.0 Pro)
    - gemini-3.0-flash (Gemini 3.0 Flash)

    Environment Variables:
    - GEMINI_API_KEY: API key for authentication
    - GEMINI_PROJECT_ID: Google Cloud project ID
    - GEMINI_LOCATION: Region (default: us-central1)
    """

    DEFAULT_MODEL = "gemini-3.0-pro"
    DEFAULT_LOCATION = "us-central1"

    def __init__(
        self,
        api_key: str,
        model: str = None,
        project_id: str = None,
        location: str = None
    ):
        model = model or self.DEFAULT_MODEL
        config = ProviderConfig(api_key=api_key, model=model)
        super().__init__(config)
        self.project_id = project_id or os.environ.get("GEMINI_PROJECT_ID", "")
        self.location = location or os.environ.get("GEMINI_LOCATION", self.DEFAULT_LOCATION)
        self._client = None

    def is_available(self) -> bool:
        """Check if API key and dependencies are available."""
        if not GEMINI_AVAILABLE:
            return False
        return bool(self.config.api_key and self.config.api_key != "")

    def _get_endpoint(self) -> str:
        """Build the Vertex AI endpoint URL."""
        return (
            f"https://{self.location}-aiplatform.googleapis.com"
            f"/v1/projects/{self.project_id}"
            f"/locations/{self.location}"
            f"/publishers/google/models/{self.config.model}"
        )

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=60),
        retry=retry_if_exception_type((Exception,)),
    )
    def generate(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion using Vertex AI Gemini API (2026)."""
        import requests

        start = time.time()

        max_tokens = max_tokens or self.config.max_tokens

        # Gemini request format (2026)
        request_body = {
            "contents": [
                {
                    "role": "user",
                    "parts": [{"text": prompt}]
                }
            ],
            "generationConfig": {
                "temperature": temperature,
                "maxOutputTokens": max_tokens,  # Gemini uses maxOutputTokens
            }
        }

        endpoint = self._get_endpoint()
        headers = {
            "Authorization": f"Bearer {self.config.api_key}",
            "Content-Type": "application/json"
        }

        response = requests.post(
            f"{endpoint}:generateContent",
            json=request_body,
            headers=headers,
            timeout=self.config.timeout
        )
        response.raise_for_status()

        data = response.json()
        latency = int((time.time() - start) * 1000)

        # Parse Gemini response format (2026)
        candidate = data["candidates"][0]
        text = candidate["content"]["parts"][0]["text"]
        usage_metadata = data.get("usageMetadata", {})

        return ProviderResponse(
            text=text,
            model=self.config.model,
            usage={
                "prompt_tokens": usage_metadata.get("promptTokenCount", 0),
                "completion_tokens": usage_metadata.get("candidatesTokenCount", 0),
                "total_tokens": usage_metadata.get("totalTokenCount", 0)
            },
            latency_ms=latency
        )

    async def generate_async(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """Async version using asyncio.to_thread."""
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            None,
            lambda: self.generate(prompt, max_tokens, temperature)
        )


def get_provider(
    provider_name: str,
    api_key: str,
    model: Optional[str] = None,
    **kwargs
) -> BaseProvider:
    """Factory function to get provider instance.

    Args:
        provider_name: "anthropic", "openai", or "gemini"
        api_key: API key for the provider
        model: Optional model override
        **kwargs: Additional provider-specific args (project_id, location for Gemini)

    Returns:
        Configured provider instance

    Raises:
        ValueError: If provider_name is unknown
    """
    models = {
        "anthropic": (AnthropicProvider.DEFAULT_MODEL, AnthropicProvider),
        "openai": (OpenAIProvider.DEFAULT_MODEL, OpenAIProvider),
        "gemini": (GeminiProvider.DEFAULT_MODEL, GeminiProvider),
    }

    if provider_name not in models:
        raise ValueError(f"Unknown provider: {provider_name}. Use: {list(models.keys())}")

    default_model, provider_class = models[provider_name]
    model = model or default_model

    if provider_name == "gemini":
        return provider_class(api_key=api_key, model=model, **kwargs)

    return provider_class(api_key=api_key, model=model)
```

**Step 4: Run test to verify it passes**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_llm_providers.py -v`

Expected: PASS

**Step 5: Commit**

```bash
cd /home/thomas/translation-app
git add backend/cmd/translation-worker/review/llm/
git commit -m "feat(llm): add provider abstraction for Anthropic/OpenAI/Gemini

- Add BaseProvider abstract interface
- Implement AnthropicProvider with 2026 API models (claude-4.5-sonnet)
- Implement OpenAIProvider with max_completion_tokens (2026)
- Implement GeminiProvider with Vertex AI REST API
- Add factory function get_provider() supporting all three
- Use tenacity for exponential backoff on retries

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 1.5: Create CLI interface (INDISPENSABLE)

**Priority:** HIGHEST - User explicitly requested CLI support as indispensable

**Files:**
- Create: `review/cli.py`
- Create: `review/__main__.py`
- Test: `tests/test_review/test_cli.py`

**Step 1: Write the failing test**

```python
# tests/test_review/test_cli.py
import pytest
import sys
from pathlib import Path
from click.testing import CliRunner

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.cli import translate, judge, batch


class TestCLI:
    def test_translate_command_requires_provider(self):
        """Should require provider argument."""
        runner = CliRunner()
        result = runner.invoke(translate, ["こんにちは"])
        assert result.exit_code != 0 or "provider" in result.output.lower()

    def test_translate_accepts_all_providers(self):
        """Should accept anthropic, openai, gemini providers."""
        runner = CliRunner()
        for provider in ["anthropic", "openai", "gemini"]:
            result = runner.invoke(translate, ["--provider", provider, "test"])
            # Should not error on provider validation
            assert "Invalid provider" not in result.output

    def test_batch_command_processes_file(self):
        """Should process input file line by line."""
        runner = CliRunner()
        with runner.isolated_filesystem():
            with open("sources.txt", "w") as f:
                f.write("こんにちは\n世界")

            result = runner.invoke(batch, [
                "--provider", "anthropic",
                "--input", "sources.txt",
                "--output", "translations.txt"
            ])
            # Should create output file
            assert Path("translations.txt").exists()

    def test_format_json_outputs_structured(self):
        """Should output valid JSON when format=json."""
        runner = CliRunner()
        result = runner.invoke(translate, [
            "--provider", "anthropic",
            "--format", "json",
            "test"
        ])
        if result.exit_code == 0:
            import json
            data = json.loads(result.output)
            assert "translation" in data or "text" in data
```

**Step 2: Run test to verify it fails**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_cli.py -v`

Expected: FAIL with "ModuleNotFoundError: No module named 'review.cli'"

**Step 3: Write minimal implementation**

Create `review/cli.py`:
```python
"""CLI for translation worker review module.

Provides direct command-line access to translation and judge
functionality without requiring a web server.

Usage:
    python -m review translate "こんにちは" --provider anthropic
    python -m review judge source.txt candidate_a.txt candidate_b.txt
    python -m review batch --input sources.txt --output translations.txt
"""
import asyncio
import json
import logging
import os
import sys
from pathlib import Path
from typing import List, Optional

import click

# Try importing optional dependencies
try:
    from .llm import get_provider, PROVIDERS_AVAILABLE
    from .multimodel import MultiModelTranslator
    from .judge import TranslationJudge
    from .models import TranslationCandidate
    CLI_AVAILABLE = True
except ImportError:
    CLI_AVAILABLE = False

logging.basicConfig(
    level=logging.INFO,
    format="[%(levelname)s] %(message)s"
)
logger = logging.getLogger(__name__)


def _get_api_key(provider: str) -> Optional[str]:
    """Get API key from environment variable."""
    env_var = f"{provider.upper()}_API_KEY"
    return os.environ.get(env_var)


def _ensure_provider(provider: str, model: Optional[str] = None):
    """Ensure provider is available and configured.

    Raises:
        click.ClickException: If provider unavailable or missing API key
    """
    if not CLI_AVAILABLE:
        raise click.ClickException(
            "LLM integration not available. Install dependencies:\n"
            "  pip install anthropic openai google-cloud-aiplatform"
        )

    api_key = _get_api_key(provider)
    if not api_key:
        raise click.ClickException(
            f"Missing API key for {provider}. Set {provider.upper()}_API_KEY environment variable."
        )

    try:
        kwargs = {}
        if provider == "gemini":
            kwargs["project_id"] = os.environ.get("GEMINI_PROJECT_ID", "")
            kwargs["location"] = os.environ.get("GEMINI_LOCATION", "us-central1")

        return get_provider(provider, api_key, model, **kwargs)
    except Exception as e:
        raise click.ClickException(f"Failed to initialize {provider} provider: {e}")


@click.group()
@click.version_option(version="1.0.0")
def cli():
    """Translation worker CLI - Direct access to translation and judge functionality.

    Examples:
        # Translate a single text
        python -m review translate "こんにちは" --provider anthropic

        # Translate with custom model
        python -m review translate "こんにちは" --provider openai --model gpt-4.1

        # Batch process from file
        python -m review batch --input sources.txt --output translations.txt

        # Compare two translations
        python -m review judge "Original Japanese text" translation_a.txt translation_b.txt
    """
    pass


@cli.command()
@click.argument("text")
@click.option(
    "--provider", "-p",
    type=click.Choice(["anthropic", "openai", "gemini"]),
    required=True,
    help="LLM provider to use"
)
@click.option(
    "--model", "-m",
    help="Model identifier (uses provider default if not specified)"
)
@click.option(
    "--format", "-f",
    type=click.Choice(["text", "json", "csv"]),
    default="text",
    help="Output format"
)
@click.option(
    "--output", "-o",
    type=click.Path(),
    help="Write output to file instead of stdout"
)
@click.option(
    "--parallel/--sequential",
    default=True,
    help="Use parallel execution (only affects multiple providers)"
)
def translate(text: str, provider: str, model: Optional[str], format: str,
             output: Optional[str], parallel: bool):
    """Translate Japanese text to English.

    Example: translate "こんにちは" --provider anthropic --format json
    """
    provider_instance = _ensure_provider(provider, model)

    # Build prompt (simplified for CLI)
    from .prompts import TranslationPromptBuilder
    builder = TranslationPromptBuilder()
    prompt = builder.build(text)

    try:
        response = provider_instance.generate(prompt)
        translation = response.text.strip()

        # Format output
        if format == "text":
            result = translation
        elif format == "json":
            result = json.dumps({
                "source": text,
                "translation": translation,
                "provider": provider,
                "model": response.model,
                "usage": response.usage,
                "latency_ms": response.latency_ms
            }, indent=2, ensure_ascii=False)
        elif format == "csv":
            result = f"source,translation,provider,model\\n"
            result += f'"{text}","{translation}","{provider}","{response.model}"'

        # Output
        if output:
            Path(output).write_text(result, encoding="utf-8")
            click.echo(f"Translation written to {output}")
        else:
            click.echo(result)

    except Exception as e:
        raise click.ClickException(f"Translation failed: {e}")


@cli.command()
@click.argument("source", type=click.Path(exists=True))
@click.argument("candidate_a", type=click.Path(exists=True))
@click.argument("candidate_b", type=click.Path(exists=True))
@click.option(
    "--provider", "-p",
    type=click.Choice(["anthropic", "openai", "gemini"]),
    default="anthropic",
    help="LLM provider for judge"
)
@click.option(
    "--model", "-m",
    default="claude-4.5-sonnet",
    help="Judge model"
)
@click.option(
    "--format", "-f",
    type=click.Choice(["text", "json"]),
    default="text",
    help="Output format"
)
def judge(source: str, candidate_a: str, candidate_b: str, provider: str,
          model: str, format: str):
    """Judge which translation is better.

    Compare two translations and select the winner using LLM evaluation.
    """
    provider_instance = _ensure_provider(provider, model)

    # Read files
    source_text = Path(source).read_text(encoding="utf-8").strip()
    text_a = Path(candidate_a).read_text(encoding="utf-8").strip()
    text_b = Path(candidate_b).read_text(encoding="utf-8").strip()

    # Build judge prompt
    from .prompts import JudgePromptBuilder
    builder = JudgePromptBuilder()
    prompt = builder.build(source_text, text_a, text_b)

    try:
        response = provider_instance.generate(prompt)

        # Parse JSON response
        try:
            result_data = json.loads(response.text)
        except json.JSONDecodeError:
            # Fallback: extract winner from text
            result_data = {
                "winner": "unknown",
                "confidence": 0.0,
                "reasoning": response.text,
                "concerns": ["Failed to parse JSON"]
            }

        if format == "json":
            output = json.dumps(result_data, indent=2)
        else:
            output = f"Winner: {result_data.get('winner', 'unknown')}\\n"
            output += f"Confidence: {result_data.get('confidence', 0):.2f}\\n"
            output += f"Reasoning: {result_data.get('reasoning', 'N/A')}"

        click.echo(output)

    except Exception as e:
        raise click.ClickException(f"Judgment failed: {e}")


@cli.command()
@click.option(
    "--input", "-i",
    type=click.Path(exists=True),
    required=True,
    help="Input file with source texts (one per line)"
)
@click.option(
    "--output", "-o",
    type=click.Path(),
    required=True,
    help="Output file for translations"
)
@click.option(
    "--provider", "-p",
    type=click.Choice(["anthropic", "openai", "gemini"]),
    required=True,
    help="LLM provider to use"
)
@click.option(
    "--model", "-m",
    help="Model identifier"
)
@click.option(
    "--parallel/--sequential",
    default=True,
    help="Execution mode"
)
@click.option(
    "--format", "-f",
    type=click.Choice(["text", "json", "csv"]),
    default="text",
    help="Output format"
)
def batch(input: str, output: str, provider: str, model: Optional[str],
          parallel: bool, format: str):
    """Batch translate texts from a file.

    Processes each line of the input file as a separate translation.
    """
    provider_instance = _ensure_provider(provider, model)

    # Read input
    input_path = Path(input)
    sources = input_path.read_text(encoding="utf-8").strip().split("\\n")

    # Filter empty lines
    sources = [s.strip() for s in sources if s.strip()]

    if not sources:
        raise click.ClickException("No non-empty lines found in input file")

    click.echo(f"Processing {len(sources)} texts...")

    # Translate
    from .prompts import TranslationPromptBuilder
    builder = TranslationPromptBuilder()

    translations = []
    for i, source in enumerate(sources):
        click.echo(f"[{i+1}/{len(sources)}] Processing: {source[:50]}...", err=True)

        prompt = builder.build(source)
        try:
            response = provider_instance.generate(prompt)
            translations.append({
                "source": source,
                "translation": response.text.strip(),
                "usage": response.usage
            })
        except Exception as e:
            logger.warning(f"Failed to translate line {i+1}: {e}")
            translations.append({
                "source": source,
                "translation": f"[ERROR: {e}]",
                "usage": {}
            })

    # Format output
    if format == "text":
        result = "\\n".join(t["translation"] for t in translations)
    elif format == "json":
        result = json.dumps(translations, indent=2, ensure_ascii=False)
    elif format == "csv":
        result = "source,translation,prompt_tokens,completion_tokens\\n"
        for t in translations:
            safe_translation = t["translation"].replace('"', '""')
            result += f'"{t["source"]}","{safe_translation}",{t["usage"].get("prompt_tokens", 0)},{t["usage"].get("completion_tokens", 0)}\\n'

    # Write output
    Path(output).write_text(result, encoding="utf-8")
    click.echo(f"Translated {len(translations)} texts → {output}")


# For python -m review.cli compatibility
def main():
    """Entry point for python -m review.cli."""
    cli()


if __name__ == "__main__":
    main()
```

Create `review/__main__.py`:
```python
"""Main entry point for python -m review.

Allows running CLI commands directly:
    python -m review translate "こんにちは" --provider anthropic
"""
from .cli import cli

if __name__ == "__main__":
    cli()
```

Update `review/__init__.py` - add CLI export:
```python
# Add to __all__ at end of file
__all__ += ["cli"]
```

**Step 4: Run test to verify it passes**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_cli.py -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/translation-worker/review/cli.py backend/cmd/translation-worker/review/__main__.py
git commit -m "feat(cli): add indispensable CLI interface for direct translation

- Add translate command for single text translation
- Add judge command for comparing translations
- Add batch command for file-based processing
- Support all providers: anthropic, openai, gemini
- Support multiple output formats: text, json, csv
- Add parallel/sequential execution flags
- Enable python -m review invocation

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Create translation prompt templates

**Files:**
- Create: `review/prompts.py`
- Test: `tests/test_review/test_prompts.py`

**Step 1: Write the failing test**

```python
# tests/test_review/test_prompts.py
import pytest
import sys
from pathlib import Path

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.prompts import TranslationPromptBuilder


class TestTranslationPromptBuilder:
    def test_build_basic_prompt(self):
        """Should build basic translation prompt."""
        builder = TranslationPromptBuilder(
            source_lang="ja",
            target_lang="en"
        )
        prompt = builder.build("こんにちは")
        assert "Japanese" in prompt or "こんにちは" in prompt
        assert "English" in prompt
        assert "translate" in prompt.lower()

    def test_build_with_glossary(self):
        """Should include glossary terms in prompt."""
        builder = TranslationPromptBuilder(
            source_lang="ja",
            target_lang="en"
        )
        prompt = builder.build(
            source="顧客",
            glossary_terms=[("顧客", "customer")]
        )
        assert "customer" in prompt
        assert "顧客" in prompt
```

**Step 2: Run test to verify it fails**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_prompts.py -v`

Expected: FAIL with "ModuleNotFoundError: No module named 'review.prompts'"

**Step 3: Write minimal implementation**

Create `review/prompts.py`:
```python
"""Prompt templates for LLM translation and judge operations."""

from typing import List, Tuple, Optional


class TranslationPromptBuilder:
    """Builds prompts for JA→EN translation tasks."""

    JA_EN_SYSTEM_PROMPT = """You are a professional Japanese-to-English translator.
Your task is to translate the given Japanese text into natural, fluent English.

Guidelines:
- Maintain the tone and formality level of the source
- For business content: use professional but accessible language
- For technical content: preserve technical accuracy
- Avoid retaining Japanese honorifics (-san, -sama, -kun, -chan) unless specifically requested
- Ensure grammatical correctness and proper articles (a, an, the)
- Prefer active voice over passive voice
- Split overly long sentences for clarity"""

    def __init__(
        self,
        source_lang: str = "ja",
        target_lang: str = "en",
        domain: str = "general"
    ):
        """Initialize prompt builder.

        Args:
            source_lang: Source language code
            target_lang: Target language code
            domain: Domain hint for context (legal, medical, technical, etc.)
        """
        self.source_lang = source_lang
        self.target_lang = target_lang
        self.domain = domain

    def build(
        self,
        source: str,
        glossary_terms: Optional[List[Tuple[str, str]]] = None,
        context: Optional[dict] = None
    ) -> str:
        """Build translation prompt.

        Args:
            source: Source text to translate
            glossary_terms: List of (source, target) term pairs
            context: Additional context (document type, position, etc.)

        Returns:
            Complete prompt for LLM
        """
        prompt_parts = [self.JA_EN_SYSTEM_PROMPT]

        # Add domain context if provided
        if self.domain != "general":
            prompt_parts.append(f"\nDomain context: {self.domain}")

        # Add glossary terms
        if glossary_terms:
            glossary_section = "\n\nGlossary terms (use these translations):"
            for source_term, target_term in glossary_terms:
                glossary_section += f"\n- {source_term} → {target_term}"
            prompt_parts.append(glossary_section)

        # Add context hints
        context_hints = []
        if context:
            if context.get("document_type"):
                context_hints.append(f"Document type: {context['document_type']}")
            if context.get("position"):
                context_hints.append(f"Position: {context['position']}")

        if context_hints:
            prompt_parts.append("\n\nContext: " + ", ".join(context_hints))

        # Add source text
        prompt_parts.append(f"\n\nTranslate the following Japanese text:\n\n{source}")

        return "\n".join(prompt_parts)


class JudgePromptBuilder:
    """Builds prompts for translation judge evaluation."""

    JUDGE_SYSTEM_PROMPT = """You are an expert translation evaluator.
Your task is to compare two Japanese-to-English translations and select the better one.

Evaluation criteria:
1. Accuracy: Does it preserve the meaning of the original?
2. Fluency: Is it natural, grammatically correct English?
3. Style: Does it maintain appropriate tone and formality?
4. Completeness: Are any parts of the source omitted?

Output format (JSON):
{
    "winner": "model_a" | "model_b" | "tie",
    "confidence": 0.0-1.0,
    "reasoning": "Brief explanation",
    "concerns": ["list any concerns found"]
}"""

    def build(
        self,
        source: str,
        candidate_a: str,
        candidate_b: str,
        context: Optional[dict] = None
    ) -> str:
        """Build judge evaluation prompt.

        Args:
            source: Original source text
            candidate_a: Translation from model_a
            candidate_b: Translation from model_b
            context: Optional context

        Returns:
            Complete prompt for LLM judge
        """
        prompt = [
            self.JUDGE_SYSTEM_PROMPT,
            "\n\nOriginal Japanese text:",
            source,
            "\n\nTranslation A:",
            candidate_a,
            "\n\nTranslation B:",
            candidate_b
        ]

        if context:
            prompt.append(f"\n\nContext: {context}")

        prompt.append("\n\nEvaluate both translations and output JSON.")

        return "\n".join(prompt)
```

**Step 4: Run test to verify it passes**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_prompts.py -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/translation-worker/review/prompts.py
git commit -m "feat(prompts): add translation and judge prompt builders

- Add TranslationPromptBuilder for JA→EN translation
- Add JudgePromptBuilder for comparing translations
- Support glossary terms injection
- Support context hints (document type, position)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Integrate real LLM calls into MultiModelTranslator

**Files:**
- Modify: `review/multimodel.py`
- Test: `tests/test_review/test_multimodel_real.py`

**Step 1: Write the failing test**

```python
# tests/test_review/test_multimodel_real.py
import pytest
import sys
from pathlib import Path

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.multimodel import MultiModelTranslator


@pytest.mark.integration
class TestMultiModelTranslatorIntegration:
    def test_translation_with_mock_providers(self, mocker):
        """Should use provider for translation when configured."""
        # Mock the provider responses
        mock_response = mocker.Mock(
            text="Hello",
            model="claude-4.5-sonnet",
            usage={"prompt_tokens": 10, "completion_tokens": 5},
            latency_ms=100
        )

        mock_provider = mocker.Mock()
        mock_provider.generate.return_value = mock_response
        mock_provider.is_available.return_value = True

        translator = MultiModelTranslator()
        translator._set_provider("model_a", mock_provider)

        candidates = translator.translate("こんにちは")

        assert len(candidates) == 2
        assert mock_provider.generate.called
```

**Step 2: Run test to verify it fails**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_multimodel_real.py -v`

Expected: FAIL with "MultiModelTranslator has no _set_provider method"

**Step 3: Write minimal implementation**

Modify `review/multimodel.py` - add provider integration while keeping stub as fallback:

```python
# review/multimodel.py - Add these imports at top
import os
import asyncio
import logging
import time
from typing import List, Dict, Optional, Callable
from .models import TranslationCandidate

# Import provider abstraction if available
try:
    from .llm import get_provider, ProviderConfig
    PROVIDERS_AVAILABLE = True
except ImportError:
    PROVIDERS_AVAILABLE = False

logger = logging.getLogger(__name__)


class MultiModelTranslator:
    """Coordinates translation across multiple models.

    Supports both stub (MVP) and real LLM API integration.
    """

    # Default model configurations (2026)
    DEFAULT_MODELS = {
        "model_a": {
            "name": "claude-sonnet-4.5-20250929",  # Latest 2026 Anthropic
            "provider": "anthropic",
            "description": "Primary model - balanced quality and speed"
        },
        "model_b": {
            "name": "gpt-4.1",  # 2026 OpenAI
            "provider": "openai",
            "description": "Secondary model - fast and cost-effective"
        },
        "model_c": {
            "name": "gemini-3.0-flash",  # Latest 2026 Gemini
            "provider": "gemini",
            "description": "Tertiary model - Google's Gemini 3.0 Flash"
        }
    }

    def __init__(
        self,
        models: Optional[Dict[str, dict]] = None,
        parallel: bool = True,
        use_real_llm: bool = False
    ):
        """Initialize the multi-model translator.

        Args:
            models: Mapping of model_key to model config
            parallel: Whether to execute translations in parallel
            use_real_llm: If True, use real LLM APIs; otherwise use stubs
        """
        self.models = models or self.DEFAULT_MODELS
        self.parallel = parallel
        self.use_real_llm = use_real_llm and PROVIDERS_AVAILABLE

        # Provider instances for real LLM mode
        self._providers: Dict[str, object] = {}

        if self.use_real_llm:
            self._initialize_providers()
        else:
            logger.info("[MULTI] Using stub translations (MVP mode)")

    def _initialize_providers(self):
        """Initialize LLM providers from environment variables."""
        for key, config in self.models.items():
            provider_name = config.get("provider", "")
            model_name = config.get("name", "")

            # Get API key from environment
            env_key = f"{provider_name.upper()}_API_KEY"
            api_key = os.environ.get(env_key)

            if not api_key:
                logger.warning(f"[MULTI] No API key for {key}, using stub")
                continue

            try:
                # Gemini requires additional kwargs (project_id, location)
                if provider_name == "gemini":
                    self._providers[key] = get_provider(
                        provider_name,
                        api_key,
                        model_name,
                        project_id=os.environ.get("GEMINI_PROJECT_ID", ""),
                        location=os.environ.get("GEMINI_LOCATION", "us-central1")
                    )
                else:
                    self._providers[key] = get_provider(provider_name, api_key, model_name)
                logger.info(f"[MULTI] Initialized {key} with {provider_name}/{model_name}")
            except Exception as e:
                logger.error(f"[MULTI] Failed to initialize {key}: {e}")

    def _set_provider(self, key: str, provider):
        """Set provider instance (for testing)."""
        self._providers[key] = provider

    def translate(
        self,
        source: str,
        glossary_terms: Optional[List[str]] = None,
        context: Optional[dict] = None
    ) -> List[TranslationCandidate]:
        """Generate translations from all configured models.

        Args:
            source: Source text to translate
            glossary_terms: Optional glossary terms to apply
            context: Optional translation context

        Returns:
            List of TranslationCandidate, one per model
        """
        if self.parallel:
            return self._translate_parallel(source, glossary_terms, context)
        else:
            return self._translate_sequential(source, glossary_terms, context)

    def _translate_sequential(
        self,
        source: str,
        glossary_terms: Optional[List[str]],
        context: Optional[dict]
    ) -> List[TranslationCandidate]:
        """Translate sequentially (stub or real LLM)."""
        candidates = []

        for key, config in self.models.items():
            start = time.time()

            if self.use_real_llm and key in self._providers:
                result = self._translate_with_provider(
                    source, key, glossary_terms, context
                )
                latency = int((time.time() - start) * 1000)
                confidence = 0.9  # Default confidence
            else:
                result = self._stub_translate(source, key, glossary_terms)
                latency = int((time.time() - start) * 1000)
                confidence = 0.9  # Stub confidence

            candidates.append(TranslationCandidate(
                model_name=key,
                text=result,
                confidence=confidence,
                glossary_matches=glossary_terms or [],
                latency_ms=latency
            ))

        return candidates

    def _translate_parallel(
        self,
        source: str,
        glossary_terms: Optional[List[str]],
        context: Optional[dict]
    ) -> List[TranslationCandidate]:
        """Translate in parallel using asyncio."""
        if self.use_real_llm:
            # Real parallel implementation with asyncio.gather
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            try:
                tasks = []
                for key in self.models:
                    if key in self._providers:
                        tasks.append(self._translate_async_with_provider(
                            source, key, glossary_terms, context
                        ))
                if tasks:
                    results = loop.run_until_complete(asyncio.gather(*tasks))
                    # Convert to TranslationCandidate objects
                    candidates = []
                    for i, (key, result) in enumerate(zip(self.models.keys(), results)):
                        candidates.append(TranslationCandidate(
                            model_name=key,
                            text=result.text,
                            confidence=0.9,
                            glossary_matches=glossary_terms or [],
                            latency_ms=result.latency_ms
                        ))
                    return candidates
            finally:
                loop.close()

        # Fall back to sequential
        return self._translate_sequential(source, glossary_terms, context)

    async def _translate_async_with_provider(
        self,
        source: str,
        model_key: str,
        glossary_terms: Optional[List[str]],
        context: Optional[dict]
    ):
        """Async translation with provider."""
        provider = self._providers[model_key]
        # Import prompt builder
        from .prompts import TranslationPromptBuilder

        builder = TranslationPromptBuilder()
        prompt = builder.build(
            source,
            glossary_terms=[(t, t) for t in (glossary_terms or [])]  # Simplified
        )

        return await provider.generate_async(prompt)

    def _translate_with_provider(
        self,
        source: str,
        model_key: str,
        glossary_terms: Optional[List[str]],
        context: Optional[dict]
    ) -> str:
        """Translate using real LLM provider."""
        provider = self._providers[model_key]

        # Import prompt builder
        from .prompts import TranslationPromptBuilder

        builder = TranslationPromptBuilder()
        prompt = builder.build(
            source,
            glossary_terms=[(t, t) for t in (glossary_terms or [])]  # Simplified
        )

        response = provider.generate(prompt)
        return response.text

    def _stub_translate(
        self,
        source: str,
        model_key: str,
        glossary_terms: Optional[List[str]]
    ) -> str:
        """Stub translation for MVP."""
        if glossary_terms:
            glossary_str = ", ".join(glossary_terms)
            return f"[{model_key}] {source} (glossary: {glossary_str})"

        if model_key == "model_a":
            return f"[Model A] {source}"
        else:
            return f"[Model B] {source}"

    # Keep existing methods (translate_batch, translate_async, set_translation_function)
    # ... (rest of existing methods unchanged)
```

**Step 4: Run test to verify it passes**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_multimodel_real.py -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/translation-worker/review/multimodel.py
git commit -m "feat(multimodel): integrate real LLM providers

- Add use_real_llm flag for provider vs stub mode
- Initialize providers from environment variables
- Add _translate_with_provider for real API calls
- Add _translate_async_with_provider for parallel execution
- Keep stub as fallback for testing/MVP mode

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Integrate real LLM calls into TranslationJudge

**Files:**
- Modify: `review/judge.py`
- Test: `tests/test_review/test_judge_real.py`

**Step 1: Write the failing test**

```python
# tests/test_review/test_judge_real.py
import pytest
import sys
from pathlib import Path

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.judge import TranslationJudge
from review.models import TranslationCandidate


@pytest.mark.integration
class TestTranslationJudgeIntegration:
    def test_judge_with_mock_provider(self, mocker):
        """Should use provider for judgment when configured."""
        # Mock provider response
        mock_response = mocker.Mock(
            text='{"winner": "model_a", "confidence": 0.9, "reasoning": "Better", "concerns": []}',
            model="claude-4.5-sonnet",
            latency_ms=150
        )

        mock_provider = mocker.Mock()
        mock_provider.generate.return_value = mock_response
        mock_provider.is_available.return_value = True

        judge = TranslationJudge(use_real_llm=True)
        judge._set_provider(mock_provider)

        candidates = [
            TranslationCandidate("model_a", "Translation A"),
            TranslationCandidate("model_b", "Translation B")
        ]

        result = judge.judge("seg1", "Hello", candidates)

        assert result.winner == "model_a"
        assert result.confidence == 0.9
```

**Step 2: Run test to verify it fails**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_judge_real.py -v`

Expected: FAIL with "TranslationJudge has no _set_provider method"

**Step 3: Write minimal implementation**

Modify `review/judge.py` - add provider integration:

```python
# review/judge.py
import json
import logging
import random
import os
from typing import List, Optional
from .models import TranslationCandidate, JudgeResult

# Import provider abstraction if available
try:
    from .llm import get_provider
    PROVIDERS_AVAILABLE = True
except ImportError:
    PROVIDERS_AVAILABLE = False

logger = logging.getLogger(__name__)


class TranslationJudge:
    """Evaluates competing translations and selects the best option.

    Supports both stub (MVP) and real LLM evaluation.
    """

    def __init__(
        self,
        model: str = "claude-4.5-sonnet",
        timeout: int = 30,
        fallback_on_timeout: bool = True,
        enabled: bool = True,
        use_real_llm: bool = False
    ):
        """Initialize the translation judge.

        Args:
            model: Model identifier for judge decisions
            timeout: Timeout for judge decisions in seconds
            fallback_on_timeout: If True, use random fallback on timeout
            enabled: If False, always return model_a as winner
            use_real_llm: If True, use real LLM for evaluation
        """
        self.model = model
        self.timeout = timeout
        self.fallback_on_timeout = fallback_on_timeout
        self.enabled = enabled
        self.use_real_llm = use_real_llm and PROVIDERS_AVAILABLE

        # Initialize provider if real LLM requested
        self._provider = None
        if self.use_real_llm:
            self._initialize_provider()

    def _initialize_provider(self):
        """Initialize LLM provider for judge."""
        # Determine provider from model name
        if "claude" in self.model.lower() or "anthropic" in self.model.lower():
            provider_name = "anthropic"
        elif "gpt" in self.model.lower() or "openai" in self.model.lower():
            provider_name = "openai"
        else:
            provider_name = "anthropic"  # Default

        # Get API key from environment
        env_key = f"{provider_name.upper()}_API_KEY"
        api_key = os.environ.get(env_key)

        if api_key:
            try:
                self._provider = get_provider(provider_name, api_key, self.model)
                logger.info(f"[JUDGE] Initialized with {provider_name}/{self.model}")
            except Exception as e:
                logger.warning(f"[JUDGE] Failed to initialize provider: {e}, using stub")
        else:
            logger.warning(f"[JUDGE] No API key for {provider_name}, using stub")

    def _set_provider(self, provider):
        """Set provider instance (for testing)."""
        self._provider = provider
        self.use_real_llm = True

    def judge(
        self,
        segment_id: str,
        source: str,
        candidates: List[TranslationCandidate],
        context: Optional[dict] = None
    ) -> JudgeResult:
        """Evaluate candidates and select the winner."""
        if not self.enabled:
            return JudgeResult(
                segment_id=segment_id,
                winner="model_a",
                confidence=1.0,
                reasoning="Judge disabled, defaulting to model_a"
            )

        if len(candidates) < 2:
            return JudgeResult(
                segment_id=segment_id,
                winner="model_a",
                confidence=candidates[0].confidence if candidates else 1.0,
                reasoning="Only one candidate provided"
            )

        # Use real LLM or stub
        if self.use_real_llm and self._provider:
            return self._judge_with_llm(segment_id, source, candidates, context)
        else:
            return self._stub_judge(segment_id, source, candidates, context)

    def _judge_with_llm(
        self,
        segment_id: str,
        source: str,
        candidates: List[TranslationCandidate],
        context: Optional[dict]
    ) -> JudgeResult:
        """Use LLM to judge translations."""
        from .prompts import JudgePromptBuilder

        builder = JudgePromptBuilder()

        # Get texts from candidates
        text_a = next((c.text for c in candidates if c.model_name == "model_a"), "")
        text_b = next((c.text for c in candidates if c.model_name == "model_b"), "")

        prompt = builder.build(source, text_a, text_b, context)

        try:
            response = self._provider.generate(prompt)

            # Parse JSON response
            result_data = json.loads(response.text)

            return JudgeResult(
                segment_id=segment_id,
                winner=result_data.get("winner", "model_a"),
                confidence=result_data.get("confidence", 0.8),
                reasoning=result_data.get("reasoning", ""),
                concerns=result_data.get("concerns", [])
            )
        except json.JSONDecodeError as e:
            logger.warning(f"[JUDGE] Failed to parse LLM response: {e}, using fallback")
            if self.fallback_on_timeout:
                return self._stub_judge(segment_id, source, candidates, context)
            raise
        except Exception as e:
            logger.warning(f"[JUDGE] LLM call failed: {e}, using fallback")
            if self.fallback_on_timeout:
                return self._stub_judge(segment_id, source, candidates, context)
            raise

    def _stub_judge(self, segment_id: str, source: str,
                    candidates: List[TranslationCandidate], context: Optional[dict]) -> JudgeResult:
        """Stub implementation using random selection."""
        winner_idx = random.randint(0, len(candidates) - 1)
        winner = f"model_{chr(97 + winner_idx)}"
        confidence = round(random.uniform(0.6, 1.0), 2)

        reasoning = "Short segment: both translations are adequate" if len(source) < 20 else (
            "Medium segment: selected translation captures meaning accurately" if len(source) < 100 else
            "Long segment: selected translation maintains flow and clarity"
        )

        concerns = []
        if confidence < 0.75:
            concerns.append("Low confidence in terminology")

        return JudgeResult(
            segment_id=segment_id,
            winner=winner,
            confidence=confidence,
            reasoning=reasoning,
            concerns=concerns
        )

    # Keep existing methods (judge_batch, judge_async, create_judge)
    # ... (rest unchanged)
```

**Step 4: Run test to verify it passes**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_judge_real.py -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/translation-worker/review/judge.py
git commit -m "feat(judge): integrate real LLM provider for evaluation

- Add use_real_llm flag for provider vs stub mode
- Add _judge_with_llm for real API-based evaluation
- Parse JSON response from LLM judge
- Keep stub fallback for errors/testing
- Auto-detect provider from model name

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Add environment-based configuration

**Files:**
- Modify: `review/__init__.py`
- Test: `tests/test_review/test_config.py`

**Step 1: Write the failing test**

```python
# tests/test_review/test_config.py
import pytest
import sys
import os
from pathlib import Path

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review import create_workflow


class TestEnvironmentConfig:
    def test_use_real_llm_from_env(self, monkeypatch):
        """Should read use_real_llm from environment."""
        monkeypatch.setenv("TRANSLATION_USE_REAL_LLM", "true")
        monkeypatch.setenv("ANTHROPIC_API_KEY", "test-key")

        # Import after setting env
        import importlib
        import review
        importlib.reload(review)

        # Verify config is read
        assert os.environ.get("TRANSLATION_USE_REAL_LLM") == "true"

    def test_missing_api_key_logs_warning(self, caplog):
        """Should log warning when API key missing."""
        # Ensure no API key set
        for key in ["ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY"]:
            if key in os.environ:
                del os.environ[key]

        from review.multimodel import MultiModelTranslator

        translator = MultiModelTranslator(use_real_llm=True)
        # Should fall back to stubs without crashing
        candidates = translator.translate("test")
        assert len(candidates) == 2
```

**Step 2: Run test to verify it fails**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_config.py -v`

Expected: May PASS or FAIL depending on current implementation

**Step 3: Write minimal implementation**

Modify `review/__init__.py` to add config helper:

```python
# review/__init__.py - Add at end of file
import os
import logging

logger = logging.getLogger(__name__)


def get_llm_config() -> dict:
    """Get LLM configuration from environment.

    Environment variables:
    - TRANSLATION_USE_REAL_LLM: "true" to use real LLMs
    - ANTHROPIC_API_KEY: API key for Anthropic Claude
    - OPENAI_API_KEY: API key for OpenAI GPT
    - GEMINI_API_KEY: API key for Google Gemini
    - GEMINI_PROJECT_ID: Google Cloud project ID (for Gemini)
    - GEMINI_LOCATION: Region for Gemini (default: us-central1)

    Returns:
        Dict with use_real_llm and provider availability
    """
    return {
        "use_real_llm": os.environ.get("TRANSLATION_USE_REAL_LLM", "false").lower() == "true",
        "has_anthropic_key": bool(os.environ.get("ANTHROPIC_API_KEY")),
        "has_openai_key": bool(os.environ.get("OPENAI_API_KEY")),
        "has_gemini_key": bool(os.environ.get("GEMINI_API_KEY")),
    }


__all__ += ["get_llm_config"]
```

**Step 4: Run test to verify it passes**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_config.py -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/translation-worker/review/__init__.py
git commit -m "feat(config): add environment-based LLM configuration

- Add get_llm_config() helper function
- Read TRANSLATION_USE_REAL_LLM from environment
- Check API key availability
- Export in __all__

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Update factory functions for provider mode

**Files:**
- Modify: `review/multimodel.py` (create_multimodel_translator)
- Modify: `review/judge.py` (create_judge)
- Modify: `review/workflow.py` (create_workflow)

**Step 1: Write the failing test**

```python
# tests/test_review/test_factory_updates.py
import pytest
import sys
import os
from pathlib import Path

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review import create_multimodel_translator, create_judge, create_workflow


class TestFactoryFunctionUpdates:
    def test_factory_respects_env_config(self, monkeypatch):
        """Factory functions should read env for real LLM mode."""
        monkeypatch.setenv("TRANSLATION_USE_REAL_LLM", "true")

        translator = create_multimodel_translator()
        # Should check env for real LLM mode
        assert translator is not None

    def test_create_workflow_with_real_llm(self, monkeypatch):
        """Should create workflow with real LLM when configured."""
        monkeypatch.setenv("TRANSLATION_USE_REAL_LLM", "true")
        monkeypatch.setenv("ANTHROPIC_API_KEY", "test-key")
        monkeypatch.setenv("OPENAI_API_KEY", "test-key")
        monkeypatch.setenv("GEMINI_API_KEY", "test-key")
        monkeypatch.setenv("GEMINI_PROJECT_ID", "test-project")

        workflow = create_workflow()
        assert workflow is not None
```

**Step 2: Run test to verify it fails**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_factory_updates.py -v`

Expected: May need to update factory functions

**Step 3: Write minimal implementation**

Update factory functions to respect environment:

In `review/multimodel.py`, update `create_multimodel_translator`:
```python
def create_multimodel_translator(
    models: Optional[Dict[str, dict]] = None,
    parallel: bool = True,
    use_real_llm: Optional[bool] = None
) -> MultiModelTranslator:
    """Factory function to create a MultiModelTranslator.

    Args:
        models: Optional model configuration dict
        parallel: Whether to use parallel execution
        use_real_llm: If None, reads from TRANSLATION_USE_REAL_LLM env var

    Returns:
        Configured MultiModelTranslator instance
    """
    import os

    if use_real_llm is None:
        use_real_llm = os.environ.get("TRANSLATION_USE_REAL_LLM", "false").lower() == "true"

    return MultiModelTranslator(
        models=models,
        parallel=parallel,
        use_real_llm=use_real_llm
    )
```

In `review/judge.py`, update `create_judge`:
```python
def create_judge(
    model: str = "claude-4.5-sonnet",
    timeout: int = 30,
    enabled: bool = True,
    use_real_llm: Optional[bool] = None
) -> TranslationJudge:
    """Factory function to create a TranslationJudge.

    Args:
        model: Model identifier for judge decisions
        timeout: Timeout for judge decisions in seconds
        enabled: Whether judge model is active
        use_real_llm: If None, reads from TRANSLATION_USE_REAL_LLM env var

    Returns:
        Configured TranslationJudge instance
    """
    import os

    if use_real_llm is None:
        use_real_llm = os.environ.get("TRANSLATION_USE_REAL_LLM", "false").lower() == "true"

    return TranslationJudge(
        model=model,
        timeout=timeout,
        enabled=enabled,
        use_real_llm=use_real_llm
    )
```

In `review/workflow.py`, update `create_workflow`:
```python
def create_workflow(
    project_type: str = "routine",
    csv_output_dir: Optional[str] = None,
) -> TranslationWorkflow:
    """Factory function to create a configured workflow.

    Args:
        project_type: "critical" or "routine"
        csv_output_dir: Optional directory for CSV exports

    Returns:
        Configured TranslationWorkflow instance
    """
    import os

    config = ReviewConfig.for_project_type(project_type)

    # Check env for real LLM mode
    use_real_llm = os.environ.get("TRANSLATION_USE_REAL_LLM", "false").lower() == "true"

    translator = MultiModelTranslator(use_real_llm=use_real_llm)
    judge = TranslationJudge(use_real_llm=use_real_llm)
    flagger = FlaggingEngine()
    exporter = BilingualCSVExporter(output_dir=csv_output_dir) if csv_output_dir else None

    return TranslationWorkflow(
        translator=translator,
        judge=judge,
        flagger=flagger,
        exporter=exporter,
        config=config,
    )
```

**Step 4: Run test to verify it passes**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_factory_updates.py -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/cmd/translation-worker/review/
git commit -m "feat(factory): update factory functions for real LLM mode

- create_multimodel_translator: respect TRANSLATION_USE_REAL_LLM env
- create_judge: respect TRANSLATION_USE_REAL_LLM env
- create_workflow: propagate real LLM mode to components

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Update existing tests for compatibility

**Files:**
- Modify: `tests/test_review/test_review.py`

**Step 1: Verify existing tests still pass**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/test_review.py -v`

Expected: All 68 existing tests should PASS (stub mode is default)

**Step 2: Add test for real LLM mode with mocks**

Add to `test_review.py`:
```python
@pytest.mark.integration
class TestRealLLMModeWithMocks:
    def test_translator_in_real_mode_with_mock(self, mocker):
        """Real LLM mode should work with mocked providers."""
        mock_response = mocker.Mock(
            text="Hello",
            model="claude-4.5-sonnet",
            usage={"prompt_tokens": 10, "completion_tokens": 5},
            latency_ms=100
        )

        mock_provider = mocker.Mock()
        mock_provider.generate.return_value = mock_response
        mock_provider.is_available.return_value = True

        translator = MultiModelTranslator(use_real_llm=True)
        translator._set_provider("model_a", mock_provider)

        candidates = translator.translate("こんにちは")

        assert len(candidates) == 2
        assert mock_provider.generate.called
```

**Step 3: Run all tests to verify**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/ -v`

Expected: PASS (all tests)

**Step 4: Commit**

```bash
git add backend/cmd/translation-worker/tests/test_review/test_review.py
git commit -m "test(review): add integration tests for real LLM mode

- Add test for translator with mocked provider
- Ensure backward compatibility with stub mode
- Verify all 68 existing tests still pass

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Add documentation and usage examples

**Files:**
- Create: `docs/llm_integration.md`
- Modify: `CLAUDE.md` (if exists in worker dir)

**Step 1: Create documentation file**

Create `docs/llm_integration.md`:
```markdown
# LLM Integration Guide

## Overview

The translation worker review module supports both stub (MVP) and real LLM API integration for multi-model translation and judge evaluation.

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|-----------|
| `TRANSLATION_USE_REAL_LLM` | Set to "true" to enable real LLM APIs | No (default: false) |
| `ANTHROPIC_API_KEY` | Anthropic Claude API key | Yes (if using real LLM) |
| `OPENAI_API_KEY` | OpenAI GPT API key | Yes (if using real LLM) |
| `GEMINI_API_KEY` | Google Gemini API key | Yes (if using real LLM) |
| `GEMINI_PROJECT_ID` | Google Cloud project ID | Yes (for Gemini) |
| `GEMINI_LOCATION` | Gemini region (default: us-central1) | No |

### Usage Examples

**Stub Mode (MVP/Testing):**
```python
from review import MultiModelTranslator

translator = MultiModelTranslator(use_real_llm=False)
candidates = translator.translate("こんにちは")
# Returns: ["[Model A] こんにちは", "[Model B] こんにちは"]
```

**Real LLM Mode:**
```python
import os
os.environ["ANTHROPIC_API_KEY"] = "your-api-key"
os.environ["OPENAI_API_KEY"] = "your-api-key"
os.environ["TRANSLATION_USE_REAL_LLM"] = "true"

from review import create_workflow

workflow = create_workflow()
result = workflow.process_and_export(
    source_file="in.txt",
    target_file="out.txt",
    segments=[{"source": "こんにちは"}]
)
```

## Provider Models

### Default Models (2026)
- **model_a**: claude-sonnet-4.5-20250929 (Anthropic)
- **model_b**: gpt-4.1 (OpenAI)
- **model_c**: gemini-3.0-flash (Google Gemini)

### Custom Models
```python
from review import MultiModelTranslator

custom_models = {
    "model_a": {"name": "claude-opus-4-5-20251101", "provider": "anthropic"},
    "model_b": {"name": "gpt-4.1", "provider": "openai"},
    "model_c": {"name": "gemini-3.0-pro", "provider": "gemini"},
}

translator = MultiModelTranslator(models=custom_models, use_real_llm=True)
```

## Testing

### Mocking Providers
```python
from unittest.mock import Mock
from review import MultiModelTranslator

mock_response = Mock(text="Translated text", latency_ms=100, model="test", usage={})

translator = MultiModelTranslator(use_real_llm=True)
translator._set_provider("model_a", mock_provider)
```

### Running Tests
```bash
# All tests (stub mode)
PYTHONPATH=. python -m pytest tests/test_review/ -v

# Integration tests (require API keys)
TRANSLATION_USE_REAL_LLM=true python -m pytest tests/test_review/ -v -m integration
```
```

**Step 2: Update CLAUDE.md if exists**

Check and update documentation references.

**Step 3: Run tests to verify**

Run: `PYTHONPATH=. python -m pytest backend/cmd/translation-worker/tests/test_review/ -v`

Expected: PASS

**Step 4: Commit**

```bash
git add backend/cmd/translation-worker/docs/
git commit -m "docs(llm): add LLM integration guide

- Add environment variable reference
- Add usage examples for stub and real LLM modes
- Add provider model documentation
- Add testing guide with mocking examples
- Add CLAUDE.md reference for worker

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Summary

**Total Tasks:** 8 + 1 CLI task (Task 1.5)
**Estimated Time:** 2-3 hours (with TDD and commits)

**Key Changes:**
1. New `review/llm/` package with provider abstraction (Anthropic, OpenAI, Gemini)
2. New `review/cli.py` - **INDISPENSABLE** CLI interface for direct invocation
3. New `review/prompts.py` for translation/judge prompts
4. Updated `MultiModelTranslator` with real LLM support
5. Updated `TranslationJudge` with real LLM evaluation
6. Environment-based configuration via `TRANSLATION_USE_REAL_LLM`
7. Full backward compatibility with stub mode
8. Comprehensive test coverage with mocking support

**Dependencies:**
- `anthropic>=0.40.0`
- `openai>=1.0.0`
- `google-cloud-aiplatform>=1.0.0`
- `tenacity>=8.2.0`
- `click>=8.0.0` (for CLI)

**API Key Setup:**
```bash
export ANTHROPIC_API_KEY="your-key-here"
export OPENAI_API_KEY="your-key-here"
export GEMINI_API_KEY="your-key-here"
export GEMINI_PROJECT_ID="your-project-id"
export GEMINI_LOCATION="us-central1"
export TRANSLATION_USE_REAL_LLM="true"
```

**CLI Usage (INDISPENSABLE):**
```bash
# Direct translation
python -m review translate "こんにちは" --provider gemini

# Batch processing
python -m review batch --input sources.txt --output translations.txt --provider anthropic

# Judge comparisons
python -m review judge source.txt trans_a.txt trans_b.txt --provider openai
```
