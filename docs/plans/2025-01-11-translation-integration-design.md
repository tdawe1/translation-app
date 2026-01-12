# Translation Tools Integration Design

**Date:** 2025-01-11
**Status:** Design
**Author:** Design session via superpowers:brainstorming

## Overview

Integrate `/home/translation-tools` (Python-based JA→EN translation pipeline) into `translation-app` (GengoWatcher SaaS) via a **hybrid architecture** combining folder watching (for loose coupling with Gengo downloads) and Redis job queue (for horizontal scaling). The system features:

- **Multi-provider support** — Cloud APIs (Anthropic, OpenAI, Gemini), CLI tools (claude_code, gemini_cli, codex), and local models (Ollama, LM Studio)
- **Glossary system** — JSON-based terminology with exact, fuzzy, and contextual matching
- **Translation cache** — File/Redis backends with intelligent invalidation
- **Layout preservation** — Autofit algorithm and XML-level formatting for JA→EN expansion (1.5-2x)
- **Quality reports** — Confidence scoring, terminology compliance, consistency checks
- **Bilingual CSV output** — Review workflow with re-import capability
- **Audit tools** — JP character count, style compliance checking
- **Plugin architecture** — Extensible parsers, quality checks, pipeline stages, and upload destinations
- **Horizontal scaling** — Redis job queue with checkpoint/resume for fault tolerance

## Architecture

### Components

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Next.js   │────│  Go Backend  │────│   Redis     │
│  Frontend   │    │  (Fiber)     │    │  Queue/Pub  │
└─────────────┘    └──────────────┘    └─────────────┘
                           │                     │
                           │                     │
                           ▼                     ▼
                    ┌──────────────┐    ┌─────────────┐
                    │  PostgreSQL  │    │   Python    │
                    │  (jobs DB)   │    │   Worker    │
                    └──────────────┘    │  + C++ ext  │
                                            │
                                            ▼
                                     ┌─────────────────┐
                                     │  /watch/ folders│
                                     │  incoming/      │
                                     │  processing/    │
                                     │  translated/    │
                                     │  failed/        │
                                     └─────────────────┘
```

### File-Based Integration

**Key insight:** Folder watching as loose coupling point.

- Gengo downloads (or manual drops) → `/watch/incoming/`
- Watcher detects new files → triggers translation job
- Completed translations → `/watch/translated/`
- Failed jobs → `/watch/failed/` with error logs

**Benefits:**
- Works with automatic downloads and manual drops
- Decouples watcher from translation system
- Simple testing (drop file, watch it work)

## C++ Performance Extensions

**Hot paths moved to C++ via pybind11:**

| Component | Current (Python) | C++ Extension | Speedup |
|-----------|-----------------|---------------|---------|
| PPTX parsing | `python-pptx` | Custom C++ | ~5-10x |
| DOCX parsing | `python-docx` | Custom C++ | ~3-5x |
| PDF parsing | `PyPDF2` | `poppler` C++ | ~10-20x |
| Text chunking | Python loops | C++ string ops | ~2-3x |

**Structure:**
```
cpp/
├── src/
│   ├── pptx_parser.cpp
│   ├── docx_parser.cpp
│   ├── pdf_parser.cpp
│   ├── chunker.cpp
│   └── module.cpp          # pybind11 bindings
├── include/
└── build/                  # Compiled .so output
```

**Preserved:** All existing `translation-tools` logic (cache, glossary, style guide, layout preservation) remains intact and is enhanced.

## Glossary System

### Glossary Format

Glossary maintains terminology consistency across translations. Stored as JSON with flexible matching:

```json
{
  "version": "1.0",
  "source_language": "ja",
  "target_language": "en",
  "entries": [
    {
      "source": "顧客",
      "target": "customer",
      "part_of_speech": "noun",
      "context": "business",
      "variants": ["お客様", "お客さま"],
      "forbidden_translations": ["guest", "visitor"]
    },
    {
      "source": "見積書",
      "target": "quotation",
      "part_of_speech": "noun",
      "context": "business",
      "forbidden_translations": ["estimate", "bid"]
    }
  ],
  "compound_terms": [
    {
      "source": "顧客満足度",
      "target": "customer satisfaction level",
      "components": ["顧客", "満足度"]
    }
  ]
}
```

### Configuration

```toml
[glossary]
enabled = true
file = "/config/glossary.json"
# Matching strategy: exact, fuzzy, or both
matching_mode = "both"
# Minimum similarity score for fuzzy matching (0.0-1.0)
fuzzy_threshold = 0.85
# Case sensitivity for matching
case_sensitive = false

