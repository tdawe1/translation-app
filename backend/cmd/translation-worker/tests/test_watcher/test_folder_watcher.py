# tests/test_watcher/test_folder_watcher.py
"""
Unit tests for folder watcher.

Tests file detection, stabilization, and lifecycle management.
"""

import pytest
import sys
import tempfile
import time
from pathlib import Path

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from watcher.folder_watcher import (
    FolderWatcher,
    FolderWatcherHandler,
    FileStatus,
    FileInfo,
)


class TestFolderWatcherHandler:
    """Test FolderWatcherHandler functionality."""

    def test_handler_initialization(self):
        """Handler should initialize with watch directory."""
        with tempfile.TemporaryDirectory() as tmpdir:
            handler = FolderWatcherHandler(
                tmpdir,
                supported_extensions=[".pdf", ".pptx"]
            )

            assert handler.watch_dir == Path(tmpdir)
            assert handler.supported_extensions == {".pdf", ".pptx"}

    def test_should_ignore_unsupported_extensions(self):
        """Should ignore files with unsupported extensions."""
        with tempfile.TemporaryDirectory() as tmpdir:
            handler = FolderWatcherHandler(
                tmpdir,
                supported_extensions=[".pdf"]
            )

            assert handler._should_ignore("/path/to/file.pdf") is False
            assert handler._should_ignore("/path/to/file.pptx") is True
            assert handler._should_ignore("/path/to/file.txt") is True

    def test_should_accept_all_when_no_extensions(self):
        """Should accept all files when no extensions filter."""
        with tempfile.TemporaryDirectory() as tmpdir:
            handler = FolderWatcherHandler(tmpdir)

            assert handler._should_ignore("/path/to/file.anything") is False

    def test_file_info_properties(self):
        """FileInfo should provide convenient properties."""
        info = FileInfo(
            path="/incoming/test_document.pdf",
            status=FileStatus.DETECTED,
            size=1024
        )

        assert info.filename == "test_document.pdf"
        assert info.extension == ".pdf"


