# review/flagging.py
"""
Flagging engine for identifying segments needing human review.

Evaluates segments based on multiple signals:
- Judge confidence
- Model disagreements (ties)
- Length ratio anomalies
- Manual flags
- Random sampling for quality assurance
"""

import logging
import random
from typing import List, Optional
from .models import TranslationSegment, JudgeResult, ReviewConfig

logger = logging.getLogger(__name__)


class FlaggingEngine:
    """Identifies translation segments that need human review.

    Segments are flagged based on configurable thresholds and signals.
    """

    # Threshold constants
    BLOCK_THRESHOLD = 0.70  # Below this, block output
    REVIEW_THRESHOLD = 0.30  # Above this, include in review

    def __init__(
        self,
        block_threshold: float = BLOCK_THRESHOLD,
        review_threshold: float = REVIEW_THRESHOLD,
        random_sample_rate: float = 0.02
    ):
        """Initialize the flagging engine.

        Args:
            block_threshold: Confidence below which output is blocked
            review_threshold: Minimum priority score to include in review
            random_sample_rate: Fraction of low-risk segments to sample
        """
        self.block_threshold = block_threshold
        self.review_threshold = review_threshold
        self.random_sample_rate = random_sample_rate

    def evaluate(
        self,
        segment: TranslationSegment,
        judge_result: JudgeResult,
        config: Optional[ReviewConfig] = None
    ) -> tuple[bool, Optional[str]]:
        """Evaluate whether a segment should be flagged for review.

        Args:
            segment: The translation segment to evaluate
            judge_result: Result from the judge model
            config: Optional review config (uses defaults if None)

        Returns:
            Tuple of (is_flagged, flag_reason)
        """
        if config is None:
            config = ReviewConfig()

        # Check judge confidence
        if judge_result.confidence < config.block_threshold:
            reason = f"Low confidence ({judge_result.confidence:.2f} < {config.block_threshold})"
            return True, reason

        # Check for judge concerns
        if judge_result.concerns:
            reason = f"Concerns: {', '.join(judge_result.concerns)}"
            return True, reason

        # Check for tie (model disagreement)
        if judge_result.winner == "tie":
            return True, "Model disagreement (tie)"

        # Check length ratio anomaly
        source_len = len(segment.source.strip())
        target_len = len(segment.target.strip())
        if source_len > 0:
            ratio = target_len / source_len
            if ratio < 0.5 or ratio > 2.0:
                return True, f"Length ratio anomaly: {ratio:.2f}"

        # Random sampling (for quality assurance)
        if random.random() < config.random_sample_rate:
            return True, "Random quality sample"

        return False, None

    def calculate_priority(
        self,
        segment: TranslationSegment,
        judge_result: JudgeResult
    ) -> float:
        """Calculate review priority score for a segment.

        Higher score = higher priority for human review.

        Args:
            segment: The translation segment
            judge_result: Result from the judge model

        Returns:
            Priority score from 0.0 (low) to 1.0 (high)
        """
        # Low confidence contributes most to priority
        confidence_penalty = (1.0 - judge_result.confidence) * 0.5

        # Concerns add to priority
        concern_penalty = len(judge_result.concerns) * 0.15

        # Manual flag (if present) is highest priority
        manual_penalty = 1.0 if segment.is_flagged else 0.0

        priority = confidence_penalty + concern_penalty + (manual_penalty * 0.2)
        return min(1.0, priority)

    def flag_segment(
        self,
        segment: TranslationSegment,
        judge_result: JudgeResult,
        config: Optional[ReviewConfig] = None
    ) -> None:
        """Flag a segment for review in place.

        Modifies the segment with is_flagged and flag_reason.

        Args:
            segment: The segment to potentially flag
            judge_result: Result from the judge model
            config: Optional review config
        """
        is_flagged, reason = self.evaluate(segment, judge_result, config)

        segment.is_flagged = is_flagged
        if is_flagged:
            segment.flag_reason = reason
            logger.debug(f"[FLAG] {segment.id}: {reason}")

    def flag_batch(
        self,
        segments: List[TranslationSegment],
        judge_results: List[JudgeResult],
        config: Optional[ReviewConfig] = None
    ) -> int:
        """Flag multiple segments for review.

        Args:
            segments: List of translation segments
            judge_results: Corresponding judge results
            config: Optional review config

        Returns:
            Number of segments flagged
        """
        count = 0
        for segment, judge_result in zip(segments, judge_results):
            self.flag_segment(segment, judge_result, config)
            if segment.is_flagged:
                count += 1

        logger.info(f"[FLAG] Flagged {count}/{len(segments)} segments")
        return count

    def get_blocking_segments(
        self,
        segments: List[TranslationSegment]
    ) -> List[TranslationSegment]:
        """Get segments that block output (require review before export).

        Args:
            segments: List of segments to check

        Returns:
            List of segments that must be reviewed before output
        """
        return [s for s in segments if s.is_flagged and s.judge_confidence < self.block_threshold]

    def should_block_output(
        self,
        job_segments: List[TranslationSegment],
        config: Optional[ReviewConfig] = None
    ) -> bool:
        """Check if any segments block the output.

        Args:
            job_segments: All segments in the job
            config: Optional review config

        Returns:
            True if output should be blocked pending review
        """
        if config is None:
            config = ReviewConfig()

        blocking = self.get_blocking_segments(job_segments)
        should_block = len(blocking) > 0

        if should_block:
            logger.warning(f"[FLAG] {len(blocking)} segments block output")

        return should_block


def create_flagging_engine(
    block_threshold: float = 0.70,
    review_threshold: float = 0.30,
    random_sample_rate: float = 0.02
) -> FlaggingEngine:
    """Factory function to create a FlaggingEngine.

    Args:
        block_threshold: Confidence below which output is blocked
        review_threshold: Minimum priority score to include in review
        random_sample_rate: Fraction of low-risk segments to sample

    Returns:
        Configured FlaggingEngine instance
    """
    return FlaggingEngine(
        block_threshold=block_threshold,
        review_threshold=review_threshold,
        random_sample_rate=random_sample_rate
    )
