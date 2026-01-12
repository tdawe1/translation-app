# queue/__init__.py
"""
Redis-backed job queue for translation worker.

Provides distributed job queue with:
- Priority support (urgent, normal, bulk)
- State management with worker tracking
- Checkpoint/resume for fault tolerance
- Progress publishing via pub/sub

Example:
    >>> from queue import create_job_manager
    >>> manager = create_job_manager(redis_host="localhost")
    >>> job_id = manager.enqueue({"source_file": "/path/to/file.pptx"})
    >>> job = manager.dequeue(worker_id="worker-1")
"""

from .job import (
    JobState,
    Job,
    Checkpoint,
    create_job,
)

from .manager import (
    JobManager,
    create_job_manager,
)

__all__ = [
    # Job types
    "JobState",
    "Job",
    "Checkpoint",
    "create_job",
    # Manager
    "JobManager",
    "create_job_manager",
]
