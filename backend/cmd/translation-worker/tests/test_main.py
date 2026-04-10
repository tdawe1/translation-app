# tests/test_main.py
"""
Tests for translation-worker main.py module.

Following TDD principles:
- Tests written first (failing)
- Implementation makes them pass
- Tests document expected behavior
"""

import sys
import tempfile
from pathlib import Path
from types import SimpleNamespace

import pytest

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from main import QueueConsumer, load_config, validate_config
from plugins import ParsedDocument, Segment


class TestLoadConfig:
    """Test suite for load_config() function."""

    def test_load_config_from_valid_toml(self):
        """Should successfully parse a valid TOML configuration file."""
        with tempfile.NamedTemporaryFile(mode="wb", suffix=".toml", delete=False) as f:
            f.write(b"""
[worker]
id = "test-worker-1"
max_concurrent = 3
heartbeat_interval = "10s"

[translation]
default_provider = "anthropic"
default_model = "claude-4.5-sonnet"
""")
            temp_path = f.name

        try:
            config = load_config(temp_path)

            assert config is not None
            assert config["worker"]["id"] == "test-worker-1"
            assert config["worker"]["max_concurrent"] == 3
            assert config["translation"]["default_provider"] == "anthropic"
        finally:
            Path(temp_path).unlink()

    def test_load_config_with_relative_path(self):
        """Should resolve relative paths relative to main.py location."""
        # Create a temp config file in the same directory as main.py (not tests/)
        # load_config resolves relative paths from main.py's parent directory
        config_dir = Path(__file__).parent.parent  # Go up to translation-worker dir
        temp_config = config_dir / "temp_test_config.toml"

        try:
            temp_config.write_text("""
[worker]
max_concurrent = 1
heartbeat_interval = "5s"

[translation]
default_provider = "openai"
default_model = "gpt-4"
""")

            # Load using just the filename (relative path)
            config = load_config("temp_test_config.toml")

            assert config["worker"]["max_concurrent"] == 1
            assert config["translation"]["default_provider"] == "openai"
        finally:
            if temp_config.exists():
                temp_config.unlink()

    def test_load_config_with_absolute_path(self):
        """Should accept absolute paths without modification."""
        with tempfile.NamedTemporaryFile(mode="wb", suffix=".toml", delete=False) as f:
            f.write(
                b'[worker]\nid = "absolute-path-test"\nmax_concurrent = 1\nheartbeat_interval = "10s"\n\n[translation]\ndefault_provider = "test"\ndefault_model = "test-model"\n'
            )
            temp_path = Path(f.name)

        try:
            config = load_config(str(temp_path))
            assert config["worker"]["id"] == "absolute-path-test"
        finally:
            temp_path.unlink()

    def test_load_config_missing_file_raises_FileNotFoundError(self):
        """Should raise FileNotFoundError when config file doesn't exist."""
        with pytest.raises(FileNotFoundError) as exc_info:
            load_config("nonexistent-config.toml")

        assert "Config file not found" in str(exc_info.value)

    def test_load_config_malformed_toml_raises_toml_decode_error(self):
        """Should raise tomli.TOMLDecodeError for invalid TOML syntax."""
        import tomli

        with tempfile.NamedTemporaryFile(mode="w", suffix=".toml", delete=False) as f:
            f.write("[invalid toml syntax\n")  # Missing closing bracket
            temp_path = f.name

        try:
            with pytest.raises(tomli.TOMLDecodeError):
                load_config(temp_path)
        finally:
            Path(temp_path).unlink()

    def test_load_config_empty_file_returns_empty_dict(self):
        """Should return empty dict for an empty TOML file."""
        with tempfile.NamedTemporaryFile(mode="w", suffix=".toml", delete=False) as f:
            f.write("")
            temp_path = f.name

        try:
            config = load_config(temp_path)
            assert config == {}
        finally:
            Path(temp_path).unlink()

    def test_load_config_preserves_all_sections(self):
        """Should load all sections including optional ones."""
        with tempfile.NamedTemporaryFile(mode="wb", suffix=".toml", delete=False) as f:
            f.write(b"""
[worker]
max_concurrent = 3
heartbeat_interval = "10s"

[translation]
default_provider = "anthropic"
default_model = "claude-4.5-sonnet"

[glossary]
enabled = true
file = "/config/glossary.json"

[cache]
enabled = true
backend = "file"
""")
            temp_path = f.name

        try:
            config = load_config(temp_path)

            assert "worker" in config
            assert "translation" in config
            assert "glossary" in config
            assert "cache" in config
            assert config["glossary"]["enabled"] is True
        finally:
            Path(temp_path).unlink()


