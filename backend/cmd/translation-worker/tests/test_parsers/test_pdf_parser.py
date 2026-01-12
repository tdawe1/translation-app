# tests/test_parsers/test_pdf_parser.py
"""
Unit tests for PDF parser functionality.

Tests pymupdf-based PDF parsing, extraction, and rendering.
Tests gracefully skip when pymupdf is not installed.
"""

import pytest
import sys
import tempfile
from pathlib import Path

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

# Import parser module
try:
    from parsers.pdf_parser import (
        pymupdfParser,
        PDFPage,
        TextBlock,
        create_pdf_parser,
        PYMUPDF_AVAILABLE,
    )
    PYMUPDF_INSTALLED = PYMUPDF_AVAILABLE
except ImportError:
    PYMUPDF_INSTALLED = False
    pymupdfParser = None
    PDFPage = None
    TextBlock = None
    create_pdf_parser = None


@pytest.mark.skipif(not PYMUPDF_INSTALLED, reason="pymupdf not installed")
class TestPDFPage:
    """Test PDFPage dataclass."""

    def test_create_pdf_page(self):
        """Should create PDFPage with correct attributes."""
        page = PDFPage(
            number=1,
            width=595.0,
            height=842.0,
            text="Sample text",
            char_count=11,
            line_count=1
        )

        assert page.number == 1
        assert page.width == 595.0
        assert page.height == 842.0
        assert page.text == "Sample text"
        assert page.char_count == 11
        assert page.line_count == 1


@pytest.mark.skipif(not PYMUPDF_INSTALLED, reason="pymupdf not installed")
class TestTextBlock:
    """Test TextBlock dataclass."""

    def test_create_text_block(self):
        """Should create TextBlock with bbox."""
        block = TextBlock(
            bbox=(10.0, 20.0, 100.0, 50.0),
            text="Hello",
            font="Arial",
            font_size=12.0
        )

        assert block.bbox == (10.0, 20.0, 100.0, 50.0)
        assert block.text == "Hello"
        assert block.font == "Arial"
        assert block.font_size == 12.0
        assert block.is_bold is False
        assert block.is_italic is False

    def test_text_block_style_flags(self):
        """Should track bold and italic flags."""
        block = TextBlock(
            bbox=(0, 0, 50, 20),
            text="Styled",
            is_bold=True,
            is_italic=True
        )

        assert block.is_bold is True
        assert block.is_italic is True


