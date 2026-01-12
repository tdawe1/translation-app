# layout/__init__.py
"""
Layout preservation module for JA→EN translation.

Handles text expansion from Japanese to English with font size
adjustments and overflow detection.
"""

from .preserver import (
    AutofitResult,
    Rectangle,
    Font,
    AutofitCalculator,
    LayoutPreserver,
    create_layout_preserver,
)

__all__ = [
    "AutofitResult",
    "Rectangle",
    "Font",
    "AutofitCalculator",
    "LayoutPreserver",
    "create_layout_preserver",
]
