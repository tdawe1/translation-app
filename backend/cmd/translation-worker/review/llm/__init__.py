"""LLM provider abstraction for translation and judge operations."""

from .base import BaseProvider, ProviderConfig, ProviderResponse
from .providers import AnthropicProvider, OpenAIProvider, GeminiProvider, get_provider
from .cli import CLIProvider, get_cli_provider
from .ollama import OllamaProvider, get_ollama_provider, TranslationResult
from .lm_studio import LMStudioProvider, get_lm_studio_provider
from .pool import (
    ProviderPool,
    PoolConfig,
    ProviderStats,
    create_openai_pool,
    create_anthropic_pool,
    create_mixed_pool,
)
from .batch import (
    BatchTranslator,
    ChunkedBatchTranslator,
    TranslationRequest,
    TranslationResponse,
)
from .cloud_providers import (
    OpenRouterProvider,
    GitHubModelsProvider,
    AWSBedrockProvider,
    VertexAIProvider,
    get_openrouter_provider,
    get_github_models_provider,
    get_bedrock_provider,
    get_vertex_provider,
)

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
    "ProviderPool",
    "PoolConfig",
    "ProviderStats",
    "create_openai_pool",
    "create_anthropic_pool",
    "create_mixed_pool",
    "BatchTranslator",
    "ChunkedBatchTranslator",
    "TranslationRequest",
    "TranslationResponse",
    "OpenRouterProvider",
    "GitHubModelsProvider",
    "AWSBedrockProvider",
    "VertexAIProvider",
    "get_openrouter_provider",
    "get_github_models_provider",
    "get_bedrock_provider",
    "get_vertex_provider",
]
