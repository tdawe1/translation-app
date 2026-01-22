# cmd/translation-worker/main.py
"""
Translation Worker - Main Entry Point

A hybrid worker that combines:
1. Folder watching (for Gengo downloads - loose coupling)
2. Redis job queue (for horizontal scaling)

Supports multi-provider LLM translation, glossary system, cache,
layout preservation, and plugin-based document parsers.
"""

import signal
import sys
import time
import tomli
from pathlib import Path
from typing import Optional

import json
import os
from job_queue import JobManager, JobState
from review.workflow import TranslationWorkflow, ReviewWorkflowBuilder
from review.multimodel import MultiModelTranslator
from review.judge import TranslationJudge
from review.llm import get_provider, AnthropicProvider, OpenAIProvider


def load_config(config_path: str = "config.toml") -> dict:
    """Load configuration from TOML file.

    Args:
        config_path: Path to TOML config file (relative or absolute)

    Returns:
        Parsed configuration as nested dict

    Raises:
        FileNotFoundError: If config file doesn't exist
        tomli.TOMLDecodeError: If TOML is malformed
    """
    config_file = Path(config_path)
    if not config_file.is_absolute():
        # Relative to main.py
        config_file = Path(__file__).parent / config_path

    if not config_file.exists():
        raise FileNotFoundError(f"Config file not found: {config_file}")

    with open(config_file, "rb") as f:
        return tomli.load(f)


def validate_config(cfg: dict) -> list[str]:
    """Validate required configuration sections.

    Args:
        cfg: Parsed configuration dict

    Returns:
        List of validation errors (empty if valid)
    """
    errors = []

    # Check required sections
    required_sections = ["worker", "translation"]
    for section in required_sections:
        if section not in cfg:
            errors.append(f"Missing required section: [{section}]")

    # Validate worker section
    if "worker" in cfg:
        if "max_concurrent" not in cfg["worker"]:
            errors.append("Missing worker.max_concurrent")
        if "heartbeat_interval" not in cfg["worker"]:
            errors.append("Missing worker.heartbeat_interval")

    # Validate translation section
    if "translation" in cfg:
        if "default_provider" not in cfg["translation"]:
            errors.append("Missing translation.default_provider")
        if "default_model" not in cfg["translation"]:
            errors.append("Missing translation.default_model")

    # Validate job_queue section if enabled
    if "job_queue" in cfg and cfg["job_queue"].get("enabled", False):
        if "backend" not in cfg["job_queue"]:
            errors.append("Missing job_queue.backend")
        elif cfg["job_queue"]["backend"] != "redis":
            errors.append(
                f"Unsupported job_queue.backend: {cfg['job_queue']['backend']}"
            )
        if "max_concurrent" not in cfg["job_queue"]:
            errors.append("Missing job_queue.max_concurrent")

    # Validate style_guide section if enabled
    if "style_guide" in cfg and cfg["style_guide"].get("enabled", False):
        if "path" not in cfg["style_guide"]:
            errors.append("style_guide.path required when style_guide.enabled=true")
        else:
            guide_path = Path(cfg["style_guide"]["path"])
            if not guide_path.exists():
                errors.append(f"style_guide.path file not found: {guide_path}")

    return errors


def parse_duration(duration_str: str) -> int:
    """Parse duration string to seconds.

    Args:
        duration_str: Duration string like "10s", "5m", "1h"

    Returns:
        Duration in seconds
    """
    duration_str = duration_str.strip().lower()
    if duration_str.endswith("s"):
        return int(duration_str[:-1])
    elif duration_str.endswith("m"):
        return int(duration_str[:-1]) * 60
    elif duration_str.endswith("h"):
        return int(duration_str[:-1]) * 3600
    else:
        raise ValueError(f"Invalid duration format: {duration_str}")


def get_redis_config(config: dict) -> tuple[str, int, int, Optional[str]]:
    """Extract Redis configuration from config dict.

    Args:
        config: Parsed configuration dict

    Returns:
        Tuple of (host, port, db, password)
    """
    cache_redis = config.get("cache", {}).get("redis", {})
    return (
        cache_redis.get("host", "localhost"),
        cache_redis.get("port", 6379),
        cache_redis.get("db", 0),
        cache_redis.get("password"),
    )


