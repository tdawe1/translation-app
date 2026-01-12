# tests/test_audit/test_counter.py
"""
Tests for JapaneseCharacterCounter.

Tests character counting by type:
- Kanji
- Hiragana
- Katakana
- Punctuation
- Whitespace
- Latin letters
- English length estimation
"""

import pytest
import sys
from pathlib import Path

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from audit.counter import (
    CharacterCount,
    JapaneseCharacterCounter,
    create_counter,
)


class TestCharacterCount:
    """Test CharacterCount dataclass."""

    def test_creation_default(self):
        """Should create dataclass with defaults."""
        count = CharacterCount(total=10)

        assert count.total == 10
        assert count.kanji == 0
        assert count.hiragana == 0
        assert count.katakana == 0
        assert count.punctuation == 0
        assert count.whitespace == 0
        assert count.latin == 0
        assert count.estimated_english == 0

    def test_creation_with_values(self):
        """Should create dataclass with all values."""
        count = CharacterCount(
            total=20,
            kanji=5,
            hiragana=4,
            katakana=3,
            punctuation=2,
            whitespace=4,
            latin=2,
            estimated_english=18,
        )

        assert count.total == 20
        assert count.kanji == 5
        assert count.hiragana == 4
        assert count.katakana == 3
        assert count.punctuation == 2
        assert count.whitespace == 4
        assert count.latin == 2
        assert count.estimated_english == 18

    def test_to_dict(self):
        """Should convert to dictionary for JSON."""
        count = CharacterCount(
            total=10,
            kanji=5,
            hiragana=3,
        )

        result = count.to_dict()

        assert result["total"] == 10
        assert result["kanji"] == 5
        assert result["hiragana"] == 3


