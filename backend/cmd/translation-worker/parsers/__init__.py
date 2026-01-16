# parsers/__init__.py
"""
Document parsers for translation worker.

Supports PDF, PPTX, DOCX, XLSX with pymupdf, python-pptx, python-docx, and openpyxl.
"""

from .base import (
    BaseParser,
    FormattingInfo,
    check_library_available,
    require_library,
)

from .pdf_parser import (
    pymupdfParser,
    PDFPage,
    TextBlock,
    create_pdf_parser,
)

from .pptx_parser import (
    PPTXParser,
    SlideInfo,
    TextFrameInfo,
    FormattingInfo as PPTXFormattingInfo,
    create_pptx_parser,
    PYTHON_PPTX_AVAILABLE,
)

from .docx_parser import (
    DOCXParser,
    ParagraphInfo,
    TableInfo,
    FormattingInfo as DOCXFormattingInfo,
    create_docx_parser,
    PYTHON_DOCX_AVAILABLE,
)

from .xlsx_parser import (
    XLSXParser,
    CellInfo,
    WorksheetInfo,
    FormattingInfo as XLSXFormattingInfo,
    create_xlsx_parser,
    OPENPYXL_AVAILABLE,
)

__all__ = [
    # Base module
    "BaseParser",
    "FormattingInfo",
    "check_library_available",
    "require_library",
    # PDF parser
    "pymupdfParser",
    "PDFPage",
    "TextBlock",
    "create_pdf_parser",
    # PPTX parser
    "PPTXParser",
    "SlideInfo",
    "TextFrameInfo",
    "PPTXFormattingInfo",
    "create_pptx_parser",
    "PYTHON_PPTX_AVAILABLE",
    # DOCX parser
    "DOCXParser",
    "ParagraphInfo",
    "TableInfo",
    "DOCXFormattingInfo",
    "create_docx_parser",
    "PYTHON_DOCX_AVAILABLE",
    # XLSX parser
    "XLSXParser",
    "CellInfo",
    "WorksheetInfo",
    "XLSXFormattingInfo",
    "create_xlsx_parser",
    "OPENPYXL_AVAILABLE",
]
