# review/workflow.py
"""
Translation review workflow orchestrator.

Coordinates the complete bilingual translation review workflow:
1. Multi-model translation generates candidates
2. Judge model evaluates and selects winner
3. Flagging engine identifies segments needing review
4. Optional style checker validates against Gengo rules
5. CSV exporter creates audit trail
6. Job status tracking and approval logic
7. Structured metrics for observability

MVP: Stub components coordinated with real orchestration logic.
Full: Real LLM integration for translation and judge.
"""

import logging
import uuid
from datetime import datetime
from pathlib import Path
from typing import List, Optional, Dict, Any, Callable

from .models import (
    TranslationJob,
    TranslationSegment,
    TranslationCandidate,
    JudgeResult,
    ReviewConfig,
)
from .multimodel import MultiModelTranslator
from .judge import TranslationJudge
from .flagging import FlaggingEngine
from .exporter import BilingualCSVExporter
from .metrics import JobMetrics

logger = logging.getLogger(__name__)


class TranslationWorkflow:
    """Orchestrates the complete translation review workflow.

    Manages the end-to-end process:
    - Creates and tracks translation jobs
    - Coordinates multi-model translation
    - Runs judge evaluation
    - Applies flagging logic
    - Exports bilingual CSV
    """

    def __init__(
        self,
        translator: Optional[MultiModelTranslator] = None,
        judge: Optional[TranslationJudge] = None,
        flagger: Optional[FlaggingEngine] = None,
        exporter: Optional[BilingualCSVExporter] = None,
        config: Optional[ReviewConfig] = None,
        style_checker: Optional[Any] = None,
    ):
        """Initialize the workflow orchestrator.

        Args:
            translator: Multi-model translator instance
            judge: Judge model instance
            flagger: Flagging engine instance
            exporter: CSV exporter instance
            config: Review configuration
            style_checker: Optional StyleChecker for Gengo rule validation
        """
        self.translator = translator or MultiModelTranslator()
        self.judge = judge or TranslationJudge()
        self.flagger = flagger or FlaggingEngine()
        self.exporter = exporter or BilingualCSVExporter()
        self.config = config or ReviewConfig()
        self.style_checker = style_checker
        self.last_metrics: Optional[JobMetrics] = None

    def create_job(
        self,
        source_file: str,
        target_file: str,
        project_type: str = "routine",
        segments: Optional[List[Dict[str, Any]]] = None,
    ) -> TranslationJob:
        """Create a new translation job.

        Args:
            source_file: Path to source file
            target_file: Path for output file
            project_type: "critical" or "routine" (affects thresholds)
            segments: Optional pre-parsed segments

        Returns:
            New TranslationJob instance
        """
        job_id = str(uuid.uuid4())[:8]

        # Get project-specific config
        job_config = ReviewConfig.for_project_type(project_type)

        job = TranslationJob(
            id=job_id,
            source_file=source_file,
            target_file=target_file,
            project_type=project_type,
            approval_mode="blocking" if project_type == "critical" else "async",
        )

        # Add initial segments if provided
        if segments:
            for seg_data in segments:
                segment = TranslationSegment(
                    id=seg_data.get("id", f"seg_{len(job.segments) + 1}"),
                    job_id=job_id,
                    source=seg_data.get("source", ""),
                    target="",  # Will be filled during processing
                    context=seg_data.get("context", {}),
                )
                job.segments.append(segment)

        job.update_metrics()
        logger.info(f"[WORKFLOW] Created job {job_id} with {job.segment_count} segments")
        return job

    def process_job(
        self,
        job: TranslationJob,
        progress_callback: Optional[Callable[[str, int, int], None]] = None,
    ) -> TranslationJob:
        """Process a translation job through the full workflow.

        Args:
            job: The translation job to process
            progress_callback: Optional callback(message, current, total)

        Returns:
            The processed job with translations and flags
        """
        total = len(job.segments)
        logger.info(f"[WORKFLOW] Processing job {job.id} with {total} segments")

        # Initialize metrics
        metrics = JobMetrics(
            job_id=job.id,
            segment_count=total,
            style_guide_enabled=self.style_checker is not None,
        )
        metrics.start_timer()

        for idx, segment in enumerate(job.segments):
            if progress_callback:
                progress_callback(f"Translating segment {idx + 1}", idx + 1, total)

            # Step 1: Multi-model translation
            candidates = self.translator.translate(
                segment.source,
                glossary_terms=segment.glossary_terms,
                context=segment.context,
            )

            # Store model outputs for audit trail
            if len(candidates) > 0:
                segment.model_a_output = candidates[0].text
            if len(candidates) > 1:
                segment.model_b_output = candidates[1].text

            # Step 2: Judge evaluation
            judge_result = self.judge.judge(
                segment_id=segment.id,
                source=segment.source,
                candidates=candidates,
                context=segment.context,
            )

            # Apply judge results to segment
            segment.judge_winner = judge_result.winner
            segment.judge_confidence = judge_result.confidence
            segment.judge_reasoning = judge_result.reasoning

            # Step 3: Set final target (winner's translation)
            segment.target = self._get_winner_text(judge_result.winner, candidates)

            # Step 4: Flag for review if needed
            self.flagger.flag_segment(segment, judge_result, self.config)

            # Step 5: Style check (if enabled)
            if self.style_checker and segment.target:
                style_issues = self.style_checker.check(segment.target, segment.source)
                if style_issues:
                    segment.style_issues = [issue.to_dict() for issue in style_issues]
                    metrics.record_style_issues(segment.style_issues)

                    # Flag for review if not already flagged
                    if not segment.is_flagged:
                        # Summarize: pick the highest-severity issue
                        top_issue = max(
                            style_issues,
                            key=lambda i: {"error": 2, "warning": 1, "info": 0}.get(
                                i.severity, 0
                            ),
                        )
                        segment.is_flagged = True
                        segment.flag_reason = (
                            f"Style: {top_issue.category} - {top_issue.message}"
                        )

            # Track flag in metrics
            if segment.is_flagged:
                metrics.record_flag(segment.flag_reason or "unknown")

            logger.debug(
                f"[WORKFLOW] {segment.id}: winner={judge_result.winner}, "
                f"conf={judge_result.confidence:.2f}, flagged={segment.is_flagged}"
            )

        # Finalize job
        job.update_metrics()
        job.status = "pending_approval"
        job.completed_at = datetime.now()

        # Finalize metrics
        metrics.stop_timer()
        metrics.overall_score = job.overall_score
        metrics.flagged_count = job.flagged_count
        self.last_metrics = metrics

        logger.info(
            f"[WORKFLOW] Job {job.id} complete: "
            f"score={job.overall_score:.2f}, flagged={job.flagged_count}/{total}, "
            f"style_violations={metrics.style_violation_count}"
        )
        logger.info(f"[METRICS] {metrics.to_json()}")

        return job

    def _get_winner_text(
        self,
        winner: str,
        candidates: List[TranslationCandidate],
    ) -> str:
        """Extract the winning translation text.

        Args:
            winner: Which model won ("model_a", "model_b", etc.)
            candidates: List of translation candidates

        Returns:
            The winning translation text
        """
        if winner == "model_a" and len(candidates) > 0:
            return candidates[0].text
        elif winner == "model_b" and len(candidates) > 1:
            return candidates[1].text
        elif winner.startswith("model_"):
            # Extract index for model_c, model_d, etc.
            try:
                idx = int(winner.split("_")[1]) - 1  # model_a -> 0, model_b -> 1
                if 0 <= idx < len(candidates):
                    return candidates[idx].text
            except (ValueError, IndexError):
                pass
        return candidates[0].text if candidates else ""

    def export_job(
        self,
        job: TranslationJob,
        output_dir: Optional[str] = None,
    ) -> Optional[str]:
        """Export a job to bilingual CSV.

        Args:
            job: The translation job to export
            output_dir: Optional custom output directory

        Returns:
            Path to exported CSV file, or None if export disabled
        """
        if not self.config.csv_export_enabled:
            logger.debug(f"[WORKFLOW] CSV export disabled, skipping job {job.id}")
            return None

        if output_dir:
            exporter = BilingualCSVExporter(output_dir=output_dir)
        else:
            exporter = self.exporter

        filepath = exporter.export_job(job)
        logger.info(f"[WORKFLOW] Exported job {job.id} to {filepath}")
        return filepath

    def approve_job(
        self,
        job: TranslationJob,
        approved_by: str,
    ) -> TranslationJob:
        """Mark a job as approved.

        Args:
            job: The translation job to approve
            approved_by: User or system that approved

        Returns:
            The updated job
        """
        if job.status != "pending_approval":
            logger.warning(
                f"[WORKFLOW] Job {job.id} status is {job.status}, "
                f"not pending_approval"
            )

        job.status = "approved"
        job.approved_at = datetime.now()
        job.approved_by = approved_by

        logger.info(f"[WORKFLOW] Job {job.id} approved by {approved_by}")
        return job

    def reject_job(
        self,
        job: TranslationJob,
        reason: str = "",
    ) -> TranslationJob:
        """Mark a job as rejected.

        Args:
            job: The translation job to reject
            reason: Optional reason for rejection

        Returns:
            The updated job
        """
        job.status = "rejected"
        logger.info(f"[WORKFLOW] Job {job.id} rejected: {reason}")
        return job

    def can_auto_approve(self, job: TranslationJob) -> bool:
        """Check if a job can be auto-approved.

        Args:
            job: The translation job to check

        Returns:
            True if job meets auto-approve criteria
        """
        if job.status != "pending_approval":
            return False

        return job.can_auto_approve(threshold=self.config.auto_approve_threshold)

    def process_and_export(
        self,
        source_file: str,
        target_file: str,
        segments: List[Dict[str, Any]],
        project_type: str = "routine",
        progress_callback: Optional[Callable[[str, int, int], None]] = None,
    ) -> Dict[str, Any]:
        """Convenience method to process and export in one call.

        Args:
            source_file: Path to source file
            target_file: Path for output file
            segments: List of segment dicts with "source" and optional "context"
            project_type: "critical" or "routine"
            progress_callback: Optional progress callback

        Returns:
            Dict with job, csv_path, and auto_approve status
        """
        # Create job
        job = self.create_job(source_file, target_file, project_type, segments)

        # Process translation
        job = self.process_job(job, progress_callback)

        # Export CSV
        csv_path = self.export_job(job)

        # Check auto-approve
        can_auto_approve = self.can_auto_approve(job)
        if can_auto_approve:
            job = self.approve_job(approved_by="auto-approve")

        return {
            "job": job,
            "csv_path": csv_path,
            "can_auto_approve": can_auto_approve,
            "needs_review": not can_auto_approve,
        }


