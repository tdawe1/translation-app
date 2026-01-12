# tests/test_queue/test_manager.py
"""
Tests for JobManager.

Tests Redis-backed job queue management:
- Enqueue/dequeue with priorities
- State transitions
- Checkpoint save/load
- Progress publishing
- Job recovery and cleanup
"""

import json
import sys
import time
from pathlib import Path
from unittest.mock import Mock, MagicMock, patch

import pytest

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from job_queue.job import JobState, Job, Checkpoint
from job_queue.manager import JobManager, create_job_manager


@pytest.fixture
def mock_redis():
    """Create a mock Redis client."""
    mock = MagicMock()
    # Mock pipeline
    pipeline = MagicMock()
    pipeline.execute.return_value = None
    mock.pipeline.return_value = pipeline
    # Mock common operations
    mock.set.return_value = True
    mock.setex.return_value = True
    mock.get.return_value = None
    mock.zpopmin.return_value = None
    mock.zcard.return_value = 0
    mock.scan_iter.return_value = []
    mock.delete.return_value = 1
    mock.publish.return_value = 1
    return mock


@pytest.fixture
def manager(mock_redis):
    """Create JobManager with mock Redis."""
    # Create a mock redis module with a Redis class that returns our mock
    import sys

    mock_redis_module = MagicMock()
    mock_redis_module.Redis = MagicMock(return_value=mock_redis)

    # Save original and inject mock
    original_redis = sys.modules.get("redis")
    sys.modules["redis"] = mock_redis_module

    try:
        manager = JobManager(decode_responses=False)
        # Ensure our mock is being used
        manager.redis_client = mock_redis
        return manager
    finally:
        # Restore original redis module
        if original_redis is not None:
            sys.modules["redis"] = original_redis
        elif "redis" in sys.modules:
            del sys.modules["redis"]


class TestJobManagerInit:
    """Test JobManager initialization."""

    def test_initialization_default(self):
        """Should initialize with default settings."""
        import sys

        mock_redis_module = MagicMock()
        mock_redis_module.Redis = MagicMock()
        original_redis = sys.modules.get("redis")
        sys.modules["redis"] = mock_redis_module

        try:
            manager = JobManager()

            assert manager.redis_host == "localhost"
            assert manager.redis_port == 6379
            assert manager.redis_db == 0
            mock_redis_module.Redis.assert_called_once()
        finally:
            if original_redis is not None:
                sys.modules["redis"] = original_redis
            elif "redis" in sys.modules:
                del sys.modules["redis"]

    def test_initialization_custom(self):
        """Should accept custom connection settings."""
        import sys

        mock_redis_module = MagicMock()
        mock_redis_module.Redis = MagicMock()
        original_redis = sys.modules.get("redis")
        sys.modules["redis"] = mock_redis_module

        try:
            manager = JobManager(
                redis_host="custom.host",
                redis_port=6380,
                redis_db=1,
                redis_password="secret",
            )

            assert manager.redis_host == "custom.host"
            assert manager.redis_port == 6380
            assert manager.redis_db == 1
        finally:
            if original_redis is not None:
                sys.modules["redis"] = original_redis
            elif "redis" in sys.modules:
                del sys.modules["redis"]

    def test_initialization_without_redis(self):
        """Should raise error if redis not installed."""
        import sys

        original_redis = sys.modules.get("redis")
        sys.modules["redis"] = None

        try:
            with pytest.raises(RuntimeError, match="redis package not installed"):
                JobManager()
        finally:
            if original_redis is not None:
                sys.modules["redis"] = original_redis
            elif "redis" in sys.modules:
                del sys.modules["redis"]

    def test_constants_defined(self):
        """Should have proper Redis key prefixes and TTLs."""
        assert JobManager.QUEUE_PREFIX == "trans:queue:"
        assert JobManager.JOB_PREFIX == "trans:job:"
        assert JobManager.STATE_PREFIX == "trans:state:"
        assert JobManager.CHECKPOINT_PREFIX == "trans:checkpoint:"
        assert JobManager.JOB_TTL == 86400
        assert JobManager.STATE_TTL == 86400
        assert JobManager.CHECKPOINT_TTL == 604800

    def test_priorities_defined(self):
        """Should have proper priority levels."""
        assert JobManager.PRIORITIES == {"urgent": 0, "normal": 1, "bulk": 2}


