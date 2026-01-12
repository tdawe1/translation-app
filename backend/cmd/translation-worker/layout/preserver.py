# layout/preserver.py
"""
Layout preservation for JA→EN translation expansion.

Handles font size adjustments and overflow detection when Japanese text
expands to English (typically 30-40% more characters).
"""

from dataclasses import dataclass
from typing import Optional, List, Dict
import logging

logger = logging.getLogger(__name__)


@dataclass
class AutofitResult:
    """Result of autofit calculation."""
    new_font_size: float
    expansion_ratio: float
    original_font_size: float
    will_overflow: bool
    source_width: float = 0.0
    target_width: float = 0.0


@dataclass
class Rectangle:
    """A rectangular bounding box."""
    width: float
    height: float

    def __post_init__(self):
        if self.width <= 0:
            raise ValueError("Rectangle width must be positive")
        if self.height <= 0:
            raise ValueError("Rectangle height must be positive")


@dataclass
class Font:
    """Font description."""
    size: float
    min_size: float = 8.0
    max_size: float = 72.0
    name: str = "Arial"

    def __post_init__(self):
        if self.size <= 0:
            raise ValueError("Font size must be positive")
        if self.min_size <= 0:
            raise ValueError("Minimum font size must be positive")
        if self.min_size > self.max_size:
            raise ValueError("Minimum font size cannot exceed maximum")


class AutofitCalculator:
    """Calculates font size adjustments for text expansion.

    Uses character width estimates to predict text rendering width
    and calculate appropriate font size reductions.

    Character width ratios (relative to font size):
    - Japanese (CJK): ~1.0 (roughly square)
    - Latin: ~0.5 (approximately half-width)
    - Space: ~0.3 (narrower than letters)
    """

    # Approximate character widths (relative to font size)
    CHAR_WIDTHS = {
        "ja": 1.0,      # Japanese characters are roughly square
        "en": 0.5,      # English characters are ~half width
        "space": 0.3
    }

    # Unicode ranges for Japanese characters
    HIRAGANA_RANGE = (0x3040, 0x309F)
    KATAKANA_RANGE = (0x30A0, 0x30FF)
    KANJI_RANGE = (0x4E00, 0x9FFF)
    KANJI_HIRAGANA_RANGE = (0x3400, 0x4DBF)

    def __init__(self, char_widths: Optional[Dict[str, float]] = None):
        """Initialize calculator with optional custom character widths.

        Args:
            char_widths: Custom character width ratios (overrides defaults)
        """
        if char_widths:
            self.CHAR_WIDTHS = {**self.CHAR_WIDTHS, **char_widths}

    def calculate(
        self,
        source_text: str,
        target_text: str,
        source_font_size: float,
        bounds: Rectangle,
        font: Optional[Font] = None
    ) -> AutofitResult:
        """Calculate required font size adjustment.

        Args:
            source_text: Original Japanese text
            target_text: Translated English text
            source_font_size: Original font size
            bounds: Container bounding box
            font: Font description (uses defaults if None)

        Returns:
            AutofitResult with calculated adjustments
        """
        if font is None:
            font = Font(size=source_font_size)

        # Estimate character counts
        source_chars = self._count_chars(source_text)
        target_chars = self._count_chars(target_text)

        # Calculate widths at source font size
        source_width = self._calculate_width(source_chars, source_font_size)
        target_width = self._calculate_width(target_chars, source_font_size)

        # Calculate expansion ratio
        expansion_ratio = target_width / source_width if source_width > 0 else 1.0

        # Calculate new font size
        if expansion_ratio > 1.0:
            new_font_size = source_font_size / expansion_ratio
            # Clamp to valid range
            new_font_size = max(font.min_size, min(new_font_size, font.max_size))
        else:
            new_font_size = source_font_size

        # Check if target will still overflow at new size
        adjusted_target_width = target_width * (new_font_size / source_font_size)
        will_overflow = adjusted_target_width > bounds.width

        return AutofitResult(
            new_font_size=new_font_size,
            expansion_ratio=expansion_ratio,
            original_font_size=source_font_size,
            will_overflow=will_overflow,
            source_width=source_width,
            target_width=target_width
        )

    def _calculate_width(self, char_counts: Dict[str, int], font_size: float) -> float:
        """Calculate estimated text width from character counts.

        Args:
            char_counts: Dict with 'ja', 'en', 'space' counts
            font_size: Font size to scale by

        Returns:
            Estimated width in points
        """
        return (
            char_counts["ja"] * self.CHAR_WIDTHS["ja"] +
            char_counts["en"] * self.CHAR_WIDTHS["en"] +
            char_counts["space"] * self.CHAR_WIDTHS["space"]
        ) * font_size

    def _count_chars(self, text: str) -> Dict[str, int]:
        """Count characters by type.

        Args:
            text: Text to analyze

        Returns:
            Dict with counts for 'ja', 'en', 'space'
        """
        counts = {"ja": 0, "en": 0, "space": 0}

        for char in text:
            code = ord(char)
            if char.isspace():
                counts["space"] += 1
            elif self._is_japanese_char(code):
                counts["ja"] += 1
            else:
                counts["en"] += 1

        return counts

    def _is_japanese_char(self, code: int) -> bool:
        """Check if character is in Japanese Unicode ranges.

        Args:
            code: Unicode code point

        Returns:
            True if character is Japanese (hiragana, katakana, or kanji)
        """
        return (
            self.HIRAGANA_RANGE[0] <= code <= self.HIRAGANA_RANGE[1] or
            self.KATAKANA_RANGE[0] <= code <= self.KATAKANA_RANGE[1] or
            self.KANJI_RANGE[0] <= code <= self.KANJI_RANGE[1] or
            self.KANJI_HIRAGANA_RANGE[0] <= code <= self.KANJI_HIRAGANA_RANGE[1]
        )


