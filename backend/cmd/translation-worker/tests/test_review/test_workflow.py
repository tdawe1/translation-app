# tests/test_review/test_workflow.py
"""
Tests for translation review workflow orchestrator.

Tests the end-to-end workflow coordination including:
- Job creation and management
- Multi-model translation coordination
- Judge evaluation integration
- Flagging application
- CSV export
- Auto-approve logic
"""

import pytest
import sys
import tempfile
from pathlib import Path
from unittest.mock import Mock, patch

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.models import (
    TranslationCandidate,
    TranslationJob,
    ReviewConfig,
)
from review.workflow import (
    TranslationWorkflow,
    ReviewWorkflowBuilder,
    create_workflow,
)
from review.multimodel import MultiModelTranslator
from review.judge import TranslationJudge
from review.flagging import FlaggingEngine
from review.exporter import BilingualCSVExporter


class TestTranslationWorkflow:
    """Test TranslationWorkflow orchestrator."""

    def test_initialization_default(self):
        """Should initialize with default components."""
        workflow = TranslationWorkflow()

        assert workflow.translator is not None
        assert isinstance(workflow.translator, MultiModelTranslator)
        assert workflow.judge is not None
        assert isinstance(workflow.judge, TranslationJudge)
        assert workflow.flagger is not None
        assert isinstance(workflow.flagger, FlaggingEngine)
        assert workflow.exporter is not None
        assert isinstance(workflow.exporter, BilingualCSVExporter)

    def test_initialization_custom(self):
        """Should accept custom components."""
        custom_translator = MultiModelTranslator(parallel=False)
        custom_judge = TranslationJudge(enabled=True)
        custom_flagger = FlaggingEngine(block_threshold=0.8)
        custom_config = ReviewConfig(auto_approve_threshold=0.9)

        workflow = TranslationWorkflow(
            translator=custom_translator,
            judge=custom_judge,
            flagger=custom_flagger,
            config=custom_config,
        )

        assert workflow.translator is custom_translator
        assert workflow.judge is custom_judge
        assert workflow.flagger is custom_flagger
        assert workflow.config.auto_approve_threshold == 0.9

    def test_create_job_empty(self):
        """Should create job with no segments."""
        workflow = TranslationWorkflow()

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
        )

        assert job.id
        assert job.source_file == "in.txt"
        assert job.target_file == "out.txt"
        assert job.segment_count == 0
        assert job.status == "processing"
        assert job.project_type == "routine"

    def test_create_job_with_segments(self):
        """Should create job with pre-defined segments."""
        workflow = TranslationWorkflow()

        segments = [
            {"source": "Hello", "context": {"slide": 1}},
            {"source": "World", "context": {"slide": 1}},
        ]

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=segments,
        )

        assert job.segment_count == 2
        assert job.segments[0].source == "Hello"
        assert job.segments[1].source == "World"
        assert job.segments[0].context == {"slide": 1}

    def test_create_job_critical_project(self):
        """Should use blocking mode for critical projects."""
        workflow = TranslationWorkflow()

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            project_type="critical",
        )

        assert job.project_type == "critical"
        assert job.approval_mode == "blocking"

    def test_process_job_translates_segments(self):
        """Should translate all segments in job using mock providers."""
        from review.models import TranslationCandidate

        # Provide a mock provider so translations actually happen
        mock_provider = Mock()
        mock_provider.generate.return_value = Mock(text="Hello translated")

        translator = MultiModelTranslator(providers=[mock_provider])
        workflow = TranslationWorkflow(translator=translator)

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=[
                {"source": "Hello"},
                {"source": "World"},
            ],
        )

        processed = workflow.process_job(job)

        assert len(processed.segments) == 2
        assert processed.segments[0].target  # Should have translation
        assert processed.segments[1].target  # Should have translation
        assert processed.status == "pending_approval"
        assert processed.completed_at is not None

    def test_process_job_applies_judge_results(self):
        """Should apply judge evaluation to segments."""
        workflow = TranslationWorkflow()

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=[{"source": "Hello"}],
        )

        # Set seed for reproducible judge results
        import random
        random.seed(42)

        processed = workflow.process_job(job)

        segment = processed.segments[0]
        assert segment.judge_winner in ["model_a", "model_b"]
        assert 0.0 <= segment.judge_confidence <= 1.0
        assert segment.judge_reasoning

    def test_process_job_applies_flagging(self):
        """Should flag segments based on judge results."""
        from review.models import JudgeResult

        workflow = TranslationWorkflow(
            flagger=FlaggingEngine(block_threshold=0.95),  # Very strict
        )

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=[{"source": "Hello"}],
        )

        # Create a mock judge that returns low confidence
        low_conf_judge = Mock(return_value=JudgeResult(
            segment_id="seg1",
            winner="model_a",
            confidence=0.5,  # Low confidence
            reasoning="Uncertain",
            concerns=["Terminology"],
        ))
        workflow.judge.judge = low_conf_judge

        processed = workflow.process_job(job)

        # Low confidence should trigger flagging
        assert processed.segments[0].is_flagged is True
        assert "0.5" in processed.segments[0].flag_reason or "Uncertain" in processed.segments[0].flag_reason

    def test_process_job_with_progress_callback(self):
        """Should call progress callback during processing."""
        workflow = TranslationWorkflow()

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=[
                {"source": "A"},
                {"source": "B"},
                {"source": "C"},
            ],
        )

        progress_calls = []

        def callback(msg, current, total):
            progress_calls.append((msg, current, total))

        workflow.process_job(job, progress_callback=callback)

        assert len(progress_calls) == 3
        assert progress_calls[0][0] == "Translating segment 1"
        assert progress_calls[0][1] == 1
        assert progress_calls[0][2] == 3

    def test_approve_job(self):
        """Should mark job as approved."""
        workflow = TranslationWorkflow()

        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt",
            status="pending_approval",
        )

        approved = workflow.approve_job(job, approved_by="test_user")

        assert approved.status == "approved"
        assert approved.approved_by == "test_user"
        assert approved.approved_at is not None

    def test_reject_job(self):
        """Should mark job as rejected."""
        workflow = TranslationWorkflow()

        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt",
            status="pending_approval",
        )

        rejected = workflow.reject_job(job, reason="Quality issues")

        assert rejected.status == "rejected"

    def test_can_auto_approve_high_score(self):
        """Should auto-approve high-scoring jobs."""
        workflow = TranslationWorkflow(
            config=ReviewConfig(auto_approve_threshold=0.85),
        )

        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt",
            status="pending_approval",
            overall_score=0.90,
        )

        assert workflow.can_auto_approve(job) is True

    def test_can_auto_approve_low_score(self):
        """Should not auto-approve low-scoring jobs."""
        workflow = TranslationWorkflow(
            config=ReviewConfig(auto_approve_threshold=0.85),
        )

        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt",
            status="pending_approval",
            overall_score=0.70,
        )

        assert workflow.can_auto_approve(job) is False

    def test_can_auto_approve_blocking_mode_with_flags(self):
        """Blocking mode should not auto-approve with flags."""
        workflow = TranslationWorkflow(
            config=ReviewConfig(auto_approve_threshold=0.85),
        )

        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt",
            status="pending_approval",
            approval_mode="blocking",
            overall_score=0.95,
            flagged_count=1,
        )

        assert workflow.can_auto_approve(job) is False

    def test_export_job(self):
        """Should export job to CSV."""
        with tempfile.TemporaryDirectory() as tmpdir:
            workflow = TranslationWorkflow()
            workflow.config.csv_export_enabled = True

            job = TranslationJob(
                id="job1",
                source_file="in.txt",
                target_file="out.txt",
            )
            job.segments.append(
                TranslationJob(
                    id="", source_file="", target_file=""
                ).segments.__class__.__bases__[0]()
            )
            job.segments[0] = type("Seg", (), {
                "id": "s1", "source": "Hello", "target": "こんにちは",
                "judge_winner": "model_a", "judge_confidence": 0.95,
                "judge_reasoning": "", "is_flagged": False,
                "flag_reason": None, "model_a_output": "[Model A] Hello",
                "model_b_output": "[Model B] Hello", "glossary_terms": [],
                "context": {}
            })()

            filepath = workflow.export_job(job, output_dir=tmpdir)

            assert filepath is not None
            assert Path(filepath).exists()

    def test_export_job_disabled(self):
        """Should return None when export disabled."""
        workflow = TranslationWorkflow()
        workflow.config.csv_export_enabled = False

        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt",
        )

        filepath = workflow.export_job(job)

        assert filepath is None

    def test_process_and_export(self):
        """Should process job and export in one call."""
        with tempfile.TemporaryDirectory() as tmpdir:
            # Provide mock provider so translations happen
            mock_provider = Mock()
            mock_provider.generate.return_value = Mock(text="Translated text")

            translator = MultiModelTranslator(providers=[mock_provider])
            workflow = TranslationWorkflow(translator=translator)
            workflow.exporter = BilingualCSVExporter(output_dir=tmpdir)

            result = workflow.process_and_export(
                source_file="in.txt",
                target_file="out.txt",
                segments=[{"source": "Hello"}],
                project_type="routine",
            )

            assert "job" in result
            assert "csv_path" in result
            assert "can_auto_approve" in result
            assert "needs_review" in result

            job = result["job"]
            assert job.status in ["pending_approval", "approved"]
            assert len(job.segments) == 1
            assert job.segments[0].target

            if result["csv_path"]:
                assert Path(result["csv_path"]).exists()

    def test_get_winner_text_model_a(self):
        """Should extract model_a translation."""
        workflow = TranslationWorkflow()

        candidates = [
            TranslationCandidate("model_a", "Translation A"),
            TranslationCandidate("model_b", "Translation B"),
        ]

        result = workflow._get_winner_text("model_a", candidates)
        assert result == "Translation A"

    def test_get_winner_text_model_b(self):
        """Should extract model_b translation."""
        workflow = TranslationWorkflow()

        candidates = [
            TranslationCandidate("model_a", "Translation A"),
            TranslationCandidate("model_b", "Translation B"),
        ]

        result = workflow._get_winner_text("model_b", candidates)
        assert result == "Translation B"

    def test_get_winner_text_fallback(self):
        """Should fall back to first candidate for unknown winner."""
        workflow = TranslationWorkflow()

        candidates = [
            TranslationCandidate("model_a", "Translation A"),
        ]

        result = workflow._get_winner_text("unknown", candidates)
        assert result == "Translation A"


