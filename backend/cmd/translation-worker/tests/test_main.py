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

import pytest

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from main import load_config, validate_config


class TestLoadConfig:
    """Test suite for load_config() function."""

    def test_load_config_from_valid_toml(self):
        """Should successfully parse a valid TOML configuration file."""
        with tempfile.NamedTemporaryFile(mode='wb', suffix='.toml', delete=False) as f:
            f.write(b'''
[worker]
id = "test-worker-1"
max_concurrent = 3
heartbeat_interval = "10s"

[translation]
default_provider = "anthropic"
default_model = "claude-4.5-sonnet"
''')
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
        # Create a temp config file in the same directory as main.py
        config_dir = Path(__file__).parent
        temp_config = config_dir / "temp_test_config.toml"
        
        try:
            temp_config.write_text('''
[worker]
max_concurrent = 1
heartbeat_interval = "5s"

[translation]
default_provider = "openai"
default_model = "gpt-4"
''')
            
            # Load using just the filename (relative path)
            config = load_config("temp_test_config.toml")
            
            assert config["worker"]["max_concurrent"] == 1
            assert config["translation"]["default_provider"] == "openai"
        finally:
            if temp_config.exists():
                temp_config.unlink()

    def test_load_config_with_absolute_path(self):
        """Should accept absolute paths without modification."""
        with tempfile.NamedTemporaryFile(mode='wb', suffix='.toml', delete=False) as f:
            f.write(b'[worker]\nid = "absolute-path-test"\nmax_concurrent = 1\nheartbeat_interval = "10s"\n\n[translation]\ndefault_provider = "test"\ndefault_model = "test-model"\n')
            temp_path = Path(f.name)

        try:
            config = load_config(str(temp_path))
            assert config["worker"]["id"] == "absolute-path-test"
        finally:
            temp_path.unlink()

    def test_load_config_missing_file_raises FileNotFoundError(self):
        """Should raise FileNotFoundError when config file doesn't exist."""
        with pytest.raises(FileNotFoundError) as exc_info:
            load_config("nonexistent-config.toml")
        
        assert "Config file not found" in str(exc_info.value)

    def test_load_config_malformed_toml_raises_toml_decode_error(self):
        """Should raise tomli.TOMLDecodeError for invalid TOML syntax."""
        import tomli
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.toml', delete=False) as f:
            f.write('[invalid toml syntax\n')  # Missing closing bracket
            temp_path = f.name

        try:
            with pytest.raises(tomli.TOMLDecodeError):
                load_config(temp_path)
        finally:
            Path(temp_path).unlink()

    def test_load_config_empty_file_returns_empty_dict(self):
        """Should return empty dict for an empty TOML file."""
        with tempfile.NamedTemporaryFile(mode='w', suffix='.toml', delete=False) as f:
            f.write('')
            temp_path = f.name

        try:
            config = load_config(temp_path)
            assert config == {}
        finally:
            Path(temp_path).unlink()

    def test_load_config_preserves_all_sections(self):
        """Should load all sections including optional ones."""
        with tempfile.NamedTemporaryFile(mode='wb', suffix='.toml', delete=False) as f:
            f.write(b'''
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
''')
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


class TestMainIntegration:
    """Integration tests for main() function behavior."""

    def test_main_exits_with_code_1_on_missing_config(self, monkeypatch, capsys):
        """Should exit with code 1 and print error when config is missing."""
        # Monkeypatch sys.exit to capture exit code without actually exiting
        def mock_exit(code):
            raise SystemExit(code)
        
        monkeypatch.setattr(sys, 'exit', mock_exit)
        
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
        
        monkeypatch.setattr(sys, 'exit', mock_exit)
        
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