[glossary.enforcement]
# How strictly to enforce glossary terms
level = "suggest"  # "suggest", "warn", "enforce"
# Whether to flag forbidden translations
check_forbidden = true
# Context-aware matching (requires POS tagging)
context_aware = true
```

### Matching Algorithm

**Priority Order:**
1. Exact match (with variants)
2. Compound term match (multi-word phrases)
3. Fuzzy match (Levenshtein distance, adjusted for character sets)
4. Contextual match (POS-aware)

**Implementation:**
```python
class GlossaryMatcher:
    def match(self, text_segment: str, context: dict) -> list[GlossaryMatch]:
        # 1. Tokenize and POS-tag
        tokens = self.tokenize(text_segment)
        pos_tags = self.pos_tag(tokens)

        # 2. Exact match with variant lookup
        exact_matches = self.find_exact(tokens, pos_tags)

        # 3. Compound term detection (n-gram scan)
        compound_matches = self.find_compounds(tokens)

        # 4. Fuzzy match for remaining tokens
        fuzzy_matches = self.fuzzy_match(tokens, threshold=self.fuzzy_threshold)

        # 5. Apply context filters
        return self.filter_by_context(
            exact_matches + compound_matches + fuzzy_matches,
            context
        )
```

### Integration Points

- **Pre-translation:** Inject glossary into system prompt
- **Post-translation:** Verify glossary compliance in quality report
- **CLI override:** Allow per-job glossary selection

## Translation Cache

### Cache Strategy

Cache stores completed translations to avoid redundant API calls. Critical for cost reduction during iterative translation and re-translation workflows.

### Cache Format

JSON sidecar files alongside source documents:

```
/watch/cache/
├── file.pptx.json
├── file.docx.json
└── file.pdf.json
```

```json
{
  "cache_key": "sha256:a1b2c3d4",
  "source_file": "file.pptx",
  "source_hash": "sha256:e5f6g7h8",
  "created_at": "2025-01-11T10:00:00Z",
  "provider": "anthropic",
  "model": "claude-4.5-sonnet",
  "glossary_version": "1.0",
  "segments": {
    "slide1_title": {
      "source": "営業報告書",
      "target": "Business Report",
      "context": {"slide_number": 1, "element_type": "title"},
      "cached_at": "2025-01-11T10:00:01Z"
    }
  }
}
```

### Configuration

```toml
[cache]
enabled = true
backend = "file"  # "file", "redis", "postgres"
directory = "/watch/cache"
# Maximum cache size (per file, in MB)
max_file_size = 10
# Global cache limit (in GB)
max_total_size = 50
# Cache entry TTL
ttl = "720h"  # 30 days

[cache.invalidation]
# Invalidate on source file change
on_source_change = true
# Invalidate on glossary change
on_glossary_change = true
# Invalidate on model/version change
on_model_change = false

[cache.redis]
# Redis backend configuration
host = "localhost:6379"
db = 0
key_prefix = "trans:"
```

### Cache Key Generation

```python
def cache_key(
    source_text: str,
    provider: str,
    model: str,
    glossary_hash: str,
    context: dict
) -> str:
    """Generate deterministic cache key."""
    key_data = {
        "text": normalize_text(source_text),
        "provider": provider,
        "model": model,
        "glossary": glossary_hash,
        "context": frozenset(context.items())
    }
    return f"sha256:{hash_json(key_data)}"
```

### Cache Warming

Optional pre-translation cache population for common phrases:

```toml
[cache.warming]
enabled = true
source = "/config/common_phrases.json"
# Warm cache on worker startup
on_startup = true
```

## Layout Preservation

### The Challenge

Japanese→English translation expands text by **1.5-2x**. Layout preservation is critical to prevent:
- Text overflow beyond slide boundaries
- Broken formatting (misaligned bullets, wrapped text)
- Loss of visual hierarchy

### Preservation Strategies

**1. Autofit Calculation**

```python
def calculate_autofit(
    source_text: str,
    target_text: str,
    bounds: Rectangle,
    font: Font
) -> dict:
    """Calculate font size adjustment to fit target text in source bounds."""
    source_char_count = count_chars(source_text, script="ja")
    target_char_count = count_chars(target_text, script="en")

    # Estimate expansion ratio
    expansion_ratio = target_char_count / source_char_count

    # Calculate required font size reduction
    if expansion_ratio > 1.0:
        new_font_size = font.size / expansion_ratio
        # Clamp to minimum readable size
        return {"font_size": max(new_font_size, font.min_size)}

    return {"font_size": font.size}
```

**2. XML-Level Formatting (PPTX/DOCX)**

Preserve formatting XML structure, only replacing text content:

```python
class XMLAwareReplacer:
    def replace_text(self, xml_element: Element, old: str, new: str):
        """Replace text while preserving XML formatting."""
        for elem in xml_element.iter():
            if elem.text and old in elem.text:
                # Preserve element, only update text node
                elem.text = elem.text.replace(old, new)
            if elem.tail and old in elem.tail:
                elem.tail = elem.tail.replace(old, new)
