"""Ollama local model provider for translation.

Supports local inference with models like llama3.1, qwen2.5, etc.
"""

import asyncio
import logging
import time
from dataclasses import dataclass
from typing import List, Optional

from tenacity import (
    retry,
    stop_after_attempt,
    wait_exponential,
    retry_if_exception_type,
)

from .base import BaseProvider, ProviderConfig, ProviderResponse

logger = logging.getLogger(__name__)

try:
    import httpx

    HTTPX_AVAILABLE = True
except ImportError:
    HTTPX_AVAILABLE = False


@dataclass
class OllamaConfig:
    base_url: str = "http://localhost:11434"
    model: str = "llama3.1:8b"
    timeout: int = 300


class OllamaProvider(BaseProvider):
    """Ollama local model provider.

    Supports models: llama3.1:8b, llama3.1:70b, qwen2.5:72b, etc.
    """

    DEFAULT_MODEL = "llama3.1:8b"
    DEFAULT_BASE_URL = "http://localhost:11434"

    def __init__(self, base_url: str = None, model: str = None, timeout: int = 300):
        base_url = base_url or self.DEFAULT_BASE_URL
        model = model or self.DEFAULT_MODEL

        config = ProviderConfig(
            api_key="", base_url=base_url, model=model, timeout=timeout
        )
        super().__init__(config)
        self._ollama_config = OllamaConfig(
            base_url=base_url, model=model, timeout=timeout
        )

    @property
    def name(self) -> str:
        return "ollama"

    @property
    def base_url(self) -> str:
        return self._ollama_config.base_url

    @property
    def model(self) -> str:
        return self._ollama_config.model

    async def is_available_async(self) -> bool:
        if not HTTPX_AVAILABLE:
            return False
        try:
            async with httpx.AsyncClient(timeout=5) as client:
                response = await client.get(f"{self._ollama_config.base_url}/api/tags")
            return response.status_code == 200
        except httpx.RequestError:
            return False

    def is_available(self) -> bool:
        return asyncio.run(self.is_available_async())

    async def list_models_async(self) -> List[str]:
        if not HTTPX_AVAILABLE:
            return []
        try:
            async with httpx.AsyncClient(timeout=10) as client:
                response = await client.get(f"{self._ollama_config.base_url}/api/tags")
            response.raise_for_status()
            data = response.json()
            return [m["name"] for m in data.get("models", [])]
        except httpx.HTTPStatusError:
            return []

    def list_models(self) -> List[str]:
        return asyncio.run(self.list_models_async())

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=60),
        retry=retry_if_exception_type((Exception,)),
        reraise=True,
    )
    async def generate_async(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.3
    ) -> ProviderResponse:
        if not HTTPX_AVAILABLE:
            raise ImportError("httpx package is required")

        start = time.time()

        payload = {
            "model": self._ollama_config.model,
            "prompt": prompt,
            "stream": False,
            "options": {
                "temperature": temperature,
            },
        }

        if max_tokens:
            payload["options"]["num_predict"] = max_tokens

        try:
            async with httpx.AsyncClient(timeout=self._ollama_config.timeout) as client:
                response = await client.post(
                    f"{self._ollama_config.base_url}/api/generate",
                    json=payload,
                )
            response.raise_for_status()
            data = response.json()

            latency = int((time.time() - start) * 1000)
            text = self._parse_response(data)

            return ProviderResponse(
                text=text,
                model=self._ollama_config.model,
                usage={
                    "prompt_tokens": data.get("prompt_eval_count", 0),
                    "completion_tokens": data.get("eval_count", 0),
                    "total_tokens": data.get("prompt_eval_count", 0)
                    + data.get("eval_count", 0),
                },
                latency_ms=latency,
                raw_response=data,
            )

        except httpx.RequestError as e:
            logger.error(f"Ollama request failed: {e}")
            raise RuntimeError(f"Ollama request failed: {e}") from e

    def generate(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.3
    ) -> ProviderResponse:
        return asyncio.run(self.generate_async(prompt, max_tokens, temperature))

    def _parse_response(self, data: dict) -> str:
        if isinstance(data, dict):
            return data.get("response", "").strip()

        if isinstance(data, str):
            import json

            lines = data.strip().split("\n")
            text_parts = []
            for line in lines:
                try:
                    obj = json.loads(line)
                    text_parts.append(obj.get("response", ""))
                except json.JSONDecodeError:
                    continue
            return "".join(text_parts).strip()

        return ""

    async def translate_async(
        self, text: str, source_lang: str = "ja", target_lang: str = "en"
    ) -> "TranslationResult":
        prompt = self._build_translation_prompt(text, source_lang, target_lang)

        if not await self.is_available_async():
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._ollama_config.model,
                error="Ollama server not available",
            )

        try:
            response = await self.generate_async(prompt)
            return TranslationResult(
                success=True,
                translated_text=response.text,
                confidence=0.85,
                provider=self.name,
                model=self._ollama_config.model,
            )
        except Exception as e:
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._ollama_config.model,
                error=str(e),
            )

    def translate(
        self, text: str, source_lang: str = "ja", target_lang: str = "en"
    ) -> "TranslationResult":
        return asyncio.run(self.translate_async(text, source_lang, target_lang))

    def _build_translation_prompt(
        self, text: str, source_lang: str, target_lang: str
    ) -> str:
        return f"""You are a professional translator. Translate the following text from {source_lang} to {target_lang}.

Output ONLY the translation, nothing else.

Text: {text}

Translation:"""


@dataclass
class TranslationResult:
    success: bool
    translated_text: str
    confidence: float
    provider: str
    model: str
    error: Optional[str] = None


def get_ollama_provider(
    base_url: str = None, model: str = None, **kwargs
) -> OllamaProvider:
    return OllamaProvider(base_url=base_url, model=model, **kwargs)
