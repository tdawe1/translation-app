# Translation Tools Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate `/home/translation-tools` Python translation pipeline into `translation-app` via **hybrid architecture** (folder watching + Redis job queue) with glossary system, translation cache, layout preservation, bilingual output, audit tools, and plugin-based extensibility.

**Architecture:** A hybrid system combining folder watching (loose coupling for Gengo downloads) and Redis job queue (horizontal scaling). Features multi-provider support, glossary with fuzzy/POS matching, file/Redis cache backends, layout preservation for JA→EN expansion, bilingual CSV review workflow, JP audit tools, Protocol-based plugin architecture, and checkpoint/resume for fault tolerance.

**Tech Stack:** Python 3.11+, Go 1.21+ (Fiber backend), fugashi (Japanese NLP), pymupdf (PDF processing), TOML config, Redis, PostgreSQL, Watchdog, Docker, COMET (quality scoring)

**Open Questions to Resolve:**
- [ ] Validate pymupdf performance vs custom C++ PDF parser
- [ ] Define glossary loading strategy (file-only vs. database-backed)
- [ ] Specify "discuss" stage algorithm (how models critique each other's work)
- [ ] Security review required before production (API keys, file access, upload destinations)

---

## Phase 1: Foundation & Skeleton

### Task 1: Create project structure for translation worker

**Files:**
- Create: `cmd/translation-worker/main.py`
- Create: `cmd/translation-worker/config.toml`
- Create: `cmd/translation-worker/requirements.txt`
- Create: `cmd/translation-worker/Dockerfile`

**Step 1: Create main.py entry point**

```python
# cmd/translation-worker/main.py
import sys
import tomli
from pathlib import Path

def load_config(config_path: str = "config.toml") -> dict:
    """Load configuration from TOML file."""
    with open(config_path, "rb") as f:
        return tomli.load(f)

def main():
    config = load_config()
    print(f"Translation worker starting with backend: {config.get('translation', {}).get('default_provider')}")
    print(f"Mode: hybrid (folder watch + Redis job queue)")

if __name__ == "__main__":
    main()
```

**Step 2: Create config.toml skeleton**

```toml
# cmd/translation-worker/config.toml
[worker]
id = ""
max_concurrent = 3
heartbeat_interval = "10s"

[translation]
default_provider = "anthropic"
default_model = "claude-4.5-sonnet"

[translation.retry]
max_attempts = 3
base_delay = "1s"
max_delay = "60s"
multiplier = 2.0
jitter = true

[translation.providers.anthropic]
enabled = true
api_key_env = "ANTHROPIC_API_KEY"

[translation.providers.openai]
enabled = true
api_key_env = "OPENAI_API_KEY"

[glossary]
enabled = true
file = "/config/glossary.json"
matching_mode = "both"
fuzzy_threshold = 0.85

[cache]
enabled = true
backend = "file"
directory = "/watch/cache"
ttl = "720h"

[layout.preservation]
strategy = "autofit"
min_font_size_pct = 60
warn_threshold = 0.95

[output.bilingual_csv]
enabled = true
path = "/watch/bilingual/"
encoding = "utf-8-sig"

[job_queue]
enabled = true
backend = "redis"
max_concurrent = 3
poll_interval = "1s"
job_timeout = "30m"
```

**Step 3: Create requirements.txt**

```text
# cmd/translation-worker/requirements.txt
watchdog>=4.0.0
tomli>=2.0.0
redis>=5.0.0
psutil>=5.9.0
pybind11>=2.11.0

# Japanese NLP
fugashi>=1.3.0

# PDF processing
pymupdf>=1.23.0

# Quality assessment
unbabel-comet>=2.2.0

# Document processing
python-pptx>=0.6.21
python-docx>=1.1.0
openpyxl>=3.1.0
```

**Step 4: Create Dockerfile**

```dockerfile
# cmd/translation-worker/Dockerfile
FROM python:3.11-slim

WORKDIR /app

# Install system dependencies for fugashi (MeCab)
RUN apt-get update && apt-get install -y \
    libmecab-dev \
    mecab-ipadic-utf8 \
    mecab-utils \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY main.py .
COPY config.toml .

VOLUME ["/watch"]

CMD ["python", "main.py"]
```

**Step 5: Commit skeleton**

```bash
git add cmd/translation-worker/
git commit -m "feat(worker): add translation worker skeleton

- Add main.py entry point with config loading
- Add comprehensive TOML configuration structure
- Add requirements.txt with fugashi, pymupdf, COMET
- Add Dockerfile with MeCab dependencies
- Add placeholder configs for glossary, cache, layout, bilingual CSV
"
```

---

### Task 2: Implement Protocol-based plugin architecture

**Files:**
- Create: `cmd/translation-worker/plugins/base.py`
- Create: `tests/test_plugins/test_base.py`

**Step 1: Write test for Plugin Protocol**

```python
# tests/test_plugins/test_base.py
import pytest
from plugins.base import ParserPlugin, Plugin

def test_plugin_protocol():
    """Plugin protocol should define required attributes."""

    class TestParser(ParserPlugin):
        name = "test"
        version = "1.0"
        dependencies = []

        def supported_extensions(self):
            return [".test"]

        def parse(self, file_path: str):
            return None

        def render(self, doc, output_path: str):
            pass

    parser = TestParser()
    assert parser.name == "test"
    assert isinstance(parser, Plugin)
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_plugins/test_base.py -v
# Expected: ImportError
```

**Step 3: Implement Protocol-based plugin system**

```python
# cmd/translation-worker/plugins/base.py
from typing import Protocol, Any, runtime_checkable, list
from dataclasses import dataclass

@runtime_checkable
class Plugin(Protocol):
    """Base plugin protocol using structural subtyping."""
    name: str
    version: str
    dependencies: list[str]

@runtime_checkable
class ParserPlugin(Plugin):
    """Document parser plugins."""

    def supported_extensions(self) -> list[str]:
        """Return list of supported file extensions."""
        ...

    def parse(self, file_path: str) -> "ParsedDocument":
        """Parse document into translatable segments."""
        ...

    def render(self, doc: "ParsedDocument", output_path: str) -> None:
        """Render translated document back to original format."""
        ...

@runtime_checkable
class QualityCheckPlugin(Plugin):
    """Quality assessment plugins."""

    def check(self, translation: str, source: str, context: dict) -> "QualityReport":
        """Run quality check and return report."""
        ...

@runtime_checkable
class PipelineStagePlugin(Plugin):
    """Custom pipeline stage plugins."""

    def execute(self, job: "Job", context: "PipelineContext") -> "StageResult":
        """Execute custom pipeline stage."""
        ...

@runtime_checkable
class UploadDestinationPlugin(Plugin):
    """Upload destination plugins."""

    def upload(self, file_path: str, metadata: dict) -> "UploadResult":
        """Upload file to destination."""
        ...

# Data structures for plugin interfaces

@dataclass
class Segment:
    """A translatable text segment."""
    id: str
    text: str
    context: dict

@dataclass
class ParsedDocument:
    """Parsed document structure."""
    segments: list[Segment]
    metadata: dict
    format: str

@dataclass
class QualityReport:
    """Quality assessment report."""
    score: float
    issues: list["QualityIssue"]

@dataclass
class QualityIssue:
    """A quality issue found during checking."""
    severity: str
    message: str
    location: str
```

**Step 4: Run tests to verify they pass**

```bash
python -m pytest tests/test_plugins/test_base.py -v
# Expected: PASS
```

**Step 5: Commit plugin architecture**

```bash
git add cmd/translation-worker/plugins/ tests/test_plugins/
git commit -m "feat(plugins): implement Protocol-based plugin architecture

- Add Plugin base protocol using structural subtyping
- Add ParserPlugin, QualityCheckPlugin, PipelineStagePlugin, UploadDestinationPlugin
- Use @runtime_checkable for isinstance() support
- Add supporting dataclasses: Segment, ParsedDocument, QualityReport, QualityIssue
- Add unit tests for Plugin protocol
"
```

---

### Task 2.5: Implement watchdog-based folder watcher

**Files:**
- Create: `cmd/translation-worker/watcher/folder_watcher.py`
- Create: `tests/test_watcher/test_folder_watcher.py`

**Step 1: Write test for folder watcher**

```python
# tests/test_watcher/test_folder_watcher.py
import pytest
import tempfile
from pathlib import Path
from watcher.folder_watcher import FolderWatcher

def test_detects_new_files(tmp_path):
    """Should detect new files in watch directory."""
    incoming_dir = tmp_path / "incoming"
    processing_dir = tmp_path / "processing"
    translated_dir = tmp_path / "translated"
    failed_dir = tmp_path / "failed"

    for d in [incoming_dir, processing_dir, translated_dir, failed_dir]:
        d.mkdir()

    events = []
    watcher = FolderWatcher(
        incoming_dir=str(incoming_dir),
        processing_dir=str(processing_dir),
        translated_dir=str(translated_dir),
        failed_dir=str(failed_dir),
        callback=lambda e: events.append(e)
    )

    # Create a test file
    (incoming_dir / "test.pptx").write_text("test content")

    # Trigger scan
    watcher.scan()

    assert len(events) > 0
    assert events[0]["file"] == "test.pptx"
    assert events[0]["type"] == "new"

def test_moves_file_on_completion(tmp_path):
    """Should move files through processing stages."""
    incoming_dir = tmp_path / "incoming"
    processing_dir = tmp_path / "processing"
    translated_dir = tmp_path / "translated"

    for d in [incoming_dir, processing_dir, translated_dir]:
        d.mkdir()

    watcher = FolderWatcher(
        incoming_dir=str(incoming_dir),
        processing_dir=str(processing_dir),
        translated_dir=str(translated_dir)
    )

    # Create initial file
    test_file = incoming_dir / "test.docx"
    test_file.write_text("content")

    # Simulate processing complete
    watcher.move_to_translated(str(test_file))

    assert not test_file.exists()
    assert (translated_dir / "test.docx").exists()
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_watcher/test_folder_watcher.py -v
# Expected: ImportError
```

**Step 3: Implement folder watcher with watchdog**

```python
# cmd/translation-worker/watcher/folder_watcher.py
import os
import shutil
import time
from pathlib import Path
from typing import Callable, Optional
from dataclasses import dataclass
from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler, FileCreatedEvent

@dataclass
class FileEvent:
    """A file system event."""
    file: str
    path: str
    type: str  # "new", "modified", "deleted"
    timestamp: float

class FolderWatcherHandler(FileSystemEventHandler):
    """Handler for watchdog file system events."""

    def __init__(
        self,
        incoming_dir: str,
        callback: Callable[[FileEvent], None]
    ):
        self.incoming_dir = Path(incoming_dir)
        self.callback = callback
        self._pending_files = set()

    def on_created(self, event: FileCreatedEvent):
        """Handle file creation event."""
        if event.is_directory:
            return

        file_path = Path(event.src_path)

        # Check if file write is complete (file size stable)
        if self._is_write_complete(file_path):
            self.callback(FileEvent(
                file=file_path.name,
                path=str(file_path),
                type="new",
                timestamp=time.time()
            ))

    def _is_write_complete(self, file_path: Path, check_interval: float = 0.1, max_checks: int = 10) -> bool:
        """Check if file write is complete by monitoring size stability."""
        if not file_path.exists():
            return False

        last_size = -1
        for _ in range(max_checks):
            current_size = file_path.stat().st_size
            if current_size == last_size and current_size > 0:
                return True
            last_size = current_size
            time.sleep(check_interval)

        return False

class FolderWatcher:
    """Watches folders for new files and manages file lifecycle."""

    SUPPORTED_EXTENSIONS = {".pptx", ".docx", ".xlsx", ".pdf"}

    def __init__(
        self,
        incoming_dir: str,
        processing_dir: str,
        translated_dir: str,
        failed_dir: str,
        callback: Callable[[FileEvent], None] = None,
        poll_interval: float = 1.0
    ):
        self.incoming_dir = Path(incoming_dir)
        self.processing_dir = Path(processing_dir)
        self.translated_dir = Path(translated_dir)
        self.failed_dir = Path(failed_dir)
        self.callback = callback
        self.poll_interval = poll_interval

        # Create directories
        for d in [self.incoming_dir, self.processing_dir, self.translated_dir, self.failed_dir]:
            d.mkdir(parents=True, exist_ok=True)

        self._observer: Optional[Observer] = None

    def start(self):
        """Start watching for file system events."""
        self._observer = Observer()
        handler = FolderWatcherHandler(str(self.incoming_dir), self.callback or self._default_callback)
        self._observer.schedule(handler, str(self.incoming_dir))
        self._observer.start()

    def stop(self):
        """Stop watching."""
        if self._observer:
            self._observer.stop()
            self._observer.join()

    def scan(self):
        """Scan for existing files in incoming directory."""
        for file_path in self.incoming_dir.iterdir():
            if file_path.is_file() and file_path.suffix in self.SUPPORTED_EXTENSIONS:
                if self.callback:
                    self.callback(FileEvent(
                        file=file_path.name,
                        path=str(file_path),
                        type="new",
                        timestamp=time.time()
                    ))

    def move_to_processing(self, file_path: str) -> str:
        """Move file to processing directory."""
        src = Path(file_path)
        dst = self.processing_dir / src.name
        shutil.move(str(src), str(dst))
        return str(dst)

    def move_to_translated(self, file_path: str) -> str:
        """Move file to translated directory."""
        src = Path(file_path)
        dst = self.translated_dir / src.name
        shutil.move(str(src), str(dst))
        return str(dst)

    def move_to_failed(self, file_path: str, error_log: str = ""):
        """Move file to failed directory with error log."""
        src = Path(file_path)
        dst = self.failed_dir / src.name
        shutil.move(str(src), str(dst))

        # Write error log
        if error_log:
            log_path = self.failed_dir / f"{src.stem}_error.txt"
            log_path.write_text(error_log)

    def _default_callback(self, event: FileEvent):
        """Default callback when none provided."""
        print(f"[WATCHER] New file detected: {event.file}")
```

**Step 4: Run tests to verify they pass**

```bash
python -m pytest tests/test_watcher/test_folder_watcher.py -v
# Expected: PASS
```

**Step 5: Add watchdog to requirements.txt**

```text
# Add to cmd/translation-worker/requirements.txt
watchdog>=4.0.0
```

**Step 6: Commit folder watcher implementation**

```bash
git add cmd/translation-worker/watcher/ tests/test_watcher/ cmd/translation-worker/requirements.txt
git commit -m "feat(watcher): implement watchdog-based folder watching

- Add FolderWatcher with watchdog library integration
- Add FileCreatedEvent detection with write completion check
- Add file lifecycle management (incoming→processing→translated/failed)
- Add move_to_processing/move_to_translated/move_to_failed methods
- Add scan() method for existing files
- Add unit tests for file detection and movement
"
```

---

## Phase 2: Glossary System

### Task 3: Implement fugashi-based Japanese tokenization

**Files:**
- Create: `cmd/translation-worker/nlp/tokenizer.py`
- Create: `tests/test_nlp/test_tokenizer.py`

**Step 1: Write test for Japanese tokenizer**

```python
# tests/test_nlp/test_tokenizer.py
import pytest
from nlp.tokenizer import JapaneseTokenizer

def test_tokenize_japanese_text():
    """Should tokenize Japanese text with POS tags."""
    tokenizer = JapaneseTokenizer()
    result = tokenizer.tokenize("顧客満足度を調査します")

    assert len(result.tokens) > 0
    assert "kanji" in result.char_counts
    assert any(t.pos == "名詞" for t in result.tokens)
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_nlp/test_tokenizer.py -v
# Expected: ImportError
```

**Step 3: Implement Japanese tokenizer with fugashi**

```python
# cmd/translation-worker/nlp/tokenizer.py
from dataclasses import dataclass
from typing import Optional
import fugashi

@dataclass
class Token:
    """A Japanese token with POS tag."""
    text: str
    pos: str
    reading: Optional[str] = None

@dataclass
class TokenizationResult:
    """Result of tokenizing Japanese text."""
    tokens: list[Token]
    char_counts: dict

class JapaneseTokenizer:
    """Japanese text tokenizer using fugashi (MeCab wrapper)."""

    # Character ranges for counting
    HIRAGANA_RANGES = [(0x3040, 0x309F), (0x30A0, 0x30FF)]
    KATAKANA_RANGES = [(0x30A0, 0x30FF), (0x31F0, 0x31FF)]
    KANJI_RANGES = [(0x4E00, 0x9FFF), (0x3400, 0x4DBF)]

    def __init__(self):
        try:
            self.tagger = fugashi.Tagger()
        except Exception as e:
            raise RuntimeError(f"Failed to initialize fugashi/MeCab: {e}")

    def tokenize(self, text: str) -> TokenizationResult:
        """Tokenize Japanese text and return tokens with POS tags."""
        tokens = []

        for node in self.tagger.parse(text):
            tokens.append(Token(
                text=node.surface,
                pos=node.pos,
                reading=node.feature.reading if hasattr(node, 'feature') else None
            ))

        char_counts = self._count_characters(text)

        return TokenizationResult(tokens=tokens, char_counts=char_counts)

    def _count_characters(self, text: str) -> dict:
        """Count Japanese character types."""
        counts = {
            "total": len(text),
            "kanji": 0,
            "hiragana": 0,
            "katakana": 0,
            "punctuation": 0,
            "whitespace": 0,
            "latin": 0
        }

        for char in text:
            code = ord(char)

            if char.isspace():
                counts["whitespace"] += 1
            elif 0x4E00 <= code <= 0x9FFF or 0x3400 <= code <= 0x4DBF:
                counts["kanji"] += 1
            elif 0x3040 <= code <= 0x309F:
                counts["hiragana"] += 1
            elif 0x30A0 <= code <= 0x30FF or 0x31F0 <= code <= 0x31FF:
                counts["katakana"] += 1
            elif char in '、。！？「」『』（）':
                counts["punctuation"] += 1
            elif char.isalpha():
                counts["latin"] += 1

        return counts

    def estimate_english_length(self, text: str) -> int:
        """Estimate English character count for Japanese text."""
        counts = self._count_characters(text)
        # Typical expansion: JA → EN is 1.5-2x
        # Kanji tends to expand more than kana
        return int(
            counts["kanji"] * 2.0 +
            counts["hiragana"] * 1.5 +
            counts["katakana"] * 1.5 +
            counts["punctuation"] +
            counts["latin"]
        )
```

**Step 4: Run tests to verify they pass**

```bash
python -m pytest tests/test_nlp/test_tokenizer.py -v
# Expected: PASS
```

**Step 5: Commit tokenizer implementation**

```bash
git add cmd/translation-worker/nlp/ tests/test_nlp/
git commit -m "feat(nlp): implement fugashi-based Japanese tokenizer

- Add JapaneseTokenizer class using fugashi (MeCab wrapper)
- Add Token dataclass with text, POS tag, and reading
- Add TokenizationResult with tokens and character counts
- Add character counting for kanji, hiragana, katakana, punctuation
- Add estimate_english_length() for billing estimation
- Add unit tests
"
```

---

### Task 4: Implement glossary matching with fuzzy/POS support

**Files:**
- Create: `cmd/translation-worker/glossary/matcher.py`
- Create: `cmd/translation-worker/glossary/loader.py`
- Create: `tests/test_glossary/test_matcher.py`

**Step 1: Write test for glossary matching**

```python
# tests/test_glossary/test_matcher.py
import pytest
from glossary.matcher import GlossaryMatcher, GlossaryMatch
from glossary.loader import load_glossary_from_dict

def test_exact_match_with_variants():
    """Should find exact match including variants."""
    glossary_data = {
        "entries": [
            {
                "source": "顧客",
                "target": "customer",
                "part_of_speech": "noun",
                "variants": ["お客様", "お客さま"]
            }
        ]
    }
    glossary = load_glossary_from_dict(glossary_data)
    matcher = GlossaryMatcher(glossary)

    # Direct match
    matches = matcher.match("顧客について")
    assert len(matches) > 0
    assert matches[0].target == "customer"

    # Variant match
    matches = matcher.match("お客様について")
    assert len(matches) > 0
    assert matches[0].target == "customer"

def test_fuzzy_match():
    """Should find fuzzy match within threshold."""
    glossary_data = {
        "entries": [
            {
                "source": "見積書",
                "target": "quotation",
                "part_of_speech": "noun"
            }
        ]
    }
    glossary = load_glossary_from_dict(glossary_data)
    matcher = GlossaryMatcher(glossary, fuzzy_threshold=0.85)

    # Small typo
    matches = matcher.match("込見積書です")
    assert any("quotation" in m.target for m in matches)

def test_pos_aware_matching():
    """Should match based on part of speech."""
    glossary_data = {
        "entries": [
            {
                "source": "は",
                "target": "",  # Particle, no translation
                "part_of_speech": "particle"
            }
        ]
    }
    glossary = load_glossary_from_dict(glossary_data)
    matcher = GlossaryMatcher(glossary)

    # "は" as particle should be skipped
    matches = matcher.match("私は学生です")
    # The particle "は" should not trigger a translation
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_glossary/test_matcher.py -v
# Expected: ImportError
```

**Step 3: Implement glossary matcher**

```python
# cmd/translation-worker/glossary/loader.py
from dataclasses import dataclass
from typing import Optional

@dataclass
class GlossaryEntry:
    """A single glossary entry."""
    source: str
    target: str
    part_of_speech: str
    context: str = ""
    variants: list = None
    forbidden_translations: list = None

@dataclass
class Glossary:
    """Loaded glossary with entries and compound terms."""
    entries: list
    compound_terms: list
    version: str = "1.0"

    def __post_init__(self):
        if self.entries[0].variants is None:
            for entry in self.entries:
                entry.variants = []
        if self.entries[0].forbidden_translations is None:
            for entry in self.entries:
                entry.forbidden_translations = []

def load_glossary_from_dict(data: dict) -> Glossary:
    """Load glossary from dictionary (parsed JSON)."""
    entries = [
        GlossaryEntry(
            source=e["source"],
            target=e["target"],
            part_of_speech=e.get("part_of_speech", ""),
            context=e.get("context", ""),
            variants=e.get("variants", []),
            forbidden_translations=e.get("forbidden_translations", [])
        )
        for e in data.get("entries", [])
    ]

    compound_terms = data.get("compound_terms", [])

    return Glossary(
        entries=entries,
        compound_terms=compound_terms,
        version=data.get("version", "1.0")
    )

def load_glossary_from_file(filepath: str) -> Glossary:
    """Load glossary from JSON file."""
    import json
    with open(filepath, "r", encoding="utf-8") as f:
        data = json.load(f)
    return load_glossary_from_dict(data)
```

```python
# cmd/translation-worker/glossary/matcher.py
from dataclasses import dataclass
from typing import List, Optional
import Levenshtein
from ..nlp.tokenizer import JapaneseTokenizer

@dataclass
class GlossaryMatch:
    """A glossary match result."""
    source: str
    target: str
    matched_text: str
    entry: GlossaryEntry
    match_type: str  # "exact", "variant", "compound", "fuzzy"
    confidence: float
    pos_match: bool

class GlossaryMatcher:
    """Matches text against glossary with exact, fuzzy, and POS-aware matching."""

    # POS tags that should not be translated
    SKIP_POS = {"助詞", "助動詞", "記号", "接続詞", " particle"}

    def __init__(
        self,
        glossary: Glossary,
        fuzzy_threshold: float = 0.85,
        enable_pos_matching: bool = True
    ):
        self.glossary = glossary
        self.fuzzy_threshold = fuzzy_threshold
        self.enable_pos_matching = enable_pos_matching
        self.tokenizer = JapaneseTokenizer() if enable_pos_matching else None

        # Build lookup indices
        self._build_indices()

    def _build_indices(self):
        """Build lookup indices for efficient matching."""
        self.exact_index = {e.source: e for e in self.glossary.entries}

        # Variant index
        self.variant_index = {}
        for entry in self.glossary.entries:
            for variant in entry.variants:
                self.variant_index[variant] = entry

        # Compound term index (sorted by length desc)
        self.compound_terms = sorted(
            self.glossary.compound_terms,
            key=lambda c: len(c["source"]),
            reverse=True
        )

    def match(self, text_segment: str, context: dict = None) -> List[GlossaryMatch]:
        """Find all glossary matches in text segment."""
        matches = []

        if self.enable_pos_matching and self.tokenizer:
            tokens = self.tokenizer.tokenize(text_segment)
            matches.extend(self._match_with_pos(tokens, text_segment))
        else:
            matches.extend(self._match_without_pos(text_segment))

        # Sort by confidence desc
        matches.sort(key=lambda m: m.confidence, reverse=True)

        return matches

    def _match_with_pos(self, tokens, original_text: str) -> List[GlossaryMatch]:
        """Match using POS-aware tokenization."""
        matches = []

        for token in tokens.tokens:
            # Skip particles and function words
            if token.pos in self.SKIP_POS or "助詞" in token.pos:
                continue

            # Exact match
            if token.text in self.exact_index:
                entry = self.exact_index[token.text]
                matches.append(GlossaryMatch(
                    source=entry.source,
                    target=entry.target,
                    matched_text=token.text,
                    entry=entry,
                    match_type="exact",
                    confidence=1.0,
                    pos_match=True
                ))

            # Variant match
            if token.text in self.variant_index:
                entry = self.variant_index[token.text]
                matches.append(GlossaryMatch(
                    source=entry.source,
                    target=entry.target,
                    matched_text=token.text,
                    entry=entry,
                    match_type="variant",
                    confidence=0.95,
                    pos_match=True
                ))

        return matches

    def _match_without_pos(self, text: str) -> List[GlossaryMatch]:
        """Match without POS tagging (simpler, faster)."""
        matches = []

        # Exact matches
        for source, entry in self.exact_index.items():
            if source in text:
                matches.append(GlossaryMatch(
                    source=source,
                    target=entry.target,
                    matched_text=source,
                    entry=entry,
                    match_type="exact",
                    confidence=1.0,
                    pos_match=False
                ))

        # Variant matches
        for variant, entry in self.variant_index.items():
            if variant in text:
                matches.append(GlossaryMatch(
                    source=entry.source,
                    target=entry.target,
                    matched_text=variant,
                    entry=entry,
                    match_type="variant",
                    confidence=0.95,
                    pos_match=False
                ))

        # Compound term matching
        for compound in self.compound_terms:
            source = compound["source"]
            if source in text:
                # Create a pseudo-entry for compound term
                entry = GlossaryEntry(
                    source=source,
                    target=compound["target"],
                    part_of_speech="compound"
                )
                matches.append(GlossaryMatch(
                    source=source,
                    target=compound["target"],
                    matched_text=source,
                    entry=entry,
                    match_type="compound",
                    confidence=0.98,
                    pos_match=False
                ))

        # Fuzzy matching for remaining
        matches.extend(self._fuzzy_match(text, matches))

        return matches

    def _fuzzy_match(self, text: str, existing_matches: List[GlossaryMatch]) -> List[GlossaryMatch]:
        """Apply fuzzy matching for entries not already matched."""
        matches = []
        matched_sources = {m.source for m in existing_matches}

        for entry in self.glossary.entries:
            if entry.source in matched_sources:
                continue

            # Check for fuzzy match using Levenshtein ratio
            if entry.source in text:
                ratio = Levenshtein.ratio(entry.source, text)
                if ratio >= self.fuzzy_threshold:
                    matches.append(GlossaryMatch(
                        source=entry.source,
                        target=entry.target,
                        matched_text=entry.source,
                        entry=entry,
                        match_type="fuzzy",
                        confidence=ratio * 0.8,  # Reduce confidence for fuzzy
                        pos_match=False
                    ))

        return matches

    def inject_into_prompt(self, matches: List[GlossaryMatch]) -> str:
        """Generate glossary section for system prompt."""
        if not matches:
            return ""

        lines = ["Glossary terms to use:", ""]
        for match in matches[:20]:  # Limit to top 20
            lines.append(f"- {match.source} → {match.target}")

        return "\n".join(lines)
```

**Step 4: Run tests to verify they pass**

```bash
python -m pytest tests/test_glossary/test_matcher.py -v
# Expected: PASS (may need Levenshtein install)
```

**Step 5: Add Levenshtein to requirements.txt**

```text
# Add to cmd/translation-worker/requirements.txt
Levenshtein>=0.23.0
```

**Step 6: Commit glossary system**

```bash
git add cmd/translation-worker/glossary/ cmd/translation-worker/requirements.txt tests/test_glossary/
git commit -m "feat(glossary): implement fuzzy and POS-aware glossary matching

- Add GlossaryEntry and Glossary dataclasses
- Add load_glossary_from_file() for JSON glossary loading
- Add GlossaryMatcher with exact, variant, compound, and fuzzy matching
- Integrate fugashi tokenizer for POS-aware matching
- Skip particles and function words from translation
- Add inject_into_prompt() for LLM system prompt injection
- Add unit tests for matching strategies
"
```

---

## Phase 3: Translation Cache

### Task 5: Implement translation cache with file/Redis backends

**Files:**
- Create: `cmd/translation-worker/cache/backend.py`
- Create: `cmd/translation-worker/cache/manager.py`
- Create: `tests/test_cache/test_backend.py`

**Step 1: Write test for cache backend**

```python
# tests/test_cache/test_backend.py
import pytest
import tempfile
from cache.backend import FileCacheBackend, RedisCacheBackend
from cache.manager import CacheManager

def test_file_cache_store_and_retrieve():
    """File cache should store and retrieve translations."""
    with tempfile.TemporaryDirectory() as tmpdir:
        backend = FileCacheBackend(tmpdir)
        manager = CacheManager(backend)

        # Store
        manager.store(
            cache_key="test:123",
            source="こんにちは",
            target="Hello",
            provider="anthropic",
            model="claude-4.5-sonnet"
        )

        # Retrieve
        result = manager.retrieve("test:123")
        assert result is not None
        assert result["target"] == "Hello"

def test_cache_key_generation():
    """Cache keys should be deterministic."""
    backend = FileCacheBackend("/tmp/cache")
    manager = CacheManager(backend)

    key1 = manager.generate_key(
        source="テスト",
        provider="anthropic",
        model="claude-4.5-sonnet",
        glossary_hash="abc123"
    )

    key2 = manager.generate_key(
        source="テスト",
        provider="anthropic",
        model="claude-4.5-sonnet",
        glossary_hash="abc123"
    )

    assert key1 == key2
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_cache/test_backend.py -v
# Expected: ImportError
```

**Step 3: Implement cache backend and manager**

```python
# cmd/translation-worker/cache/backend.py
from abc import ABC, abstractmethod
from typing import Optional
import json
import hashlib
from pathlib import Path

class CacheBackend(ABC):
    """Abstract cache backend."""

    @abstractmethod
    def get(self, key: str) -> Optional[dict]:
        """Retrieve cached value by key."""
        pass

    @abstractmethod
    def set(self, key: str, value: dict, ttl: int) -> bool:
        """Store value with TTL in seconds."""
        pass

    @abstractmethod
    def delete(self, key: str) -> bool:
        """Delete cached value."""
        pass

    @abstractmethod
    def exists(self, key: str) -> bool:
        """Check if key exists."""
        pass

class FileCacheBackend(CacheBackend):
    """File-based cache backend using JSON sidecar files."""

    def __init__(self, directory: str):
        self.directory = Path(directory)
        self.directory.mkdir(parents=True, exist_ok=True)

    def _get_file_path(self, key: str) -> Path:
        """Get file path for cache key."""
        # Use SHA256 hash for safe filenames
        safe_key = hashlib.sha256(key.encode()).hexdigest()[:16]
        return self.directory / f"{safe_key}.json"

    def get(self, key: str) -> Optional[dict]:
        file_path = self._get_file_path(key)
        if not file_path.exists():
            return None

        try:
            with open(file_path, "r", encoding="utf-8") as f:
                return json.load(f)
        except (json.JSONDecodeError, IOError):
            return None

    def set(self, key: str, value: dict, ttl: int = 0) -> bool:
        file_path = self._get_file_path(key)
        try:
            with open(file_path, "w", encoding="utf-8") as f:
                json.dump(value, f, ensure_ascii=False, indent=2)
            return True
        except IOError:
            return False

    def delete(self, key: str) -> bool:
        file_path = self._get_file_path(key)
        if file_path.exists():
            file_path.unlink()
            return True
        return False

    def exists(self, key: str) -> bool:
        return self._get_file_path(key).exists()

class RedisCacheBackend(CacheBackend):
    """Redis-based cache backend for distributed caching."""

    def __init__(self, host: str = "localhost", port: int = 6379, db: int = 0, key_prefix: str = "trans:"):
        try:
            import redis
            self.client = redis.Redis(host=host, port=port, db=db, decode_responses=True)
            self.key_prefix = key_prefix
        except ImportError:
            raise RuntimeError("redis package not installed")

    def _make_key(self, key: str) -> str:
        """Add prefix to cache key."""
        return f"{self.key_prefix}{key}"

    def get(self, key: str) -> Optional[dict]:
        redis_key = self._make_key(key)
        try:
            value = self.client.get(redis_key)
            return json.loads(value) if value else None
        except (json.JSONDecodeError, AttributeError):
            return None

    def set(self, key: str, value: dict, ttl: int = 0) -> bool:
        redis_key = self._make_key(key)
        try:
            serialized = json.dumps(value, ensure_ascii=False)
            if ttl > 0:
                return self.client.setex(redis_key, ttl, serialized)
            else:
                return self.client.set(redis_key, serialized)
        except Exception:
            return False

    def delete(self, key: str) -> bool:
        redis_key = self._make_key(key)
        return bool(self.client.delete(redis_key))

    def exists(self, key: str) -> bool:
        redis_key = self._make_key(key)
        return bool(self.client.exists(redis_key))
```

```python
# cmd/translation-worker/cache/manager.py
from typing import Optional
from .backend import CacheBackend, FileCacheBackend, RedisCacheBackend
import hashlib
import json

class CacheManager:
    """Manages translation caching with intelligent invalidation."""

    def __init__(self, backend: CacheBackend, default_ttl: int = 7200 * 30):
        """Initialize cache manager.

        Args:
            backend: Cache backend instance
            default_ttl: Default TTL in seconds (30 days)
        """
        self.backend = backend
        self.default_ttl = default_ttl

    def generate_key(
        self,
        source: str,
        provider: str,
        model: str,
        glossary_hash: str = "",
        context: dict = None
    ) -> str:
        """Generate deterministic cache key."""
        key_data = {
            "text": source.strip(),
            "provider": provider,
            "model": model,
            "glossary": glossary_hash
        }
        # Include context in key if provided
        if context:
            key_data["context"] = sorted(context.items())

        # Hash the key data
        key_json = json.dumps(key_data, sort_keys=True)
        return f"sha256:{hashlib.sha256(key_json.encode()).hexdigest()[:16]}"

    def store(
        self,
        cache_key: str,
        source: str,
        target: str,
        provider: str,
        model: str,
        glossary_version: str = "",
        metadata: dict = None
    ) -> bool:
        """Store translation in cache."""
        value = {
            "cache_key": cache_key,
            "source": source,
            "target": target,
            "provider": provider,
            "model": model,
            "glossary_version": glossary_version,
            "metadata": metadata or {}
        }
        return self.backend.set(cache_key, value, self.default_ttl)

    def retrieve(self, cache_key: str) -> Optional[dict]:
        """Retrieve translation from cache."""
        return self.backend.get(cache_key)

    def invalidate(self, cache_key: str) -> bool:
        """Invalidate cached entry."""
        return self.backend.delete(cache_key)

    def warm(
        self,
        phrases: list[tuple[str, str]],
        provider: str,
        model: str
    ) -> int:
        """Warm cache with common phrases."""
        count = 0
        for source, target in phrases:
            key = self.generate_key(source, provider, model)
            if self.store(key, source, target, provider, model):
                count += 1
        return count
```

**Step 4: Run tests to verify they pass**

```bash
python -m pytest tests/test_cache/test_backend.py -v
# Expected: PASS
```

**Step 5: Commit cache implementation**

```bash
git add cmd/translation-worker/cache/ tests/test_cache/
git commit -m "feat(cache): implement translation cache with backends

- Add CacheBackend abstract interface
- Add FileCacheBackend for JSON sidecar files
- Add RedisCacheBackend for distributed caching
- Add CacheManager with key generation and store/retrieve
- Add deterministic cache key generation with SHA256
- Add cache warming support for common phrases
- Add unit tests for both backends
"
```

---

## Phase 4: Layout Preservation

### Task 6: Implement layout preservation for JA→EN expansion

**Files:**
- Create: `cmd/translation-worker/layout/preserver.py`
- Create: `tests/test_layout/test_preserver.py`

**Step 1: Write test for autofit calculation**

```python
# tests/test_layout/test_preserver.py
import pytest
from layout.preserver import AutofitCalculator, LayoutPreserver

def test_autofit_calculation():
    """Should calculate font size reduction for text expansion."""
    calc = AutofitCalculator()

    # JA text that expands to 2x English
    result = calc.calculate(
        source_text="営業報告書",  # 6 chars
        target_text="Business Report",  # ~16 chars with spaces
        source_font_size=18.0,
        bounds_width=100.0
    )

    assert result["new_font_size"] < 18.0
    assert result["expansion_ratio"] >= 1.5

def test_overflow_detection():
    """Should detect when text overflows container."""
    preserver = LayoutPreserver()

    will_overflow = preserver.will_overflow(
        text="This is a very long text that will definitely overflow the container",
        bounds_width=100,
        font_size=12,
        max_lines=2
    )

    assert will_overflow == True
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_layout/test_preserver.py -v
# Expected: ImportError
```

**Step 3: Implement layout preservation**

```python
# cmd/translation-worker/layout/preserver.py
from dataclasses import dataclass
from typing import Optional

@dataclass
class AutofitResult:
    """Result of autofit calculation."""
    new_font_size: float
    expansion_ratio: float
    original_font_size: float
    will_overflow: bool

@dataclass
class Rectangle:
    """A rectangular bounding box."""
    width: float
    height: float

@dataclass
class Font:
    """Font description."""
    size: float
    min_size: float = 8.0
    name: str = "Arial"

class AutofitCalculator:
    """Calculates font size adjustments for text expansion."""

    # Approximate character widths (relative)
    CHAR_WIDTHS = {
        "ja": 1.0,      # Japanese characters are roughly square
        "en": 0.5,      # English characters are ~half width
        "space": 0.3
    }

    def calculate(
        self,
        source_text: str,
        target_text: str,
        source_font_size: float,
        bounds: Rectangle,
        font: Font = None
    ) -> AutofitResult:
        """Calculate required font size adjustment."""
        if font is None:
            font = Font(size=source_font_size)

        # Estimate character counts
        source_chars = self._count_chars(source_text)
        target_chars = self._count_chars(target_text)

        # Calculate widths
        source_width = (
            source_chars["ja"] * self.CHAR_WIDTHS["ja"] +
            source_chars["en"] * self.CHAR_WIDTHS["en"] +
            source_chars["space"] * self.CHAR_WIDTHS["space"]
        ) * source_font_size

        target_width = (
            target_chars["ja"] * self.CHAR_WIDTHS["ja"] +
            target_chars["en"] * self.CHAR_WIDTHS["en"] +
            target_chars["space"] * self.CHAR_WIDTHS["space"]
        ) * source_font_size

        # Calculate expansion ratio
        expansion_ratio = target_width / source_width if source_width > 0 else 1.0

        # Calculate new font size
        if expansion_ratio > 1.0:
            new_font_size = source_font_size / expansion_ratio
            new_font_size = max(new_font_size, font.min_size)
        else:
            new_font_size = source_font_size

        # Check if still overflows
        will_overflow = (target_width / expansion_ratio) > bounds.width

        return AutofitResult(
            new_font_size=new_font_size,
            expansion_ratio=expansion_ratio,
            original_font_size=source_font_size,
            will_overflow=will_overflow
        )

    def _count_chars(self, text: str) -> dict:
        """Count characters by type."""
        counts = {"ja": 0, "en": 0, "space": 0}

        for char in text:
            code = ord(char)
            if char.isspace():
                counts["space"] += 1
            elif 0x4E00 <= code <= 0x9FFF or 0x3040 <= code <= 0x30FF:
                counts["ja"] += 1
            else:
                counts["en"] += 1

        return counts

class LayoutPreserver:
    """Preserves layout during JA→EN translation."""

    def __init__(
        self,
        strategy: str = "autofit",
        min_font_size_pct: float = 60.0,
        warn_threshold: float = 0.95
    ):
        self.strategy = strategy
        self.min_font_size_pct = min_font_size_pct
        self.warn_threshold = warn_threshold
        self.calculator = AutofitCalculator()

    def will_overflow(
        self,
        text: str,
        bounds_width: float,
        font_size: float,
        max_lines: int = 1
    ) -> bool:
        """Check if text will overflow its container."""
        # Simple estimation
        avg_char_width = font_size * 0.5  # Approximate
        estimated_width = len(text) * avg_char_width
        return estimated_width > bounds_width

    def adjust_font_size(
        self,
        source_text: str,
        target_text: str,
        current_font_size: float,
        bounds: Rectangle
    ) -> float:
        """Calculate adjusted font size for target text."""
        result = self.calculator.calculate(
            source_text=source_text,
            target_text=target_text,
            source_font_size=current_font_size,
            bounds=bounds
        )

        if result.will_overflow and self.strategy == "autofit":
            return result.new_font_size

        return current_font_size

    def check_forbidden_translations(self, text: str, forbidden: list) -> list:
        """Check for translations that should not be used."""
        found = []
        for forbidden_term in forbidden:
            if forbidden_term.lower() in text.lower():
                found.append(forbidden_term)
        return found
```

**Step 4: Run tests to verify they pass**

```bash
python -m pytest tests/test_layout/test_preserver.py -v
# Expected: PASS
```

**Step 5: Commit layout preservation**

```bash
git add cmd/translation-worker/layout/ tests/test_layout/
git commit -m "feat(layout): implement layout preservation for JA→EN expansion

- Add AutofitCalculator for font size adjustment
- Add LayoutPreserver with overflow detection
- Add character counting for JA/EN/space
- Add will_overflow() for text boundary checking
- Add adjust_font_size() for autofit calculation
- Add check_forbidden_translations() for glossary enforcement
- Add unit tests for autofit and overflow detection
"
```

---

## Phase 5: Document Parsers (with pymupdf)

### Task 7: Adopt pymupdf for PDF parsing

**Files:**
- Create: `cmd/translation-worker/parsers/pdf_parser.py`
- Create: `tests/test_parsers/test_pdf_parser.py`

**Step 1: Write test for PDF parser**

```python
# tests/test_parsers/test_pdf_parser.py
import pytest
from parsers.pdf_parser import pymupdfParser

def test_pdf_parser_extracts_text(tmp_path):
    """Should extract text from PDF using pymupdf."""
    # This test requires a sample PDF file
    parser = pymupdfParser()

    # Test parsing method exists
    assert hasattr(parser, 'parse')
    assert hasattr(parser, 'render')
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_parsers/test_pdf_parser.py -v
# Expected: ImportError
```

**Step 3: Implement pymupdf-based PDF parser**

```python
# cmd/translation-worker/parsers/pdf_parser.py
from typing import Optional
from ..plugins.base import ParserPlugin, ParsedDocument, Segment
import fitz  # pymupdf

class pymupdfParser(ParserPlugin):
    """PDF parser using pymupdf (MuPDF) for fast text extraction."""

    name = "pymupdf_pdf"
    version = "1.0.0"
    dependencies = ["pymupdf"]

    def supported_extensions(self) -> list:
        return [".pdf"]

    def parse(self, file_path: str) -> ParsedDocument:
        """Parse PDF and extract translatable text segments."""
        doc = fitz.open(file_path)

        segments = []
        for page_num in range(len(doc)):
            page = doc[page_num]
            text = page.get_text()

            if text.strip():
                segments.append(Segment(
                    id=f"page_{page_num}",
                    text=text.strip(),
                    context={
                        "type": "page",
                        "page_number": page_num + 1
                    }
                ))

        doc.close()

        return ParsedDocument(
            segments=segments,
            metadata={"format": "pdf"},
            format="pdf"
        )

    def render(self, doc: ParsedDocument, output_path: str) -> None:
        """Render translated PDF with original layout."""
        # Create new PDF
        out_doc = fitz.open()

        # For each segment, create a page with translated text
        for segment in doc.segments:
            page_num = segment.context.get("page_number", 1) - 1

            # Add new page (use first page of source as template)
            # In production, would load original PDF as template
            out_doc.new_page(width=595, height=842)  # A4 default

            # Insert translated text
            page = out_doc[page_num]
            page.insert_text(
                segment.text,
                fontsize=11,
                pos=(50, 800 - (page_num * 100))
            )

        out_doc.save(output_path)
        out_doc.close()
```

**Step 4: Run tests to verify they pass**

```bash
python -m pytest tests/test_parsers/test_pdf_parser.py -v
# Expected: PASS
```

**Step 5: Commit pymupdf parser**

```bash
git add cmd/translation-worker/parsers/pdf_parser.py tests/test_parsers/
git commit -m "feat(parsers): adopt pymupdf for PDF parsing

- Add pymupdfParser class using fitz (MuPDF binding)
- Add parse() method to extract text by page
- Add render() method to generate translated PDFs
- Add A4 page defaults for output
- Note: pymupdf is C-backed, ~10x faster than PyPDF2
- May eliminate need for custom C++ PDF parser
"
```

---

### Task 7.5: Implement PPTX parser using python-pptx

**Files:**
- Create: `cmd/translation-worker/parsers/pptx_parser.py`
- Create: `tests/test_parsers/test_pptx_parser.py`

**Step 1: Write failing test for PPTX text extraction**

```python
# tests/test_parsers/test_pptx_parser.py
import pytest
from cmd.translation_worker.parsers.pptx_parser import PPTXParser

@pytest.fixture
def sample_pptx(tmp_path):
    """Create a minimal test PPTX file."""
    from pptx import Presentation
    prs = Presentation()
    slide = prs.slides.add_slide(prs.slide_layouts[0])
    text_box = slide.shapes.add_textbox(100, 100, 300, 200)
    text_frame = text_box.text_frame
    text_frame.text = "Hello World"
    p = tmp_path / "test.pptx"
    prs.save(str(p))
    return str(p)

def test_pptx_parser_extract_text(sample_pptx):
    """Should extract text from PPTX slides."""
    parser = PPTXParser()
    segments = parser.parse(sample_pptx)
    assert len(segments) > 0
    assert any("Hello World" in s.text for s in segments)

def test_pptx_parser_slide_numbering(sample_pptx):
    """Should include slide number in segments."""
    parser = PPTXParser()
    segments = parser.parse(sample_pptx)
    assert all(s.metadata.get("slide_number") is not None for s in segments)

def test_pptx_parser_preserve_formatting(tmp_path):
    """Should preserve font size and formatting metadata."""
    from pptx import Presentation
    from pptx.util import Pt
    prs = Presentation()
    slide = prs.slides.add_slide(prs.slide_layouts[0])
    text_box = slide.shapes.add_textbox(100, 100, 300, 200)
    text_frame = text_box.text_frame
    p = text_frame.paragraphs[0]
    p.text = "Bold text"
    run = p.runs[0]
    run.font.bold = True
    run.font.size = Pt(14)
    p = tmp_path / "formatted.pptx"
    prs.save(str(p))

    parser = PPTXParser()
    segments = parser.parse(str(p))
    assert segments[0].metadata.get("font_size") == 14
    assert segments[0].metadata.get("bold") is True
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_parsers/test_pptx_parser.py -v
# Expected: FAIL with "PPTXParser not defined"
```

**Step 3: Write minimal PPTX parser implementation**

```python
# cmd/translation-worker/parsers/pptx_parser.py
from typing import List, Optional
from dataclasses import dataclass
from pathlib import Path

try:
    from pptx import Presentation
except ImportError:
    raise ImportError(
        "python-pptx is required. Install with: pip install python-pptx"
    )

@dataclass
class TextSegment:
    """A segment of text with formatting context."""
    text: str
    metadata: dict

class PPTXParser:
    """Parse PowerPoint PPTX files preserving formatting context."""

    def parse(self, file_path: str) -> List[TextSegment]:
        """Extract text segments from PPTX with slide context."""
        segments = []
        prs = Presentation(file_path)

        for slide_idx, slide in enumerate(prs.slides, start=1):
            for shape in slide.shapes:
                if not hasattr(shape, "text_frame"):
                    continue

                segments.extend(self._parse_text_frame(
                    shape.text_frame,
                    slide_number=slide_idx,
                    shape_position=(shape.left, shape.top)
                ))

        return segments

    def _parse_text_frame(
        self,
        text_frame,
        slide_number: int,
        shape_position: tuple
    ) -> List[TextSegment]:
        """Parse a text frame extracting text with formatting."""
        segments = []

        for para_idx, paragraph in enumerate(text_frame.paragraphs):
            if not paragraph.text.strip():
                continue

            for run in paragraph.runs:
                if not run.text.strip():
                    continue

                segments.append(TextSegment(
                    text=run.text,
                    metadata={
                        "slide_number": slide_number,
                        "paragraph_index": para_idx,
                        "position": shape_position,
                        "font_size": self._get_font_size(run),
                        "bold": run.font.bold,
                        "italic": run.font.italic,
                        "font_name": run.font.name,
                    }
                ))

        return segments

    def _get_font_size(self, run) -> Optional[int]:
        """Extract font size from run, converting to points."""
        if run.font.size is None:
            return None
        # pptx uses English Metric Units (EMU), convert to points
        return int(run.font.size / 12700)

    def render(
        self,
        original_path: str,
        translated_segments: List[TextSegment],
        output_path: str
    ) -> None:
        """Render translated segments back to PPTX file."""
        prs = Presentation(original_path)
        segment_idx = 0

        for slide in prs.slides:
            for shape in slide.shapes:
                if not hasattr(shape, "text_frame"):
                    continue

                for paragraph in shape.text_frame.paragraphs:
                    for run in paragraph.runs:
                        if segment_idx >= len(translated_segments):
                            break
                        run.text = translated_segments[segment_idx].text
                        segment_idx += 1

        prs.save(output_path)
```

**Step 4: Run test to verify it passes**

```bash
python -m pytest tests/test_parsers/test_pptx_parser.py -v
# Expected: PASS
```

**Step 5: Commit PPTX parser**

```bash
git add cmd/translation-worker/parsers/pptx_parser.py tests/test_parsers/
git commit -m "feat(parsers): add PPTX parser using python-pptx

- Add PPTXParser class with text extraction by slide
- Parse text frames with formatting metadata (font size, bold, italic)
- Add render() method to rebuild translated PPTX
- Add unit tests for slide numbering and formatting preservation
- Note: python-pptx is pure Python; C++ extension could provide 5-10x speedup
"
```

---

### Task 7.6: Implement DOCX parser using python-docx

**Files:**
- Create: `cmd/translation-worker/parsers/docx_parser.py`
- Create: `tests/test_parsers/test_docx_parser.py`

**Step 1: Write failing test for DOCX text extraction**

```python
# tests/test_parsers/test_docx_parser.py
import pytest
from cmd.translation_worker.parsers.docx_parser import DOCXParser

@pytest.fixture
def sample_docx(tmp_path):
    """Create a minimal test DOCX file."""
    from docx import Document
    doc = Document()
    doc.add_heading("Test Document", 0)
    p = doc.add_paragraph("This is a test paragraph with ")
    p.add_run("bold text").bold = True
    p = tmp_path / "test.docx"
    doc.save(str(p))
    return str(p)

def test_docx_parser_extract_paragraphs(sample_docx):
    """Should extract text from DOCX paragraphs."""
    parser = DOCXParser()
    segments = parser.parse(sample_docx)
    assert len(segments) > 0
    assert any("test paragraph" in s.text.lower() for s in segments)

def test_docx_parser_preserve_styles(sample_docx):
    """Should preserve paragraph and run-level formatting."""
    parser = DOCXParser()
    segments = parser.parse(sample_docx)
    # Find the bold text segment
    bold_segments = [s for s in segments if s.metadata.get("bold")]
    assert len(bold_segments) > 0

def test_docx_parser_table_handling(tmp_path):
    """Should extract text from tables."""
    from docx import Document
    doc = Document()
    table = doc.add_table(rows=2, cols=2)
    table.rows[0].cells[0].text = "Header 1"
    table.rows[0].cells[1].text = "Header 2"
    table.rows[1].cells[0].text = "Data 1"
    table.rows[1].cells[1].text = "Data 2"
    p = tmp_path / "table.docx"
    doc.save(str(p))

    parser = DOCXParser()
    segments = parser.parse(str(p))
    table_texts = [s.text for s in segments if s.metadata.get("is_table")]
    assert len(table_texts) >= 4
    assert "Header 1" in table_texts
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_parsers/test_docx_parser.py -v
# Expected: FAIL with "DOCXParser not defined"
```

**Step 3: Write minimal DOCX parser implementation**

```python
# cmd/translation-worker/parsers/docx_parser.py
from typing import List, Optional
from dataclasses import dataclass
from pathlib import Path

try:
    from docx import Document
except ImportError:
    raise ImportError(
        "python-docx is required. Install with: pip install python-docx"
    )

@dataclass
class TextSegment:
    """A segment of text with formatting context."""
    text: str
    metadata: dict

class DOCXParser:
    """Parse Word DOCX files preserving formatting context."""

    def parse(self, file_path: str) -> List[TextSegment]:
        """Extract text segments from DOCX with paragraph context."""
        segments = []
        doc = Document(file_path)

        para_idx = 0
        for paragraph in doc.paragraphs:
            if paragraph.text.strip():
                segments.extend(self._parse_paragraph(paragraph, para_idx))
                para_idx += 1

        # Handle tables
        for table_idx, table in enumerate(doc.tables):
            for row_idx, row in enumerate(table.rows):
                for cell_idx, cell in enumerate(row.cells):
                    for para in cell.paragraphs:
                        if para.text.strip():
                            segments.append(TextSegment(
                                text=para.text.strip(),
                                metadata={
                                    "is_table": True,
                                    "table_index": table_idx,
                                    "row_index": row_idx,
                                    "cell_index": cell_idx,
                                }
                            ))

        return segments

    def _parse_paragraph(
        self,
        paragraph,
        para_idx: int
    ) -> List[TextSegment]:
        """Parse a paragraph extracting runs with formatting."""
        segments = []

        # Detect paragraph style
        style_name = paragraph.style.name if paragraph.style else "Normal"

        for run in paragraph.runs:
            if not run.text.strip():
                continue

            segments.append(TextSegment(
                text=run.text,
                metadata={
                    "paragraph_index": para_idx,
                    "style": style_name,
                    "alignment": str(paragraph.alignment),
                    "font_size": self._get_font_size(run),
                    "bold": run.bold,
                    "italic": run.italic,
                    "underline": run.underline,
                    "font_name": run.font.name,
                }
            ))

        return segments

    def _get_font_size(self, run) -> Optional[int]:
        """Extract font size from run in points."""
        if run.font.size is None:
            return None
        return int(run.font.size)

    def render(
        self,
        original_path: str,
        translated_segments: List[TextSegment],
        output_path: str
    ) -> None:
        """Render translated segments back to DOCX file."""
        doc = Document(original_path)
        segment_idx = 0

        # Replace paragraph text
        for paragraph in doc.paragraphs:
            for run in paragraph.runs:
                if segment_idx >= len(translated_segments):
                    break
                # Skip table segments (they have is_table metadata)
                if translated_segments[segment_idx].metadata.get("is_table"):
                    continue
                run.text = translated_segments[segment_idx].text
                segment_idx += 1

        # Replace table text
        for table in doc.tables:
            for row in table.rows:
                for cell in row.cells:
                    for para in cell.paragraphs:
                        for run in para.runs:
                            if segment_idx >= len(translated_segments):
                                break
                            if not translated_segments[segment_idx].metadata.get("is_table"):
                                segment_idx += 1
                                continue
                            run.text = translated_segments[segment_idx].text
                            segment_idx += 1

        doc.save(output_path)
```

**Step 4: Run test to verify it passes**

```bash
python -m pytest tests/test_parsers/test_docx_parser.py -v
# Expected: PASS
```

**Step 5: Commit DOCX parser**

```bash
git add cmd/translation-worker/parsers/docx_parser.py tests/test_parsers/
git commit -m "feat(parsers): add DOCX parser using python-docx

- Add DOCXParser class with paragraph and run-level parsing
- Preserve paragraph styles, alignment, and text formatting
- Handle table extraction with cell position metadata
- Add render() method for DOCX reconstruction
- Add unit tests for table handling and style preservation
- Note: python-docx is pure Python; C++ extension could provide 3-5x speedup
"
```

---

### Task 7.7: Implement XLSX parser using openpyxl

**Files:**
- Create: `cmd/translation-worker/parsers/xlsx_parser.py`
- Create: `tests/test_parsers/test_xlsx_parser.py`

**Step 1: Write failing test for XLSX text extraction**

```python
# tests/test_parsers/test_xlsx_parser.py
import pytest
from cmd.translation_worker.parsers.xlsx_parser import XLSXParser

@pytest.fixture
def sample_xlsx(tmp_path):
    """Create a minimal test XLSX file."""
    from openpyxl import Workbook
    wb = Workbook()
    ws = wb.active
    ws.title = "Test Sheet"
    ws["A1"] = "Header 1"
    ws["B1"] = "Header 2"
    ws["A2"] = "Data 1"
    ws["B2"] = "Data 2"
    p = tmp_path / "test.xlsx"
    wb.save(str(p))
    return str(p)

def test_xlsx_parser_extract_cells(sample_xlsx):
    """Should extract text from XLSX cells."""
    parser = XLSXParser()
    segments = parser.parse(sample_xlsx)
    assert len(segments) >= 4
    cell_texts = [s.text for s in segments]
    assert "Header 1" in cell_texts
    assert "Data 2" in cell_texts

def test_xlsx_parser_cell_reference(sample_xlsx):
    """Should include cell reference in metadata."""
    parser = XLSXParser()
    segments = parser.parse(sample_xlsx)
    assert all(s.metadata.get("cell") for s in segments)
    # Check for specific cells
    a1_segments = [s for s in segments if s.metadata["cell"] == "A1"]
    assert len(a1_segments) == 1

def test_xlsx_parser_multiple_sheets(tmp_path):
    """Should handle multiple worksheets."""
    from openpyxl import Workbook
    wb = Workbook()
    ws1 = wb.active
    ws1.title = "Sheet1"
    ws1["A1"] = "First Sheet"
    ws2 = wb.create_sheet("Sheet2")
    ws2["A1"] = "Second Sheet"
    p = tmp_path / "multi.xlsx"
    wb.save(str(p))

    parser = XLSXParser()
    segments = parser.parse(str(p))
    sheet_names = {s.metadata["sheet"] for s in segments}
    assert "Sheet1" in sheet_names
    assert "Sheet2" in sheet_names

def test_xlsx_parser_preserve_formulas(tmp_path):
    """Should distinguish formulas from values."""
    from openpyxl import Workbook
    wb = Workbook()
    ws = wb.active
    ws["A1"] = 10
    ws["A2"] = 20
    ws["A3"] = "=SUM(A1:A2)"
    p = tmp_path / "formula.xlsx"
    wb.save(str(p))

    parser = XLSXParser()
    segments = parser.parse(str(p))
    # Find the formula cell
    formula_segments = [s for s in segments if s.metadata.get("is_formula")]
    assert len(formula_segments) == 1
    assert "SUM" in formula_segments[0].metadata.get("formula", "")
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_parsers/test_xlsx_parser.py -v
# Expected: FAIL with "XLSXParser not defined"
```

**Step 3: Write minimal XLSX parser implementation**

```python
# cmd/translation-worker/parsers/xlsx_parser.py
from typing import List
from dataclasses import dataclass
from pathlib import Path

try:
    from openpyxl import load_workbook
except ImportError:
    raise ImportError(
        "openpyxl is required. Install with: pip install openpyxl"
    )

@dataclass
class TextSegment:
    """A segment of text with formatting context."""
    text: str
    metadata: dict

class XLSXParser:
    """Parse Excel XLSX files with cell reference context."""

    def __init__(self, translate_formulas: bool = False):
        """
        Initialize XLSX parser.

        Args:
            translate_formulas: If False, preserve formulas unchanged.
                                If True, translate formula text (risky).
        """
        self.translate_formulas = translate_formulas

    def parse(self, file_path: str) -> List[TextSegment]:
        """Extract text segments from XLSX with cell context."""
        segments = []
        wb = load_workbook(file_path, data_only=False)

        for sheet in wb.worksheets:
            for row in sheet.iter_rows():
                for cell in row:
                    if cell.value is None or str(cell.value).strip() == "":
                        continue

                    cell_value = str(cell.value)
                    is_formula = cell_value.startswith("=")

                    segments.append(TextSegment(
                        text=cell_value,
                        metadata={
                            "sheet": sheet.title,
                            "cell": cell.coordinate,
                            "row": cell.row,
                            "column": cell.column,
                            "is_formula": is_formula,
                            "formula": cell_value if is_formula else None,
                            "data_type": str(cell.data_type),
                        }
                    ))

        return segments

    def render(
        self,
        original_path: str,
        translated_segments: List[TextSegment],
        output_path: str
    ) -> None:
        """Render translated segments back to XLSX file."""
        wb = load_workbook(original_path)
        segment_map = {
            (s.metadata["sheet"], s.metadata["cell"]): s
            for s in translated_segments
        }

        for sheet in wb.worksheets:
            for row in sheet.iter_rows():
                for cell in row:
                    key = (sheet.title, cell.coordinate)
                    if key not in segment_map:
                        continue

                    segment = segment_map[key]

                    # Preserve formulas unless explicitly translating them
                    if segment.metadata.get("is_formula") and not self.translate_formulas:
                        continue

                    cell.value = segment.text

        wb.save(output_path)
```

**Step 4: Run test to verify it passes**

```bash
python -m pytest tests/test_parsers/test_xlsx_parser.py -v
# Expected: PASS
```

**Step 5: Commit XLSX parser**

```bash
git add cmd/translation-worker/parsers/xlsx_parser.py tests/test_parsers/
git commit -m "feat(parsers): add XLSX parser using openpyxl

- Add XLSXParser class with cell-level text extraction
- Include sheet name, cell reference, and formula metadata
- Preserve formulas by default (configurable via translate_formulas)
- Handle multiple worksheets correctly
- Add unit tests for formulas, multi-sheet, and cell references
"
```

---

## Phase 6: Provider Abstraction Layer

### Task 8: Design provider interface using Protocol

**Files:**
- Create: `cmd/translation-worker/providers/base.py`
- Create: `cmd/translation-worker/providers/factory.py`
- Modify: `cmd/translation-worker/providers/base.py` (if exists from Task 2)

**Step 1: Update providers base to use Protocol**

```python
# cmd/translation-worker/providers/base.py
from typing import Protocol, runtime_checkable, Optional
from dataclasses import dataclass

@runtime_checkable
class TranslationProvider(Protocol):
    """Translation provider protocol using structural subtyping."""

    def translate(self, text: str, source_lang: str = "ja", target_lang: str = "en") -> "TranslationResult":
        """Translate text from source to target language."""
        ...

    def is_available(self) -> bool:
        """Check if provider is available (auth valid, service up)."""
        ...

    @property
    def name(self) -> str:
        """Provider name."""
        ...

@dataclass
class TranslationResult:
    """Result of a translation operation."""
    success: bool
    translated_text: str
    confidence: float
    provider: str
    model: str
    error: Optional[str] = None
```

**Step 2: Update provider factory**

```python
# cmd/translation-worker/providers/factory.py
from typing import list
from .base import TranslationProvider

PROVIDER_REGISTRY: dict[str, type] = {}

def register_provider(name: str, provider_class: type):
    """Register a provider class."""
    PROVIDER_REGISTRY[name] = provider_class

def create_provider(config: dict) -> TranslationProvider:
    """Create a provider instance from configuration."""
    provider_type = config.get("type") or config.get("provider")
    if provider_type not in PROVIDER_REGISTRY:
        raise ValueError(f"Unknown provider type: {provider_type}")

    provider_class = PROVIDER_REGISTRY[provider_type]
    return provider_class(config)

def create_routing_chain(configs: list) -> list[TranslationProvider]:
    """Create a chain of providers for fallback routing."""
    return [create_provider(c) for c in configs]
```

**Step 3: Commit provider abstraction update**

```bash
git add cmd/translation-worker/providers/
git commit -m "feat(providers): update provider interface to use Protocol

- Change from ABC to Protocol for structural subtyping
- Add TranslationResult dataclass
- Add PROVIDER_REGISTRY for dynamic provider registration
- Add register_provider() for extensibility
- Maintain compatibility with existing provider implementations
"
```

---

### Task 9: Implement Anthropic provider (unchanged from original)

**Files:**
- Create: `cmd/translation-worker/providers/anthropic.py`
- Create: `tests/test_providers/test_anthropic.py`

*(Implementation remains same as original plan - provider uses Protocol via duck typing)*

---

### Task 10: Implement OpenAI provider (unchanged from original)

*(Implementation remains same as original plan)*

---

### Task 11: Implement CLI tool provider (unchanged from original)

*(Implementation remains same as original plan)*

---

### Task 11.5: Implement Gemini provider

**Files:**
- Create: `cmd/translation-worker/providers/gemini.py`
- Create: `tests/test_providers/test_gemini.py`

**Step 1: Write failing test for Gemini provider**

```python
# tests/test_providers/test_gemini.py
import pytest
from cmd.translation_worker.providers.gemini import GeminiProvider

@pytest.fixture
def gemini_provider(monkeypatch):
    """Create Gemini provider with test configuration."""
    monkeypatch.setenv("GEMINI_API_KEY", "test-key")
    return GeminiProvider(model="gemini-2.5-flash")

def test_gemini_provider_initialization(gemini_provider):
    """Should initialize with API key from environment."""
    assert gemini_provider.name == "gemini"
    assert gemini_provider.model == "gemini-2.5-flash"

def test_gemini_provider_is_available(gemini_provider, monkeypatch):
    """Should check availability via API key presence."""
    # With key
    assert gemini_provider.is_available() is True

    # Without key
    monkeypatch.delenv("GEMINI_API_KEY")
    provider_no_key = GeminiProvider(model="gemini-2.5-flash")
    assert provider_no_key.is_available() is False

def test_gemini_provider_translate(gemini_provider, requests_mock):
    """Should translate text using Gemini API."""
    requests_mock.post(
        "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent",
        json={
            "candidates": [{
                "content": {
                    "parts": [{"text": "Hello"}]
                }
            }]
        }
    )

    result = gemini_provider.translate(
        text="こんにちは",
        source_lang="ja",
        target_lang="en"
    )

    assert result.success is True
    assert result.translated_text == "Hello"
    assert result.provider == "gemini"
    assert result.model == "gemini-2.5-flash"

def test_gemini_provider_error_handling(gemini_provider, requests_mock):
    """Should handle API errors gracefully."""
    requests_mock.post(
        "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent",
        status_code=400,
        json={"error": {"message": "Invalid request"}}
    )

    result = gemini_provider.translate(
        text="test",
        source_lang="en",
        target_lang="ja"
    )

    assert result.success is False
    assert "Invalid request" in result.error
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_providers/test_gemini.py -v
# Expected: FAIL with "GeminiProvider not defined"
```

**Step 3: Write Gemini provider implementation**

```python
# cmd/translation-worker/providers/gemini.py
import os
from typing import Optional
from dataclasses import dataclass
import requests

from cmd.translation_worker.providers.base import (
    TranslationProvider,
    TranslationResult
)

@dataclass
class GeminiConfig:
    """Gemini provider configuration."""
    api_key: str
    base_url: str = "https://generativelanguage.googleapis.com/v1beta"
    model: str = "gemini-2.5-flash"
    timeout: int = 60

class GeminiProvider:
    """Google Gemini translation provider."""

    def __init__(self, model: str = "gemini-2.5-flash"):
        """
        Initialize Gemini provider.

        Args:
            model: Model name (gemini-2.5-flash, gemini-2.5-pro, etc.)
        """
        self._config = GeminiConfig(
            api_key=os.environ.get("GEMINI_API_KEY", ""),
            model=model
        )

    @property
    def name(self) -> str:
        """Provider name."""
        return "gemini"

    def is_available(self) -> bool:
        """Check if provider is available."""
        return bool(self._config.api_key)

    def translate(
        self,
        text: str,
        source_lang: str = "ja",
        target_lang: str = "en"
    ) -> TranslationResult:
        """
        Translate text using Gemini API.

        Args:
            text: Source text to translate
            source_lang: Source language code
            target_lang: Target language code

        Returns:
            TranslationResult with translated text or error
        """
        if not self.is_available():
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._config.model,
                error="GEMINI_API_KEY not set"
            )

        headers = {"Content-Type": "application/json"}

        prompt = self._build_prompt(text, source_lang, target_lang)

        payload = {
            "contents": [{
                "parts": [{"text": prompt}]
            }],
            "generationConfig": {
                "temperature": 0.3,
                "maxOutputTokens": 4096,
            }
        }

        url = (
            f"{self._config.base_url}/models/{self._config.model}:generateContent"
            f"?key={self._config.api_key}"
        )

        try:
            response = requests.post(
                url,
                json=payload,
                headers=headers,
                timeout=self._config.timeout
            )
            response.raise_for_status()

            data = response.json()
            translated_text = data["candidates"][0]["content"]["parts"][0]["text"]

            return TranslationResult(
                success=True,
                translated_text=translated_text,
                confidence=0.9,  # Gemini doesn't provide confidence scores
                provider=self.name,
                model=self._config.model
            )

        except requests.exceptions.RequestException as e:
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._config.model,
                error=str(e)
            )
        except (KeyError, IndexError) as e:
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._config.model,
                error=f"Unexpected response format: {e}"
            )

    def _build_prompt(
        self,
        text: str,
        source_lang: str,
        target_lang: str
    ) -> str:
        """Build translation prompt for Gemini."""
        return f"""Translate the following text from {source_lang} to {target_lang}.

Output ONLY the translated text, no explanations or additional content.

Text:
{text}"""
```

**Step 4: Run test to verify it passes**

```bash
python -m pytest tests/test_providers/test_gemini.py -v
# Expected: PASS
```

**Step 5: Commit Gemini provider**

```bash
git add cmd/translation-worker/providers/gemini.py tests/test_providers/
git commit -m "feat(providers): add Gemini provider

- Add GeminiProvider class with gemini-2.5-flash/pro support
- Read GEMINI_API_KEY from environment
- Implement translate() method with REST API calls
- Add unit tests for availability checking and translation
- Handle API errors gracefully with structured error responses
"
```

---

### Task 11.6: Implement Ollama provider (local models)

**Files:**
- Create: `cmd/translation-worker/providers/ollama.py`
- Create: `tests/test_providers/test_ollama.py`

**Step 1: Write failing test for Ollama provider**

```python
# tests/test_providers/test_ollama.py
import pytest
from cmd.translation_worker.providers.ollama import OllamaProvider

@pytest.fixture
def ollama_provider():
    """Create Ollama provider with test configuration."""
    return OllamaProvider(
        base_url="http://localhost:11434",
        model="llama3.1:8b"
    )

def test_ollama_provider_initialization(ollama_provider):
    """Should initialize with base URL and model."""
    assert ollama_provider.name == "ollama"
    assert ollama_provider.model == "llama3.1:8b"
    assert ollama_provider.base_url == "http://localhost:11434"

def test_ollama_provider_is_available(ollama_provider, requests_mock):
    """Should check availability via Ollama /api/tags endpoint."""
    # Ollama is available
    requests_mock.get(
        "http://localhost:11434/api/tags",
        json={"models": [{"name": "llama3.1:8b"}]}
    )
    assert ollama_provider.is_available() is True

    # Ollama is not available
    requests_mock.get(
        "http://localhost:11434/api/tags",
        status_code=503
    )
    assert ollama_provider.is_available() is False

def test_ollama_provider_translate(ollama_provider, requests_mock):
    """Should translate using local Ollama model."""
    requests_mock.post(
        "http://localhost:11434/api/generate",
        json={
            "model": "llama3.1:8b",
            "response": "Hello",
            "done": True
        }
    )

    result = ollama_provider.translate(
        text="こんにちは",
        source_lang="ja",
        target_lang="en"
    )

    assert result.success is True
    assert result.translated_text == "Hello"
    assert result.provider == "ollama"

def test_ollama_provider_streaming(ollama_provider, requests_mock):
    """Should handle streaming responses from Ollama."""
    # Ollama returns multiple JSON objects for streaming
    responses = [
        {"model": "llama3.1:8b", "response": "Hel", "done": False},
        {"model": "llama3.1:8b", "response": "lo", "done": False},
        {"model": "llama3.1:8b", "response": "", "done": True},
    ]

    def stream_response(request, context):
        import json
        # Return concatenated JSON responses (Ollama format)
        return "\n".join(json.dumps(r) for r in responses)

    requests_mock.post(
        "http://localhost:11434/api/generate",
        text=stream_response(None, None)
    )

    result = ollama_provider.translate(
        text="こんにちは",
        source_lang="ja",
        target_lang="en"
    )

    assert result.success is True
    assert result.translated_text == "Hello"

def test_ollama_provider_model_list(ollama_provider, requests_mock):
    """Should list available models from Ollama."""
    requests_mock.get(
        "http://localhost:11434/api/tags",
        json={
            "models": [
                {"name": "llama3.1:8b", "size": 4882919328},
                {"name": "llama3.1:70b", "size": 42101985824},
                {"name": "qwen2.5:72b", "size": 43807961024},
            ]
        }
    )

    models = ollama_provider.list_models()
    assert len(models) == 3
    assert "llama3.1:8b" in models
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_providers/test_ollama.py -v
# Expected: FAIL with "OllamaProvider not defined"
```

**Step 3: Write Ollama provider implementation**

```python
# cmd/translation-worker/providers/ollama.py
import json
from typing import List, Optional
from dataclasses import dataclass
import requests

from cmd.translation_worker.providers.base import (
    TranslationProvider,
    TranslationResult
)

@dataclass
class OllamaConfig:
    """Ollama provider configuration."""
    base_url: str = "http://localhost:11434"
    model: str = "llama3.1:8b"
    timeout: int = 300  # Local models can be slow

class OllamaProvider:
    """Ollama local model provider for translation."""

    def __init__(
        self,
        base_url: str = "http://localhost:11434",
        model: str = "llama3.1:8b"
    ):
        """
        Initialize Ollama provider.

        Args:
            base_url: Ollama API base URL
            model: Model name (llama3.1:8b, llama3.1:70b, qwen2.5:72b, etc.)
        """
        self._config = OllamaConfig(base_url=base_url, model=model)

    @property
    def name(self) -> str:
        """Provider name."""
        return "ollama"

    @property
    def base_url(self) -> str:
        """Base URL for API requests."""
        return self._config.base_url

    @property
    def model(self) -> str:
        """Model name."""
        return self._config.model

    def is_available(self) -> bool:
        """Check if Ollama server is running."""
        try:
            response = requests.get(
                f"{self._config.base_url}/api/tags",
                timeout=5
            )
            return response.status_code == 200
        except requests.exceptions.RequestException:
            return False

    def list_models(self) -> List[str]:
        """List available models from Ollama."""
        try:
            response = requests.get(
                f"{self._config.base_url}/api/tags",
                timeout=10
            )
            response.raise_for_status()
            data = response.json()
            return [m["name"] for m in data.get("models", [])]
        except requests.exceptions.RequestException:
            return []

    def translate(
        self,
        text: str,
        source_lang: str = "ja",
        target_lang: str = "en"
    ) -> TranslationResult:
        """
        Translate text using local Ollama model.

        Args:
            text: Source text to translate
            source_lang: Source language code
            target_lang: Target language code

        Returns:
            TranslationResult with translated text or error
        """
        if not self.is_available():
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._config.model,
                error="Ollama server not available"
            )

        prompt = self._build_prompt(text, source_lang, target_lang)

        payload = {
            "model": self._config.model,
            "prompt": prompt,
            "stream": False,
            "options": {
                "temperature": 0.3,
                "num_predict": 4096,
            }
        }

        try:
            response = requests.post(
                f"{self._config.base_url}/api/generate",
                json=payload,
                timeout=self._config.timeout
            )
            response.raise_for_status()

            data = response.json()
            translated_text = data.get("response", "").strip()

            return TranslationResult(
                success=True,
                translated_text=translated_text,
                confidence=0.85,  # Local models generally less consistent
                provider=self.name,
                model=self._config.model
            )

        except requests.exceptions.RequestException as e:
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._config.model,
                error=str(e)
            )

    def _build_prompt(
        self,
        text: str,
        source_lang: str,
        target_lang: str
    ) -> str:
        """Build translation prompt for Ollama."""
        return f"""You are a professional translator. Translate the following text from {source_lang} to {target_lang}.

Output ONLY the translation, nothing else.

Text: {text}

Translation:"""
```

**Step 4: Run test to verify it passes**

```bash
python -m pytest tests/test_providers/test_ollama.py -v
# Expected: PASS
```

**Step 5: Commit Ollama provider**

```bash
git add cmd/translation-worker/providers/ollama.py tests/test_providers/
git commit -m "feat(providers): add Ollama provider for local models

- Add OllamaProvider class for local model inference
- Support models: llama3.1:8b, llama3.1:70b, qwen2.5:72b
- Implement is_available() via /api/tags health check
- Add list_models() to discover available local models
- Handle Ollama's streaming response format
- Add unit tests for availability, translation, and model listing
"
```

---

### Task 11.7: Implement LM Studio provider

**Files:**
- Create: `cmd/translation-worker/providers/lm_studio.py`
- Create: `tests/test_providers/test_lm_studio.py`

**Step 1: Write failing test for LM Studio provider**

```python
# tests/test_providers/test_lm_studio.py
import pytest
from cmd.translation_worker.providers.lm_studio import LMStudioProvider

@pytest.fixture
def lm_studio_provider():
    """Create LM Studio provider with test configuration."""
    return LMStudioProvider(
        base_url="http://localhost:1234/v1",
        model="local-model"
    )

def test_lm_studio_provider_initialization(lm_studio_provider):
    """Should initialize with OpenAI-compatible endpoint."""
    assert lm_studio_provider.name == "lm_studio"
    assert lm_studio_provider.model == "local-model"
    assert lm_studio_provider.base_url == "http://localhost:1234/v1"

def test_lm_studio_provider_is_available(lm_studio_provider, requests_mock):
    """Should check availability via /models endpoint."""
    # LM Studio is available
    requests_mock.get(
        "http://localhost:1234/v1/models",
        json={"data": [{"id": "local-model"}]}
    )
    assert lm_studio_provider.is_available() is True

    # LM Studio is not available
    requests_mock.get(
        "http://localhost:1234/v1/models",
        status_code=503
    )
    assert lm_studio_provider.is_available() is False

def test_lm_studio_provider_translate(lm_studio_provider, requests_mock):
    """Should translate using LM Studio's OpenAI-compatible API."""
    requests_mock.post(
        "http://localhost:1234/v1/chat/completions",
        json={
            "choices": [{
                "message": {
                    "content": "Hello"
                },
                "finish_reason": "stop"
            }]
        }
    )

    result = lm_studio_provider.translate(
        text="こんにちは",
        source_lang="ja",
        target_lang="en"
    )

    assert result.success is True
    assert result.translated_text == "Hello"
    assert result.provider == "lm_studio"

def test_lm_studio_provider_model_list(lm_studio_provider, requests_mock):
    """Should list available models from LM Studio."""
    requests_mock.get(
        "http://localhost:1234/v1/models",
        json={
            "data": [
                {"id": "llama-3.1-8b-instruct"},
                {"id": "qwen2.5-72b-instruct"},
            ]
        }
    )

    models = lm_studio_provider.list_models()
    assert len(models) == 2
    assert "llama-3.1-8b-instruct" in models
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_providers/test_lm_studio.py -v
# Expected: FAIL with "LMStudioProvider not defined"
```

**Step 3: Write LM Studio provider implementation**

```python
# cmd/translation-worker/providers/lm_studio.py
from typing import List
from dataclasses import dataclass
import requests

from cmd.translation_worker.providers.base import (
    TranslationProvider,
    TranslationResult
)

@dataclass
class LMStudioConfig:
    """LM Studio provider configuration."""
    base_url: str = "http://localhost:1234/v1"
    model: str = "local-model"
    timeout: int = 300

class LMStudioProvider:
    """LM Studio provider using OpenAI-compatible API."""

    def __init__(
        self,
        base_url: str = "http://localhost:1234/v1",
        model: str = "local-model"
    ):
        """
        Initialize LM Studio provider.

        Args:
            base_url: LM Studio API base URL (OpenAI-compatible)
            model: Model name (must match loaded model in LM Studio)
        """
        self._config = LMStudioConfig(base_url=base_url, model=model)

    @property
    def name(self) -> str:
        """Provider name."""
        return "lm_studio"

    @property
    def base_url(self) -> str:
        """Base URL for API requests."""
        return self._config.base_url

    @property
    def model(self) -> str:
        """Model name."""
        return self._config.model

    def is_available(self) -> bool:
        """Check if LM Studio server is running."""
        try:
            response = requests.get(
                f"{self._config.base_url}/models",
                timeout=5
            )
            return response.status_code == 200
        except requests.exceptions.RequestException:
            return False

    def list_models(self) -> List[str]:
        """List available models from LM Studio."""
        try:
            response = requests.get(
                f"{self._config.base_url}/models",
                timeout=10
            )
            response.raise_for_status()
            data = response.json()
            return [m["id"] for m in data.get("data", [])]
        except requests.exceptions.RequestException:
            return []

    def translate(
        self,
        text: str,
        source_lang: str = "ja",
        target_lang: str = "en"
    ) -> TranslationResult:
        """
        Translate text using LM Studio model.

        Args:
            text: Source text to translate
            source_lang: Source language code
            target_lang: Target language code

        Returns:
            TranslationResult with translated text or error
        """
        if not self.is_available():
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._config.model,
                error="LM Studio server not available"
            )

        headers = {"Content-Type": "application/json"}

        system_prompt = self._build_system_prompt(source_lang, target_lang)
        user_prompt = f"Translate this text from {source_lang} to {target_lang}:\n\n{text}"

        payload = {
            "model": self._config.model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt}
            ],
            "temperature": 0.3,
            "max_tokens": 4096,
        }

        try:
            response = requests.post(
                f"{self._config.base_url}/chat/completions",
                json=payload,
                headers=headers,
                timeout=self._config.timeout
            )
            response.raise_for_status()

            data = response.json()
            translated_text = data["choices"][0]["message"]["content"]

            return TranslationResult(
                success=True,
                translated_text=translated_text.strip(),
                confidence=0.85,  # Local models generally less consistent
                provider=self.name,
                model=self._config.model
            )

        except (requests.exceptions.RequestException, KeyError, IndexError) as e:
            return TranslationResult(
                success=False,
                translated_text="",
                confidence=0.0,
                provider=self.name,
                model=self._config.model,
                error=str(e)
            )

    def _build_system_prompt(
        self,
        source_lang: str,
        target_lang: str
    ) -> str:
        """Build system prompt for LM Studio."""
        return f"""You are a professional translator specializing in {source_lang} to {target_lang} translation.

Rules:
1. Translate ONLY the provided text, no explanations
2. Preserve formatting and structure where possible
3. Output the translation directly, no preamble"""
```

**Step 4: Run test to verify it passes**

```bash
python -m pytest tests/test_providers/test_lm_studio.py -v
# Expected: PASS
```

**Step 5: Commit LM Studio provider**

```bash
git add cmd/translation-worker/providers/lm_studio.py tests/test_providers/
git commit -m "feat(providers): add LM Studio provider

- Add LMStudioProvider class using OpenAI-compatible API
- Support any locally loaded model in LM Studio
- Implement is_available() via /models health check
- Add list_models() to discover loaded models
- Use chat/completions endpoint with system/user messages
- Add unit tests for availability, translation, and model listing
"
```

---

## Phase 7: Redis Job Queue & State Management

### Task 12: Implement Redis job queue for horizontal scaling

**Files:**
- Create: `cmd/translation-worker/queue/job.py`
- Create: `cmd/translation-worker/queue/manager.py`
- Create: `tests/test_queue/test_manager.py`

**Step 1: Write test for job queue**

```python
# tests/test_queue/test_manager.py
import pytest
from queue.manager import JobManager, JobState

def test_enqueue_and_dequeue():
    """Should enqueue and dequeue jobs from Redis."""
    manager = JobManager()

    # Enqueue
    job_id = manager.enqueue({
        "source_file": "/watch/incoming/test.pptx",
        "source_lang": "ja",
        "target_lang": "en"
    })

    assert job_id is not None

    # Dequeue
    job = manager.dequeue(worker_id="worker-1")
    assert job is not None
    assert job["source_file"] == "/watch/incoming/test.pptx"

def test_job_state_transitions():
    """Job state should transition correctly."""
    manager = JobManager()

    job_id = "test-job-123"

    # Initial state
    manager.set_state(job_id, JobState.PENDING)
    assert manager.get_state(job_id) == JobState.PENDING

    # Transition to processing
    manager.set_state(job_id, JobState.PROCESSING)
    assert manager.get_state(job_id) == JobState.PROCESSING

    # Complete
    manager.set_state(job_id, JobState.COMPLETED)
    assert manager.get_state(job_id) == JobState.COMPLETED
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_queue/test_manager.py -v
# Expected: ImportError
```

**Step 3: Implement job queue and state management**

```python
# cmd/translation-worker/queue/job.py
from dataclasses import dataclass, field
from enum import Enum
from typing import Optional
import uuid

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
    state: JobState = JobState.PENDING
    worker_id: Optional[str] = None
    progress: float = 0.0
    error: Optional[str] = None
    metadata: dict = field(default_factory=dict)
    created_at: Optional[str] = None
    started_at: Optional[str] = None
    completed_at: Optional[str] = None

    def __post_init__(self):
        if self.id is None:
            self.id = str(uuid.uuid4())
```

```python
# cmd/translation-worker/queue/manager.py
from typing import Optional
from enum import Enum
import json
from datetime import datetime
import time

from .job import Job, JobState

class JobManager:
    """Manages job queue and state in Redis."""

    # Redis key prefixes
    QUEUE_PREFIX = "trans:queue:"
    JOB_PREFIX = "trans:job:"
    STATE_PREFIX = "trans:state:"
    CHECKPOINT_PREFIX = "trans:checkpoint:"

    # Priorities (lower = higher priority)
    PRIORITIES = {
        "urgent": 0,
        "normal": 1,
        "bulk": 2
    }

    def __init__(
        self,
        redis_host: str = "localhost",
        redis_port: int = 6379,
        redis_db: int = 0
    ):
        try:
            import redis
            self.redis = redis.Redis(
                host=redis_host,
                port=redis_port,
                db=redis_db,
                decode_responses=False
            )
        except ImportError:
            raise RuntimeError("redis package not installed")

        self.checkpoint_ttl = 604800  # 7 days

    def enqueue(
        self,
        job_data: dict,
        priority: str = "normal",
        delay_seconds: int = 0
    ) -> Optional[str]:
        """Enqueue a job for translation."""
        from queue.job import Job

        job = Job(
            id=None,
            source_file=job_data.get("source_file"),
            state=JobState.PENDING,
            metadata=job_data
        )

        # Store job data
        job_key = f"{self.JOB_PREFIX}{job.id}"
        job_json = json.dumps({
            **job_data,
            "id": job.id,
            "state": job.state.value,
            "created_at": datetime.utcnow().isoformat()
        })

        self.redis.set(job_key, job_json, ex=86400)  # 24 hour TTL

        # Add to queue
        queue_key = f"{self.QUEUE_PREFIX}{priority}"
        score = time.time() + delay_seconds

        self.redis.zadd(queue_key, {job.id: score})

        return job.id

    def dequeue(self, worker_id: str, timeout: int = 1) -> Optional[dict]:
        """Dequeue next available job."""
        # Check queues by priority
        for priority in ["urgent", "normal", "bulk"]:
            queue_key = f"{self.QUEUE_PREFIX}{priority}"

            # Get and remove oldest job
            result = self.redis.zpopmin(queue_key)

            if result:
                job_id, score = result
                return self._get_job(job_id)

        return None

    def _get_job(self, job_id: str) -> Optional[dict]:
        """Get job data."""
        job_key = f"{self.JOB_PREFIX}{job_id}"
        data = self.redis.get(job_key)
        if data:
            return json.loads(data)
        return None

    def set_state(self, job_id: str, state: JobState, worker_id: str = None):
        """Update job state."""
        state_key = f"{self.STATE_PREFIX}{job_id}"
        self.redis.set(
            state_key,
            json.dumps({
                "state": state.value,
                "worker_id": worker_id,
                "timestamp": datetime.utcnow().isoformat()
            }),
            ex=86400
        )

        # Update job data too
        job_key = f"{self.JOB_PREFIX}{job_id}"
        job_data = self._get_job(job_id)
        if job_data:
            job_data["state"] = state.value
            if worker_id:
                job_data["worker_id"] = worker_id

            self.redis.set(job_key, json.dumps(job_data), ex=86400)

    def get_state(self, job_id: str) -> Optional[JobState]:
        """Get current job state."""
        state_key = f"{self.STATE_PREFIX}{job_id}"
        data = self.redis.get(state_key)
        if data:
            state_data = json.loads(data)
            return JobState(state_data["state"])
        return None

    def save_checkpoint(self, job_id: str, checkpoint_data: dict):
        """Save job progress checkpoint for resume."""
        key = f"{self.CHECKPOINT_PREFIX}{job_id}"
        self.redis.setex(
            key,
            self.checkpoint_ttl,
            json.dumps(checkpoint_data)
        )

    def load_checkpoint(self, job_id: str) -> Optional[dict]:
        """Load job checkpoint for resume."""
        key = f"{self.CHECKPOINT_PREFIX}{job_id}"
        data = self.redis.get(key)
        if data:
            return json.loads(data)
        return None

    def publish_progress(self, job_id: str, progress: float, message: str = ""):
        """Publish job progress to Redis pub/sub."""
        channel = "translation:progress"
        data = json.dumps({
            "job_id": job_id,
            "progress": progress,
            "message": message,
            "timestamp": datetime.utcnow().isoformat()
        })
        self.redis.publish(channel, data)
```

**Step 4: Run tests to verify they pass**

```bash
python -m pytest tests/test_queue/test_manager.py -v
# Expected: PASS
```

**Step 5: Commit job queue implementation**

```bash
git add cmd/translation-worker/queue/ tests/test_queue/
git commit -m "feat(queue): implement Redis job queue for horizontal scaling

- Add JobState enum with all job states
- Add Job dataclass for job structure
- Add JobManager with enqueue/dequeue methods
- Add priority queue support (urgent, normal, bulk)
- Add state management with set_state/get_state
- Add checkpoint/save and load for fault tolerance
- Add progress updates via Redis pub/sub
- Add unit tests for queue operations
"
```

---

## Phase 8: Bilingual Review Workflow (Multi-Model + Judge + Web UI)

> **Design Document:** See `docs/plans/2025-01-12-bilingual-review-design.md` for full architecture

### Overview

Automated multi-model translation produces a final output that a human approves via web UI. The bilingual CSV serves as an audit trail and fine-tuning data source, not as the primary review interface.

**Key Changes from Original Plan:**
- **Web UI is the review interface**, not CSV
- **Judge model** resolves most disagreements automatically
- **Human reviews final output**, not individual segments
- **CSV** is for audit trail + fine-tuning data only

### Task 13: Implement multi-model translation with judge

**Files:**
- Create: `cmd/translation-worker/judge/model.py`
- Create: `cmd/translation-worker/judge/evaluator.py`
- Create: `cmd/translation-worker/output/bilingual_csv.py` (audit trail only)
- Create: `tests/test_judge/test_evaluator.py`
- Create: `tests/test_output/test_bilingual_csv.py`

**Step 1: Write test for judge evaluator**

```python
# tests/test_judge/test_evaluator.py
import pytest
from judge.evaluator import JudgeEvaluator, JudgeResult, TranslationCandidate
from datetime import datetime

def test_judge_resolves_disagreement():
    """Judge should select better translation when models disagree."""
    evaluator = JudgeEvaluator()

    candidates = [
        TranslationCandidate(
            model_name="claude-4.5-sonnet",
            text="Customer support",
            confidence=0.85,
            glossary_matches=["support"],
            latency_ms=1200
        ),
        TranslationCandidate(
            model_name="gpt-4o",
            text="Client assistance",
            confidence=0.78,
            glossary_matches=[],
            latency_ms=800
        )
    ]

    result = evaluator.evaluate(
        source="顧客対応",
        candidates=candidates,
        context={"segment_id": "s1", "slide": 1}
    )

    assert isinstance(result, JudgeResult)
    assert result.winner in ["model_a", "model_b"]
    assert 0 <= result.confidence <= 1
    assert len(result.reasoning) > 0

def test_judge_handles_timeout():
    """Judge should fallback gracefully on timeout."""
    evaluator = JudgeEvaluator(timeout_ms=100)
    evaluator._judge_call = lambda *args, **kwargs: None  # Simulate timeout

    candidates = [
        TranslationCandidate("model_a", "Translation A", 0.9, [], 500)
    ]

    result = evaluator.evaluate("source", candidates, {})

    # Should return fallback result
    assert result.confidence < 1.0  # Lower confidence due to fallback
```

**Step 2: Implement judge evaluator (stub with full interface)**

```python
# cmd/translation-worker/judge/model.py
from dataclasses import dataclass, field
from typing import List, Dict, Optional, Literal
from enum import Enum

class JudgeWinner(Enum):
    MODEL_A = "model_a"
    MODEL_B = "model_b"
    EDITED = "edited"
    TIE = "tie"

@dataclass
class TranslationCandidate:
    """A translation output from a model."""
    model_name: str
    text: str
    confidence: float
    glossary_matches: List[str] = field(default_factory=list)
    latency_ms: int = 0
    metadata: Dict = field(default_factory=dict)

@dataclass
class JudgeResult:
    """Result of judge evaluation."""
    segment_id: str
    winner: JudgeWinner
    selected_text: str
    confidence: float  # 0-1, how sure judge is
    reasoning: str
    concerns: List[str] = field(default_factory=list)
    suggested_edits: Optional[str] = None
    fallback_used: bool = False
    evaluated_at: datetime = field(default_factory=datetime.utcnow)
```

```python
# cmd/translation-worker/judge/evaluator.py
import logging
import time
from typing import List, Dict, Optional
from .model import TranslationCandidate, JudgeResult, JudgeWinner

logger = logging.getLogger(__name__)

class JudgeEvaluator:
    """Evaluates competing translations and selects the best option.

    In MVP: Returns random selection with placeholder reasoning.
    Full implementation: Uses LLM judge to evaluate quality.
    """

    def __init__(
        self,
        judge_model: str = "claude-4.5-sonnet",
        timeout_ms: int = 30000,
        fallback_on_timeout: bool = True
    ):
        self.judge_model = judge_model
        self.timeout_ms = timeout_ms
        self.fallback_on_timeout = fallback_on_timeout

    def evaluate(
        self,
        source: str,
        candidates: List[TranslationCandidate],
        context: Dict
    ) -> JudgeResult:
        """Evaluate candidates and select winner.

        Args:
            source: Original source text
            candidates: Translation outputs to compare
            context: Segment context (slide, position, etc.)

        Returns:
            JudgeResult with winner and reasoning
        """
        if len(candidates) < 2:
            # Only one candidate, return it
            c = candidates[0]
            return JudgeResult(
                segment_id=context.get("segment_id", "unknown"),
                winner=JudgeWinner.MODEL_A,
                selected_text=c.text,
                confidence=c.confidence,
                reasoning="Only candidate available",
                fallback_used=False
            )

        # MVP: Simple stub - pick by confidence
        # In full implementation, this calls the LLM judge
        sorted_candidates = sorted(
            candidates,
            key=lambda c: (c.confidence, -len(c.glossary_matches)),
            reverse=True
        )

        winner = sorted_candidates[0]
        winner_idx = candidates.index(winner)

        return JudgeResult(
            segment_id=context.get("segment_id", "unknown"),
            winner=JudgeWinner.MODEL_A if winner_idx == 0 else JudgeWinner.MODEL_B,
            selected_text=winner.text,
            confidence=winner.confidence,
            reasoning=f"MVP stub: selected based on confidence ({winner.confidence:.2f})",
            fallback_used=False
        )

    def evaluate_batch(
        self,
        segments: List[Dict],
        candidates_by_segment: Dict[str, List[TranslationCandidate]]
    ) -> List[JudgeResult]:
        """Evaluate multiple segments.

        Args:
            segments: List of segment dicts with source and context
            candidates_by_segment: Map of segment_id to candidates

        Returns:
            List of JudgeResult, one per segment
        """
        results = []

        for segment in segments:
            segment_id = segment.get("id", "")
            candidates = candidates_by_segment.get(segment_id, [])

            result = self.evaluate(
                source=segment.get("source", ""),
                candidates=candidates,
                context=segment.get("context", {})
            )
            results.append(result)

        return results
```

**Step 3: Implement bilingual CSV for audit trail**

```python
# cmd/translation-worker/output/bilingual_csv.py
import csv
from pathlib import Path
from typing import List, Dict, Optional
from dataclasses import dataclass
from datetime import datetime
import logging

logger = logging.getLogger(__name__)

@dataclass
class BilingualCSVSegment:
    """A segment for CSV audit output."""
    segment_id: str
    source: str
    target: str
    judge_winner: str
    judge_confidence: float
    judge_reasoning: str
    is_flagged: bool
    flag_reason: Optional[str]
    model_a_output: Optional[str]
    model_b_output: Optional[str]
    glossary_terms: List[str]
    context: Dict

class BilingualCSV:
    """Generates bilingual CSV as audit trail and fine-tuning data.

    This is NOT for human review - use the web UI for that.
    The CSV records what happened for analysis and learning.
    """

    AUDIT_COLUMNS = [
        "segment_id",
        "source",
        "target",
        "judge_winner",
        "judge_confidence",
        "judge_reasoning",
        "is_flagged",
        "flag_reason",
        "model_a",
        "model_b",
        "glossary_terms",
        "context_json"
    ]

    def __init__(
        self,
        encoding: str = "utf-8-sig",
        delimiter: str = ",",
        include_alternatives: bool = True
    ):
        self.encoding = encoding
        self.delimiter = delimiter
        self.include_alternatives = include_alternatives

    def generate(
        self,
        segments: List[BilingualCSVSegment],
        output_path: str
    ) -> None:
        """Generate bilingual CSV audit trail.

        Args:
            segments: Segments with judge decisions
            output_path: Where to write CSV
        """
        Path(output_path).parent.mkdir(parents=True, exist_ok=True)

        with open(output_path, 'w', newline='', encoding=self.encoding) as f:
            writer = csv.DictWriter(
                f,
                fieldnames=self.AUDIT_COLUMNS,
                delimiter=self.delimiter
            )
            writer.writeheader()

            for seg in segments:
                import json
                row = {
                    "segment_id": seg.segment_id,
                    "source": seg.source,
                    "target": seg.target,
                    "judge_winner": seg.judge_winner,
                    "judge_confidence": seg.judge_confidence,
                    "judge_reasoning": seg.judge_reasoning,
                    "is_flagged": seg.is_flagged,
                    "flag_reason": seg.flag_reason or "",
                    "model_a": seg.model_a_output if self.include_alternatives else "",
                    "model_b": seg.model_b_output if self.include_alternatives else "",
                    "glossary_terms": ",".join(seg.glossary_terms),
                    "context_json": json.dumps(seg.context, ensure_ascii=False)
                }
                writer.writerow(row)

        logger.info(f"[BILINGUAL_CSV] Generated audit trail: {output_path}")

    def import_edited(self, csv_path: str) -> Dict[str, Dict]:
        """Import edited CSV with human corrections.

        This is for loading fine-tuning data or post-hoc corrections.

        Args:
            csv_path: Path to edited CSV

        Returns:
            Dict mapping segment_id to edited data
        """
        import json

        segments_by_id = {}

        with open(csv_path, 'r', encoding=self.encoding) as f:
            reader = csv.DictReader(f, delimiter=self.delimiter)

            for row in reader:
                segment_id = row.get("segment_id")
                if not segment_id:
                    continue

                segments_by_id[segment_id] = {
                    "source": row.get("source", ""),
                    "target": row.get("target", ""),  # Human-edited
                    "original_target": row.get("original_target", row.get("target", "")),
                    "flag_reason": row.get("flag_reason", "")
                }

        logger.info(f"[BILINGUAL_CSV] Imported {len(segments_by_id)} edited segments")
        return segments_by_id
```

**Step 4: Run tests**

```bash
python -m pytest tests/test_judge/test_evaluator.py -v
python -m pytest tests/test_output/test_bilingual_csv.py -v
```

**Step 5: Commit judge and CSV implementation**

```bash
git add cmd/translation-worker/judge/ cmd/translation-worker/output/
git add tests/test_judge/ tests/test_output/
git commit -m "feat(review): implement multi-model judge with audit CSV

- Add JudgeEvaluator for resolving model disagreements
- Add JudgeResult and TranslationCandidate data models
- Add BilingualCSV for audit trail (not review interface)
- MVP stub: judge uses confidence-based selection
- Full interface ready for LLM judge implementation
- Add test coverage for judge and CSV export
"
```

---

## Phase 9: Web UI Integration (Backend API)

### Task 14: Implement approval workflow API

**Files:**
- Create: `backend/internal/handlers/translation.go`
- Update: `backend/internal/models/translation_job.go`
- Create: `backend/tests/translation_test.go`

**Step 1: Add translation job models**

```go
// backend/internal/models/translation_job.go
package models

import (
	"time"
	"github.com/google/uuid"
)

// TranslationJobStatus represents the status of a translation job
type TranslationJobStatus string

const (
	TranslationStatusProcessing   TranslationJobStatus = "processing"
	TranslationStatusPendingApproval TranslationJobStatus = "pending_approval"
	TranslationStatusApproved      TranslationJobStatus = "approved"
	TranslationStatusRejected      TranslationJobStatus = "rejected"
	TranslationStatusExported      TranslationJobStatus = "exported"
)

// TranslationJob represents a translation job in the system
type TranslationJob struct {
	Base

	// File information
	SourceFile string    `gorm:"type:varchar(500)"`
	TargetFile string    `gorm:"type:varchar(500)"`
	UserID     uuid.UUID `gorm:"type:uuid;index"`

	// Status
	Status TranslationJobStatus `gorm:"type:varchar(50);index"`

	// Quality metrics
	OverallScore    float64 `gorm:"type:decimal(5,2)"`
	SegmentCount    int
	FlaggedCount    int
	JudgeResolutions int

	// Timestamps
	CompletedAt   *time.Time
	ApprovedAt    *time.Time
	ApprovedBy    *uuid.UUID

	// Relations
	Segments []TranslationSegment `gorm:"foreignKey:JobID"`
}

// TranslationSegment represents a single translated segment
type TranslationSegment struct {
	Base

	JobID uuid.UUID `gorm:"type:uuid;index"`

	// Text
	Source string `gorm:"type:text"`
	Target string `gorm:"type:text"`
	Context string `gorm:"type:jsonb"` // JSON: slide, position, etc.

	// Judge decision
	JudgeWinner string  `gorm:"type:varchar(20)"` // "model_a", "model_b", "edited", "tie"
	JudgeConfidence float64 `gorm:"type:decimal(5,2)"`
	JudgeReasoning string `gorm:"type:text"`

	// Flags
	IsFlagged  bool   `gorm:"index"`
	FlagReason  string `gorm:"type:varchar(500)"`

	// Alternatives (for review UI)
	ModelAOutput *string `gorm:"type:text"`
	ModelBOutput *string `gorm:"type:text"`
	GlossaryTerms string `gorm:"type:jsonb"` // JSON array
}

// TableName specifies the table name for TranslationJob
func (TranslationJob) TableName() string {
	return "translation_jobs"
}

// TableName specifies the table name for TranslationSegment
func (TranslationSegment) TableName() string {
	return "translation_segments"
}
```

**Step 2: Add translation handlers**

```go
// backend/internal/handlers/translation.go
package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/database"
)

type TranslationHandler struct {
	db *database.DB
}

func NewTranslationHandler(db *database.DB) *TranslationHandler {
	return &TranslationHandler{db: db}
}

type JobListResponse struct {
	Jobs []JobSummary `json:"jobs"`
}

type JobSummary struct {
	ID             string              `json:"id"`
	SourceFile     string              `json:"source_file"`
	Status         string              `json:"status"`
	OverallScore   float64             `json:"overall_score"`
	SegmentCount   int                 `json:"segment_count"`
	FlaggedCount   int                 `json:"flagged_count"`
	CreatedAt      string              `json:"created_at"`
	CompletedAt    *string             `json:"completed_at"`
}

type JobDetailResponse struct {
	JobSummary
	Segments []SegmentDetail `json:"segments"`
}

type SegmentDetail struct {
	ID             string  `json:"id"`
	Source         string  `json:"source"`
	Target         string  `json:"target"`
	Context        string  `json:"context"`
	JudgeWinner    string  `json:"judge_winner"`
	JudgeConfidence float64 `json:"judge_confidence"`
	IsFlagged      bool    `json:"is_flagged"`
	FlagReason     *string `json:"flag_reason"`
	ModelA         *string `json:"model_a,omitempty"`
	ModelB         *string `json:"model_b,omitempty"`
}

// ListJobs returns all translation jobs for the current user
func (h *TranslationHandler) ListJobs(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Not authenticated"})
	}

	var jobs []models.TranslationJob
	err := h.db.DB().Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&jobs)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch jobs"})
	}

	response := make([]JobSummary, len(jobs))
	for i, job := range jobs {
		response[i] = JobSummary{
			ID:           job.ID.String(),
			SourceFile:   job.SourceFile,
			Status:       string(job.Status),
			OverallScore: job.OverallScore,
			SegmentCount: job.SegmentCount,
			FlaggedCount: job.FlaggedCount,
			CreatedAt:    job.CreatedAt.Format("2006-01-02T15:04:05Z"),
			CompletedAt:  formatTimePtr(job.CompletedAt),
		}
	}

	return c.JSON(response)
}

// GetJob returns details for a specific job
func (h *TranslationHandler) GetJob(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	jobID := c.Params("id")

	var job models.TranslationJob
	err := h.db.DB().Where("id = ? AND user_id = ?", jobID, userID).
		Preload("Segments").
		First(&job).Error
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	segments := make([]SegmentDetail, len(job.Segments))
	for i, seg := range job.Segments {
		segments[i] = SegmentDetail{
			ID:             seg.ID.String(),
			Source:         seg.Source,
			Target:         seg.Target,
			Context:        seg.Context,
			JudgeWinner:    seg.JudgeWinner,
			JudgeConfidence: float64(seg.JudgeConfidence),
			IsFlagged:      seg.IsFlagged,
			FlagReason:     stringPtr(seg.FlagReason),
			ModelA:         seg.ModelAOutput,
			ModelB:         seg.ModelBOutput,
		}
	}

	return c.JSON(JobDetailResponse{
		JobSummary: JobSummary{
			ID:           job.ID.String(),
			SourceFile:   job.SourceFile,
			Status:       string(job.Status),
			OverallScore: job.OverallScore,
			SegmentCount: job.SegmentCount,
			FlaggedCount: job.FlaggedCount,
			CreatedAt:    job.CreatedAt.Format("2006-01-02T15:04:05Z"),
			CompletedAt:  formatTimePtr(job.CompletedAt),
		},
		Segments: segments,
	})
}

// ApproveJob approves a translation job
func (h *TranslationHandler) ApproveJob(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	jobID := c.Params("id")

	var job models.TranslationJob
	err := h.db.DB().Where("id = ? AND user_id = ?", jobID, userID).
		First(&job).Error
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	now := time.Now()
	job.Status = models.TranslationStatusApproved
	job.ApprovedAt = &now
	job.ApprovedBy = uuid.MustParse(userID)

	h.db.DB().Save(&job)

	return c.JSON(fiber.Map{"success": true, "status": "approved"})
}

// RejectJob rejects a translation job
func (h *TranslationHandler) RejectJob(c *fiber.Ctx) error {
	// TODO: Implement rejection with reason capture
	return c.JSON(fiber.Map{"success": true, "status": "rejected"})
}

// UpdateSegment edits a specific segment
func (h *TranslationHandler) UpdateSegment(c *fiber.Ctx) error {
	type UpdateRequest struct {
		Target string `json:"target"`
	}

	userID := c.Locals("user_id").(string)
	jobID := c.Params("id")
	segmentID := c.Params("segment_id")

	var body UpdateRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Verify job belongs to user
	var job models.TranslationJob
	err := h.db.DB().Where("id = ? AND user_id = ?", jobID, userID).
		First(&job).Error
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	// Update segment
	var segment models.TranslationSegment
	err = h.db.DB().Where("id = ? AND job_id = ?", segmentID, jobID).
		First(&segment).Error
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Segment not found"})
	}

	segment.Target = body.Target
	segment.JudgeWinner = "edited"  # Human-edited overrides judge
	h.db.DB().Save(&segment)

	return c.JSON(fiber.Map{"success": true})
}

// Helper functions
func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format("2006-01-02T15:04:05Z")
	return &formatted
}

func stringPtr(s string) *string {
	return &s
}
```

**Step 3: Register routes**

Update `backend/cmd/server/main.go`:

```go
// Register translation routes
translationHandler := handlers.NewTranslationHandler(db)

api.Get("/translation/jobs", translationHandler.ListJobs)
api.Get("/translation/jobs/:id", translationHandler.GetJob)
api.Post("/translation/jobs/:id/approve", translationHandler.ApproveJob)
api.Post("/translation/jobs/:id/reject", translationHandler.RejectJob)
api.Put("/translation/jobs/:id/segments/:segment_id", translationHandler.UpdateSegment)
```

---

## Phase 10: Audit Tools
    ):
        self.encoding = encoding
        self.delimiter = delimiter
        self.columns = columns or self.DEFAULT_COLUMNS

    def generate(self, segments: List[Dict], output_path: str):
        """Generate bilingual CSV from translation segments."""
        with open(output_path, 'w', newline='', encoding=self.encoding) as f:
            writer = csv.DictWriter(f, fieldnames=self.columns, delimiter=self.delimiter)
            writer.writeheader()

            for seg in segments:
                row = {
                    "segment_id": seg.get("id", ""),
                    "source": seg.get("source", ""),
                    "target": seg.get("target", ""),
                    "context": seg.get("context", ""),
                    "confidence": seg.get("confidence", 0.0),
                    "glossary_used": seg.get("glossary_used", False),
                    "notes": seg.get("notes", "")
                }
                writer.writerow(row)

    def import_csv(self, csv_path: str) -> Dict[str, Dict]:
        """Import edited CSV and return updated segments."""
        segments_by_id = {}

        with open(csv_path, 'r', encoding=self.encoding) as f:
            reader = csv.DictReader(f, delimiter=self.delimiter)

            for row in reader:
                segment_id = row.get("segment_id")
                if segment_id:
                    segments_by_id[segment_id] = {
                        "source": row["source"],
                        "target": row["target"],
                        "context": row.get("context", ""),
                        "notes": row.get("notes", "")
                    }

        return segments_by_id