```

**3. Text Overflow Detection**

```python
def detect_overflow(
    text: str,
    bounds: Rectangle,
    font: Font,
    max_lines: int
) -> bool:
    """Check if text will overflow its container."""
    measured = measure_text(text, font, bounds.width)
    return (
        measured.height > bounds.height or
        measured.lines > max_lines
    )
```

### Configuration

```toml
[layout.preservation]
# Strategy: "autofit", "overflow_warn", "truncate", "split"
strategy = "autofit"
# Minimum font size as percentage of original
min_font_size_pct = 60
# Warn if text would exceed this ratio of container
warn_threshold = 0.95

[layout.preservation.pptx]
# Preserve exact positioning
preserve_position = true
# Handle text frames with autofit
respect_autofit = true
# Preserve master slide layout
respect_master = true

[layout.preservation.docx]
# Preserve paragraph styles
preserve_styles = true
# Preserve table structure
preserve_tables = true
# Handle section breaks
preserve_sections = true
```

## Bilingual Output

### CSV Generation

Generate side-by-side Japanese/English CSV for review workflows:

```csv
segment_id,source,target,context,confidence,notes
slide1_title,営業報告書,Business Report,slide:1|type:title,0.98,
slide1_bullet1,売上高が増加,Sales increased,slide:1|type:bullet,0.95,
```

### Configuration

```toml
[output.bilingual_csv]
enabled = true
# Output location
path = "/watch/bilingual/"
# Include segment metadata
include_context = true
# Include confidence scores
include_confidence = true
# CSV encoding
encoding = "utf-8-sig"  # Excel-compatible UTF-8
# Delimiter
delimiter = ","

[output.bilingual_csv.columns]
# Column order and selection
columns = [
    "segment_id",
    "source",
    "target",
    "context",
    "confidence",
    "glossary_used",
    "notes"
]
```

### Review Workflow Integration

The bilingual CSV enables:
1. Human reviewer edits target column
2. Re-import CSV to update translations
3. Regenerate final document with approved translations

```toml
[review.csv_import]
enabled = true
# Column mapping for import
source_col = "source"
target_col = "target"
# Require all segments to be reviewed
require_complete = true
# Validate against original before applying
validate_checksum = true
```

## Audit Tools

### Japanese Character Count

```python
def count_japanese_characters(text: str) -> dict:
    """Count Japanese text for billing/estimation."""
    return {
        "total": len(text),
        "kanji": count_kanji(text),
        "hiragana": count_hiragana(text),
        "katakana": count_katakana(text),
        "punctuation": count_punctuation(text),
        "whitespace": count_whitespace(text),
        "estimated_english": estimate_english_length(text)
    }
```

### Style Compliance Checker

```python
class StyleChecker:
    def check(self, translation: str, source: str) -> list[StyleIssue]:
        """Check translation against style guide."""
        issues = []

        # Check for common JA→EN errors
        issues.extend(self.check_honorifics(translation))
        issues.extend(self.check_keigo(translation))
        issues.extend(self.check_sentence_endings(translation))
        issues.extend(self.check_consistency(translation, source))

        return issues
```

### Configuration

```toml
[audit]
enabled = true
# Run style checks automatically
auto_check = true
# Fail job if critical issues found
fail_on_critical = true

[audit.character_count]
# Include detailed character breakdown
detailed = true
# Estimate billing characters
billing_estimate = true

[audit.style]
# Style guide to enforce
style_guide = "/config/style_guide.json"
# Minimum acceptable score (0-100)
min_score = 70
# Specific checks to enable/disable
checks = [
    "honorifics",
    "keigo",
    "sentence_endings",
    "consistency",
    "numbers",
    "dates"
]
```

## Horizontal Scaling

### Problem with Folder Watching

Folder watching alone **does not scale horizontally**:
- Multiple workers on same host = race conditions
- Shared filesystem across hosts = latency, file locking issues
- No built-in load balancing or job distribution

### Solution: Job Queue Architecture

Add Redis-based job queue for distributed processing:

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Next.js   │────│  Go Backend  │────│   Redis     │
│  Frontend   │    │  (Fiber)     │    │   Queue     │
└─────────────┘    └──────────────┘    └─────────────┘
                           │                     │
                           │                     │
                           ▼                     ▼
                    ┌──────────────┐    ┌─────────────┐
                    │  PostgreSQL  │    │   Workers   │
                    │  (jobs DB)   │    │  (Python)   │
                    └──────────────┘    │  + C++ ext  │
                                            │    │
                           ┌────────────────┴─────┴────────────────┐
                           │                 │                  │
                           ▼                 ▼                  ▼
                    ┌───────────┐     ┌───────────┐     ┌───────────┐
                    │  Worker 1 │     │  Worker 2 │     │  Worker N │
                    │ (local)   │     │ (remote)  │     │ (cloud)   │
                    └───────────┘     └───────────┘     └───────────┘
```

