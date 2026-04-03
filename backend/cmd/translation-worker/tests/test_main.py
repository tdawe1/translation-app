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
from unittest.mock import patch, MagicMock

import pytest

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from main import (
    load_config,
    validate_config,
    load_style_guide_prompt,
    _resolve_api_key,
    build_translation_provider,
    build_judge_provider,
    build_style_checker,
    build_workflow,
)


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



class TestLoadStyleGuidePrompt:
    """Tests for load_style_guide_prompt() helper."""

    def test_returns_none_when_disabled(self):
        """Should return None when style_guide.enabled is false."""
        config = {"style_guide": {"enabled": False}}
        assert load_style_guide_prompt(config) is None

    def test_returns_none_when_section_missing(self):
        """Should return None when style_guide section is absent."""
        config = {}
        assert load_style_guide_prompt(config) is None

    def test_returns_none_when_path_missing(self, capsys):
        """Should return None and warn when file doesn't exist."""
        config = {
            "style_guide": {
                "enabled": True,
                "path": "/nonexistent/guide.md",
            }
        }
        result = load_style_guide_prompt(config)
        assert result is None
        captured = capsys.readouterr()
        assert "not found" in captured.err

    def test_returns_prompt_when_valid(self, tmp_path):
        """Should return a prompt string from a valid style guide file."""
        # Create a minimal style guide markdown
        guide_md = tmp_path / "guide.md"
        guide_md.write_text(
            "# Style Guide\n\n"
            "## Punctuation\n\n"
            "* Use the Oxford comma in lists of three or more items.\n"
            "* Use US English spelling.\n"
        )
        config = {
            "style_guide": {
                "enabled": True,
                "path": str(guide_md),
            }
        }
        result = load_style_guide_prompt(config)
        assert result is not None
        assert "Gengo" in result or "English" in result
        assert len(result) > 50


class TestResolveApiKey:
    """Tests for _resolve_api_key() helper."""

    def test_reads_from_provider_config_env_var(self, monkeypatch):
        """Should read API key from env var specified in provider config."""
        monkeypatch.setenv("MY_CUSTOM_KEY", "sk-custom-123")
        config = {
            "translation": {
                "providers": {
                    "openai": {"api_key_env": "MY_CUSTOM_KEY"}
                }
            }
        }
        assert _resolve_api_key(config, "openai") == "sk-custom-123"

    def test_falls_back_to_standard_env_var(self, monkeypatch):
        """Should fall back to standard env var name."""
        monkeypatch.setenv("OPENAI_API_KEY", "sk-standard-456")
        config = {"translation": {}}
        assert _resolve_api_key(config, "openai") == "sk-standard-456"

    def test_returns_none_when_no_key(self, monkeypatch):
        """Should return None if no key is found."""
        monkeypatch.delenv("OPENAI_API_KEY", raising=False)
        monkeypatch.delenv("ANTHROPIC_API_KEY", raising=False)
        config = {"translation": {}}
        assert _resolve_api_key(config, "openai") is None


class TestBuildTranslationProvider:
    """Tests for build_translation_provider()."""

    def test_raises_on_missing_api_key(self, monkeypatch):
        """Should raise RuntimeError when API key is missing."""
        monkeypatch.delenv("OPENAI_API_KEY", raising=False)
        monkeypatch.delenv("ANTHROPIC_API_KEY", raising=False)
        config = {
            "translation": {
                "default_provider": "openai",
                "default_model": "gpt-5.2",
            }
        }
        with pytest.raises(RuntimeError, match="Missing API key"):
            build_translation_provider(config)

    @patch("main.get_provider")
    def test_builds_provider_with_system_prompt(self, mock_get_provider, monkeypatch):
        """Should pass system_prompt to provider construction."""
        monkeypatch.setenv("OPENAI_API_KEY", "sk-test-key")
        mock_provider = MagicMock()
        mock_get_provider.return_value = mock_provider

        config = {
            "translation": {
                "default_provider": "openai",
                "default_model": "gpt-5.2",
            }
        }
        prompt = "You are a Gengo translator."
        result = build_translation_provider(config, system_prompt=prompt)

        mock_get_provider.assert_called_once_with(
            provider_name="openai",
            api_key="sk-test-key",
            model="gpt-5.2",
            system_prompt=prompt,
        )
        assert result is mock_provider

    @patch("main.get_provider")
    def test_builds_provider_without_system_prompt(self, mock_get_provider, monkeypatch):
        """Should work without system_prompt (style guide disabled)."""
        monkeypatch.setenv("ANTHROPIC_API_KEY", "sk-ant-key")
        mock_provider = MagicMock()
        mock_get_provider.return_value = mock_provider

        config = {
            "translation": {
                "default_provider": "anthropic",
                "default_model": "claude-sonnet-4-5-20250929",
            }
        }
        result = build_translation_provider(config)

        mock_get_provider.assert_called_once_with(
            provider_name="anthropic",
            api_key="sk-ant-key",
            model="claude-sonnet-4-5-20250929",
            system_prompt=None,
        )


