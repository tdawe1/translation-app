# review/exporter.py
"""
Bilingual CSV export for translation audit trail.

Exports translation jobs with full metadata including:
- Source and target text
- Judge decisions and reasoning
- Alternative translations from all models
- Glossary terms used
- Flagging information
"""

import csv
import logging
import os
import tempfile
from datetime import datetime
from pathlib import Path
from typing import List, Optional
from .models import TranslationJob, TranslationSegment

logger = logging.getLogger(__name__)


class BilingualCSVExporter:
    """Exports translation jobs to bilingual CSV format.

    The CSV serves as:
    - Audit trail of all translations
    - Fine-tuning data source
    - Human review backup (though web UI is primary)
    """

    # CSV column headers
    HEADERS = [
        "segment_id",
        "source",
        "target",
        "judge_winner",
        "judge_confidence",
        "judge_reasoning",
        "is_flagged",
        "flag_reason",
        "model_a_output",
        "model_b_output",
        "glossary_terms",
        "context"
    ]

    def __init__(
        self,
        output_dir: Optional[str] = None,
        encoding: str = "utf-8-sig",
        include_judge_reasoning: bool = True,
        include_alternatives: bool = True
    ):
        """Initialize the CSV exporter.

        Args:
            output_dir: Directory to write CSV files (default: temp directory)
            encoding: File encoding (utf-8-sig for Excel compatibility)
            include_judge_reasoning: Include judge reasoning in output
            include_alternatives: Include alternative model outputs
        """
        if output_dir is None:
            output_dir = tempfile.gettempdir()
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)
        self.encoding = encoding
        self.include_judge_reasoning = include_judge_reasoning
        self.include_alternatives = include_alternatives

    def export_job(
        self,
        job: TranslationJob,
        filename: Optional[str] = None
    ) -> str:
        """Export a translation job to CSV.

        Args:
            job: The translation job to export
            filename: Optional custom filename (default: {job_id}_{timestamp}.csv)

        Returns:
            Path to the exported CSV file
        """
        if filename is None:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"{job.id}_{timestamp}.csv"

        filepath = self.output_dir / filename

        with open(filepath, "w", encoding=self.encoding, newline="") as f:
            writer = csv.DictWriter(f, fieldnames=self.HEADERS)
            writer.writeheader()

            for segment in job.segments:
                row = self._segment_to_row(segment)
                writer.writerow(row)

        logger.info(f"[EXPORT] Job {job.id} exported to {filepath}")
        return str(filepath)

    def _segment_to_row(self, segment: TranslationSegment) -> dict:
        """Convert a segment to a CSV row dict.

        Args:
            segment: The translation segment

        Returns:
            Dict mapping CSV headers to values
        """
        return {
            "segment_id": segment.id,
            "source": segment.source,
            "target": segment.target,
            "judge_winner": segment.judge_winner,
            "judge_confidence": str(segment.judge_confidence),
            "judge_reasoning": segment.judge_reasoning if self.include_judge_reasoning else "",
            "is_flagged": "1" if segment.is_flagged else "0",
            "flag_reason": segment.flag_reason or "",
            "model_a_output": segment.model_a_output if self.include_alternatives else "",
            "model_b_output": segment.model_b_output if self.include_alternatives else "",
            "glossary_terms": ",".join(segment.glossary_terms),
            "context": str(segment.context) if segment.context else ""
        }

    def export_batch(
        self,
        jobs: List[TranslationJob],
        combined: bool = False
    ) -> List[str]:
        """Export multiple jobs to CSV.

        Args:
            jobs: List of translation jobs to export
            combined: If True, combine all jobs into one CSV file

        Returns:
            List of paths to exported CSV files
        """
        if combined and jobs:
            return [self._export_combined(jobs)]
        else:
            paths = []
            for job in jobs:
                path = self.export_job(job)
                paths.append(path)
            return paths

    def _export_combined(self, jobs: List[TranslationJob]) -> str:
        """Export multiple jobs to a single combined CSV file.

        Args:
            jobs: List of translation jobs to export

        Returns:
            Path to the combined CSV file
        """
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        filename = f"combined_{timestamp}.csv"
        filepath = self.output_dir / filename

        with open(filepath, "w", encoding=self.encoding, newline="") as f:
            writer = csv.DictWriter(f, fieldnames=self.HEADERS)
            writer.writeheader()

            for job in jobs:
                for segment in job.segments:
                    # Add job_id as context
                    segment.context = {**segment.context, "_job_id": job.id}
                    row = self._segment_to_row(segment)
                    writer.writerow(row)

        logger.info(f"[EXPORT] Combined export ({len(jobs)} jobs) to {filepath}")
        return str(filepath)

    def get_export_summary(self, job: TranslationJob) -> dict:
        """Get a summary of export statistics for a job.

        Args:
            job: The translation job

        Returns:
            Dict with export statistics
        """
        return {
            "job_id": job.id,
            "segment_count": job.segment_count,
            "flagged_count": job.flagged_count,
            "overall_score": job.overall_score,
            "status": job.status,
            "judge_resolutions": job.judge_resolutions,
            "estimated_csv_size_kb": job.segment_count * 0.5  # Rough estimate
        }

    def export_summary(self, jobs: List[TranslationJob]) -> str:
        """Export a summary report of multiple jobs.

        Args:
            jobs: List of translation jobs

        Returns:
            Path to the summary file
        """
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        filename = f"summary_{timestamp}.csv"
        filepath = self.output_dir / filename

        with open(filepath, "w", encoding=self.encoding, newline="") as f:
            writer = csv.writer(f)
            writer.writerow(["job_id", "status", "segments", "flagged", "score", "created_at"])

            for job in jobs:
                writer.writerow([
                    job.id,
                    job.status,
                    job.segment_count,
                    job.flagged_count,
                    f"{job.overall_score:.2f}",
                    job.created_at.isoformat()
                ])

        logger.info(f"[EXPORT] Summary exported to {filepath}")
        return str(filepath)


def create_exporter(
    output_dir: Optional[str] = None,
    encoding: str = "utf-8-sig"
) -> BilingualCSVExporter:
    """Factory function to create a BilingualCSVExporter.

    Args:
        output_dir: Directory to write CSV files (default: temp directory)
        encoding: File encoding for output

    Returns:
        Configured BilingualCSVExporter instance
    """
    return BilingualCSVExporter(
        output_dir=output_dir,
        encoding=encoding
    )