### Job Queue Design

```toml
[job_queue]
enabled = true
backend = "redis"
# Job priority levels
priorities = ["urgent", "normal", "bulk"]
# Maximum job age before retry
max_age = "24h"

[job_queue.redis]
# Redis configuration
host = "localhost:6379"
db = 0
# Queue name prefix
queue_prefix = "trans:"
# Job result TTL
result_ttl = "168h"  # 7 days

[job_queue.concurrency]
# Max concurrent jobs per worker
max_concurrent = 3
# Queue polling interval
poll_interval = "1s"
# Job timeout
job_timeout = "30m"
```

### State Management

```python
class JobState(Enum):
    PENDING = "pending"
    PROCESSING = "processing"
    TRANSLATING = "translating"
    REVIEW_PENDING = "review_pending"
    APPROVED = "approved"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"

@dataclass
class Job:
    id: str
    source_file: str
    state: JobState
    worker_id: Optional[str]
    progress: float
    error: Optional[str]
    metadata: dict
```

### Checkpoint/Resume

```python
class CheckpointManager:
    def save(self, job_id: str, state: JobState, data: dict):
        """Save job progress checkpoint."""
        key = f"checkpoint:{job_id}"
        self.redis.setex(
            key,
            ttl=self.checkpoint_ttl,
            value=encode_json({"state": state, **data})
        )

    def load(self, job_id: str) -> Optional[Checkpoint]:
        """Load job checkpoint for resume."""
        key = f"checkpoint:{job_id}"
        data = self.redis.get(key)
        return decode_json(data) if data else None
```

### Configuration

```toml
[worker]
# Unique worker ID (auto-generated if empty)
id = ""
# Maximum concurrent jobs
max_concurrent = 3
# Heartbeat interval
heartbeat_interval = "10s"
# Worker timeout (no heartbeat = worker dead)
worker_timeout = "60s"

[worker.checkpoint]
# Enable checkpointing for resume
enabled = true
# Checkpoint interval
interval = "10s"
# Checkpoint TTL
ttl = "168h"  # 7 days
```

## Provider Configuration

### Provider Types

**Cloud APIs:**
- Anthropic: `claude-4.5-opus`, `claude-4.5-sonnet`, `claude-4.5-haiku`
- OpenAI: `gpt-5-high`, `gpt-5.1-high`, `gpt-5`, `gpt-5.1-med`
- Gemini: `gemini-3-flash`, `gemini-3-pro`

**CLI Tools (agentic):**
- `claude_code` — Uses Opus/Sonnet internally
- `gemini_cli` — Uses gemini-3-pro internally
- `codex`
- `opencode`

**Local Models:**
- Ollama: `llama3.1:70b`, `llama3.1:8b`, `qwen2.5:72b`
- LM Studio: Configurable endpoint

### Configuration Format: TOML

```toml
[translation]
default_provider = "anthropic"
default_model = "claude-4.5-sonnet"

# Routing fallback chain (configurable, not prescribed)
[[translation.routing]]
provider = "cli_tools"
tool = "claude_code"

[[translation.routing]]
provider = "anthropic"
model = "claude-4.5-sonnet"

[translation.providers.anthropic]
enabled = true
api_key_env = "ANTHROPIC_API_KEY"
```

**Why TOML:** Native Go support, readable, comments, explicit structure.

## Multi-Stage Pipeline

### Pipeline Modes (User-Selected)

| Mode | Description |
|------|-------------|
| `manual` | Full multi-model discussion + human review required |
| `semi_auto` | Translate + quality report + auto-approve if score good |
| `auto` | Translate and complete, no review |

### Available Stages (Pick and Choose)

```toml
[pipeline.stages.available]
translate = { required = true }
discuss = { required = false }
quality_report = { required = false }
human_review = { required = false }
approve = { required = false }
upload = { required = true }
```

### Quality Reports

Generated reports include:
- **Confidence score** — Overall translation quality
- **Terminology check** — Glossary compliance
- **Consistency check** — Term consistency across document
- **Formatting check** — Layout preservation
- **Suggestions** — Specific issues flagged for review

## Per-Model Configuration

Each model can have:
- Custom system prompts
- Temperature settings
- Style guide mappings
- Glossary associations

