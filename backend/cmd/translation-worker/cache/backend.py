# cache/backend.py
"""
Cache backends for translation caching.

Provides abstract interface and implementations for
file-based and Redis-based caching.
"""

from abc import ABC, abstractmethod
from typing import Optional, Dict, Any, List
import json
import hashlib
import logging
from pathlib import Path

logger = logging.getLogger(__name__)


class CacheBackend(ABC):
    """Abstract cache backend interface.

    All cache backends must implement these methods.
    """

    @abstractmethod
    def get(self, key: str) -> Optional[Dict[str, Any]]:
        """Retrieve cached value by key.

        Args:
            key: Cache key

        Returns:
            Cached value dict or None if not found
        """
        pass

    @abstractmethod
    def set(self, key: str, value: Dict[str, Any], ttl: int = 0) -> bool:
        """Store value with TTL in seconds.

        Args:
            key: Cache key
            value: Value to store (must be JSON-serializable)
            ttl: Time to live in seconds (0 = no expiration)

        Returns:
            True if successful, False otherwise
        """
        pass

    @abstractmethod
    def delete(self, key: str) -> bool:
        """Delete cached value.

        Args:
            key: Cache key

        Returns:
            True if key existed and was deleted, False otherwise
        """
        pass

    @abstractmethod
    def exists(self, key: str) -> bool:
        """Check if key exists in cache.

        Args:
            key: Cache key

        Returns:
            True if key exists, False otherwise
        """
        pass

    @abstractmethod
    def clear(self) -> bool:
        """Clear all cached values.

        Returns:
            True if successful, False otherwise
        """
        pass

    @abstractmethod
    def keys(self) -> List[str]:
        """Get all cache keys.

        Returns:
            List of cache keys
        """
        pass


class FileCacheBackend(CacheBackend):
    """File-based cache backend using JSON sidecar files.

    Stores each cache entry as a separate JSON file.
    Suitable for single-process deployments and development.
    """

    def __init__(self, directory: str):
        """Initialize file cache backend.

        Args:
            directory: Directory to store cache files
        """
        self.directory = Path(directory)
        self.directory.mkdir(parents=True, exist_ok=True)
        logger.info(f"[FILE_CACHE] Initialized with directory: {self.directory}")

    def _get_file_path(self, key: str) -> Path:
        """Get file path for cache key.

        Uses SHA256 hash for safe filenames (avoids issues with
        special characters in keys).

        Args:
            key: Cache key

        Returns:
            Path to cache file
        """
        # Use SHA256 hash for safe filenames
        safe_key = hashlib.sha256(key.encode()).hexdigest()[:16]
        return self.directory / f"{safe_key}.json"

    def get(self, key: str) -> Optional[Dict[str, Any]]:
        """Retrieve cached value from file.

        Returns the cached value directly, or None if not found/expired.
        Checks expiration if TTL was set during storage.
        """
        file_path = self._get_file_path(key)
        if not file_path.exists():
            return None

        try:
            with open(file_path, "r", encoding="utf-8") as f:
                data = json.load(f)

            # Check if expired (if TTL metadata exists)
            ttl = data.get("_ttl", 0)
            cached_at = data.get("_cached_at", 0)
            if ttl > 0 and cached_at > 0:
                import time
                if int(time.time()) - cached_at > ttl:
                    # Expired - delete and return None
                    file_path.unlink()
                    return None

            # Return the value, stripping metadata fields if present
            # Only unwrap if it looks like old format (has "data" + "ttl"/"cached_at")
            # New format uses underscore-prefixed metadata keys
            has_old_format_wrapping = (
                "data" in data and
                ("ttl" in data or "cached_at" in data)
            )
            if has_old_format_wrapping:
                return data["data"]
            # Strip internal metadata fields (new format)
            return {k: v for k, v in data.items() if not k.startswith("_")}
        except (json.JSONDecodeError, IOError) as e:
            logger.warning(f"[FILE_CACHE] Failed to read {file_path}: {e}")
            return None

    def set(self, key: str, value: Dict[str, Any], ttl: int = 0) -> bool:
        """Store value to file.

        Note: File backend doesn't enforce TTL at read time (except via cleanup_expired()),
        but stores it in metadata for potential cleanup. For real TTL enforcement,
        use RedisCacheBackend.
        """
        file_path = self._get_file_path(key)
        try:
            import time
            # Store value with internal metadata (keys prefixed with _)
            cache_value = dict(value)  # Copy to avoid mutating input
            cache_value["_ttl"] = ttl
            cache_value["_cached_at"] = int(time.time())
            with open(file_path, "w", encoding="utf-8") as f:
                json.dump(cache_value, f, ensure_ascii=False, indent=2)
            return True
        except IOError as e:
            logger.warning(f"[FILE_CACHE] Failed to write {file_path}: {e}")
            return False

    def delete(self, key: str) -> bool:
        """Delete cached file."""
        file_path = self._get_file_path(key)
        if file_path.exists():
            try:
                file_path.unlink()
                return True
            except OSError as e:
                logger.warning(f"[FILE_CACHE] Failed to delete {file_path}: {e}")
        return False

    def exists(self, key: str) -> bool:
        """Check if cache file exists."""
        return self._get_file_path(key).exists()

    def clear(self) -> bool:
        """Clear all cache files."""
        try:
            for file_path in self.directory.glob("*.json"):
                file_path.unlink()
            return True
        except OSError as e:
            logger.warning(f"[FILE_CACHE] Failed to clear directory: {e}")
            return False

    def keys(self) -> List[str]:
        """Get all cache keys from files.

        Note: This returns hashed keys, not original keys.
        For a production system, you'd want to store the key mapping.
        """
        return [f.stem for f in self.directory.glob("*.json")]

    def cleanup_expired(self) -> int:
        """Remove expired cache entries based on TTL.

        Returns:
            Number of entries removed
        """
        import time

        count = 0
        current_time = int(time.time())

        for file_path in self.directory.glob("*.json"):
            try:
                with open(file_path, "r", encoding="utf-8") as f:
                    cache_data = json.load(f)

                # Check if expired (new format uses _ttl and _cached_at)
                ttl = cache_data.get("_ttl", cache_data.get("ttl", 0))
                cached_at = cache_data.get("_cached_at", cache_data.get("cached_at", 0))

                if ttl > 0 and (current_time - cached_at) > ttl:
                    file_path.unlink()
                    count += 1
            except (json.JSONDecodeError, OSError):
                # Invalid cache file, remove it
                try:
                    file_path.unlink()
                    count += 1
                except OSError:
                    pass

        return count