```

**Step 4: Run tests to verify they pass**

```bash
python -m pytest tests/test_output/test_bilingual_csv.py -v
# Expected: PASS
```

**Step 5: Commit bilingual CSV implementation**

```bash
git add cmd/translation-worker/output/ tests/test_output/
git commit -m "feat(output): implement bilingual CSV generation and import

- Add BilingualCSV class for review workflow
- Add generate() method to create side-by-side JA/EN CSV
- Add import_csv() method to load edited translations
- Support utf-8-sig encoding for Excel compatibility
- Add configurable column selection
- Add unit tests for generate/import cycle
"
```

---

## Phase 9: Audit Tools

### Task 14: Implement Japanese character count and style checking

**Files:**
- Create: `cmd/translation-worker/audit/counter.py`
- Create: `cmd/translation-worker/audit/style_checker.py`
- Create: `tests/test_audit/test_counter.py`

**Step 1: Write test for audit tools**

```python
# tests/test_audit/test_counter.py
import pytest
from audit.counter import JapaneseCharacterCounter
from audit.style_checker import StyleChecker

def test_character_count():
    """Should count Japanese characters by type."""
    counter = JapaneseCharacterCounter()

    result = counter.count("顧客満足度は高いです")

    assert result["kanji"] == 4  # 顧客満足度
    assert result["hiragana"] > 0  # はです
    assert result["total"] == 11

