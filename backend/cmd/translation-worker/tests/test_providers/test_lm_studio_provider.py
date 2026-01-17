"""
Tests for LM Studio provider.

LM Studio exposes an OpenAI-compatible API for locally loaded models.
These tests use mocking to avoid requiring an actual LM Studio instance.
"""

import pytest
from unittest.mock import Mock, patch, AsyncMock
import httpx
from review.llm.lm_studio import LMStudioProvider


@pytest.fixture
def lm_studio_provider():
    """Create LM Studio provider with test configuration."""
    return LMStudioProvider(base_url="http://localhost:1234/v1", model="local-model")


def test_lm_studio_provider_initialization(lm_studio_provider):
    """Should initialize with OpenAI-compatible endpoint."""
    assert lm_studio_provider.name == "lm_studio"
    assert lm_studio_provider.model == "local-model"
    assert lm_studio_provider.base_url == "http://localhost:1234/v1"


def test_lm_studio_provider_is_available(lm_studio_provider):
    """Should check availability via /models endpoint."""
    # LM Studio is available
    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_client.get.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        assert lm_studio_provider.is_available() is True

    # LM Studio is not available
    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 503
        mock_client.get.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        assert lm_studio_provider.is_available() is False


def test_lm_studio_provider_is_available_timeout(lm_studio_provider):
    """Should return False on request timeout."""
    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_client.get.side_effect = httpx.TimeoutException("Timeout")
        mock_client_class.return_value.__aenter__.return_value = mock_client

        assert lm_studio_provider.is_available() is False


def test_lm_studio_provider_is_available_connection_error(lm_studio_provider):
    """Should return False on connection error."""
    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_client.get.side_effect = httpx.ConnectError("Connection error")
        mock_client_class.return_value.__aenter__.return_value = mock_client

        assert lm_studio_provider.is_available() is False


def test_lm_studio_provider_list_models(lm_studio_provider):
    """Should list available models from LM Studio."""
    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "data": [
                {"id": "llama-3.1-8b-instruct"},
                {"id": "qwen2.5-72b-instruct"},
            ]
        }
        mock_client.get.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        models = lm_studio_provider.list_models()
        assert len(models) == 2
        assert "llama-3.1-8b-instruct" in models


def test_lm_studio_provider_list_models_error(lm_studio_provider):
    """Should return empty list on error."""
    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 500
        mock_response.raise_for_status.side_effect = httpx.HTTPStatusError(
            "Error", request=Mock(), response=mock_response
        )
        mock_client.get.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        models = lm_studio_provider.list_models()
        assert models == []


def test_lm_studio_provider_translate(lm_studio_provider):
    """Should translate using LM Studio's OpenAI-compatible API."""
    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "choices": [{"message": {"content": "Hello"}, "finish_reason": "stop"}]
        }
        mock_client.post.return_value = mock_response
        mock_client.get.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        result = lm_studio_provider.translate(
            text="こんにちは", source_lang="ja", target_lang="en"
        )

        assert result.success is True
        assert result.translated_text == "Hello"
        assert result.provider == "lm_studio"


def test_lm_studio_provider_translate_unavailable(lm_studio_provider):
    """Should return error when LM Studio is not available."""
    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 503
        mock_client.get.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        result = lm_studio_provider.translate(
            text="こんにちは", source_lang="ja", target_lang="en"
        )

        assert result.success is False
        assert result.translated_text == ""
        assert "not available" in result.error.lower()


def test_lm_studio_provider_translate_api_error(lm_studio_provider):
    """Should handle API errors gracefully."""
    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_get_response = Mock()
        mock_get_response.status_code = 200
        mock_client.get.return_value = mock_get_response

        mock_post_response = Mock()
        mock_post_response.status_code = 500
        mock_post_response.raise_for_status.side_effect = httpx.HTTPStatusError(
            "Error", request=Mock(), response=mock_post_response
        )
        mock_client.post.return_value = mock_post_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        result = lm_studio_provider.translate(
            text="こんにちは", source_lang="ja", target_lang="en"
        )

        assert result.success is False
        assert result.error is not None


def test_lm_studio_provider_generate(lm_studio_provider):
    """Should generate text using LM Studio and return ProviderResponse."""
    from review.llm.base import ProviderResponse

    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.json.return_value = {
            "choices": [{"message": {"content": "Generated response"}}],
            "usage": {
                "prompt_tokens": 5,
                "completion_tokens": 3,
                "total_tokens": 8,
            },
        }
        mock_client.post.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        with patch.object(
            lm_studio_provider,
            "is_available_async",
            return_value=AsyncMock(return_value=True)(),
        ):
            result = lm_studio_provider.generate(
                prompt="Hello, world!", temperature=0.5
            )

        assert isinstance(result, ProviderResponse)
        assert result.text == "Generated response"
        assert result.model == "local-model"


def test_lm_studio_provider_generate_error(lm_studio_provider):
    """Should raise RuntimeError on generation errors."""
    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 500
        mock_response.raise_for_status.side_effect = httpx.HTTPStatusError(
            "Error", request=Mock(), response=mock_response
        )
        mock_client.post.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        with patch.object(
            lm_studio_provider,
            "is_available_async",
            return_value=AsyncMock(return_value=True)(),
        ):
            with pytest.raises(httpx.HTTPStatusError):
                lm_studio_provider.generate(prompt="Hello, world!")


def test_lm_studio_provider_custom_base_url():
    """Should allow custom base URL."""
    provider = LMStudioProvider(
        base_url="http://192.168.1.100:5678/v1", model="custom-model"
    )

    assert provider.base_url == "http://192.168.1.100:5678/v1"
    assert provider.model == "custom-model"


def test_lm_studio_provider_default_config():
    """Should use default configuration."""
    provider = LMStudioProvider()

    assert provider.base_url == "http://localhost:1234/v1"
    assert provider.model == "local-model"


def test_generate_returns_provider_response(lm_studio_provider):
    """LMStudioProvider.generate() should return ProviderResponse, not str.

    This test verifies contract compliance with BaseProvider abstract class.
    """
    from review.llm.base import ProviderResponse

    with patch("review.llm.lm_studio.httpx.AsyncClient") as mock_client_class:
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.json.return_value = {
            "choices": [{"message": {"content": "Test translation"}}],
            "usage": {
                "prompt_tokens": 10,
                "completion_tokens": 5,
                "total_tokens": 15,
            },
        }
        mock_client.post.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        with patch.object(
            lm_studio_provider,
            "is_available_async",
            return_value=AsyncMock(return_value=True)(),
        ):
            result = lm_studio_provider.generate("Translate this", temperature=0.5)

        # Should return ProviderResponse, not str
        assert isinstance(result, ProviderResponse), (
            f"Expected ProviderResponse, got {type(result)}"
        )
        assert hasattr(result, "text")
        assert hasattr(result, "model")
        assert hasattr(result, "usage")
        assert hasattr(result, "latency_ms")
        assert result.text == "Test translation"
        assert result.model == "local-model"
