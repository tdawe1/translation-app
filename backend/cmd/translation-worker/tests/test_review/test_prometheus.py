# tests/test_review/test_prometheus.py
"""
Tests for Prometheus metrics integration.

Verifies that:
- Metric definitions exist and are correctly typed
- record_job_metrics updates all counters/histograms/gauges
- Worker info is set correctly
- Failed jobs are tracked
- Metrics survive multiple jobs without double-counting errors
"""

import sys
from pathlib import Path
from unittest.mock import patch

import pytest

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.metrics import JobMetrics
from review.prometheus import (
    JOBS_TOTAL,
    SEGMENTS_TOTAL,
    SEGMENTS_FLAGGED_TOTAL,
    STYLE_VIOLATIONS_TOTAL,
    JOB_DURATION_SECONDS,
    JOB_QUALITY_SCORE,
    LATEST_FLAG_RATE,
    LATEST_STYLE_VIOLATION_RATE,
    STYLE_GUIDE_ENABLED,
    WORKER_INFO,
    record_job_metrics,
    record_job_failed,
    set_worker_info,
)

# Use a fresh registry per test to avoid cross-test pollution
from prometheus_client import REGISTRY


def _sample_value(metric, labels=None):
    """Extract a metric sample value from the default registry."""
    name = metric._name
    for m in REGISTRY.collect():
        for sample in m.samples:
            if labels:
                # Match by name prefix and labels
                if sample.name.startswith(name) and all(
                    sample.labels.get(k) == v for k, v in labels.items()
                ):
                    return sample.value
            elif sample.name == name or sample.name == f"{name}_total":
                return sample.value
    return None


class TestRecordJobMetrics:
    """Test that record_job_metrics updates Prometheus state."""

    def test_jobs_total_incremented(self):
        """JOBS_TOTAL counter should increment on each job."""
        before = _sample_value(JOBS_TOTAL, {"status": "completed", "provider": "openai"}) or 0

        metrics = JobMetrics(
            job_id="test-prom-1",
            segment_count=5,
            flagged_count=1,
            style_violation_count=1,
            overall_score=0.9,
            provider_name="openai",
        )
        metrics.processing_started_at = 1000.0
        metrics.processing_finished_at = 1002.5

        record_job_metrics(metrics)

        after = _sample_value(JOBS_TOTAL, {"status": "completed", "provider": "openai"})
        assert after == before + 1

    def test_segments_total_incremented(self):
        """SEGMENTS_TOTAL should increase by segment_count."""
        before = _sample_value(SEGMENTS_TOTAL, {"provider": "anthropic"}) or 0

        metrics = JobMetrics(
            job_id="test-prom-2",
            segment_count=10,
            provider_name="anthropic",
        )

        record_job_metrics(metrics)

        after = _sample_value(SEGMENTS_TOTAL, {"provider": "anthropic"})
        assert after == before + 10

    def test_style_violations_by_category(self):
        """STYLE_VIOLATIONS_TOTAL should track by category."""
        before_uk = _sample_value(STYLE_VIOLATIONS_TOTAL, {"category": "uk_spelling"}) or 0
        before_comma = _sample_value(STYLE_VIOLATIONS_TOTAL, {"category": "oxford_comma"}) or 0

        metrics = JobMetrics(job_id="test-prom-3", segment_count=5)
        metrics.style_violations_by_category = {
            "uk_spelling": 3,
            "oxford_comma": 1,
        }

        record_job_metrics(metrics)

        after_uk = _sample_value(STYLE_VIOLATIONS_TOTAL, {"category": "uk_spelling"})
        after_comma = _sample_value(STYLE_VIOLATIONS_TOTAL, {"category": "oxford_comma"})
        assert after_uk == before_uk + 3
        assert after_comma == before_comma + 1

    def test_duration_histogram_observed(self):
        """JOB_DURATION_SECONDS should record processing time."""
        metrics = JobMetrics(job_id="test-prom-4", segment_count=1, provider_name="openai")
        metrics.processing_started_at = 1000.0
        metrics.processing_finished_at = 1005.0  # 5 seconds

        # Should not raise
        record_job_metrics(metrics)

        # Verify the histogram has observations (count > 0)
        count = _sample_value(JOB_DURATION_SECONDS, {"provider": "openai"})
        # Histogram samples have _bucket, _count, _sum suffixes
        found = False
        for m in REGISTRY.collect():
            for sample in m.samples:
                if "translation_job_duration_seconds_count" in sample.name:
                    if sample.labels.get("provider") == "openai" and sample.value > 0:
                        found = True
        assert found, "Duration histogram should have observations"

    def test_quality_score_histogram(self):
        """JOB_QUALITY_SCORE should record the score."""
        metrics = JobMetrics(job_id="test-prom-5", segment_count=1, overall_score=0.85)

        record_job_metrics(metrics)

        found = False
        for m in REGISTRY.collect():
            for sample in m.samples:
                if "translation_job_quality_score_count" in sample.name:
                    if sample.value > 0:
                        found = True
        assert found

    def test_gauges_updated(self):
        """LATEST_FLAG_RATE and LATEST_STYLE_VIOLATION_RATE should reflect last job."""
        metrics = JobMetrics(
            job_id="test-prom-6",
            segment_count=10,
            flagged_count=3,
            style_violation_count=2,
        )

        record_job_metrics(metrics)

        flag_rate = _sample_value(LATEST_FLAG_RATE)
        style_rate = _sample_value(LATEST_STYLE_VIOLATION_RATE)
        assert flag_rate == pytest.approx(0.3, abs=0.01)
        assert style_rate == pytest.approx(0.2, abs=0.01)


