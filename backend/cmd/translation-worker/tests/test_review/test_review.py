# tests/test_review/test_review.py
"""
Tests for bilingual review workflow components.

Tests all stub implementations and data models.
"""

import pytest
import sys
import tempfile
from pathlib import Path

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.models import (
    TranslationCandidate,
    JudgeResult,
    TranslationSegment,
    TranslationJob,
    ReviewConfig,
)

from review.multimodel import MultiModelTranslator, create_multimodel_translator
from review.judge import TranslationJudge, create_judge
from review.flagging import FlaggingEngine, create_flagging_engine
from review.exporter import BilingualCSVExporter, create_exporter


class TestTranslationCandidate:
    """Test TranslationCandidate dataclass."""

    def test_creation(self):
        """Should create candidate with defaults."""
        candidate = TranslationCandidate(
            model_name="model_a",
            text="Hello"
        )
        assert candidate.model_name == "model_a"
        assert candidate.text == "Hello"
        assert candidate.confidence == 1.0
        assert candidate.glossary_matches == []
        assert candidate.latency_ms == 0

    def test_with_glossary(self):
        """Should store glossary matches."""
        candidate = TranslationCandidate(
            model_name="model_a",
            text="Hello",
            glossary_matches=["term1", "term2"]
        )
        assert candidate.glossary_matches == ["term1", "term2"]


class TestJudgeResult:
    """Test JudgeResult dataclass."""

    def test_creation(self):
        """Should create result with required fields."""
        result = JudgeResult(
            segment_id="seg1",
            winner="model_a",
            confidence=0.9,
            reasoning="Good translation"
        )
        assert result.segment_id == "seg1"
        assert result.winner == "model_a"
        assert result.confidence == 0.9
        assert result.concerns == []

    def test_with_concerns(self):
        """Should store concerns."""
        result = JudgeResult(
            segment_id="seg1",
            winner="tie",
            confidence=0.5,
            reasoning="Disagreement",
            concerns=["Terminology", "Style"]
        )
        assert result.concerns == ["Terminology", "Style"]


class TestTranslationSegment:
    """Test TranslationSegment dataclass."""

    def test_creation(self):
        """Should create segment with defaults."""
        segment = TranslationSegment(
            id="seg1",
            job_id="job1",
            source="Hello",
            target="こんにちは"
        )
        assert segment.id == "seg1"
        assert segment.job_id == "job1"
        assert segment.source == "Hello"
        assert segment.target == "こんにちは"
        assert segment.is_flagged is False
        assert segment.judge_winner == "model_a"

    def test_to_csv_row(self):
        """Should convert to CSV-compatible dict."""
        segment = TranslationSegment(
            id="seg1",
            job_id="job1",
            source="Hello",
            target="こんにちは",
            judge_confidence=0.95,
            is_flagged=False
        )
        row = segment.to_csv_row()
        assert row["segment_id"] == "seg1"
        assert row["source"] == "Hello"
        assert row["target"] == "こんにちは"
        assert row["judge_confidence"] == "0.95"
        assert row["is_flagged"] == "0"


class TestTranslationJob:
    """Test TranslationJob dataclass."""

    def test_creation(self):
        """Should create job with defaults."""
        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt"
        )
        assert job.id == "job1"
        assert job.status == "processing"
        assert job.segments == []

    def test_calculate_score(self):
        """Should calculate score from segments."""
        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt"
        )

        # Add high-confidence segment
        job.segments.append(TranslationSegment(
            id="seg1",
            job_id="job1",
            source="Hello",
            target="こんにちは",
            judge_confidence=0.95
        ))

        # Add low-confidence flagged segment
        job.segments.append(TranslationSegment(
            id="seg2",
            job_id="job1",
            source="World",
            target="世界",
            judge_confidence=0.6,
            is_flagged=True
        ))

        score = job.calculate_score()
        assert 0.5 < score < 1.0  # Penalized for flag

    def test_update_metrics(self):
        """Should recalculate aggregate metrics."""
        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt"
        )

        job.segments.append(TranslationSegment(
            id="seg1",
            job_id="job1",
            source="Hello",
            target="こんにちは",
            is_flagged=True
        ))

        job.segments.append(TranslationSegment(
            id="seg2",
            job_id="job1",
            source="World",
            target="世界"
        ))

        job.update_metrics()
        assert job.segment_count == 2
        assert job.flagged_count == 1

    def test_can_auto_approve(self):
        """Should check auto-approve conditions."""
        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt",
            status="pending_approval",
            approval_mode="async"
        )
        job.overall_score = 0.90

        assert job.can_auto_approve(threshold=0.85) is True
        assert job.can_auto_approve(threshold=0.95) is False

    def test_blocking_mode_blocks_flagged(self):
        """Blocking mode should not auto-approve with flags."""
        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt",
            status="pending_approval",
            approval_mode="blocking"
        )
        job.overall_score = 0.95
        job.flagged_count = 1

        assert job.can_auto_approve(threshold=0.85) is False


