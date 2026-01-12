# Job Queue API Reference

Complete API reference for the Redis-backed job queue system.

## Overview

The `JobManager` class provides a distributed job queue with priority support, state management, checkpoint/resume for fault tolerance, and progress publishing via pub/sub.

## Initialization

```python
from job_queue import JobManager

manager = JobManager(
    redis_host="localhost",
    redis_port=6379,
    redis_db=0,
    redis_password=None,  # Optional
    decode_responses=False
)
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `redis_host` | str | `"localhost"` | Redis server host |
| `redis_port` | int | `6379` | Redis server port |
| `redis_db` | int | `0` | Redis database number |
| `redis_password` | str | `None` | Optional Redis password |
| `decode_responses` | bool | `False` | Whether to decode responses (False for binary) |

## Methods

### `enqueue()` - Add a Job to Queue

Adds a translation job to the priority queue.

```python
job_id = manager.enqueue(
    job_data={
        "source_file": "/path/to/document.docx",
        "source_lang": "ja",
        "target_lang": "en",
        "glossary_id": "terms-2024"
    },
    priority="normal",
    delay_seconds=0
)
```

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `job_data` | dict | required | Job metadata (source_file, langs, etc.) |
| `priority` | str | `"normal"` | Queue priority: `"urgent"`, `"normal"`, `"bulk"` |
| `delay_seconds` | int | `0` | Delay before job becomes available |

#### Returns

- `str`: Job ID (UUID) if successful, `None` otherwise

#### Priority Levels

| Priority | Score | Use Case |
|----------|-------|----------|
| `urgent` | 0 | Critical translations, VIP clients |
| `normal` | 1 | Standard translations |
| `bulk` | 2 | Batch/offline processing |

---

### `dequeue()` - Get Next Job

Retrieves the highest priority available job and assigns it to a worker.

```python
job = manager.dequeue(
    worker_id="worker-1",
    timeout=1,
    priorities=None
)
```

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `worker_id` | str | required | ID of worker requesting job |
| `timeout` | int | `1` | Seconds to wait (blocking not implemented) |
| `priorities` | list[str] | `None` | Priorities to check (default: all) |

#### Returns

- `dict | None`: Job data if available, `None` otherwise

#### Job Data Structure

```python
{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "source_file": "/path/to/document.docx",
    "state": "processing",
    "worker_id": "worker-1",
    "created_at": "2026-01-12T10:00:00Z",
    "started_at": "2026-01-12T10:00:05Z"
}
```

---

### `set_state()` - Update Job State

Changes the state of a job and updates timestamps.

```python
success = manager.set_state(
    job_id="550e8400-e29b-41d4-a716-446655440000",
    state=JobState.TRANSLATING,
    worker_id="worker-1"
)
```

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `job_id` | str | required | Job identifier |
| `state` | JobState | required | New state |
| `worker_id` | str | `None` | Optional worker ID to associate |

#### Returns

- `bool`: `True` if state updated successfully, `False` otherwise

#### Job States

```python
class JobState(Enum):
    PENDING = "pending"       # In queue, not started
    PROCESSING = "processing" # Worker assigned
    TRANSLATING = "translating" # Active translation
    COMPLETED = "completed"   # Finished successfully
    FAILED = "failed"         # Errored out
    CANCELLED = "cancelled"   # Cancelled by user
```

Terminal states: `COMPLETED`, `FAILED`, `CANCELLED`

---

### `get_state()` - Get Job State

Retrieves the current state of a job.

```python
state = manager.get_state(job_id="550e8400-...")
```

#### Returns

- `JobState | None`: Current state if found, `None` otherwise

---

### `get_job()` - Get Full Job Data

Retrieves complete job data including metadata.

```python
job = manager.get_job(job_id="550e8400-...")
```

#### Returns

- `dict | None`: Job data if found, `None` otherwise

---

### `save_checkpoint()` - Save Progress Checkpoint

Saves job progress for fault tolerance and resume capability.

```python
checkpoint_id = manager.save_checkpoint(
    job_id="550e8400-...",
    checkpoint_data={
        "segments_completed": 15,
        "total_segments": 50,
        "last_segment_hash": "abc123"
    },
    progress=0.3,
    source_hash="sha256:..."
)
```

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `job_id` | str | required | Job identifier |
| `checkpoint_data` | dict | required | Arbitrary checkpoint data |
| `progress` | float | `None` | Optional progress (0.0 to 1.0) |
| `source_hash` | str | `None` | Optional hash for change detection |

#### Returns

- `str | None`: Checkpoint ID if saved, `None` otherwise

#### TTL

- Checkpoints expire after **7 days** (`CHECKPOINT_TTL = 604800` seconds)

---

### `load_checkpoint()` - Load Checkpoint

Retrieves saved checkpoint for resuming a job.

```python
checkpoint = manager.load_checkpoint(job_id="550e8400-...")
```

#### Returns

- `dict | None`: Checkpoint data if found, `None` otherwise

---

### `update_progress()` - Update Job Progress

Updates job progress and publishes to pub/sub.

```python
success = manager.update_progress(
    job_id="550e8400-...",
    progress=0.75,
    message="Translating slide 15 of 20"
)
```

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `job_id` | str | required | Job identifier |
| `progress` | float | required | Progress (0.0 to 1.0, auto-clamped) |
| `message` | str | `""` | Optional progress message |

#### Returns

- `bool`: `True` if updated successfully, `False` otherwise

---

### `publish_progress()` - Publish Progress Event

Publishes progress to Redis pub/sub for real-time updates.

```python
success = manager.publish_progress(
    job_id="550e8400-...",
    progress=0.5,
    message="Halfway through translation"
)
```

#### Pub/Sub Channel

- Channel: `translation:progress`
- Message format: JSON with `job_id`, `progress`, `message`, `timestamp`

#### Subscribing to Progress

```python
import redis
import json