class TestEnqueue:
    """Test job enqueue operations."""

    def test_enqueue_basic(self, manager, mock_redis):
        """Should enqueue a job and return job ID."""
        mock_pipeline = MagicMock()
        mock_pipeline.execute.return_value = None
        mock_redis.pipeline.return_value = mock_pipeline

        job_id = manager.enqueue({"source_file": "/path/to/file.pptx"})

        assert job_id is not None
        assert len(job_id) == 36  # UUID format

        # Verify Redis operations
        assert mock_pipeline.set.call_count == 2  # job + state
        assert mock_pipeline.zadd.call_count == 1
        assert mock_pipeline.execute.call_count == 1

    def test_enqueue_with_priority(self, manager, mock_redis):
        """Should respect priority parameter."""
        mock_pipeline = MagicMock()
        mock_pipeline.execute.return_value = None
        mock_redis.pipeline.return_value = mock_pipeline

        manager.enqueue({"source_file": "file.pptx"}, priority="urgent")

        # Check that zadd was called with urgent queue
        call_args = mock_pipeline.zadd.call_args
        queue_key = call_args[0][0]
        assert "urgent" in queue_key

    def test_enqueue_invalid_priority_defaults_to_normal(self, manager, mock_redis):
        """Should default to normal priority for invalid values."""
        mock_pipeline = MagicMock()
        mock_pipeline.execute.return_value = None
        mock_redis.pipeline.return_value = mock_pipeline

        manager.enqueue({"source_file": "file.pptx"}, priority="invalid")

        call_args = mock_pipeline.zadd.call_args
        queue_key = call_args[0][0]
        assert "normal" in queue_key

    def test_enqueue_with_delay(self, manager, mock_redis):
        """Should add delay to score for delayed jobs."""
        mock_pipeline = MagicMock()
        mock_pipeline.execute.return_value = None
        mock_redis.pipeline.return_value = mock_pipeline

        manager.enqueue({"source_file": "file.pptx"}, delay_seconds=60)

        call_args = mock_pipeline.zadd.call_args
        score = list(call_args[0][1].values())[0]
        # Score should be current time + delay
        assert score > time.time()

    def test_enqueue_with_metadata(self, manager, mock_redis):
        """Should store job metadata."""
        mock_pipeline = MagicMock()
        mock_pipeline.execute.return_value = None
        mock_redis.pipeline.return_value = mock_pipeline

        job_data = {
            "source_file": "/path/to/file.pptx",
            "source_lang": "ja",
            "target_lang": "en",
        }
        job_id = manager.enqueue(job_data)

        # Verify job data was stored
        set_call = mock_pipeline.set.call_args_list[0]
        stored_data = json.loads(set_call[0][1])
        assert stored_data["source_file"] == "/path/to/file.pptx"
        assert stored_data["source_lang"] == "ja"
        assert stored_data["target_lang"] == "en"

    def test_enqueue_redis_failure(self, manager, mock_redis):
        """Should return None on Redis failure."""
        mock_pipeline = MagicMock()
        mock_pipeline.execute.side_effect = Exception("Redis error")
        mock_redis.pipeline.return_value = mock_pipeline

        result = manager.enqueue({"source_file": "file.pptx"})

        assert result is None


