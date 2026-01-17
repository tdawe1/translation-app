# queue/manager.py
"""
Redis-backed job queue manager for translation worker.

Provides distributed job queue with priority support, state management,
checkpoint/resume for fault tolerance, and progress publishing via pub/sub.
"""

import json
import time
from datetime import datetime, timezone
from typing import Optional, Dict, List, Any

from .job import Job, JobState, Checkpoint


class JobManager:
    """Manages job queue and state in Redis.

    Uses Redis sorted sets for priority queues and separate keys for:
    - Jobs: full job data with TTL
    - States: current job state with worker tracking
    - Checkpoints: progress snapshots for resume
    - Progress: pub/sub channel for real-time updates

    Multi-tenancy: When user_id is provided, all keys are namespaced:
        user:{user_id}:trans:queue:* (instead of trans:queue:*)

    Example:
        >>> manager = JobManager(redis_host="localhost", user_id="user-123")
        >>> job_id = manager.enqueue({"source_file": "/path/to/file.pptx"})
        >>> job = manager.dequeue(worker_id="worker-1")
        >>> manager.set_state(job_id, JobState.PROCESSING, worker_id="worker-1")
        >>> manager.save_checkpoint(job_id, {"progress": 0.5})
    """

    # Redis key base prefixes (without user namespace)
    _BASE_QUEUE_PREFIX = "trans:queue:"
    _BASE_JOB_PREFIX = "trans:job:"
    _BASE_STATE_PREFIX = "trans:state:"
    _BASE_CHECKPOINT_PREFIX = "trans:checkpoint:"
    _BASE_WORKER_QUEUE_PREFIX = "trans:worker:"
    _BASE_PROGRESS_CHANNEL = "translation:progress"

    # TTL settings (in seconds)
    JOB_TTL = 86400  # 24 hours
    STATE_TTL = 86400
    CHECKPOINT_TTL = 604800  # 7 days

    # Priorities (lower score = higher priority)
    PRIORITIES = {"urgent": 0, "normal": 1, "bulk": 2}

    def __init__(
        self,
        redis_host: str = "localhost",
        redis_port: int = 6379,
        redis_db: int = 0,
        redis_password: Optional[str] = None,
        decode_responses: bool = False,
        user_id: Optional[str] = None,
    ):
        """Initialize the job manager.

        Args:
            redis_host: Redis server host
            redis_port: Redis server port
            redis_db: Redis database number
            redis_password: Optional Redis password
            decode_responses: Whether to decode responses (False for binary data)
            user_id: Optional user ID for multi-tenancy namespacing.
                     When provided, all Redis keys are prefixed with user:{user_id}:
                     When None, uses legacy global keys for backward compatibility.
        """
        try:
            import redis
        except ImportError:
            raise RuntimeError(
                "redis package not installed. Install with: pip install redis"
            )

        self.redis_client = redis.Redis(
            host=redis_host,
            port=redis_port,
            db=redis_db,
            password=redis_password,
            decode_responses=decode_responses,
            socket_connect_timeout=5,
            socket_timeout=5,
        )

        self.redis_host = redis_host
        self.redis_port = redis_port
        self.redis_db = redis_db
        self.user_id = user_id

    def _key_prefix(self) -> str:
        """Get the Redis key prefix for multi-tenancy.

        Returns:
            "user:{user_id}:" if user_id is set, "" otherwise.
        """
        if self.user_id:
            return f"user:{self.user_id}:"
        return ""

    def _encode(self, data: Any) -> bytes:
        """Encode data for Redis storage."""
        return json.dumps(data).encode("utf-8")

    def _decode(self, data: bytes) -> Any:
        """Decode data from Redis storage."""
        return json.loads(data.decode("utf-8"))

    def enqueue(
        self,
        job_data: Dict[str, Any],
        priority: str = "normal",
        delay_seconds: int = 0,
    ) -> Optional[str]:
        """Enqueue a job for translation.

        Args:
            job_data: Job data including source_file, source_lang, target_lang
            priority: Queue priority ("urgent", "normal", "bulk")
            delay_seconds: Delay before job becomes available

        Returns:
            Job ID if successfully enqueued, None otherwise
        """
        if priority not in self.PRIORITIES:
            priority = "normal"

        job = Job(
            id=None,
            source_file=job_data.get("source_file", ""),
            state=JobState.PENDING,
            metadata=job_data,
        )

        # Store job data
        job_key = f"{self._key_prefix()}{self._BASE_JOB_PREFIX}{job.id}"
        job_json = self._encode(
            {
                **job_data,
                "id": job.id,
                "source_file": job.source_file,
                "state": job.state.value,
                "created_at": job.created_at,
            }
        )

        pipeline = self.redis_client.pipeline()
        pipeline.set(job_key, job_json, ex=self.JOB_TTL)

        # Add to priority queue (sorted by score)
        queue_key = f"{self._key_prefix()}{self._BASE_QUEUE_PREFIX}{priority}"
        score = time.time() + delay_seconds
        pipeline.zadd(queue_key, {job.id: score})

        # Set initial state
        state_key = f"{self._key_prefix()}{self._BASE_STATE_PREFIX}{job.id}"
        state_json = self._encode(
            {
                "state": job.state.value,
                "worker_id": None,
                "timestamp": datetime.now(timezone.utc).isoformat(),
            }
        )
        pipeline.set(state_key, state_json, ex=self.STATE_TTL)

        try:
            pipeline.execute()
            return job.id
        except Exception:
            return None

    def dequeue(
        self,
        worker_id: str,
        timeout: int = 1,
        priorities: Optional[List[str]] = None,
    ) -> Optional[Dict[str, Any]]:
        """Dequeue next available job.

        Checks queues by priority (urgent → normal → bulk) and returns
        the highest priority job that's available.

        Args:
            worker_id: ID of worker requesting job
            timeout: Seconds to wait for job (blocking not implemented)
            priorities: List of priorities to check (default: all)

        Returns:
            Job data dict if job available, None otherwise
        """
        if priorities is None:
            priorities = ["urgent", "normal", "bulk"]

        for priority in priorities:
            queue_key = f"{self._key_prefix()}{self._BASE_QUEUE_PREFIX}{priority}"

            # Get and remove oldest job (lowest score)
            result = self.redis_client.zpopmin(queue_key)

            if result:
                job_id, score = result
                return self._get_job(job_id, worker_id)

        return None

    def _get_job(
        self, job_id: str, worker_id: Optional[str] = None
    ) -> Optional[Dict[str, Any]]:
        """Get job data and mark as processing.

        Args:
            job_id: Job identifier
            worker_id: Optional worker ID to assign

        Returns:
            Job data dict if found, None otherwise
        """
        job_key = f"{self._key_prefix()}{self._BASE_JOB_PREFIX}{job_id}"
        data = self.redis_client.get(job_key)

        if data:
            job_data = self._decode(data)

            # Update state to PROCESSING and assign worker
            self.set_state(job_id, JobState.PROCESSING, worker_id)

            # Update job data with worker and start time
            job_data["worker_id"] = worker_id
            job_data["started_at"] = datetime.now(timezone.utc).isoformat()

            # Store updated job data
            self.redis_client.set(job_key, self._encode(job_data), ex=self.JOB_TTL)

            return job_data

        return None

    def set_state(
        self,
        job_id: str,
        state: JobState,
        worker_id: Optional[str] = None,
    ) -> bool:
        """Update job state.

        Args:
            job_id: Job identifier
            state: New state
            worker_id: Optional worker ID to associate

        Returns:
            True if state updated successfully, False otherwise
        """
        try:
            state_key = f"{self._key_prefix()}{self._BASE_STATE_PREFIX}{job_id}"
            state_json = self._encode(
                {
                    "state": state.value,
                    "worker_id": worker_id,
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                }
            )
            self.redis_client.set(state_key, state_json, ex=self.STATE_TTL)

            # Update job data too
            job_key = f"{self._key_prefix()}{self._BASE_JOB_PREFIX}{job_id}"
            job_data = self.redis_client.get(job_key)
            if job_data:
                data = self._decode(job_data)
                data["state"] = state.value
                if worker_id:
                    data["worker_id"] = worker_id

                # Set completion timestamp for terminal states
                if state.value in JobState.TERMINAL.value:
                    data["completed_at"] = datetime.now(timezone.utc).isoformat()

                self.redis_client.set(job_key, self._encode(data), ex=self.JOB_TTL)

            return True
        except Exception:
            return False

    def get_state(self, job_id: str) -> Optional[JobState]:
        """Get current job state.

        Args:
            job_id: Job identifier

        Returns:
            Current JobState if found, None otherwise
        """
        state_key = f"{self._key_prefix()}{self._BASE_STATE_PREFIX}{job_id}"
        data = self.redis_client.get(state_key)

        if data:
            state_data = self._decode(data)
            try:
                return JobState(state_data["state"])
            except (ValueError, KeyError):
                return JobState.PENDING

        return None

    def get_job(self, job_id: str) -> Optional[Dict[str, Any]]:
        """Get full job data.

        Args:
            job_id: Job identifier

        Returns:
            Job data dict if found, None otherwise
        """
        job_key = f"{self._key_prefix()}{self._BASE_JOB_PREFIX}{job_id}"
        data = self.redis_client.get(job_key)

        if data:
            return self._decode(data)

        return None

    def save_checkpoint(
        self,
        job_id: str,
        checkpoint_data: Dict[str, Any],
        progress: Optional[float] = None,
        source_hash: Optional[str] = None,
    ) -> Optional[str]:
        """Save job progress checkpoint for resume.

        Checkpoints enable fault tolerance by saving progress that can be
        loaded to resume translation after a crash or restart.

        Args:
            job_id: Job identifier
            checkpoint_data: Arbitrary checkpoint data (segments, state, etc.)
            progress: Optional progress override (0.0 to 1.0)
            source_hash: Optional hash of source for change detection

        Returns:
            Checkpoint ID if saved successfully, None otherwise
        """
        try:
            checkpoint = Checkpoint(
                job_id=job_id,
                checkpoint_id="",
                timestamp="",
                progress=progress or 0.0,
                data=checkpoint_data,
                source_hash=source_hash,
            )

            key = f"{self._key_prefix()}{self._BASE_CHECKPOINT_PREFIX}{job_id}"
            self.redis_client.setex(
                key, self.CHECKPOINT_TTL, self._encode(checkpoint.to_dict())
            )

            # Update job progress if provided
            if progress is not None:
                self.update_progress(job_id, progress)

            # Update job's checkpoint_id reference
            job_key = f"{self._key_prefix()}{self._BASE_JOB_PREFIX}{job_id}"
            job_data = self.redis_client.get(job_key)
            if job_data:
                data = self._decode(job_data)
                data["checkpoint_id"] = checkpoint.checkpoint_id
                self.redis_client.set(job_key, self._encode(data), ex=self.JOB_TTL)

            return checkpoint.checkpoint_id
        except Exception:
            return None

    def load_checkpoint(self, job_id: str) -> Optional[Dict[str, Any]]:
        """Load job checkpoint for resume.

        Args:
            job_id: Job identifier

        Returns:
            Checkpoint data dict if found, None otherwise
        """
        key = f"{self._key_prefix()}{self._BASE_CHECKPOINT_PREFIX}{job_id}"
        data = self.redis_client.get(key)

        if data:
            return self._decode(data)

        return None

    def update_progress(
        self,
        job_id: str,
        progress: float,
        message: str = "",
    ) -> bool:
        """Update job progress and publish to pub/sub.

        Args:
            job_id: Job identifier
            progress: Progress percentage (0.0 to 1.0)
            message: Optional progress message

        Returns:
            True if update successful, False otherwise
        """
        try:
            # Clamp progress to valid range
            progress = max(0.0, min(1.0, progress))

            # Update job data
            job_key = f"{self._key_prefix()}{self._BASE_JOB_PREFIX}{job_id}"
            job_data = self.redis_client.get(job_key)
            if job_data:
                data = self._decode(job_data)
                data["progress"] = progress
                self.redis_client.set(job_key, self._encode(data), ex=self.JOB_TTL)

            # Publish to pub/sub
            self.publish_progress(job_id, progress, message)

            return True
        except Exception:
            return False

    def publish_progress(
        self,
        job_id: str,
        progress: float,
        message: str = "",
    ) -> bool:
        """Publish job progress to Redis pub/sub.

        Args:
            job_id: Job identifier
            progress: Progress percentage (0.0 to 1.0)
            message: Optional progress message

        Returns:
            True if published successfully, False otherwise
        """
        try:
            channel = f"{self._key_prefix()}{self._BASE_PROGRESS_CHANNEL}"
            data = self._encode(
                {
                    "job_id": job_id,
                    "progress": progress,
                    "message": message,
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                }
            )
            self.redis_client.publish(channel, data)
            return True
        except Exception:
            return False

    def cancel_job(self, job_id: str, reason: str = "") -> bool:
        """Cancel a job.

        Args:
            job_id: Job identifier
            reason: Optional cancellation reason

        Returns:
            True if cancelled successfully, False otherwise
        """
        return self.set_state(job_id, JobState.CANCELLED)

    def fail_job(self, job_id: str, error: str) -> bool:
        """Mark a job as failed.

        Args:
            job_id: Job identifier
            error: Error message

        Returns:
            True if marked failed successfully, False otherwise
        """
        try:
            # Update state
            self.set_state(job_id, JobState.FAILED)

            # Store error message
            job_key = f"{self._key_prefix()}{self._BASE_JOB_PREFIX}{job_id}"
            job_data = self.redis_client.get(job_key)
            if job_data:
                data = self._decode(job_data)
                data["error"] = error
                self.redis_client.set(job_key, self._encode(data), ex=self.JOB_TTL)

            return True
        except Exception:
            return False

    def get_worker_jobs(self, worker_id: str) -> List[Dict[str, Any]]:
        """Get all jobs assigned to a worker.

        Args:
            worker_id: Worker identifier

        Returns:
            List of job data dicts assigned to worker
        """
        try:
            # Scan all job keys and filter by worker_id
            pattern = f"{self._key_prefix()}{self._BASE_JOB_PREFIX}*"
            jobs = []

            for key in self.redis_client.scan_iter(match=pattern):
                data = self.redis_client.get(key)
                if data:
                    job_data = self._decode(data)
                    if job_data.get("worker_id") == worker_id:
                        jobs.append(job_data)

            # Sort by created_at
            jobs.sort(key=lambda j: j.get("created_at", ""))

            return jobs
        except Exception:
            return []

    def get_queue_stats(self) -> Dict[str, int]:
        """Get queue statistics.

        Returns:
            Dict with queue sizes and status
        """
        stats = {"urgent": 0, "normal": 0, "bulk": 0, "total": 0}

        try:
            for priority in self.PRIORITIES:
                queue_key = f"{self._key_prefix()}{self._BASE_QUEUE_PREFIX}{priority}"
                count = self.redis_client.zcard(queue_key)
                stats[priority] = count or 0
                stats["total"] += stats[priority]
        except Exception:
            pass

        return stats

    def cleanup_old_jobs(self, max_age_seconds: int = 86400) -> int:
        """Remove completed/failed jobs older than max age.

        Args:
            max_age_seconds: Maximum age in seconds (default 24 hours)

        Returns:
            Number of jobs cleaned up
        """
        cleaned = 0
        try:
            pattern = f"{self._key_prefix()}{self._BASE_JOB_PREFIX}*"
            cutoff = datetime.now(timezone.utc).timestamp() - max_age_seconds

            for key in self.redis_client.scan_iter(match=pattern):
                data = self.redis_client.get(key)
                if data:
                    job_data = self._decode(data)
                    completed_at = job_data.get("completed_at")

                    if completed_at:
                        # Parse ISO timestamp
                        try:
                            from datetime import datetime as dt

                            comp_time = dt.fromisoformat(completed_at)
                            if comp_time.timestamp() < cutoff:
                                self.redis_client.delete(key)
                                # Also delete state and checkpoint
                                job_id = job_data.get("id")
                                if job_id:
                                    self.redis_client.delete(
                                        f"{self._key_prefix()}{self._BASE_STATE_PREFIX}{job_id}"
                                    )
                                    self.redis_client.delete(
                                        f"{self._key_prefix()}{self._BASE_CHECKPOINT_PREFIX}{job_id}"
                                    )
                                cleaned += 1
                        except (ValueError, TypeError):
                            pass
        except Exception:
            pass

        return cleaned

    def close(self):
        """Close Redis connection."""
        try:
            self.redis_client.close()
        except Exception:
            pass

    def __enter__(self):
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()


def create_job_manager(
    redis_host: str = "localhost",
    redis_port: int = 6379,
    redis_db: int = 0,
    user_id: Optional[str] = None,
) -> JobManager:
    """Factory function to create a configured job manager.

    Args:
        redis_host: Redis server host
        redis_port: Redis server port
        redis_db: Redis database number
        user_id: Optional user ID for multi-tenancy namespacing

    Returns:
        Configured JobManager instance
    """
    return JobManager(
        redis_host=redis_host,
        redis_port=redis_port,
        redis_db=redis_db,
        user_id=user_id,
    )