class TestReviewWorkflowBuilder:
    """Test workflow builder pattern."""

    def test_builder_default(self):
        """Should build workflow with defaults."""
        builder = ReviewWorkflowBuilder()
        workflow = builder.build()

        assert isinstance(workflow, TranslationWorkflow)
        assert workflow.config.auto_approve_threshold == 0.85

    def test_builder_with_translator(self):
        """Should set custom translator."""
        custom_translator = MultiModelTranslator(parallel=False)
        builder = ReviewWorkflowBuilder()
        workflow = builder.with_translator(custom_translator).build()

        assert workflow.translator is custom_translator

    def test_builder_with_judge(self):
        """Should set custom judge."""
        custom_judge = TranslationJudge(enabled=True)
        builder = ReviewWorkflowBuilder()
        workflow = builder.with_judge(custom_judge).build()

        assert workflow.judge is custom_judge

    def test_builder_with_flagger(self):
        """Should set custom flagger."""
        custom_flagger = FlaggingEngine(block_threshold=0.9)
        builder = ReviewWorkflowBuilder()
        workflow = builder.with_flagger(custom_flagger).build()

        assert workflow.flagger is custom_flagger

    def test_builder_with_exporter(self):
        """Should set custom exporter."""
        custom_exporter = BilingualCSVExporter(output_dir="/tmp/custom")
        builder = ReviewWorkflowBuilder()
        workflow = builder.with_exporter(custom_exporter).build()

        assert workflow.exporter is custom_exporter

    def test_builder_with_config(self):
        """Should set custom config."""
        custom_config = ReviewConfig(auto_approve_threshold=0.95)
        builder = ReviewWorkflowBuilder()
        workflow = builder.with_config(custom_config).build()

        assert workflow.config is custom_config

    def test_builder_for_project_critical(self):
        """Should configure for critical project."""
        builder = ReviewWorkflowBuilder()
        workflow = builder.for_project("critical").build()

        assert workflow.config.auto_approve_threshold == 0.95
        assert workflow.config.random_sample_rate == 0.10

    def test_builder_for_project_routine(self):
        """Should configure for routine project."""
        builder = ReviewWorkflowBuilder()
        workflow = builder.for_project("routine").build()

        assert workflow.config.auto_approve_threshold == 0.85
        assert workflow.config.random_sample_rate == 0.02

    def test_builder_chaining(self):
        """Should support method chaining."""
        custom_translator = MultiModelTranslator(parallel=False)
        custom_config = ReviewConfig(auto_approve_threshold=0.99)

        builder = ReviewWorkflowBuilder()
        workflow = (
            builder
            .with_translator(custom_translator)
            .for_project("critical")  # This sets config
            .with_config(custom_config)  # This overrides the project config
            .build()
        )

        assert workflow.translator is custom_translator
        assert workflow.config.auto_approve_threshold == 0.99  # Custom config applied last


