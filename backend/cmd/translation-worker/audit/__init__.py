# audit/__init__.py
"""
JP audit tools for translation quality assessment.

Provides utilities for:
- Japanese character counting (kanji, hiragana, katakana breakdown)
- Style compliance checking for JA→EN translations
- English length estimation from Japanese source
"""

from .counter import (
    CharacterCount,
    JapaneseCharacterCounter,
    create_counter,
)

from .style_checker import (
    StyleIssue,
    StyleChecker,
    create_style_checker,
)

__all__ = [
    # Counter
    "CharacterCount",
    "JapaneseCharacterCounter",
    "create_counter",
    # Style checker
    "StyleIssue",
    "StyleChecker",
    "create_style_checker",
]
