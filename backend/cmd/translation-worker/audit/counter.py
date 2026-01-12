# audit/counter.py
"""
Japanese character counter for billing and estimation.

Counts Japanese text by character type:
- Kanji (CJK Unified Ideographs)
- Hiragana
- Katakana (including half-width and extended)
- Punctuation
- Whitespace
- Latin letters

Also estimates English character count for billing purposes
using typical JA→EN expansion ratios.
"""

from dataclasses import dataclass, field
from typing import Dict, Optional


@dataclass
class CharacterCount:
    """Japanese character count breakdown.

    Attributes:
        total: Total character count
        kanji: Number of CJK Unified Ideographs
        hiragana: Number of hiragana characters
        katakana: Number of katakana characters (including extensions)
        punctuation: Number of Japanese punctuation marks
        whitespace: Number of whitespace characters
        latin: Number of Latin alphabet characters
        estimated_english: Estimated English character count
    """

    total: int
    kanji: int = 0
    hiragana: int = 0
    katakana: int = 0
    punctuation: int = 0
    whitespace: int = 0
    latin: int = 0
    estimated_english: int = 0

    def to_dict(self) -> Dict[str, int]:
        """Convert to dictionary for JSON serialization."""
        return {
            "total": self.total,
            "kanji": self.kanji,
            "hiragana": self.hiragana,
            "katakana": self.katakana,
            "punctuation": self.punctuation,
            "whitespace": self.whitespace,
            "latin": self.latin,
            "estimated_english": self.estimated_english,
        }


class JapaneseCharacterCounter:
    """Counts Japanese text for billing/estimation.

    Uses Unicode code point ranges to identify character types:
    - Kanji: U+4E00-U+9FFF (CJK Unified Ideographs)
            U+3400-U+4DBF (CJK Unified Ideographs Extension A)
    - Hiragana: U+3040-U+309F
    - Katakana: U+30A0-U+30FF (basic)
               U+31F0-U+31FF (Katakana Phonetic Extensions)
    - Punctuation: Japanese-specific marks
    - Latin: A-Z, a-z (detected via isalpha())

    Example:
        >>> counter = JapaneseCharacterCounter()
        >>> result = counter.count("顧客満足度は高いです")
        >>> result["kanji"]
        5
        >>> result["hiragana"]
        4
    """

    # Japanese punctuation marks to count separately
    JA_PUNCTUATION = set('、。！？「」『』（）【】〈〉《》・：；')

    # Unicode ranges for Japanese text
    # Kanji: CJK Unified Ideographs + Extension A
    KANJI_START = 0x4E00
    KANJI_END = 0x9FFF
    KANJI_EXT_A_START = 0x3400
    KANJI_EXT_A_END = 0x4DBF

    # Hiragana
    HIRAGANA_START = 0x3040
    HIRAGANA_END = 0x309F

    # Katakana (basic + phonetic extensions)
    KATAKANA_START = 0x30A0
    KATAKANA_END = 0x30FF
    KATAKANA_EXT_START = 0x31F0
    KATAKANA_EXT_END = 0x31FF

    # Expansion ratios for English estimation (based on typical JA→EN)
    EXPANSION_RATIO_KANJI = 2.0
    EXPANSION_RATIO_HIRAGANA = 1.5
    EXPANSION_RATIO_KATAKANA = 1.5
    EXPANSION_RATIO_PUNCTUATION = 1.0
    EXPANSION_RATIO_LATIN = 1.0

    def __init__(
        self,
        expansion_ratios: Optional[Dict[str, float]] = None,
    ):
        """Initialize the character counter.

        Args:
            expansion_ratios: Optional custom expansion ratios for English estimation
        """
        if expansion_ratios:
            self.expansion_ratios = expansion_ratios
        else:
            self.expansion_ratios = {
                "kanji": self.EXPANSION_RATIO_KANJI,
                "hiragana": self.EXPANSION_RATIO_HIRAGANA,
                "katakana": self.EXPANSION_RATIO_KATAKANA,
                "punctuation": self.EXPANSION_RATIO_PUNCTUATION,
                "latin": self.EXPANSION_RATIO_LATIN,
            }

    def count(self, text: str) -> Dict[str, int]:
        """Count Japanese text characters by type.

        Args:
            text: The Japanese text to count

        Returns:
            Dictionary with counts for each character type plus
            estimated English character count
        """
        counts = {
            "total": len(text),
            "kanji": 0,
            "hiragana": 0,
            "katakana": 0,
            "punctuation": 0,
            "whitespace": 0,
            "latin": 0,
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
            elif char.isalpha():
                counts["latin"] += 1

        # Estimate English length
        counts["estimated_english"] = self._estimate_english(counts)

        return counts

    def count_as_dataclass(self, text: str) -> CharacterCount:
        """Count Japanese text and return as CharacterCount dataclass.

        Args:
            text: The Japanese text to count

        Returns:
            CharacterCount dataclass with detailed breakdown
        """
        counts = self.count(text)
        return CharacterCount(
            total=counts["total"],
            kanji=counts["kanji"],
            hiragana=counts["hiragana"],
            katakana=counts["katakana"],
            punctuation=counts["punctuation"],
            whitespace=counts["whitespace"],
            latin=counts["latin"],
            estimated_english=counts["estimated_english"],
        )

    def _is_kanji(self, code: int) -> bool:
        """Check if code point is kanji."""
        return (
            self.KANJI_START <= code <= self.KANJI_END or
            self.KANJI_EXT_A_START <= code <= self.KANJI_EXT_A_END
        )

    def _is_hiragana(self, code: int) -> bool:
        """Check if code point is hiragana."""
        return self.HIRAGANA_START <= code <= self.HIRAGANA_END

    def _is_katakana(self, code: int) -> bool:
        """Check if code point is katakana."""
        return (
            self.KATAKANA_START <= code <= self.KATAKANA_END or
            self.KATAKANA_EXT_START <= code <= self.KATAKANA_EXT_END
        )

    def _estimate_english(self, counts: Dict[str, int]) -> int:
        """Estimate English character count from Japanese.

        Uses typical JA→EN expansion ratios:
        - Kanji: ~2x expansion (one character → ~2 English words)
        - Hiragana: ~1.5x expansion (grammatical particles)
        - Katakana: ~1.5x expansion (loanwords)
        - Punctuation/Latin: 1x (direct mapping)

        Args:
            counts: Character count dictionary

        Returns:
            Estimated English character count
        """
        estimated = (
            counts["kanji"] * self.expansion_ratios["kanji"] +
            counts["hiragana"] * self.expansion_ratios["hiragana"] +
            counts["katakana"] * self.expansion_ratios["katakana"] +
            counts["punctuation"] * self.expansion_ratios["punctuation"] +
            counts["latin"] * self.expansion_ratios["latin"]
        )
        return int(estimated)


def create_counter(
    expansion_ratios: Optional[Dict[str, float]] = None,
) -> JapaneseCharacterCounter:
    """Factory function to create a configured character counter.

    Args:
        expansion_ratios: Optional custom expansion ratios

    Returns:
        Configured JapaneseCharacterCounter instance
    """
    return JapaneseCharacterCounter(expansion_ratios=expansion_ratios)
