import asyncio
import pytest
import time
from unittest.mock import Mock, AsyncMock, patch, MagicMock
from dataclasses import dataclass

from review.llm.pool import ProviderPool, PoolConfig, ProviderStats
from review.llm.base import BaseProvider, ProviderConfig, ProviderResponse


class MockProvider(BaseProvider):
    def __init__(self, api_key: str = "test", delay: float = 0.01, fail_count: int = 0):
        config = ProviderConfig(api_key=api_key, model="mock-model")
        super().__init__(config)
        self.delay = delay
        self.fail_count = fail_count
        self.call_count = 0

    def is_available(self) -> bool:
        return True

    def generate(
        self, prompt: str, max_tokens=None, temperature=0.0
    ) -> ProviderResponse:
        self.call_count += 1
        if self.call_count <= self.fail_count:
            raise RuntimeError("Simulated failure")
        time.sleep(self.delay)
        return ProviderResponse(
            text=f"Response to: {prompt[:20]}",
            model="mock-model",
            usage={"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30},
            latency_ms=int(self.delay * 1000),
        )

    async def generate_async(
        self, prompt: str, max_tokens=None, temperature=0.0
    ) -> ProviderResponse:
        self.call_count += 1
        if self.call_count <= self.fail_count:
            raise RuntimeError("Simulated failure")
        await asyncio.sleep(self.delay)
        return ProviderResponse(
            text=f"Response to: {prompt[:20]}",
            model="mock-model",
            usage={"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30},
            latency_ms=int(self.delay * 1000),
        )


class TestProviderPool:
    def test_create_empty_pool(self):
        pool = ProviderPool()
        assert len(pool) == 0
        assert pool.get_available_providers() == []

    def test_add_provider(self):
        pool = ProviderPool()
        provider = MockProvider()
        pool.add_provider("test_1", provider)

        assert len(pool) == 1
        assert "test_1" in pool.get_available_providers()

    def test_add_multiple_providers(self):
        pool = ProviderPool()
        for i in range(3):
            pool.add_provider(f"provider_{i}", MockProvider())

        assert len(pool) == 3
        assert len(pool.get_available_providers()) == 3

    def test_remove_provider(self):
        pool = ProviderPool()
        pool.add_provider("test_1", MockProvider())
        pool.add_provider("test_2", MockProvider())

        result = pool.remove_provider("test_1")
        assert result is True
        assert len(pool) == 1
        assert "test_1" not in pool.get_available_providers()

    def test_remove_nonexistent_provider(self):
        pool = ProviderPool()
        result = pool.remove_provider("nonexistent")
        assert result is False

    def test_custom_config(self):
        config = PoolConfig(
            max_concurrent_per_provider=10,
            cooldown_duration_seconds=120.0,
            cooldown_after_failures=5,
        )
        pool = ProviderPool(config)

        assert pool.config.max_concurrent_per_provider == 10
        assert pool.config.cooldown_duration_seconds == 120.0
        assert pool.config.cooldown_after_failures == 5


class TestProviderPoolAsync:
    @pytest.mark.asyncio
    async def test_generate_single(self):
        pool = ProviderPool()
        pool.add_provider("test_1", MockProvider(delay=0.001))

        response = await pool.generate("Hello world")

        assert response.text.startswith("Response to:")
        assert response.model == "mock-model"

    @pytest.mark.asyncio
    async def test_generate_no_providers(self):
        pool = ProviderPool()

        with pytest.raises(RuntimeError, match="No providers available"):
            await pool.generate("Hello")

    @pytest.mark.asyncio
    async def test_generate_batch(self):
        pool = ProviderPool()
        pool.add_provider("test_1", MockProvider(delay=0.001))
        pool.add_provider("test_2", MockProvider(delay=0.001))

        prompts = [f"Prompt {i}" for i in range(5)]
        responses = await pool.generate_batch(prompts)

        assert len(responses) == 5
        for resp in responses:
            assert resp.text.startswith("Response to:")

    @pytest.mark.asyncio
    async def test_generate_batch_empty(self):
        pool = ProviderPool()
        pool.add_provider("test_1", MockProvider())

        responses = await pool.generate_batch([])
        assert responses == []

    @pytest.mark.asyncio
    async def test_round_robin_distribution(self):
        pool = ProviderPool()
        provider1 = MockProvider(delay=0.001)
        provider2 = MockProvider(delay=0.001)
        pool.add_provider("p1", provider1)
        pool.add_provider("p2", provider2)

        for _ in range(4):
            await pool.generate("test")

        assert provider1.call_count == 2
        assert provider2.call_count == 2

    @pytest.mark.asyncio
    async def test_progress_callback(self):
        pool = ProviderPool()
        pool.add_provider("test_1", MockProvider(delay=0.001))

        progress_updates = []

        def on_progress(done, total):
            progress_updates.append((done, total))

        prompts = [f"Prompt {i}" for i in range(3)]
        await pool.generate_batch(prompts, on_progress=on_progress)

        assert len(progress_updates) == 3
        assert progress_updates[-1] == (3, 3)


class TestProviderPoolFailures:
    @pytest.mark.asyncio
    async def test_retry_on_failure(self):
        pool = ProviderPool()
        provider = MockProvider(delay=0.001, fail_count=1)
        pool.add_provider("test_1", provider)
        pool.add_provider("test_2", MockProvider(delay=0.001))

        response = await pool.generate("test")
        assert response.text.startswith("Response to:")

    @pytest.mark.asyncio
    async def test_cooldown_after_failures(self):
        config = PoolConfig(cooldown_after_failures=2, cooldown_duration_seconds=1.0)
        pool = ProviderPool(config)

        failing_provider = MockProvider(delay=0.001, fail_count=100)
        working_provider = MockProvider(delay=0.001)
        pool.add_provider("failing", failing_provider)
        pool.add_provider("working", working_provider)

        for _ in range(5):
            try:
                await pool.generate("test", max_retries=1)
            except RuntimeError:
                pass

        stats = pool.get_stats()
        assert (
            stats["failing"].consecutive_failures >= 2
            or stats["failing"].cooldown_until > 0
        )

    @pytest.mark.asyncio
    async def test_rate_limit_detection(self):
        pool = ProviderPool()

        assert pool._is_rate_limit_error(RuntimeError("rate limit exceeded"))
        assert pool._is_rate_limit_error(RuntimeError("429 Too Many Requests"))
        assert pool._is_rate_limit_error(RuntimeError("quota exceeded"))
        assert not pool._is_rate_limit_error(RuntimeError("connection failed"))


class TestProviderPoolStats:
    @pytest.mark.asyncio
    async def test_stats_tracking(self):
        pool = ProviderPool()
        pool.add_provider("test_1", MockProvider(delay=0.001))

        await pool.generate("test prompt")

        stats = pool.get_stats()
        assert "test_1" in stats
        assert stats["test_1"].requests_total == 1
        assert stats["test_1"].requests_success == 1
        assert stats["test_1"].requests_failed == 0

    @pytest.mark.asyncio
    async def test_stats_failure_tracking(self):
        pool = ProviderPool()
        failing_provider = MockProvider(delay=0.001, fail_count=100)
        pool.add_provider("test_1", failing_provider)
        pool.add_provider("test_2", MockProvider(delay=0.001))

        await pool.generate("test")

        stats = pool.get_stats()
        assert stats["test_1"].requests_failed >= 1

    def test_reset_stats(self):
        pool = ProviderPool()
        pool.add_provider("test_1", MockProvider())
        pool._stats["test_1"].requests_total = 100
        pool._stats["test_1"].requests_success = 95

        pool.reset_stats()

        stats = pool.get_stats()
        assert stats["test_1"].requests_total == 0

    def test_reset_cooldowns(self):
        pool = ProviderPool()
        pool.add_provider("test_1", MockProvider())
        pool._stats["test_1"].cooldown_until = time.time() + 1000
        pool._stats["test_1"].consecutive_failures = 5

        pool.reset_cooldowns()

        stats = pool.get_stats()
        assert stats["test_1"].cooldown_until == 0.0
        assert stats["test_1"].consecutive_failures == 0


class TestProviderPoolConcurrency:
    @pytest.mark.asyncio
    async def test_concurrent_batch_faster_than_sequential(self):
        config = PoolConfig(max_concurrent_per_provider=4)
        pool = ProviderPool(config)

        for i in range(4):
            pool.add_provider(f"p{i}", MockProvider(delay=0.05))

        prompts = [f"Prompt {i}" for i in range(8)]

        start = time.time()
        await pool.generate_batch(prompts)
        elapsed = time.time() - start

        assert elapsed < 0.3

    @pytest.mark.asyncio
    async def test_max_concurrent_respected(self):
        config = PoolConfig(max_concurrent_per_provider=2)
        pool = ProviderPool(config)
        pool.add_provider("test_1", MockProvider(delay=0.05))

        prompts = [f"Prompt {i}" for i in range(4)]

        start = time.time()
        await pool.generate_batch(prompts, max_concurrent=2)
        elapsed = time.time() - start

        assert elapsed >= 0.08
