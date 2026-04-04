# tests/test_review/test_metrics.py
"""Tests for review/metrics.py — structured job metrics."""

import json
import sys
import time
from pathlib import Path

import pytest

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.metrics import JobMetrics


class TestJobMetrics:
    """Tests for JobMetrics dataclass."""

    def test_basic_creation(self):
        m = JobMetrics(job_id="test-1", segment_count=10)
        assert m.job_id == "test-1"
        assert m.segment_count == 10
        assert m.flagged_count == 0
        assert m.style_violation_count == 0

    def test_flag_rate(self):
        m = JobMetrics(job_id="j1", segment_count=10, flagged_count=3)
        assert m.flag_rate == 0.3

    def test_flag_rate_zero_segments(self):
        m = JobMetrics(job_id="j1", segment_count=0)
        assert m.flag_rate == 0.0

    def test_style_violation_rate(self):
        m = JobMetrics(job_id="j1", segment_count=4, style_violation_count=2)
        assert m.style_violation_rate == 0.5

    def test_record_style_issues(self):
        m = JobMetrics(job_id="j1", segment_count=2)
        m.record_style_issues([
            {"category": "uk_spelling", "severity": "warning"},
            {"category": "currency_format", "severity": "warning"},
        ])
        assert m.style_violation_count == 1  # 1 segment had issues
        assert m.style_violations_by_category["uk_spelling"] == 1
        assert m.style_violations_by_category["currency_format"] == 1

    def test_record_style_issues_accumulates(self):
        m = JobMetrics(job_id="j1", segment_count=3)
        m.record_style_issues([{"category": "uk_spelling", "severity": "warning"}])
        m.record_style_issues([{"category": "uk_spelling", "severity": "warning"}])
        assert m.style_violation_count == 2
        assert m.style_violations_by_category["uk_spelling"] == 2

    def test_record_style_issues_empty_noop(self):
        m = JobMetrics(job_id="j1", segment_count=1)
        m.record_style_issues([])
        assert m.style_violation_count == 0

    def test_record_flag(self):
        m = JobMetrics(job_id="j1", segment_count=5)
        m.record_flag("Low confidence")
        m.record_flag("Style: uk_spelling")
        assert m.flagged_count == 2
        assert len(m.flag_reasons) == 2

    def test_timer(self):
        m = JobMetrics(job_id="j1")
        m.start_timer()
        time.sleep(0.01)
        m.stop_timer()
        assert m.processing_duration_ms is not None
        assert m.processing_duration_ms >= 10

    def test_timer_not_started(self):
        m = JobMetrics(job_id="j1")
        assert m.processing_duration_ms is None

    def test_to_dict(self):
        m = JobMetrics(
            job_id="j1",
            segment_count=10,
            flagged_count=2,
            style_violation_count=1,
            overall_score=0.85,
            provider_name="openai",
            model_name="gpt-5.2",
            style_guide_enabled=True,
        )
        d = m.to_dict()

        assert d["job_id"] == "j1"
        assert d["segment_count"] == 10
        assert d["flag_rate"] == 0.2
        assert d["style_violation_count"] == 1
        assert d["style_violation_rate"] == 0.1
        assert d["provider_name"] == "openai"
        assert d["style_guide_enabled"] is True

    def test_to_json(self):
        m = JobMetrics(job_id="j1", segment_count=5)
        j = m.to_json()
        parsed = json.loads(j)
        assert parsed["job_id"] == "j1"
        assert parsed["segment_count"] == 5
        assert "flag_rate" in parsed
        assert "style_violation_rate" in parsed
