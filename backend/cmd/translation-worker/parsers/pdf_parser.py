# parsers/pdf_parser.py
"""
PDF parser using pymupdf (MuPDF) for fast text extraction.

Provides fast, accurate PDF parsing with layout preservation
and multilingual text support.
"""

from typing import List, Dict, Any, Optional, Tuple
from dataclasses import dataclass, field
import logging

logger = logging.getLogger(__name__)

# Try to import pymupdf (imported as fitz)
try:
    import fitz

    PYMUPDF_AVAILABLE = True
except ImportError:
    PYMUPDF_AVAILABLE = False
    fitz = None

# Import from plugins module
import sys
from pathlib import Path

worker_dir = Path(__file__).parent.parent
sys.path.insert(0, str(worker_dir))

from plugins import (
    ParserPlugin,
    ParsedDocument,
    Segment,
)


@dataclass
class PDFPage:
    """Information about a PDF page."""

    number: int  # 1-based page number
    width: float
    height: float
    text: str
    char_count: int
    line_count: int = 0
    blocks: List[Dict[str, Any]] = field(default_factory=list)
    images: int = 0  # Number of images on page


@dataclass
class TextBlock:
    """A text block within a PDF page."""

    bbox: Tuple[float, float, float, float]  # (x0, y0, x1, y1)
    text: str
    font: str = ""
    font_size: float = 0.0
    is_bold: bool = False
    is_italic: bool = False


