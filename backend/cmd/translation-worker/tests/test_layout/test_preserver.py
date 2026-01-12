# tests/test_layout/test_preserver.py
"""
Unit tests for layout preservation functionality.

Tests autofit calculation, overflow detection, and layout strategies.
"""

import pytest
import sys
from pathlib import Path

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from layout.preserver import (
    AutofitResult,
    Rectangle,
    Font,
    AutofitCalculator,
    LayoutPreserver,
    create_layout_preserver,
)


class TestRectangle:
    """Test Rectangle dataclass."""

    def test_valid_rectangle(self):
        """Should create valid rectangle."""
        rect = Rectangle(width=100.0, height=50.0)
        assert rect.width == 100.0
        assert rect.height == 50.0

    def test_invalid_width(self):
        """Should reject non-positive width."""
        with pytest.raises(ValueError, match="width must be positive"):
            Rectangle(width=0, height=50)

        with pytest.raises(ValueError, match="width must be positive"):
            Rectangle(width=-10, height=50)

    def test_invalid_height(self):
        """Should reject non-positive height."""
        with pytest.raises(ValueError, match="height must be positive"):
            Rectangle(width=100, height=0)

        with pytest.raises(ValueError, match="height must be positive"):
            Rectangle(width=100, height=-10)


class TestFont:
    """Test Font dataclass."""

    def test_valid_font(self):
        """Should create valid font."""
        font = Font(size=12.0)
        assert font.size == 12.0
        assert font.min_size == 8.0
        assert font.max_size == 72.0
        assert font.name == "Arial"

    def test_custom_font(self):
        """Should create font with custom values."""
        font = Font(size=14.0, min_size=10.0, max_size=48.0, name="Helvetica")
        assert font.size == 14.0
        assert font.min_size == 10.0
        assert font.max_size == 48.0
        assert font.name == "Helvetica"

    def test_invalid_size(self):
        """Should reject non-positive size."""
        with pytest.raises(ValueError, match="Font size must be positive"):
            Font(size=0)

        with pytest.raises(ValueError, match="Font size must be positive"):
            Font(size=-12)

    def test_invalid_min_size(self):
        """Should reject non-positive min_size."""
        with pytest.raises(ValueError, match="Minimum font size must be positive"):
            Font(size=12, min_size=0)

    def test_min_exceeds_max(self):
        """Should reject min_size > max_size."""
        with pytest.raises(ValueError, match="Minimum font size cannot exceed maximum"):
            Font(size=12, min_size=50, max_size=20)