def test_style_checker():
    """Should detect style issues."""
    checker = StyleChecker()

    issues = checker.check("We are honored to meet you.", source="会いできて光栄です")

    # Should detect potential style issues
    assert isinstance(issues, list)
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_audit/test_counter.py -v
# Expected: ImportError
```

**Step 3: Implement audit tools**

```python
# cmd/translation-worker/audit/counter.py
from dataclasses import dataclass
from typing import Dict

@dataclass
class CharacterCount:
    """Japanese character count breakdown."""
    total: int
    kanji: int
    hiragana: int
    katakana: int
    punctuation: int
    whitespace: int
    latin: int
    estimated_english: int

class JapaneseCharacterCounter:
    """Counts Japanese text for billing/estimation."""

    def count(self, text: str) -> Dict:
        """Count Japanese text characters by type."""
        counts = {
            "total": len(text),
            "kanji": 0,
            "hiragana": 0,
            "katakana": 0,
            "punctuation": 0,
            "whitespace": 0,
            "latin": 0
        }

        for char in text:
            code = ord(char)

            if char.isspace():
                counts["whitespace"] += 1
            elif 0x4E00 <= code <= 0x9FFF or 0x3400 <= code <= 0x4DBF:
                counts["kanji"] += 1
            elif 0x3040 <= code <= 0x309F:
                counts["hiragana"] += 1
            elif 0x30A0 <= code <= 0x30FF or 0x31F0 <= code <= 0x31FF:
                counts["katakana"] += 1
            elif char in '、。！？「」『』（）':
                counts["punctuation"] += 1
            elif char.isalpha():
                counts["latin"] += 1

        # Estimate English length
        counts["estimated_english"] = self._estimate_english(counts)

        return counts

    def _estimate_english(self, counts: Dict) -> int:
        """Estimate English character count."""
        # Typical JA→EN expansion ratios
        return int(
            counts["kanji"] * 2.0 +
            counts["hiragana"] * 1.5 +
            counts["katakana"] * 1.5 +
            counts["punctuation"] +
            counts["latin"]
        )