class TestCreateWorkflow:
    """Test workflow factory function."""

    def test_create_workflow_default(self):
        """Should create routine workflow by default."""
        workflow = create_workflow()

        assert isinstance(workflow, TranslationWorkflow)
        assert workflow.config.auto_approve_threshold == 0.85

    def test_create_workflow_critical(self):
        """Should create critical workflow."""
        workflow = create_workflow(project_type="critical")

        assert workflow.config.auto_approve_threshold == 0.95
        assert workflow.config.random_sample_rate == 0.10

    def test_create_workflow_with_output_dir(self):
        """Should create workflow with CSV output dir."""
        with tempfile.TemporaryDirectory() as tmpdir:
            workflow = create_workflow(csv_output_dir=tmpdir)

            assert workflow.exporter.output_dir == Path(tmpdir)


class TestWorkflowIntegration:
    """Integration tests for complete workflow."""

    def test_full_workflow_end_to_end(self):
        """Should process complete workflow from creation to export."""
        with tempfile.TemporaryDirectory() as tmpdir:
            # Provide mock provider for real translations
            mock_provider = Mock()
            mock_provider.generate.return_value = Mock(text="Translated text")

            translator = MultiModelTranslator(providers=[mock_provider])
            workflow = TranslationWorkflow(
                translator=translator,
                config=ReviewConfig.for_project_type("routine"),
                exporter=BilingualCSVExporter(output_dir=tmpdir),
            )

            # Create job
            segments = [
                {"source": "こんにちは", "context": {"position": "title"}},
                {"source": "世界", "context": {"position": "body"}},
            ]

            job = workflow.create_job(
                source_file="source.txt",
                target_file="target.txt",
                segments=segments,
            )

            assert job.segment_count == 2

            # Process
            processed = workflow.process_job(job)

            assert processed.status == "pending_approval"
            assert processed.segments[0].target  # Has translation
            assert processed.segments[1].target  # Has translation
            assert processed.completed_at is not None

            # Export
            csv_path = workflow.export_job(processed)

            assert csv_path is not None
            assert Path(csv_path).exists()

            # Verify CSV content
            with open(csv_path, "r", encoding="utf-8-sig") as f:
                content = f.read()
                assert "こんにちは" in content
                assert "世界" in content

    def test_auto_approve_high_quality_job(self):
        """Should auto-approve high quality job."""
        from review.models import JudgeResult, TranslationCandidate

        workflow = create_workflow(project_type="routine")

        # Use longer source text to avoid length ratio issues with stub
        segments = [{"source": "This is a longer text segment"}] * 10

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=segments,
        )

        # Mock high confidence judge results
        def high_confidence_judge(segment_id, source, candidates, context=None):
            return JudgeResult(
                segment_id=segment_id,
                winner="model_a",
                confidence=0.98,
                reasoning="Excellent",
                concerns=[],
            )

        # Mock translator to return same length output
        def mock_translate(source, glossary_terms=None, context=None):
            return [
                TranslationCandidate("model_a", source),  # Same length
                TranslationCandidate("model_b", source),
            ]

        workflow.judge.judge = high_confidence_judge
        workflow.translator.translate = mock_translate

        processed = workflow.process_job(job)

        # Should have high score
        assert processed.overall_score >= 0.85

        # Should be auto-approvable
        assert workflow.can_auto_approve(processed) is True

    def test_blocking_job_requires_review(self):
        """Critical job with flags should require review."""
        workflow = create_workflow(project_type="critical")

        job = TranslationJob(
            id="job1",
            source_file="in.txt",
            target_file="out.txt",
            approval_mode="blocking",
            flagged_count=1,
            status="pending_approval",
        )

        assert workflow.can_auto_approve(job) is False


