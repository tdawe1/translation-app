# review/metrics.py
"""
Job-level metrics for translation workflow observability.

Collects per-job signals that feed dashboards and alerting:
- Style violation counts by category
- Provider latency distribution
- Flag rate and reasons
- Overall quality score
"""

import json
import time
from dataclasses import dataclass, field
from typing import Dict, List, Optional


@dataclass
class JobMetrics:
    """Structured metrics for a single translation job.

    Designed to be JSON-serializable for logging pipelines,
    dashboards, and audit trails.
    """

    job_id: str
    segment_count: int = 0
    flagged_count: int = 0
    style_violation_count: int = 0
    style_violations_by_category: Dict[str, int] = field(default_factory=dict)
    flag_reasons: List[str] = field(default_factory=list)
    overall_score: float = 1.0
    provider_name: str = ""
    model_name: str = ""
    style_guide_enabled: bool = False
    processing_started_at: Optional[float] = None
    processing_finished_at: Optional[float] = None

    def start_timer(self):
        """Mark processing start."""
        self.processing_started_at = time.time()

    def stop_timer(self):
        """Mark processing end."""
        self.processing_finished_at = time.time()

    @property
    def processing_duration_ms(self) -> Optional[int]:
        """Total processing time in milliseconds."""
        if self.processing_started_at and self.processing_finished_at:
            return int((self.processing_finished_at - self.processing_started_at) * 1000)
        return None

    @property
    def flag_rate(self) -> float:
        """Fraction of segments flagged (0.0-1.0)."""
        if self.segment_count == 0:
            return 0.0
        return self.flagged_count / self.segment_count

    @property
    def style_violation_rate(self) -> float:
        """Fraction of segments with style issues (0.0-1.0)."""
        if self.segment_count == 0:
            return 0.0
        return self.style_violation_count / self.segment_count

    def record_style_issues(self, issues: List[dict]):
        """Record style issues from a single segment."""
        if not issues:
            return
        self.style_violation_count += 1
        for issue in issues:
            cat = issue.get("category", "unknown")
            self.style_violations_by_category[cat] = (
                self.style_violations_by_category.get(cat, 0) + 1
            )

    def record_flag(self, reason: str):
        """Record a flagged segment."""
        self.flagged_count += 1
        self.flag_reasons.append(reason)

    def to_dict(self) -> dict:
        """Convert to JSON-serializable dictionary."""
        return {
            "job_id": self.job_id,
            "segment_count": self.segment_count,
            "flagged_count": self.flagged_count,
            "flag_rate": round(self.flag_rate, 4),
            "style_violation_count": self.style_violation_count,
            "style_violation_rate": round(self.style_violation_rate, 4),
            "style_violations_by_category": self.style_violations_by_category,
            "flag_reasons": self.flag_reasons,
            "overall_score": round(self.overall_score, 4),
            "provider_name": self.provider_name,
            "model_name": self.model_name,
            "style_guide_enabled": self.style_guide_enabled,
            "processing_duration_ms": self.processing_duration_ms,
        }

    def to_json(self) -> str:
        """Serialize to JSON string."""
        return json.dumps(self.to_dict())