```

```python
# cmd/translation-worker/audit/style_checker.py
from dataclasses import dataclass
from typing import List

@dataclass
class StyleIssue:
    """A style compliance issue."""
    severity: str  # "warning", "error", "info"
    message: str
    location: str = ""

class StyleChecker:
    """Checks translation against style guide."""

    # Common JA→EN style issues
    HONORIFICS_PATTERNS = [
        (r"\b-san\b", "Avoid using '-san' suffix excessively"),
        (r"\b-sama\b", "Avoid using '-sama' suffix excessively"),
        (r"\b-kun\b", "Avoid using '-kun' suffix excessively"),
    ]

    def __init__(self, style_guide_path: str = None):
        self.style_guide_path = style_guide_path
        # TODO: Load style guide if provided

    def check(self, translation: str, source: str = None) -> List[StyleIssue]:
        """Check translation against style guide."""
        issues = []

        # Check honorifics
        issues.extend(self._check_honorifics(translation))

        # Check consistency
        if source:
            issues.extend(self._check_consistency(translation, source))

        # Check sentence endings
        issues.extend(self._check_sentence_endings(translation))

        return issues

    def _check_honorifics(self, text: str) -> List[StyleIssue]:
        """Check for excessive honorific usage."""
        issues = []
        import re

        # Check for patterns
        for pattern, message in self.HONORIFICS_PATTERNS:
            if re.search(pattern, text, re.IGNORECASE):
                issues.append(StyleIssue(
                    severity="warning",
                    message=message,
                    location="honorifics"
                ))

        return issues

    def _check_consistency(self, translation: str, source: str) -> List[StyleIssue]:
        """Check for term consistency."""
        issues = []
        # TODO: Implement term consistency checking
        return issues

    def _check_sentence_endings(self, text: str) -> List[StyleIssue]:
        """Check for proper sentence endings."""
        issues = []

        # Check for run-on sentences
        sentences = text.split('.')
        for sentence in sentences:
            if len(sentence.strip()) > 200:
                issues.append(StyleIssue(
                    severity="warning",
                    message="Very long sentence detected (>200 chars)",
                    location="sentence_length"
                ))

        return issues
