# tests/test_review/test_llm_providers.py
"""Tests for LLM provider abstraction layer."""
import pytest
import sys
from pathlib import Path

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.llm.providers import AnthropicProvider, OpenAIProvider, GeminiProvider, get_provider


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