```toml
[prompts.templates]
[[prompts.templates]]
name = "formal_business"
system_prompt = "Formal business translation. Preserve keigo."
temperature = 0.3

[[prompts.templates]]
name = "technical_precise"
system_prompt = "Technical accuracy focus."
temperature = 0.1

[translation.user_profile]
claude_4_5_opus_prompt = "technical_precise"
haiku_prompt = "casual_fast"
```

## Upload Destinations

After successful translation/approval, upload to configured destinations:

```toml
[upload.destinations.gengo]
enabled = true
method = "api"
endpoint = "https://api.gengo.com/v2/translate/upload"

[upload.destinations.s3]
enabled = true
method = "s3"
bucket = "translations-output"

[upload.destinations.drive]
enabled = false
method = "api"
folder_id = "your-drive-folder-id"
```

## Error Handling & Retry

```toml
[translation.retry]
max_attempts = 3
base_delay = "1s"
max_delay = "60s"
multiplier = 2.0
jitter = true

[translation.retry.errors]
rate_limit = { retry = true, fallback = true }
timeout = { retry = true, fallback = true }
auth_failed = { retry = false, alert = true }
provider_unavailable = { retry = true, fallback = true }
```

**Fallback logic:** Try next provider in routing chain on retryable failures.

## File Lifecycle

```
incoming/file.pptx
    │
    ▼ (watchdog detects)
processing/file.pptx
    │
    ▼ (translating...)
    │
    ▼ (complete)
translated/file_en.pptx
    │
    ▼ (upload stage)
→ Gengo / S3 / Drive / etc.

OR on failure:
failed/file.pptx + error.log
```

## Testing Strategy

### Test Layers

1. **Unit tests** — Parsers, providers, config loading in isolation
2. **Integration tests** — Watcher triggers, provider fallback, C++↔Python boundary
3. **End-to-end tests** — Full pipeline with mock LLM responses
4. **Upload tests** — Verify upload to each destination type

### Fixtures

```
tests/
├── fixtures/
│   ├── documents/      # Sample files
│   ├── translations/   # Expected outputs
│   └── mock_responses/  # Pre-recorded API responses
└── integration/
    ├── test_pipeline.py
    ├── test_upload_gengo.py
    └── test_end_to_end.py
```

## Deployment

### Docker Compose (local/dev)

```yaml
services:
  translation-worker:
    build: ./cmd/translation-worker
    volumes:
      - ./watch:/watch
      - /home/translation-tools:/src/translation-tools:ro
    environment:
      - TRANSLATION_BACKEND=${TRANSLATION_BACKEND}
    depends_on: [redis]
```

### Production Considerations

| Concern | Solution |
|---------|----------|
| Worker restarts | Checkpoint progress; resume interrupted jobs |
| Config changes | Hot-reload TOML without restart |
| Monitoring | Prometheus metrics: queue depth, success rate, latency |
| Scale workers | Stateless design; run multiple instances |
| C++ compilation | Multi-stage Docker builds |

## Plugin Architecture

### Extensibility Design

To support future feature additions without modifying core code, the system uses a plugin architecture for parsers, quality checks, pipeline stages, and upload destinations.

### Plugin Interface

```python
from abc import ABC, abstractmethod
from typing import Any, Protocol

class Plugin(Protocol):
    """Base plugin protocol."""
    name: str
    version: str
    dependencies: list[str]

class ParserPlugin(Plugin):
    """Document parser plugins."""

    @abstractmethod
    def supported_extensions(self) -> list[str]:
        """Return list of supported file extensions."""
        ...

    @abstractmethod
    def parse(self, file_path: str) -> ParsedDocument:
        """Parse document into translatable segments."""
        ...

    @abstractmethod
    def render(self, doc: ParsedDocument, output_path: str) -> None:
        """Render translated document back to original format."""
        ...

class QualityCheckPlugin(Plugin):
    """Quality assessment plugins."""

    @abstractmethod
    def check(self, translation: str, source: str, context: dict) -> QualityReport:
        """Run quality check and return report."""
        ...

class PipelineStagePlugin(Plugin):
    """Custom pipeline stage plugins."""

    @abstractmethod
    def execute(self, job: Job, context: PipelineContext) -> StageResult:
        """Execute custom pipeline stage."""
        ...

class UploadDestinationPlugin(Plugin):
    """Upload destination plugins."""

    @abstractmethod
    def upload(self, file_path: str, metadata: dict) -> UploadResult:
        """Upload file to destination."""
        ...
```

### Plugin Discovery

```toml
[plugins]
# Directory to scan for plugins
directory = "/usr/lib/translation/plugins"
# Enable auto-discovery
auto_discover = true
# Plugin allowlist (if not empty, only these are loaded)
allowlist = []

[plugins.builtin]
# Builtin plugins always loaded
parsers = ["pptx", "docx", "pdf", "xlsx"]
quality_checks = ["glossary", "style", "consistency"]
upload_destinations = ["gengo", "s3", "drive"]
```