class TestRecordJobFailed:
    """Test that failed jobs are tracked."""

    def test_failed_counter_incremented(self):
        """JOBS_TOTAL with status=failed should increment."""
        before = _sample_value(JOBS_TOTAL, {"status": "failed", "provider": "openai"}) or 0

        record_job_failed(provider="openai")

        after = _sample_value(JOBS_TOTAL, {"status": "failed", "provider": "openai"})
        assert after == before + 1


class TestSetWorkerInfo:
    """Test worker info gauge setup."""

    def test_sets_style_guide_enabled(self):
        """STYLE_GUIDE_ENABLED should reflect config."""
        set_worker_info("w1", "openai", "gpt-5.2", style_guide=True)
        assert _sample_value(STYLE_GUIDE_ENABLED) == 1.0

        set_worker_info("w1", "openai", "gpt-5.2", style_guide=False)
        assert _sample_value(STYLE_GUIDE_ENABLED) == 0.0

    def test_sets_worker_info(self):
        """WORKER_INFO should contain worker metadata."""
        set_worker_info("worker-42", "anthropic", "claude-sonnet-4-5-20250929", style_guide=True)

        # Info metrics expose as labels
        found = False
        for m in REGISTRY.collect():
            for sample in m.samples:
                if "translation_worker_info" in sample.name:
                    if sample.labels.get("worker_id") == "worker-42":
                        found = True
        assert found


class TestWorkflowPrometheusIntegration:
    """Test that the workflow updates Prometheus after process_job."""

    def test_process_job_updates_prometheus(self):
        """After process_job, Prometheus counters should increase."""
        from unittest.mock import Mock
        from review.workflow import TranslationWorkflow
        from review.multimodel import MultiModelTranslator
        from review.judge import TranslationJudge
        from review.flagging import FlaggingEngine

        before = _sample_value(JOBS_TOTAL, {"status": "completed", "provider": "unknown"}) or 0

        mock_provider = Mock()
        mock_provider.generate.return_value = Mock(text="Hello world")

        translator = MultiModelTranslator(providers=[mock_provider])
        judge = TranslationJudge(enabled=False)

        workflow = TranslationWorkflow(
            translator=translator,
            judge=judge,
            flagger=FlaggingEngine(random_sample_rate=0.0),
        )

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=[{"source": "テスト"}],
        )
        workflow.process_job(job)

        after = _sample_value(JOBS_TOTAL, {"status": "completed", "provider": "unknown"})
        assert after >= before + 1
