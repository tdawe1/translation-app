import asyncio
import logging
import time
from dataclasses import dataclass, field
from typing import Optional, List, Dict, Any, Callable, TypeVar
from collections import deque
from threading import Lock
import random

from .base import BaseProvider, ProviderResponse

logger = logging.getLogger(__name__)

T = TypeVar("T")


@dataclass
class ProviderStats:
    requests_total: int = 0
    requests_success: int = 0
    requests_failed: int = 0
    total_latency_ms: int = 0
    last_error: Optional[str] = None
    last_error_time: Optional[float] = None
    consecutive_failures: int = 0
    cooldown_until: float = 0.0


@dataclass
class PoolConfig:
    max_concurrent_per_provider: int = 4
    cooldown_after_failures: int = 3
    cooldown_duration_seconds: float = 60.0
    request_timeout_seconds: float = 120.0
    retry_on_rate_limit: bool = True
    rate_limit_backoff_base: float = 1.0
    rate_limit_backoff_max: float = 32.0


class ProviderPool:
    """Pool of LLM providers for high-throughput concurrent API calls.

    Enables scaling beyond single-API-key rate limits by distributing
    requests across multiple provider instances (same model, different keys).

    Features:
    - Round-robin load balancing across providers
    - Per-provider concurrency limiting (default: 4 concurrent)
    - Automatic cooldown on consecutive failures
    - Rate limit detection and backoff
    - Statistics tracking per provider

    Usage:
        pool = ProviderPool()
        pool.add_provider("key1", OpenAIProvider(api_key="sk-key1"))
        pool.add_provider("key2", OpenAIProvider(api_key="sk-key2"))
        pool.add_provider("key3", OpenAIProvider(api_key="sk-key3"))

        results = await pool.generate_batch(prompts, max_tokens=1000)
    """

    def __init__(self, config: Optional[PoolConfig] = None):
        self.config = config or PoolConfig()
        self._providers: Dict[str, BaseProvider] = {}
        self._stats: Dict[str, ProviderStats] = {}
        self._semaphores: Dict[str, asyncio.Semaphore] = {}
        self._provider_order: deque = deque()
        self._lock = Lock()
        self._async_lock: Optional[asyncio.Lock] = None

    def _get_async_lock(self) -> asyncio.Lock:
        if self._async_lock is None:
            self._async_lock = asyncio.Lock()
        return self._async_lock

    def add_provider(self, provider_id: str, provider: BaseProvider) -> None:
        with self._lock:
            self._providers[provider_id] = provider
            self._stats[provider_id] = ProviderStats()
            self._semaphores[provider_id] = asyncio.Semaphore(
                self.config.max_concurrent_per_provider
            )
            self._provider_order.append(provider_id)
            logger.info(
                f"Added provider {provider_id} to pool (total: {len(self._providers)})"
            )

    def remove_provider(self, provider_id: str) -> bool:
        with self._lock:
            if provider_id not in self._providers:
                return False
            del self._providers[provider_id]
            del self._stats[provider_id]
            del self._semaphores[provider_id]
            self._provider_order = deque(
                [p for p in self._provider_order if p != provider_id]
            )
            logger.info(f"Removed provider {provider_id} from pool")
            return True

    def get_stats(self) -> Dict[str, ProviderStats]:
        with self._lock:
            return {k: ProviderStats(**v.__dict__) for k, v in self._stats.items()}

    def get_available_providers(self) -> List[str]:
        now = time.time()
        with self._lock:
            return [
                pid
                for pid in self._provider_order
                if self._stats[pid].cooldown_until < now
            ]

    async def _select_provider(self) -> Optional[str]:
        now = time.time()
        async with self._get_async_lock():
            available = [
                pid
                for pid in self._provider_order
                if self._stats[pid].cooldown_until < now
            ]
            if not available:
                return None
            selected = available[0]
            self._provider_order.rotate(-1)
            return selected

    def _is_rate_limit_error(self, error: Exception) -> bool:
        error_str = str(error).lower()
        return any(
            indicator in error_str
            for indicator in [
                "rate limit",
                "rate_limit",
                "429",
                "too many requests",
                "quota exceeded",
                "throttl",
                "capacity",
            ]
        )

    async def _generate_with_provider(
        self,
        provider_id: str,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0,
    ) -> ProviderResponse:
        provider = self._providers[provider_id]
        semaphore = self._semaphores[provider_id]
        stats = self._stats[provider_id]

        async with semaphore:
            start = time.time()
            try:
                response = await provider.generate_async(
                    prompt=prompt, max_tokens=max_tokens, temperature=temperature
                )
                latency = int((time.time() - start) * 1000)

                with self._lock:
                    stats.requests_total += 1
                    stats.requests_success += 1
                    stats.total_latency_ms += latency
                    stats.consecutive_failures = 0

                return response

            except Exception as e:
                with self._lock:
                    stats.requests_total += 1
                    stats.requests_failed += 1
                    stats.consecutive_failures += 1
                    stats.last_error = str(e)
                    stats.last_error_time = time.time()

                    if (
                        stats.consecutive_failures
                        >= self.config.cooldown_after_failures
                    ):
                        stats.cooldown_until = (
                            time.time() + self.config.cooldown_duration_seconds
                        )
                        logger.warning(
                            f"Provider {provider_id} entering cooldown after "
                            f"{stats.consecutive_failures} failures"
                        )

                raise

    async def generate(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0,
        max_retries: int = 3,
    ) -> ProviderResponse:
        last_error = None
        backoff = self.config.rate_limit_backoff_base

        for attempt in range(max_retries):
            provider_id = await self._select_provider()
            if provider_id is None:
                if last_error:
                    raise RuntimeError(
                        f"No providers available. Last error: {last_error}"
                    )
                raise RuntimeError("No providers available in pool")

            try:
                return await self._generate_with_provider(
                    provider_id, prompt, max_tokens, temperature
                )
            except Exception as e:
                last_error = e
                if self._is_rate_limit_error(e) and self.config.retry_on_rate_limit:
                    jitter = random.uniform(0, backoff * 0.1)
                    await asyncio.sleep(backoff + jitter)
                    backoff = min(backoff * 2, self.config.rate_limit_backoff_max)
                    continue
                if attempt == max_retries - 1:
                    raise

        raise RuntimeError(
            f"Failed after {max_retries} attempts. Last error: {last_error}"
        )

    async def generate_batch(
        self,
        prompts: List[str],
        max_tokens: Optional[int] = None,
        temperature: float = 0.0,
        max_concurrent: Optional[int] = None,
        on_progress: Optional[Callable[[int, int], None]] = None,
    ) -> List[ProviderResponse]:
        if not prompts:
            return []

        effective_concurrent = max_concurrent or (
            len(self._providers) * self.config.max_concurrent_per_provider
        )
        semaphore = asyncio.Semaphore(effective_concurrent)
        results: List[Optional[ProviderResponse]] = [None] * len(prompts)
        completed = 0
        lock = asyncio.Lock()

        async def process_prompt(idx: int, prompt: str) -> None:
            nonlocal completed
            async with semaphore:
                response = await self.generate(prompt, max_tokens, temperature)
                results[idx] = response
                async with lock:
                    completed += 1
                    if on_progress:
                        on_progress(completed, len(prompts))

        tasks = [process_prompt(i, p) for i, p in enumerate(prompts)]
        await asyncio.gather(*tasks)

        return [r for r in results if r is not None]

    def reset_stats(self) -> None:
        with self._lock:
            for provider_id in self._stats:
                self._stats[provider_id] = ProviderStats()

    def reset_cooldowns(self) -> None:
        with self._lock:
            for provider_id in self._stats:
                self._stats[provider_id].cooldown_until = 0.0
                self._stats[provider_id].consecutive_failures = 0

    def __len__(self) -> int:
        return len(self._providers)

    def __repr__(self) -> str:
        available = len(self.get_available_providers())
        return f"ProviderPool(providers={len(self._providers)}, available={available})"


