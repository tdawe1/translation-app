import asyncio
import json
import logging
from dataclasses import dataclass
from typing import List, Optional, Dict, Any, Callable

from .pool import ProviderPool, PoolConfig
from .base import ProviderResponse

logger = logging.getLogger(__name__)


@dataclass
class TranslationRequest:
    text: str
    source_lang: str = "ja"
    target_lang: str = "en"
    context: Optional[str] = None
    glossary: Optional[Dict[str, str]] = None


@dataclass
class TranslationResponse:
    original: str
    translated: str
    model: str
    latency_ms: int
    tokens_used: int


class BatchTranslator:
    def __init__(
        self,
        pool: ProviderPool,
        system_prompt: Optional[str] = None,
        max_tokens: int = 2048,
        temperature: float = 0.1,
    ):
        self._pool = pool
        self._max_tokens = max_tokens
        self._temperature = temperature
        self._system_prompt = system_prompt or self._default_system_prompt()

    def _default_system_prompt(self) -> str:
        return (
            "You are a professional Japanese to English translator. "
            "Translate the following text naturally and accurately. "
            "Preserve formatting, line breaks, and any special markers. "
            "Do not add explanations or commentary. "
            "Return only the translation."
        )

    def _build_prompt(
        self,
        text: str,
        source_lang: str,
        target_lang: str,
        context: Optional[str] = None,
        glossary: Optional[Dict[str, str]] = None,
    ) -> str:
        parts = [self._system_prompt, ""]

        if glossary:
            parts.append("Glossary (use these exact translations):")
            for src, tgt in glossary.items():
                parts.append(f"  {src} → {tgt}")
            parts.append("")

        if context:
            parts.append(f"Context: {context}")
            parts.append("")

        parts.append(f"Translate from {source_lang} to {target_lang}:")
        parts.append(text)

        return "\n".join(parts)

    async def translate_one(self, request: TranslationRequest) -> TranslationResponse:
        prompt = self._build_prompt(
            text=request.text,
            source_lang=request.source_lang,
            target_lang=request.target_lang,
            context=request.context,
            glossary=request.glossary,
        )

        response = await self._pool.generate(
            prompt=prompt, max_tokens=self._max_tokens, temperature=self._temperature
        )

        return TranslationResponse(
            original=request.text,
            translated=response.text.strip(),
            model=response.model,
            latency_ms=response.latency_ms,
            tokens_used=response.usage.get("total_tokens", 0),
        )

    async def translate_batch(
        self,
        requests: List[TranslationRequest],
        on_progress: Optional[Callable[[int, int], None]] = None,
        max_concurrent: Optional[int] = None,
    ) -> List[TranslationResponse]:
        if not requests:
            return []

        prompts = [
            self._build_prompt(
                text=req.text,
                source_lang=req.source_lang,
                target_lang=req.target_lang,
                context=req.context,
                glossary=req.glossary,
            )
            for req in requests
        ]

        responses = await self._pool.generate_batch(
            prompts=prompts,
            max_tokens=self._max_tokens,
            temperature=self._temperature,
            max_concurrent=max_concurrent,
            on_progress=on_progress,
        )

        return [
            TranslationResponse(
                original=req.text,
                translated=resp.text.strip(),
                model=resp.model,
                latency_ms=resp.latency_ms,
                tokens_used=resp.usage.get("total_tokens", 0),
            )
            for req, resp in zip(requests, responses)
        ]

    async def translate_texts(
        self,
        texts: List[str],
        source_lang: str = "ja",
        target_lang: str = "en",
        glossary: Optional[Dict[str, str]] = None,
        on_progress: Optional[Callable[[int, int], None]] = None,
    ) -> List[str]:
        requests = [
            TranslationRequest(
                text=text,
                source_lang=source_lang,
                target_lang=target_lang,
                glossary=glossary,
            )
            for text in texts
        ]

        responses = await self.translate_batch(requests, on_progress)
        return [r.translated for r in responses]


class ChunkedBatchTranslator(BatchTranslator):
    def __init__(
        self,
        pool: ProviderPool,
        chunk_size: int = 10,
        system_prompt: Optional[str] = None,
        max_tokens: int = 4096,
        temperature: float = 0.1,
    ):
        super().__init__(pool, system_prompt, max_tokens, temperature)
        self._chunk_size = chunk_size
        self._system_prompt = system_prompt or self._chunked_system_prompt()

    def _chunked_system_prompt(self) -> str:
        return (
            "You are a professional Japanese to English translator. "
            "You will receive a JSON array of strings to translate. "
            "Return a JSON array of translated strings in the exact same order. "
            "Preserve formatting and special markers. "
            "Return ONLY the JSON array, no other text."
        )

    def _build_chunked_prompt(
        self,
        texts: List[str],
        source_lang: str,
        target_lang: str,
        glossary: Optional[Dict[str, str]] = None,
    ) -> str:
        parts = [self._system_prompt, ""]

        if glossary:
            parts.append("Glossary (use these exact translations):")
            for src, tgt in glossary.items():
                parts.append(f"  {src} → {tgt}")
            parts.append("")

        parts.append(f"Translate from {source_lang} to {target_lang}:")
        parts.append(json.dumps(texts, ensure_ascii=False))

        return "\n".join(parts)

    def _parse_json_array(self, response_text: str, expected_count: int) -> List[str]:
        text = response_text.strip()
        if text.startswith("```"):
            lines = text.split("\n")
            text = "\n".join(lines[1:-1] if lines[-1].startswith("```") else lines[1:])

        try:
            result = json.loads(text)
            if isinstance(result, list) and len(result) == expected_count:
                return [str(item) for item in result]
        except json.JSONDecodeError:
            pass

        logger.warning(f"Failed to parse JSON array, expected {expected_count} items")
        return []

    async def translate_texts_chunked(
        self,
        texts: List[str],
        source_lang: str = "ja",
        target_lang: str = "en",
        glossary: Optional[Dict[str, str]] = None,
        on_progress: Optional[Callable[[int, int], None]] = None,
    ) -> List[str]:
        if not texts:
            return []

        chunks = [
            texts[i : i + self._chunk_size]
            for i in range(0, len(texts), self._chunk_size)
        ]

        prompts = [
            self._build_chunked_prompt(chunk, source_lang, target_lang, glossary)
            for chunk in chunks
        ]

        completed_chunks = 0

        def chunk_progress(done: int, total: int):
            nonlocal completed_chunks
            completed_chunks = done
            if on_progress:
                items_done = min(done * self._chunk_size, len(texts))
                on_progress(items_done, len(texts))

        responses = await self._pool.generate_batch(
            prompts=prompts,
            max_tokens=self._max_tokens,
            temperature=self._temperature,
            on_progress=chunk_progress,
        )

        results = []
        for chunk, response in zip(chunks, responses):
            parsed = self._parse_json_array(response.text, len(chunk))
            if parsed:
                results.extend(parsed)
            else:
                results.extend(chunk)

        return results
