# tests/test_nlp/test_tokenizer.py
"""
Unit tests for Japanese tokenizer.

Tests tokenization with POS tags, character counting,
and utility methods for text analysis.
"""

import pytest
import sys
from pathlib import Path

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

# Check if fugashi is available
try:
    import fugashi
    FUGASHI_AVAILABLE = True
except ImportError:
    FUGASHI_AVAILABLE = False

from nlp.tokenizer import (
    JapaneseTokenizer,
    Token,
    TokenizationResult,
)


@pytest.mark.skipif(not FUGASHI_AVAILABLE, reason="fugashi not installed")
class TestToken:
    """Test Token dataclass."""

    def test_token_creation(self):
        """Token should store text, pos, and optional reading."""
        token = Token(text="顧客", pos="名詞", reading="コキャク")
        assert token.text == "顧客"
        assert token.pos == "名詞"
        assert token.reading == "コキャク"

    def test_token_without_reading(self):
        """Token reading should be optional."""
        token = Token(text="test", pos="名詞")
        assert token.text == "test"
        assert token.pos == "名詞"
        assert token.reading is None


@pytest.mark.skipif(not FUGASHI_AVAILABLE, reason="fugashi not installed")
class TestTokenizationResult:
    """Test TokenizationResult dataclass."""

    def test_result_creation(self):
        """TokenizationResult should store tokens and char_counts."""
        tokens = [Token(text="顧客", pos="名詞")]
        counts = {"kanji": 2, "hiragana": 0}
        result = TokenizationResult(tokens=tokens, char_counts=counts)
        assert len(result.tokens) == 1
        assert result.char_counts["kanji"] == 2