def create_openai_pool(
    api_keys: List[str],
    model: Optional[str] = None,
    config: Optional[PoolConfig] = None,
) -> ProviderPool:
    from .providers import OpenAIProvider

    pool = ProviderPool(config)
    for i, key in enumerate(api_keys):
        provider = OpenAIProvider(api_key=key, model=model)
        pool.add_provider(f"openai_{i}", provider)
    return pool


def create_anthropic_pool(
    api_keys: List[str],
    model: Optional[str] = None,
    config: Optional[PoolConfig] = None,
) -> ProviderPool:
    from .providers import AnthropicProvider

    pool = ProviderPool(config)
    for i, key in enumerate(api_keys):
        provider = AnthropicProvider(api_key=key, model=model)
        pool.add_provider(f"anthropic_{i}", provider)
    return pool


def create_mixed_pool(
    providers_config: List[Dict[str, Any]], pool_config: Optional[PoolConfig] = None
) -> ProviderPool:
    from .providers import get_provider

    pool = ProviderPool(pool_config)
    for i, cfg in enumerate(providers_config):
        provider_type = cfg.pop("type")
        provider_id = cfg.pop("id", f"{provider_type}_{i}")
        api_key = cfg.pop("api_key")
        model = cfg.pop("model", None)

        provider = get_provider(provider_type, api_key, model, **cfg)
        pool.add_provider(provider_id, provider)

    return pool
