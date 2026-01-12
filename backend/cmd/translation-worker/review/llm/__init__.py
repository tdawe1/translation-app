"""LLM provider abstraction for translation and judge operations."""

from .base import BaseProvider, ProviderConfig, ProviderResponse
from .providers import AnthropicProvider, OpenAIProvider, GeminiProvider, get_provider
from .cli import CLIProvider, get_cli_provider
from .ollama import OllamaProvider, get_ollama_provider, TranslationResult
from .lm_studio import LMStudioProvider, get_lm_studio_provider

__all__ = [
    "BaseProvider",
    "ProviderConfig",
    "ProviderResponse",
    "AnthropicProvider",
    "OpenAIProvider",
    "GeminiProvider",
    "CLIProvider",
    "OllamaProvider",
    "LMStudioProvider",
    "TranslationResult",
    "get_provider",
    "get_cli_provider",
    "get_ollama_provider",
    "get_lm_studio_provider",
]
