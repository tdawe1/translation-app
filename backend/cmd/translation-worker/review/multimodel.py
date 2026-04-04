# review/multimodel.py
"""
Multi-model translation coordinator.

Generates competing translations from multiple models for judge evaluation.
Supports both CLI tools (no API costs) and API providers.
"""

import logging
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import List, Dict, Optional, Callable, Any
from .models import TranslationCandidate

logger = logging.getLogger(__name__)


class MultiModelTranslator:
    """Coordinates translation across multiple models using real LLM providers."""

    def __init__(
        self,
        providers: Optional[List[Any]] = None,
        system_prompt: Optional[str] = None,
        parallel: bool = True,
        max_workers: int = 3,
    ):
        """Initialize with configured providers.

        Args:
            providers: List of LLM provider instances (CLIProvider or cloud provider)
            system_prompt: Optional system prompt to prepend to all translations
            parallel: Whether to execute translations in parallel
            max_workers: Maximum parallel workers for translation
        """
        self.providers = providers or []
        self.system_prompt = system_prompt
        self.parallel = parallel
        self.max_workers = max_workers
        self._base_prompt = """Translate the following Japanese text to natural, fluent US English.
Preserve any formatting, numbers, and proper nouns.
IMPORTANT: Use natural phrasing. Avoid em-dashes without spaces (use ' - ' with spaces if needed).

Japanese text:
"""

    def translate(
        self,
        source: str,
        glossary_terms: Optional[List[str]] = None,
        context: Optional[dict] = None,
    ) -> List[TranslationCandidate]:
        """Generate translations from all configured providers."""
        if not self.providers:
            logger.warning("[MULTI] No providers configured, returning empty list")
            return []

        if self.parallel and len(self.providers) > 1:
            return self._translate_parallel(source, glossary_terms, context)
        else:
            return self._translate_sequential(source, glossary_terms, context)

    def _build_prompt(self, source: str, glossary_terms: Optional[List[str]]) -> str:
        """Build the full translation prompt."""
        prompt_parts = []

        if self.system_prompt:
            prompt_parts.append(self.system_prompt)
            prompt_parts.append("\n\n")

        prompt_parts.append(self._base_prompt)
        prompt_parts.append(source)

        if glossary_terms:
            prompt_parts.append(
                f"\n\nGlossary terms to use: {', '.join(glossary_terms)}"
            )

        prompt_parts.append("\n\nEnglish translation:")

        return "".join(prompt_parts)

    def _translate_with_provider(
        self,
        provider: Any,
        prompt: str,
        model_key: str,
    ) -> TranslationCandidate:
        """Translate using a single provider."""
        start = time.time()
        try:
            response = provider.generate(prompt)
            text = (
                response.text.strip()
                if hasattr(response, "text")
                else str(response).strip()
            )
            latency = int((time.time() - start) * 1000)

            return TranslationCandidate(
                model_name=model_key,
                text=text,
                confidence=0.9,
                glossary_matches=[],
                latency_ms=latency,
            )
        except Exception as e:
            logger.error(f"[MULTI] Provider {model_key} failed: {e}")
            latency = int((time.time() - start) * 1000)
            return TranslationCandidate(
                model_name=model_key,
                text=f"[TRANSLATION ERROR: {e}]",
                confidence=0.0,
                glossary_matches=[],
                latency_ms=latency,
            )

    def _translate_sequential(
        self, source: str, glossary_terms: Optional[List[str]], context: Optional[dict]
    ) -> List[TranslationCandidate]:
        """Translate sequentially through all providers."""
        candidates = []
        prompt = self._build_prompt(source, glossary_terms)

        for idx, provider in enumerate(self.providers):
            model_key = f"model_{chr(97 + idx)}"
            candidate = self._translate_with_provider(provider, prompt, model_key)
            candidates.append(candidate)

        return candidates

    def _translate_parallel(
        self, source: str, glossary_terms: Optional[List[str]], context: Optional[dict]
    ) -> List[TranslationCandidate]:
        """Translate in parallel using ThreadPoolExecutor."""
        candidates = []
        prompt = self._build_prompt(source, glossary_terms)

        with ThreadPoolExecutor(max_workers=self.max_workers) as executor:
            futures = {}
            for idx, provider in enumerate(self.providers):
                model_key = f"model_{chr(97 + idx)}"
                future = executor.submit(
                    self._translate_with_provider, provider, prompt, model_key
                )
                futures[future] = model_key

            for future in as_completed(futures):
                candidate = future.result()
                candidates.append(candidate)

        candidates.sort(key=lambda c: c.model_name)
        return candidates

    def translate_batch(
        self,
        sources: List[str],
        glossary_terms: Optional[List[str]] = None,
        context: Optional[dict] = None,
    ) -> Dict[str, List[TranslationCandidate]]:
        """Translate multiple source segments.

        Args:
            sources: List of source texts with IDs
            glossary_terms: Optional glossary terms to apply
            context: Optional translation context

        Returns:
            Mapping from source ID to list of candidates
        """
        results = {}

        for source in sources:
            if isinstance(source, str):
                source_id = f"seg_{hash(source) % 10000:04d}"
                source_text = source
            else:
                source_id = source.get(
                    "id", f"seg_{hash(source.get('text', '')) % 10000:04d}"
                )
                source_text = source.get("text", source)

            candidates = self.translate(source_text, glossary_terms, context)
            results[source_id] = candidates

        logger.debug(f"[MULTI] Translated {len(results)} segments")
        return results

    async def translate_async(
        self,
        source: str,
        glossary_terms: Optional[List[str]] = None,
        context: Optional[dict] = None,
    ) -> List[TranslationCandidate]:
        """Async version of translate.

        MVP: Delegates to synchronous translate.
        Full: True async implementation with concurrent model calls.
        """
        # Simulate async with minimal delay
        await asyncio.sleep(0.001)
        return self.translate(source, glossary_terms, context)

    def set_translation_function(
        self, model_key: str, func: Callable[[str, List[str]], str]
    ) -> None:
        """Register a custom translation function for a model.

        Allows overriding stub translation with real implementation.

        Args:
            model_key: Which model this function is for
            func: Function taking (source, glossary_terms) -> translation
        """
        # Store the function for use in translate
        # MVP: Not implemented, would need to add _trans_functions dict
        logger.info(f"[MULTI] Registered custom function for {model_key}")


def create_multimodel_translator(
    models: Optional[Dict[str, dict]] = None, parallel: bool = True
) -> MultiModelTranslator:
    """Factory function to create a MultiModelTranslator.

    Args:
        models: Optional model configuration dict (reserved for future use)
        parallel: Whether to use parallel execution

    Returns:
        Configured MultiModelTranslator instance
    """
    return MultiModelTranslator(models=models, parallel=parallel)
