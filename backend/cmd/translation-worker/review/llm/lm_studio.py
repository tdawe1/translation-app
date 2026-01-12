"""
LM Studio provider for local model inference.

LM Studio exposes an OpenAI-compatible API for locally loaded models.
This provider connects to that API for translation tasks.
"""

from typing import List
from dataclasses import dataclass
import requests
import asyncio
from concurrent.futures import ThreadPoolExecutor

from review.llm.base import BaseProvider
from review.llm.ollama import TranslationResult


@dataclass
class LMStudioConfig:
    """LM Studio provider configuration."""

    base_url: str = "http://localhost:1234/v1"
    model: str = "local-model"
    timeout: int = 300


class LMStudioProvider(BaseProvider):
    """LM Studio provider using OpenAI-compatible API."""

    def __init__(
        self, base_url: str = "http://localhost:1234/v1", model: str = "local-model"
    ):
        """
        Initialize LM Studio provider.

        Args:
            base_url: LM Studio API base URL (OpenAI-compatible)
            model: Model name (must match loaded model in LM Studio)
        """
        self._config = LMStudioConfig(base_url=base_url, model=model)
        self._executor = ThreadPoolExecutor(max_workers=3)

    @property
    def name(self) -> str:
        """Provider name."""
        return "lm_studio"

    @property
    def base_url(self) -> str:
        """Base URL for API requests."""
        return self._config.base_url

    @property
    def model(self) -> str:
        """Model name."""
        return self._config.model

    def is_available(self) -> bool:
        """Check if LM Studio server is running."""
        try:
            response = requests.get(f"{self._config.base_url}/models", timeout=5)
            return response.status_code == 200
        except requests.exceptions.RequestException:
            return False

    def list_models(self) -> List[str]:
        """List available models from LM Studio."""
        try:
            response = requests.get(f"{self._config.base_url}/models", timeout=10)
            response.raise_for_status()
            data = response.json()
            return [m["id"] for m in data.get("data", [])]
        except requests.exceptions.RequestException:
            return []

    def generate(self, prompt: str, **kwargs) -> str:
        """
        Generate text using LM Studio.

        Args:
            prompt: Input prompt
            **kwargs: Additional parameters (temperature, max_tokens, etc.)

        Returns:
            Generated text as string
        """
        if not self.is_available():
            raise RuntimeError("LM Studio server not available")

        headers = {"Content-Type": "application/json"}

        payload = {
            "model": self._config.model,
            "messages": [{"role": "user", "content": prompt}],
            **kwargs,
        }

        response = requests.post(
            f"{self._config.base_url}/chat/completions",
            json=payload,
            headers=headers,
            timeout=self._config.timeout,
        )
        response.raise_for_status()

        data = response.json()
        return data["choices"][0]["message"]["content"]

    async def generate_async(self, prompt: str, **kwargs) -> str:
        """
        Generate text asynchronously using LM Studio.

        Args:
            prompt: Input prompt
            **kwargs: Additional parameters

        Returns:
            Generated text as string
        """
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            self._executor, self.generate, prompt, **kwargs
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
                model=self._config.model,
                error="LM Studio server not available",
            )

        headers = {"Content-Type": "application/json"}

        system_prompt = self._build_system_prompt(source_lang, target_lang)
        user_prompt = (
            f"Translate this text from {source_lang} to {target_lang}:\n\n{text}"
        )

        payload = {
            "model": self._config.model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
            "temperature": 0.3,
            "max_tokens": 4096,
        }

        try:
            response = requests.post(
                f"{self._config.base_url}/chat/completions",
                json=payload,
                headers=headers,
                timeout=self._config.timeout,
            )
            response.raise_for_status()

            data = response.json()
            translated_text = data["choices"][0]["message"]["content"]

            return TranslationResult(
                success=True,
                translated_text=translated_text.strip(),
                confidence=0.85,  # Local models generally less consistent
                provider=self.name,
                model=self._config.model,
            )

        except (requests.exceptions.RequestException, KeyError, IndexError) as e:
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._config.model,
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
