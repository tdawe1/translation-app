# cache/manager.py
"""
Translation cache manager.

Provides intelligent caching with deterministic key generation,
cache warming, and multi-backend support.
"""

from typing import Optional, Dict, Any, List, Tuple
import hashlib
import json
import logging
import threading

from .backend import CacheBackend, FileCacheBackend, RedisCacheBackend, create_backend

logger = logging.getLogger(__name__)


class CacheManager:
    """Manages translation caching with intelligent invalidation.

    Example:
        >>> backend = FileCacheBackend("/tmp/cache")
        >>> manager = CacheManager(backend)
        >>> key = manager.generate_key("こんにちは", "anthropic", "claude-4")
        >>> manager.store(key, "こんにちは", "Hello", "anthropic", "claude-4")
        >>> result = manager.retrieve(key)
        >>> result["target"]
        'Hello'
    """

    # Default TTL: 30 days
    DEFAULT_TTL = 7200 * 30  # 2 hours * 30 days = 30 days in seconds

    def __init__(
        self,
        backend: CacheBackend,
        default_ttl: int = DEFAULT_TTL,
        enabled: bool = True
    ):
        """Initialize cache manager.

        Args:
            backend: Cache backend instance
            default_ttl: Default TTL in seconds (30 days)
            enabled: Whether caching is enabled
        """
        self.backend = backend
        self.default_ttl = default_ttl
        self.enabled = enabled
        self._lock = threading.Lock()
        self._stats = {
            "hits": 0,
            "misses": 0,
            "stores": 0,
            "invalidations": 0
        }

    def generate_key(
        self,
        source: str,
        provider: str,
        model: str,
        glossary_hash: str = "",
        context: Optional[Dict[str, Any]] = None
    ) -> str:
        """Generate deterministic cache key.

        The key is based on:
        - Source text (normalized)
        - Provider name
        - Model name
        - Glossary hash (to invalidate when glossary changes)
        - Context (for domain-aware caching)

        Args:
            source: Source text to translate
            provider: LLM provider name
            model: Model identifier
            glossary_hash: Hash of glossary for invalidation
            context: Optional context dict

        Returns:
            Deterministic cache key
        """
        key_data = {
            "text": source.strip(),
            "provider": provider,
            "model": model,
            "glossary": glossary_hash
        }

        # Include context in key if provided (sorted for determinism)
        if context:
            # Sort items for deterministic key generation
            key_data["context"] = sorted(context.items())

        # Hash the key data
        key_json = json.dumps(key_data, sort_keys=True)
        hash_hex = hashlib.sha256(key_json.encode()).hexdigest()[:16]
        return f"sha256:{hash_hex}"

    def generate_glossary_hash(self, glossary_data: Dict[str, Any]) -> str:
        """Generate hash for glossary to use in cache keys.

        Args:
            glossary_data: Glossary data dict

        Returns:
            Hexadecimal hash string
        """
        glossary_json = json.dumps(glossary_data, sort_keys=True)
        return hashlib.sha256(glossary_json.encode()).hexdigest()[:16]

    def store(
        self,
        cache_key: str,
        source: str,
        target: str,
        provider: str,
        model: str,
        glossary_version: str = "",
        metadata: Optional[Dict[str, Any]] = None
    ) -> bool:
        """Store translation in cache.

        Args:
            cache_key: Cache key from generate_key()
            source: Original source text
            target: Translated text
            provider: LLM provider used
            model: Model used
            glossary_version: Glossary version identifier
            metadata: Additional metadata

        Returns:
            True if stored successfully
        """
        if not self.enabled:
            return False

        value = {
            "cache_key": cache_key,
            "source": source,
            "target": target,
            "provider": provider,
            "model": model,
            "glossary_version": glossary_version,
            "metadata": metadata or {},
            "cached_at": int(__import__("time").time())
        }

        result = self.backend.set(cache_key, value, self.default_ttl)
        if result:
            with self._lock:
                self._stats["stores"] += 1
            logger.debug(f"[CACHE] Stored: {cache_key}")

        return result

    def retrieve(self, cache_key: str) -> Optional[Dict[str, Any]]:
        """Retrieve translation from cache.

        Args:
            cache_key: Cache key from generate_key()

        Returns:
            Cached value dict or None if not found/expired
        """
        if not self.enabled:
            return None

        value = self.backend.get(cache_key)

        if value is None:
            with self._lock:
                self._stats["misses"] += 1
            logger.debug(f"[CACHE] Miss: {cache_key}")
            return None

        with self._lock:
            self._stats["hits"] += 1
        logger.debug(f"[CACHE] Hit: {cache_key}")
        return value

    def invalidate(self, cache_key: str) -> bool:
        """Invalidate cached entry.

        Args:
            cache_key: Cache key to invalidate

        Returns:
            True if entry existed and was deleted
        """
        if not self.enabled:
            return False

        result = self.backend.delete(cache_key)
        if result:
            with self._lock:
                self._stats["invalidations"] += 1
            logger.debug(f"[CACHE] Invalidated: {cache_key}")

        return result

    def get_or_translate(
        self,
        source: str,
        provider: str,
        model: str,
        translate_func: callable,
        glossary_hash: str = "",
        context: Optional[Dict[str, Any]] = None
    ) -> Tuple[str, bool]:
        """Get cached translation or translate using provided function.

        This is a convenience method that handles the common pattern of:
        1. Check cache
        2. If miss, call translate function
        3. Store result in cache

        Args:
            source: Source text to translate
            provider: LLM provider name
            model: Model identifier
            translate_func: Function to call if cache miss (takes source, returns target)
            glossary_hash: Glossary hash for cache key
            context: Optional context

        Returns:
            Tuple of (translated_text, was_cached)
        """
        key = self.generate_key(source, provider, model, glossary_hash, context)
        cached = self.retrieve(key)

        if cached is not None:
            return cached["target"], True

        # Cache miss - translate
        target = translate_func(source)
        self.store(key, source, target, provider, model)

        return target, False

    def warm(
        self,
        phrases: List[Tuple[str, str]],
        provider: str,
        model: str,
        glossary_hash: str = ""
    ) -> int:
        """Warm cache with common phrases.

        Args:
            phrases: List of (source, target) tuples
            provider: LLM provider name
            model: Model identifier
            glossary_hash: Optional glossary hash

        Returns:
            Number of phrases successfully cached
        """
        count = 0
        for source, target in phrases:
            key = self.generate_key(source, provider, model, glossary_hash)
            if self.store(key, source, target, provider, model):
                count += 1

        logger.info(f"[CACHE] Warmed {count} phrases")
        return count

    def batch_get(
        self,
        cache_keys: List[str]
    ) -> Dict[str, Optional[Dict[str, Any]]]:
        """Retrieve multiple cache entries.

        Args:
            cache_keys: List of cache keys

        Returns:
            Dict mapping key to cached value (or None if not found)
        """
        result = {}
        for key in cache_keys:
            result[key] = self.retrieve(key)
        return result

    def batch_store(
        self,
        entries: List[Dict[str, Any]]
    ) -> int:
        """Store multiple cache entries.

        Args:
            entries: List of dicts with keys: cache_key, source, target, provider, model

        Returns:
            Number of entries successfully stored
        """
        count = 0
        for entry in entries:
            if self.store(
                entry["cache_key"],
                entry["source"],
                entry["target"],
                entry["provider"],
                entry.get("model", ""),
                entry.get("glossary_version", ""),
                entry.get("metadata")
            ):
                count += 1

        return count

    def invalidate_by_prefix(self, prefix: str) -> int:
        """Invalidate all cache entries with a given prefix.

        Note: This requires the backend to support listing keys.
        For file cache, this matches hashed keys that start with the prefix.

        Args:
            prefix: Key prefix to invalidate

        Returns:
            Number of entries invalidated
        """
        count = 0
        keys = self.backend.keys()

        # Filter keys by prefix (may need adjustment based on backend)
        for key in keys:
            if key.startswith(prefix):
                if self.invalidate(key):
                    count += 1

        if count > 0:
            logger.info(f"[CACHE] Invalidated {count} entries with prefix: {prefix}")

        return count

    def get_stats(self) -> Dict[str, int]:
        """Get cache statistics.

        Returns:
            Dict with hit/miss/store/invalidation counts
        """
        with self._lock:
            return self._stats.copy()

    def reset_stats(self):
        """Reset cache statistics."""
        with self._lock:
            self._stats = {
                "hits": 0,
                "misses": 0,
                "stores": 0,
                "invalidations": 0
            }

    def get_hit_rate(self) -> float:
        """Calculate cache hit rate.

        Returns:
            Hit rate as percentage (0-100)
        """
        with self._lock:
            total = self._stats["hits"] + self._stats["misses"]
            if total == 0:
                return 0.0
            return (self._stats["hits"] / total) * 100

    def clear(self) -> bool:
        """Clear all cache entries.

        Returns:
            True if successful
        """
        if not self.enabled:
            return False

        result = self.backend.clear()
        if result:
            logger.info("[CACHE] Cleared all entries")

        return result


def create_cache_manager(
    backend_type: str = "file",
    backend_config: Optional[Dict[str, Any]] = None,
    default_ttl: int = CacheManager.DEFAULT_TTL,
    enabled: bool = True
) -> CacheManager:
    """Factory function to create cache manager with backend.

    Args:
        backend_type: Type of backend ("file" or "redis")
        backend_config: Config dict for backend
        default_ttl: Default TTL in seconds
        enabled: Whether caching is enabled

    Returns:
        CacheManager instance

    Example:
        >>> manager = create_cache_manager(
        ...     backend_type="file",
        ...     backend_config={"directory": "/tmp/cache"}
        ... )
    """
    backend_config = backend_config or {}
    backend = create_backend(backend_type, **backend_config)
    return CacheManager(backend, default_ttl, enabled)