class TestDequeue:
    """Test job dequeue operations."""

    def test_dequeue_basic(self, manager, mock_redis):
        """Should dequeue and return job data."""
        # Mock queue having a job
        mock_redis.zpopmin.return_value = ("test-job-id", 123.0)

        # Mock job data retrieval
        job_data = {
            "id": "test-job-id",
            "source_file": "/path/to/file.pptx",
            "state": "pending",
            "created_at": "2025-01-12T00:00:00",
        }
        mock_redis.get.return_value = json.dumps(job_data).encode()

        result = manager.dequeue(worker_id="worker-1")

        assert result is not None
        assert result["id"] == "test-job-id"
        assert result["worker_id"] == "worker-1"
        assert result["started_at"] is not None

    def test_dequeue_empty_queue(self, manager, mock_redis):
        """Should return None when queue is empty."""
        mock_redis.zpopmin.return_value = None

        result = manager.dequeue(worker_id="worker-1")

        assert result is None

    def test_dequeue_by_priority(self, manager, mock_redis):
        """Should check queues in priority order."""
        # urgent queue empty, normal has job
        call_count = [0]

        def zpopmin_side_effect(key):
            call_count[0] += 1
            if "urgent" in key:
                return None
            elif "normal" in key:
                return ("job-id", 100.0)
            return None

        mock_redis.zpopmin.side_effect = zpopmin_side_effect
        mock_redis.get.return_value = json.dumps(
            {
                "id": "job-id",
                "source_file": "file.pptx",
            }
        ).encode()

        result = manager.dequeue(worker_id="worker-1", priorities=["urgent", "normal"])

        assert result is not None
        assert call_count[0] == 2  # Checked urgent then normal

    def test_dequeue_custom_priorities(self, manager, mock_redis):
        """Should respect custom priority list."""
        mock_redis.zpopmin.return_value = ("job-id", 100.0)
        mock_redis.get.return_value = json.dumps({"id": "job-id"}).encode()

        manager.dequeue(worker_id="worker-1", priorities=["bulk", "normal"])

        # Should have checked bulk first
        call_key = mock_redis.zpopmin.call_args[0][0]
        assert "bulk" in call_key

    def test_dequeue_missing_job_data(self, manager, mock_redis):
        """Should return None when job data missing."""
        mock_redis.zpopmin.return_value = ("job-id", 100.0)
        mock_redis.get.return_value = None

        result = manager.dequeue(worker_id="worker-1")

        assert result is None


class TestSetState:
    """Test job state management."""

    def test_set_state_processing(self, manager, mock_redis):
        """Should update job to processing state."""
        mock_redis.get.return_value = json.dumps(
            {
                "id": "job-id",
                "source_file": "file.pptx",
            }
        ).encode()

        result = manager.set_state("job-id", JobState.PROCESSING, worker_id="worker-1")

        assert result is True

    def test_set_state_completed(self, manager, mock_redis):
        """Should set completion timestamp for terminal states."""
        mock_redis.get.return_value = json.dumps(
            {
                "id": "job-id",
                "source_file": "file.pptx",
            }
        ).encode()

        manager.set_state("job-id", JobState.COMPLETED)

        # Verify completed_at was set in job data
        found_completed_at = False
        for call in mock_redis.set.call_args_list:
            call_data = call[0][1]  # The value argument
            try:
                data = json.loads(call_data)
                if "completed_at" in data and data.get("state") == "completed":
                    found_completed_at = True
                    break
            except (json.JSONDecodeError, TypeError):
                continue

        assert found_completed_at, "completed_at not found in job data"

    def test_set_state_failed(self, manager, mock_redis):
        """Should set completion timestamp for failed state."""
        mock_redis.get.return_value = json.dumps(
            {
                "id": "job-id",
                "source_file": "file.pptx",
            }
        ).encode()

        manager.set_state("job-id", JobState.FAILED)

        # Verify failed state and completed_at
        found_failed = False
        for call in mock_redis.set.call_args_list:
            call_data = call[0][1]
            try:
                data = json.loads(call_data)
                if data.get("state") == "failed" and "completed_at" in data:
                    found_failed = True
                    break
            except (json.JSONDecodeError, TypeError):
                continue

        assert found_failed, "Failed state with completed_at not found"

    def test_set_state_with_worker(self, manager, mock_redis):
        """Should associate worker with state."""
        mock_redis.get.return_value = json.dumps(
            {
                "id": "job-id",
                "source_file": "file.pptx",
            }
        ).encode()

        manager.set_state("job-id", JobState.PROCESSING, worker_id="worker-1")

        state_call = mock_redis.set.call_args_list[0]
        state_data = json.loads(state_call[0][1])
        assert state_data["worker_id"] == "worker-1"

    def test_set_state_redis_error(self, manager, mock_redis):
        """Should return False on Redis error."""
        mock_redis.get.side_effect = Exception("Redis error")

        result = manager.set_state("job-id", JobState.PROCESSING)

        assert result is False