@pytest.mark.skipif(not PYMUPDF_INSTALLED, reason="pymupdf not installed")
class TestPymupdfParser:
    """Test pymupdfParser functionality."""

    def test_plugin_attributes(self):
        """Should have required plugin attributes."""
        parser = pymupdfParser()

        assert hasattr(parser, 'name')
        assert hasattr(parser, 'version')
        assert hasattr(parser, 'dependencies')
        assert parser.name == "pymupdf_pdf"
        assert parser.version == "1.0.0"
        assert "pymupdf" in parser.dependencies

    def test_supported_extensions(self):
        """Should return PDF extension."""
        parser = pymupdfParser()

        extensions = parser.supported_extensions()
        assert ".pdf" in extensions
        assert len(extensions) == 1

    def test_initialization_default_config(self):
        """Should initialize with default config."""
        parser = pymupdfParser()

        assert parser._extract_images is False
        assert parser._preserve_layout is True
        assert parser._merge_blocks is True

    def test_initialization_custom_config(self):
        """Should accept custom configuration."""
        parser = pymupdfParser(config={
            "extract_images": True,
            "preserve_layout": False,
            "merge_blocks": False
        })

        assert parser._extract_images is True
        assert parser._preserve_layout is False
        assert parser._merge_blocks is False

    def test_initialize_method(self):
        """Should update config via initialize method."""
        parser = pymupdfParser()
        parser.initialize({"extract_images": True})

        assert parser._extract_images is True

    def test_shutdown(self):
        """Should shutdown without errors."""
        parser = pymupdfParser()
        parser.shutdown()  # Should not raise

    def test_parse_nonexistent_file(self):
        """Should raise FileNotFoundError for missing file."""
        parser = pymupdfParser()

        with pytest.raises(FileNotFoundError):
            parser.parse("/nonexistent/file.pdf")

    def test_create_pdf_parser_factory(self):
        """Should create parser via factory function."""
        parser = create_pdf_parser()

        assert isinstance(parser, pymupdfParser)

    def test_create_pdf_parser_with_config(self):
        """Should create parser with config via factory."""
        parser = create_pdf_parser({"merge_blocks": False})

        assert parser._merge_blocks is False

    def test_extract_text_by_page_missing_file(self):
        """Should raise error for missing file."""
        parser = pymupdfParser()

        with pytest.raises(Exception):
            parser.extract_text_by_page("/nonexistent/file.pdf")

    def test_get_page_count_missing_file(self):
        """Should raise error for missing file."""
        parser = pymupdfParser()

        with pytest.raises(Exception):
            parser.get_page_count("/nonexistent/file.pdf")

    def test_extract_metadata_missing_file(self):
        """Should raise error for missing file."""
        parser = pymupdfParser()

        with pytest.raises(Exception):
            parser.extract_metadata("/nonexistent/file.pdf")

    def test_merge_blocks_empty(self):
        """Should handle empty blocks list."""
        parser = pymupdfParser()
        result = parser._merge_blocks_into_text([])

        assert result == ""

    def test_merge_blocks_single(self):
        """Should handle single block."""
        parser = pymupdfParser()
        block = TextBlock(bbox=(0, 100, 100, 120), text="Hello")

        result = parser._merge_blocks_into_text([block])

        assert "Hello" in result

    def test_merge_blocks_multiple(self):
        """Should merge multiple blocks."""
        parser = pymupdfParser()

        # Two blocks at similar Y position (same line)
        blocks = [
            TextBlock(bbox=(0, 100, 50, 120), text="Hello "),
            TextBlock(bbox=(60, 100, 120, 120), text="World")
        ]

        result = parser._merge_blocks_into_text(blocks)

        assert "Hello" in result
        assert "World" in result

    def test_merge_blocks_multiline(self):
        """Should handle multiple lines."""
        parser = pymupdfParser()

        # Blocks at different Y positions (different lines)
        blocks = [
            TextBlock(bbox=(0, 100, 100, 120), text="Line 1"),
            TextBlock(bbox=(0, 80, 100, 95), text="Line 2"),
            TextBlock(bbox=(0, 60, 100, 75), text="Line 3")
        ]

        result = parser._merge_blocks_into_text(blocks)

        # Each should be on separate line
        lines = result.strip().split('\n')
        assert len(lines) >= 2  # At least 2 lines