### Example: Custom Parser Plugin

```python
# /usr/lib/translation/parsers/custom_parser.py

class CustomFormatParser(ParserPlugin):
    name = "custom_format"
    version = "1.0.0"
    dependencies = []

    def supported_extensions(self) -> list[str]:
        return [".custom"]

    def parse(self, file_path: str) -> ParsedDocument:
        # Custom parsing logic
        with open(file_path, 'r') as f:
            data = json.load(f)
        return ParsedDocument(
            segments=[
                Segment(
                    id=f"seg_{i}",
                    text=seg["text"],
                    context={"type": seg["type"]}
                )
                for i, seg in enumerate(data["segments"])
            ],
            metadata={"format": "custom"}
        )

    def render(self, doc: ParsedDocument, output_path: str) -> None:
        # Custom rendering logic
        data = {
            "segments": [
                {"text": seg.text, "type": seg.context.get("type")}
                for seg in doc.segments
            ]
        }
        with open(output_path, 'w') as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
```

### Example: Custom Quality Check Plugin

```python
# /usr/lib/translation/quality_checks/custom_checker.py

class BrandTerminologyChecker(QualityCheckPlugin):
    name = "brand_terminology"
    version = "1.0.0"
    dependencies = []

    def __init__(self, brand_terms: dict[str, str]):
        self.brand_terms = brand_terms

    def check(self, translation: str, source: str, context: dict) -> QualityReport:
        issues = []
        for source_term, required_target in self.brand_terms.items():
            if source_term in source and required_target not in translation:
                issues.append(QualityIssue(
                    severity="warning",
                    message=f"Brand term '{source_term}' should translate to '{required_target}'",
                    location=context.get("segment_id")
                ))
        return QualityReport(
            score=100 - (len(issues) * 10),
            issues=issues
        )
```

### Example: Custom Upload Destination Plugin

```python
# /usr/lib/translation/upload/custom_uploader.py

class CustomStorageUploader(UploadDestinationPlugin):
    name = "custom_storage"
    version = "1.0.0"
    dependencies = ["requests"]

    def __init__(self, endpoint: str, api_key: str):
        self.endpoint = endpoint
        self.api_key = api_key

    def upload(self, file_path: str, metadata: dict) -> UploadResult:
        with open(file_path, 'rb') as f:
            response = requests.post(
                self.endpoint,
                headers={"Authorization": f"Bearer {self.api_key}"},
                files={"file": f},
                data={"metadata": json.dumps(metadata)}
            )
        response.raise_for_status()
        return UploadResult(
            success=True,
            location=response.json()["location"],
            metadata=response.json()
        )
```

### Plugin Configuration

Plugins can be configured via TOML:

```toml
[plugins.custom_parser]
enabled = true
priority = 10  # Higher priority = checked first

[plugins.brand_terminology]
enabled = true
# Brand-specific term mappings
brand_terms = { "ProductName" = "Official Product Name" }

[plugins.custom_storage]
enabled = true
endpoint = "https://storage.example.com/upload"
api_key_env = "CUSTOM_STORAGE_API_KEY"
```

## Directory Structure

```
/home/translation-app/
├── watch/
│   ├── incoming/
│   ├── processing/
│   ├── translated/
│   ├── failed/
│   ├── bilingual/         # Bilingual CSV output
│   └── cache/             # Translation cache
├── config/
│   ├── glossary.json      # Terminology glossary
│   ├── style_guide.json   # Style rules
│   └── common_phrases.json # Cache warming data
├── cmd/translation-worker/
│   ├── main.py
│   ├── config.toml
│   └── requirements.txt
├── cpp/
│   ├── src/              # C++ extensions
│   │   ├── pptx_parser.cpp
│   │   ├── docx_parser.cpp
│   │   ├── pdf_parser.cpp
│   │   ├── chunker.cpp
│   │   └── module.cpp    # pybind11 bindings
│   ├── include/
│   └── build/            # Compiled .so output
├── internal/translation/
│   ├── status.go
│   └── client.go
├── plugins/              # Plugin system
│   ├── parsers/
│   ├── quality_checks/
│   ├── stages/
│   └── uploaders/
└── scripts/
    └── link-tools.py
```

## Security Considerations

### API Key Management

```toml
[security]
# Never log API keys or sensitive data
redact_logs = true
# Encrypt cache at rest
encrypt_cache = true
# Require authentication for worker API
require_auth = true

[security.api_keys]
# Load keys from environment only
# Never store keys in config files
anthropic_key_env = "ANTHROPIC_API_KEY"
openai_key_env = "OPENAI_API_KEY"
gemini_key_env = "GEMINI_API_KEY"
```

