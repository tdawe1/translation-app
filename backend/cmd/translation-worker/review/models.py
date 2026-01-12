# review/models.py
"""
Data models for bilingual translation review workflow.

Defines the core data structures used throughout the review system:
- TranslationJob: Top-level job tracking
- TranslationSegment: Individual translated segments
- JudgeResult: Judge model output
- TranslationCandidate: Multi-model translation output
"""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional, List, Dict, Any, Literal


@dataclass
class TranslationCandidate:
    """A single translation candidate from one model.

    Attributes:
        model_name: Identifier for the model that produced this translation
        text: The translated text
        confidence: Model's confidence score (0.0-1.0)
        glossary_matches: List of glossary terms that were matched/applied
        latency_ms: Time taken to generate this translation
    """

    model_name: str
    text: str
    confidence: float = 1.0
    glossary_matches: List[str] = field(default_factory=list)
    latency_ms: int = 0


@dataclass
class JudgeResult:
    """Result from the judge model evaluating translations.

    Attributes:
        segment_id: ID of the segment being judged
        winner: Which translation won ("model_a", "model_b", "edited", "tie")
        confidence: Judge's confidence in the decision (0.0-1.0)
        reasoning: Explanation for the decision
        concerns: List of issues identified (empty if no concerns)
        suggested_edits: Merged/improved version if tie, or edits to winner
    """

    segment_id: str
    winner: Literal["model_a", "model_b", "edited", "tie"]
    confidence: float
    reasoning: str
    concerns: List[str] = field(default_factory=list)
    suggested_edits: Optional[str] = None


@dataclass
class TranslationSegment:
    """A single translated segment with full metadata.

    Attributes:
        id: Unique segment identifier
        job_id: Parent job ID
        source: Original source text
        target: Final translated text (after judge selection)
        context: Additional context (slide number, position, etc.)
        judge_winner: Which model's output was selected
        judge_confidence: Judge's confidence in this selection
        judge_reasoning: Explanation for the selection
        is_flagged: Whether this segment needs human review
        flag_reason: Reason for flagging, if any
        model_a_output: Output from model A (for audit trail)
        model_b_output: Output from model B (for audit trail)
        glossary_terms: Glossary terms that were matched
    """

    id: str
    job_id: str
    source: str
    target: str
    context: Dict[str, Any] = field(default_factory=dict)
    judge_winner: Literal["model_a", "model_b", "edited", "tie"] = "model_a"
    judge_confidence: float = 1.0
    judge_reasoning: str = ""
    is_flagged: bool = False
    flag_reason: Optional[str] = None
    model_a_output: Optional[str] = None
    model_b_output: Optional[str] = None
    glossary_terms: List[str] = field(default_factory=list)

    def to_csv_row(self) -> Dict[str, str]:
        """Convert to CSV-compatible dictionary.

        Returns a dict with flattened string values for CSV export.
        """
        return {
            "segment_id": self.id,
            "source": self.source,
            "target": self.target,
            "judge_winner": self.judge_winner,
            "judge_confidence": str(self.judge_confidence),
            "judge_reasoning": self.judge_reasoning,
            "is_flagged": "1" if self.is_flagged else "0",
            "flag_reason": self.flag_reason or "",
            "model_a_output": self.model_a_output or "",
            "model_b_output": self.model_b_output or "",
            "glossary_terms": ",".join(self.glossary_terms),
            "context": str(self.context),
        }


