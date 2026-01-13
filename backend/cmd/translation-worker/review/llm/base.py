"""Base provider interface and data structures."""

from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Optional, List


@dataclass
class ProviderConfig:
    """Configuration for LLM provider."""

    api_key: str
    base_url: Optional[str] = None
    model: str = "claude-sonnet-4-5-20250929"
    max_tokens: int = 8192
    timeout: int = 120
    system_prompt: Optional[str] = None


@dataclass
class ProviderResponse:
    """Response from LLM provider."""

    text: str
    model: str
    usage: dict
    latency_ms: int
    raw_response: Optional[dict] = None


class BaseProvider(ABC):
    """Abstract base for LLM providers."""

    def __init__(self, config: ProviderConfig):
        self.config = config

    @abstractmethod
    def is_available(self) -> bool:
        """Check if provider is properly configured."""
        pass

    @abstractmethod
    def generate(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion from prompt."""
        pass

    @abstractmethod
    async def generate_async(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        """Async version of generate."""
        pass

    def _build_messages(self, prompt: str) -> List[dict]:
        """Build message list with optional system prompt."""
        messages = []
        if self.config.system_prompt:
            messages.append({"role": "system", "content": self.config.system_prompt})
        messages.append({"role": "user", "content": prompt})
        return messages
