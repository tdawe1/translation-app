# cmd/translation-worker/watcher/folder_watcher.py
"""
Folder watcher for detecting new translation files.

Uses watchdog library to monitor directories for new files.
Handles file write completion detection via size stability checking.
"""

import os
import shutil
import time
import hashlib
from pathlib import Path
from typing import Optional, Callable, Dict, Any, List
from dataclasses import dataclass, field
from enum import Enum
from threading import Lock, Thread
import logging

from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler, FileCreatedEvent, FileMovedEvent

logger = logging.getLogger(__name__)


class FileStatus(Enum):
    """Status of a detected file."""
    DETECTED = "detected"          # File first seen
    STABILIZING = "stabilizing"    # Monitoring for size stability
    READY = "ready"                # Ready for processing
    PROCESSING = "processing"      # Being processed
    COMPLETED = "completed"        # Translation complete
    FAILED = "failed"              # Processing failed


@dataclass
class FileInfo:
    """Information about a file being watched."""
    path: str
    status: FileStatus = FileStatus.DETECTED
    size: int = 0
    last_modified: float = 0.0
    stable_size_count: int = 0
    checksum: str = ""
    metadata: Dict[str, Any] = field(default_factory=dict)

    @property
    def filename(self) -> str:
        return Path(self.path).name

    @property
    def extension(self) -> str:
        return Path(self.path).suffix.lower()


class FolderWatcherHandler(FileSystemEventHandler):
    """Handler for watchdog file system events.

    Filters for supported file extensions and tracks file state.
    """

    def __init__(
        self,
        watch_dir: str,
        callback: Optional[Callable[[str], None]] = None,
        supported_extensions: Optional[List[str]] = None,
        stabilization_interval: float = 0.5,
        stabilization_checks: int = 3
    ):
        """Initialize the file system event handler.

        Args:
            watch_dir: Directory to watch
            callback: Function to call when file is ready
            supported_extensions: List of extensions to watch (e.g., [".pdf", ".pptx"])
            stabilization_interval: Seconds between size checks
            stabilization_checks: Number of consecutive stable sizes required
        """
        self.watch_dir = Path(watch_dir)
        self.callback = callback
        self.supported_extensions = set(ext.lower() for ext in (supported_extensions or []))
        self.stabilization_interval = stabilization_interval
        self.stabilization_checks = stabilization_checks
        self._files: Dict[str, FileInfo] = {}
        self._lock = Lock()

    def on_created(self, event):
        """Handle file creation event."""
        if event.is_directory:
            return

        file_path = event.src_path
        if self._should_ignore(file_path):
            return

        logger.info(f"[WATCHER] File created: {file_path}")
        self._register_file(file_path)

    def on_moved(self, event):
        """Handle file move event."""
        if event.is_directory:
            return

        dest_path = event.dest_path
        if self._should_ignore(dest_path):
            return

        logger.info(f"[WATCHER] File moved: {dest_path}")
        self._register_file(dest_path)

    def _should_ignore(self, file_path: str) -> bool:
        """Check if file should be ignored based on extension."""
        ext = Path(file_path).suffix.lower()
        if self.supported_extensions and ext not in self.supported_extensions:
            return True
        return False

    def _register_file(self, file_path: str):
        """Register a new file and begin stabilization monitoring."""
        with self._lock:
            self._files[file_path] = FileInfo(
                path=file_path,
                status=FileStatus.STABILIZING,
                size=0,
                last_modified=time.time()
            )

    def get_ready_files(self) -> List[str]:
        """Get list of files that are ready for processing.

        Returns:
            List of file paths that have stabilized
        """
        ready = []

        with self._lock:
            for file_path, info in list(self._files.items()):
                if info.status == FileStatus.DETECTED:
                    # Begin stabilization check
                    info.status = FileStatus.STABILIZING
                    info.last_modified = time.time()
                    info.stable_size_count = 0

                elif info.status == FileStatus.STABILIZING:
                    if self._check_stability(file_path, info):
                        info.status = FileStatus.READY
                        ready.append(file_path)

        return ready

    def _check_stability(self, file_path: str, info: FileInfo) -> bool:
        """Check if file write is complete by monitoring size stability.

        Args:
            file_path: Path to file
            info: FileInfo tracking this file

        Returns:
            True if file is stable (write complete)
        """
        try:
            stat = os.stat(file_path)
            current_size = stat.st_size
            current_mtime = stat.st_mtime

            # Check if file has been modified recently
            time_since_modify = time.time() - current_mtime
            if time_since_modify < self.stabilization_interval:
                # File still being written - update size for next check
                info.size = current_size
                info.last_modified = current_mtime
                return False

            # Check if size is stable
            if current_size == info.size:
                info.stable_size_count += 1
                if info.stable_size_count >= self.stabilization_checks:
                    # File is stable - compute checksum
                    info.checksum = self._compute_checksum(file_path)
                    return True
            else:
                # Size changed, reset counter and update size
                info.size = current_size
                info.stable_size_count = 0
                info.last_modified = current_mtime

            return False

        except (OSError, FileNotFoundError):
            # File was deleted or is inaccessible
            with self._lock:
                self._files.pop(file_path, None)
            return False

    def _compute_checksum(self, file_path: str) -> str:
        """Compute MD5 checksum of file.

        Args:
            file_path: Path to file

        Returns:
            Hexadecimal checksum string
        """
        md5 = hashlib.md5()
        try:
            with open(file_path, 'rb') as f:
                for chunk in iter(lambda: f.read(8192), b''):
                    md5.update(chunk)
        except (OSError, IOError):
            return ""
        return md5.hexdigest()

    def mark_status(self, file_path: str, status: FileStatus):
        """Update file status.

        Args:
            file_path: Path to file
            status: New status
        """
        with self._lock:
            if file_path in self._files:
                self._files[file_path].status = status

    def remove_file(self, file_path: str):
        """Remove file from tracking.

        Args:
            file_path: Path to file
        """
        with self._lock:
            self._files.pop(file_path, None)

    def get_file_info(self, file_path: str) -> Optional[FileInfo]:
        """Get info for a tracked file.

        Args:
            file_path: Path to file

        Returns:
            FileInfo or None if not tracked
        """
        return self._files.get(file_path)


