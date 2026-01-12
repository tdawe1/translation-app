# cmd/translation-worker/plugins/base.py
"""
Protocol-based plugin system for translation worker.

Uses structural subtyping (Protocol) instead of inheritance:
- Plugins don't need to inherit from base classes
- @runtime_checkable enables isinstance() validation
- Any class with matching methods is compatible

This design enables third-party plugins without dependency coupling.

NOTE: In Python 3.11+, @runtime_checkable Protocols cannot inherit from
each other. Each plugin protocol is independent but shares common attributes.
"""

from typing import Protocol, Any, runtime_checkable, List, Optional, Dict
from dataclasses import dataclass
from enum import Enum


class Severity(Enum):
    """Severity levels for quality issues."""
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"
    INFO = "info"


# === Common Plugin Attributes ===

# All plugins should have these attributes
# We check for them separately since Protocols can't inherit with @runtime_checkable
REQUIRED_PLUGIN_ATTRIBUTES = {"name", "version", "dependencies"}


# === Data Structures ===

@dataclass
class Segment:
    """A translatable text segment.

    Attributes:
        id: Unique identifier for this segment
        text: The source text to translate
        context: Additional context (slide number, position, etc.)
        metadata: Formatting information (font size, bold, etc.)
    """
    id: str
    text: str
    context: Dict[str, Any]
    metadata: Dict[str, Any] = None

    def __post_init__(self):
        if self.metadata is None:
            self.metadata = {}


@dataclass
class ParsedDocument:
    """Parsed document structure.

    Attributes:
        segments: List of translatable segments
        metadata: Document-level metadata (format, page count, etc.)
        format: File format identifier (pdf, pptx, docx, etc.)
        source_path: Original file path
    """
    segments: List[Segment]
    metadata: Dict[str, Any]
    format: str
    source_path: str = ""

    def segment_count(self) -> int:
        """Return number of segments."""
        return len(self.segments)

    def total_characters(self) -> int:
        """Return total character count across all segments."""
        return sum(len(s.text) for s in self.segments)

    def filter_by_type(self, segment_type: str) -> List[Segment]:
        """Filter segments by context type."""
        return [
            s for s in self.segments
            if s.context.get("type") == segment_type
        ]


@dataclass
class QualityIssue:
    """A quality issue found during checking.

    Attributes:
        severity: How severe the issue is
        message: Human-readable description
        location: Where the issue was found (segment ID, line, etc.)
        category: Type of issue (grammar, terminology, fluency, etc.)
        suggestion: Optional suggested fix
    """
    severity: Severity
    message: str
    location: str
    category: str = "general"
    suggestion: Optional[str] = None

    def to_dict(self) -> dict:
        """Convert to dictionary for JSON serialization."""
        return {
            "severity": self.severity.value,
            "message": self.message,
            "location": self.location,
            "category": self.category,
            "suggestion": self.suggestion,
        }


@dataclass
class QualityReport:
    """Quality assessment report.

    Attributes:
        score: Overall quality score (0.0-1.0)
        issues: List of quality issues found
        metrics: Additional quality metrics
        passed: Whether quality meets threshold
    """
    score: float
    issues: List[QualityIssue]
    metrics: Dict[str, Any]
    passed: bool = True

    @property
    def critical_count(self) -> int:
        """Count of critical issues."""
        return sum(1 for i in self.issues if i.severity == Severity.CRITICAL)

    @property
    def high_count(self) -> int:
        """Count of high severity issues."""
        return sum(1 for i in self.issues if i.severity == Severity.HIGH)

    def to_dict(self) -> dict:
        """Convert to dictionary for JSON serialization."""
        return {
            "score": self.score,
            "passed": self.passed,
            "issues": [i.to_dict() for i in self.issues],
            "metrics": self.metrics,
            "summary": {
                "critical": self.critical_count,
                "high": self.high_count,
                "total": len(self.issues),
            }
        }


@dataclass
class StageResult:
    """Result of a pipeline stage execution.

    Attributes:
        success: Whether the stage completed successfully
        data: Output data from the stage
        error: Error message if failed
        metadata: Additional metadata
    """
    success: bool
    data: Any = None
    error: Optional[str] = None
    metadata: Dict[str, Any] = None

    def __post_init__(self):
        if self.metadata is None:
            self.metadata = {}


@dataclass
class UploadResult:
    """Result of a file upload operation.

    Attributes:
        success: Whether upload succeeded
        url: URL of uploaded file
        provider: Upload provider name
        metadata: Additional metadata from provider
    """
    success: bool
    url: str = ""
    provider: str = ""
    metadata: Dict[str, Any] = None

    def __post_init__(self):
        if self.metadata is None:
            self.metadata = {}


# === Base Plugin Protocol ===

@runtime_checkable
class Plugin(Protocol):
    """
    Base plugin protocol using structural subtyping.

    All plugins must have these attributes. Classes don't need
    to explicitly inherit - they just need to match this structure.

    NOTE: In Python 3.11+, specialized plugin protocols do NOT
    inherit from this due to @runtime_checkable limitations.
    They duplicate the required attributes instead.
    """
    name: str
    version: str
    dependencies: List[str]

    def initialize(self, config: Dict[str, Any]) -> None:
        """Initialize plugin with configuration."""
        ...

    def shutdown(self) -> None:
        """Clean up resources."""
        ...


# === Parser Plugin Protocol ===