class TestFolderWatcher:
    """Test FolderWatcher functionality."""

    def test_watcher_initialization(self):
        """Watcher should create directories on initialization."""
        with tempfile.TemporaryDirectory() as tmpdir:
            base = Path(tmpdir)

            watcher = FolderWatcher(
                incoming_dir=str(base / "incoming"),
                translated_dir=str(base / "translated"),
                failed_dir=str(base / "failed"),
                supported_extensions=[".pdf"]
            )

            assert watcher.incoming_dir.exists()
            assert watcher.translated_dir.exists()
            assert watcher.failed_dir.exists()

    def test_watcher_without_failed_dir(self):
        """Watcher should work without failed directory."""
        with tempfile.TemporaryDirectory() as tmpdir:
            base = Path(tmpdir)

            watcher = FolderWatcher(
                incoming_dir=str(base / "incoming"),
                translated_dir=str(base / "translated")
            )

            assert watcher.failed_dir is None

    def test_scan_finds_ready_files(self):
        """Should detect new files in watch directory."""
        with tempfile.TemporaryDirectory() as tmpdir:
            base = Path(tmpdir)
            incoming = base / "incoming"
            translated = base / "translated"
            incoming.mkdir()
            translated.mkdir()

            watcher = FolderWatcher(
                incoming_dir=str(incoming),
                translated_dir=str(translated),
                supported_extensions=[".pdf"],
                stabilization_interval=0.1,
                stabilization_checks=1
            )

            # Create a test file
            test_file = incoming / "test.pdf"
            test_file.write_text("test content")

            # Scan should find the file
            found = watcher.scan()
            # File needs stabilization time
            time.sleep(0.2)

            found = watcher.scan()
            assert len(found) > 0

    def test_move_to_translated(self):
        """Should move file to translated directory."""
        with tempfile.TemporaryDirectory() as tmpdir:
            base = Path(tmpdir)
            incoming = base / "incoming"
            translated = base / "translated"
            incoming.mkdir()
            translated.mkdir()

            watcher = FolderWatcher(
                incoming_dir=str(incoming),
                translated_dir=str(translated)
            )

            # Create a test file
            test_file = incoming / "test.pdf"
            test_file.write_text("test content")

            # Move to translated
            result = watcher.move_to_translated(str(test_file))

            assert Path(result).parent == translated
            assert not test_file.exists()
            assert Path(result).exists()

    def test_move_to_translated_with_new_name(self):
        """Should move file with new name."""
        with tempfile.TemporaryDirectory() as tmpdir:
            base = Path(tmpdir)
            incoming = base / "incoming"
            translated = base / "translated"
            incoming.mkdir()
            translated.mkdir()

            watcher = FolderWatcher(
                incoming_dir=str(incoming),
                translated_dir=str(translated)
            )

            test_file = incoming / "original.pdf"
            test_file.write_text("content")

            result = watcher.move_to_translated(str(test_file), "renamed.pdf")

            assert Path(result).name == "renamed.pdf"
            assert (translated / "renamed.pdf").exists()

    def test_move_to_failed(self):
        """Should move file to failed directory."""
        with tempfile.TemporaryDirectory() as tmpdir:
            base = Path(tmpdir)
            incoming = base / "incoming"
            translated = base / "translated"
            failed = base / "failed"
            incoming.mkdir()
            translated.mkdir()
            failed.mkdir()

            watcher = FolderWatcher(
                incoming_dir=str(incoming),
                translated_dir=str(translated),
                failed_dir=str(failed)
            )

            test_file = incoming / "test.pdf"
            test_file.write_text("content")

            result = watcher.move_to_failed(str(test_file))

            assert Path(result).parent == failed
            assert (failed / "test.pdf").exists()

    def test_move_to_failed_without_dir_raises(self):
        """Should raise error when failed dir not configured."""
        with tempfile.TemporaryDirectory() as tmpdir:
            base = Path(tmpdir)
            incoming = base / "incoming"
            translated = base / "translated"
            incoming.mkdir()
            translated.mkdir()

            watcher = FolderWatcher(
                incoming_dir=str(incoming),
                translated_dir=str(translated)
            )

            with pytest.raises(ValueError, match="Failed directory not configured"):
                watcher.move_to_failed("/some/file.pdf")

    def test_get_file_info(self):
        """Should return file info for tracked files."""
        with tempfile.TemporaryDirectory() as tmpdir:
            base = Path(tmpdir)
            incoming = base / "incoming"
            translated = base / "translated"
            incoming.mkdir()
            translated.mkdir()

            watcher = FolderWatcher(
                incoming_dir=str(incoming),
                translated_dir=str(translated),
                supported_extensions=[".pdf"],
                stabilization_interval=0.1,
                stabilization_checks=1
            )

            # Create and register file
            test_file = incoming / "test.pdf"
            test_file.write_text("test content")

            watcher.scan()
            time.sleep(0.2)  # Let file stabilize
            watcher.scan()

            info = watcher.get_file_info(str(test_file))
            assert info is not None
            assert info.filename == "test.pdf"

    def test_is_running_property(self):
        """Should track running state."""
        with tempfile.TemporaryDirectory() as tmpdir:
            base = Path(tmpdir)
            incoming = base / "incoming"
            translated = base / "translated"
            incoming.mkdir()
            translated.mkdir()

            watcher = FolderWatcher(
                incoming_dir=str(incoming),
                translated_dir=str(translated)
            )

            assert not watcher.is_running

            watcher.start()
            assert watcher.is_running

            watcher.stop()
            assert not watcher.is_running

    def test_context_manager(self):
        """Should work as context manager."""
        with tempfile.TemporaryDirectory() as tmpdir:
            base = Path(tmpdir)
            incoming = base / "incoming"
            translated = base / "translated"
            incoming.mkdir()
            translated.mkdir()

            with FolderWatcher(
                incoming_dir=str(incoming),
                translated_dir=str(translated)
            ) as watcher:
                assert watcher.is_running

            # Should be stopped after context
            assert not watcher.is_running
