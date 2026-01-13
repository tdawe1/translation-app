import pytest

from style_guide.parser import ParsedStyleGuide, StyleSection
from style_guide.prompt_builder import build_system_prompt


def test_build_system_prompt_basic():
    guide = ParsedStyleGuide(
        sections={
            "punctuation": StyleSection(
                name="punctuation",
                description="Use standard English punctuation",
                rules=["Replace Japanese punctuation with English equivalents"],
                examples=[],
            ),
        }
    )

    prompt = build_system_prompt(guide)

    assert "punctuation" in prompt.lower()
    assert "japanese punctuation" in prompt.lower()
    assert "english equivalents" in prompt.lower()


def test_build_system_prompt_includes_all_sections():
    guide = ParsedStyleGuide(
        sections={
            "punctuation": StyleSection(
                name="punctuation", description="Punctuation rules", rules=[]
            ),
            "spelling": StyleSection(
                name="spelling", description="Spelling rules", rules=[]
            ),
            "grammar": StyleSection(
                name="grammar", description="Grammar rules", rules=[]
            ),
        }
    )

    prompt = build_system_prompt(guide)

    assert prompt.count("##") >= 3
