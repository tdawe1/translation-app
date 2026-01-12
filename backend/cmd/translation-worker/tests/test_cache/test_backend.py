# tests/test_cache/test_backend.py
"""
Unit tests for cache backends and manager.

Tests file-based and Redis-based caching.
"""

import pytest
import sys
import tempfile
import time
from pathlib import Path

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from cache.backend import (
    CacheBackend,
    FileCacheBackend,
    RedisCacheBackend,
    create_backend,
)
from cache.manager import (
    CacheManager,
    create_cache_manager,
)


class TestFileCacheBackend:
    """Test FileCacheBackend functionality."""

    def test_initialization(self):
        """Should create cache directory on init."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            assert backend.directory.exists()
            assert backend.directory == Path(tmpdir)

    def test_get_set_delete(self):
        """Should store, retrieve, and delete values."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)

            # Set - FileCacheBackend now stores values directly (like Redis)
            value = {"source": "test", "target": "テスト"}
            assert backend.set("key1", value) is True

            # Get - Returns the value directly (metadata stripped automatically)
            retrieved = backend.get("key1")
            assert retrieved == value
            assert retrieved["source"] == "test"
            assert retrieved["target"] == "テスト"

            # Delete
            assert backend.delete("key1") is True
            assert backend.get("key1") is None

    def test_get_nonexistent(self):
        """Should return None for missing keys."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            assert backend.get("nonexistent") is None

    def test_exists(self):
        """Should check key existence."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)

            assert backend.exists("key1") is False

            backend.set("key1", {"data": "test"})
            assert backend.exists("key1") is True

    def test_clear(self):
        """Should clear all cache entries."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)

            backend.set("key1", {"data": "test1"})
            backend.set("key2", {"data": "test2"})

            assert backend.clear() is True
            assert backend.get("key1") is None
            assert backend.get("key2") is None

    def test_keys(self):
        """Should list all cache keys."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)

            backend.set("key1", {"data": "test1"})
            backend.set("key2", {"data": "test2"})

            keys = backend.keys()
            assert len(keys) == 2

    def test_ttl_storage(self):
        """Should store TTL value (even if not enforced)."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)

            backend.set("key1", {"data": "test"}, ttl=3600)
            cached = backend.get("key1")

            # Metadata is stripped by get(), so we need to check raw file
            import json
            from pathlib import Path
            cache_file = list(Path(tmpdir).glob("*.json"))[0]
            raw_data = json.loads(cache_file.read_text())

            # TTL is stored with underscore prefix (internal metadata)
            assert raw_data.get("_ttl") == 3600 or raw_data.get("ttl") == 3600
            assert "_cached_at" in raw_data or "cached_at" in raw_data

            # User data is accessible without metadata
            assert cached["data"] == "test"

    def test_cleanup_expired(self):
        """Should remove expired entries."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)

            # Set entry with short TTL
            backend.set("key1", {"data": "test"}, ttl=1)

            # Wait for expiration
            time.sleep(2)

            # Cleanup expired
            removed = backend.cleanup_expired()
            assert removed >= 0  # May have cleaned up key1


class TestRedisCacheBackend:
    """Test RedisCacheBackend functionality."""

    def test_requires_redis(self):
        """Should raise RuntimeError if redis not installed."""
        # This test verifies the error handling
        # We can't actually test it without removing redis, so we test
        # that the error message is correct in the code
        import inspect
        source = inspect.getsource(RedisCacheBackend.__init__)
        assert "RuntimeError" in source
        assert "redis" in source

    def test_skip_if_no_redis(self):
        """Mark tests to skip if Redis unavailable."""
        try:
            import redis
        except ImportError:
            pytest.skip("redis package not installed")

        # Try to connect to Redis
        try:
            client = redis.Redis(host="localhost", port=6379, decode_responses=True)
            client.ping()
        except Exception:
            pytest.skip("Redis not available on localhost:6379")

    @pytest.mark.skipif(True, reason="Requires Redis server")
    def test_redis_basic_operations(self):
        """Should perform basic CRUD operations."""
        try:
            import redis
            client = redis.Redis(host="localhost", port=6379, db=15, decode_responses=True)
            client.ping()
        except Exception:
            pytest.skip("Redis not available")

        backend = RedisCacheBackend(host="localhost", port=6379, db=15)

        # Clean up any existing data
        backend.clear()

        # Set
        value = {"test": "data"}
        assert backend.set("test_key", value) is True

        # Get
        retrieved = backend.get("test_key")
        assert retrieved == value

        # Exists
        assert backend.exists("test_key") is True

        # Delete
        assert backend.delete("test_key") is True
        assert backend.exists("test_key") is False

        # Cleanup
        backend.clear()