class TestGetState:
    """Test job state retrieval."""

    def test_get_state_pending(self, manager, mock_redis):
        """Should retrieve pending state."""
        mock_redis.get.return_value = json.dumps(
            {
                "state": "pending",
                "worker_id": None,
                "timestamp": "2025-01-12T00:00:00",
            }
        ).encode()

        result = manager.get_state("job-id")

        assert result == JobState.PENDING

    def test_get_state_processing(self, manager, mock_redis):
        """Should retrieve processing state."""
        mock_redis.get.return_value = json.dumps(
            {
                "state": "processing",
                "worker_id": "worker-1",
                "timestamp": "2025-01-12T00:00:00",
            }
        ).encode()

        result = manager.get_state("job-id")

        assert result == JobState.PROCESSING

    def test_get_state_not_found(self, manager, mock_redis):
        """Should return None for non-existent job."""
        mock_redis.get.return_value = None

        result = manager.get_state("nonexistent")

        assert result is None

    def test_get_state_invalid_value(self, manager, mock_redis):
        """Should default to PENDING for invalid state values."""
        mock_redis.get.return_value = json.dumps(
            {
                "state": "invalid_state",
                "worker_id": None,
                "timestamp": "2025-01-12T00:00:00",
            }
        ).encode()

        result = manager.get_state("job-id")

        assert result == JobState.PENDING


class TestGetJob:
    """Test job data retrieval."""

    def test_get_job_basic(self, manager, mock_redis):
        """Should retrieve full job data."""
        job_data = {
            "id": "job-id",
            "source_file": "/path/to/file.pptx",
            "state": "processing",
            "worker_id": "worker-1",
            "progress": 0.5,
        }
        mock_redis.get.return_value = json.dumps(job_data).encode()

        result = manager.get_job("job-id")

        assert result["id"] == "job-id"
        assert result["source_file"] == "/path/to/file.pptx"
        assert result["progress"] == 0.5

    def test_get_job_not_found(self, manager, mock_redis):
        """Should return None for non-existent job."""
        mock_redis.get.return_value = None

        result = manager.get_job("nonexistent")

        assert result is None


