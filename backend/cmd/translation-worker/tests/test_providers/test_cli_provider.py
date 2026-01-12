# tests/test_providers/test_cli.py
"""Tests for CLI tool provider."""

import pytest
import sys
from pathlib import Path
from unittest.mock import Mock, patch, MagicMock
import subprocess

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.llm.cli import CLIProvider, get_cli_provider
from review.llm.base import ProviderConfig


class TestCLIProviderInitialization:
    """Test CLI provider initialization and configuration."""

    def test_initialization_with_tool_name(self):
        """Should initialize with tool name."""
        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        assert provider.tool_name == "claude_code"

    def test_initialization_with_custom_command(self):
        """Should accept custom command path."""
        config = ProviderConfig(api_key="test")
        provider = CLIProvider(
            tool_name="codex", config=config, command="/usr/local/bin/codex"
        )
        assert provider.command == "/usr/local/bin/codex"

    def test_initialization_with_default_commands(self):
        """Should have default commands for each tool."""
        config = ProviderConfig(api_key="test")
        claude = CLIProvider(tool_name="claude_code", config=config)
        assert claude.command == "claude"

        gemini = CLIProvider(tool_name="gemini_cli", config=config)
        assert gemini.command == "gemini-cli"

        codex = CLIProvider(tool_name="codex", config=config)
        assert codex.command == "codex"

    def test_initialization_with_custom_timeout(self):
        """Should accept custom timeout."""
        config = ProviderConfig(api_key="test", timeout=180)
        provider = CLIProvider(tool_name="claude_code", config=config)
        assert provider.config.timeout == 180


class TestCLIProviderAvailability:
    """Test provider availability checks."""

    def test_is_available_requires_tool_name(self):
        """Should raise ValueError if tool_name is empty."""
        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="", config=config)
        with pytest.raises(ValueError, match="tool name is required"):
            provider.is_available()

    @patch("shutil.which")
    def test_is_available_requires_tool_installed(self, mock_which):
        """Should raise ValueError if command not found."""
        mock_which.return_value = None
        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        with pytest.raises(ValueError, match="not found"):
            provider.is_available()

    @patch("shutil.which")
    def test_is_available_succeeds_when_tool_exists(self, mock_which):
        """Should return True when tool is installed."""
        mock_which.return_value = "/usr/bin/claude"
        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        assert provider.is_available() is True


class TestCLIProviderGenerate:
    """Test generate method."""

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_generate_successful_translation(self, mock_which, mock_run):
        """Should successfully translate text."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(stdout="Hello world", stderr="", returncode=0)

        provider = CLIProvider(tool_name="claude_code", config={"api_key": "test"})
        response = provider.generate("Hello")

        assert response.text == "Hello world"
        assert "prompt_tokens" in response.usage
        assert "completion_tokens" in response.usage
        assert response.latency_ms >= 0

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_generate_with_max_tokens(self, mock_which, mock_run):
        """Should pass max_tokens to CLI."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(stdout="Translated text", stderr="", returncode=0)

        provider = CLIProvider(tool_name="claude_code", config={"api_key": "test"})
        response = provider.generate("Hello", max_tokens=500)

        assert response.text == "Translated text"

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_generate_with_temperature(self, mock_which, mock_run):
        """Should pass temperature to CLI."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(stdout="Output", stderr="", returncode=0)

        provider = CLIProvider(tool_name="claude_code", config={"api_key": "test"})
        response = provider.generate("Hello", temperature=0.7)

        assert response.text == "Output"

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_generate_successful_translation(self, mock_which, mock_run):
        """Should successfully translate text."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(stdout="Hello world", stderr="", returncode=0)

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = provider.generate("Hello")

        assert response.text == "Hello world"
        assert "prompt_tokens" in response.usage
        assert "completion_tokens" in response.usage
        assert response.latency_ms >= 0

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_generate_with_max_tokens(self, mock_which, mock_run):
        """Should pass max_tokens to CLI."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(stdout="Translated text", stderr="", returncode=0)

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = provider.generate("Hello", max_tokens=500)

        assert response.text == "Translated text"

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_generate_with_temperature(self, mock_which, mock_run):
        """Should pass temperature to CLI."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(stdout="Output", stderr="", returncode=0)

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = provider.generate("Hello", temperature=0.7)

        assert response.text == "Output"

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_generate_command_timeout(self, mock_which, mock_run):
        """Should use configured timeout."""
        mock_which.return_value = "/usr/bin/gemini"
        mock_run.return_value = Mock(stdout="Result", stderr="", returncode=0)

        config = ProviderConfig(api_key="test", timeout=180)
        provider = CLIProvider(tool_name="gemini_cli", config=config)
        provider.generate("Test")

        mock_run.assert_called()
        call_kwargs = mock_run.call_args[1]
        assert call_kwargs["timeout"] == 180

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_generate_handles_non_zero_exit(self, mock_which, mock_run):
        """Should raise RuntimeError on non-zero exit code."""
        mock_which.return_value = "/usr/bin/codex"
        mock_run.side_effect = subprocess.CalledProcessError(
            returncode=1, cmd=["codex", "Test"], stderr="Error: API key invalid"
        )

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="codex", config=config)
        with pytest.raises(RuntimeError, match="CLI tool failed"):
            provider.generate("Test")