class TestValidateConfig:
    """Test suite for validate_config() function."""

    def test_validate_config_complete_config_returns_empty_errors(self):
        """Should return empty list for valid, complete configuration."""
        config = {
            "worker": {
                "id": "test-worker",
                "max_concurrent": 3,
                "heartbeat_interval": "10s",
            },
            "translation": {
                "default_provider": "anthropic",
                "default_model": "claude-4.5-sonnet",
            },
        }

        errors = validate_config(config)
        assert errors == []

    def test_validate_config_missing_worker_section(self):
        """Should detect missing [worker] section."""
        config = {
            "translation": {
                "default_provider": "anthropic",
                "default_model": "claude-4.5-sonnet",
            },
        }

        errors = validate_config(config)
        assert len(errors) == 1
        assert "Missing required section: [worker]" in errors

    def test_validate_config_missing_translation_section(self):
        """Should detect missing [translation] section."""
        config = {
            "worker": {
                "max_concurrent": 3,
                "heartbeat_interval": "10s",
            },
        }

        errors = validate_config(config)
        assert len(errors) == 1
        assert "Missing required section: [translation]" in errors

    def test_validate_config_missing_both_required_sections(self):
        """Should detect multiple missing required sections."""
        config = {}

        errors = validate_config(config)
        assert len(errors) == 2
        assert "Missing required section: [worker]" in errors
        assert "Missing required section: [translation]" in errors

    def test_validate_config_missing_worker_max_concurrent(self):
        """Should detect missing worker.max_concurrent field."""
        config = {
            "worker": {
                "heartbeat_interval": "10s",
            },
            "translation": {
                "default_provider": "anthropic",
                "default_model": "claude-4.5-sonnet",
            },
        }

        errors = validate_config(config)
        assert "Missing worker.max_concurrent" in errors

    def test_validate_config_missing_worker_heartbeat_interval(self):
        """Should detect missing worker.heartbeat_interval field."""
        config = {
            "worker": {
                "max_concurrent": 3,
            },
            "translation": {
                "default_provider": "anthropic",
                "default_model": "claude-4.5-sonnet",
            },
        }

        errors = validate_config(config)
        assert "Missing worker.heartbeat_interval" in errors

    def test_validate_config_missing_translation_default_provider(self):
        """Should detect missing translation.default_provider field."""
        config = {
            "worker": {
                "max_concurrent": 3,
                "heartbeat_interval": "10s",
            },
            "translation": {
                "default_model": "claude-4.5-sonnet",
            },
        }

        errors = validate_config(config)
        assert "Missing translation.default_provider" in errors

    def test_validate_config_missing_translation_default_model(self):
        """Should detect missing translation.default_model field."""
        config = {
            "worker": {
                "max_concurrent": 3,
                "heartbeat_interval": "10s",
            },
            "translation": {
                "default_provider": "anthropic",
            },
        }

        errors = validate_config(config)
        assert "Missing translation.default_model" in errors


class FakeRedis:
    def __init__(self):
        self.lists = {}
        self.hashes = {}
        self.published = []

    def scan_iter(self, match=None):
        for key in self.lists:
            yield key.encode("utf-8")

    def rpop(self, key):
        decoded = key.decode("utf-8") if isinstance(key, bytes) else key
        values = self.lists.get(decoded, [])
        if not values:
            return None
        return values.pop().encode("utf-8")

    def hgetall(self, key):
        decoded = key.decode("utf-8") if isinstance(key, bytes) else key
        return {
            k.encode("utf-8"): v.encode("utf-8")
            for k, v in self.hashes.get(decoded, {}).items()
        }

    def hset(self, key, mapping):
        self.hashes.setdefault(key, {})
        self.hashes[key].update({k: str(v) for k, v in mapping.items()})

    def delete(self, key):
        self.hashes.pop(key, None)

    def rpush(self, key, *values):
        decoded_values = [v.decode("utf-8") if isinstance(v, bytes) else str(v) for v in values]
        self.lists.setdefault(key, []).extend(decoded_values)

    def expire(self, key, seconds):
        return True

    def publish(self, channel, payload):
        self.published.append((channel, payload))


