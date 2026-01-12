# tests/test_plugins/test_base.py
"""
Unit tests for plugin base protocols.

Tests Protocol-based plugin system, structural subtyping,
and dataclass structures.
"""

import pytest
import sys
from pathlib import Path

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from plugins.base import (
    Plugin,
    ParserPlugin,
    QualityCheckPlugin,
    PipelineStagePlugin,
    UploadDestinationPlugin,
    Segment,
    ParsedDocument,
    QualityIssue,
    QualityReport,
    Severity,
    UploadResult,
)


class TestParserPlugin:
    """Test parser plugin protocol compliance."""

    def test_parser_plugin_protocol(self):
        """Plugin protocol should define required attributes."""
        # Create a class that structurally matches ParserPlugin
        class TestParser:
            name = "test"
            version = "1.0"
            dependencies = []

            def initialize(self, config):
                pass

            def shutdown(self):
                pass

            def supported_extensions(self):
                return [".test"]

            def parse(self, file_path):
                return ParsedDocument(
                    segments=[],
                    metadata={},
                    format="test"
                )

            def render(self, doc, output_path):
                pass

        parser = TestParser()

        # Should pass isinstance check with @runtime_checkable Protocol
        assert isinstance(parser, Plugin)
        assert isinstance(parser, ParserPlugin)
        assert parser.name == "test"
        assert parser.version == "1.0"

    def test_parser_with_minimal_interface(self):
        """Should accept parser with minimal required interface."""
        class MinimalParser:
            name = "minimal"
            version = "1.0"
            dependencies = []

            def initialize(self, config):
                pass

            def shutdown(self):
                pass

            def supported_extensions(self):
                return [".txt"]

            def parse(self, file_path):
                return ParsedDocument(
                    segments=[Segment(id="1", text="test", context={})],
                    metadata={},
                    format="txt"
                )

            def render(self, doc, output_path):
                pass

        parser = MinimalParser()
        assert isinstance(parser, ParserPlugin)

        # Test parsing
        result = parser.parse("dummy.txt")
        assert len(result.segments) == 1
        assert result.segments[0].text == "test"

    def test_parser_not_matching_protocol(self):
        """Should reject class missing required methods."""
        class InvalidParser:
            name = "invalid"
            version = "1.0"
            dependencies = []

            # Missing parse() and render() methods

        parser = InvalidParser()
        assert not isinstance(parser, ParserPlugin)


class TestSegment:
    """Test Segment dataclass."""

    def test_segment_creation(self):
        """Should create segment with required fields."""
        segment = Segment(
            id="seg-1",
            text="Hello World",
            context={"slide": 1}
        )

        assert segment.id == "seg-1"
        assert segment.text == "Hello World"
        assert segment.context["slide"] == 1
        assert segment.metadata == {}  # Default initialized

    def test_segment_with_metadata(self):
        """Should store formatting metadata."""
        segment = Segment(
            id="seg-2",
            text="Bold text",
            context={},
            metadata={"font_size": 14, "bold": True}
        )

        assert segment.metadata["font_size"] == 14
        assert segment.metadata["bold"] is True


class TestParsedDocument:
    """Test ParsedDocument dataclass."""

    def test_document_creation(self):
        """Should create document with segments."""
        segments = [
            Segment(id="1", text="First", context={"page": 1}),
            Segment(id="2", text="Second", context={"page": 2}),
        ]

        doc = ParsedDocument(
            segments=segments,
            metadata={"pages": 2},
            format="pdf"
        )

        assert doc.segment_count() == 2
        assert doc.total_characters() == 11  # "First" (5) + "Second" (6)
        assert doc.format == "pdf"

    def test_filter_by_type(self):
        """Should filter segments by context type."""
        segments = [
            Segment(id="1", text="Title", context={"type": "heading"}),
            Segment(id="2", text="Body", context={"type": "paragraph"}),
            Segment(id="3", text="More body", context={"type": "paragraph"}),
        ]

        doc = ParsedDocument(
            segments=segments,
            metadata={},
            format="docx"
        )

        paragraphs = doc.filter_by_type("paragraph")
        assert len(paragraphs) == 2
        assert all(s.context["type"] == "paragraph" for s in paragraphs)


class TestQualityIssue:
    """Test QualityIssue dataclass."""

    def test_quality_issue_creation(self):
        """Should create quality issue with all fields."""
        issue = QualityIssue(
            severity=Severity.HIGH,
            message="Missing translation",
            location="segment-5",
            category="completeness",
            suggestion="Add translation for this segment"
        )

        assert issue.severity == Severity.HIGH
        assert issue.message == "Missing translation"
        assert issue.suggestion == "Add translation for this segment"

    def test_to_dict(self):
        """Should serialize to dictionary."""
        issue = QualityIssue(
            severity=Severity.MEDIUM,
            message="Terminology inconsistency",
            location="segment-10",
            category="terminology"
        )

        result = issue.to_dict()
        assert result["severity"] == "medium"
        assert result["message"] == "Terminology inconsistency"
        assert result["category"] == "terminology"
        assert result["suggestion"] is None