@runtime_checkable
class ParserPlugin(Protocol):
    """
    Document parser plugins.

    Responsible for:
    1. Parsing documents into translatable segments
    2. Preserving layout and formatting information
    3. Rendering translated documents back to original format

    NOTE: Due to Python 3.11+ limitations, this protocol duplicates
    the base Plugin attributes rather than inheriting.
    """
    # Required base attributes (duplicated from Plugin)
    name: str
    version: str
    dependencies: List[str]

    # Optional lifecycle methods
    def initialize(self, config: Dict[str, Any]) -> None:
        """Initialize plugin with configuration."""
        ...

    def shutdown(self) -> None:
        """Clean up resources."""
        ...

    # Parser-specific methods
    def supported_extensions(self) -> List[str]:
        """Return list of supported file extensions.

        Examples: [".pdf"], [".pptx", ".ppt"]
        """
        ...

    def parse(self, file_path: str) -> ParsedDocument:
        """Parse document into translatable segments.

        Args:
            file_path: Path to document file

        Returns:
            ParsedDocument with segments and metadata

        Raises:
            FileNotFoundError: If file doesn't exist
            ParseError: If file format is invalid
        """
        ...

    def render(self, doc: ParsedDocument, output_path: str) -> None:
        """Render translated document back to original format.

        Args:
            doc: ParsedDocument with translated segments
            output_path: Where to write the output file

        Raises:
            RenderError: If rendering fails
        """
        ...


# === Quality Check Plugin Protocol ===

@runtime_checkable
class QualityCheckPlugin(Protocol):
    """
    Quality assessment plugins.

    Evaluates translation quality using various metrics:
    - COMET scores (neural MT evaluation)
    - Grammar checking
    - Terminology consistency
    - Fluency assessment
    """
    # Required base attributes (duplicated from Plugin)
    name: str
    version: str
    dependencies: List[str]

    # Optional lifecycle methods
    def initialize(self, config: Dict[str, Any]) -> None:
        """Initialize plugin with configuration."""
        ...

    def shutdown(self) -> None:
        """Clean up resources."""
        ...

    # Quality check specific methods
    def check(
        self,
        translation: str,
        source: str,
        context: Dict[str, Any]
    ) -> QualityReport:
        """Run quality check and return report.

        Args:
            translation: The translated text
            source: The original source text
            context: Additional context (document type, domain, etc.)

        Returns:
            QualityReport with score and issues
        """
        ...

    def check_batch(
        self,
        translations: List[str],
        sources: List[str],
        contexts: List[Dict[str, Any]]
    ) -> List[QualityReport]:
        """Run quality checks on multiple segments.

        Default implementation calls check() repeatedly.
        Override for batch processing optimization.

        Args:
            translations: List of translated texts
            sources: List of source texts
            contexts: List of context dicts

        Returns:
            List of QualityReports
        """
        ...


# === Pipeline Stage Plugin Protocol ===

@runtime_checkable
class PipelineStagePlugin(Protocol):
    """
    Custom pipeline stage plugins.

    Enables injecting custom processing into the translation pipeline:
    - Pre-processing (text normalization, entity extraction)
    - Post-processing (formatting, cleanup)
    - Custom translation steps
    """
    # Required base attributes
    name: str
    version: str
    dependencies: List[str]
    stage_name: str
    stage_position: str  # "pre", "translation", "post"

    # Optional lifecycle methods
    def initialize(self, config: Dict[str, Any]) -> None:
        """Initialize plugin with configuration."""
        ...

    def shutdown(self) -> None:
        """Clean up resources."""
        ...

    # Pipeline stage specific method
    def execute(
        self,
        job: Any,  # Job type - forward reference
        context: Any  # PipelineContext type - forward reference
    ) -> StageResult:
        """Execute custom pipeline stage.

        Args:
            job: The translation job being processed
            context: Pipeline context with segments, config, etc.

        Returns:
            StageResult with success status and output data
        """
        ...


# === Upload Destination Plugin Protocol ===

@runtime_checkable
class UploadDestinationPlugin(Protocol):
    """
    Upload destination plugins.

    Supports uploading translated documents to various destinations:
    - Google Drive
    - OneDrive
    - S3-compatible storage
    - FTP/SFTP servers
    """
    # Required base attributes
    name: str
    version: str
    dependencies: List[str]

    # Optional lifecycle methods
    def initialize(self, config: Dict[str, Any]) -> None:
        """Initialize plugin with configuration."""
        ...

    def shutdown(self) -> None:
        """Clean up resources."""
        ...

    # Upload specific methods
    def upload(
        self,
        file_path: str,
        metadata: Dict[str, Any]
    ) -> UploadResult:
        """Upload file to destination.

        Args:
            file_path: Path to file to upload
            metadata: Additional metadata (filename, user_id, etc.)

        Returns:
            UploadResult with success status and URL
        """
        ...

    def delete(self, url: str) -> bool:
        """Delete uploaded file.

        Args:
            url: URL of file to delete

        Returns:
            True if deleted successfully
        """
        ...


# === Forward References (to avoid circular imports) ===

# These are defined elsewhere but referenced in protocols above
class Job:
    """Translation job with segments and configuration."""
    pass


class PipelineContext:
    """Context passed between pipeline stages."""
    pass


# === Helper Functions ===

def has_plugin_attributes(obj: Any) -> bool:
    """Check if object has required plugin attributes.

    Args:
        obj: Object to check

    Returns:
        True if object has name, version, dependencies attributes
    """
    return all(hasattr(obj, attr) for attr in REQUIRED_PLUGIN_ATTRIBUTES)


def is_plugin(obj: Any) -> bool:
    """Check if object is a valid plugin.

    Args:
        obj: Object to check

    Returns:
        True if object has required plugin attributes
    """
    return has_plugin_attributes(obj)