class TestAutofitCalculator:
    """Test AutofitCalculator functionality."""

    def test_initialization(self):
        """Should create calculator with defaults."""
        calc = AutofitCalculator()
        assert calc.CHAR_WIDTHS["ja"] == 1.0
        assert calc.CHAR_WIDTHS["en"] == 0.5
        assert calc.CHAR_WIDTHS["space"] == 0.3

    def test_custom_char_widths(self):
        """Should accept custom character widths."""
        calc = AutofitCalculator(char_widths={"ja": 0.9, "en": 0.6})
        assert calc.CHAR_WIDTHS["ja"] == 0.9
        assert calc.CHAR_WIDTHS["en"] == 0.6
        # Default space width preserved
        assert calc.CHAR_WIDTHS["space"] == 0.3

    def test_count_japanese_chars(self):
        """Should count Japanese characters correctly."""
        calc = AutofitCalculator()

        # Hiragana
        counts = calc._count_chars("こんにちは")
        assert counts["ja"] == 5
        assert counts["en"] == 0
        assert counts["space"] == 0

        # Katakana
        counts = calc._count_chars("コンニチハ")
        assert counts["ja"] == 5

        # Kanji
        counts = calc._count_chars("日本語")
        assert counts["ja"] == 3

    def test_count_mixed_chars(self):
        """Should count mixed character types."""
        calc = AutofitCalculator()

        counts = calc._count_chars("Hello 世界 123")
        assert counts["en"] == 8  # Hello, space, 123
        assert counts["ja"] == 2  # 世界
        assert counts["space"] == 2  # two spaces

    def test_count_spaces(self):
        """Should count spaces separately."""
        calc = AutofitCalculator()

        counts = calc._count_chars("a b c")
        assert counts["space"] == 2
        assert counts["en"] == 3

        # Multiple spaces
        counts = calc._count_chars("a   b")
        assert counts["space"] == 3

    def test_is_japanese_char(self):
        """Should identify Japanese characters correctly."""
        calc = AutofitCalculator()

        # Hiragana range
        assert calc._is_japanese_char(0x3042)  # あ
        assert calc._is_japanese_char(0x3093)  # ん

        # Katakana range
        assert calc._is_japanese_char(0x30A2)  # ア
        assert calc._is_japanese_char(0x30FC)  # ー

        # Kanji range
        assert calc._is_japanese_char(0x65E5)  # 日
        assert calc._is_japanese_char(0x672C)  # 本

        # Non-Japanese
        assert not calc._is_japanese_char(ord('A'))
        assert not calc._is_japanese_char(ord(' '))

    def test_calculate_width(self):
        """Should calculate text width correctly."""
        calc = AutofitCalculator()

        # Japanese text: 3 chars * 1.0 * 12 = 36
        counts = {"ja": 3, "en": 0, "space": 0}
        width = calc._calculate_width(counts, 12.0)
        assert width == 36.0

        # English text: 5 chars * 0.5 * 12 = 30
        counts = {"ja": 0, "en": 5, "space": 0}
        width = calc._calculate_width(counts, 12.0)
        assert width == 30.0

        # Mixed: (2 * 1.0 + 4 * 0.5) * 12 = (2 + 2) * 12 = 48
        counts = {"ja": 2, "en": 4, "space": 0}
        width = calc._calculate_width(counts, 12.0)
        assert width == 48.0

    def test_autofit_calculation_no_expansion(self):
        """Should not adjust when text doesn't expand."""
        calc = AutofitCalculator()

        result = calc.calculate(
            source_text="Test",
            target_text="Test",  # Same length
            source_font_size=18.0,
            bounds=Rectangle(width=100, height=50)
        )

        assert result.expansion_ratio == 1.0
        assert result.new_font_size == 18.0
        assert result.will_overflow is False

    def test_autofit_calculation_with_expansion(self):
        """Should calculate font size reduction for text expansion."""
        calc = AutofitCalculator()

        # JA text that expands to 2x English
        result = calc.calculate(
            source_text="営業報告書",  # 6 Japanese chars
            target_text="Business Report",  # ~16 chars with spaces
            source_font_size=18.0,
            bounds=Rectangle(width=100, height=50)
        )

        assert result.new_font_size < 18.0
        assert result.expansion_ratio > 1.0
        assert result.original_font_size == 18.0

    def test_autofit_clamps_to_min_size(self):
        """Should clamp font size to minimum."""
        calc = AutofitCalculator()

        # Very long text that would require tiny font
        result = calc.calculate(
            source_text="日",
            target_text="This is a very long piece of text that would require a tiny font size",
            source_font_size=18.0,
            bounds=Rectangle(width=50, height=20)
        )

        assert result.new_font_size == 8.0  # Default min size

    def test_autofit_clamps_to_max_size(self):
        """Should clamp font size to maximum when reducing."""
        calc = AutofitCalculator()
        font = Font(size=12, min_size=8, max_size=20)

        result = calc.calculate(
            source_text="テスト",  # Shrinks when translated
            target_text="T",  # Much shorter
            source_font_size=12.0,
            bounds=Rectangle(width=100, height=50),
            font=font
        )

        # When text contracts, font size shouldn't exceed max
        assert result.new_font_size <= font.max_size

    def test_autofit_width_tracking(self):
        """Should track source and target widths."""
        calc = AutofitCalculator()

        result = calc.calculate(
            source_text="日本語",
            target_text="Japanese Language",
            source_font_size=12.0,
            bounds=Rectangle(width=200, height=50)
        )

        assert result.source_width > 0
        assert result.target_width > 0