```

**Step 4: Run tests to verify they pass**

```bash
python -m pytest tests/test_audit/test_counter.py -v
# Expected: PASS
```

**Step 5: Commit audit tools**

```bash
git add cmd/translation-worker/audit/ tests/test_audit/
git commit -m "feat(audit): implement Japanese character count and style checking

- Add JapaneseCharacterCounter for detailed breakdown
- Add StyleChecker for JA→EN style compliance
- Add counts for kanji, hiragana, katakana, punctuation
- Add estimate_english() for billing estimation
- Add honorific checking for translation quality
- Add sentence length checking
- Add unit tests
"
```

---

## Phase 10: Quality Scoring with COMET

### Task 15: Implement COMET quality scoring

**Files:**
- Create: `cmd/translation-worker/quality/comet_scorer.py`
- Create: `tests/test_quality/test_comet_scorer.py`

**Step 1: Write test for COMET scorer**

```python
# tests/test_quality/test_comet_scorer.py
import pytest
from quality.comet_scorer import COMETScorer

def test_comet_scoring():
    """Should score translation quality using COMET."""
    scorer = COMETScorer()

    # Test that COMET scoring works
    score = scorer.score(
        source="こんにちは世界",
        target="Hello World",
        reference="Hello World"  # Optional reference
    )

    assert 0.0 <= score <= 1.0
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_quality/test_comet_scorer.py -v
# Expected: ImportError
```

**Step 3: Implement COMET scorer**

```python
# cmd/translation-worker/quality/comet_scorer.py
from typing import Optional