class SegmentExtractor:
    """Extracts translatable segments from various file formats."""

    def extract(self, source_file: str) -> list[dict]:
        """Extract segments from a source file.

        Args:
            source_file: Path to the source file

        Returns:
            List of segment dicts with id, source, context
        """
        from pathlib import Path

        file_path = Path(source_file)
        if not file_path.exists():
            print(f"Warning: Source file not found: {source_file}")
            return []

        ext = file_path.suffix.lower()

        extractors = {
            ".docx": self._extract_docx,
            ".pptx": self._extract_pptx,
            ".pdf": self._extract_pdf,
            ".xlsx": self._extract_xlsx,
            ".txt": self._extract_text,
            ".md": self._extract_text,
        }

        extractor = extractors.get(ext)
        if not extractor:
            print(f"Warning: Unsupported file type: {ext}")
            return []

        try:
            return extractor(file_path)
        except Exception as e:
            print(f"Error extracting segments from {source_file}: {e}")
            return []

    def _extract_docx(self, file_path: Path) -> list[dict]:
        from parsers import create_docx_parser

        parser = create_docx_parser()
        parsed = parser.parse(str(file_path))
        return [
            {
                "id": seg.id or f"para_{i + 1}",
                "source": seg.text,
                "context": seg.context,
            }
            for i, seg in enumerate(parsed.segments)
        ]

    def _extract_pptx(self, file_path: Path) -> list[dict]:
        from parsers import create_pptx_parser

        parser = create_pptx_parser()
        parsed = parser.parse(str(file_path))
        return [
            {
                "id": seg.id or f"slide_text_{i + 1}",
                "source": seg.text,
                "context": seg.context,
            }
            for i, seg in enumerate(parsed.segments)
        ]

    def _extract_pdf(self, file_path: Path) -> list[dict]:
        from parsers import create_pdf_parser

        parser = create_pdf_parser()
        parsed = parser.parse(str(file_path))
        return [
            {
                "id": seg.id or f"page_block_{i + 1}",
                "source": seg.text,
                "context": seg.context,
            }
            for i, seg in enumerate(parsed.segments)
        ]

    def _extract_xlsx(self, file_path: Path) -> list[dict]:
        from parsers import create_xlsx_parser

        parser = create_xlsx_parser()
        parsed = parser.parse(str(file_path))
        segments = []
        for i, seg in enumerate(parsed.segments):
            text = seg.text.strip()
            if text and not text.replace(".", "").isdigit():
                segments.append(
                    {
                        "id": seg.id or f"cell_{i + 1}",
                        "source": text,
                        "context": seg.context,
                    }
                )
        return segments

    def _extract_text(self, file_path: Path) -> list[dict]:
        with open(file_path, "r", encoding="utf-8") as f:
            lines = f.readlines()
        return [
            {
                "id": f"line_{i + 1}",
                "source": line.strip(),
                "context": {"type": "line", "index": i},
            }
            for i, line in enumerate(lines)
            if line.strip()
        ]


class SegmentStore:
    """Stores translation segments to Redis."""

    def __init__(self, redis_client):
        self.redis_client = redis_client

    def store(self, job_id: str, workflow_job, user_id: Optional[str] = None) -> None:
        """Store workflow job segments to Redis.

        Args:
            job_id: The job identifier
            workflow_job: WorkflowJob with segments
            user_id: Optional user ID for namespacing
        """
        job_key = f"user:{user_id}:trans:{job_id}" if user_id else f"trans:{job_id}"
        redis_key = f"{job_key}:segments"

        try:
            segments_data = []
            for seg in workflow_job.segments:
                seg_data = {
                    "segment_id": seg.id,
                    "job_id": job_id,
                    "user_id": user_id,
                    "source": seg.source,
                    "target": seg.target,
                    "judge_winner": seg.judge_winner,
                    "judge_confidence": seg.judge_confidence,
                    "judge_reasoning": seg.judge_reasoning,
                    "is_flagged": seg.is_flagged,
                    "flag_reason": getattr(seg, "flag_reason", ""),
                    "model_a_output": getattr(seg, "model_a_output", ""),
                    "model_b_output": getattr(seg, "model_b_output", ""),
                }
                segments_data.append(json.dumps(seg_data))

            if segments_data:
                self.redis_client.delete(redis_key)
                self.redis_client.rpush(redis_key, *segments_data)
                self.redis_client.expire(redis_key, 86400 * 7)

            self._store_job_meta(job_id, workflow_job, user_id)

        except Exception as e:
            print(f"Warning: Failed to store segments to Redis: {e}")

    def _store_job_meta(
        self, job_id: str, workflow_job, user_id: Optional[str]
    ) -> None:
        job_meta = {
            "status": workflow_job.status,
            "overall_score": str(workflow_job.overall_score),
            "segment_count": str(workflow_job.segment_count),
            "flagged_count": str(workflow_job.flagged_count),
            "progress": "1.0"
            if workflow_job.status in ["completed", "approved", "pending_approval"]
            else "0.5",
        }
        if job_id:
            job_meta["job_id"] = job_id
        if user_id:
            job_meta["user_id"] = user_id
        job_key = f"user:{user_id}:trans:{job_id}" if user_id else f"trans:{job_id}"
        self.redis_client.hset(job_key, mapping=job_meta)