class ReviewWorkflowBuilder:
    """Builder pattern for creating configured workflow instances."""

    def __init__(self):
        """Initialize builder with defaults."""
        self._translator = None
        self._judge = None
        self._flagger = None
        self._exporter = None
        self._config = None
        self._style_checker = None

    def with_translator(self, translator: MultiModelTranslator) -> "ReviewWorkflowBuilder":
        """Set custom translator."""
        self._translator = translator
        return self

    def with_judge(self, judge: TranslationJudge) -> "ReviewWorkflowBuilder":
        """Set custom judge."""
        self._judge = judge
        return self

    def with_flagger(self, flagger: FlaggingEngine) -> "ReviewWorkflowBuilder":
        """Set custom flagger."""
        self._flagger = flagger
        return self

    def with_exporter(self, exporter: BilingualCSVExporter) -> "ReviewWorkflowBuilder":
        """Set custom exporter."""
        self._exporter = exporter
        return self

    def with_config(self, config: ReviewConfig) -> "ReviewWorkflowBuilder":
        """Set custom config."""
        self._config = config
        return self

    def with_style_checker(self, style_checker) -> "ReviewWorkflowBuilder":
        """Set style checker for Gengo rule validation."""
        self._style_checker = style_checker
        return self

    def for_project(self, project_type: str) -> "ReviewWorkflowBuilder":
        """Configure for specific project type."""
        self._config = ReviewConfig.for_project_type(project_type)
        return self

    def build(self) -> TranslationWorkflow:
        """Build the configured workflow."""
        return TranslationWorkflow(
            translator=self._translator,
            judge=self._judge,
            flagger=self._flagger,
            exporter=self._exporter,
            config=self._config,
            style_checker=self._style_checker,
        )


def create_workflow(
    project_type: str = "routine",
    csv_output_dir: Optional[str] = None,
) -> TranslationWorkflow:
    """Factory function to create a configured workflow.

    Args:
        project_type: "critical" or "routine"
        csv_output_dir: Optional directory for CSV exports

    Returns:
        Configured TranslationWorkflow instance
    """
    config = ReviewConfig.for_project_type(project_type)
    exporter = BilingualCSVExporter(output_dir=csv_output_dir) if csv_output_dir else None

    return TranslationWorkflow(config=config, exporter=exporter)