class COMETScorer:
    """Quality scoring using COMET (reference-free)."""

    def __init__(self, model_name: str = "Unbabel/wmt22-comet-da"):
        self.model_name = model_name
        self._model = None

    def _load_model(self):
        """Lazy load COMET model."""
        try:
            from comet import download_model, load_from_checkpoint
            model_path = download_model(self.model_name)
            self._model = load_from_checkpoint(model_path)
        except ImportError:
            raise RuntimeError("COMET package not installed")

    def score(
        self,
        source: str,
        target: str,
        reference: Optional[str] = None
    ) -> float:
        """Score translation quality using COMET.

        Args:
            source: Source text
            target: Translated text
            reference: Optional reference translation

        Returns:
            Quality score between 0.0 and 1.0
        """
        if self._model is None:
            self._load_model()

        # COMET expects source and target (and optionally reference)
        data = [{"src": source, "mt": target}]
        if reference:
            data[0]["ref"] = reference

        # Score and return mean score
        model_output = self._model.predict(data, batch_size=1)
        scores = model_output.scores

        # Return mean score or 0 if error
        return float(scores.mean()) if hasattr(scores, 'mean') else 0.0

    def is_available(self) -> bool:
        """Check if COMET scorer is available."""
        try:
            from comet import download_model
            return True
        except ImportError:
            return False
