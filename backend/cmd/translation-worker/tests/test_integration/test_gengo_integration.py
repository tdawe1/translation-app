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


class TestGengoWorkflowIntegration:
    """Integration tests: style guide → workflow → style checker → metrics."""

    @pytest.fixture
    def gengo_style_guide_path(self) -> Path:
        return (
            Path(__file__).parent.parent
            / "test_style_guide"
            / "fixtures"
            / "gengo_style_guide.md"
        )

    def test_workflow_with_gengo_style_checker_flags_violations(self):
        """Full pipeline: mock provider returns violating text, workflow flags it."""
        from unittest.mock import Mock
        from review.workflow import TranslationWorkflow
        from review.multimodel import MultiModelTranslator
        from review.judge import TranslationJudge
        from review.flagging import FlaggingEngine

        mock_provider = Mock()
        mock_provider.generate.return_value = Mock(
            text=(
                "The colour costs 1,000 dollars at 3:00 PM on 21 September 2025. "
                "We bought apples, oranges and bananas."
            )
        )

        checker = create_style_checker(gengo_rules_enabled=True)
        translator = MultiModelTranslator(providers=[mock_provider])
        judge = TranslationJudge(enabled=False)

        workflow = TranslationWorkflow(
            translator=translator,
            judge=judge,
            style_checker=checker,
            flagger=FlaggingEngine(random_sample_rate=0.0),
        )

        job = workflow.create_job(
            source_file="input.txt",
            target_file="output.txt",
            segments=[{"source": "テスト文章"}],
        )
        processed = workflow.process_job(job)

        seg = processed.segments[0]
        assert seg.is_flagged is True
        categories = {i["category"] for i in seg.style_issues}
        assert "uk_spelling" in categories
        assert "currency_format" in categories
        assert "time_format" in categories
        assert "date_format" in categories
        assert "oxford_comma" in categories

        # Verify metrics
        m = workflow.last_metrics
        assert m is not None
        assert m.style_violation_count == 1
        assert m.flag_rate == 1.0
        assert len(m.style_violations_by_category) >= 5

    def test_workflow_with_gengo_clean_text_passes(self):
        """Clean Gengo text should pass through without style flags."""
        from unittest.mock import Mock
        from review.workflow import TranslationWorkflow
        from review.multimodel import MultiModelTranslator
        from review.judge import TranslationJudge
        from review.flagging import FlaggingEngine

        mock_provider = Mock()
        mock_provider.generate.return_value = Mock(
            text=(
                "The color costs US$1,000 at 3:00 p.m. on September 21, 2025. "
                "We bought apples, oranges, and bananas."
            )
        )

        checker = create_style_checker(gengo_rules_enabled=True)
        translator = MultiModelTranslator(providers=[mock_provider])
        judge = TranslationJudge(enabled=False)

        workflow = TranslationWorkflow(
            translator=translator,
            judge=judge,
            style_checker=checker,
            flagger=FlaggingEngine(random_sample_rate=0.0),
        )

        job = workflow.create_job(
            source_file="input.txt",
            target_file="output.txt",
            segments=[{"source": "テスト文章"}],
        )
        processed = workflow.process_job(job)

        seg = processed.segments[0]
        gengo_categories = {"uk_spelling", "currency_format", "time_format", "date_format", "oxford_comma"}
        gengo_issues = [i for i in seg.style_issues if i.get("category") in gengo_categories]
        assert len(gengo_issues) == 0

    def test_style_guide_prompt_reaches_provider(self, gengo_style_guide_path):
        """The style guide prompt should be passed through to the provider config."""
        if not gengo_style_guide_path.exists():
            pytest.skip("Gengo style guide fixture not found")

        parsed = parse_gengo_style_guide(gengo_style_guide_path)
        prompt = build_system_prompt(parsed)

        from review.llm import get_provider
        from unittest.mock import patch, MagicMock

        with patch("review.llm.providers.OpenAIProvider") as MockOpenAI:
            mock_instance = MagicMock()
            MockOpenAI.return_value = mock_instance

            provider = get_provider(
                "openai",
                api_key="sk-test",
                model="gpt-5.2",
                system_prompt=prompt,
            )

            # The provider should have received the system_prompt
            MockOpenAI.assert_called_once_with(
                api_key="sk-test",
                model="gpt-5.2",
                system_prompt=prompt,
            )