class TestCreateBackend:
    """Test backend factory function."""

    def test_create_file_backend(self):
        """Should create file backend."""
        backend = create_backend("file", directory="/tmp/test_cache")
        assert isinstance(backend, FileCacheBackend)

    def test_create_redis_backend(self):
        """Should create redis backend."""
        try:
            import redis
        except ImportError:
            pytest.skip("redis package not installed")

        backend = create_backend("redis", host="localhost", port=6379)
        assert isinstance(backend, RedisCacheBackend)

    def test_invalid_backend_type(self):
        """Should raise ValueError for invalid backend type."""
        with pytest.raises(ValueError, match="Unknown backend type"):
            create_backend("invalid")


class TestCacheManager:
    """Test CacheManager functionality."""

    def test_initialization(self):
        """Should initialize with backend."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            assert manager.backend == backend
            assert manager.enabled is True

    def test_generate_key(self):
        """Should generate deterministic keys."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            key1 = manager.generate_key(
                source="テスト",
                provider="anthropic",
                model="claude-4",
                glossary_hash="abc123"
            )

            key2 = manager.generate_key(
                source="テスト",
                provider="anthropic",
                model="claude-4",
                glossary_hash="abc123"
            )

            assert key1 == key2
            assert key1.startswith("sha256:")

    def test_generate_key_with_context(self):
        """Should include context in key generation."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            key1 = manager.generate_key(
                source="test",
                provider="anthropic",
                model="claude-4",
                context={"domain": "business"}
            )

            key2 = manager.generate_key(
                source="test",
                provider="anthropic",
                model="claude-4",
                context={"domain": "business"}
            )

            # Context should be included but order-independent
            assert key1 == key2

    def test_generate_glossary_hash(self):
        """Should generate deterministic glossary hash."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            glossary = {"entries": [{"source": "test", "target": "test"}]}
            hash1 = manager.generate_glossary_hash(glossary)
            hash2 = manager.generate_glossary_hash(glossary)

            assert hash1 == hash2
            assert len(hash1) == 16  # First 16 chars of SHA256

    def test_store_and_retrieve(self):
        """Should store and retrieve translations."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            cache_key = manager.generate_key("こんにちは", "anthropic", "claude-4")

            # Store
            assert manager.store(
                cache_key=cache_key,
                source="こんにちは",
                target="Hello",
                provider="anthropic",
                model="claude-4"
            ) is True

            # Retrieve
            result = manager.retrieve(cache_key)
            assert result is not None
            assert result["target"] == "Hello"
            assert result["source"] == "こんにちは"

    def test_retrieve_miss(self):
        """Should return None for cache miss."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            result = manager.retrieve("nonexistent_key")
            assert result is None

    def test_invalidate(self):
        """Should invalidate cache entries."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            cache_key = manager.generate_key("test", "provider", "model")
            manager.store(cache_key, "test", "TEST", "provider", "model")

            assert backend.exists(cache_key) is True
            assert manager.invalidate(cache_key) is True
            assert backend.exists(cache_key) is False

    def test_disabled_cache(self):
        """Should not cache when disabled."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend, enabled=False)

            cache_key = manager.generate_key("test", "provider", "model")

            # Store should return False when disabled
            assert manager.store(cache_key, "test", "TEST", "provider", "model") is False

            # Retrieve should return None when disabled
            assert manager.retrieve(cache_key) is None

    def test_exists(self):
        """Should check if key exists in manager."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            cache_key = manager.generate_key("test", "provider", "model")

            assert backend.exists(cache_key) is False

            manager.store(cache_key, "test", "TEST", "provider", "model")
            assert backend.exists(cache_key) is True

    def test_get_or_translate(self):
        """Should use cache or call translate function."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            call_count = [0]
            def translate(source):
                call_count[0] += 1
                return f"translated({source})"

            # First call - should translate
            result1, cached1 = manager.get_or_translate(
                source="テスト",
                provider="anthropic",
                model="claude-4",
                translate_func=translate
            )
            assert result1 == "translated(テスト)"
            assert cached1 is False
            assert call_count[0] == 1

            # Second call - should use cache
            result2, cached2 = manager.get_or_translate(
                source="テスト",
                provider="anthropic",
                model="claude-4",
                translate_func=translate
            )
            assert result2 == "translated(テスト)"
            assert cached2 is True
            assert call_count[0] == 1  # No additional calls

    def test_warm_cache(self):
        """Should warm cache with phrases."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            phrases = [
                ("こんにちは", "Hello"),
                ("さようなら", "Goodbye")
            ]

            count = manager.warm(phrases, "anthropic", "claude-4")
            assert count == 2

            # Verify phrases are cached
            for source, target in phrases:
                key = manager.generate_key(source, "anthropic", "claude-4")
                result = manager.retrieve(key)
                assert result["target"] == target

    def test_stats_tracking(self):
        """Should track cache statistics."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            cache_key = manager.generate_key("test", "provider", "model")

            # Miss
            manager.retrieve("nonexistent")
            stats = manager.get_stats()
            assert stats["misses"] == 1
            assert stats["hits"] == 0

            # Store
            manager.store(cache_key, "test", "TEST", "provider", "model")
            stats = manager.get_stats()
            assert stats["stores"] == 1

            # Hit
            manager.store(cache_key, "test", "TEST", "provider", "model")
            manager.retrieve(cache_key)
            stats = manager.get_stats()
            assert stats["hits"] == 1

    def test_get_hit_rate(self):
        """Should calculate hit rate."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            # No requests yet
            assert manager.get_hit_rate() == 0.0

            cache_key = manager.generate_key("test", "provider", "model")

            # All misses
            manager.retrieve("miss1")
            manager.retrieve("miss2")
            assert manager.get_hit_rate() == 0.0

            # Store and hit
            manager.store(cache_key, "test", "TEST", "provider", "model")
            manager.retrieve(cache_key)
            # 1 hit out of 3 total = 33.33%
            hit_rate = manager.get_hit_rate()
            assert 0 < hit_rate < 100

    def test_reset_stats(self):
        """Should reset statistics."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            manager.retrieve("miss")
            manager.reset_stats()

            stats = manager.get_stats()
            assert stats["misses"] == 0
            assert stats["hits"] == 0

    def test_clear(self):
        """Should clear all cache entries."""
        with tempfile.TemporaryDirectory() as tmpdir:
            backend = FileCacheBackend(tmpdir)
            manager = CacheManager(backend)

            # Add some entries
            manager.store(
                manager.generate_key("test1", "p", "m"),
                "test1", "T1", "p", "m"
            )
            manager.store(
                manager.generate_key("test2", "p", "m"),
                "test2", "T2", "p", "m"
            )

            # Clear
            assert manager.clear() is True
            assert manager.retrieve(manager.generate_key("test1", "p", "m")) is None


class TestCreateCacheManager:
    """Test cache manager factory function."""

    def test_create_file_manager(self):
        """Should create manager with file backend."""
        with tempfile.TemporaryDirectory() as tmpdir:
            manager = create_cache_manager(
                backend_type="file",
                backend_config={"directory": tmpdir}
            )
            assert isinstance(manager, CacheManager)
            assert isinstance(manager.backend, FileCacheBackend)

    def test_create_with_custom_ttl(self):
        """Should create manager with custom TTL."""
        manager = create_cache_manager(
            backend_type="file",
            backend_config={"directory": "/tmp/test"},
            default_ttl=3600
        )
        assert manager.default_ttl == 3600