class TestCheckpoint:
    """Test checkpoint save/load operations."""

    def test_save_checkpoint_basic(self, manager, mock_redis):
        """Should save checkpoint data."""
        mock_redis.get.return_value = json.dumps({"id": "job-id"}).encode()

        result = manager.save_checkpoint(
            "job-id",
            {"segments": ["segment1", "segment2"]},
            progress=0.5,
        )

        assert result is not None
        assert len(result) == 36  # UUID format

        # Verify checkpoint was saved
        checkpoint_call = mock_redis.setex.call_args
        key = checkpoint_call[0][0]
        assert "checkpoint" in key

    def test_save_checkpoint_with_source_hash(self, manager, mock_redis):
        """Should save source hash for change detection."""
        mock_redis.get.return_value = json.dumps({"id": "job-id"}).encode()

        manager.save_checkpoint(
            "job-id",
            {"segments": []},
            source_hash="abc123",
        )

        checkpoint_call = mock_redis.setex.call_args
        # setex signature: (key, seconds, value) - value is at index 2
        data = json.loads(checkpoint_call[0][2])
        assert data["source_hash"] == "abc123"

    def test_save_checkpoint_updates_progress(self, manager, mock_redis):
        """Should update job progress when saving checkpoint."""
        # Configure get to return job data (called twice: for checkpoint_id update and progress update)
        mock_redis.get.return_value = json.dumps(
            {
                "id": "job-id",
                "progress": 0.0,
            }
        ).encode()

        manager.save_checkpoint("job-id", {"data": "value"}, progress=0.75)

        # Verify progress was updated in job data
        # Find the set call that updates the job with progress
        found_progress_update = False
        for call in mock_redis.set.call_args_list:
            call_data = call[0][1]  # The value argument
            try:
                data = json.loads(call_data)
                if "progress" in data and data["progress"] == 0.75:
                    found_progress_update = True
                    break
            except (json.JSONDecodeError, TypeError):
                # Skip non-JSON calls (like state updates)
                continue

        assert found_progress_update, "Progress update not found in set calls"

    def test_save_checkpoint_references_job(self, manager, mock_redis):
        """Should add checkpoint_id to job data."""
        mock_redis.get.return_value = json.dumps({"id": "job-id"}).encode()

        checkpoint_id = manager.save_checkpoint("job-id", {"data": "value"})

        # Find the set call that updates the job with checkpoint_id
        found_checkpoint_ref = False
        for call in mock_redis.set.call_args_list:
            call_data = call[0][1]  # The value argument
            try:
                data = json.loads(call_data)
                if "checkpoint_id" in data and data["checkpoint_id"] == checkpoint_id:
                    found_checkpoint_ref = True
                    break
            except (json.JSONDecodeError, TypeError):
                # Skip non-JSON calls
                continue

        assert found_checkpoint_ref, "Checkpoint ID reference not found in set calls"

    def test_load_checkpoint(self, manager, mock_redis):
        """Should load checkpoint data."""
        checkpoint_data = {
            "job_id": "job-id",
            "checkpoint_id": "ckpt-123",
            "timestamp": "2025-01-12T00:00:00",
            "progress": 0.5,
            "data": {"segments": ["seg1", "seg2"]},
            "source_hash": "hash123",
        }
        mock_redis.get.return_value = json.dumps(checkpoint_data).encode()

        result = manager.load_checkpoint("job-id")

        assert result is not None
        assert result["checkpoint_id"] == "ckpt-123"
        assert result["progress"] == 0.5
        assert result["data"]["segments"] == ["seg1", "seg2"]

    def test_load_checkpoint_not_found(self, manager, mock_redis):
        """Should return None for non-existent checkpoint."""
        mock_redis.get.return_value = None

        result = manager.load_checkpoint("nonexistent")

        assert result is None


