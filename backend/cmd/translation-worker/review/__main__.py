"""Main entry point for python -m review.

Allows running CLI commands directly:
    python -m review translate "こんにちは" --provider anthropic
"""
from .cli import cli

if __name__ == "__main__":
    cli()
