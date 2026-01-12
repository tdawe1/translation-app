# cmd/translation-worker/main.py
"""
Translation Worker - Main Entry Point

A hybrid worker that combines:
1. Folder watching (for Gengo downloads - loose coupling)
2. Redis job queue (for horizontal scaling)

Supports multi-provider LLM translation, glossary system, cache,
layout preservation, and plugin-based document parsers.
"""

import sys
import tomli
from pathlib import Path
from typing import Optional


def load_config(config_path: str = "config.toml") -> dict:
    """Load configuration from TOML file.

    Args:
        config_path: Path to TOML config file (relative or absolute)

    Returns:
        Parsed configuration as nested dict

    Raises:
        FileNotFoundError: If config file doesn't exist
        tomli.TOMLDecodeError: If TOML is malformed
    """
    config_file = Path(config_path)
    if not config_file.is_absolute():
        # Relative to main.py
        config_file = Path(__file__).parent / config_path

    if not config_file.exists():
        raise FileNotFoundError(f"Config file not found: {config_file}")

    with open(config_file, "rb") as f:
        return tomli.load(f)


def validate_config(cfg: dict) -> list[str]:
    """Validate required configuration sections.

    Args:
        cfg: Parsed configuration dict

    Returns:
        List of validation errors (empty if valid)
    """
    errors = []

    # Check required sections
    required_sections = ["worker", "translation"]
    for section in required_sections:
        if section not in cfg:
            errors.append(f"Missing required section: [{section}]")

    # Validate worker section
    if "worker" in cfg:
        if "max_concurrent" not in cfg["worker"]:
            errors.append("Missing worker.max_concurrent")
        if "heartbeat_interval" not in cfg["worker"]:
            errors.append("Missing worker.heartbeat_interval")

    # Validate translation section
    if "translation" in cfg:
        if "default_provider" not in cfg["translation"]:
            errors.append("Missing translation.default_provider")
        if "default_model" not in cfg["translation"]:
            errors.append("Missing translation.default_model")

    return errors


def main():
    """Main entry point for the translation worker."""
    try:
        # Load configuration
        config = load_config()

        # Validate configuration
        validation_errors = validate_config(config)
        if validation_errors:
            print("Configuration errors:", file=sys.stderr)
            for error in validation_errors:
                print(f"  - {error}", file=sys.stderr)
            sys.exit(1)

        # Display worker info
        worker_id = config.get("worker", {}).get("id", "unspecified")
        provider = config.get("translation", {}).get("default_provider")
        model = config.get("translation", {}).get("default_model")

        print(f"Translation Worker v1.0.0 starting...")
        print(f"  Worker ID: {worker_id}")
        print(f"  Translation Backend: {provider}/{model}")
        print(f"  Mode: hybrid (folder watch + Redis job queue)")

        # TODO: Initialize components
        # - Glossary loader
        # - Cache manager
        # - Plugin registry
        # - Folder watcher
        # - Redis job queue consumer

        print("Worker initialized successfully.")
        print("Press Ctrl+C to stop.")

        # TODO: Start event loop
        # - Start folder watcher
        # - Start Redis queue consumer
        # - Start heartbeat publisher

    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        print("Create a config.toml file or specify path with --config", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nShutting down gracefully...")
        sys.exit(0)
    except Exception as e:
        print(f"Unexpected error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