class LayoutPreserver:
    """Preserves layout during JA→EN translation.

    Strategies:
    - 'autofit': Automatically reduce font size to fit
    - 'warn': Only warn about overflow, don't adjust
    - 'truncate': Truncate text with ellipsis

    Default strategy is 'autofit' with minimum 60% of original font size.
    """

    STRATEGIES = {"autofit", "warn", "truncate"}

    def __init__(
        self,
        strategy: str = "autofit",
        min_font_size_pct: float = 60.0,
        warn_threshold: float = 0.95,
        ellipsis: str = "...",
        calculator: Optional[AutofitCalculator] = None
    ):
        """Initialize layout preserver.

        Args:
            strategy: Layout preservation strategy
            min_font_size_pct: Minimum font size as percentage of original
            warn_threshold: Width usage threshold for warnings (0-1)
            ellipsis: Truncation ellipsis string
            calculator: Optional custom calculator instance

        Raises:
            ValueError: If strategy is not recognized
        """
        if strategy not in self.STRATEGIES:
            raise ValueError(f"Unknown strategy: {strategy}. Use one of {self.STRATEGIES}")

        self.strategy = strategy
        self.min_font_size_pct = min_font_size_pct / 100.0
        self.warn_threshold = warn_threshold
        self.ellipsis = ellipsis
        self.calculator = calculator or AutofitCalculator()

    def will_overflow(
        self,
        text: str,
        bounds_width: float,
        font_size: float,
        max_lines: int = 1
    ) -> bool:
        """Check if text will overflow its container.

        Args:
            text: Text to check
            bounds_width: Container width
            font_size: Current font size
            max_lines: Maximum number of lines allowed

        Returns:
            True if text will overflow
        """
        if bounds_width <= 0 or font_size <= 0:
            return True

        # Simple estimation using average character width
        avg_char_width = font_size * 0.5
        estimated_width = len(text) * avg_char_width

        # Account for multiple lines
        max_width = bounds_width * max_lines

        return estimated_width > max_width

    def adjust_font_size(
        self,
        source_text: str,
        target_text: str,
        current_font_size: float,
        bounds: Rectangle,
        font: Optional[Font] = None
    ) -> float:
        """Calculate adjusted font size for target text.

        Args:
            source_text: Original text
            target_text: Translated text
            current_font_size: Current font size
            bounds: Container bounds
            font: Font description

        Returns:
            Adjusted font size (may be same as current)
        """
        if self.strategy == "warn":
            return current_font_size

        result = self.calculator.calculate(
            source_text=source_text,
            target_text=target_text,
            source_font_size=current_font_size,
            bounds=bounds,
            font=font
        )

        if result.will_overflow and self.strategy == "autofit":
            # Apply minimum percentage constraint
            min_size = current_font_size * self.min_font_size_pct
            return max(result.new_font_size, min_size)

        return current_font_size

    def truncate_text(
        self,
        text: str,
        bounds_width: float,
        font_size: float,
        max_lines: int = 1
    ) -> str:
        """Truncate text to fit within bounds.

        Args:
            text: Text to truncate
            bounds_width: Container width
            font_size: Current font size
            max_lines: Maximum lines

        Returns:
            Truncated text with ellipsis if needed
        """
        avg_char_width = font_size * 0.5
        max_chars = int((bounds_width * max_lines) / avg_char_width)

        if len(text) <= max_chars:
            return text

        # Reserve space for ellipsis
        ellipsis_len = len(self.ellipsis)
        if max_chars > ellipsis_len:
            return text[:max_chars - ellipsis_len] + self.ellipsis

        # Very small bounds - just return ellipsis or first char
        return self.ellipsis if max_chars > ellipsis_len else text[:max_chars]

    def check_forbidden_translations(
        self,
        text: str,
        forbidden: List[str]
    ) -> List[str]:
        """Check for translations that should not be used.

        Args:
            text: Translated text to check
            forbidden: List of forbidden terms/phrases

        Returns:
            List of found forbidden terms (empty if none)
        """
        found = []
        text_lower = text.lower()

        for forbidden_term in forbidden:
            if forbidden_term.lower() in text_lower:
                found.append(forbidden_term)

        return found

    def analyze_expansion(
        self,
        source_text: str,
        target_text: str,
        font_size: float = 12.0
    ) -> Dict[str, any]:
        """Analyze text expansion without bounds checking.

        Useful for reporting and statistics.

        Args:
            source_text: Original text
            target_text: Translated text
            font_size: Font size for calculations

        Returns:
            Dict with expansion analysis
        """
        source_chars = self.calculator._count_chars(source_text)
        target_chars = self.calculator._count_chars(target_text)

        source_len = len(source_text)
        target_len = len(target_text)

        source_width = self.calculator._calculate_width(source_chars, font_size)
        target_width = self.calculator._calculate_width(target_chars, font_size)

        char_expansion_ratio = target_len / source_len if source_len > 0 else 1.0
        width_expansion_ratio = target_width / source_width if source_width > 0 else 1.0

        return {
            "source_length": source_len,
            "target_length": target_len,
            "char_expansion_ratio": char_expansion_ratio,
            "width_expansion_ratio": width_expansion_ratio,
            "source_width": source_width,
            "target_width": target_width,
            "width_delta": target_width - source_width,
            "width_delta_pct": ((target_width - source_width) / source_width * 100) if source_width > 0 else 0
        }

    def suggest_font_size(
        self,
        source_text: str,
        target_text: str,
        container_width: float,
        current_font_size: float = 12.0,
        min_font_size: float = 8.0
    ) -> float:
        """Suggest an appropriate font size for the translated text.

        A convenience method that combines calculation with clamping.

        Args:
            source_text: Original Japanese text
            target_text: Translated English text
            container_width: Width of the text container
            current_font_size: Current/original font size
            min_font_size: Absolute minimum font size

        Returns:
            Suggested font size (clamped to min_font_size at minimum)
        """
        result = self.calculator.calculate(
            source_text=source_text,
            target_text=target_text,
            source_font_size=current_font_size,
            bounds=Rectangle(width=container_width, height=100)  # height not used
        )

        if result.expansion_ratio > 1.0:
            suggested = current_font_size / result.expansion_ratio
            return max(suggested, min_font_size)

        return current_font_size


def create_layout_preserver(
    strategy: str = "autofit",
    min_font_size_pct: float = 60.0,
    warn_threshold: float = 0.95
) -> LayoutPreserver:
    """Factory function to create a LayoutPreserver.

    Args:
        strategy: Layout preservation strategy
        min_font_size_pct: Minimum font size percentage
        warn_threshold: Warning threshold

    Returns:
        Configured LayoutPreserver instance
    """
    return LayoutPreserver(
        strategy=strategy,
        min_font_size_pct=min_font_size_pct,
        warn_threshold=warn_threshold
    )