class TestReviewConfig:
    """Test ReviewConfig configuration."""

    def test_defaults(self):
        """Should have sensible defaults."""
        config = ReviewConfig()
        assert config.auto_approve_threshold == 0.85
        assert config.block_threshold == 0.70
        assert config.random_sample_rate == 0.02

    def test_for_project_type(self):
        """Should return appropriate config per type."""
        critical = ReviewConfig.for_project_type("critical")
        assert critical.auto_approve_threshold == 0.95
        assert critical.random_sample_rate == 0.10

        routine = ReviewConfig.for_project_type("routine")
        assert routine.auto_approve_threshold == 0.85
        assert routine.random_sample_rate == 0.02


class TestMultiModelTranslator:
    """Test multi-model translator stub."""

    def test_translate(self):
        """Should generate stub translations."""
        translator = MultiModelTranslator()
        candidates = translator.translate("Hello")

        assert len(candidates) == 2
        assert candidates[0].model_name == "model_a"
        assert candidates[1].model_name == "model_b"
        assert "[Model A]" in candidates[0].text
        assert "[Model B]" in candidates[1].text

    def test_with_glossary(self):
        """Should include glossary in stub output."""
        translator = MultiModelTranslator()
        candidates = translator.translate(
            "Hello",
            glossary_terms=["term1"]
        )

        assert any("glossary: term1" in c.text for c in candidates)

    def test_translate_batch(self):
        """Should translate multiple segments."""
        translator = MultiModelTranslator()
        sources = ["Hello", "World"]

        results = translator.translate_batch(sources)

        assert len(results) == 2
        for candidates in results.values():
            assert len(candidates) == 2


class TestTranslationJudge:
    """Test translation judge stub."""

    def test_judge(self):
        """Should return stub result."""
        judge = TranslationJudge()
        candidates = [
            TranslationCandidate("model_a", "Translation A"),
            TranslationCandidate("model_b", "Translation B")
        ]

        result = judge.judge("seg1", "Hello", candidates)

        assert result.segment_id == "seg1"
        assert result.winner in ["model_a", "model_b"]
        assert 0.0 <= result.confidence <= 1.0

    def test_disabled_returns_default(self):
        """Disabled judge should return model_a."""
        judge = TranslationJudge(enabled=False)
        candidates = [
            TranslationCandidate("model_a", "Translation A"),
            TranslationCandidate("model_b", "Translation B")
        ]

        result = judge.judge("seg1", "Hello", candidates)

        assert result.winner == "model_a"
        assert result.confidence == 1.0

    def test_single_candidate(self):
        """Should handle single candidate."""
        judge = TranslationJudge()
        candidates = [
            TranslationCandidate("model_a", "Translation A")
        ]

        result = judge.judge("seg1", "Hello", candidates)

        assert result.winner == "model_a"

    def test_judge_batch(self):
        """Should judge multiple segments."""
        judge = TranslationJudge()
        segments = [
            {"id": "s1", "source": "Hello"},
            {"id": "s2", "source": "World"}
        ]
        candidates_map = {
            "s1": [
                TranslationCandidate("model_a", "A"),
                TranslationCandidate("model_b", "B")
            ],
            "s2": [
                TranslationCandidate("model_a", "C"),
                TranslationCandidate("model_b", "D")
            ]
        }

        results = judge.judge_batch(segments, candidates_map)

        assert len(results) == 2
        for r in results:
            assert r.winner in ["model_a", "model_b"]