### File Access Controls

```toml
[security.files]
# Allowed input directories
allowed_input_dirs = ["/watch/incoming", "/watch/processing"]
# Allowed output directories
allowed_output_dirs = ["/watch/translated", "/watch/bilingual"]
# Maximum file size (100 MB)
max_file_size = 104857600
# Allowed file extensions
allowed_extensions = [".pptx", ".docx", ".pdf", ".xlsx"]
```

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Folder watching + Job Queue | Folder watching for loose coupling (auto + manual); Job Queue for horizontal scaling |
| C++ extensions for hot paths | 5-20x speedup on document parsing; pybind11 makes integration clean |
| TOML configuration | Go-native, readable, supports comments |
| User-controlled pipelines | System provides tools; user decides workflow |
| Multi-provider support | Cost optimization, resilience, flexibility |
| Glossary system | Terminology consistency critical for JA→EN business translation |
| Translation cache | Cost reduction; critical for iterative workflows |
| Layout preservation | JA→EN expands 1.5-2x; autofit prevents overflow |
| Bilingual CSV output | Enables human review workflow |
| Plugin architecture | Extensibility without core code changes |
| Redis state management | Checkpoint/resume for worker failures |

## Migration from translation-tools

### Features Preserved

All existing `/home/translation-tools` functionality is preserved and enhanced:

| Original Feature | Integration Status |
|------------------|-------------------|
| Document adapters (PPTX, DOCX, PDF, XLSX) | ✅ Migrated to plugin system |
| Translation cache (JSON sidecar) | ✅ Enhanced with Redis backend option |
| Glossary matching | ✅ Enhanced with fuzzy matching + POS awareness |
| Layout preservation | ✅ Formalized as configurable strategy |
| Style checker | ✅ Migrated to quality check plugin |
| Bilingual CSV generation | ✅ Added with review workflow |
| JP character count | ✅ Migrated to audit tools |
| CLI translation scripts | ✅ Preserved via worker API |

### New Capabilities

Beyond the original `translation-tools`:

1. **Horizontal scaling** via Redis job queue
2. **Multi-provider support** (API + CLI + local)
3. **Quality reports** with confidence scoring
4. **Plugin architecture** for extensibility
5. **Checkpoint/resume** for fault tolerance
6. **Web UI** via Next.js frontend
7. **Real-time progress** via Redis pub/sub

## Next Steps

### Phase 1: Core Infrastructure (Weeks 1-2)
1. **Validation** — Review updated design with stakeholders
2. **Implementation planning** — Detailed TDD-based plan already created
3. **Proof of concept** — Folder watcher + single provider end-to-end
4. **Job queue** — Redis-based queue with checkpoint/resume

### Phase 2: Translation Engine (Weeks 3-4)
5. **Provider abstraction** — Multi-provider (API + CLI + local)
6. **C++ extensions** — Profile existing code, extract hot paths
7. **Glossary system** — Matching algorithm + enforcement
8. **Translation cache** — File + Redis backends

### Phase 3: Quality & Review (Weeks 5-6)
9. **Quality reports** — Confidence scoring + terminology check
10. **Layout preservation** — Autofit + XML-level formatting
11. **Bilingual CSV** — Generation + re-import workflow
12. **Audit tools** — JP count + style compliance

### Phase 4: Extensibility & Production (Weeks 7-8)
13. **Plugin system** — Parser, quality check, stage, uploader plugins
14. **Upload destinations** — Gengo API + S3 + Drive
15. **Security hardening** — API key management + file access controls
16. **Production deployment** — Docker compose + monitoring

## Future Work

Features identified for future iterations. Not in current implementation scope.

### Document Processing

| Feature | Description | Complexity | Notes |
|---------|-------------|------------|-------|
| **PDF OCR** | Extract text from scanned/image-based PDFs | Medium | Tesseract or cloud vision API; adds preprocessing stage |
| **Handwriting recognition** | Support handwritten Japanese documents | High | Requires specialized models |
| **Image text extraction** | Translate text embedded in images within PPTX/DOCX | Medium | OCR + image modification pipeline |
| **Subtitle files** | SRT/VTT support | Low | Simple text format |

### Translation Quality

| Feature | Description | Complexity | Notes |
|---------|-------------|------------|-------|
| **Translation memory** | Leverage past translations for consistency | Medium | Fuzzy segment matching; TMX format support |
| **Back-translation verification** | EN→JA→EN round-trip for quality check | Low | Additional API cost; catches meaning drift |
| **Domain-specific models** | Fine-tuned models for legal, medical, etc. | High | Training pipeline needed |
| **Confidence highlighting** | Visual markup of low-confidence segments | Low | UI enhancement for review workflow |

