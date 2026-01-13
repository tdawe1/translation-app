import pytest
from pathlib import Path

from style_guide.parser import parse_gengo_style_guide


def test_parse_punctuation_rules():
    guide_path = Path(__file__).parent / "fixtures" / "gengo_style_guide.md"
    rules = parse_gengo_style_guide(guide_path)

    assert "punctuation" in rules.sections
    assert (
        "standard english punctuation"
        in rules.sections["punctuation"].description.lower()
    )
    assert any(
        "japanese punctuation" in rule.lower()
        for rule in rules.sections["punctuation"].rules
    )


def test_parse_spelling_rules():
    guide_path = Path(__file__).parent / "fixtures" / "gengo_style_guide.md"
    rules = parse_gengo_style_guide(guide_path)

    assert "spelling" in rules.sections
    assert "us english" in rules.sections["spelling"].description.lower()
    assert any(
        "color" in rule.lower() or "spelling" in rule.lower()
        for rule in rules.sections["spelling"].rules
    )


def test_parse_grammar_rules():
    guide_path = Path(__file__).parent / "fixtures" / "gengo_style_guide.md"
    rules = parse_gengo_style_guide(guide_path)

    assert "grammar" in rules.sections
    assert len(rules.sections["grammar"].rules) > 0
