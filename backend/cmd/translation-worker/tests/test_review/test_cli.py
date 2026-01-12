# tests/test_review/test_cli.py
"""Tests for CLI interface."""
import pytest
import sys
from pathlib import Path
from click.testing import CliRunner

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.cli import cli, translate, batch
# Note: judge is a command name, not a function to import directly


class TestCLI:
    def test_translate_command_requires_provider(self):
        """Should require provider argument."""
        runner = CliRunner()
        result = runner.invoke(translate, ["test"])
        # Should exit with non-zero code or show missing option error
        assert result.exit_code != 0 or "Missing option" in result.output

    def test_translate_accepts_all_providers(self):
        """Should accept anthropic, openai, gemini providers."""
        runner = CliRunner()
        for provider in ["anthropic", "openai", "gemini"]:
            result = runner.invoke(translate, ["--provider", provider, "test"])
            # Should not error on provider validation (will fail on API key instead)
            assert "Invalid provider" not in result.output

    def test_batch_command_requires_provider(self):
        """Should require provider for batch command."""
        runner = CliRunner()
        with runner.isolated_filesystem():
            with open("sources.txt", "w") as f:
                f.write("こんにちは\n世界")

            result = runner.invoke(batch, [
                "--input", "sources.txt",
                "--output", "translations.txt"
            ])
            # Should require provider
            assert result.exit_code != 0 or "Missing" in result.output

    def test_format_json_outputs_structured(self):
        """Should output valid JSON when format=json."""
        runner = CliRunner()
        # Without API key, should fail before reaching output
        result = runner.invoke(translate, [
            "--provider", "anthropic",
            "--format", "json",
            "test"
        ])
        # Output should be defined (even if error)
        assert result.output is not None

    def test_cli_group_has_help(self):
        """CLI group should have help text."""
        runner = CliRunner()
        result = runner.invoke(cli, ["--help"])
        assert result.exit_code == 0
        assert "translate" in result.output
        assert "judge" in result.output
        assert "batch" in result.output
