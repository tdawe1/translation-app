# cmd/translation-worker/watcher/__init__.py
"""
Folder watching for Gengo downloads.

Uses watchdog library for cross-platform file system monitoring.
"""

from .folder_watcher import (
    FolderWatcher,
    FolderWatcherHandler,
    FileInfo,
)

__all__ = [
    "FolderWatcher",
    "FolderWatcherHandler",
    "FileInfo",
]