class QueueConsumer:
    """Consumes jobs from Redis queue and processes them."""

    def __init__(
        self,
        job_manager: JobManager,
        worker_id: str,
        max_concurrent: int = 3,
        workflow: Optional[TranslationWorkflow] = None,
        config: Optional[dict] = None,
    ):
        self.job_manager = job_manager
        self.worker_id = worker_id
        self.max_concurrent = max_concurrent
        self.running = False
        self.active_jobs: dict[str, dict] = {}
        self.active_job_order: list[str] = []
        self.workflow = workflow
        self.config = config or {}
        self.segment_extractor = SegmentExtractor()
        self.segment_store = SegmentStore(job_manager.redis_client)

    def start(self, poll_interval: int = 1):
        """Start the queue consumer loop.

        Args:
            poll_interval: Seconds between queue polls
        """
        self.running = True
        print(f"Queue consumer started (max_concurrent={self.max_concurrent})")

        try:
            while self.running:
                self._process_cycle()
                self._cleanup_completed_jobs()
                time.sleep(poll_interval)
        finally:
            self._cleanup_completed_jobs()

    def stop(self):
        """Stop the queue consumer gracefully."""
        print("Stopping queue consumer...")
        self.running = False

    def _process_cycle(self):
        """Process one cycle of job dequeuing and execution."""
        # Check if we're at capacity
        if len(self.active_jobs) >= self.max_concurrent:
            return

        # Try to get a job from the queue
        job = self.job_manager.dequeue(
            worker_id=self.worker_id,
            timeout=0,  # Non-blocking
            priorities=None,  # Check all priorities
        )

        if not job:
            return

        job_id = job.get("id")
        if not job_id:
            return

        print(f"Dequeued job: {job_id} ({job.get('source_file', 'unknown file')})")

        # Process job (placeholder - in real implementation, this would
        # invoke the translation pipeline with checkpoint/resume)
        self._process_job(job)

    def _process_job(self, job: dict):
        """Process a single translation job through the workflow."""
        job_id = job.get("id")
        if not job_id:
            print("Warning: Job missing ID, skipping")
            return

        source_file = job.get("source_file", "")
        project_type = job.get("project_type", "routine")

        self.job_manager.set_state(job_id, JobState.TRANSLATING, self.worker_id)
        self.active_jobs[job_id] = job
        self.active_job_order.append(job_id)

        try:
            if not self.workflow:
                print(
                    f"Warning: No workflow configured, completing job {job_id} as stub"
                )
                self.job_manager.set_state(job_id, JobState.COMPLETED, self.worker_id)
                self.job_manager.publish_progress(job_id, 1.0, "No workflow configured")
                del self.active_jobs[job_id]
                self.active_job_order.remove(job_id)
                return

            segments = self.segment_extractor.extract(source_file)
            if not segments:
                self.job_manager.set_state(job_id, JobState.FAILED, self.worker_id)
                self.job_manager.publish_progress(job_id, 0.0, "No segments extracted")
                del self.active_jobs[job_id]
                self.active_job_order.remove(job_id)
                return

            workflow_job = self.workflow.create_job(
                source_file=source_file,
                target_file=job.get(
                    "target_file", source_file.replace(".", "_translated.")
                ),
                project_type=project_type,
                segments=segments,
            )
            user_id = job.get("user_id")

            def progress_callback(message: str, current: int, total: int):
                progress = current / total if total > 0 else 0
                self.job_manager.publish_progress(job_id, progress, message)
                self.segment_store.store(job_id, workflow_job, user_id)

            processed_job = self.workflow.process_job(workflow_job, progress_callback)
            self.segment_store.store(job_id, processed_job, user_id)

            final_status = JobState.REVIEW_PENDING
            if processed_job.status == "approved":
                final_status = JobState.COMPLETED
            elif processed_job.status == "rejected":
                final_status = JobState.FAILED

            self.job_manager.set_state(job_id, final_status, self.worker_id)

            self.job_manager.publish_progress(
                job_id,
                1.0,
                f"Translation complete: score={processed_job.overall_score:.2f}, flagged={processed_job.flagged_count}",
            )

            print(
                f"Job {job_id} processed: score={processed_job.overall_score:.2f}, flagged={processed_job.flagged_count}"
            )

        except Exception as e:
            print(f"Error processing job {job_id}: {e}")
            import traceback

            traceback.print_exc()
            self.job_manager.set_state(job_id, JobState.FAILED, self.worker_id)
            self.job_manager.publish_progress(job_id, 0.0, f"Error: {str(e)}")
        finally:
            if job_id in self.active_jobs:
                del self.active_jobs[job_id]
            if job_id in self.active_job_order:
                self.active_job_order.remove(job_id)

    def _cleanup_completed_jobs(self):
        """Remove completed jobs from active tracking."""
        completed = []
        for job_id, job in self.active_jobs.items():
            state = self.job_manager.get_state(job_id)
            if state and state in {
                JobState.COMPLETED,
                JobState.FAILED,
                JobState.CANCELLED,
            }:
                completed.append(job_id)

        for job_id in completed:
            if job_id in self.active_jobs:
                del self.active_jobs[job_id]
            if job_id in self.active_job_order:
                self.active_job_order.remove(job_id)

        self._enforce_lru_limit()

    def _enforce_lru_limit(self, max_size: int = 1000):
        while len(self.active_jobs) > max_size:
            oldest_id = self.active_job_order.pop(0)
            if oldest_id in self.active_jobs:
                del self.active_jobs[oldest_id]