class TestQualityReport:
    """Test QualityReport dataclass."""

    def test_quality_report_creation(self):
        """Should create quality report."""
        issues = [
            QualityIssue(Severity.LOW, "Minor issue", "seg-1"),
            QualityIssue(Severity.HIGH, "Major issue", "seg-2"),
        ]

        report = QualityReport(
            score=0.85,
            issues=issues,
            metrics={"comet": 0.82, "fluency": 0.88}
        )

        assert report.score == 0.85
        assert len(report.issues) == 2
        assert report.passed is True  # Default

    def test_issue_counts(self):
        """Should count issues by severity."""
        issues = [
            QualityIssue(Severity.CRITICAL, "Critical bug", "seg-1"),
            QualityIssue(Severity.CRITICAL, "Another critical", "seg-2"),
            QualityIssue(Severity.HIGH, "High issue", "seg-3"),
            QualityIssue(Severity.LOW, "Minor issue", "seg-4"),
        ]

        report = QualityReport(
            score=0.5,
            issues=issues,
            metrics={}
        )

        assert report.critical_count == 2
        assert report.high_count == 1

    def test_to_dict(self):
        """Should serialize report to dictionary."""
        issues = [
            QualityIssue(Severity.HIGH, "Issue", "seg-1"),
        ]

        report = QualityReport(
            score=0.9,
            issues=issues,
            metrics={"comet": 0.88},
            passed=True
        )

        result = report.to_dict()
        assert result["score"] == 0.9
        assert result["passed"] is True
        assert result["summary"]["critical"] == 0
        assert result["summary"]["high"] == 1
        assert result["summary"]["total"] == 1


class TestQualityCheckPlugin:
    """Test quality check plugin protocol."""

    def test_quality_check_plugin(self):
        """Should accept class implementing QualityCheckPlugin protocol."""
        class MockQualityChecker:
            name = "mock_checker"
            version = "1.0"
            dependencies = []

            def initialize(self, config):
                pass

            def shutdown(self):
                pass

            def check(self, translation, source, context):
                return QualityReport(
                    score=0.8,
                    issues=[],
                    metrics={}
                )

            def check_batch(self, translations, sources, contexts):
                # Default implementation
                return [self.check(t, s, c) for t, s, c in zip(translations, sources, contexts)]

        checker = MockQualityChecker()
        assert isinstance(checker, Plugin)
        assert isinstance(checker, QualityCheckPlugin)

    def test_check_method(self):
        """Should call check method correctly."""
        class MockQualityChecker:
            name = "mock_checker"
            version = "1.0"
            dependencies = []

            def check(self, translation, source, context):
                # Simple heuristic: score based on length ratio
                ratio = len(translation) / max(len(source), 1)
                score = max(0, 1 - abs(ratio - 1.5) * 0.5)  # Expect 1.5x expansion
                return QualityReport(
                    score=score,
                    issues=[],
                    metrics={"length_ratio": ratio}
                )

        checker = MockQualityChecker()
        report = checker.check("Hello world", "こんにちは", {})

        assert 0 <= report.score <= 1
        assert "length_ratio" in report.metrics


class TestUploadDestinationPlugin:
    """Test upload destination plugin protocol."""

    def test_upload_plugin(self):
        """Should accept class implementing UploadDestinationPlugin protocol."""
        class MockUploader:
            name = "mock_upload"
            version = "1.0"
            dependencies = []

            def initialize(self, config):
                pass

            def shutdown(self):
                pass

            def upload(self, file_path, metadata):
                return UploadResult(
                    success=True,
                    url="https://example.com/file.pdf",
                    provider="mock"
                )

            def delete(self, url):
                return True

        uploader = MockUploader()
        assert isinstance(uploader, Plugin)
        assert isinstance(uploader, UploadDestinationPlugin)

        # Test upload
        result = uploader.upload("/path/to/file.pdf", {})
        assert result.success is True
        assert result.url == "https://example.com/file.pdf"

        # Test delete
        assert uploader.delete("https://example.com/file.pdf") is True


class TestSeverity:
    """Test Severity enum."""

    def test_severity_values(self):
        """Should have correct severity levels."""
        assert Severity.CRITICAL.value == "critical"
        assert Severity.HIGH.value == "high"
        assert Severity.MEDIUM.value == "medium"
        assert Severity.LOW.value == "low"
        assert Severity.INFO.value == "info"

    def test_severity_comparison(self):
        """Should allow comparison by value."""
        assert Severity.CRITICAL.value == "critical"
        assert Severity.HIGH.value != "critical"
