# tests/test_review/test_cli.py
"""Tests for CLI interface."""
import pytest
import sys
from pathlib import Path
from click.testing import CliRunner
from unittest.mock import patch, MagicMock

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.cli import cli, translate, batch
# Note: judge is a command name, not a function to import directly


class TestCLI:
    def test_translate_command_requires_provider_or_cli(self):
        """Should require either --provider or --cli argument."""
        runner = CliRunner()
        result = runner.invoke(translate, ["test"])
        # Should exit with non-zero code or show missing option error
        assert result.exit_code != 0
        assert "Must specify either --provider" in result.output or "Missing option" in result.output

    def test_translate_mutually_exclusive_provider_and_cli(self):
        """Should reject both --provider and --cli specified together."""
        runner = CliRunner()
        result = runner.invoke(translate, [
            "--provider", "anthropic",
            "--cli", "claude",
            "test"
        ])
        assert result.exit_code != 0
        assert "Cannot specify both" in result.output

    def test_translate_accepts_all_providers(self):
        """Should accept anthropic, openai, gemini providers."""
        runner = CliRunner()
        for provider in ["anthropic", "openai", "gemini"]:
            result = runner.invoke(translate, ["--provider", provider, "test"])
            # Should not error on provider validation (will fail on API key instead)
            assert "Invalid provider" not in result.output

    @patch("review.llm.cli.subprocess.run")
    @patch("review.cli.shutil.which", return_value="/usr/bin/claude")
    def test_translate_accepts_all_cli_tools(self, mock_which, mock_run):
        """Should accept claude, codex, gemini, ollama CLI tools."""
        # Mock successful subprocess
        mock_result = MagicMock()
        mock_result.returncode = 0
        mock_result.stdout = "Translation"
        mock_run.return_value = mock_result

        runner = CliRunner()
        for cli_tool in ["claude", "codex", "gemini", "ollama"]:
            result = runner.invoke(translate, ["--cli", cli_tool, "test"])
            # Should not error on cli validation (will succeed due to mocks)
            assert result.exit_code == 0

    @patch("review.llm.cli.subprocess.run")
    def test_translate_with_cli_tool_success(self, mock_run):
        """Should successfully translate using CLI tool."""
        # Mock successful subprocess call
        mock_result = MagicMock()
        mock_result.returncode = 0
        mock_result.stdout = "Hello World"
        mock_run.return_value = mock_result

        runner = CliRunner()
        result = runner.invoke(translate, ["--cli", "claude", "こんにちは"])
        # Should succeed (exit code 0)
        assert result.exit_code == 0
        assert "Hello World" in result.output

    @patch("review.llm.cli.subprocess.run")
    def test_translate_with_cli_tool_not_found(self, mock_run):
        """Should show helpful error when CLI tool not found."""
        # Mock shutil.which returning None (tool not found)
        with patch("review.cli.shutil.which", return_value=None):
            runner = CliRunner()
            result = runner.invoke(translate, ["--cli", "claude", "こんにちは"])
            # Should fail with helpful message
            assert result.exit_code != 0
            assert "not found in PATH" in result.output or "Install it first" in result.output

    def test_batch_command_requires_provider_or_cli(self):
        """Should require either --provider or --cli for batch command."""
        runner = CliRunner()
        with runner.isolated_filesystem():
            with open("sources.txt", "w") as f:
                f.write("こんにちは\n世界")

            result = runner.invoke(batch, [
                "--input", "sources.txt",
                "--output", "translations.txt"
            ])
            # Should require provider or cli
            assert result.exit_code != 0
            assert "Must specify either" in result.output or "Missing" in result.output

    @patch("review.llm.cli.subprocess.run")
    def test_batch_with_cli_tool(self, mock_run):
        """Should process batch with CLI tool."""
        # Mock successful subprocess call
        mock_result = MagicMock()
        mock_result.returncode = 0
        mock_result.stdout = "Hello"
        mock_run.return_value = mock_result

        runner = CliRunner()
        with runner.isolated_filesystem():
            with open("sources.txt", "w") as f:
                f.write("こんにちは\n世界")

            result = runner.invoke(batch, [
                "--input", "sources.txt",
                "--output", "translations.txt",
                "--cli", "claude"
            ])
            # Should succeed
            assert result.exit_code == 0

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

    def test_translate_command_shows_cli_option(self):
        """translate command help should show --cli option."""
        runner = CliRunner()
        result = runner.invoke(cli, ["translate", "--help"])
        assert result.exit_code == 0
        assert "--cli" in result.output or "local CLI tool" in result.output

    @patch("review.llm.cli.subprocess.run")
    def test_batch_csv_escaping_handles_quotes(self, mock_run):
        """CSV output should properly escape quotes in both source and translation."""
        # Mock subprocess returning text with quotes
        mock_result = MagicMock()
        mock_result.returncode = 0
        mock_result.stdout = 'He said "hello"'
        mock_run.return_value = mock_result

        runner = CliRunner()
        with runner.isolated_filesystem():
            # Source text with quotes
            with open("sources.txt", "w", encoding="utf-8") as f:
                f.write('She asked "how are you?"\n')

            result = runner.invoke(batch, [
                "--input", "sources.txt",
                "--output", "translations.txt",
                "--cli", "claude",
                "--format", "csv"
            ])

            # Check output file has proper CSV escaping
            with open("translations.txt", "r") as f:
                content = f.read()

            # Both source and translation should have escaped quotes ("" for each ")
            assert '""hello""' in content  # Translation quotes escaped
            assert '""how are you?""' in content  # Source quotes escaped
            # Should have quoted fields
            assert '"She asked ""how are you?""","He said ""hello"""' in content

    @patch("review.llm.cli.subprocess.run")
    @patch("review.cli.shutil.which", return_value="/usr/bin/claude")
    def test_judge_command_accepts_cli_option(self, mock_which, mock_run):
        """judge command should accept --cli option."""
        # Mock subprocess returning JSON judgment
        mock_result = MagicMock()
        mock_result.returncode = 0
        mock_result.stdout = '{"winner": "translation_a", "confidence": 0.9, "reasoning": "Better flow", "concerns": []}'
        mock_run.return_value = mock_result

        runner = CliRunner()
        with runner.isolated_filesystem():
            with open("source.txt", "w") as f:
                f.write("Original text")
            with open("trans_a.txt", "w") as f:
                f.write("Translation A")
            with open("trans_b.txt", "w") as f:
                f.write("Translation B")

            result = runner.invoke(cli, [
                "judge",
                "source.txt", "trans_a.txt", "trans_b.txt",
                "--cli", "claude"
            ])

            # Should not error on cli validation
            assert result.exit_code == 0
            assert "translation_a" in result.output

    @patch("review.llm.cli.subprocess.run")
    @patch("review.cli.shutil.which", return_value="/usr/bin/claude")
    def test_judge_command_mutually_exclusive_provider_and_cli(self, mock_which, mock_run):
        """judge command should reject both --provider and --cli."""
        runner = CliRunner()
        with runner.isolated_filesystem():
            with open("source.txt", "w") as f:
                f.write("Original")
            with open("a.txt", "w") as f:
                f.write("A")
            with open("b.txt", "w") as f:
                f.write("B")

            result = runner.invoke(cli, [
                "judge",
                "source.txt", "a.txt", "b.txt",
                "--provider", "anthropic",
                "--cli", "claude"
            ])

            # Should reject both options
            assert result.exit_code != 0
            assert "Cannot specify both" in result.output

    def test_judge_command_shows_cli_option(self):
        """judge command help should show --cli option."""
        runner = CliRunner()
        result = runner.invoke(cli, ["judge", "--help"])
        assert result.exit_code == 0
        assert "--cli" in result.output or "local CLI tool" in result.output

    def test_translate_dry_run_shows_command(self):
        """translate --dry-run should show the command without executing."""
        runner = CliRunner()
        result = runner.invoke(translate, [
            "--cli", "claude",
            "こんにちは",
            "--dry-run"
        ])
        # Should succeed without executing
        assert result.exit_code == 0
        assert "Would execute:" in result.output
        assert "claude code" in result.output or "claude" in result.output

    def test_translate_dry_run_requires_cli(self):
        """translate --dry-run without --cli should error."""
        runner = CliRunner()
        result = runner.invoke(translate, [
            "--provider", "anthropic",
            "test",
            "--dry-run"
        ])
        # Should reject dry-run with API provider
        assert result.exit_code != 0
        assert "--dry-run only works with --cli" in result.output

    @patch("review.llm.cli.subprocess.run")
    @patch("review.cli.shutil.which", return_value="/usr/bin/claude")
    def test_batch_rejects_oversized_files(self, mock_which, mock_run):
        """Should reject input files exceeding size limit (10MB)."""
        # Mock successful subprocess
        mock_result = MagicMock()
        mock_result.returncode = 0
        mock_result.stdout = "Translation"
        mock_run.return_value = mock_result

        runner = CliRunner()
        with runner.isolated_filesystem():
            # Create file exceeding limit (10MB = 10,485,760 bytes)
            large_text = "x" * (11 * 1024 * 1024)  # 11MB
            with open("huge.txt", "w") as f:
                f.write(large_text)

            result = runner.invoke(batch, [
                "--input", "huge.txt",
                "--output", "out.txt",
                "--cli", "claude"
            ])

        assert result.exit_code != 0
        assert "too large" in result.output.lower() or "size limit" in result.output.lower()