r = redis.Redis(host="localhost", port=6379, db=0)
pubsub = r.pubsub()
pubsub.subscribe("translation:progress")

for message in pubsub.listen():
    if message["type"] == "message":
        data = json.loads(message["data"])
        print(f"{data['job_id']}: {data['progress']*100}% - {data['message']}")
```

---

### `cancel_job()` - Cancel a Job

Marks a job as cancelled.

```python
success = manager.cancel_job(
    job_id="550e8400-...",
    reason="User requested cancellation"
)
```

#### Returns

- `bool`: `True` if cancelled successfully, `False` otherwise

---

### `fail_job()` - Mark Job as Failed

Marks a job as failed with an error message.

```python
success = manager.fail_job(
    job_id="550e8400-...",
    error="Translation failed: API rate limit exceeded"
)
```

#### Returns

- `bool`: `True` if marked failed successfully, `False` otherwise

---

### `get_worker_jobs()` - Get Worker's Jobs

Retrieves all jobs assigned to a specific worker.

```python
jobs = manager.get_worker_jobs(worker_id="worker-1")
```

#### Returns

- `list[dict]`: List of job data dicts assigned to worker

---

### `get_queue_stats()` - Get Queue Statistics

Returns current queue sizes by priority.

```python
stats = manager.get_queue_stats()
# {"urgent": 0, "normal": 5, "bulk": 12, "total": 17}
```

#### Returns

- `dict`: Queue statistics

---

### `cleanup_old_jobs()` - Remove Old Jobs

Removes completed/failed jobs older than specified age.

```python
count = manager.cleanup_old_jobs(max_age_seconds=86400)
```

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `max_age_seconds` | int | `86400` | Maximum age (default: 24 hours) |

#### Returns

- `int`: Number of jobs cleaned up

---

### `close()` - Close Connection

Closes the Redis connection.

```python
manager.close()
```

#### Context Manager

The `JobManager` supports context manager protocol:

```python
with JobManager(redis_host="localhost") as manager:
    job_id = manager.enqueue(job_data)
    # Connection auto-closes on exit
```

## Redis Key Structure

| Key Pattern | Type | TTL | Purpose |
|-------------|------|-----|---------|
| `trans:queue:{priority}` | sorted set | ∞ | Priority queue |
| `trans:job:{job_id}` | string | 24h | Job data |
| `trans:state:{job_id}` | string | 24h | Job state |
| `trans:checkpoint:{job_id}` | string | 7 days | Checkpoint data |

## TTL Settings

| Data Type | TTL | Constant |
|-----------|-----|----------|
| Job data | 24 hours | `JOB_TTL = 86400` |
| Job state | 24 hours | `STATE_TTL = 86400` |
| Checkpoint | 7 days | `CHECKPOINT_TTL = 604800` |

## Factory Function

Create a pre-configured `JobManager`:

```python
from job_queue import create_job_manager

manager = create_job_manager(
    redis_host="localhost",
    redis_port=6379,
    redis_db=0
)
```

## Usage Examples

### Basic Workflow

```python
from job_queue import JobManager, JobState

# Initialize
manager = JobManager(redis_host="localhost")

# Enqueue a job
job_id = manager.enqueue({
    "source_file": "/data/document.docx",
    "source_lang": "ja",
    "target_lang": "en"
}, priority="normal")

# Worker dequeues
job = manager.dequeue(worker_id="worker-1")

# Update state
manager.set_state(job["id"], JobState.TRANSLATING, "worker-1")

# Save checkpoint
manager.save_checkpoint(
    job["id"],
    {"completed": 10, "total": 50},
    progress=0.2
)

# Complete job
manager.set_state(job["id"], JobState.COMPLETED)
manager.update_progress(job["id"], 1.0, "Translation complete")

# Cleanup
manager.close()
```

### Progress Monitoring

```python
import redis
import json

r = redis.Redis()
pubsub = r.pubsub()
pubsub.subscribe("translation:progress")

for message in pubsub.listen():
    if message["type"] == "message":
        data = json.loads(message["data"])
        print(f"Job {data['job_id']}: {data['progress']*100:.1f}%")
        if data['progress'] >= 1.0:
            print("  Complete!")
```

### Fault Tolerance with Checkpoints

```python
# On worker startup, check for existing checkpoint
checkpoint = manager.load_checkpoint(job_id)
if checkpoint:
    # Resume from checkpoint
    start_segment = checkpoint.get("segments_completed", 0)
    print(f"Resuming from segment {start_segment}")
else:
    # Start fresh
    start_segment = 0

# Process and periodically save checkpoints
for i in range(start_segment, total_segments):
    # Translate segment
    translate_segment(i)

    # Save checkpoint every 5 segments
    if i % 5 == 0:
        manager.save_checkpoint(
            job_id,
            {"segments_completed": i},
            progress=i/total_segments
        )
```

## Error Handling

The `JobManager` methods handle Redis errors gracefully:

```python
# enqueue returns None on failure
job_id = manager.enqueue(job_data)
if job_id is None:
    print("Failed to enqueue job")

# Most methods return bool for success/failure
if not manager.set_state(job_id, JobState.COMPLETED):
    print("Failed to update state")
```