@dataclass
class TranslationJob:
    """A complete translation job with review tracking.

    Attributes:
        id: Unique job identifier
        source_file: Path to the source file
        target_file: Path to the output file
        status: Current job status
        overall_score: Aggregate quality score (0.0-1.0)
        segment_count: Total number of segments
        flagged_count: Number of segments flagged for review
        judge_resolutions: Number of segments resolved by judge
        created_at: When the job was created
        completed_at: When translation finished
        approved_at: When human approved the output
        approved_by: User who approved (if applicable)
        segments: List of translation segments
        project_type: Project classification (affects thresholds)
        approval_mode: "blocking" or "async" review
    """

    id: str
    source_file: str
    target_file: str
    status: Literal[
        "processing",
        "pending_approval",
        "approved",
        "rejected",
        "exported"
    ] = "processing"
    overall_score: float = 1.0
    segment_count: int = 0
    flagged_count: int = 0
    judge_resolutions: int = 0
    created_at: datetime = field(default_factory=datetime.now)
    completed_at: Optional[datetime] = None
    approved_at: Optional[datetime] = None
    approved_by: Optional[str] = None
    segments: List[TranslationSegment] = field(default_factory=list)
    project_type: str = "routine"
    approval_mode: Literal["blocking", "async"] = "async"

    def calculate_score(self) -> float:
        """Calculate overall job quality score.

        Returns a score 0.0-1.0 based on:
        - Average judge confidence across segments
        - Penalty for flagged segments
        - Penalty for ties (disagreement between models)
        """
        if not self.segments:
            return 1.0

        total_confidence = sum(s.judge_confidence for s in self.segments)
        avg_confidence = total_confidence / len(self.segments)

        # Penalty for flagged segments
        flag_penalty = (self.flagged_count / len(self.segments)) * 0.2

        # Penalty for ties (model disagreement)
        tie_count = sum(1 for s in self.segments if s.judge_winner == "tie")
        tie_penalty = (tie_count / len(self.segments)) * 0.1

        return max(0.0, min(1.0, avg_confidence - flag_penalty - tie_penalty))

    def update_metrics(self):
        """Recalculate aggregate metrics from segments."""
        self.segment_count = len(self.segments)
        self.flagged_count = sum(1 for s in self.segments if s.is_flagged)
        self.judge_resolutions = sum(
            1 for s in self.segments
            if s.judge_winner in ("model_a", "model_b")
        )
        self.overall_score = self.calculate_score()

    def get_flagged_segments(self) -> List[TranslationSegment]:
        """Return all flagged segments, sorted by priority (lowest confidence first)."""
        flagged = [s for s in self.segments if s.is_flagged]
        return sorted(flagged, key=lambda s: s.judge_confidence)

    def can_auto_approve(self, threshold: float = 0.85) -> bool:
        """Check if job meets criteria for auto-approval.

        Args:
            threshold: Minimum score required for auto-approval

        Returns:
            True if job can be auto-approved
        """
        if self.status != "pending_approval":
            return False
        if self.overall_score < threshold:
            return False
        if self.approval_mode == "blocking" and self.flagged_count > 0:
            return False
        return True


@dataclass
class ReviewConfig:
    """Configuration for the review workflow.

    Attributes:
        auto_approve_threshold: Score above which jobs auto-approve
        block_threshold: Score below which jobs block output
        random_sample_rate: Fraction of low-risk jobs to sample for review
        judge_enabled: Whether judge model is active
        judge_model: Model identifier for judge
        judge_timeout: Timeout for judge decisions (seconds)
        csv_export_enabled: Whether bilingual CSV export is enabled
        csv_export_path: Directory for CSV exports
    """

    auto_approve_threshold: float = 0.85
    block_threshold: float = 0.70
    random_sample_rate: float = 0.02
    judge_enabled: bool = True
    judge_model: str = "claude-4.5-sonnet"
    judge_timeout: int = 30
    csv_export_enabled: bool = True
    csv_export_path: str = "."  # Current directory, more portable than /watch/bilingual/

    @classmethod
    def for_project_type(cls, project_type: str) -> "ReviewConfig":
        """Get configuration for a specific project type.

        Args:
            project_type: "critical" or "routine"

        Returns:
            Configuration with appropriate thresholds
        """
        configs = {
            "critical": cls(
                auto_approve_threshold=0.95,
                block_threshold=0.70,
                random_sample_rate=0.10,
            ),
            "routine": cls(
                auto_approve_threshold=0.85,
                block_threshold=0.70,
                random_sample_rate=0.02,
            ),
        }
        return configs.get(project_type, cls())