class RedisCacheBackend(CacheBackend):
    """Redis-based cache backend for distributed caching.

    Suitable for multi-process deployments and production use.
    Requires redis package.
    """

    def __init__(
        self,
        host: str = "localhost",
        port: int = 6379,
        db: int = 0,
        password: Optional[str] = None,
        key_prefix: str = "trans:",
        decode_responses: bool = True
    ):
        """Initialize Redis cache backend.

        Args:
            host: Redis host
            port: Redis port
            db: Redis database number
            password: Optional Redis password
            key_prefix: Prefix for all cache keys
            decode_responses: Whether to decode responses to strings

        Raises:
            RuntimeError: If redis package not installed
            ConnectionError: If cannot connect to Redis
        """
        try:
            import redis
        except ImportError as e:
            raise RuntimeError("redis package not installed. Install with: pip install redis") from e

        self.key_prefix = key_prefix

        try:
            self.client = redis.Redis(
                host=host,
                port=port,
                db=db,
                password=password,
                decode_responses=decode_responses,
                socket_connect_timeout=5,
                socket_timeout=5
            )
            # Test connection
            self.client.ping()
            logger.info(f"[REDIS_CACHE] Connected to {host}:{port}/{db}")
        except Exception as e:
            logger.error(f"[REDIS_CACHE] Failed to connect: {e}")
            raise

    def _make_key(self, key: str) -> str:
        """Add prefix to cache key."""
        return f"{self.key_prefix}{key}"

    def get(self, key: str) -> Optional[Dict[str, Any]]:
        """Retrieve cached value from Redis."""
        redis_key = self._make_key(key)
        try:
            value = self.client.get(redis_key)
            if value is None:
                return None
            return json.loads(value)
        except (json.JSONDecodeError, AttributeError) as e:
            logger.warning(f"[REDIS_CACHE] Failed to decode value for {redis_key}: {e}")
            return None

    def set(self, key: str, value: Dict[str, Any], ttl: int = 0) -> bool:
        """Store value in Redis."""
        redis_key = self._make_key(key)
        try:
            serialized = json.dumps(value, ensure_ascii=False)
            if ttl > 0:
                return bool(self.client.setex(redis_key, ttl, serialized))
            else:
                return bool(self.client.set(redis_key, serialized))
        except Exception as e:
            logger.warning(f"[REDIS_CACHE] Failed to set {redis_key}: {e}")
            return False

    def delete(self, key: str) -> bool:
        """Delete cached value from Redis."""
        redis_key = self._make_key(key)
        try:
            return bool(self.client.delete(redis_key))
        except Exception as e:
            logger.warning(f"[REDIS_CACHE] Failed to delete {redis_key}: {e}")
            return False

    def exists(self, key: str) -> bool:
        """Check if key exists in Redis."""
        redis_key = self._make_key(key)
        try:
            return bool(self.client.exists(redis_key))
        except Exception as e:
            logger.debug(f"[REDIS_CACHE] exists() failed for {redis_key}: {e}")
            return False

    def clear(self) -> bool:
        """Clear all keys with the configured prefix."""
        try:
            pattern = f"{self.key_prefix}*"
            keys = list(self.client.scan_iter(match=pattern, count=1000))
            if keys:
                return bool(self.client.delete(*keys))
            return True
        except Exception as e:
            logger.warning(f"[REDIS_CACHE] Failed to clear: {e}")
            return False

    def keys(self) -> List[str]:
        """Get all cache keys with the configured prefix."""
        try:
            pattern = f"{self.key_prefix}*"
            keys = list(self.client.scan_iter(match=pattern, count=1000))
            # Remove prefix to return clean keys
            prefix_len = len(self.key_prefix)
            return [k[prefix_len:] for k in keys]
        except Exception as e:
            logger.warning(f"[REDIS_CACHE] Failed to list keys: {e}")
            return []

    def ping(self) -> bool:
        """Check if Redis connection is alive."""
        try:
            return bool(self.client.ping())
        except Exception as e:
            logger.debug(f"[REDIS_CACHE] ping() failed: {e}")
            return False


def create_backend(
    backend_type: str = "file",
    **kwargs
) -> CacheBackend:
    """Factory function to create cache backend.

    Args:
        backend_type: Type of backend ("file" or "redis")
        **kwargs: Additional arguments passed to backend constructor

    Returns:
        CacheBackend instance

    Raises:
        ValueError: If backend_type is unknown
    """
    if backend_type == "file":
        directory = kwargs.get("directory", "/tmp/translation_cache")
        return FileCacheBackend(directory)
    elif backend_type == "redis":
        return RedisCacheBackend(
            host=kwargs.get("host", "localhost"),
            port=kwargs.get("port", 6379),
            db=kwargs.get("db", 0),
            password=kwargs.get("password"),
            key_prefix=kwargs.get("key_prefix", "trans:")
        )
    else:
        raise ValueError(f"Unknown backend type: {backend_type}")