class TestJapaneseCharacterCounter:
    """Test JapaneseCharacterCounter."""

    def test_initialization_default(self):
        """Should initialize with default expansion ratios."""
        counter = JapaneseCharacterCounter()

        assert counter.expansion_ratios["kanji"] == 2.0
        assert counter.expansion_ratios["hiragana"] == 1.5
        assert counter.expansion_ratios["katakana"] == 1.5
        assert counter.expansion_ratios["punctuation"] == 1.0
        assert counter.expansion_ratios["latin"] == 1.0

    def test_initialization_custom_ratios(self):
        """Should accept custom expansion ratios."""
        custom_ratios = {
            "kanji": 1.8,
            "hiragana": 1.3,
        }
        counter = JapaneseCharacterCounter(expansion_ratios=custom_ratios)

        assert counter.expansion_ratios["kanji"] == 1.8
        assert counter.expansion_ratios["hiragana"] == 1.3

    def test_count_empty_string(self):
        """Should handle empty string."""
        counter = JapaneseCharacterCounter()
        result = counter.count("")

        assert result["total"] == 0
        assert result["kanji"] == 0
        assert result["hiragana"] == 0

    def test_count_kanji_only(self):
        """Should count kanji characters."""
        counter = JapaneseCharacterCounter()
        result = counter.count("日本語")

        assert result["kanji"] == 3
        assert result["hiragana"] == 0
        assert result["katakana"] == 0

    def test_count_hiragana_only(self):
        """Should count hiragana characters."""
        counter = JapaneseCharacterCounter()
        result = counter.count("こんにちは")

        assert result["kanji"] == 0
        assert result["hiragana"] == 5
        assert result["katakana"] == 0

    def test_count_katakana_only(self):
        """Should count katakana characters."""
        counter = JapaneseCharacterCounter()
        result = counter.count("コンニチハ")

        assert result["kanji"] == 0
        assert result["hiragana"] == 0
        assert result["katakana"] == 5

    def test_count_mixed_japanese(self):
        """Should count mixed Japanese text."""
        counter = JapaneseCharacterCounter()
        result = counter.count("顧客満足度は高いです")

        # 顧客満足度 = 5 kanji
        # は = 1 hiragana
        # 高い = 1 kanji + 1 hiragana
        # です = 2 hiragana
        assert result["total"] == 10
        assert result["kanji"] == 6
        assert result["hiragana"] == 4
        assert result["katakana"] == 0

    def test_count_punctuation(self):
        """Should count Japanese punctuation."""
        counter = JapaneseCharacterCounter()
        result = counter.count("、。！？「」")

        assert result["punctuation"] == 6
        assert result["kanji"] == 0
        assert result["hiragana"] == 0

    def test_count_whitespace(self):
        """Should count whitespace."""
        counter = JapaneseCharacterCounter()
        result = counter.count("こんにちは 　世界\n")

        # Space + ideographic space + newline
        assert result["whitespace"] == 3
        assert result["hiragana"] == 5  # こんにちは

    def test_count_latin_letters(self):
        """Should count Latin alphabet characters."""
        counter = JapaneseCharacterCounter()
        result = counter.count("Hello World")

        assert result["latin"] == 10  # H,e,l,l,o,W,o,r,l,d
        assert result["kanji"] == 0

    def test_count_mixed_text(self):
        """Should count mixed Japanese and Latin text."""
        counter = JapaneseCharacterCounter()
        result = counter.count("顧客満足度は95%です。")

        # 顧客満足度 = 5 kanji
        # は = 1 hiragana
        # 95 = digits (not counted as latin - isalpha() returns False for digits)
        # % = ASCII punctuation (not in JA_PUNCTUATION, only counted in total)
        # です = 2 hiragana
        # 。 = punctuation
        assert result["total"] == 12
        assert result["kanji"] == 5
        assert result["hiragana"] == 3
        assert result["punctuation"] == 1  # only 。 (ASCII % not in JA_PUNCTUATION)
        assert result["latin"] == 0  # digits are not counted as latin

    def test_count_mixed_with_latin(self):
        """Should count mixed Japanese and Latin text."""
        counter = JapaneseCharacterCounter()
        result = counter.count("顧客満足度はHighです。")

        # 顧客満足度 = 5 kanji
        # は = 1 hiragana
        # High = 4 latin letters
        # です = 2 hiragana
        # 。 = punctuation
        assert result["total"] == 13
        assert result["kanji"] == 5
        assert result["hiragana"] == 3
        assert result["punctuation"] == 1
        assert result["latin"] == 4  # "High"

    def test_estimate_english_kanji_only(self):
        """Should estimate English from kanji text."""
        counter = JapaneseCharacterCounter()
        # "日本語" - all three are kanji (語 is U+8A9E, a CJK ideograph)
        result = counter.count("日本語")

        # 3 kanji * 2.0 expansion = 6
        assert result["kanji"] == 3
        assert result["estimated_english"] == 6

    def test_estimate_english_hiragana_only(self):
        """Should estimate English from hiragana text."""
        counter = JapaneseCharacterCounter()
        result = counter.count("こんにちは")

        # 5 hiragana * 1.5 expansion = 7.5 -> 7
        assert result["estimated_english"] == 7

    def test_estimate_english_katakana_only(self):
        """Should estimate English from katakana text."""
        counter = JapaneseCharacterCounter()
        # "コンピュータ" - コ(30B3) ン(30F3) ピ(30D4) ュ(30E5) ー(30FC) タ(30BF)
        # The dash (ー) is U+30FC, within katakana range
        result = counter.count("コンピュータ")

        # 6 katakana * 1.5 expansion = 9
        assert result["katakana"] == 6
        assert result["estimated_english"] == 9

    def test_estimate_english_mixed(self):
        """Should estimate English from mixed text."""
        counter = JapaneseCharacterCounter()
        result = counter.count("顧客満足度は高いです")

        # 顧客満足度高い = 7 kanji (高い has 2 kanji)
        # はです = 3 hiragana
        # 7 * 2.0 + 3 * 1.5 = 14 + 4.5 = 18.5 -> 18
        assert result["estimated_english"] == 18

    def test_count_as_dataclass(self):
        """Should return CharacterCount dataclass."""
        counter = JapaneseCharacterCounter()
        # "日本語です" - 日本語 are kanji, です is hiragana
        result = counter.count_as_dataclass("日本語です")

        assert isinstance(result, CharacterCount)
        assert result.total == 5
        assert result.kanji == 3  # 日本語 are all kanji
        assert result.hiragana == 2  # です

    def test_extended_kanji_range(self):
        """Should count CJK Extension A characters."""
        counter = JapaneseCharacterCounter()
        # U+3400 is in CJK Extension A
        result = counter.count("\u3400")

        assert result["kanji"] == 1

    def test_extended_katakana_range(self):
        """Should count Katakana Phonetic Extensions."""
        counter = JapaneseCharacterCounter()
        # U+31F0 is Katakana Phonetic Extension
        result = counter.count("\u31F0")

        assert result["katakana"] == 1


class TestFactoryFunction:
    """Test factory function."""

    def test_create_counter_default(self):
        """Should create counter with defaults."""
        counter = create_counter()

        assert isinstance(counter, JapaneseCharacterCounter)
        assert counter.expansion_ratios["kanji"] == 2.0

    def test_create_counter_custom_ratios(self):
        """Should create counter with custom ratios."""
        counter = create_counter(expansion_ratios={"kanji": 1.5})

        assert counter.expansion_ratios["kanji"] == 1.5


