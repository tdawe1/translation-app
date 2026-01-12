# Translation Worker Troubleshooting Guide

Common issues and solutions for the translation worker system.

`★ Insight ─────────────────────────────────────`
**Debugging Philosophy**: The translation worker is designed with observability in mind.
Every job state transition, progress update, and error is logged. Enable debug logging early when troubleshooting.
`─────────────────────────────────────────────────`

## Table of Contents

- [Installation Issues](#installation-issues)
- [Configuration Problems](#configuration-problems)
- [Redis Connection Issues](#redis-connection-issues)
- [LLM Provider Errors](#llm-provider-errors)
- [Job Queue Problems](#job-queue-problems)
- [Document Parsing Failures](#document-parsing-failures)
- [Performance Issues](#performance-issues)
- [Debug Mode](#debug-mode)

---

## Installation Issues

### Missing Dependencies

**Symptom**: `ModuleNotFoundError: No module named 'anthropic'`

**Solution**:
```bash
cd backend/cmd/translation-worker
pip install anthropic openai requests redis click
```

Or install from requirements file:
```bash
pip install -r requirements.txt
```

### Python Version Incompatibility

**Symptom**: `SyntaxError` or import errors

**Solution**: The translation worker requires Python 3.11+
```bash
python --version  # Should be 3.11+
```

---

## Configuration Problems

### Config File Not Found

**Symptom**: `FileNotFoundError: Config file not found: config.toml`

**Solutions**:
1. Create `config.toml` in the translation-worker directory
2. Specify path explicitly: `python main.py --config /path/to/config.toml`
3. Use absolute path in config option

### Missing Required Sections

**Symptom**: `Configuration errors: Missing required section: [worker]`

**Solution**: Ensure your `config.toml` has all required sections:
```toml
[worker]
id = "worker-1"
max_concurrent = 3
heartbeat_interval = "30s"

[translation]
default_provider = "anthropic"
default_model = "claude-sonnet-4-5-20250929"
```

### Invalid Duration Format

**Symptom**: `ValueError: Invalid duration format: 30`

**Solution**: Duration strings must end with `s`, `m`, or `h`:
```toml
# Correct
heartbeat_interval = "30s"
poll_interval = "1m"

# Incorrect
heartbeat_interval = 30
```

---

## Redis Connection Issues

### Connection Refused

**Symptom**: `Error 111 connecting to localhost:6379: Connection refused`

**Solutions**:
1. Start Redis:
```bash
docker-compose up -d redis
# Or
redis-server
```

2. Check if Redis is running:
```bash
redis-cli ping  # Should return PONG
```

3. Verify host/port in `config.toml`:
```toml
[cache.redis]
host = "localhost"  # Check this
port = 6379          # And this
```

### Authentication Failed

**Symptom**: `NOAUTH Authentication required`

**Solution**: Set Redis password in config:
```toml
[cache.redis]
password = "your-redis-password"
```

### Wrong Database

**Symptom**: Jobs not found, queue appears empty

**Solution**: Verify `redis_db` matches your setup:
```toml
[cache.redis]
db = 0  # Default database
```

Check which DB has data:
```bash
redis-cli
> INFO keyspace
db0:keys=123,expires=0
db1:keys=456,expires=0
```

---

## LLM Provider Errors

### Missing API Key

**Symptom**: `ValueError: API key is required for anthropic`

**Solution**: Set environment variable before running:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
python main.py
```

Or in your shell profile (~/.bashrc or ~/.zshrc):
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
export GEMINI_API_KEY="..."
```

### Rate Limit Exceeded

**Symptom**: `RateLimitError: Rate limit exceeded`

**Solutions**:
1. Reduce concurrent requests in `config.toml`:
```toml
[worker]
max_concurrent = 1  # Reduce from 3
```

2. Add delay between requests:
```toml
[job_queue]
poll_interval = "5s"  # Increase from 1s
```

3. Upgrade your API tier for higher limits

### Invalid Model Name

**Symptom**: `InvalidRequestError: Model 'claude-3' does not exist`

**Solution**: Use correct 2026 model names:
```toml
# Anthropic (2026 models)
claude-opus-4-5-20251101
claude-sonnet-4-5-20250929
claude-haiku-4-5-20250319

# OpenAI (2026 models)
gpt-4.1-2025-04-14
gpt-4.1-mini-2025-04-14
gpt-5.2

# Gemini (2026 models)
gemini-3.0-pro
gemini-3.0-flash
```

### Timeout Errors

**Symptom**: `ReadTimeoutError: Read timed out`

**Solutions**:
1. Increase timeout in provider config
2. Reduce text size per request
3. Check network connectivity to API

---

## Job Queue Problems

### Job Stuck in PROCESSING

**Symptom**: Job never completes, state stays at `PROCESSING`

**Solutions**:
1. Worker may have crashed - check logs
2. Reset job to PENDING:
```python
from job_queue import JobManager, JobState
manager = JobManager()
manager.set_state(job_id, JobState.PENDING)
```

3. Check worker status:
```python
jobs = manager.get_worker_jobs("worker-1")
print(f"Worker has {len(jobs)} jobs")
```

### Job Not Appearing in Queue

**Symptom**: `enqueue()` returns `None`

**Solutions**:
1. Check Redis connection:
```python
from job_queue import JobManager
manager = JobManager()
stats = manager.get_queue_stats()
print(stats)  # Should show queue sizes
```

2. Verify job data structure:
```python
job_data = {
    "source_file": "/path/to/file",  # Required
    "source_lang": "ja",              # Required
    "target_lang": "en"               # Required
}
```

3. Check for Redis errors in logs

### Checkpoint Not Saving

**Symptom**: `save_checkpoint()` returns `None`

**Solutions**:
1. Verify job_id is valid
2. Check Redis memory: `redis-cli INFO memory`
3. Verify checkpoint TTL (7 days) hasn't expired

---

## Document Parsing Failures

### Unsupported File Format

**Symptom**: `ValueError: Unsupported file format: .xyz`

**Solution**: Supported formats are `.docx`, `.pptx`, `.pdf`, `.xlsx`

Convert file to supported format first.

### Corrupted Document

**Symptom**: `BadZipFileError: File is not a zip file`

**Solutions**:
1. Verify file isn't corrupted: open in native application
2. Re-export document from source application
3. Check file permissions

### Empty Document

**Symptom**: No segments extracted

**Solutions**:
1. Verify document has content
2. Check for encoding issues
3. Try opening and re-saving document

---

## Performance Issues

### Slow Translation

**Symptom**: Jobs taking too long

**Solutions**:
1. Use faster model:
```toml
[translation]
default_model = "claude-haiku-4-5-20250319"  # Faster than Sonnet
```

2. Enable caching:
```toml
[cache]
enabled = true
```

3. Increase concurrent workers:
```toml
[worker]
max_concurrent = 5  # Increase from 3
```

### High Memory Usage

**Symptom**: Worker process consuming too much RAM

**Solutions**:
1. Reduce concurrent jobs:
```toml
[worker]
max_concurrent = 1
```

2. Process smaller batches
3. Check for memory leaks in parsers

### Redis Memory Full

**Symptom**: `OOM command not allowed when used memory > 'maxmemory'`

**Solutions**:
1. Clean up old jobs:
```python
manager.cleanup_old_jobs(max_age_seconds=3600)  # 1 hour
```

2. Configure Redis with maxmemory policy:
```bash
# In redis.conf
maxmemory 256mb
maxmemory-policy allkeys-lru
```

---

## Debug Mode

Enable detailed logging to diagnose issues:

```bash
# Set log level
export LOG_LEVEL=DEBUG

# Run worker
python main.py
```

### Verbose Output

```bash
# For CLI commands
python -m review translate "test" -p anthropic --verbose
```

### Check Redis Keys

```bash
# List all translation worker keys
redis-cli KEYS "trans:*"

# View job data
redis-cli GET "trans:job:JOB_ID"

# View job state
redis-cli GET "trans:state:JOB_ID"

# View checkpoint
redis-cli GET "trans:checkpoint:JOB_ID"

# Queue sizes
redis-cli ZCARD "trans:queue:urgent"
redis-cli ZCARD "trans:queue:normal"
redis-cli ZCARD "trans:queue:bulk"
```

### Monitor Progress Channel

```bash
# Listen to progress updates
redis-cli MONITOR | grep "translation:progress"
```

Or use pub/sub:
```bash
redis-cli SUBSCRIBE "translation:progress"
```

---

## Getting Help

If issues persist:

1. **Check logs**: Look for error messages in worker output
2. **Verify config**: Ensure all required settings are present
3. **Test Redis**: `redis-cli ping`
4. **Test API keys**: Use CLI to test provider connection
5. **Create minimal reproducible example**: Isolate the problem

### Log Issue with Debug Info

Include when reporting:
- Python version: `python --version`
- Redis version: `redis-cli --version`
- Config file (sanitized)
- Full error traceback
- Steps to reproduce