class TestProgress:
    """Test progress tracking and publishing."""

    def test_update_progress(self, manager, mock_redis):
        """Should update job progress."""
        mock_redis.get.return_value = json.dumps(
            {
                "id": "job-id",
                "progress": 0.0,
            }
        ).encode()

        result = manager.update_progress("job-id", 0.5)

        assert result is True

        # Verify progress was clamped and saved
        job_update = mock_redis.set.call_args
        job_data = json.loads(job_update[0][1])
        assert job_data["progress"] == 0.5

    def test_update_progress_clamps_max(self, manager, mock_redis):
        """Should clamp progress to maximum 1.0."""
        mock_redis.get.return_value = json.dumps({"id": "job-id"}).encode()

        manager.update_progress("job-id", 1.5)

        job_data = json.loads(mock_redis.set.call_args[0][1])
        assert job_data["progress"] == 1.0

    def test_update_progress_clamps_min(self, manager, mock_redis):
        """Should clamp progress to minimum 0.0."""
        mock_redis.get.return_value = json.dumps({"id": "job-id"}).encode()

        manager.update_progress("job-id", -0.5)

        job_data = json.loads(mock_redis.set.call_args[0][1])
        assert job_data["progress"] == 0.0

    def test_update_progress_with_message(self, manager, mock_redis):
        """Should include progress message."""
        mock_redis.get.return_value = json.dumps({"id": "job-id"}).encode()

        manager.update_progress("job-id", 0.5, message="Processing slide 3")

        # Should have called publish_progress
        assert mock_redis.publish.call_count == 1

    def test_publish_progress(self, manager, mock_redis):
        """Should publish progress to pub/sub."""
        result = manager.publish_progress("job-id", 0.5, "Test message")

        assert result is True

        publish_call = mock_redis.publish.call_args
        channel = publish_call[0][0]
        assert channel == "translation:progress"

        data = json.loads(publish_call[0][1])
        assert data["job_id"] == "job-id"
        assert data["progress"] == 0.5
        assert data["message"] == "Test message"

    def test_publish_progress_error(self, manager, mock_redis):
        """Should return False on publish error."""
        mock_redis.publish.side_effect = Exception("Redis error")

        result = manager.publish_progress("job-id", 0.5)

        assert result is False


class TestJobControl:
    """Test job control operations."""

    def test_cancel_job(self, manager, mock_redis):
        """Should cancel a job."""
        mock_redis.get.return_value = json.dumps(
            {
                "id": "job-id",
                "source_file": "file.pptx",
            }
        ).encode()

        result = manager.cancel_job("job-id", reason="User requested")

        assert result is True

        # Verify state was set to cancelled
        state_call = mock_redis.set.call_args_list[0]
        state_data = json.loads(state_call[0][1])
        assert state_data["state"] == "cancelled"

    def test_fail_job(self, manager, mock_redis):
        """Should mark job as failed with error."""
        mock_redis.get.return_value = json.dumps(
            {
                "id": "job-id",
                "source_file": "file.pptx",
            }
        ).encode()

        result = manager.fail_job("job-id", "Translation service error")

        assert result is True

        # Verify error was stored in one of the set calls
        found_error = False
        for call in mock_redis.set.call_args_list:
            call_data = call[0][1]  # The value argument
            try:
                data = json.loads(call_data)
                if data.get("error") == "Translation service error":
                    found_error = True
                    break
            except (json.JSONDecodeError, TypeError):
                # Skip non-JSON calls (like state updates)
                continue

        assert found_error, "Error not found in any set call"

    def test_fail_job_state(self, manager, mock_redis):
        """Should set state to failed."""
        mock_redis.get.return_value = json.dumps(
            {
                "id": "job-id",
                "source_file": "file.pptx",
            }
        ).encode()

        manager.fail_job("job-id", "Error")

        # Check state update
        state_call = mock_redis.set.call_args_list[0]
        state_data = json.loads(state_call[0][1])
        assert state_data["state"] == "failed"


