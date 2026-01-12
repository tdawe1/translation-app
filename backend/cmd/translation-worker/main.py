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

from job_queue import JobManager, JobState


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
            errors.append(f"Unsupported job_queue.backend: {cfg['job_queue']['backend']}")
        if "max_concurrent" not in cfg["job_queue"]:
            errors.append("Missing job_queue.max_concurrent")

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


class QueueConsumer:
    """Consumes jobs from Redis queue and processes them."""

    def __init__(self, job_manager: JobManager, worker_id: str, max_concurrent: int = 3):
        """Initialize queue consumer.

        Args:
            job_manager: JobManager instance for queue operations
            worker_id: Worker identifier
            max_concurrent: Maximum concurrent jobs to process
        """
        self.job_manager = job_manager
        self.worker_id = worker_id
        self.max_concurrent = max_concurrent
        self.running = False
        self.active_jobs: dict[str, dict] = {}

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
        """Process a single job.

        Args:
            job: Job data dict
        """
        job_id = job.get("id")

        # Mark as translating
        self.job_manager.set_state(job_id, JobState.TRANSLATING, self.worker_id)

        # TODO: Execute actual translation pipeline
        # For now, simulate processing
        self.active_jobs[job_id] = job

        # Placeholder: In real implementation, this would:
        # 1. Load any existing checkpoint
        # 2. Resume or start translation
        # 3. Save checkpoints periodically
        # 4. Update progress via publish_progress

        # Mark as completed (placeholder)
        self.job_manager.set_state(job_id, JobState.COMPLETED, self.worker_id)
        self.job_manager.publish_progress(job_id, 1.0, "Translation completed")
        print(f"Job completed: {job_id}")

        del self.active_jobs[job_id]

    def _cleanup_completed_jobs(self):
        """Remove completed jobs from active tracking."""
        completed = []
        for job_id, job in self.active_jobs.items():
            state = self.job_manager.get_state(job_id)
            if state and state in JobState.TERMINAL.value:
                completed.append(job_id)

        for job_id in completed:
            del self.active_jobs[job_id]


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

        print(f"Translation Worker v1.0.0 starting...")
        print(f"  Worker ID: {worker_id}")
        print(f"  Translation Backend: {provider}/{model}")
        print(f"  Mode: hybrid (folder watch + Redis job queue)")

        # Initialize JobManager if job queue enabled
        job_queue_config = config.get("job_queue", {})
        job_queue_enabled = job_queue_config.get("enabled", False)

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
        print("Create a config.toml file or specify path with --config", file=sys.stderr)
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
