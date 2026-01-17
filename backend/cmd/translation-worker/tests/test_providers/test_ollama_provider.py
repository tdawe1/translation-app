import pytest
import sys
from pathlib import Path
from unittest.mock import Mock, patch, MagicMock, AsyncMock

worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from review.llm.ollama import OllamaProvider, TranslationResult, get_ollama_provider


class TestOllamaProviderInitialization:
    def test_initialization_with_defaults(self):
        provider = OllamaProvider()
        assert provider.name == "ollama"
        assert provider.model == "llama3.1:8b"
        assert provider.base_url == "http://localhost:11434"

    def test_initialization_with_custom_values(self):
        provider = OllamaProvider(
            base_url="http://192.168.1.100:11434", model="qwen2.5:72b"
        )
        assert provider.base_url == "http://192.168.1.100:11434"
        assert provider.model == "qwen2.5:72b"


class TestOllamaProviderAvailability:
    @patch("review.llm.ollama.httpx.AsyncClient")
    def test_is_available_when_server_running(self, mock_client_class):
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_client.get.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        provider = OllamaProvider()
        assert provider.is_available() is True

    @patch("review.llm.ollama.httpx.AsyncClient")
    def test_is_available_when_server_down(self, mock_client_class):
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 503
        mock_client.get.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        provider = OllamaProvider()
        assert provider.is_available() is False

    @patch("review.llm.ollama.httpx.AsyncClient")
    def test_is_available_handles_connection_error(self, mock_client_class):
        import httpx

        mock_client = AsyncMock()
        mock_client.get.side_effect = httpx.ConnectError()
        mock_client_class.return_value.__aenter__.return_value = mock_client

        provider = OllamaProvider()
        assert provider.is_available() is False


class TestOllamaProviderListModels:
    @patch("review.llm.ollama.httpx.AsyncClient")
    def test_list_models_returns_model_names(self, mock_client_class):
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "models": [
                {"name": "llama3.1:8b", "size": 4882919328},
                {"name": "llama3.1:70b", "size": 42101985824},
                {"name": "qwen2.5:72b", "size": 43807961024},
            ]
        }
        mock_client.get.return_value = mock_response
        mock_response.raise_for_status = Mock()
        mock_client_class.return_value.__aenter__.return_value = mock_client

        provider = OllamaProvider()
        models = provider.list_models()

        assert len(models) == 3
        assert "llama3.1:8b" in models
        assert "qwen2.5:72b" in models

    @patch("review.llm.ollama.httpx.AsyncClient")
    def test_list_models_handles_error(self, mock_client_class):
        import httpx

        mock_client = AsyncMock()
        mock_client.get.side_effect = httpx.ConnectError()
        mock_client_class.return_value.__aenter__.return_value = mock_client

        provider = OllamaProvider()
        models = provider.list_models()

        assert models == []


class TestOllamaProviderGenerate:
    @patch("review.llm.ollama.httpx.AsyncClient")
    def test_generate_successful(self, mock_client_class):
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "model": "llama3.1:8b",
            "response": "Hello world",
            "done": True,
            "prompt_eval_count": 10,
            "eval_count": 5,
        }
        mock_client.post.return_value = mock_response
        mock_response.raise_for_status = Mock()
        mock_client_class.return_value.__aenter__.return_value = mock_client

        provider = OllamaProvider()
        response = provider.generate("Translate: こんにちは")

        assert response.text == "Hello world"
        assert response.model == "llama3.1:8b"
        assert response.usage["prompt_tokens"] == 10
        assert response.usage["completion_tokens"] == 5

    @patch("review.llm.ollama.httpx.AsyncClient")
    def test_generate_with_max_tokens(self, mock_client_class):
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "model": "llama3.1:8b",
            "response": "Result",
            "done": True,
        }
        mock_client.post.return_value = mock_response
        mock_response.raise_for_status = Mock()
        mock_client_class.return_value.__aenter__.return_value = mock_client

        provider = OllamaProvider()
        provider.generate("Test", max_tokens=100)

        call_kwargs = mock_client.post.call_args[1]
        payload = call_kwargs["json"]
        assert payload["options"]["num_predict"] == 100

    @patch("review.llm.ollama.httpx.AsyncClient")
    def test_generate_handles_request_error(self, mock_client_class):
        import httpx

        mock_client = AsyncMock()
        mock_client.post.side_effect = httpx.TimeoutException("Timeout")
        mock_client_class.return_value.__aenter__.return_value = mock_client

        provider = OllamaProvider()
        with pytest.raises(RuntimeError, match="Ollama request failed"):
            provider.generate("Test")


class TestOllamaProviderTranslate:
    @patch("review.llm.ollama.httpx.AsyncClient")
    def test_translate_successful(self, mock_client_class):
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "model": "llama3.1:8b",
            "response": "Hello",
            "done": True,
        }
        mock_client.post.return_value = mock_response
        mock_client.get.return_value = mock_response
        mock_response.raise_for_status = Mock()
        mock_client_class.return_value.__aenter__.return_value = mock_client

        provider = OllamaProvider()
        result = provider.translate("こんにちは", source_lang="ja", target_lang="en")

        assert result.success is True
        assert result.translated_text == "Hello"
        assert result.provider == "ollama"

    @patch("review.llm.ollama.httpx.AsyncClient")
    def test_translate_fails_when_server_unavailable(self, mock_client_class):
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 503
        mock_client.get.return_value = mock_response
        mock_client_class.return_value.__aenter__.return_value = mock_client

        provider = OllamaProvider()
        result = provider.translate("こんにちは", source_lang="ja", target_lang="en")

        assert result.success is False
        assert result.error == "Ollama server not available"


class TestOllamaProviderAsync:
    @pytest.mark.asyncio
    @patch("review.llm.ollama.httpx.AsyncClient")
    async def test_generate_async_delegates_to_generate(self, mock_client_class):
        mock_client = AsyncMock()
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "model": "llama3.1:8b",
            "response": "Async result",
            "done": True,
        }
        mock_client.post.return_value = mock_response
        mock_response.raise_for_status = Mock()
        mock_client_class.return_value.__aenter__.return_value = mock_client

        provider = OllamaProvider()
        response = await provider.generate_async("Test")

        assert response.text == "Async result"


class TestGetOllamaProviderFactory:
    def test_factory_returns_ollama_provider(self):
        provider = get_ollama_provider()
        assert isinstance(provider, OllamaProvider)
        assert provider.name == "ollama"

    def test_factory_accepts_custom_config(self):
        provider = get_ollama_provider(
            base_url="http://custom:11434", model="qwen2.5:72b"
        )
        assert provider.base_url == "http://custom:11434"
        assert provider.model == "qwen2.5:72b"


class TestTranslationResult:
    def test_translation_result_success(self):
        result = TranslationResult(
            success=True,
            translated_text="Hello",
            confidence=0.85,
            provider="ollama",
            model="llama3.1:8b",
        )
        assert result.success is True
        assert result.error is None

    def test_translation_result_failure(self):
        result = TranslationResult(
            success=False,
            translated_text="",
            confidence=0.0,
            provider="ollama",
            model="llama3.1:8b",
            error="Connection refused",
        )
        assert result.success is False
        assert result.error == "Connection refused"
