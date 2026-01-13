# Translation Worker

A hybrid translation worker that combines folder watching with a Redis-backed job queue for scalable, fault-tolerant document translation using multi-provider LLM support.

## Overview

The translation worker is designed for high-volume Japanese-to-English translation with the following key features:

- **Hybrid Architecture**: Folder watching for Gengo downloads + Redis job queue for horizontal scaling
- **Multi-Provider LLM Support**: Anthropic Claude, OpenAI GPT, Google Gemini
- **Fault Tolerance**: Checkpoint/resume capability for long-running jobs
- **Layout Preservation**: Maintains document formatting during translation
- **Glossary System**: Term consistency across translations
- **Translation Cache**: Avoids re-translating identical content
- **Audit Trail**: Complete tracking of all translation activities

## Quick Start

### Installation

```bash
cd backend/cmd/translation-worker

# Install dependencies
pip install -r requirements.txt  # Or: pip install anthropic openai requests redis click
```

### Configuration

Create a `config.toml` file in the translation-worker directory:

```toml
[worker]
id = "worker-1"
max_concurrent = 3
heartbeat_interval = "30s"

[translation]
default_provider = "anthropic"
default_model = "claude-sonnet-4-5-20250929"

[cache.redis]
host = "localhost"
port = 6379
db = 0
# password = "your-password"  # Optional

[job_queue]
enabled = true
backend = "redis"
max_concurrent = 3
poll_interval = "1s"

# Optional: Gengo style guide integration
[style_guide]
enabled = true
path = "/path/to/gengo-style-guide.md"
```

### Running the Worker

```bash
# From the translation-worker directory
python main.py

# Or with a custom config
python main.py --config /path/to/config.toml
```

### Environment Variables

**For Development (Recommended):** Use subscription-based endpoints to avoid per-token costs:

```bash
# Claude Code (requires Claude Code CLI running)
export ANTHROPIC_BASE_URL="http://localhost:8000"
export ANTHROPIC_API_KEY="sk-claude-code"  # Claude Code uses this format
```

**For Production:** Set your LLM provider API keys:

```bash
# Anthropic Claude
export ANTHROPIC_API_KEY="sk-ant-..."

# OpenAI
export OPENAI_API_KEY="sk-..."

# Google Gemini (generous free tier)
export GEMINI_API_KEY="..."
export GEMINI_PROJECT_ID="your-project-id"
export GEMINI_LOCATION="us-central1"  # Optional, default: us-central1
```

> **Note**: For development and internal tools, consider using subscription-based services:
> - **Claude Code** ($20/mo): No per-token costs via local endpoint
> - **Cursor** ($20/mo): Built-in Claude access in IDE
> - **Gemini Free Tier**: Generous quota for testing

## Architecture

### Components

| Module | Purpose |
|--------|---------|
| `main.py` | Worker entry point, queue consumer, signal handling |
| `job_queue/` | Redis-backed job queue with priority and checkpoint support |
| `review/` | Translation evaluation, judge system, multi-provider comparison |
| `parsers/` | Document parsing (DOCX, PPTX, PDF, XLSX) |
| `glossary/` | Term management and matching |
| `cache/` | Translation result caching |
| `layout/` | Document layout preservation |
| `audit/` | Translation audit logging and style checking |
| `watcher/` | Folder watching for new files |
| `nlp/` | Natural language processing utilities |
| `style_guide/` | Gengo style guide parsing and system prompt generation |

### Job Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│ Redis Queue │────▶│   Worker    │
└─────────────┘     └─────────────┘     └─────────────┘
                                                │
                                                ▼
                                          ┌──────────┐
                                          │  Parse   │
                                          └──────────┘
                                                │
                                                ▼
                                          ┌──────────┐
                                          │ Translate│
                                          └──────────┘
                                                │
                                                ▼
                                          ┌──────────┐
                                          │ Preserve │
                                          │  Layout  │
                                          └──────────┘
                                                │
                                                ▼
                                          ┌──────────┐
                                          │  Output  │
                                          └──────────┘
```

### Job States

| State | Description |
|-------|-------------|
| `PENDING` | Job enqueued, waiting for processing |
| `PROCESSING` | Worker has picked up the job |
| `TRANSLATING` | Active translation in progress |
| `COMPLETED` | Translation finished successfully |
| `FAILED` | Translation failed with error |
| `CANCELLED` | Job was cancelled |

## Job Queue API

### Enqueuing a Job

```python
from job_queue import JobManager

manager = JobManager(
    redis_host="localhost",
    redis_port=6379,
    redis_db=0
)

# Enqueue a translation job
job_id = manager.enqueue(
    job_data={
        "source_file": "/path/to/document.docx",
        "source_lang": "ja",
        "target_lang": "en",
        "glossary_id": "terms-2024"
    },
    priority="normal"  # "urgent", "normal", "bulk"
)
```

### Checking Job Status

```python
# Get job state
state = manager.get_state(job_id)

# Get full job data
job = manager.get_job(job_id)

