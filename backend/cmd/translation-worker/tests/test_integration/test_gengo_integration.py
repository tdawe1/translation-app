# tests/test_integration/test_gengo_integration.py
import pytest
import sys
from pathlib import Path

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from style_guide import (
    parse_gengo_style_guide,
    ParsedStyleGuide,
    build_system_prompt,
    SystemPromptConfig,
)
from audit.style_checker import StyleChecker, create_style_checker


class TestGengoStyleGuideIntegration:
    @pytest.fixture
    def gengo_style_guide_path(self) -> Path:
        return (
            Path(__file__).parent.parent
            / "test_style_guide"
            / "fixtures"
            / "gengo_style_guide.md"
        )

    @pytest.fixture
    def sample_ja_text_path(self) -> Path:
        return Path(__file__).parent / "fixtures" / "sample_ja.txt"

    def test_full_pipeline_parse_to_prompt(self, gengo_style_guide_path):
        if not gengo_style_guide_path.exists():
            pytest.skip("Gengo style guide fixture not found")

        parsed = parse_gengo_style_guide(gengo_style_guide_path)

        assert isinstance(parsed, ParsedStyleGuide)
        assert len(parsed.sections) > 0

        config = SystemPromptConfig(
            include_examples=True,
            include_tone=True,
        )
        prompt = build_system_prompt(parsed, config)

        assert "Japanese" in prompt
        assert "English" in prompt
        assert len(prompt) > 100

    def test_style_checker_with_gengo_rules(self):
        checker = create_style_checker(gengo_rules_enabled=True)

        violations_text = (
            "The colour costs 1,000 dollars at 3:00 PM on 21 September 2025. "
            "We bought apples, oranges and bananas."
        )
        issues = checker.check(violations_text)

        categories = {i.category for i in issues}
        assert "uk_spelling" in categories
        assert "currency_format" in categories
        assert "time_format" in categories
        assert "date_format" in categories
        assert "oxford_comma" in categories

    def test_style_checker_clean_gengo_text(self):
        checker = create_style_checker(gengo_rules_enabled=True)

        clean_text = (
            "The color costs US$1,000 at 3:00 p.m. on September 21, 2025. "
            "We bought apples, oranges, and bananas."
        )
        issues = checker.check(clean_text)

        gengo_categories = {
            "uk_spelling",
            "currency_format",
            "time_format",
            "date_format",
            "oxford_comma",
        }
        gengo_issues = [i for i in issues if i.category in gengo_categories]
        assert len(gengo_issues) == 0

    def test_style_checker_detects_honorifics(self):
        checker = StyleChecker()
        issues = checker.check("Please contact Tanaka-san for more details.")

        honorific_issues = [i for i in issues if i.category == "honorifics"]
        assert len(honorific_issues) == 1

    def test_prompt_builder_includes_style_sections(self, gengo_style_guide_path):
        if not gengo_style_guide_path.exists():
            pytest.skip("Gengo style guide fixture not found")

        parsed = parse_gengo_style_guide(gengo_style_guide_path)

        config = SystemPromptConfig(include_examples=True)
        prompt = build_system_prompt(parsed, config)

        assert len(prompt) > 200

    def test_read_sample_ja_file(self, sample_ja_text_path):
        assert sample_ja_text_path.exists()

        content = sample_ja_text_path.read_text(encoding="utf-8")

        assert "2025年9月21日" in content
        assert "1,000円" in content
        assert "田中様" in content