class TestWorkerJobs:
    """Test worker job retrieval."""

    def test_get_worker_jobs_empty(self, manager, mock_redis):
        """Should return empty list for new worker."""
        mock_redis.scan_iter.return_value = []

        result = manager.get_worker_jobs("worker-1")

        assert result == []

    def test_get_worker_jobs_filters_by_worker(self, manager, mock_redis):
        """Should return only jobs assigned to worker."""
        job1 = json.dumps(
            {
                "id": "job-1",
                "worker_id": "worker-1",
                "created_at": "2025-01-12T00:00:00",
            }
        )
        job2 = json.dumps(
            {
                "id": "job-2",
                "worker_id": "worker-2",
                "created_at": "2025-01-12T00:00:00",
            }
        )

        mock_redis.scan_iter.return_value = [
            b"trans:job:job-1",
            b"trans:job:job-2",
        ]
        mock_redis.get.side_effect = [
            job1.encode(),
            job2.encode(),
        ]

        result = manager.get_worker_jobs("worker-1")

        assert len(result) == 1
        assert result[0]["id"] == "job-1"

    def test_get_worker_jobs_sorted(self, manager, mock_redis):
        """Should return jobs sorted by created_at."""
        job2 = json.dumps(
            {
                "id": "job-2",
                "worker_id": "worker-1",
                "created_at": "2025-01-12T00:01:00",
            }
        )
        job1 = json.dumps(
            {
                "id": "job-1",
                "worker_id": "worker-1",
                "created_at": "2025-01-12T00:00:00",
            }
        )

        mock_redis.scan_iter.return_value = [
            b"trans:job:job-2",
            b"trans:job:job-1",
        ]
        mock_redis.get.side_effect = [
            job2.encode(),
            job1.encode(),
        ]

        result = manager.get_worker_jobs("worker-1")

        assert result[0]["id"] == "job-1"  # Earlier first
        assert result[1]["id"] == "job-2"


class TestQueueStats:
    """Test queue statistics."""

    def test_get_queue_stats_empty(self, manager, mock_redis):
        """Should return zeros for empty queues."""
        mock_redis.zcard.return_value = 0

        stats = manager.get_queue_stats()

        assert stats == {"urgent": 0, "normal": 0, "bulk": 0, "total": 0}

    def test_get_queue_stats_with_jobs(self, manager, mock_redis):
        """Should count jobs in each priority queue."""

        def zcard_side_effect(key):
            if "urgent" in key:
                return 5
            elif "normal" in key:
                return 10
            elif "bulk" in key:
                return 3
            return 0

        mock_redis.zcard.side_effect = zcard_side_effect

        stats = manager.get_queue_stats()

        assert stats["urgent"] == 5
        assert stats["normal"] == 10
        assert stats["bulk"] == 3
        assert stats["total"] == 18


class TestCleanup:
    """Test old job cleanup."""

    def test_cleanup_old_jobs(self, manager, mock_redis):
        """Should remove jobs older than max age."""
        from datetime import datetime, timedelta, timezone

        old_time = (datetime.now(timezone.utc) - timedelta(days=2)).isoformat()
        recent_time = datetime.now(timezone.utc).isoformat()

        old_job = json.dumps(
            {
                "id": "old-job",
                "completed_at": old_time,
            }
        )
        recent_job = json.dumps(
            {
                "id": "recent-job",
                "completed_at": recent_time,
            }
        )

        mock_redis.scan_iter.return_value = [
            b"trans:job:old-job",
            b"trans:job:recent-job",
        ]
        mock_redis.get.side_effect = [
            old_job.encode(),
            recent_job.encode(),
        ]

        cleaned = manager.cleanup_old_jobs(max_age_seconds=86400)

        assert cleaned == 1
        assert mock_redis.delete.call_count == 3  # job + state + checkpoint

    def test_cleanup_incomplete_jobs_not_removed(self, manager, mock_redis):
        """Should not remove jobs without completed_at."""
        job = json.dumps(
            {
                "id": "job-id",
                "state": "processing",
            }
        )

        mock_redis.scan_iter.return_value = [b"trans:job:job-id"]
        mock_redis.get.return_value = job.encode()

        cleaned = manager.cleanup_old_jobs()

        assert cleaned == 0

    def test_cleanup_handles_invalid_timestamps(self, manager, mock_redis):
        """Should handle invalid timestamps gracefully."""
        job = json.dumps(
            {
                "id": "job-id",
                "completed_at": "invalid-timestamp",
            }
        )

        mock_redis.scan_iter.return_value = [b"trans:job:job-id"]
        mock_redis.get.return_value = job.encode()

        cleaned = manager.cleanup_old_jobs()

        assert cleaned == 0