class FolderWatcher:
    """Watches folders for new files and manages file lifecycle.

    Monitors an incoming directory for new files, detects when
    writes are complete, and manages moving files to appropriate
    output directories (translated, failed, etc.).
    """

    def __init__(
        self,
        incoming_dir: str,
        translated_dir: str,
        failed_dir: str = "",
        supported_extensions: Optional[List[str]] = None,
        callback: Optional[Callable[[str], None]] = None,
        stabilization_interval: float = 0.5,
        stabilization_checks: int = 3
    ):
        """Initialize the folder watcher.

        Args:
            incoming_dir: Directory to watch for new files
            translated_dir: Directory for completed translations
            failed_dir: Directory for failed processing (optional)
            supported_extensions: File extensions to watch (e.g., [".pdf", ".pptx"])
            callback: Optional callback for ready files
            stabilization_interval: Seconds between size checks
            stabilization_checks: Number of stable checks before ready
        """
        self.incoming_dir = Path(incoming_dir)
        self.translated_dir = Path(translated_dir)
        self.failed_dir = Path(failed_dir) if failed_dir else None

        # Ensure directories exist
        self.incoming_dir.mkdir(parents=True, exist_ok=True)
        self.translated_dir.mkdir(parents=True, exist_ok=True)
        if self.failed_dir:
            self.failed_dir.mkdir(parents=True, exist_ok=True)

        self.supported_extensions = supported_extensions
        self.callback = callback
        self.stabilization_interval = stabilization_interval
        self.stabilization_checks = stabilization_checks

        self._observer: Optional[Observer] = None
        self._handler: Optional[FolderWatcherHandler] = None
        self._scan_thread: Optional[Thread] = None
        self._running = False
        self._lock = Lock()

    def start(self) -> bool:
        """Start watching for file system events.

        Returns:
            True if started successfully
        """
        with self._lock:
            if self._running:
                return True

            self._handler = FolderWatcherHandler(
                str(self.incoming_dir),
                self.callback,
                self.supported_extensions,
                self.stabilization_interval,
                self.stabilization_checks
            )

            self._observer = Observer()
            self._observer.schedule(
                self._handler,
                str(self.incoming_dir),
                recursive=False
            )

            self._observer.start()
            self._running = True

            # Start scan thread for periodic stability checks
            self._scan_thread = Thread(target=self._scan_loop, daemon=True)
            self._scan_thread.start()

            logger.info(f"[WATCHER] Started watching: {self.incoming_dir}")
            return True

    def stop(self):
        """Stop watching for file system events."""
        with self._lock:
            if not self._running:
                return

            self._running = False

            if self._observer:
                self._observer.stop()
                self._observer.join(timeout=5)
                self._observer = None

            if self._scan_thread:
                self._scan_thread.join(timeout=5)
                self._scan_thread = None

            logger.info("[WATCHER] Stopped")

    def _scan_loop(self):
        """Background loop to check for ready files."""
        while self._running:
            try:
                if self._handler:
                    ready_files = self._handler.get_ready_files()
                    for file_path in ready_files:
                        self._process_ready_file(file_path)
            except Exception as e:
                logger.error(f"[WATCHER] Error in scan loop: {e}")

            time.sleep(self.stabilization_interval)

    def _process_ready_file(self, file_path: str):
        """Process a file that is ready for translation.

        Args:
            file_path: Path to ready file
        """
        logger.info(f"[WATCHER] Ready file: {file_path}")
        self._handler.mark_status(file_path, FileStatus.READY)

        if self.callback:
            try:
                self.callback(file_path)
            except Exception as e:
                logger.error(f"[WATCHER] Callback error: {e}")

    def scan(self) -> List[str]:
        """Scan for ready files (one-time check).

        Returns:
            List of file paths that are ready
        """
        # Initialize handler if not already done
        if not self._handler:
            self._handler = FolderWatcherHandler(
                str(self.incoming_dir),
                self.callback,
                self.supported_extensions,
                self.stabilization_interval,
                self.stabilization_checks
            )

        # First, scan for any existing files in incoming dir
        for file_path in self.incoming_dir.iterdir():
            if file_path.is_file():
                ext = file_path.suffix.lower()
                if self.supported_extensions and ext not in self.supported_extensions:
                    continue
                if str(file_path) not in self._handler._files:
                    self._handler._register_file(str(file_path))

        return self._handler.get_ready_files()

    def move_to_translated(self, file_path: str, new_filename: Optional[str] = None) -> str:
        """Move file to translated directory.

        Args:
            file_path: Path to source file
            new_filename: Optional new filename (default: keep original)

        Returns:
            Path to moved file
        """
        source = Path(file_path)
        if new_filename:
            dest = self.translated_dir / new_filename
        else:
            dest = self.translated_dir / source.name

        shutil.move(str(source), str(dest))
        logger.info(f"[WATCHER] Moved to translated: {dest}")

        if self._handler:
            self._handler.mark_status(file_path, FileStatus.COMPLETED)
            self._handler.remove_file(file_path)

        return str(dest)

    def move_to_failed(self, file_path: str, new_filename: Optional[str] = None) -> str:
        """Move file to failed directory.

        Args:
            file_path: Path to source file
            new_filename: Optional new filename (default: keep original)

        Returns:
            Path to moved file

        Raises:
            ValueError if failed_dir not configured
        """
        if not self.failed_dir:
            raise ValueError("Failed directory not configured")

        source = Path(file_path)
        if new_filename:
            dest = self.failed_dir / new_filename
        else:
            dest = self.failed_dir / source.name

        shutil.move(str(source), str(dest))
        logger.info(f"[WATCHER] Moved to failed: {dest}")

        if self._handler:
            self._handler.mark_status(file_path, FileStatus.FAILED)
            self._handler.remove_file(file_path)

        return str(dest)

    def get_file_info(self, file_path: str) -> Optional[FileInfo]:
        """Get information about a tracked file.

        Args:
            file_path: Path to file

        Returns:
            FileInfo or None if not tracked
        """
        if self._handler:
            return self._handler.get_file_info(file_path)
        return None

    @property
    def is_running(self) -> bool:
        """Check if watcher is running."""
        return self._running

    def __enter__(self):
        """Context manager entry."""
        self.start()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.stop()