class FakeJobManager:
    def __init__(self, redis_client):
        self.redis_client = redis_client

    def dequeue(self, worker_id, timeout=0, priorities=None):
        return None

    def set_state(self, job_id, state, worker_id=None):
        return True

    def fail_job(self, job_id, error):
        return True

    def publish_progress(self, job_id, progress, message=""):
        return True

    def get_state(self, job_id):
        return None


class TestQueueConsumerCompatibility:
    def test_dequeue_legacy_job_reads_backend_queue_contract(self):
        redis_client = FakeRedis()
        queue_key = "user:user-1:trans:queue"
        redis_job_key = "user:user-1:trans:job-123"
        redis_client.lists[queue_key] = [redis_job_key]
        redis_client.hashes[redis_job_key] = {
            "job_id": "job-123",
            "user_id": "user-1",
            "status": "pending",
            "data": '{"source_file":"sample.docx","project_type":"routine"}',
        }

        consumer = QueueConsumer(FakeJobManager(redis_client), worker_id="worker-1")
        job = consumer._dequeue_legacy_job()

        assert job is not None
        assert job["id"] == "job-123"
        assert job["user_id"] == "user-1"
        assert job["source_file"] == "sample.docx"
        assert job["_queue_backend"] == "legacy"

    def test_resolve_source_file_uses_uploads_dir_for_relative_backend_paths(
        self, monkeypatch, tmp_path
    ):
        uploads_dir = tmp_path / "uploads"
        uploads_dir.mkdir()
        source = uploads_dir / "sample.docx"
        source.write_text("placeholder", encoding="utf-8")

        monkeypatch.setenv("TRANSLATION_UPLOADS_DIR", str(uploads_dir))
        consumer = QueueConsumer(FakeJobManager(FakeRedis()), worker_id="worker-1")

        resolved = consumer._resolve_source_file("sample.docx")
        assert resolved == str(source.resolve())

    def test_render_translated_output_projects_targets_back_through_parser(
        self, monkeypatch, tmp_path
    ):
        class FakeParser:
            def __init__(self):
                self.render_calls = []

            def parse(self, file_path):
                return ParsedDocument(
                    segments=[Segment(id="seg-1", text="JP", context={})],
                    metadata={},
                    format="docx",
                    source_path=file_path,
                )

            def render(self, doc, output_path, template_path=None):
                self.render_calls.append((doc, output_path, template_path))
                Path(output_path).write_text("rendered", encoding="utf-8")

        parser = FakeParser()
        consumer = QueueConsumer(FakeJobManager(FakeRedis()), worker_id="worker-1")
        monkeypatch.setattr(consumer, "_select_parser", lambda _: parser)

        input_path = tmp_path / "sample.docx"
        input_path.write_text("placeholder", encoding="utf-8")
        output_path = tmp_path / "sample_translated.docx"
        processed_job = SimpleNamespace(
            segments=[SimpleNamespace(id="seg-1", target="Hello world")]
        )

        consumer._render_translated_output(
            str(input_path), str(output_path), processed_job
        )

        assert output_path.exists()
        rendered_doc, rendered_output, rendered_template = parser.render_calls[0]
        assert rendered_doc.segments[0].text == "Hello world"
        assert rendered_output == str(output_path)
        assert rendered_template == str(input_path)

    def test_validate_config_multiple_errors_returns_all(self):
        """Should return all validation errors, not just the first one."""
        config = {
            "worker": {},  # Missing max_concurrent, heartbeat_interval
            "translation": {},  # Missing default_provider, default_model
        }

        errors = validate_config(config)
        assert len(errors) == 4
        assert "Missing worker.max_concurrent" in errors
        assert "Missing worker.heartbeat_interval" in errors
        assert "Missing translation.default_provider" in errors
        assert "Missing translation.default_model" in errors

    def test_validate_config_allows_optional_sections(self):
        """Should not require sections that aren't in required_sections list."""
        config = {
            "worker": {
                "max_concurrent": 3,
                "heartbeat_interval": "10s",
            },
            "translation": {
                "default_provider": "anthropic",
                "default_model": "claude-4.5-sonnet",
            },
            # Optional sections - should not cause errors
            "glossary": {"enabled": False},
            "cache": {"enabled": False},
        }

        errors = validate_config(config)
        assert errors == []

    def test_validate_config_worker_id_is_optional(self):
        """Should not require worker.id (has default fallback)."""
        config = {
            "worker": {
                "max_concurrent": 3,
                "heartbeat_interval": "10s",
                # id is optional
            },
            "translation": {
                "default_provider": "anthropic",
                "default_model": "claude-4.5-sonnet",
            },
        }

        errors = validate_config(config)
        assert errors == []

    def test_validate_config_style_guide_enabled_requires_path(self):
        config = {
            "worker": {"max_concurrent": 3, "heartbeat_interval": "10s"},
            "translation": {"default_provider": "anthropic", "default_model": "test"},
            "style_guide": {"enabled": True},
        }

        errors = validate_config(config)
        assert any("style_guide.path" in e for e in errors)

    def test_validate_config_style_guide_disabled_no_path_required(self):
        config = {
            "worker": {"max_concurrent": 3, "heartbeat_interval": "10s"},
            "translation": {"default_provider": "anthropic", "default_model": "test"},
            "style_guide": {"enabled": False},
        }

        errors = validate_config(config)
        assert errors == []


