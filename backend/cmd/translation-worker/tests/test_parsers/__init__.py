# tests/test_parsers/__init__.py
"""Tests for parsers module."""

# Test availability flags
PDF_AVAILABLE = True
PPTX_AVAILABLE = True
DOCX_AVAILABLE = True

# Optional: Import parsers if dependencies are available
try:
    import fitz  # pymupdf
except ImportError:
    PDF_AVAILABLE = False

try:
    import pptx  # python-pptx
except ImportError:
    PPTX_AVAILABLE = False

try:
    import docx  # python-docx
except ImportError:
    DOCX_AVAILABLE = False