class TestContextManager:
    """Test context manager support."""

    def test_close(self, manager, mock_redis):
        """Should close Redis connection."""
        manager.close()

        mock_redis.close.assert_called_once()

    def test_context_manager_enter(self, manager):
        """Should return self on enter."""
        result = manager.__enter__()

        assert result is manager

    def test_context_manager_exit(self, manager, mock_redis):
        """Should close on exit."""
        manager.__exit__(None, None, None)

        mock_redis.close.assert_called_once()


class TestFactoryFunction:
    """Test factory function."""

    def test_create_job_manager_default(self):
        """Should create manager with defaults."""
        import sys

        mock_redis_module = MagicMock()
        mock_redis_module.Redis = MagicMock()
        original_redis = sys.modules.get("redis")
        sys.modules["redis"] = mock_redis_module

        try:
            manager = create_job_manager()
            assert isinstance(manager, JobManager)
        finally:
            if original_redis is not None:
                sys.modules["redis"] = original_redis
            elif "redis" in sys.modules:
                del sys.modules["redis"]

    def test_create_job_manager_custom(self):
        """Should create manager with custom settings."""
        import sys

        mock_redis_module = MagicMock()
        mock_redis_module.Redis = MagicMock()
        original_redis = sys.modules.get("redis")
        sys.modules["redis"] = mock_redis_module

        try:
            create_job_manager(
                redis_host="custom",
                redis_port=6380,
                redis_db=2,
            )

            mock_redis_module.Redis.assert_called_once_with(
                host="custom",
                port=6380,
                db=2,
                password=None,
                decode_responses=False,
                socket_connect_timeout=5,
                socket_timeout=5,
            )
        finally:
            if original_redis is not None:
                sys.modules["redis"] = original_redis
            elif "redis" in sys.modules:
                del sys.modules["redis"]


class TestTerminalStates:
    """Test terminal state detection."""

    def test_completed_is_terminal(self):
        """Should mark COMPLETED as terminal."""
        # TERMINAL is an enum member with value being a set of state strings
        assert JobState.COMPLETED.value in JobState.TERMINAL.value

    def test_failed_is_terminal(self):
        """Should mark FAILED as terminal."""
        assert JobState.FAILED.value in JobState.TERMINAL.value

    def test_cancelled_is_terminal(self):
        """Should mark CANCELLED as terminal."""
        assert JobState.CANCELLED.value in JobState.TERMINAL.value

    def test_processing_not_terminal(self):
        """Should not mark PROCESSING as terminal."""
        assert JobState.PROCESSING.value not in JobState.TERMINAL.value


class TestEncoding:
    """Test data encoding/decoding."""

    def test_encode_creates_bytes(self, manager):
        """Should encode data to bytes."""
        data = {"key": "value"}
        result = manager._encode(data)

        assert isinstance(result, bytes)

    def test_decode_creates_dict(self, manager):
        """Should decode bytes to dict."""
        data = b'{"key": "value"}'
        result = manager._decode(data)

        assert isinstance(result, dict)
        assert result["key"] == "value"


class TestEdgeCases:
    """Test edge cases and error handling."""

    def test_enqueue_empty_job_data(self, manager, mock_redis):
        """Should handle empty job data."""
        mock_pipeline = MagicMock()
        mock_pipeline.execute.return_value = None
        mock_redis.pipeline.return_value = mock_pipeline

        job_id = manager.enqueue({})

        assert job_id is not None

    def test_set_state_nonexistent_job(self, manager, mock_redis):
        """Should handle state change for non-existent job."""
        mock_redis.get.return_value = None

        result = manager.set_state("nonexistent", JobState.PROCESSING)

        # Should succeed (only state key is set)
        assert result is True

    def test_update_progress_nonexistent_job(self, manager, mock_redis):
        """Should handle progress update for non-existent job."""
        mock_redis.get.return_value = None

        result = manager.update_progress("nonexistent", 0.5)

        assert result is True  # Still publishes progress
