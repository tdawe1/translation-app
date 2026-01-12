# review/judge.py
"""
Judge model for evaluating competing translations.

MVP: Stub implementation with random selection.
Full: Sophisticated prompting with contextual evaluation.
"""

import logging
import random
from typing import List, Optional
from .models import TranslationCandidate, JudgeResult

logger = logging.getLogger(__name__)


class TranslationJudge:
    """Evaluates competing translations and selects the best option.

    MVP behavior: Returns random selection with placeholder reasoning.
    Full implementation: Uses LLM with detailed prompt for evaluation.
    """

    def __init__(
        self,
        model: str = "claude-4.5-sonnet",
        timeout: int = 30,
        fallback_on_timeout: bool = True,
        enabled: bool = True
    ):
        """Initialize the translation judge.

        Args:
            model: Model identifier for judge decisions
            timeout: Timeout for judge decisions in seconds
            fallback_on_timeout: If True, use random fallback on timeout
            enabled: If False, always return model_a as winner
        """
        self.model = model
        self.timeout = timeout
        self.fallback_on_timeout = fallback_on_timeout
        self.enabled = enabled

    def judge(
        self,
        segment_id: str,
        source: str,
        candidates: List[TranslationCandidate],
        context: Optional[dict] = None
    ) -> JudgeResult:
        """Evaluate candidates and select the winner.

        Args:
            segment_id: Unique identifier for this segment
            source: Original source text
            candidates: List of translation candidates to evaluate
            context: Additional context (document type, position, etc.)

        Returns:
            JudgeResult with winner, confidence, and reasoning
        """
        if not self.enabled:
            return JudgeResult(
                segment_id=segment_id,
                winner="model_a",
                confidence=1.0,
                reasoning="Judge disabled, defaulting to model_a"
            )

        if len(candidates) < 2:
            # Only one candidate, return it
            return JudgeResult(
                segment_id=segment_id,
                winner="model_a",
                confidence=candidates[0].confidence if candidates else 1.0,
                reasoning="Only one candidate provided"
            )

        # MVP: Random selection with stubbed reasoning
        return self._stub_judge(segment_id, source, candidates, context)

    def _stub_judge(
        self,
        segment_id: str,
        source: str,
        candidates: List[TranslationCandidate],
        context: Optional[dict]
    ) -> JudgeResult:
        """Stub implementation using random selection.

        Full implementation would use LLM with detailed prompt.
        """
        # Random selection for MVP
        winner_idx = random.randint(0, len(candidates) - 1)
        winner = f"model_{chr(97 + winner_idx)}"  # model_a, model_b, etc. (lowercase)
        chosen = candidates[winner_idx]

        # Generate placeholder reasoning based on source length
        if len(source) < 20:
            reasoning = "Short segment: both translations are adequate"
        elif len(source) < 100:
            reasoning = "Medium segment: selected translation captures meaning accurately"
        else:
            reasoning = "Long segment: selected translation maintains flow and clarity"

        # Random confidence between 0.6 and 1.0
        confidence = round(random.uniform(0.6, 1.0), 2)

        # 10% chance of flagging a concern for MVP
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

    def judge_batch(
        self,
        segments: List[dict],
        candidates_map: dict[str, List[TranslationCandidate]]
    ) -> List[JudgeResult]:
        """Judge multiple segments in batch.

        Args:
            segments: List of segment dicts with 'id', 'source', 'context'
            candidates_map: Mapping from segment_id to list of candidates

        Returns:
            List of JudgeResult in same order as segments
        """
        results = []
        for seg in segments:
            candidates = candidates_map.get(seg["id"], [])
            result = self.judge(
                segment_id=seg["id"],
                source=seg["source"],
                candidates=candidates,
                context=seg.get("context")
            )
            results.append(result)

        logger.debug(f"[JUDGE] Batch judged {len(results)} segments")
        return results

    async def judge_async(
        self,
        segment_id: str,
        source: str,
        candidates: List[TranslationCandidate],
        context: Optional[dict] = None
    ) -> JudgeResult:
        """Async version of judge for concurrent processing.

        MVP: Delegates to synchronous judge.
        Full: Makes actual async LLM call.
        """
        return self.judge(segment_id, source, candidates, context)


def create_judge(
    model: str = "claude-4.5-sonnet",
    timeout: int = 30,
    enabled: bool = True
) -> TranslationJudge:
    """Factory function to create a TranslationJudge.

    Args:
        model: Model identifier for judge decisions
        timeout: Timeout for judge decisions in seconds
        enabled: Whether judge model is active

    Returns:
        Configured TranslationJudge instance
    """
    return TranslationJudge(
        model=model,
        timeout=timeout,
        enabled=enabled
    )
