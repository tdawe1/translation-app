"""
LM Studio provider for local model inference.

LM Studio exposes an OpenAI-compatible API for locally loaded models.
This provider connects to that API for translation tasks.
"""

from typing import List, Optional
from dataclasses import dataclass
import requests
import asyncio
import time
from concurrent.futures import ThreadPoolExecutor

from review.llm.base import BaseProvider, ProviderConfig, ProviderResponse
from review.llm.ollama import TranslationResult


@dataclass
class LMStudioConfig:
    """LM Studio provider configuration."""

    base_url: str = "http://localhost:1234/v1"
    model: str = "local-model"
    timeout: int = 300


class LMStudioProvider(BaseProvider):
    """LM Studio provider using OpenAI-compatible API."""

    DEFAULT_MODEL = "local-model"
    DEFAULT_BASE_URL = "http://localhost:1234/v1"

    def __init__(
        self, base_url: str = None, model: str = None
    ):
        """
        Initialize LM Studio provider.

        Args:
            base_url: LM Studio API base URL (OpenAI-compatible)
            model: Model name (must match loaded model in LM Studio)
        """
        base_url = base_url or self.DEFAULT_BASE_URL
        model = model or self.DEFAULT_MODEL

        config = ProviderConfig(
            api_key="",  # LM Studio doesn't use API keys
            base_url=base_url,
            model=model
        )
        super().__init__(config)

        self._lm_config = LMStudioConfig(base_url=base_url, model=model)
        self._executor = ThreadPoolExecutor(max_workers=3)

    @property
    def name(self) -> str:
        """Provider name."""
        return "lm_studio"

    @property
    def base_url(self) -> str:
        """Base URL for API requests."""
        return self._lm_config.base_url

    @property
    def model(self) -> str:
        """Model name."""
        return self._lm_config.model

    def is_available(self) -> bool:
        """Check if LM Studio server is running."""
        try:
            response = requests.get(f"{self._lm_config.base_url}/models", timeout=5)
            return response.status_code == 200
        except requests.exceptions.RequestException:
            return False

    def list_models(self) -> List[str]:
        """List available models from LM Studio."""
        try:
            response = requests.get(f"{self._lm_config.base_url}/models", timeout=10)
            response.raise_for_status()
            data = response.json()
            return [m["id"] for m in data.get("data", [])]
        except requests.exceptions.RequestException:
            return []

    def generate(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """
        Generate text using LM Studio.

        Args:
            prompt: Input prompt
            max_tokens: Maximum tokens to generate
            temperature: Sampling temperature

        Returns:
            ProviderResponse: Standardized response object with text, model, usage, latency
        """
        if not self.is_available():
            raise RuntimeError("LM Studio server not available")

        start = time.time()

        headers = {"Content-Type": "application/json"}

        payload = {
            "model": self._lm_config.model,
            "messages": [{"role": "user", "content": prompt}],
            "temperature": temperature,
        }
        if max_tokens:
            payload["max_tokens"] = max_tokens

        response = requests.post(
            f"{self._lm_config.base_url}/chat/completions",
            json=payload,
            headers=headers,
            timeout=self._lm_config.timeout,
        )
        response.raise_for_status()

        data = response.json()
        latency = int((time.time() - start) * 1000)

        # Extract text from LM Studio response
        text = data["choices"][0]["message"]["content"]

        # Return ProviderResponse to satisfy BaseProvider contract
        return ProviderResponse(
            text=text,
            model=self._lm_config.model,
            usage={
                "prompt_tokens": data.get("usage", {}).get("prompt_tokens", len(prompt)),
                "completion_tokens": data.get("usage", {}).get("completion_tokens", len(text)),
                "total_tokens": data.get("usage", {}).get("total_tokens", len(prompt) + len(text)),
            },
            latency_ms=latency,
            raw_response=data
        )

    async def generate_async(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """
        Generate text asynchronously using LM Studio.

        Args:
            prompt: Input prompt
            max_tokens: Maximum tokens to generate
            temperature: Sampling temperature

        Returns:
            ProviderResponse: Standardized response object
        """
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            self._executor, self.generate, prompt, max_tokens, temperature
        )

    def translate(
        self, text: str, source_lang: str = "ja", target_lang: str = "en"
    ) -> TranslationResult:
        """
        Translate text using LM Studio model.

        Args:
            text: Source text to translate
            source_lang: Source language code
            target_lang: Target language code

        Returns:
            TranslationResult with translated text or error
        """
        if not self.is_available():
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._lm_config.model,
                error="LM Studio server not available",
            )

        headers = {"Content-Type": "application/json"}

        system_prompt = self._build_system_prompt(source_lang, target_lang)
        user_prompt = (
            f"Translate this text from {source_lang} to {target_lang}:\n\n{text}"
        )

        payload = {
            "model": self._lm_config.model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
            "temperature": 0.3,
            "max_tokens": 4096,
        }

        try:
            response = requests.post(
                f"{self._lm_config.base_url}/chat/completions",
                json=payload,
                headers=headers,
                timeout=self._lm_config.timeout,
            )
            response.raise_for_status()

            data = response.json()
            translated_text = data["choices"][0]["message"]["content"]

            return TranslationResult(
                success=True,
                translated_text=translated_text.strip(),
                confidence=0.85,  # Local models generally less consistent
                provider=self.name,
                model=self._lm_config.model,
            )

        except (requests.exceptions.RequestException, KeyError, IndexError) as e:
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._lm_config.model,
                error=str(e),
            )

    def _build_system_prompt(self, source_lang: str, target_lang: str) -> str:
        """Build system prompt for LM Studio."""
        return f"""You are a professional translator specializing in {source_lang} to {target_lang} translation.

Rules:
1. Translate ONLY the provided text, no explanations
2. Preserve formatting and structure where possible
3. Output the translation directly, no preamble"""


def get_lm_studio_provider(
    base_url: str = "http://localhost:1234/v1", model: str = "local-model"
) -> LMStudioProvider:
    """
    Factory function to create LM Studio provider instance.

    Args:
        base_url: LM Studio API base URL (default: localhost:1234/v1)
        model: Model name to use (must match loaded model in LM Studio)

    Returns:
        Configured LMStudioProvider instance
    """
    return LMStudioProvider(base_url=base_url, model=model)