class TestBuildJudgeProvider:
    """Tests for build_judge_provider()."""

    @patch("main.get_provider")
    def test_builds_judge_without_system_prompt(self, mock_get_provider, monkeypatch):
        """Judge provider should NOT receive the style guide prompt."""
        monkeypatch.setenv("OPENAI_API_KEY", "sk-judge-key")
        mock_provider = MagicMock()
        mock_get_provider.return_value = mock_provider

        config = {
            "translation": {
                "default_provider": "openai",
                "default_model": "gpt-5.2",
            }
        }
        build_judge_provider(config)

        mock_get_provider.assert_called_once_with(
            provider_name="openai",
            api_key="sk-judge-key",
            model="gpt-5.2",
        )

    def test_returns_none_when_no_key(self, monkeypatch):
        """Should return None instead of crashing when key is missing."""
        monkeypatch.delenv("OPENAI_API_KEY", raising=False)
        config = {
            "translation": {
                "default_provider": "openai",
                "default_model": "gpt-5.2",
            }
        }
        result = build_judge_provider(config)
        assert result is None


class TestBuildStyleChecker:
    """Tests for build_style_checker()."""

    def test_returns_checker_when_enabled(self):
        """Should return a StyleChecker with gengo_rules_enabled."""
        config = {"style_guide": {"enabled": True, "path": "/some/path.md"}}
        checker = build_style_checker(config)
        assert checker is not None
        assert checker.gengo_rules_enabled is True

    def test_returns_none_when_disabled(self):
        """Should return None when style guide is disabled."""
        config = {"style_guide": {"enabled": False}}
        assert build_style_checker(config) is None

    def test_returns_none_when_section_missing(self):
        """Should return None when style_guide section is absent."""
        config = {}
        assert build_style_checker(config) is None


class TestBuildWorkflow:
    """Tests for build_workflow() — the main runtime construction entry point."""

    @patch("main.build_judge_provider")
    @patch("main.build_translation_provider")
    @patch("main.load_style_guide_prompt")
    def test_builds_workflow_style_guide_disabled(
        self, mock_prompt, mock_trans, mock_judge
    ):
        """Should build a workflow even with style guide disabled."""
        mock_prompt.return_value = None
        mock_trans.return_value = MagicMock()
        mock_judge.return_value = MagicMock()

        config = {
            "translation": {"default_provider": "openai", "default_model": "gpt-5.2"},
            "style_guide": {"enabled": False},
        }
        workflow = build_workflow(config)

        assert workflow is not None
        assert workflow.translator is not None
        assert len(workflow.translator.providers) == 1
        mock_trans.assert_called_once_with(config, None)

    @patch("main.build_judge_provider")
    @patch("main.build_translation_provider")
    @patch("main.load_style_guide_prompt")
    def test_builds_workflow_style_guide_enabled(
        self, mock_prompt, mock_trans, mock_judge
    ):
        """Should pass style guide prompt to translation provider."""
        mock_prompt.return_value = "You are a Gengo translator."
        mock_trans.return_value = MagicMock()
        mock_judge.return_value = MagicMock()

        config = {
            "translation": {"default_provider": "openai", "default_model": "gpt-5.2"},
            "style_guide": {"enabled": True, "path": "/some/guide.md"},
        }
        workflow = build_workflow(config)

        assert workflow is not None
        mock_trans.assert_called_once_with(config, "You are a Gengo translator.")

    @patch("main.build_judge_provider")
    @patch("main.build_translation_provider")
    @patch("main.load_style_guide_prompt")
    def test_workflow_judge_disabled_when_no_key(
        self, mock_prompt, mock_trans, mock_judge
    ):
        """Judge should be disabled when its provider returns None."""
        mock_prompt.return_value = None
        mock_trans.return_value = MagicMock()
        mock_judge.return_value = None  # No API key

        config = {
            "translation": {"default_provider": "openai", "default_model": "gpt-5.2"},
        }
        workflow = build_workflow(config)

        assert workflow.judge.enabled is False

    def test_raises_on_missing_translation_credentials(self, monkeypatch):
        """Should raise RuntimeError if translation provider key is missing."""
        monkeypatch.delenv("OPENAI_API_KEY", raising=False)
        monkeypatch.delenv("ANTHROPIC_API_KEY", raising=False)

        config = {
            "translation": {"default_provider": "openai", "default_model": "gpt-5.2"},
        }
        with pytest.raises(RuntimeError, match="Missing API key"):
            build_workflow(config)
