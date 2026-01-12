# cache/__init__.py
"""
Translation cache system.

Provides file-based and Redis-based caching backends
with intelligent key generation and cache warming.
"""

from .backend import (
    CacheBackend,
    FileCacheBackend,
    RedisCacheBackend,
    create_backend,
)

from .manager import (
    CacheManager,
    create_cache_manager,
)

__all__ = [
    "CacheBackend",
    "FileCacheBackend",
    "RedisCacheBackend",
    "create_backend",
    "CacheManager",
    "create_cache_manager",
]