class pymupdfParser:
    """PDF parser using pymupdf (MuPDF) for fast text extraction.

    Features:
    - Fast text extraction with position information
    - Layout-aware segment creation
    - Font and style detection
    - Multi-page document support
    - Image detection

    Note: pymupdf must be installed. Install with: pip install pymupdf
    """

    name = "pymupdf_pdf"
    version = "1.0.0"
    dependencies = ["pymupdf"]

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """Initialize PDF parser.

        Args:
            config: Optional configuration dict

        Raises:
            RuntimeError: If pymupdf is not installed
        """
        if not PYMUPDF_AVAILABLE:
            raise RuntimeError(
                "pymupdf is not installed. Install with: pip install pymupdf"
            )

        self.config = config or {}
        self._extract_images = self.config.get("extract_images", False)
        self._preserve_layout = self.config.get("preserve_layout", True)
        self._merge_blocks = self.config.get("merge_blocks", True)

    def initialize(self, config: Dict[str, Any]) -> None:
        """Initialize plugin with configuration.

        Args:
            config: Configuration dict
        """
        self.config.update(config)
        self._extract_images = self.config.get("extract_images", False)
        self._preserve_layout = self.config.get("preserve_layout", True)
        self._merge_blocks = self.config.get("merge_blocks", True)
        logger.info(f"[{self.name}] Initialized with config: {self.config}")

    def shutdown(self) -> None:
        """Clean up resources."""
        logger.debug(f"[{self.name}] Shutdown")

    def supported_extensions(self) -> List[str]:
        """Return list of supported file extensions.

        Returns:
            List of supported extensions
        """
        return [".pdf"]

    def parse(self, file_path: str) -> ParsedDocument:
        """Parse PDF and extract translatable text segments.

        Args:
            file_path: Path to PDF file

        Returns:
            ParsedDocument with segments

        Raises:
            FileNotFoundError: If file doesn't exist
            Exception: If PDF cannot be parsed
        """
        from pathlib import Path

        path = Path(file_path)
        if not path.exists():
            raise FileNotFoundError(f"PDF file not found: {file_path}")

        doc = fitz.open(file_path)

        try:
            segments = []
            metadata = {
                "format": "pdf",
                "page_count": len(doc),
                "title": doc.metadata.get("title", ""),
                "author": doc.metadata.get("author", ""),
                "subject": doc.metadata.get("subject", ""),
            }

            for page_num in range(len(doc)):
                page = doc[page_num]
                page_info = self._extract_page(page, page_num)

                # Extract blocks for layout-aware parsing
                blocks = self._extract_blocks(page)

                if self._merge_blocks:
                    # Merge consecutive blocks into segments
                    segment_text = self._merge_blocks_into_text(blocks)
                else:
                    # Create segment per block
                    segment_text = page_info.text

                if segment_text.strip():
                    segments.append(
                        Segment(
                            id=f"page_{page_num + 1}",
                            text=segment_text.strip(),
                            context={
                                "type": "page",
                                "page_number": page_num + 1,
                                "page_width": page_info.width,
                                "page_height": page_info.height,
                            },
                            metadata={
                                "char_count": page_info.char_count,
                                "line_count": page_info.line_count,
                                "image_count": page_info.images,
                                "blocks": [
                                    {"bbox": b.bbox, "text": b.text} for b in blocks
                                ],
                            },
                        )
                    )

            return ParsedDocument(
                segments=segments,
                metadata=metadata,
                format="pdf",
                source_path=file_path,
            )

        finally:
            doc.close()

    def _extract_page(self, page: fitz.Page, page_num: int) -> PDFPage:
        """Extract information from a PDF page.

        Args:
            page: pymupdf Page object
            page_num: 0-based page number

        Returns:
            PDFPage with page information
        """
        rect = page.rect
        text = page.get_text()

        # Count images
        image_list = page.get_images()
        image_count = len(image_list)

        # Count lines
        line_count = len(text.split("\n")) if text else 0

        return PDFPage(
            number=page_num + 1,  # 1-based
            width=rect.width,
            height=rect.height,
            text=text,
            char_count=len(text),
            line_count=line_count,
            images=image_count,
        )

    def _extract_blocks(self, page: fitz.Page) -> List[TextBlock]:
        """Extract text blocks from a page.

        Args:
            page: pymupdf Page object

        Returns:
            List of TextBlock objects
        """
        blocks = []
        try:
            text_blocks = page.get_text("dict")["blocks"]

            for block in text_blocks:
                if block["type"] == 0:  # Text block
                    for line in block.get("lines", []):
                        for span in line.get("spans", []):
                            text_block = TextBlock(
                                bbox=tuple(block["bbox"]),
                                text=span["text"],
                                font=span.get("font", ""),
                                font_size=span.get("size", 0.0),
                                is_bold=span.get("flags", 0) & 2**4 != 0,
                                is_italic=span.get("flags", 0) & 2**1 != 0,
                            )
                            blocks.append(text_block)
        except Exception as e:
            logger.warning(f"[{self.name}] Failed to extract blocks: {e}")

        return blocks

    def _merge_blocks_into_text(self, blocks: List[TextBlock]) -> str:
        """Merge text blocks into a single text string.

        Args:
            blocks: List of TextBlock objects

        Returns:
            Merged text string
        """
        if not blocks:
            return ""

        # Sort blocks by vertical position (top to bottom)
        sorted_blocks = sorted(blocks, key=lambda b: b.bbox[1], reverse=True)

        # Group by line (similar y position)
        lines = []
        current_line = [sorted_blocks[0]]
        current_y = sorted_blocks[0].bbox[1]
        line_tolerance = 5.0  # Pixels

        for block in sorted_blocks[1:]:
            if abs(block.bbox[1] - current_y) < line_tolerance:
                current_line.append(block)
            else:
                lines.append(current_line)
                current_line = [block]
                current_y = block.bbox[1]

        if current_line:
            lines.append(current_line)

        # Sort each line horizontally and join
        text_lines = []
        for line in lines:
            sorted_line = sorted(line, key=lambda b: b.bbox[0])
            line_text = "".join(b.text for b in sorted_line)
            text_lines.append(line_text)

        return "\n".join(text_lines)

    def render(
        self,
        doc: ParsedDocument,
        output_path: str,
        template_path: Optional[str] = None,
        font_size: float = 11.0,
        font_name: str = "helv",
    ) -> None:
        """Render translated PDF with original layout.

        Args:
            doc: ParsedDocument with translated segments
            output_path: Where to save the output PDF
            template_path: Optional original PDF for layout template
            font_size: Font size for rendered text
            font_name: Font name (pymupdf format)

        Raises:
            Exception: If rendering fails
        """
        if not PYMUPDF_AVAILABLE:
            raise RuntimeError("pymupdf is not installed")

        # Load template PDF if provided, otherwise create new
        if template_path:
            source_doc = fitz.open(template_path)
        else:
            # Create new A4 document
            source_doc = fitz.open()
            source_doc.new_page(width=595, height=842)  # A4 size

        try:
            out_doc = fitz.open()

            for segment in doc.segments:
                page_num = segment.context.get("page_number", 1) - 1

                # Copy page from template or create new
                if template_path and page_num < len(source_doc):
                    page = out_doc.new_page(
                        width=source_doc[page_num].rect.width,
                        height=source_doc[page_num].rect.height,
                    )
                    # show_pdf_page expects (rect, src_doc, pno) - src_doc is the Document, not Page
                    page.show_pdf_page(page.rect, source_doc, pno=page_num)

                    # Clear original text by overlaying white rectangles
                    # In production, would use more sophisticated text removal
                    for block_info in segment.metadata.get("blocks", []):
                        bbox = block_info.get("bbox")
                        if bbox and len(bbox) == 4:
                            # Draw white rectangle over original text
                            rect = fitz.Rect(bbox[0], bbox[1], bbox[2], bbox[3])
                            page.draw_rect(rect, color=(1, 1, 1))
                else:
                    # Create new page
                    page_width = segment.context.get("page_width", 595)
                    page_height = segment.context.get("page_height", 842)
                    page = out_doc.new_page(width=page_width, height=page_height)

                # Insert translated text
                # For production, would use proper text positioning
                # based on original text block positions
                margin = 50
                y_position = page.rect.height - margin

                for line in segment.text.split("\n"):
                    if line.strip():
                        page.insert_text(
                            (margin, y_position),
                            line,
                            fontsize=font_size,
                            fontname=font_name,
                        )
                        y_position -= font_size * 1.5

            out_doc.save(output_path)
            logger.info(f"[{self.name}] Rendered PDF to: {output_path}")

        finally:
            source_doc.close()
            out_doc.close()

    def extract_text_by_page(self, file_path: str) -> List[str]:
        """Extract text from PDF, one page at a time.

        Convenience method for simple text extraction.

        Args:
            file_path: Path to PDF file

        Returns:
            List of text strings, one per page
        """
        if not PYMUPDF_AVAILABLE:
            raise RuntimeError("pymupdf is not installed")

        doc = fitz.open(file_path)
        pages_text = []

        try:
            for page in doc:
                text = page.get_text()
                pages_text.append(text.strip())
        finally:
            doc.close()

        return pages_text

    def get_page_count(self, file_path: str) -> int:
        """Get the number of pages in a PDF.

        Args:
            file_path: Path to PDF file

        Returns:
            Number of pages
        """
        if not PYMUPDF_AVAILABLE:
            raise RuntimeError("pymupdf is not installed")

        doc = fitz.open(file_path)
        try:
            return len(doc)
        finally:
            doc.close()

    def extract_metadata(self, file_path: str) -> Dict[str, Any]:
        """Extract metadata from PDF.

        Args:
            file_path: Path to PDF file

        Returns:
            Metadata dict with keys: title, author, subject, keywords, creator, producer, page_count
        """
        if not PYMUPDF_AVAILABLE:
            raise RuntimeError("pymupdf is not installed")

        doc = fitz.open(file_path)
        try:
            metadata = doc.metadata
            return {
                "title": metadata.get("title", ""),
                "author": metadata.get("author", ""),
                "subject": metadata.get("subject", ""),
                "keywords": metadata.get("keywords", ""),
                "creator": metadata.get("creator", ""),
                "producer": metadata.get("producer", ""),
                "page_count": len(doc),
                "pdf_version": metadata.get("format", ""),
                "encrypted": doc.is_encrypted,
            }
        finally:
            doc.close()


def create_pdf_parser(config: Optional[Dict[str, Any]] = None) -> pymupdfParser:
    """Factory function to create a PDF parser.

    Args:
        config: Optional configuration dict

    Returns:
        pymupdfParser instance

    Raises:
        RuntimeError: If pymupdf is not installed
    """
    return pymupdfParser(config)