class TestLayoutPreserver:
    """Test LayoutPreserver functionality."""

    def test_initialization(self):
        """Should initialize with defaults."""
        preserver = LayoutPreserver()
        assert preserver.strategy == "autofit"
        assert preserver.min_font_size_pct == 0.6  # 60%
        assert preserver.warn_threshold == 0.95
        assert isinstance(preserver.calculator, AutofitCalculator)

    def test_custom_initialization(self):
        """Should accept custom parameters."""
        preserver = LayoutPreserver(
            strategy="warn",
            min_font_size_pct=50.0,
            warn_threshold=0.9
        )
        assert preserver.strategy == "warn"
        assert preserver.min_font_size_pct == 0.5
        assert preserver.warn_threshold == 0.9

    def test_invalid_strategy(self):
        """Should reject invalid strategy."""
        with pytest.raises(ValueError, match="Unknown strategy"):
            LayoutPreserver(strategy="invalid")

    def test_will_overflow_true(self):
        """Should detect when text will overflow."""
        preserver = LayoutPreserver()

        will_overflow = preserver.will_overflow(
            text="This is a very long text that will definitely overflow the container",
            bounds_width=100,
            font_size=12,
            max_lines=2
        )

        assert will_overflow is True

    def test_will_overflow_false(self):
        """Should detect when text fits."""
        preserver = LayoutPreserver()

        will_overflow = preserver.will_overflow(
            text="Short text",
            bounds_width=100,
            font_size=12
        )

        assert will_overflow is False

    def test_will_overflow_multiple_lines(self):
        """Should account for multiple lines."""
        preserver = LayoutPreserver()

        # Text that fits on one line
        single = preserver.will_overflow(
            text="Medium length text here",
            bounds_width=100,
            font_size=12,
            max_lines=1
        )

        # Same text fits on two lines
        double = preserver.will_overflow(
            text="Medium length text here",
            bounds_width=100,
            font_size=12,
            max_lines=2
        )

        # Double line should be less likely to overflow
        assert double <= single

    def test_adjust_font_size_autofit(self):
        """Should reduce font size in autofit mode."""
        preserver = LayoutPreserver(strategy="autofit")

        # Use narrower bounds that force adjustment even after reduction
        new_size = preserver.adjust_font_size(
            source_text="営業報告書",
            target_text="Business Report",
            current_font_size=18.0,
            bounds=Rectangle(width=50, height=50)  # Narrower bounds
        )

        # Font size should be reduced due to expansion
        assert new_size < 18.0
        assert new_size >= 18.0 * preserver.min_font_size_pct  # Respects min

    def test_adjust_font_size_warn_strategy(self):
        """Should not adjust in warn mode."""
        preserver = LayoutPreserver(strategy="warn")

        new_size = preserver.adjust_font_size(
            source_text="営業報告書",
            target_text="Business Report",
            current_font_size=18.0,
            bounds=Rectangle(width=100, height=50)
        )

        assert new_size == 18.0  # Unchanged

    def test_adjust_font_size_no_overflow(self):
        """Should not adjust when no overflow."""
        preserver = LayoutPreserver(strategy="autofit")

        new_size = preserver.adjust_font_size(
            source_text="Test",
            target_text="Test",  # Same length
            current_font_size=18.0,
            bounds=Rectangle(width=200, height=50)
        )

        assert new_size == 18.0

    def test_truncate_text_no_truncation(self):
        """Should not truncate when text fits."""
        preserver = LayoutPreserver()

        result = preserver.truncate_text(
            text="Short text",
            bounds_width=100,
            font_size=12
        )

        assert result == "Short text"

    def test_truncate_text_with_truncation(self):
        """Should truncate with ellipsis when text overflows."""
        preserver = LayoutPreserver(ellipsis="...")

        result = preserver.truncate_text(
            text="This is very long text that needs truncation",
            bounds_width=50,
            font_size=12
        )

        assert len(result) < len("This is very long text that needs truncation")
        assert "..." in result

    def test_truncate_respects_ellipsis(self):
        """Should include ellipsis in truncated text."""
        preserver = LayoutPreserver(ellipsis="***")

        result = preserver.truncate_text(
            text="Long text here",
            bounds_width=30,
            font_size=12
        )

        assert "***" in result or len(result) <= 3

    def test_check_forbidden_translations_empty(self):
        """Should return empty list when no forbidden terms found."""
        preserver = LayoutPreserver()

        found = preserver.check_forbidden_translations(
            text="This is acceptable translation",
            forbidden=["bad", "terrible", "awful"]
        )

        assert found == []

    def test_check_forbidden_translations_found(self):
        """Should find forbidden terms."""
        preserver = LayoutPreserver()

        found = preserver.check_forbidden_translations(
            text="This translation is bad and terrible",
            forbidden=["bad", "terrible", "awful"]
        )

        assert len(found) == 2
        assert "bad" in found
        assert "terrible" in found

    def test_check_forbidden_case_insensitive(self):
        """Should be case-insensitive when checking."""
        preserver = LayoutPreserver()

        found = preserver.check_forbidden_translations(
            text="This is BAD",
            forbidden=["bad"]
        )

        assert found == ["bad"]

    def test_analyze_expansion(self):
        """Should analyze text expansion metrics."""
        preserver = LayoutPreserver()

        analysis = preserver.analyze_expansion(
            source_text="営業報告書",
            target_text="Business Report",
            font_size=12.0
        )

        assert "source_length" in analysis
        assert "target_length" in analysis
        assert "char_expansion_ratio" in analysis
        assert "width_expansion_ratio" in analysis
        assert "source_width" in analysis
        assert "target_width" in analysis
        assert "width_delta" in analysis
        assert "width_delta_pct" in analysis

        # Target should be longer
        assert analysis["target_length"] > analysis["source_length"]
        assert analysis["char_expansion_ratio"] > 1.0

    def test_suggest_font_size(self):
        """Should suggest appropriate font size."""
        preserver = LayoutPreserver()

        suggested = preserver.suggest_font_size(
            source_text="営業報告書",
            target_text="Business Report",
            container_width=100,
            current_font_size=18.0
        )

        assert suggested < 18.0
        assert suggested >= 8.0  # Default min

    def test_suggest_font_size_with_custom_min(self):
        """Should respect custom minimum font size."""
        preserver = LayoutPreserver()

        suggested = preserver.suggest_font_size(
            source_text="日",
            target_text="Very long English text here",
            container_width=50,
            current_font_size=18.0,
            min_font_size=10.0
        )

        assert suggested >= 10.0

    def test_suggest_font_size_no_change_needed(self):
        """Should not change when text fits."""
        preserver = LayoutPreserver()

        suggested = preserver.suggest_font_size(
            source_text="Test",
            target_text="Test",
            container_width=200,
            current_font_size=18.0
        )

        assert suggested == 18.0


class TestCreateLayoutPreserver:
    """Test factory function."""

    def test_factory_default(self):
        """Should create preserver with defaults."""
        preserver = create_layout_preserver()
        assert isinstance(preserver, LayoutPreserver)
        assert preserver.strategy == "autofit"

    def test_factory_custom(self):
        """Should create preserver with custom settings."""
        preserver = create_layout_preserver(
            strategy="warn",
            min_font_size_pct=50.0,
            warn_threshold=0.85
        )

        assert preserver.strategy == "warn"
        assert preserver.min_font_size_pct == 0.5
        assert preserver.warn_threshold == 0.85
