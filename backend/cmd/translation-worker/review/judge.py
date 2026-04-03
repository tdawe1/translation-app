# review/judge.py
"""
Judge model for evaluating competing translations.
Uses LLM to compare translations and select the best one with reasoning.
"""

import json
import logging
import re
from typing import List, Optional, Any
from .models import TranslationCandidate, JudgeResult

logger = logging.getLogger(__name__)


JUDGE_PROMPT_TEMPLATE = """You are a professional translation quality evaluator for Japanese to English translations.

Compare the following translations of the same Japanese source text and determine which is better.

SOURCE (Japanese):
{source}

TRANSLATION A:
{translation_a}

TRANSLATION B:
{translation_b}

Evaluate based on:
1. Accuracy - Does it preserve the original meaning?
2. Naturalness - Does it read like native English?
3. Style - Is it appropriate for the context (formal/informal)?
4. Avoid AI patterns - No em-dashes without spaces, natural phrasing

Respond in JSON format only:
{{
    "winner": "A" or "B" or "TIE",
    "confidence": 0.0-1.0,
    "reasoning": "Brief explanation (1-2 sentences)",
    "concerns": ["list of issues if any"]
}}
"""


class TranslationJudge:
    """Evaluates competing translations using LLM and selects the best option."""

    def __init__(
        self,
        provider: Optional[Any] = None,
        timeout: int = 30,
        fallback_on_error: bool = True,
        enabled: bool = True,
    ):
        """Initialize with LLM provider for judging."""
        self.provider = provider
        self.timeout = timeout
        self.fallback_on_error = fallback_on_error
        self.enabled = enabled

    def judge(
        self,
        segment_id: str,
        source: str,
        candidates: List[TranslationCandidate],
        context: Optional[dict] = None,
    ) -> JudgeResult:
        """Evaluate candidates and select the winner using LLM."""
        if not self.enabled:
            return JudgeResult(
                segment_id=segment_id,
                winner="model_a",
                confidence=1.0,
                reasoning="Judge disabled, defaulting to model_a",
            )

        if len(candidates) < 2:
            return JudgeResult(
                segment_id=segment_id,
                winner="model_a",
                confidence=candidates[0].confidence if candidates else 1.0,
                reasoning="Only one candidate provided",
            )

        if not self.provider:
            return self._fallback_judge(segment_id, source, candidates)

        return self._llm_judge(segment_id, source, candidates, context)

    def _llm_judge(
        self,
        segment_id: str,
        source: str,
        candidates: List[TranslationCandidate],
        context: Optional[dict],
    ) -> JudgeResult:
        """Use LLM to judge translations."""
        prompt = JUDGE_PROMPT_TEMPLATE.format(
            source=source,
            translation_a=candidates[0].text,
            translation_b=candidates[1].text
            if len(candidates) > 1
            else candidates[0].text,
        )

        try:
            response = self.provider.generate(prompt)
            text = (
                response.text.strip()
                if hasattr(response, "text")
                else str(response).strip()
            )
            return self._parse_judge_response(segment_id, text, candidates)
        except Exception as e:
            logger.error(f"[JUDGE] LLM judge failed: {e}")
            if self.fallback_on_error:
                return self._fallback_judge(segment_id, source, candidates)
            raise

    def _parse_judge_response(
        self,
        segment_id: str,
        response: str,
        candidates: List[TranslationCandidate],
    ) -> JudgeResult:
        """Parse LLM JSON response into JudgeResult."""
        try:
            json_match = re.search(r"\{[^}]+\}", response, re.DOTALL)
            if json_match:
                data = json.loads(json_match.group())
            else:
                data = json.loads(response)

            winner_str = data.get("winner", "A").upper()
            if winner_str == "A":
                winner = "model_a"
            elif winner_str == "B":
                winner = "model_b"
            else:
                winner = "tie"

            return JudgeResult(
                segment_id=segment_id,
                winner=winner,
                confidence=float(data.get("confidence", 0.8)),
                reasoning=data.get("reasoning", ""),
                concerns=data.get("concerns", []),
            )
        except (json.JSONDecodeError, KeyError, ValueError) as e:
            logger.warning(f"[JUDGE] Failed to parse response: {e}")
            return JudgeResult(
                segment_id=segment_id,
                winner="model_a",
                confidence=0.6,
                reasoning=f"Parse error, defaulting to model_a. Response: {response[:100]}",
                concerns=["Judge response parse error"],
            )

    def _fallback_judge(
        self,
        segment_id: str,
        source: str,
        candidates: List[TranslationCandidate],
    ) -> JudgeResult:
        """Fallback: pick translation with fewest AI patterns."""
        scores = []
        for c in candidates:
            score = 0
            if "—" in c.text:
                score -= 2
            if "–" in c.text and not " – " in c.text:
                score -= 1
            if len(c.text) > 0 and len(c.text) < len(source) * 3:
                score += 1
            scores.append(score)

        best_idx = scores.index(max(scores)) if scores else 0
        winner = f"model_{chr(97 + best_idx)}"

        return JudgeResult(
            segment_id=segment_id,
            winner=winner,
            confidence=0.7,
            reasoning="Fallback heuristic: selected translation with better style patterns",
            concerns=["No LLM judge available, used heuristic"],
        )

    def judge_batch(
        self,
        segments: List[dict],
        candidates_map: dict[str, List[TranslationCandidate]],
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
                context=seg.get("context"),
            )
            results.append(result)

        logger.debug(f"[JUDGE] Batch judged {len(results)} segments")
        return results

    async def judge_async(
        self,
        segment_id: str,
        source: str,
        candidates: List[TranslationCandidate],
        context: Optional[dict] = None,
    ) -> JudgeResult:
        """Async version of judge for concurrent processing.

        MVP: Delegates to synchronous judge.
        Full: Makes actual async LLM call.
        """
        return self.judge(segment_id, source, candidates, context)


def create_judge(
    model: str = "claude-4.5-sonnet", timeout: int = 30, enabled: bool = True
) -> TranslationJudge:
    """Factory function to create a TranslationJudge.

    Args:
        model: Model identifier (reserved for future provider selection)
        timeout: Timeout for judge decisions in seconds
        enabled: Whether judge model is active

    Returns:
        Configured TranslationJudge instance
    """
    return TranslationJudge(model=model, timeout=timeout, enabled=enabled)