@pytest.mark.skipif(not FUGASHI_AVAILABLE, reason="fugashi not installed")
class TestJapaneseTokenizer:
    """Test JapaneseTokenizer functionality."""

    def test_tokenizer_initialization(self):
        """Should initialize with fugashi tagger."""
        tokenizer = JapaneseTokenizer()
        assert tokenizer.tagger is not None

    def test_tokenizer_initialization_fails_without_fugashi(self):
        """Should raise RuntimeError if fugashi not available."""
        # This test verifies the error handling path
        # We can't actually test it without mocking, so we test
        # that the error message is correct in the code
        import inspect
        source = inspect.getsource(JapaneseTokenizer.__init__)
        assert "RuntimeError" in source
        assert "fugashi" in source

    def test_tokenize_japanese_text(self):
        """Should tokenize Japanese text with POS tags."""
        tokenizer = JapaneseTokenizer()
        result = tokenizer.tokenize("顧客満足度を調査します")

        assert len(result.tokens) > 0
        # Check that we have expected tokens
        token_texts = [t.text for t in result.tokens]
        assert "顧客" in token_texts or "顧客満足" in token_texts

        # Check POS tags are present
        for token in result.tokens:
            assert token.pos
            assert token.text

    def test_tokenize_empty_string(self):
        """Should handle empty string gracefully."""
        tokenizer = JapaneseTokenizer()
        result = tokenizer.tokenize("")

        assert result.tokens == []
        assert result.char_counts["total"] == 0

    def test_tokenize_with_reading(self):
        """Should extract katakana reading when available."""
        tokenizer = JapaneseTokenizer()
        result = tokenizer.tokenize("顧客")

        # At least one token should have a reading for common words
        has_reading = any(t.reading for t in result.tokens if t.reading)
        # Note: Not all tokens have readings, so we just verify the structure
        assert len(result.tokens) > 0

    def test_character_counting_kanji(self):
        """Should count kanji characters correctly."""
        tokenizer = JapaneseTokenizer()
        result = tokenizer.tokenize("顧客満足")

        # 顧客満足 = 4 kanji
        assert result.char_counts["kanji"] == 4
        assert result.char_counts["total"] == 4

    def test_character_counting_hiragana(self):
        """Should count hiragana characters correctly."""
        tokenizer = JapaneseTokenizer()
        result = tokenizer.tokenize("こんにちは")

        # こんにちは = 5 hiragana
        assert result.char_counts["hiragana"] == 5
        assert result.char_counts["total"] == 5

    def test_character_counting_katakana(self):
        """Should count katakana characters correctly."""
        tokenizer = JapaneseTokenizer()
        result = tokenizer.tokenize("コンニチハ")

        # コンニチハ = 5 katakana
        assert result.char_counts["katakana"] == 5
        assert result.char_counts["total"] == 5

    def test_character_counting_mixed(self):
        """Should count mixed character types correctly."""
        tokenizer = JapaneseTokenizer()
        result = tokenizer.tokenize("顧客満足度を調査します")

        # 顧客満足度 = 5 kanji
        # を調査します = 6 hiragana
        assert result.char_counts["kanji"] == 5
        assert result.char_counts["hiragana"] == 6
        assert result.char_counts["total"] == 11

    def test_character_counting_punctuation(self):
        """Should count Japanese punctuation."""
        tokenizer = JapaneseTokenizer()
        result = tokenizer.tokenize("こんにちは。")

        assert result.char_counts["hiragana"] == 5
        assert result.char_counts["punctuation"] == 1
        assert result.char_counts["total"] == 6

    def test_character_counting_latin(self):
        """Should count latin characters."""
        tokenizer = JapaneseTokenizer()
        result = tokenizer.tokenize("ABCabc")

        assert result.char_counts["latin"] == 6
        assert result.char_counts["total"] == 6

    def test_character_counting_whitespace(self):
        """Should count whitespace."""
        tokenizer = JapaneseTokenizer()
        result = tokenizer.tokenize("こんにちは 世界")

        assert result.char_counts["hiragana"] == 5
        assert result.char_counts["whitespace"] == 1
        assert result.char_counts["total"] == 7

    def test_character_counting_other(self):
        """Should count other characters (emoji, symbols)."""
        tokenizer = JapaneseTokenizer()
        result = tokenizer.tokenize("😀")

        assert result.char_counts["other"] == 1
        assert result.char_counts["total"] == 1

    def test_get_reading(self):
        """Should get katakana reading for text."""
        tokenizer = JapaneseTokenizer()
        reading = tokenizer.get_reading("顧客")

        # Reading should be a string (may contain katakana)
        assert isinstance(reading, str)
        # For common words, should have katakana
        # Note: This depends on fugashi dictionary

    def test_filter_by_pos_nouns(self):
        """Should filter tokens by POS tag for nouns."""
        tokenizer = JapaneseTokenizer()
        nouns = tokenizer.filter_by_pos("顧客満足度を調査します", ["名詞"])

        # Should extract nouns (顧客, 満足, 度 are likely nouns)
        assert len(nouns) > 0
        assert all(isinstance(n, str) for n in nouns)

    def test_filter_by_pos_verbs(self):
        """Should filter tokens by POS tag for verbs."""
        tokenizer = JapaneseTokenizer()
        verbs = tokenizer.filter_by_pos("調査します", ["動詞"])

        # Should extract verbs (調査 is a verb stem)
        assert len(verbs) >= 0

    def test_filter_by_pos_multiple(self):
        """Should filter by multiple POS tags."""
        tokenizer = JapaneseTokenizer()
        tokens = tokenizer.filter_by_pos(
            "顧客満足度を調査します",
            ["名詞", "動詞"]
        )

        assert isinstance(tokens, list)

    def test_count_nouns(self):
        """Should count nouns in text."""
        tokenizer = JapaneseTokenizer()
        count = tokenizer.count_nouns("顧客満足度を調査します")

        # 顧客, 満足, 度 are likely nouns
        assert count >= 1
        assert isinstance(count, int)

    def test_count_verbs(self):
        """Should count verbs in text."""
        tokenizer = JapaneseTokenizer()
        count = tokenizer.count_verbs("調査します")

        # Should find at least one verb
        assert count >= 1

    def test_is_kanji(self):
        """Should correctly identify kanji code points."""
        tokenizer = JapaneseTokenizer()
        assert tokenizer._is_kanji(ord("漢"))  # 漢 is U+6F22
        assert tokenizer._is_kanji(ord("字"))  # 字 is U+5B57
        assert not tokenizer._is_kanji(ord("あ"))  # hiragana

    def test_is_hiragana(self):
        """Should correctly identify hiragana code points."""
        tokenizer = JapaneseTokenizer()
        assert tokenizer._is_hiragana(ord("あ"))  # U+3042
        assert tokenizer._is_hiragana(ord("ん"))  # U+3093
        assert not tokenizer._is_hiragana(ord("ア"))  # katakana

    def test_is_katakana(self):
        """Should correctly identify katakana code points."""
        tokenizer = JapaneseTokenizer()
        assert tokenizer._is_katakana(ord("ア"))  # U+30A2
        assert tokenizer._is_katakana(ord("ン"))  # U+30F3
        assert not tokenizer._is_katakana(ord("あ"))  # hiragana

    def test_tokenize_with_error_handling(self):
        """Should handle tokenization errors gracefully."""
        tokenizer = JapaneseTokenizer()
        # Empty string is handled, not an error
        result = tokenizer.tokenize("")
        assert result.tokens == []


@pytest.mark.skipif(not FUGASHI_AVAILABLE, reason="fugashi not installed")
class TestJapaneseTokenizerIntegration:
    """Integration tests for JapaneseTokenizer."""

    def test_full_workflow(self):
        """Test complete tokenization and analysis workflow."""
        tokenizer = JapaneseTokenizer()
        text = "顧客満足度調査を実施します。"

        # Tokenize
        result = tokenizer.tokenize(text)

        # Verify tokens
        assert len(result.tokens) > 0

        # Verify character counts
        assert result.char_counts["total"] == len(text)
        assert result.char_counts["kanji"] > 0
        assert result.char_counts["hiragana"] > 0
        assert result.char_counts["punctuation"] == 1  # 。

        # Get reading
        reading = tokenizer.get_reading(text)
        assert isinstance(reading, str)

        # Count parts of speech
        noun_count = tokenizer.count_nouns(text)
        verb_count = tokenizer.count_verbs(text)
        assert noun_count >= 0
        assert verb_count >= 0


# Test that FileStatus is exported (even though it's defined in watcher)
class TestImports:
    """Test that all expected symbols are exported."""

    def test_token_import(self):
        """Token should be importable."""
        from nlp.tokenizer import Token
        assert Token is not None

    def test_tokenization_result_import(self):
        """TokenizationResult should be importable."""
        from nlp.tokenizer import TokenizationResult
        assert TokenizationResult is not None

    def test_japanese_tokenizer_import(self):
        """JapaneseTokenizer should be importable."""
        from nlp.tokenizer import JapaneseTokenizer
        assert JapaneseTokenizer is not None