def main():
    """Main entry point for the translation worker."""
    # Track queue consumer for graceful shutdown
    queue_consumer: Optional[QueueConsumer] = None
    job_manager: Optional[JobManager] = None

    try:
        # Load configuration
        config = load_config()

        # Validate configuration
        validation_errors = validate_config(config)
        if validation_errors:
            print("Configuration errors:", file=sys.stderr)
            for error in validation_errors:
                print(f"  - {error}", file=sys.stderr)
            sys.exit(1)

        # Display worker info
        worker_id = config.get("worker", {}).get("id", "unspecified")
        provider = config.get("translation", {}).get("default_provider")
        model = config.get("translation", {}).get("default_model")

        # Load Gengo style guide if enabled
        system_prompt = None
        style_guide_enabled = config.get("style_guide", {}).get("enabled", False)
        if style_guide_enabled:
            from style_guide.parser import parse_gengo_style_guide
            from style_guide.prompt_builder import build_system_prompt

            style_guide_path = Path(config["style_guide"]["path"])
            if style_guide_path.exists():
                try:
                    guide = parse_gengo_style_guide(style_guide_path)
                    system_prompt = build_system_prompt(guide)
                    print(f"  Style Guide: loaded ({len(guide.sections)} sections)")
                except Exception as e:
                    print(f"Warning: Failed to load style guide: {e}", file=sys.stderr)
            else:
                print(
                    f"Warning: Style guide not found: {style_guide_path}",
                    file=sys.stderr,
                )
        else:
            print(f"  Style Guide: disabled")

        print(f"Translation Worker v1.0.0 starting...")
        print(f"  Worker ID: {worker_id}")
        print(f"  Translation Backend: {provider}/{model}")
        print(f"  Mode: hybrid (folder watch + Redis job queue)")

        # Initialize JobManager if job queue enabled
        job_queue_config = config.get("job_queue", {})
        job_queue_enabled = job_queue_config.get("enabled", False)
        poll_interval = 1

        if job_queue_enabled:
            redis_host, redis_port, redis_db, redis_pwd = get_redis_config(config)

            print(f"  Redis: {redis_host}:{redis_port}/{redis_db}")

            job_manager = JobManager(
                redis_host=redis_host,
                redis_port=redis_port,
                redis_db=redis_db,
                redis_password=redis_pwd,
                decode_responses=False,
            )

            # Parse queue poll interval
            poll_interval_str = job_queue_config.get("poll_interval", "1s")
            poll_interval = parse_duration(poll_interval_str)

            # Get max concurrent jobs
            max_concurrent = job_queue_config.get("max_concurrent", 3)

            # Initialize queue consumer
            queue_consumer = QueueConsumer(
                job_manager=job_manager,
                worker_id=worker_id,
                max_concurrent=max_concurrent,
            )

            # Setup signal handlers for graceful shutdown
            def signal_handler(signum, frame):
                queue_consumer.stop()

            signal.signal(signal.SIGINT, signal_handler)
            signal.signal(signal.SIGTERM, signal_handler)

            print("Job queue initialized and enabled.")
        else:
            print("Job queue disabled in configuration.")

        # TODO: Initialize other components
        # - Glossary loader
        # - Cache manager
        # - Plugin registry
        # - Folder watcher

        print("Worker initialized successfully.")
        print("Press Ctrl+C to stop.")

        # Start queue consumer if enabled
        if job_queue_enabled and queue_consumer:
            queue_consumer.start(poll_interval=poll_interval)
        else:
            # If no queue enabled, just wait for interrupt
            while True:
                time.sleep(1)

    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        print(
            "Create a config.toml file or specify path with --config", file=sys.stderr
        )
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nShutting down gracefully...")
    except Exception as e:
        print(f"Unexpected error: {e}", file=sys.stderr)
        import traceback

        traceback.print_exc()
        sys.exit(1)
    finally:
        # Cleanup
        if job_manager:
            job_manager.close()


if __name__ == "__main__":
    main()