### Workflow

| Feature | Description | Complexity | Notes |
|---------|-------------|------------|-------|
| **Real-time collaboration** | Multiple reviewers on same document | High | WebSocket + conflict resolution; OT/CRDT required |
| **Version diffing** | Compare translation iterations | Medium | Segment-level diff visualization |
| **Batch scheduling** | Queue jobs for off-peak processing | Low | Cost optimization; time-based job queuing |
| **Webhook notifications** | Push status to external systems | Low | Standard integration pattern |

### Integrations

| Feature | Description | Complexity | Notes |
|---------|-------------|------------|-------|
| **CAT tool export** | XLIFF/TMX export for professional tools | Medium | Industry standard formats; translate-toolkit |
| **Slack/Teams alerts** | Notification on job completion/failure | Low | Webhook-based integration |
| **CMS connectors** | Direct pull/push from content systems | Medium | Per-CMS implementation (WordPress, Contentful, etc.) |

## External Dependencies

Libraries considered for integration to enhance functionality or reduce development effort.

### Adoption Recommendations

| Library | Category | Status | Rationale |
|---------|----------|--------|-----------|
| **pymupdf** | PDF Processing | **Adopt** | Replaces PyPDF2; C-backed (MuPDF), ~10x faster; may eliminate need for custom C++ PDF parser |
| **fugashi** | Japanese NLP | **Adopt** | MeCab wrapper for morphological analysis; required for accurate JP tokenization and POS tagging in glossary matching |
| **SudachiPy** | Japanese NLP | **Consider** | Alternative to MeCab/fugashi; better compound word handling for business Japanese |
| **ginza** | Japanese NLP | **Future** | spaCy-based Japanese NLP; NER, dependency parsing, lemmatization for advanced analysis |

### Translation & MT Providers

| Library | Description | Use Case |
|---------|-------------|----------|
| **EasyNMT** | Wrapper around OPUS, mBART, M2M-100 models | Local translation fallback; no API costs |
| **Argos Translate** | Offline neural MT | Self-hosted option for privacy-sensitive content |
| **NLLB** (No Language Left Behind) | Meta's multilingual model; strong JA→EN | HuggingFace-hosted; high quality local option |

```toml
# Future local provider configuration
[translation.providers.local_nmt]
enabled = false
library = "easynmt"
model = "opus-mt"  # or "m2m_100_1.2B" for higher quality
use_for = ["fallback", "cost_sensitive"]
```

### OCR (For Future PDF/Image Work)

| Library | Strength | Notes |
|---------|-----------|-------|
| **PaddleOCR** | Asian language OCR | Very strong JP support; active development; layout detection |
| **manga-ocr** | Japanese text in images | Trained on manga; handles vertical text well |
| **EasyOCR** | Multi-language OCR | Good balance of speed/accuracy; simpler API |
| **Surya** | Modern multilingual OCR | Layout detection included; nice API |

```toml
# Future PDF OCR configuration
[ocr.paddleocr]
enabled = false
lang = "japanese"
# Handle vertical text common in Japanese docs
detect_vertical = true
```

### Quality Assessment

| Library | Purpose | Use Case |
|---------|---------|---------|
| **COMET** | Quality estimation (reference-free) | Score translations for auto-approval in semi_auto mode |
| **sacrebleu** | BLEU scoring | Compare against reference translations for evaluation |
| **BERTScore** | Semantic similarity | Catch meaning drift that BLEU misses |

```toml
# Future quality scoring configuration
[quality.scoring]
use_comet = true
comet_model = "Unbabel/wmt22-comet-da"
min_score = 0.75  # Auto-approve threshold
```

### CAT Tool Interoperability

| Library | Purpose | Format Support |
|---------|---------|----------------|
| **translate-toolkit** | Translation memory formats | TMX, XLIFF, PO, TBX |
| **xliff** | XLIFF file handling | XLIFF 1.2/2.0 parsing and generation |

```toml
# Future CAT tool export configuration
[export.cat_tools]
# Industry-standard terminology exchange
tbx_enabled = true
# Translation memory exchange
tmx_enabled = true
# Localization interchange format
xliff_enabled = true
```

### Integration Priority

**Immediate Value (Low Effort):**
1. **pymupdf** — Replaces PyPDF2; likely eliminates C++ PDF extension need
2. **fugashi** — Required for accurate JP tokenization in glossary matching
3. **COMET** — Automated quality scoring for semi_auto pipeline mode

**Medium Term:**
4. **translate-toolkit** — TMX/TBX import for professional glossary workflows
5. **EasyNMT** — Local fallback provider for cost reduction
6. **PaddleOCR** — PDF OCR feature for scanned documents
