# cmd/translation-worker/nlp/tokenizer.py
"""
Japanese text tokenizer using fugashi (MeCab wrapper).

Provides tokenization with POS (Part of Speech) tags and
character type counting for Japanese text analysis.
"""

from dataclasses import dataclass, field
from typing import Optional, List, Dict
import logging

try:
    import fugashi
except ImportError:
    fugashi = None

logger = logging.getLogger(__name__)


@dataclass
class Token:
    """A Japanese token with POS tag and optional reading.

    Attributes:
        text: The token text surface form
        pos: Part of speech tag (e.g., "名詞", "動詞")
        reading: Katakana reading (if available)
    """
    text: str
    pos: str
    reading: Optional[str] = None


@dataclass
class TokenizationResult:
    """Result of tokenizing Japanese text.

    Attributes:
        tokens: List of tokens with POS tags
        char_counts: Counts by character type (kanji, hiragana, etc.)
    """
    tokens: List[Token]
    char_counts: Dict[str, int] = field(default_factory=dict)


class JapaneseTokenizer:
    """Japanese text tokenizer using fugashi (MeCab wrapper).

    Uses fugashi library to tokenize Japanese text and extract
    Part of Speech tags. Also provides character type counting
    for analysis.

    Example:
        >>> tokenizer = JapaneseTokenizer()
        >>> result = tokenizer.tokenize("顧客満足度を調査します")
        >>> print(result.char_counts)
        {'total': 12, 'kanji': 6, 'hiragana': 4, 'punctuation': 1, 'latin': 0}
    """

    # Unicode ranges for Japanese character types
    HIRAGANA_RANGES = [(0x3040, 0x309F)]
    KATAKANA_RANGES = [(0x30A0, 0x30FF), (0x31F0, 0x31FF)]
    KANJI_RANGES = [(0x4E00, 0x9FFF), (0x3400, 0x4DBF)]

    # Japanese punctuation marks
    JA_PUNCTUATION = set('、。！？「」『』（）【】〈〉《》')

    def __init__(self):
        """Initialize the tokenizer with fugashi tagger.

        Raises:
            RuntimeError: If fugashi/MeCab is not available
        """
        if fugashi is None:
            raise RuntimeError(
                "fugashi is not installed. Install with: pip install fugashi"
            )

        try:
            self.tagger = fugashi.Tagger()
        except Exception as e:
            raise RuntimeError(f"Failed to initialize fugashi/MeCab: {e}")

    def tokenize(self, text: str) -> TokenizationResult:
        """Tokenize Japanese text and return tokens with POS tags.

        Args:
            text: Japanese text to tokenize

        Returns:
            TokenizationResult with tokens and character counts
        """
        if not text:
            return TokenizationResult(tokens=[], char_counts=self._count_characters(""))

        tokens = []

        try:
            for node in self.tagger.parse(text):
                # Extract reading from feature if available
                reading = None
                if hasattr(node, 'feature') and node.feature:
                    # feature is a list like ['名詞', '一般', '*', '*', '*', '*', '顧客', 'カカク', 'カカク']
                    # Reading is typically at index 7 or 8
                    if len(node.feature) > 7:
                        potential_reading = node.feature[7]
                        if potential_reading and potential_reading != '*':
                            reading = potential_reading
                    elif len(node.feature) > 8:
                        potential_reading = node.feature[8]
                        if potential_reading and potential_reading != '*':
                            reading = potential_reading

                tokens.append(Token(
                    text=node.surface,
                    pos=node.pos,
                    reading=reading
                ))
        except Exception as e:
            logger.error(f"Error tokenizing text: {e}")
            # Return partial result with character counts
            return TokenizationResult(
                tokens=[],
                char_counts=self._count_characters(text)
            )

        char_counts = self._count_characters(text)

        return TokenizationResult(tokens=tokens, char_counts=char_counts)

    def _count_characters(self, text: str) -> Dict[str, int]:
        """Count Japanese character types in text.

        Args:
            text: Text to analyze

        Returns:
            Dict with counts for: total, kanji, hiragana, katakana,
            punctuation, whitespace, latin
        """
        counts = {
            "total": len(text),
            "kanji": 0,
            "hiragana": 0,
            "katakana": 0,
            "punctuation": 0,
            "whitespace": 0,
            "latin": 0,
            "other": 0
        }

        for char in text:
            code = ord(char)

            if char.isspace():
                counts["whitespace"] += 1
            elif self._is_kanji(code):
                counts["kanji"] += 1
            elif self._is_hiragana(code):
                counts["hiragana"] += 1
            elif self._is_katakana(code):
                counts["katakana"] += 1
            elif char in self.JA_PUNCTUATION:
                counts["punctuation"] += 1
            elif char.isalpha() and code < 0x0100:  # Basic Latin
                counts["latin"] += 1
            else:
                counts["other"] += 1

        return counts

    def _is_kanji(self, code: int) -> bool:
        """Check if code point is Kanji."""
        return (0x4E00 <= code <= 0x9FFF) or (0x3400 <= code <= 0x4DBF)

    def _is_hiragana(self, code: int) -> bool:
        """Check if code point is Hiragana."""
        return 0x3040 <= code <= 0x309F

    def _is_katakana(self, code: int) -> bool:
        """Check if code point is Katakana."""
        return (0x30A0 <= code <= 0x30FF) or (0x31F0 <= code <= 0x31FF)

    def get_reading(self, text: str) -> str:
        """Get the katakana reading for Japanese text.

        Useful for sorting and searching where pronunciation matters.

        Args:
            text: Japanese text

        Returns:
            Katakana reading string
        """
        result = self.tokenize(text)
        readings = []

        for token in result.tokens:
            if token.reading:
                readings.append(token.reading)
            else:
                # Fallback to using the surface form for non-Japanese
                readings.append(token.text)

        return ''.join(readings)

    def filter_by_pos(self, text: str, pos_tags: List[str]) -> List[str]:
        """Filter tokens by part of speech tags.

        Args:
            text: Japanese text to tokenize and filter
            pos_tags: List of POS tags to include (e.g., ["名詞", "動詞"])

        Returns:
            List of token texts matching the POS tags
        """
        result = self.tokenize(text)
        return [
            token.text
            for token in result.tokens
            if token.pos in pos_tags
        ]

    def count_nouns(self, text: str) -> int:
        """Count nouns in text.

        Args:
            text: Japanese text

        Returns:
            Number of nouns (名詞) found
        """
        result = self.tokenize(text)
        return sum(1 for t in result.tokens if t.pos.startswith("名詞"))

    def count_verbs(self, text: str) -> int:
        """Count verbs in text.

        Args:
            text: Japanese text

        Returns:
            Number of verbs (動詞) found
        """
        result = self.tokenize(text)
        return sum(1 for t in result.tokens if t.pos.startswith("動詞"))