@pytest.mark.skipif(not PYMUPDF_INSTALLED, reason="pymupdf not installed")
class TestPymupdfParserWithSamplePDF:
    """Test parser with actual PDF file."""

    def test_parse_simple_pdf(self):
        """Should parse a simple PDF document."""
        import fitz

        parser = pymupdfParser()

        # Create a simple test PDF
        with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as tmp:
            doc = fitz.open()
            page = doc.new_page(width=595, height=842)
            page.insert_text((50, 800), "Test Document")
            page.insert_text((50, 780), "This is page 1")
            doc.save(tmp.name)
            doc.close()

            try:
                parsed = parser.parse(tmp.name)

                assert parsed.format == "pdf"
                assert len(parsed.segments) >= 1
                assert parsed.metadata["page_count"] == 1
                assert "Test Document" in parsed.segments[0].text
            finally:
                Path(tmp.name).unlink(missing_ok=True)

    def test_parse_multi_page_pdf(self):
        """Should parse multi-page PDF."""
        import fitz

        parser = pymupdfParser()

        with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as tmp:
            doc = fitz.open()
            for i in range(3):
                page = doc.new_page(width=595, height=842)
                page.insert_text((50, 800 - (i * 100)), f"Page {i + 1}")
            doc.save(tmp.name)
            doc.close()

            try:
                parsed = parser.parse(tmp.name)

                assert parsed.metadata["page_count"] == 3
                assert len(parsed.segments) == 3
            finally:
                Path(tmp.name).unlink(missing_ok=True)

    def test_extract_text_by_page(self):
        """Should extract text page by page."""
        import fitz

        parser = pymupdfParser()

        with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as tmp:
            doc = fitz.open()
            page = doc.new_page()
            page.insert_text((50, 800), "First page")
            page.insert_text((50, 780), "Second line")
            doc.save(tmp.name)
            doc.close()

            try:
                pages = parser.extract_text_by_page(tmp.name)

                assert len(pages) == 1
                assert "First page" in pages[0]
                assert "Second line" in pages[0]
            finally:
                Path(tmp.name).unlink(missing_ok=True)

    def test_get_page_count(self):
        """Should return correct page count."""
        import fitz

        parser = pymupdfParser()

        with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as tmp:
            doc = fitz.open()
            for _ in range(5):
                doc.new_page()
            doc.save(tmp.name)
            doc.close()

            try:
                count = parser.get_page_count(tmp.name)
                assert count == 5
            finally:
                Path(tmp.name).unlink(missing_ok=True)

    def test_extract_metadata(self):
        """Should extract PDF metadata."""
        import fitz

        parser = pymupdfParser()

        with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as tmp:
            doc = fitz.open()
            doc.set_metadata({"title": "Test Title", "author": "Test Author"})
            doc.new_page()
            doc.save(tmp.name)
            doc.close()

            try:
                metadata = parser.extract_metadata(tmp.name)

                assert metadata["title"] == "Test Title"
                assert metadata["author"] == "Test Author"
                assert metadata["page_count"] == 1
                assert "pdf_version" in metadata
            finally:
                Path(tmp.name).unlink(missing_ok=True)

    def test_render_with_template(self):
        """Should render PDF using template."""
        import fitz

        parser = pymupdfParser()

        # Create source PDF
        with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as source_tmp:
            with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as output_tmp:
                doc = fitz.open()
                page = doc.new_page()
                page.insert_text((50, 800), "Original text")
                doc.save(source_tmp.name)
                doc.close()

                try:
                    # Create parsed document with translation
                    from plugins import ParsedDocument, Segment

                    parsed = ParsedDocument(
                        segments=[Segment(
                            id="page_1",
                            text="Translated text",
                            context={"page_number": 1, "page_width": 595, "page_height": 842},
                            metadata={"blocks": [{"bbox": (50, 788, 200, 800), "text": "Original text"}]}
                        )],
                        metadata={"format": "pdf", "page_count": 1},
                        format="pdf",
                        source_path=source_tmp.name
                    )

                    # Render with template
                    parser.render(
                        doc=parsed,
                        output_path=output_tmp.name,
                        template_path=source_tmp.name
                    )

                    # Verify output file exists and has content
                    assert Path(output_tmp.name).exists()
                    assert Path(output_tmp.name).stat().st_size > 0

                finally:
                    Path(source_tmp.name).unlink(missing_ok=True)
                    Path(output_tmp.name).unlink(missing_ok=True)

    def test_render_without_template(self):
        """Should render PDF without template (blank pages)."""
        import fitz

        parser = pymupdfParser()

        with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as output_tmp:
            try:
                from plugins import ParsedDocument, Segment

                parsed = ParsedDocument(
                    segments=[Segment(
                        id="page_1",
                        text="Test content",
                        context={"page_number": 1, "page_width": 595, "page_height": 842}
                    )],
                    metadata={"format": "pdf"},
                    format="pdf"
                )

                parser.render(doc=parsed, output_path=output_tmp.name)

                assert Path(output_tmp.name).exists()
                assert Path(output_tmp.name).stat().st_size > 0

            finally:
                Path(output_tmp.name).unlink(missing_ok=True)


class TestPymupdfNotInstalled:
    """Test behavior when pymupdf is not installed."""

    def test_parser_raises_when_not_installed(self):
        """Should raise RuntimeError when pymupdf unavailable."""
        if PYMUPDF_INSTALLED:
            pytest.skip("pymupdf is installed")

        with pytest.raises(RuntimeError, match="pymupdf is not installed"):
            pymupdfParser()

    def test_factory_raises_when_not_installed(self):
        """Should raise RuntimeError from factory when pymupdf unavailable."""
        if PYMUPDF_INSTALLED:
            pytest.skip("pymupdf is installed")

        with pytest.raises(RuntimeError, match="pymupdf is not installed"):
            create_pdf_parser()
