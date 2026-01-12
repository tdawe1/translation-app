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

from tenacity import (
    retry,
    stop_after_attempt,
    wait_exponential,
    retry_if_exception_type,
)

from .base import BaseProvider, ProviderConfig, ProviderResponse

logger = logging.getLogger(__name__)

# Try importing optional dependencies
try:
    import anthropic

    ANTHROPIC_AVAILABLE = True
except ImportError:
    ANTHROPIC_AVAILABLE = False

try:
    import openai

    OPENAI_AVAILABLE = True
except ImportError:
    OPENAI_AVAILABLE = False

try:
    import requests

    REQUESTS_AVAILABLE = True
except ImportError:
    REQUESTS_AVAILABLE = False


class AnthropicProvider(BaseProvider):
    """Anthropic Claude API provider.

    2026 Models:
    - claude-opus-4-5-20251101 (Opus 4.5)
    - claude-sonnet-4-5-20250929 (Sonnet 4.5)
    - claude-haiku-4-5-20250319 (Haiku 4.5)

    Supports custom base_url for local CLI tools like Claude Code.
    """

    DEFAULT_MODEL = "claude-sonnet-4-5-20250929"

    def __init__(self, api_key: str, model: str = None, base_url: str = None):
        model = model or self.DEFAULT_MODEL
        config = ProviderConfig(api_key=api_key, model=model, base_url=base_url)
        super().__init__(config)
        self._client = None

    def _get_client(self):
        """Lazy client initialization."""
        if not ANTHROPIC_AVAILABLE:
            raise ImportError(
                "anthropic package is required. Install: pip install anthropic"
            )
        if self._client is None:
            import anthropic

            kwargs = {"api_key": self.config.api_key}
            if self.config.base_url:
                # Support for local endpoints like Claude Code CLI
                kwargs["base_url"] = self.config.base_url
            self._client = anthropic.Anthropic(**kwargs)
        return self._client

    def is_available(self) -> bool:
        """Check if API key is set."""
        if not self.config.api_key or self.config.api_key == "":
            raise ValueError("API key is required for AnthropicProvider")
        return True

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=60),
        retry=retry_if_exception_type((Exception,)),
    )
    def generate(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion using Anthropic Messages API (2026)."""
        start = time.time()
        client = self._get_client()

        max_tokens = max_tokens or self.config.max_tokens

        response = client.messages.create(
            model=self.config.model,
            max_tokens=max_tokens,
            temperature=temperature,
            messages=[{"role": "user", "content": prompt}],
        )

        latency = int((time.time() - start) * 1000)

        return ProviderResponse(
            text=response.content[0].text,
            model=response.model,
            usage={
                "prompt_tokens": response.usage.input_tokens,
                "completion_tokens": response.usage.output_tokens,
                "total_tokens": response.usage.input_tokens
                + response.usage.output_tokens,
            },
            latency_ms=latency,
        )

    async def generate_async(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        """Async version using asyncio.to_thread."""
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            None, lambda: self.generate(prompt, max_tokens, temperature)
        )


class OpenAIProvider(BaseProvider):
    """OpenAI GPT API provider.

    2026 Models:
    - gpt-5 (GPT-5 base)
    - gpt-5-turbo (GPT-5 Turbo - faster, cheaper)
    - gpt-5.1 (GPT-5.1 - improved reasoning)
    - gpt-5.2 (GPT-5.2 - latest, January 2026)
    - gpt-4.1-2025-04-14 (GPT-4.1)
    - gpt-4.1-mini-2025-04-14 (GPT-4.1 Mini)
    - o3-mini (reasoning model)

    IMPORTANT: max_tokens is DEPRECATED - use max_completion_tokens
    """

    DEFAULT_MODEL = "gpt-5.2"

    def __init__(self, api_key: str, model: str = None):
        model = model or self.DEFAULT_MODEL
        config = ProviderConfig(api_key=api_key, model=model)
        super().__init__(config)
        self._client = None

    def _get_client(self):
        """Lazy client initialization."""
        if not OPENAI_AVAILABLE:
            raise ImportError("openai package is required. Install: pip install openai")
        if self._client is None:
            import openai

            self._client = openai.OpenAI(api_key=self.config.api_key)
        return self._client

    def is_available(self) -> bool:
        """Check if API key is set."""
        if not self.config.api_key or self.config.api_key == "":
            raise ValueError("API key is required for OpenAIProvider")
        return True

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=60),
        retry=retry_if_exception_type((Exception,)),
    )
    def generate(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion using OpenAI Chat Completions API (2026)."""
        start = time.time()
        client = self._get_client()

        max_tokens = max_tokens or self.config.max_tokens

        response = client.chat.completions.create(
            model=self.config.model,
            max_completion_tokens=max_tokens,  # NOT max_tokens (deprecated 2026)
            temperature=temperature,
            messages=[{"role": "user", "content": prompt}],
        )

        latency = int((time.time() - start) * 1000)

        return ProviderResponse(
            text=response.choices[0].message.content,
            model=response.model,
            usage={
                "prompt_tokens": response.usage.prompt_tokens,
                "completion_tokens": response.usage.completion_tokens,
                "total_tokens": response.usage.total_tokens,
            },
            latency_ms=latency,
        )

    async def generate_async(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        """Async version using asyncio.to_thread."""
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            None, lambda: self.generate(prompt, max_tokens, temperature)
        )


class GeminiProvider(BaseProvider):
    """Google Gemini API provider via Vertex AI.

    2026 Models:
    - gemini-3.0-flash (Gemini 3.0 Flash - fast, cheap, recommended)
    - gemini-3.0-pro (Gemini 3.0 Pro)
    - gemini-3.0-ultra (Gemini 3.0 Ultra - highest capability)

    Environment Variables:
    - GEMINI_API_KEY: API key for authentication
    - GEMINI_PROJECT_ID: Google Cloud project ID
    - GEMINI_LOCATION: Region (default: us-central1)
    """

    DEFAULT_MODEL = "gemini-3.0-flash"
    DEFAULT_LOCATION = "us-central1"

    def __init__(
        self,
        api_key: str,
        model: str = None,
        project_id: str = None,
        location: str = None,
    ):
        model = model or self.DEFAULT_MODEL
        config = ProviderConfig(api_key=api_key, model=model)
        super().__init__(config)
        self.project_id = project_id or os.environ.get("GEMINI_PROJECT_ID", "")
        self.location = location or os.environ.get(
            "GEMINI_LOCATION", self.DEFAULT_LOCATION
        )
        self._client = None

    def is_available(self) -> bool:
        """Check if API key is set."""
        if not REQUESTS_AVAILABLE:
            raise ImportError(
                "requests package is required. Install: pip install requests"
            )
        if not self.config.api_key or self.config.api_key == "":
            raise ValueError("API key is required for GeminiProvider")
        return True

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
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion using Vertex AI Gemini API (2026)."""
        import requests

        start = time.time()

        max_tokens = max_tokens or self.config.max_tokens

        # Gemini request format (2026)
        request_body = {
            "contents": [{"role": "user", "parts": [{"text": prompt}]}],
            "generationConfig": {
                "temperature": temperature,
                "maxOutputTokens": max_tokens,  # Gemini uses maxOutputTokens
            },
        }

        endpoint = self._get_endpoint()
        headers = {
            "Authorization": f"Bearer {self.config.api_key}",
            "Content-Type": "application/json",
        }

        response = requests.post(
            f"{endpoint}:generateContent",
            json=request_body,
            headers=headers,
            timeout=self.config.timeout,
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
                "total_tokens": usage_metadata.get("totalTokenCount", 0),
            },
            latency_ms=latency,
        )

    async def generate_async(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        """Async version using asyncio.to_thread."""
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            None, lambda: self.generate(prompt, max_tokens, temperature)
        )


def get_provider(
    provider_name: str, api_key: str, model: Optional[str] = None, **kwargs
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
        raise ValueError(
            f"Unknown provider: {provider_name}. Use: {list(models.keys())}"
        )

    default_model, provider_class = models[provider_name]
    model = model or default_model

    if provider_name == "gemini":
        return provider_class(api_key=api_key, model=model, **kwargs)

    return provider_class(api_key=api_key, model=model)