class TestCLIProviderGenerateAsync:
    """Test async generate method."""

    @pytest.mark.asyncio
    @patch("subprocess.run")
    @patch("shutil.which")
    async def test_generate_async_delegates_to_generate(self, mock_which, mock_run):
        """Should delegate synchronous generate to thread."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(stdout="Async result", stderr="", returncode=0)

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = await provider.generate_async("Test")

        assert response.text == "Async result"
        mock_run.assert_called_once()


class TestCLIProviderRetryLogic:
    """Test retry mechanism."""

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_retries_on_failure(self, mock_which, mock_run):
        """Should retry on subprocess errors."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.side_effect = [
            subprocess.TimeoutExpired("claude", 120),
            subprocess.TimeoutExpired("claude", 120),
            Mock(stdout="Success", stderr="", returncode=0),
        ]

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = provider.generate("Test")

        assert response.text == "Success"
        assert mock_run.call_count == 3

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_raises_after_max_retries(self, mock_which, mock_run):
        """Should raise after max retries."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.side_effect = subprocess.TimeoutExpired("claude", 120)

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        with pytest.raises(Exception):
            provider.generate("Test")


class TestCLIProviderConfidenceScores:
    """Test confidence score parsing."""

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_extracts_confidence_from_json_output(self, mock_which, mock_run):
        """Should extract confidence from JSON output."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(
            stdout='{"text": "Hello", "confidence": 0.95}',
            stderr="",
            returncode=0,
        )

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = provider.generate("Test")

        assert response.raw_response is not None
        assert "confidence" in response.raw_response

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_defaults_confidence_to_none_if_missing(self, mock_which, mock_run):
        """Should default confidence to None if not in output."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(stdout="Hello world", stderr="", returncode=0)

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = provider.generate("Test")

        assert response.raw_response is None or "confidence" not in (
            response.raw_response or {}
        )


class TestGetCLIProviderFactory:
    """Test factory function."""

    def test_factory_returns_correct_provider(self):
        """Should return CLIProvider instance."""
        provider = get_cli_provider(tool_name="claude_code", api_key="test")
        assert isinstance(provider, CLIProvider)
        assert provider.tool_name == "claude_code"

    def test_factory_accepts_all_tools(self):
        """Should accept all tool names."""
        for tool in ["claude_code", "gemini_cli", "codex"]:
            provider = get_cli_provider(tool_name=tool, api_key="test")
            assert isinstance(provider, CLIProvider)

    def test_factory_raises_on_unknown_tool(self):
        """Should raise ValueError for unknown tool."""
        with pytest.raises(ValueError, match="Unknown CLI tool"):
            get_cli_provider(tool_name="unknown_tool", api_key="test")

    def test_factory_passes_config(self):
        """Should pass additional config to provider."""
        provider = get_cli_provider(
            tool_name="claude_code", api_key="test", command="/custom/path", timeout=200
        )
        assert provider.command == "/custom/path"
        assert provider.config.timeout == 200


class TestCLIProviderEdgeCases:
    """Test edge cases and error handling."""

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_handles_empty_output(self, mock_which, mock_run):
        """Should handle empty stdout."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(stdout="", stderr="", returncode=0)

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = provider.generate("Test")

        assert response.text == ""

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_handles_stderr_output(self, mock_which, mock_run):
        """Should capture stderr in raw_response."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(
            stdout="Result", stderr="Warning: rate limit", returncode=0
        )

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = provider.generate("Test")

        assert response.text == "Result"

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_handles_unicode_output(self, mock_which, mock_run):
        """Should handle unicode characters."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(
            stdout="こんにちは 世界 🎉", stderr="", returncode=0
        )

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = provider.generate("Test")

        assert "こんにちは" in response.text
        assert "🎉" in response.text

    @patch("subprocess.run")
    @patch("shutil.which")
    def test_tracks_usage_metrics(self, mock_which, mock_run):
        """Should track latency and usage."""
        mock_which.return_value = "/usr/bin/claude"
        mock_run.return_value = Mock(stdout="Output", stderr="", returncode=0)

        config = ProviderConfig(api_key="test")
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = provider.generate("Test")

        assert "latency_ms" in dir(response)
        assert isinstance(response.latency_ms, int)
        assert response.usage is not None