class TestFlaggingEngine:
    """Test flagging engine."""

    def test_low_confidence_flags(self):
        """Low confidence should trigger flag."""
        engine = FlaggingEngine(block_threshold=0.7)
        segment = TranslationSegment(
            id="seg1", job_id="job1", source="Hi", target="ハイ"
        )
        judge_result = JudgeResult(
            segment_id="seg1",
            winner="model_a",
            confidence=0.5,  # Low confidence
            reasoning="Uncertain"
        )

        is_flagged, reason = engine.evaluate(segment, judge_result)

        assert is_flagged is True
        assert "0.5" in reason

    def test_concerns_flag(self):
        """Concerns should trigger flag."""
        engine = FlaggingEngine()
        segment = TranslationSegment(
            id="seg1", job_id="job1", source="Hi", target="ハイ"
        )
        judge_result = JudgeResult(
            segment_id="seg1",
            winner="model_a",
            confidence=0.9,
            reasoning="Good",
            concerns=["Terminology"]
        )

        is_flagged, reason = engine.evaluate(segment, judge_result)

        assert is_flagged is True
        assert "Concerns" in reason

    def test_tie_flags(self):
        """Tie should trigger flag."""
        engine = FlaggingEngine()
        segment = TranslationSegment(
            id="seg1", job_id="job1", source="Hi", target="ハイ"
        )
        judge_result = JudgeResult(
            segment_id="seg1",
            winner="tie",
            confidence=0.8,
            reasoning="Disagreement"
        )

        is_flagged, reason = engine.evaluate(segment, judge_result)

        assert is_flagged is True
        assert "disagreement" in reason.lower()

    def test_high_confidence_passes(self):
        """High confidence with no concerns should pass."""
        engine = FlaggingEngine()
        segment = TranslationSegment(
            id="seg1", job_id="job1", source="Hi", target="ハイ"
        )
        judge_result = JudgeResult(
            segment_id="seg1",
            winner="model_a",
            confidence=0.95,
            reasoning="Excellent"
        )

        is_flagged, reason = engine.evaluate(segment, judge_result)

        assert is_flagged is False
        assert reason is None

    def test_calculate_priority(self):
        """Should calculate priority score."""
        engine = FlaggingEngine()
        segment = TranslationSegment(
            id="seg1", job_id="job1", source="Hi", target="ハイ"
        )
        judge_result = JudgeResult(
            segment_id="seg1",
            winner="model_a",
            confidence=0.6,  # Low confidence
            reasoning="Uncertain",
            concerns=["Style"]
        )

        priority = engine.calculate_priority(segment, judge_result)

        assert 0.0 < priority <= 1.0

    def test_flag_segment(self):
        """Should flag segment in place."""
        engine = FlaggingEngine()
        segment = TranslationSegment(
            id="seg1", job_id="job1", source="Hi", target="ハイ"
        )
        judge_result = JudgeResult(
            segment_id="seg1",
            winner="model_a",
            confidence=0.5,
            reasoning="Uncertain"
        )

        engine.flag_segment(segment, judge_result)

        assert segment.is_flagged is True
        assert "0.5" in segment.flag_reason


class TestBilingualCSVExporter:
    """Test CSV exporter."""

    def test_export_job(self):
        """Should export job to CSV."""
        with tempfile.TemporaryDirectory() as tmpdir:
            exporter = BilingualCSVExporter(output_dir=tmpdir)

            job = TranslationJob(
                id="job1",
                source_file="in.txt",
                target_file="out.txt"
            )
            job.segments.append(TranslationSegment(
                id="seg1",
                job_id="job1",
                source="Hello",
                target="こんにちは",
                judge_confidence=0.95
            ))

            filepath = exporter.export_job(job)

            assert Path(filepath).exists()
            assert "job1" in filepath

            # Verify content
            with open(filepath, "r", encoding="utf-8-sig") as f:
                content = f.read()
                assert "Hello" in content
                assert "こんにちは" in content

    def test_export_includes_headers(self):
        """Should include correct CSV headers."""
        with tempfile.TemporaryDirectory() as tmpdir:
            exporter = BilingualCSVExporter(output_dir=tmpdir)

            job = TranslationJob(
                id="job1",
                source_file="in.txt",
                target_file="out.txt"
            )

            filepath = exporter.export_job(job)

            with open(filepath, "r", encoding="utf-8-sig") as f:
                first_line = f.readline().strip()
                headers = first_line.split(",")
                assert "segment_id" in headers
                assert "source" in headers
                assert "target" in headers
                assert "judge_winner" in headers

    def test_get_export_summary(self):
        """Should generate export summary."""
        exporter = BilingualCSVExporter()

        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt",
            segment_count=10,
            flagged_count=2,
            overall_score=0.85
        )

        summary = exporter.get_export_summary(job)

        assert summary["job_id"] == "job1"
        assert summary["segment_count"] == 10
        assert summary["flagged_count"] == 2
        assert summary["overall_score"] == 0.85


class TestFactoryFunctions:
    """Test factory functions."""

    def test_create_judge(self):
        """Should create configured judge."""
        judge = create_judge(model="gpt-4")
        assert judge.model == "gpt-4"

    def test_create_flagging_engine(self):
        """Should create configured flagging engine."""
        engine = create_flagging_engine(block_threshold=0.8)
        assert engine.block_threshold == 0.8

    def test_create_exporter(self):
        """Should create configured exporter."""
        exporter = create_exporter(output_dir="/tmp/test")
        assert exporter.output_dir == Path("/tmp/test")

    def test_create_multimodel_translator(self):
        """Should create configured translator."""
        translator = create_multimodel_translator(parallel=False)
        assert translator.parallel is False