```

**Step 4: Run tests to verify they pass**

```bash
python -m pytest tests/test_quality/test_comet_scorer.py -v
# Expected: PASS (if COMET installed, skip otherwise)
```

**Step 5: Commit COMET scorer**

```bash
git add cmd/translation-worker/quality/ tests/test_quality/
git commit -m "feat(quality): implement COMET quality scoring

- Add COMETScorer class for reference-free quality assessment
- Add lazy loading of COMET model
- Add score() method returning 0.0-1.0 quality score
- Add is_available() check for COMET installation
- Support optional reference translation for enhanced scoring
- Add unit tests (skip if COMET not installed)
"
```

---

## Phase 11: Upload Destinations

### Task 16: Implement upload abstraction (unchanged from original)

*(Implementation remains same as original plan)*

---

### Task 12.5: Implement pipeline modes (manual/semi_auto/auto)

**Files:**
- Create: `cmd/translation-worker/pipeline/modes.py`
- Create: `cmd/translation-worker/pipeline/executor.py`
- Create: `tests/test_pipeline/test_modes.py`

**Step 1: Write failing test for pipeline modes**

```python
# tests/test_pipeline/test_modes.py
import pytest
from cmd.translation_worker.pipeline.modes import PipelineMode, PipelineExecutor
from cmd.translation_worker.pipeline.executor import PipelineResult

@pytest.fixture
def mock_orchestrator():
    """Create mock orchestrator for testing."""
    from unittest.mock import Mock
    orchestrator = Mock()
    orchestrator.translate.return_value = "Translated text"
    orchestrator.score_quality.return_value = 0.85
    return orchestrator

def test_manual_mode_requires_approval(mock_orchestrator):
    """Manual mode should require human approval regardless of score."""
    executor = PipelineExecutor(mode=PipelineMode.MANUAL)
    result = executor.execute(
        text="Test text",
        source_lang="ja",
        target_lang="en",
        orchestrator=mock_orchestrator
    )

    assert result.approved is False
    assert result.requires_review is True
    assert "manual" in result.metadata["mode"]

def test_semi_auto_approves_on_good_score(mock_orchestrator):
    """Semi-auto mode should auto-approve when quality score >= threshold."""
    executor = PipelineExecutor(
        mode=PipelineMode.SEMI_AUTO,
        approval_threshold=0.80
    )

    # Score above threshold
    mock_orchestrator.score_quality.return_value = 0.85
    result = executor.execute(
        text="Good translation",
        source_lang="ja",
        target_lang="en",
        orchestrator=mock_orchestrator
    )

    assert result.approved is True
    assert result.requires_review is False

def test_semi_auto_requires_review_on_low_score(mock_orchestrator):
    """Semi-auto mode should require review when quality score < threshold."""
    executor = PipelineExecutor(
        mode=PipelineMode.SEMI_AUTO,
        approval_threshold=0.80
    )

    # Score below threshold
    mock_orchestrator.score_quality.return_value = 0.65
    result = executor.execute(
        text="Poor translation",
        source_lang="ja",
        target_lang="en",
        orchestrator=mock_orchestrator
    )

    assert result.approved is False
    assert result.requires_review is True
    assert 0.65 == result.metadata["quality_score"]