# Get queue statistics
stats = manager.get_queue_stats()
# {"urgent": 0, "normal": 5, "bulk": 12, "total": 17}
```

### Progress Updates

Subscribe to Redis pub/sub for real-time progress:

```python
import redis

r = redis.Redis(host="localhost", port=6379, db=0)
pubsub = r.pubsub()
pubsub.subscribe("translation:progress")

for message in pubsub.listen():
    if message["type"] == "message":
        data = json.loads(message["data"])
        print(f"Job {data['job_id']}: {data['progress']*100}% - {data['message']}")
```

## LLM Providers

### Supported Providers

| Provider | Default Model | Notes |
|----------|---------------|-------|
| Anthropic | `claude-sonnet-4-5-20250929` | Best for accuracy |
| OpenAI | `gpt-4.1-2025-04-14` | Good balance |
| Gemini | `gemini-3.0-pro` | Cost-effective |

### 2026 API Compatibility

The worker is updated for 2026 API changes:
- **OpenAI**: Uses `max_completion_tokens` (not deprecated `max_tokens`)
- **Anthropic**: Uses Messages API with `response.content[0].text`
- **Gemini**: Uses Vertex AI REST endpoint

### Provider Selection

```python
from review.llm import get_provider

# Anthropic
provider = get_provider("anthropic", api_key="sk-ant-...")

# OpenAI
provider = get_provider("openai", api_key="sk-...")

# Gemini (requires additional params)
provider = get_provider(
    "gemini",
    api_key="...",
    project_id="my-project",
    location="us-central1"
)
```

## Document Parsers

### Supported Formats

| Format | Extension | Capabilities |
|--------|-----------|--------------|
| Word | `.docx` | Text extraction, formatting preservation |
| PowerPoint | `.pptx` | Slide-by-slide translation |
| PDF | `.pdf` | Text extraction (experimental) |
| Excel | `.xlsx` | Cell-by-cell translation |

### Using Parsers

```python
from parsers import get_parser

# Auto-detect parser from file extension
parser = get_parser("/path/to/document.docx")

# Extract content
segments = parser.extract_segments()

# Translate and rebuild
translated_segments = [translate(s) for s in segments]
output_path = parser.rebuild_document(translated_segments)
```

## CLI Reference

The review module provides CLI access to translation functionality:

```bash
# Translate a single text
python -m review translate "こんにちは" --provider anthropic

# Translate with custom model
python -m review translate "こんにちは" --provider openai --model gpt-4.1

# Batch translate from file
python -m review batch --input sources.txt --output translations.txt --provider anthropic

# Judge between two translations
python -m review judge original.txt translation_a.txt translation_b.txt --provider anthropic
```

### CLI Options

| Command | Options | Description |
|---------|---------|-------------|
| `translate` | `--provider`, `--model`, `--format`, `--output` | Translate text |
| `batch` | `--input`, `--output`, `--provider`, `--format` | Batch processing |
| `judge` | `--provider`, `--model`, `--format` | Compare translations |

## Glossary System

The glossary system ensures term consistency:

```python
from glossary import GlossaryLoader, GlossaryMatcher

# Load glossary from CSV or JSON
loader = GlossaryLoader()
glossary = loader.load_from_file("terms.csv")

# Match terms in text
matcher = GlossaryMatcher(glossary)
matches = matcher.find_matches("日本語のテキスト")

# Apply glossary to translation
prompt = matcher.build_glossary_prompt(source_text)
```

### Glossary Format (CSV)

```csv
source,target,notes
株式会社,Ltd.,Company suffix
取締役,Director,Board member
```

## Testing

```bash
# Run all tests
cd backend/cmd/translation-worker
pytest tests/

# Run specific test modules
pytest tests/test_queue/
pytest tests/test_review/
pytest tests/test_parsers/

# With coverage
pytest --cov=. --cov-report=html
```

## Deployment

### Docker

```dockerfile
FROM python:3.12-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .

CMD ["python", "main.py"]
```

### Scaling

Run multiple worker instances:

```bash
# Worker 1
WORKER_ID=worker-1 python main.py &

# Worker 2
WORKER_ID=worker-2 python main.py &

# Worker 3
WORKER_ID=worker-3 python main.py &
```

Each worker will pull jobs from the Redis queue independently.

## Troubleshooting

### Common Issues

**Redis connection refused**
- Ensure Redis is running: `docker-compose up -d redis`
- Check connection settings in `config.toml`

**API key errors**
- Verify environment variables are set
- Check for typos in API key

**Job stuck in PROCESSING**
- Worker may have crashed
- Check logs for errors
- Use `manager.set_state(job_id, JobState.PENDING)` to requeue

**Out of memory**
- Reduce `max_concurrent` in config
- Process smaller batches
- Check for memory leaks in parsers

### Debug Mode

Enable debug logging:

```bash
export LOG_LEVEL=DEBUG
python main.py
```

### Health Check

```bash
# Check if worker is processing jobs
from job_queue import JobManager
manager = JobManager()
stats = manager.get_queue_stats()
print(f"Jobs pending: {stats['total']}")
```

## License

MIT License - See LICENSE file for details.
