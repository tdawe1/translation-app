# tests/test_review/test_llm_providers.py
"""Tests for LLM provider abstraction layer."""

import pytest
import sys
from pathlib import Path

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.llm.providers import (
    AnthropicProvider,
    OpenAIProvider,
    GeminiProvider,
    get_provider,
)


class TestProviderAbstraction:
    def test_anthropic_provider_requires_api_key(self):
        """Should raise error if API key missing."""
        provider = AnthropicProvider(api_key=None)
        with pytest.raises(ValueError, match="API key"):
            provider.is_available()

    def test_openai_provider_requires_api_key(self):
        """Should raise error if API key missing."""
        provider = OpenAIProvider(api_key=None)
        with pytest.raises(ValueError, match="API key"):
            provider.is_available()

    def test_gemini_provider_requires_api_key(self):
        """Should raise error if API key missing."""
        provider = GeminiProvider(api_key=None)
        with pytest.raises(ValueError, match="API key"):
            provider.is_available()

    def test_get_provider_returns_correct_type(self):
        """Should return correct provider instance."""
        anthropic = get_provider("anthropic", api_key="test-key")
        assert isinstance(anthropic, AnthropicProvider)

        openai = get_provider("openai", api_key="test-key")
        assert isinstance(openai, OpenAIProvider)

        gemini = get_provider("gemini", api_key="test-key", project_id="test-project")
        assert isinstance(gemini, GeminiProvider)

    def test_get_provider_raises_on_unknown(self):
        """Should raise ValueError for unknown provider."""
        with pytest.raises(ValueError, match="Unknown provider"):
            get_provider("unknown", api_key="test-key")


class TestSystemPromptInjection:
    def test_anthropic_provider_accepts_system_prompt(self):
        provider = AnthropicProvider(
            api_key="test-key", system_prompt="You are a translator."
        )
        assert provider.config.system_prompt == "You are a translator."

    def test_openai_provider_accepts_system_prompt(self):
        provider = OpenAIProvider(
            api_key="test-key", system_prompt="You are a translator."
        )
        assert provider.config.system_prompt == "You are a translator."

    def test_gemini_provider_accepts_system_prompt(self):
        provider = GeminiProvider(
            api_key="test-key", project_id="test", system_prompt="You are a translator."
        )
        assert provider.config.system_prompt == "You are a translator."

    def test_get_provider_passes_system_prompt(self):
        provider = get_provider(
            "anthropic", api_key="test-key", system_prompt="Gengo style guide"
        )
        assert provider.config.system_prompt == "Gengo style guide"

    def test_build_messages_includes_system_prompt(self):
        provider = AnthropicProvider(
            api_key="test-key", system_prompt="System instructions here"
        )
        messages = provider._build_messages("Hello")
        assert len(messages) == 2
        assert messages[0]["role"] == "system"
        assert messages[0]["content"] == "System instructions here"
        assert messages[1]["role"] == "user"
        assert messages[1]["content"] == "Hello"

    def test_build_messages_without_system_prompt(self):
        provider = AnthropicProvider(api_key="test-key")
        messages = provider._build_messages("Hello")
        assert len(messages) == 1
        assert messages[0]["role"] == "user"
        assert messages[0]["content"] == "Hello"
