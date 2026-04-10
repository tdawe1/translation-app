import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from parsers.base import BaseParser, FilePathValidationError


class DummyParser(BaseParser):
    def supported_extensions(self):
        return [".txt"]

    def parse(self, file_path: str):
        raise NotImplementedError

    def render(self, doc, output_path: str, template_path: str | None = None):
        raise NotImplementedError


def test_validate_file_path_accepts_files_within_allowed_directory(monkeypatch, tmp_path):
    allowed_dir = tmp_path / "allowed"
    allowed_dir.mkdir()
    source_file = allowed_dir / "inside.txt"
    source_file.write_text("hello", encoding="utf-8")

    monkeypatch.setenv("WATCH_INCOMING_DIR", str(allowed_dir))

    parser = DummyParser()

    assert parser.validate_file_path(str(source_file)) == source_file.resolve()


def test_validate_file_path_raises_specific_error_for_disallowed_path(
    monkeypatch, tmp_path
):
    allowed_dir = tmp_path / "allowed"
    allowed_dir.mkdir()
    outside_dir = tmp_path / "outside"
    outside_dir.mkdir()
    source_file = outside_dir / "outside.txt"
    source_file.write_text("hello", encoding="utf-8")

    monkeypatch.setenv("WATCH_INCOMING_DIR", str(allowed_dir))

    parser = DummyParser()

    with pytest.raises(FilePathValidationError) as exc_info:
        parser.validate_file_path(str(source_file))

    message = str(exc_info.value)
    assert str(source_file.resolve()) in message
    assert str(allowed_dir.resolve()) in message