class TestCharacterTypeDetection:
    """Test internal character type detection methods."""

    def test_is_kanji_basic_range(self):
        """Should detect basic kanji range."""
        counter = JapaneseCharacterCounter()

        # U+4E00 is first CJK Unified Ideograph
        assert counter._is_kanji(0x4E00) is True
        # U+9FFF is last CJK Unified Ideograph
        assert counter._is_kanji(0x9FFF) is True

    def test_is_kanji_extension_a(self):
        """Should detect CJK Extension A kanji."""
        counter = JapaneseCharacterCounter()

        # U+3400 is first in Extension A
        assert counter._is_kanji(0x3400) is True
        # U+4DBF is last in Extension A
        assert counter._is_kanji(0x4DBF) is True

    def test_is_hiragana_range(self):
        """Should detect hiragana range."""
        counter = JapaneseCharacterCounter()

        # U+3040 is first hiragana
        assert counter._is_hiragana(0x3040) is True
        # U+309F is last hiragana
        assert counter._is_hiragana(0x309F) is True

    def test_is_katakana_basic_range(self):
        """Should detect basic katakana range."""
        counter = JapaneseCharacterCounter()

        # U+30A0 is first katakana
        assert counter._is_katakana(0x30A0) is True
        # U+30FF is last basic katakana
        assert counter._is_katakana(0x30FF) is True

    def test_is_katakana_extension(self):
        """Should detect katakana phonetic extensions."""
        counter = JapaneseCharacterCounter()

        # U+31F0 is first Katakana Phonetic Extension
        assert counter._is_katakana(0x31F0) is True
        # U+31FF is last Katakana Phonetic Extension
        assert counter._is_katakana(0x31FF) is True

    def test_non_kanji_returns_false(self):
        """Should return false for non-kanji characters."""
        counter = JapaneseCharacterCounter()

        assert counter._is_kanji(0x0041) is False  # 'A'
        assert counter._is_kanji(0x3040) is False  # Hiragana

    def test_non_hiragana_returns_false(self):
        """Should return false for non-hiragana characters."""
        counter = JapaneseCharacterCounter()

        assert counter._is_hiragana(0x0041) is False  # 'A'
        assert counter._is_hiragana(0x4E00) is False  # Kanji

    def test_non_katakana_returns_false(self):
        """Should return false for non-katakana characters."""
        counter = JapaneseCharacterCounter()

        assert counter._is_katakana(0x0041) is False  # 'A'
        assert counter._is_katakana(0x4E00) is False  # Kanji


class TestEdgeCases:
    """Test edge cases and special characters."""

    def test_half_width_katakana(self):
        """Should handle half-width katakana."""
        counter = JapaneseCharacterCounter()
        # Half-width katakana are in different range
        # They're typically counted as Latin/symbols
        result = counter.count("ｶｬｯ")  # Half-width katakana

        # Half-width katakana (U+FF65-FF9F) not in our ranges
        # so they'll be counted as non-Japanese or latin if isalpha matches
        # In this case they're not Latin letters, so they fall through
        assert result["katakana"] == 0

    def test_numbers_not_counted(self):
        """Should not count numbers as Japanese or Latin."""
        counter = JapaneseCharacterCounter()
        result = counter.count("12345")

        # Numbers are not counted in any category except total
        assert result["total"] == 5
        assert result["latin"] == 0
        assert result["kanji"] == 0

    def test_symbols_not_counted(self):
        """Should not count symbols in Japanese categories."""
        counter = JapaneseCharacterCounter()
        result = counter.count("@#$%")

        # Symbols only count toward total
        assert result["total"] == 4
        assert result["punctuation"] == 0
        assert result["latin"] == 0

    def test_newlines_as_whitespace(self):
        """Should count newlines as whitespace."""
        counter = JapaneseCharacterCounter()
        result = counter.count("a\nb\nc")

        assert result["whitespace"] == 2  # 2 newlines
        assert result["latin"] == 3

    def test_tabs_as_whitespace(self):
        """Should count tabs as whitespace."""
        counter = JapaneseCharacterCounter()
        result = counter.count("a\tb\tc")

        assert result["whitespace"] == 2  # 2 tabs
        assert result["latin"] == 3

    def test_fullwidth_punctuation(self):
        """Should count fullwidth punctuation marks."""
        counter = JapaneseCharacterCounter()
        result = counter.count("。【】")

        # Fullwidth punctuation in JA_PUNCTUATION set
        assert result["punctuation"] == 3

    def test_combining_characters(self):
        """Should handle combining characters."""
        counter = JapaneseCharacterCounter()
        # Combining characters like dakuten/handakuten
        # が = U+304B + U+3099 (hiragana ga)
        result = counter.count("が")

        # The combining mark gets counted as hiragana due to range
        assert result["hiragana"] >= 1
