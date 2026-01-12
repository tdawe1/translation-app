# queue/job.py
"""
Job queue data structures for translation worker.

Defines JobState enum for job lifecycle and Job dataclass for job metadata.
Jobs flow through states: PENDING → PROCESSING → TRANSLATING →
REVIEW_PENDING → APPROVED → COMPLETED, or to FAILED/CANCELLED on errors.
"""

import uuid
from dataclasses import dataclass, field
from datetime import datetime, timezone
from enum import Enum
from typing import Optional, Any, Dict


class JobState(Enum):
    """Job lifecycle states.

    States:
        PENDING: Job queued, waiting for worker
        PROCESSING: Worker picked up job, initializing
        TRANSLATING: Active translation in progress
        REVIEW_PENDING: Translation complete, awaiting review
        APPROVED: Review passed, final output being generated
        COMPLETED: Job fully finished
        FAILED: Job failed with error
        CANCELLED: Job cancelled by user
    """

    PENDING = "pending"
    PROCESSING = "processing"
    TRANSLATING = "translating"
    REVIEW_PENDING = "review_pending"
    APPROVED = "approved"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"

    # Terminal states (no further transitions possible)
    TERMINAL = {COMPLETED, FAILED, CANCELLED}

    # Active states (job is being worked on)
    ACTIVE = {PROCESSING, TRANSLATING}

    # Waiting states (job waiting for something)
    WAITING = {PENDING, REVIEW_PENDING, APPROVED}


@dataclass
class Job:
    """Translation job metadata.

    Attributes:
        id: Unique job identifier (UUID)
        source_file: Path to source file for translation
        state: Current job state
        worker_id: ID of worker processing this job
        progress: Progress percentage (0.0 to 1.0)
        error: Error message if job failed
        metadata: Additional job metadata (source_lang, target_lang, etc.)
        created_at: Job creation timestamp
        started_at: Job start timestamp
        completed_at: Job completion timestamp
        checkpoint_id: ID of last saved checkpoint (for resume)
    """

    id: str
    source_file: str
    state: JobState = JobState.PENDING
    worker_id: Optional[str] = None
    progress: float = 0.0
    error: Optional[str] = None
    metadata: Dict[str, Any] = field(default_factory=dict)
    created_at: Optional[str] = None
    started_at: Optional[str] = None
    completed_at: Optional[str] = None
    checkpoint_id: Optional[str] = None

    def __post_init__(self):
        """Generate UUID if id not provided."""
        # Only generate if id is empty string, not if it's a valid UUID
        if not self.id:
            self.id = str(uuid.uuid4())

        # Set created_at if not provided
        if self.created_at is None:
            self.created_at = datetime.now(timezone.utc).isoformat()

    def is_terminal(self) -> bool:
        """Check if job is in a terminal state."""
        return self.state in JobState.TERMINAL

    def is_active(self) -> bool:
        """Check if job is actively being processed."""
        return self.state in JobState.ACTIVE

    def is_waiting(self) -> bool:
        """Check if job is waiting for processing."""
        return self.state in JobState.WAITING

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            "id": self.id,
            "source_file": self.source_file,
            "state": self.state.value,
            "worker_id": self.worker_id,
            "progress": self.progress,
            "error": self.error,
            "metadata": self.metadata,
            "created_at": self.created_at,
            "started_at": self.started_at,
            "completed_at": self.completed_at,
            "checkpoint_id": self.checkpoint_id,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Job":
        """Create Job from dictionary."""
        # Convert state string to JobState enum
        if "state" in data and isinstance(data["state"], str):
            data = data.copy()
            try:
                data["state"] = JobState(data["state"])
            except ValueError:
                # Invalid state, default to PENDING
                data["state"] = JobState.PENDING

        return cls(**data)


@dataclass
class Checkpoint:
    """Job progress checkpoint for fault tolerance.

    Attributes:
        job_id: Associated job ID
        checkpoint_id: Unique checkpoint ID (UUID)
        timestamp: Checkpoint creation time
        progress: Job progress at checkpoint (0.0 to 1.0)
        data: Checkpoint data (translated segments, state, etc.)
        source_hash: Hash of source file (for change detection)
    """

    job_id: str
    checkpoint_id: str
    timestamp: str
    progress: float
    data: Dict[str, Any]
    source_hash: Optional[str] = None

    def __post_init__(self):
        """Generate checkpoint_id if not provided."""
        if not self.checkpoint_id:
            self.checkpoint_id = str(uuid.uuid4())

        # Set timestamp if not provided
        if not self.timestamp:
            self.timestamp = datetime.now(timezone.utc).isoformat()

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            "job_id": self.job_id,
            "checkpoint_id": self.checkpoint_id,
            "timestamp": self.timestamp,
            "progress": self.progress,
            "data": self.data,
            "source_hash": self.source_hash,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Checkpoint":
        """Create Checkpoint from dictionary."""
        return cls(**data)


def create_job(
    source_file: str,
    metadata: Optional[Dict[str, Any]] = None,
    job_id: Optional[str] = None,
) -> Job:
    """Factory function to create a new job.

    Args:
        source_file: Path to source file
        metadata: Optional job metadata
        job_id: Optional pre-generated job ID

    Returns:
        New Job instance
    """
    return Job(
        id=job_id or str(uuid.uuid4()),
        source_file=source_file,
        metadata=metadata or {},
    )
