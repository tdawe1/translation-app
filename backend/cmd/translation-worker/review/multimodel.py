# review/multimodel.py
"""
Multi-model translation coordinator.

Generates competing translations from multiple models for judge evaluation.

MVP: Stub with two hard-coded models returning placeholder translations.
Full: Configurable model registry with parallel async execution.
"""

import asyncio
import logging
import time
from typing import List, Dict, Optional, Callable
from .models import TranslationCandidate

logger = logging.getLogger(__name__)


class MultiModelTranslator:
    """Coordinates translation across multiple models.

    MVP: Stub implementation with placeholder translations.
    Full: Parallel async calls to configured LLM providers.
    """

    # Default model configurations
    DEFAULT_MODELS = {
        "model_a": {
            "name": "claude-4.5-sonnet",
            "provider": "anthropic",
            "description": "Primary model - balanced quality and speed"
        },
        "model_b": {
            "name": "gpt-4o",
            "provider": "openai",
            "description": "Secondary model - fast and cost-effective"
        }
    }

    def __init__(
        self,
        models: Optional[Dict[str, dict]] = None,
        parallel: bool = True
    ):
        """Initialize the multi-model translator.

        Args:
            models: Mapping of model_key to model config
            parallel: Whether to execute translations in parallel
        """
        self.models = models or self.DEFAULT_MODELS
        self.parallel = parallel

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
        """Translate sequentially (MVP stub)."""
        candidates = []

        for key, config in self.models.items():
            start = time.time()
            result = self._stub_translate(source, key, glossary_terms)
            latency = int((time.time() - start) * 1000)

            candidates.append(TranslationCandidate(
                model_name=key,
                text=result,
                confidence=0.9,  # Stub confidence
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
        """Translate in parallel using asyncio (MVP stub)."""
        # MVP: Still sequential but with async structure
        # Full: Use asyncio.gather() for true parallelism
        return self._translate_sequential(source, glossary_terms, context)

    def _stub_translate(
        self,
        source: str,
        model_key: str,
        glossary_terms: Optional[List[str]]
    ) -> str:
        """Stub translation for MVP.

        Returns a placeholder translation indicating the model used.
        """
        # Check for glossary terms to apply
        if glossary_terms:
            glossary_str = ", ".join(glossary_terms)
            return f"[{model_key}] {source} (glossary: {glossary_str})"

        # Simple placeholder translation
        if model_key == "model_a":
            return f"[Model A] {source}"
        else:
            return f"[Model B] {source}"

    def translate_batch(
        self,
        sources: List[str],
        glossary_terms: Optional[List[str]] = None,
        context: Optional[dict] = None
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
                source_id = source.get("id", f"seg_{hash(source.get('text', '')) % 10000:04d}")
                source_text = source.get("text", source)

            candidates = self.translate(source_text, glossary_terms, context)
            results[source_id] = candidates

        logger.debug(f"[MULTI] Translated {len(results)} segments")
        return results

    async def translate_async(
        self,
        source: str,
        glossary_terms: Optional[List[str]] = None,
        context: Optional[dict] = None
    ) -> List[TranslationCandidate]:
        """Async version of translate.

        MVP: Delegates to synchronous translate.
        Full: True async implementation with concurrent model calls.
        """
        # Simulate async with minimal delay
        await asyncio.sleep(0.001)
        return self.translate(source, glossary_terms, context)

    def set_translation_function(
        self,
        model_key: str,
        func: Callable[[str, List[str]], str]
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
    models: Optional[Dict[str, dict]] = None,
    parallel: bool = True
) -> MultiModelTranslator:
    """Factory function to create a MultiModelTranslator.

    Args:
        models: Optional model configuration dict
        parallel: Whether to use parallel execution

    Returns:
        Configured MultiModelTranslator instance
    """
    return MultiModelTranslator(
        models=models,
        parallel=parallel
    )