def test_auto_mode_always_approves(mock_orchestrator):
    """Auto mode should always approve without quality gating."""
    executor = PipelineExecutor(mode=PipelineMode.AUTO)

    # Even low score
    mock_orchestrator.score_quality.return_value = 0.50
    result = executor.execute(
        text="Any translation",
        source_lang="ja",
        target_lang="en",
        orchestrator=mock_orchestrator
    )

    assert result.approved is True
    assert result.requires_review is False

def test_pipeline_mode_from_string():
    """Should create PipelineMode from configuration string."""
    assert PipelineMode.from_string("manual") == PipelineMode.MANUAL
    assert PipelineMode.from_string("semi_auto") == PipelineMode.SEMI_AUTO
    assert PipelineMode.from_string("auto") == PipelineMode.AUTO

    with pytest.raises(ValueError):
        PipelineMode.from_string("invalid")
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_pipeline/test_modes.py -v
# Expected: FAIL with "PipelineMode not defined"
```

**Step 3: Write pipeline modes implementation**

```python
# cmd/translation-worker/pipeline/modes.py
from enum import Enum
from typing import Optional
from dataclasses import dataclass

class PipelineMode(Enum):
    """Pipeline execution modes."""
    MANUAL = "manual"         # Requires human review always
    SEMI_AUTO = "semi_auto"  # Auto-approve if quality score good
    AUTO = "auto"            # Never requires review

    @classmethod
    def from_string(cls, value: str) -> "PipelineMode":
        """Create PipelineMode from string value."""
        try:
            return cls[value.upper()]
        except KeyError:
            raise ValueError(
                f"Invalid pipeline mode: {value}. "
                f"Must be one of: {', '.join(m.value for m in cls)}"
            )

@dataclass
class PipelineResult:
    """Result of pipeline execution."""
    success: bool
    translated_text: str
    approved: bool
    requires_review: bool
    metadata: dict
    quality_score: Optional[float] = None
    error: Optional[str] = None

# cmd/translation-worker/pipeline/executor.py
from typing import Optional
from cmd.translation_worker.pipeline.modes import PipelineMode, PipelineResult

class PipelineExecutor:
    """Execute translation pipeline with configurable modes."""

    def __init__(
        self,
        mode: PipelineMode = PipelineMode.SEMI_AUTO,
        approval_threshold: float = 0.80,
        enable_discussion: bool = False
    ):
        """
        Initialize pipeline executor.

        Args:
            mode: Pipeline execution mode
            approval_threshold: Quality score threshold for auto-approval
            enable_discussion: Enable multi-model discussion stage
        """
        self.mode = mode
        self.approval_threshold = approval_threshold
        self.enable_discussion = enable_discussion

    def execute(
        self,
        text: str,
        source_lang: str,
        target_lang: str,
        orchestrator
    ) -> PipelineResult:
        """
        Execute pipeline with configured mode.

        Args:
            text: Source text to translate
            source_lang: Source language code
            target_lang: Target language code
            orchestrator: TranslationOrchestrator instance

        Returns:
            PipelineResult with translation and approval status
        """
        try:
            # Step 1: Translate
            translated = orchestrator.translate(
                text=text,
                source_lang=source_lang,
                target_lang=target_lang
            )

            # Step 2: Quality scoring (if available)
            quality_score = None
            if hasattr(orchestrator, 'score_quality'):
                quality_score = orchestrator.score_quality(
                    source_text=text,
                    translated_text=translated
                )

            # Step 3: Determine approval based on mode
            approved, requires_review = self._determine_approval(
                quality_score=quality_score
            )

            return PipelineResult(
                success=True,
                translated_text=translated,
                approved=approved,
                requires_review=requires_review,
                metadata={
                    "mode": self.mode.value,
                    "quality_score": quality_score,
                    "threshold": self.approval_threshold,
                    "discussion_enabled": self.enable_discussion,
                },
                quality_score=quality_score
            )

        except Exception as e:
            return PipelineResult(
                success=False,
                translated_text="",
                approved=False,
                requires_review=True,
                metadata={"mode": self.mode.value},
                error=str(e)
            )

    def _determine_approval(
        self,
        quality_score: Optional[float]
    ) -> tuple[bool, bool]:
        """
        Determine approval based on mode and quality score.

        Returns:
            Tuple of (approved, requires_review)
        """
        if self.mode == PipelineMode.MANUAL:
            # Manual mode: always require review
            return False, True

        elif self.mode == PipelineMode.SEMI_AUTO:
            # Semi-auto: approve if score meets threshold
            if quality_score is None:
                # No quality score available, require review
                return False, True

            approved = quality_score >= self.approval_threshold
            return approved, not approved

        elif self.mode == PipelineMode.AUTO:
            # Auto mode: always approve
            return True, False

        # Fallback
        return False, True
```

**Step 4: Run test to verify it passes**

```bash
python -m pytest tests/test_pipeline/test_modes.py -v
# Expected: PASS
```

**Step 5: Add configuration integration**

```python
# cmd/translation-worker/config.py (add to existing config)

from dataclasses import dataclass
from cmd.translation_worker.pipeline.modes import PipelineMode

@dataclass
class PipelineConfig:
    """Pipeline execution configuration."""
    mode: PipelineMode = PipelineMode.SEMI_AUTO
    approval_threshold: float = 0.80
    enable_discussion: bool = False
    min_confidence: float = 0.70

    @classmethod
    def from_toml(cls, data: dict) -> "PipelineConfig":
        """Create config from TOML data."""
        mode_str = data.get("mode", "semi_auto")
        return cls(
            mode=PipelineMode.from_string(mode_str),
            approval_threshold=data.get("approval_threshold", 0.80),
            enable_discussion=data.get("enable_discussion", False),
            min_confidence=data.get("min_confidence", 0.70)
        )
```

**Step 6: Commit pipeline modes**

```bash
git add cmd/translation-worker/pipeline/ tests/test_pipeline/
git commit -m "feat(pipeline): add pipeline modes (manual/semi_auto/auto)

- Add PipelineMode enum with MANUAL, SEMI_AUTO, AUTO options
- Implement PipelineExecutor for mode-based execution
- Add quality-based auto-approval for semi_auto mode
- MANUAL: always requires human review
- SEMI_AUTO: auto-approve if quality score >= threshold
- AUTO: never requires review, always approve
- Add unit tests for all mode behaviors
- Add PipelineConfig for TOML configuration integration
"
```

---

## Phase 12: End-to-End Integration

### Task 17: Wire up complete pipeline with all components

**Files:**
- Modify: `cmd/translation-worker/main.py`
- Modify: `cmd/translation-worker/orchestrator.py`
- Create: `tests/test_e2e/test_pipeline.py`

**Step 1: Write comprehensive e2e test**

```python
# tests/test_e2e/test_pipeline.py
import pytest
import tempfile
from pathlib import Path
from queue.manager import JobManager
from orchestrator import TranslationOrchestrator

def test_full_pipeline_with_glossary_and_cache(tmp_path):
    """Test complete pipeline: enqueue → translate → cache → bilingual CSV."""
    # 1. Enqueue job
    manager = JobManager()
    job_id = manager.enqueue({
        "source_file": str(tmp_path / "test.pptx"),
        "source_lang": "ja",
        "target_lang": "en"
    })

    # 2. Process (with mock translation)
    orchestrator = TranslationOrchestrator()
    result = orchestrator.process_job(job_id)

    # 3. Verify bilingual CSV generated
    csv_path = tmp_path / "bilingual" / f"{job_id}.csv"
    assert csv_path.exists()

    # 4. Verify cache was used
    cached = orchestrator.cache.retrieve(f"test:{job_id}")
    # ... assertions
```

**Step 2: Update main.py for complete flow**

```python
# cmd/translation-worker/main.py (complete version)
import json
import os
import shutil
from datetime import datetime
from pathlib import Path
from queue.manager import JobManager, JobState
from cache.manager import CacheManager
from cache.backend import FileCacheBackend, RedisCacheBackend
from glossary.loader import load_glossary_from_file
from glossary.matcher import GlossaryMatcher
from output.bilingual_csv import BilingualCSV
from audit.counter import JapaneseCharacterCounter
from quality.comet_scorer import COMETScorer

# Load configuration
config = load_config()

# Initialize components
job_manager = JobManager(
    redis_host=config.get("redis", {}).get("host", "localhost")
)

# Cache backend
if config.get("cache", {}).get("backend") == "redis":
    cache_backend = RedisCacheBackend()
else:
    cache_backend = FileCacheBackend(config.get("cache", {}).get("directory", "/watch/cache"))

cache = CacheManager(cache_backend)

# Glossary
glossary = None
if config.get("glossary", {}).get("enabled"):
    glossary_file = config.get("glossary", {}).get("file", "/config/glossary.json")
    if Path(glossary_file).exists():
        glossary = load_glossary_from_file(glossary_file)

# Create orchestrator
orchestrator = TranslationOrchestrator(config, cache, glossary)

def process_job(job_id: str):
    """Process a translation job through the complete pipeline."""
    job = job_manager._get_job(job_id)
    if not job:
        print(f"[ERROR] Job {job_id} not found")
        return

    print(f"[INFO] Processing job: {job_id}")

    # Update state
    job_manager.set_state(job_id, JobState.PROCESSING, worker_id="worker-1")

    try:
        # Generate cache key and check cache
        cache_key = cache.generate_key(
            source=job.get("source_text", ""),
            provider=config["translation"]["default_provider"],
            model=config["translation"]["default_model"],
            glossary_hash=glossary.version if glossary else ""
        )

        cached = cache.retrieve(cache_key)
        if cached:
            print(f"[INFO] Cache hit for {job_id}")
            # Use cached translation
            target_text = cached["target"]
        else:
            # Translate
            job_manager.set_state(job_id, JobState.TRANSLATING)

            result = orchestrator.translate(
                text=job.get("source_text", ""),
                source_lang=job.get("source_lang", "ja"),
                target_lang=job.get("target_lang", "en")
            )

            if result.success:
                target_text = result.translated_text

                # Store in cache
                cache.store(
                    cache_key=cache_key,
                    source=job.get("source_text", ""),
                    target=target_text,
                    provider=config["translation"]["default_provider"],
                    model=config["translation"]["default_model"]
                )
            else:
                raise Exception(result.error)

        # Generate bilingual CSV
        if config.get("output", {}).get("bilingual_csv", {}).get("enabled"):
            csv_dir = Path(config["output"]["bilingual_csv"]["path"])
            csv_dir.mkdir(parents=True, exist_ok=True)

            csv_path = csv_dir / f"{job_id}.csv"
            generator = BilingualCSV()
            generator.generate([
                {
                    "id": "main",
                    "source": job.get("source_text", ""),
                    "target": target_text,
                    "context": f"job:{job_id}",
                    "confidence": 0.95,
                    "glossary_used": glossary is not None
                }
            ], str(csv_path))

        # Run audit
        if config.get("audit", {}).get("enabled"):
            counter = JapaneseCharacterCounter()
            char_counts = counter.count(job.get("source_text", ""))
            print(f"[AUDIT] Character counts: {char_counts}")

        # Complete
        job_manager.set_state(job_id, JobState.COMPLETED)
        job_manager.publish_progress(job_id, 1.0, "Translation completed")

    except Exception as e:
        print(f"[ERROR] Job {job_id} failed: {e}")
        job_manager.set_state(job_id, JobState.FAILED)
        job_manager.save_checkpoint(
            job_id,
            {"error": str(e), "failed_at": datetime.utcnow().isoformat()}
        )

def worker_loop():
    """Main worker loop for processing jobs from queue."""
    print("[INFO] Worker starting...")

    while True:
        # Dequeue next job
        job = job_manager.dequeue(worker_id=os.getenv("WORKER_ID", "worker-1"))

        if job:
            process_job(job["id"])
        else:
            # No jobs, sleep before next poll
            import time
            time.sleep(1)

def main():
    print("[INFO] Translation worker starting...")
    print(f"[INFO] Mode: hybrid (folder watch + Redis job queue)")

    # Start worker loop
    worker_loop()

if __name__ == "__main__":
    main()
```

**Step 3: Run e2e tests**

```bash
python -m pytest tests/test_e2e/test_pipeline.py -v
# Expected: PASS (with appropriate mocking)
```

**Step 4: Commit end-to-end integration**

```bash
git add cmd/translation-worker/main.py cmd/translation-worker/orchestrator.py tests/test_e2e/
git commit -m "feat(worker): wire up complete pipeline with all components

- Integrate glossary, cache, layout preservation
- Add bilingual CSV generation
- Add audit tools integration
- Add COMET quality scoring integration
- Add Redis job queue for horizontal scaling
- Add checkpoint/resume for fault tolerance
- Add progress publishing via Redis pub/sub
- Connect worker_loop() for continuous job processing
- Add comprehensive e2e test
"
```

---

## Phase 13: Documentation & Deployment

### Task 18: Write updated deployment documentation

**Files:**
- Create: `docs/deployment/translation-worker.md`
- Update: `docker-compose.yml`

**Step 1: Create deployment docs**

```markdown
# Translation Worker Deployment Guide

## Prerequisites

- Docker and Docker Compose
- Redis server
- Optional: PostgreSQL for persistent job storage

## Configuration

Configuration is done via `config.toml`. Key sections:

### Translation Providers
```toml
[translation.providers.anthropic]
enabled = true
api_key_env = "ANTHROPIC_API_KEY"
```

### Glossary
```toml
[glossary]
enabled = true
file = "/config/glossary.json"
matching_mode = "both"
```

### Cache
```toml
[cache]
enabled = true
backend = "redis"  # or "file"
directory = "/watch/cache"
```

### Layout Preservation
```toml
[layout.preservation]
strategy = "autofit"
min_font_size_pct = 60
```

### Bilingual CSV
```toml
[output.bilingual_csv]
enabled = true
path = "/watch/bilingual/"
encoding = "utf-8-sig"
```

### Job Queue
```toml
[job_queue]
enabled = true
backend = "redis"
max_concurrent = 3
```

## Running

```bash
# Development
docker-compose up translation-worker

# Production
docker-compose -f docker-compose.prod.yml up -d

# Scale workers
docker-compose up --scale translation-worker=3
```

## Monitoring

- Health: `GET /healthz`
- Metrics: Prometheus endpoint on port 9090
- Progress: Subscribe to Redis channel `translation:progress`
```

**Step 2: Update docker-compose.yml**

```yaml
version: '3.8'

services:
  translation-worker:
    build: ./cmd/translation-worker
    volumes:
      - ./watch:/watch
      - ./config:/config:ro
    environment:
      - REDIS_HOST=redis
      - WORKER_ID=${HOSTNAME:-worker}-1
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    depends_on:
      - redis
    restart: unless-stopped
    deploy:
      replicas: 1

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --appendonly yes
```

**Step 3: Commit deployment docs**

```bash
git add docs/deployment/ docker-compose.yml
git commit -m "docs(deployment): add comprehensive deployment documentation

- Add deployment guide with all configuration sections
- Document glossary, cache, layout, CSV, queue configs
- Add development and production run commands
- Add scaling instructions
- Add monitoring endpoints documentation
- Add docker-compose.yml with Redis
- Add environment variable reference
"
```

---

## Summary

This updated implementation plan breaks down the translation integration into **18 major tasks** with **85+ individual steps**, following TDD principles:

### Phases Overview

1. **Foundation & Plugin Architecture** (2 tasks)
   - Project structure and TOML configuration
   - Protocol-based plugin system

2. **Glossary System** (2 tasks)
   - Japanese tokenization with fugashi
   - Glossary matching with fuzzy/POS support

3. **Translation Cache** (1 task)
   - File and Redis backends with intelligent invalidation

4. **Layout Preservation** (1 task)
   - Autofit algorithm for JA→EN expansion
   - Overflow detection

5. **Document Parsers** (1 task)
   - pymupdf adoption for PDF (eliminates C++ need)

6. **Provider Abstraction** (4 tasks)
   - Protocol-based provider interface
   - Anthropic, OpenAI, CLI tool providers

7. **Job Queue & State** (1 task)
   - Redis-based queue for horizontal scaling
   - Checkpoint/resume for fault tolerance

8. **Bilingual Output** (1 task)
   - CSV generation and import for review workflow

9. **Audit Tools** (1 task)
   - JP character count breakdown
   - Style compliance checking

10. **Quality Scoring** (1 task)
    - COMET integration for automated quality assessment

11. **Upload Destinations** (1 task)
    - S3, Gengo, Drive upload abstraction

12. **E2E Integration** (1 task)
    - Complete pipeline wiring with all components

13. **Documentation** (1 task)
    - Deployment guide and docker-compose

**Total estimated effort:** ~60-80 hours for a developer familiar with the stack.

**Key changes from original plan:**
- Added comprehensive glossary system (2 tasks)
- Added translation cache with dual backends
- Added layout preservation for JA→EN expansion
- Added bilingual CSV output for review workflow
- Added audit tools (JP count, style check)
- Added Redis job queue for horizontal scaling
- Added checkpoint/resume for fault tolerance
- Changed from ABC to Protocol-based plugins
- Adopted pymupdf instead of custom C++ PDF parser
- Added COMET quality scoring

`★ Insight ─────────────────────────────────────`
**Protocol vs ABC**: Python's `Protocol` enables structural subtyping—classes match the interface without explicit inheritance. This simplifies plugin development and supports duck typing while maintaining type checking.

**Hybrid Architecture Value**: Folder watching provides loose coupling for Gengo downloads while the Redis job queue enables horizontal scaling across multiple workers without race conditions.

**pymupdf Adoption**: Using pymupdf (MuPDF Python bindings) provides C-level PDF performance (~10-20x faster than PyPDF2) without maintaining custom C++ code, reducing complexity while gaining speed.
`─────────────────────────────────────────────────`

**Next:** Use `superpowers:executing-plans` to implement this plan task-by-task.
