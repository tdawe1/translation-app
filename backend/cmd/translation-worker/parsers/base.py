# parsers/base.py
"""
Shared base classes and utilities for document parsers.

Provides common dataclasses, base parser class, and utility functions
to reduce duplication across xlsx, docx, pptx, and pdf parsers.
"""

from abc import ABC, abstractmethod
from typing import List, Dict, Any, Optional, Tuple
from dataclasses import dataclass
from pathlib import Path
import logging
import os
import sys

# Ensure plugins are importable
worker_dir = Path(__file__).parent.parent
sys.path.insert(0, str(worker_dir))

from plugins import ParsedDocument, Segment

logger = logging.getLogger(__name__)


class FilePathValidationError(ValueError):
    """Raised when a file path is outside the configured parser allowlist."""


@dataclass
class FormattingInfo:
    """Text formatting information shared across parsers.

    Common attributes for font styling, colors, and alignment
    used by xlsx, docx, pptx, and pdf parsers.
    """

    font_name: str = ""
    font_size: float = 0.0
    bold: bool = False
    italic: bool = False
    underline: bool = False
    color_rgb: Optional[Tuple[int, int, int]] = None
    highlight_color: Optional[str] = None
    alignment: str = ""  # left, center, right, justify


class BaseParser(ABC):
    """Abstract base class for document parsers.

    Provides common configuration handling, lifecycle methods,
    and interface for parsing and rendering documents.
    """

    # Subclasses must define these
    name: str = "base_parser"
    version: str = "1.0.0"
    dependencies: List[str] = []

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """Initialize parser with optional configuration.

        Args:
            config: Optional configuration dict
        """
        self.config = config or {}
        self._apply_config()

    def _apply_config(self) -> None:
        """Apply configuration to parser settings.

        Override in subclasses to handle specific config options.
        """
        pass

    def initialize(self, config: Dict[str, Any]) -> None:
        """Initialize plugin with configuration.

        Args:
            config: Configuration dict
        """
        self.config.update(config)
        self._apply_config()
        logger.info(f"[{self.name}] Initialized with config: {self.config}")

    def shutdown(self) -> None:
        """Clean up resources."""
        logger.debug(f"[{self.name}] Shutdown")

    @abstractmethod
    def supported_extensions(self) -> List[str]:
        """Return list of supported file extensions.

        Returns:
            List of supported extensions (e.g., ['.pdf', '.docx'])
        """
        pass

    @abstractmethod
    def parse(self, file_path: str) -> ParsedDocument:
        """Parse document and extract translatable text segments.

        Args:
            file_path: Path to document file

        Returns:
            ParsedDocument with segments

        Raises:
            FileNotFoundError: If file doesn't exist
            Exception: If document cannot be parsed
        """
        pass

    @abstractmethod
    def render(
        self,
        doc: ParsedDocument,
        output_path: str,
        template_path: Optional[str] = None,
    ) -> None:
        """Render translated document with original layout.

        Args:
            doc: ParsedDocument with translated segments
            output_path: Where to save the output
            template_path: Optional original document as template

        Raises:
            Exception: If rendering fails
        """
        pass

    def validate_file_path(self, file_path: str) -> Path:
        """Validate file exists and return Path object.

        Args:
            file_path: Path to file

        Returns:
            Path object

        Raises:
            FileNotFoundError: If file doesn't exist
        """
        path = Path(file_path)
        if not path.exists():
            raise FileNotFoundError(f"File not found: {file_path}")

        resolved = path.expanduser().resolve()
        allowed_bases = []

        env_vars = [
            "WATCH_INCOMING_DIR",
            "WATCH_PROCESSING_DIR",
            "WATCH_TRANSLATED_DIR",
            "WATCH_FAILED_DIR",
        ]
        for var in env_vars:
            value = os.getenv(var)
            if value:
                allowed_bases.append(Path(value).expanduser().resolve())

        for candidate in ("/watch", "/app/data/uploads"):
            if os.path.exists(candidate):
                allowed_bases.append(Path(candidate).resolve())

        if allowed_bases:
            for base in allowed_bases:
                try:
                    resolved.relative_to(base)
                    return resolved
                except ValueError:
                    continue
            allowed_dirs = ", ".join(str(base) for base in allowed_bases)
            raise FilePathValidationError(
                f"File path '{resolved}' is outside allowed directories: {allowed_dirs}"
            )

        return resolved


def check_library_available(library_name: str, import_name: str) -> bool:
    """Check if a library is available for import.

    Args:
        library_name: Human-readable library name (e.g., 'python-docx')
        import_name: Actual import name (e.g., 'docx')

    Returns:
        True if library is available, False otherwise
    """
    try:
        __import__(import_name)
        return True
    except ImportError:
        logger.debug(f"Library {library_name} ({import_name}) not available")
        return False


def require_library(library_name: str, install_cmd: str) -> None:
    """Raise RuntimeError if library is not available.

    Args:
        library_name: Name of required library
        install_cmd: pip install command for the library

    Raises:
        RuntimeError: If library is not installed
    """
    raise RuntimeError(f"{library_name} is not installed. Install with: {install_cmd}")