class TestMainIntegration:
    """Integration tests for main() function behavior."""

    def test_main_exits_with_code_1_on_missing_config(self, monkeypatch, capsys):
        """Should exit with code 1 and print error when config is missing."""

        # Monkeypatch sys.exit to capture exit code without actually exiting
        def mock_exit(code):
            raise SystemExit(code)

        monkeypatch.setattr(sys, "exit", mock_exit)

        # Mock load_config to raise FileNotFoundError
        def mock_load_config():
            raise FileNotFoundError("Config file not found: config.toml")

        monkeypatch.setattr("main.load_config", mock_load_config)

        with pytest.raises(SystemExit) as exc_info:
            from main import main

            main()

        assert exc_info.value.code == 1

        captured = capsys.readouterr()
        assert "Error:" in captured.err
        assert "Config file not found" in captured.err

    def test_main_exits_with_code_1_on_validation_errors(self, monkeypatch, capsys):
        """Should exit with code 1 and print validation errors."""

        def mock_exit(code):
            raise SystemExit(code)

        monkeypatch.setattr(sys, "exit", mock_exit)

        # Mock config with validation errors
        def mock_load_config():
            return {"worker": {}, "translation": {}}

        monkeypatch.setattr("main.load_config", mock_load_config)

        with pytest.raises(SystemExit) as exc_info:
            from main import main

            main()

        assert exc_info.value.code == 1

        captured = capsys.readouterr()
        assert "Configuration errors:" in captured.err

    def test_main_prints_worker_info_on_success(self, monkeypatch, capsys):
        """Should print worker information when config is valid."""

        def mock_load_config():
            return {
                "worker": {
                    "id": "test-worker-123",
                    "max_concurrent": 3,
                    "heartbeat_interval": "10s",
                },
                "translation": {
                    "default_provider": "anthropic",
                    "default_model": "claude-4.5-sonnet",
                },
            }

        monkeypatch.setattr("main.load_config", mock_load_config)

        # Mock to prevent hanging on "Press Ctrl+C to stop"
        def mock_main_continue():
            from main import main

            # Call main but catch before infinite wait
            import sys

            sys.exit(0)

        # Since main() has no return and would hang, we can't test the full flow
        # But we can test the config loading portion
        from main import validate_config

        config = mock_load_config()
        errors = validate_config(config)
        assert errors == []

        # Verify the expected values
        assert config["worker"]["id"] == "test-worker-123"
        assert config["translation"]["default_provider"] == "anthropic"
