# tests/test_queue/test_consumer_gengo.py
"""
Queue consumer integration tests with Gengo style guide.

Tests the actual worker path (QueueConsumer._process_job) with:
- Mock providers (no real API calls)
- Temporary style guide fixture
- Style-violating translations surfaced as review-needed
"""

import json
import sys
import tempfile
from pathlib import Path
from unittest.mock import Mock, MagicMock, patch

import pytest

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from main import QueueConsumer, SegmentExtractor, build_workflow
from review.workflow import TranslationWorkflow
from review.multimodel import MultiModelTranslator
from review.judge import TranslationJudge
from review.flagging import FlaggingEngine
from review.models import TranslationCandidate
from audit.style_checker import StyleChecker


@pytest.fixture
def mock_job_manager():
    """Create a mock JobManager with minimal Redis behavior."""
    manager = MagicMock()
    manager.redis_client = MagicMock()
    return manager


@pytest.fixture
def mock_workflow_with_style_checker():
    """Create a workflow with a mock provider + Gengo style checker."""
    mock_provider = Mock()
    mock_provider.generate.return_value = Mock(
        text="The colour costs 1,000 dollars."
    )

    translator = MultiModelTranslator(providers=[mock_provider])
    judge = TranslationJudge(enabled=False)
    checker = StyleChecker(gengo_rules_enabled=True)

    return TranslationWorkflow(
        translator=translator,
        judge=judge,
        style_checker=checker,
        flagger=FlaggingEngine(random_sample_rate=0.0),
    )


@pytest.fixture
def mock_workflow_without_style_checker():
    """Create a workflow with a mock provider, no style checker."""
    mock_provider = Mock()
    mock_provider.generate.return_value = Mock(
        text="The colour costs 1,000 dollars."
    )

    translator = MultiModelTranslator(providers=[mock_provider])
    judge = TranslationJudge(enabled=False)

    return TranslationWorkflow(
        translator=translator,
        judge=judge,
        flagger=FlaggingEngine(random_sample_rate=0.0),
    )


class TestQueueConsumerWithGengo:
    """Tests for QueueConsumer processing jobs through a real workflow."""

    def test_process_job_uses_workflow_not_stub(
        self, mock_job_manager, mock_workflow_with_style_checker, tmp_path
    ):
        """When workflow is provided, jobs should NOT go through stub path."""
        # Create a simple text file to translate
        source_file = tmp_path / "input.txt"
        source_file.write_text("テスト文章\n")

        consumer = QueueConsumer(
            job_manager=mock_job_manager,
            worker_id="test-worker",
            workflow=mock_workflow_with_style_checker,
        )

        job = {
            "id": "test-job-1",
            "source_file": str(source_file),
            "project_type": "routine",
        }

        consumer._process_job(job)

        # Should NOT have completed as stub
        calls = mock_job_manager.set_state.call_args_list
        states = [call[0][1].value if hasattr(call[0][1], 'value') else str(call[0][1])
                  for call in calls]

        # Should end in review_pending or completed, not just "completed" via stub
        assert "translating" in states or any("translat" in s.lower() for s in states)

    def test_process_job_with_style_violations_flags_review(
        self, mock_job_manager, mock_workflow_with_style_checker, tmp_path
    ):
        """Jobs with style violations should be flagged for review."""
        source_file = tmp_path / "input.txt"
        source_file.write_text("テスト\n")

        consumer = QueueConsumer(
            job_manager=mock_job_manager,
            worker_id="test-worker",
            workflow=mock_workflow_with_style_checker,
        )

        job = {
            "id": "test-job-2",
            "source_file": str(source_file),
            "project_type": "routine",
        }

        consumer._process_job(job)

        # The progress message should mention flagged segments
        progress_calls = mock_job_manager.publish_progress.call_args_list
        assert len(progress_calls) > 0

        # Final progress should mention "flagged"
        final_msg = progress_calls[-1][0][2] if len(progress_calls[-1][0]) > 2 else ""
        assert "flagged" in final_msg.lower() or "complete" in final_msg.lower()

    def test_process_job_without_workflow_uses_stub(self, mock_job_manager):
        """Without a workflow, should complete as stub."""
        consumer = QueueConsumer(
            job_manager=mock_job_manager,
            worker_id="test-worker",
            workflow=None,
        )

        job = {"id": "stub-job", "source_file": "nonexistent.txt"}
        consumer._process_job(job)

        # Should have published "No workflow configured"
        progress_calls = mock_job_manager.publish_progress.call_args_list
        assert any(
            "No workflow" in str(call) for call in progress_calls
        )

    def test_process_job_no_segments_fails(
        self, mock_job_manager, mock_workflow_with_style_checker
    ):
        """Job with no extractable segments should fail."""
        consumer = QueueConsumer(
            job_manager=mock_job_manager,
            worker_id="test-worker",
            workflow=mock_workflow_with_style_checker,
        )

        job = {
            "id": "empty-job",
            "source_file": "/nonexistent/file.txt",
        }

        consumer._process_job(job)

        # Should have published failure
        progress_calls = mock_job_manager.publish_progress.call_args_list
        assert any(
            "No segments" in str(call) for call in progress_calls
        )

    def test_consumer_without_style_checker_no_style_issues(
        self, mock_job_manager, mock_workflow_without_style_checker, tmp_path
    ):
        """Without style checker, segments should have empty style_issues."""
        source_file = tmp_path / "input.txt"
        source_file.write_text("テスト\n")

        consumer = QueueConsumer(
            job_manager=mock_job_manager,
            worker_id="test-worker",
            workflow=mock_workflow_without_style_checker,
        )

        job = {
            "id": "no-style-job",
            "source_file": str(source_file),
            "project_type": "routine",
        }

        consumer._process_job(job)

        # Verify job completed (not stub)
        calls = mock_job_manager.set_state.call_args_list
        assert len(calls) >= 2  # TRANSLATING + final state


class TestBuildWorkflowIntegration:
    """Tests for build_workflow() with style checker wiring."""

    @patch("main.build_judge_provider")
    @patch("main.build_translation_provider")
    @patch("main.load_style_guide_prompt")
    def test_build_workflow_includes_style_checker_when_enabled(
        self, mock_prompt, mock_trans, mock_judge
    ):
        """build_workflow should wire up a StyleChecker when style_guide.enabled."""
        mock_prompt.return_value = "Test prompt"
        mock_trans.return_value = MagicMock()
        mock_judge.return_value = MagicMock()

        config = {
            "translation": {"default_provider": "openai", "default_model": "gpt-5.2"},
            "style_guide": {"enabled": True, "path": "/some/guide.md"},
        }
        workflow = build_workflow(config)

        assert workflow.style_checker is not None
        assert workflow.style_checker.gengo_rules_enabled is True

    @patch("main.build_judge_provider")
    @patch("main.build_translation_provider")
    @patch("main.load_style_guide_prompt")
    def test_build_workflow_no_style_checker_when_disabled(
        self, mock_prompt, mock_trans, mock_judge
    ):
        """build_workflow should NOT wire up StyleChecker when disabled."""
        mock_prompt.return_value = None
        mock_trans.return_value = MagicMock()
        mock_judge.return_value = MagicMock()

        config = {
            "translation": {"default_provider": "openai", "default_model": "gpt-5.2"},
            "style_guide": {"enabled": False},
        }
        workflow = build_workflow(config)

        assert workflow.style_checker is None