class TestStyleCheckerInWorkflow:
    """Tests for style checker integration in the workflow."""

    def test_style_checker_disabled_no_extra_flags(self):
        """When style_checker is None, no style-related flagging occurs."""
        mock_provider = Mock()
        mock_provider.generate.return_value = Mock(
            text="The colour costs 1,000 dollars."
        )

        translator = MultiModelTranslator(providers=[mock_provider])
        workflow = TranslationWorkflow(
            translator=translator,
            style_checker=None,  # Disabled
        )

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=[{"source": "テスト"}],
        )
        processed = workflow.process_job(job)

        seg = processed.segments[0]
        assert seg.style_issues == []
        # Should not be flagged for style (may be flagged for other reasons)

    def test_style_checker_enabled_flags_violations(self):
        """When style_checker is enabled, Gengo violations flag segments."""
        from audit.style_checker import StyleChecker

        mock_provider = Mock()
        # Return text with UK spelling + wrong currency format
        mock_provider.generate.return_value = Mock(
            text="The colour costs 1,000 dollars at 3:00 PM."
        )

        translator = MultiModelTranslator(providers=[mock_provider])
        checker = StyleChecker(gengo_rules_enabled=True)

        # Use a high-confidence mock judge so the segment isn't flagged
        # by confidence alone — only style issues should flag it
        from review.models import JudgeResult

        mock_judge = TranslationJudge(enabled=False)  # Returns model_a, conf=1.0

        workflow = TranslationWorkflow(
            translator=translator,
            judge=mock_judge,
            style_checker=checker,
            flagger=FlaggingEngine(random_sample_rate=0.0),
        )

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=[{"source": "テスト文"}],
        )
        processed = workflow.process_job(job)

        seg = processed.segments[0]
        assert len(seg.style_issues) > 0

        # Should be flagged (possibly by flagger or style checker)
        assert seg.is_flagged is True

        # Style issues should contain the expected categories regardless
        # of whether the flagger or style checker set the flag first
        issue_categories = {i["category"] for i in seg.style_issues}
        assert "uk_spelling" in issue_categories
        assert "currency_format" in issue_categories
        assert "time_format" in issue_categories

    def test_style_checker_clean_text_no_flag(self):
        """Clean Gengo-compliant text should not get style-flagged."""
        from audit.style_checker import StyleChecker

        mock_provider = Mock()
        # Return clean, Gengo-compliant text
        mock_provider.generate.return_value = Mock(
            text="The color costs US$1,000 at 3:00 p.m."
        )

        translator = MultiModelTranslator(providers=[mock_provider])
        checker = StyleChecker(gengo_rules_enabled=True)
        mock_judge = TranslationJudge(enabled=False)

        workflow = TranslationWorkflow(
            translator=translator,
            judge=mock_judge,
            style_checker=checker,
            flagger=FlaggingEngine(random_sample_rate=0.0),
        )

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=[{"source": "テスト文"}],
        )
        processed = workflow.process_job(job)

        seg = processed.segments[0]
        gengo_categories = {"uk_spelling", "currency_format", "time_format", "date_format", "oxford_comma"}
        gengo_issues = [i for i in seg.style_issues if i.get("category") in gengo_categories]
        assert len(gengo_issues) == 0

    def test_metrics_populated_after_processing(self):
        """Workflow should populate last_metrics after process_job."""
        from audit.style_checker import StyleChecker

        mock_provider = Mock()
        mock_provider.generate.return_value = Mock(
            text="The colour is grey."
        )

        translator = MultiModelTranslator(providers=[mock_provider])
        checker = StyleChecker(gengo_rules_enabled=True)
        mock_judge = TranslationJudge(enabled=False)

        workflow = TranslationWorkflow(
            translator=translator,
            judge=mock_judge,
            style_checker=checker,
            flagger=FlaggingEngine(random_sample_rate=0.0),
        )

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=[{"source": "テスト"}, {"source": "テスト2"}],
        )
        workflow.process_job(job)

        m = workflow.last_metrics
        assert m is not None
        assert m.job_id == job.id
        assert m.segment_count == 2
        assert m.style_guide_enabled is True
        assert m.style_violation_count > 0
        assert "uk_spelling" in m.style_violations_by_category
        assert m.processing_duration_ms is not None
        assert m.processing_duration_ms >= 0

        # Check JSON serialization
        d = m.to_dict()
        assert d["job_id"] == job.id
        assert d["style_violation_rate"] > 0.0

    def test_metrics_without_style_checker(self):
        """Metrics should still work when style_checker is disabled."""
        mock_provider = Mock()
        mock_provider.generate.return_value = Mock(text="Hello world")

        translator = MultiModelTranslator(providers=[mock_provider])
        mock_judge = TranslationJudge(enabled=False)

        workflow = TranslationWorkflow(
            translator=translator,
            judge=mock_judge,
            flagger=FlaggingEngine(random_sample_rate=0.0),
        )

        job = workflow.create_job(
            source_file="in.txt",
            target_file="out.txt",
            segments=[{"source": "テスト"}],
        )
        workflow.process_job(job)

        m = workflow.last_metrics
        assert m is not None
        assert m.style_guide_enabled is False
        assert m.style_violation_count == 0

    def test_style_issues_in_csv_export(self):
        """Style issues should appear in CSV export."""
        from audit.style_checker import StyleChecker

        with tempfile.TemporaryDirectory() as tmpdir:
            mock_provider = Mock()
            mock_provider.generate.return_value = Mock(
                text="The colour is nice."
            )

            translator = MultiModelTranslator(providers=[mock_provider])
            checker = StyleChecker(gengo_rules_enabled=True)
            mock_judge = TranslationJudge(enabled=False)

            workflow = TranslationWorkflow(
                translator=translator,
                judge=mock_judge,
                style_checker=checker,
                exporter=BilingualCSVExporter(output_dir=tmpdir),
                flagger=FlaggingEngine(random_sample_rate=0.0),
            )

            job = workflow.create_job(
                source_file="in.txt",
                target_file="out.txt",
                segments=[{"source": "テスト"}],
            )
            processed = workflow.process_job(job)

            # Verify style issues are on the segment
            assert len(processed.segments[0].style_issues) > 0

            csv_path = workflow.export_job(processed)

            assert csv_path is not None
            content = Path(csv_path).read_text(encoding="utf-8-sig")
            # style_issues column should be in header and contain issue data
            assert "style_issues" in content
            assert "uk_spelling" in content
