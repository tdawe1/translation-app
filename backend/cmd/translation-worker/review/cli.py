"""CLI for translation worker review module.

Provides direct command-line access to translation and judge
functionality without requiring a web server.

Usage:
    python -m review translate "こんにちは" --provider anthropic
    python -m review judge source.txt candidate_a.txt candidate_b.txt
    python -m review batch --input sources.txt --output translations.txt
"""
import asyncio
import json
import logging
import os
import sys
from pathlib import Path
from typing import List, Optional

import click

# Try importing optional dependencies
try:
    from .llm import get_provider
    from .multimodel import MultiModelTranslator
    from .judge import TranslationJudge
    from .models import TranslationCandidate
    CLI_AVAILABLE = True
except ImportError:
    CLI_AVAILABLE = False

logging.basicConfig(
    level=logging.INFO,
    format="[%(levelname)s] %(message)s"
)
logger = logging.getLogger(__name__)


def _get_api_key(provider: str) -> Optional[str]:
    """Get API key from environment variable."""
    env_var = f"{provider.upper()}_API_KEY"
    return os.environ.get(env_var)


def _ensure_provider(provider: str, model: Optional[str] = None):
    """Ensure provider is available and configured.

    Raises:
        click.ClickException: If provider unavailable or missing API key
    """
    if not CLI_AVAILABLE:
        raise click.ClickException(
            "LLM integration not available. Install dependencies:\n"
            "  pip install anthropic openai requests"
        )

    api_key = _get_api_key(provider)
    if not api_key:
        raise click.ClickException(
            f"Missing API key for {provider}. Set {provider.upper()}_API_KEY environment variable."
        )

    try:
        kwargs = {}
        if provider == "gemini":
            kwargs["project_id"] = os.environ.get("GEMINI_PROJECT_ID", "")
            kwargs["location"] = os.environ.get("GEMINI_LOCATION", "us-central1")

        return get_provider(provider, api_key, model, **kwargs)
    except Exception as e:
        raise click.ClickException(f"Failed to initialize {provider} provider: {e}")


@click.group()
@click.version_option(version="1.0.0")
def cli():
    """Translation worker CLI - Direct access to translation and judge functionality.

    Examples:
        # Translate a single text
        translate "こんにちは" --provider anthropic

        # Translate with custom model
        translate "こんにちは" --provider openai --model gpt-4.1

        # Batch process from file
        batch --input sources.txt --output translations.txt

        # Compare two translations
        judge "Original Japanese text" translation_a.txt translation_b.txt
    """
    pass


@cli.command()
@click.argument("text")
@click.option(
    "--provider", "-p",
    type=click.Choice(["anthropic", "openai", "gemini"]),
    required=True,
    help="LLM provider to use"
)
@click.option(
    "--model", "-m",
    help="Model identifier (uses provider default if not specified)"
)
@click.option(
    "--format", "-f",
    type=click.Choice(["text", "json", "csv"]),
    default="text",
    help="Output format"
)
@click.option(
    "--output", "-o",
    type=click.Path(),
    help="Write output to file instead of stdout"
)
@click.option(
    "--parallel/--sequential",
    default=True,
    help="Use parallel execution (only affects multiple providers)"
)
def translate(text: str, provider: str, model: Optional[str], format: str,
             output: Optional[str], parallel: bool):
    """Translate Japanese text to English.

    Example: translate "こんにちは" --provider anthropic --format json
    """
    provider_instance = _ensure_provider(provider, model)

    # Build prompt (simplified for CLI)
    prompt = f"Translate the following Japanese text to English:\n\n{text}"

    try:
        response = provider_instance.generate(prompt)
        translation = response.text.strip()

        # Format output
        if format == "text":
            result = translation
        elif format == "json":
            result = json.dumps({
                "source": text,
                "translation": translation,
                "provider": provider,
                "model": response.model,
                "usage": response.usage,
                "latency_ms": response.latency_ms
            }, indent=2, ensure_ascii=False)
        elif format == "csv":
            result = f"source,translation,provider,model\n"
            result += f'"{text}","{translation}","{provider}","{response.model}"'

        # Output
        if output:
            Path(output).write_text(result, encoding="utf-8")
            click.echo(f"Translation written to {output}")
        else:
            click.echo(result)

    except Exception as e:
        raise click.ClickException(f"Translation failed: {e}")


@cli.command()
@click.argument("source", type=click.Path(exists=True))
@click.argument("candidate_a", type=click.Path(exists=True))
@click.argument("candidate_b", type=click.Path(exists=True))
@click.option(
    "--provider", "-p",
    type=click.Choice(["anthropic", "openai", "gemini"]),
    default="anthropic",
    help="LLM provider for judge"
)
@click.option(
    "--model", "-m",
    default="claude-4.5-sonnet",
    help="Judge model"
)
@click.option(
    "--format", "-f",
    type=click.Choice(["text", "json"]),
    default="text",
    help="Output format"
)
def judge(source: str, candidate_a: str, candidate_b: str, provider: str,
          model: str, format: str):
    """Judge which translation is better.

    Compare two translations and select the winner using LLM evaluation.
    """
    provider_instance = _ensure_provider(provider, model)

    # Read files
    source_text = Path(source).read_text(encoding="utf-8").strip()
    text_a = Path(candidate_a).read_text(encoding="utf-8").strip()
    text_b = Path(candidate_b).read_text(encoding="utf-8").strip()

    # Build judge prompt
    prompt = f"""You are an expert translation evaluator.
Compare the following two Japanese-to-English translations and select the better one.

Original Japanese text:
{source_text}

Translation A:
{text_a}

Translation B:
{text_b}

Output format (JSON):
{{
    "winner": "translation_a" | "translation_b" | "tie",
    "confidence": 0.0-1.0,
    "reasoning": "Brief explanation",
    "concerns": ["list any concerns found"]
}}

Evaluate both translations and output JSON."""

    try:
        response = provider_instance.generate(prompt)

        # Parse JSON response
        try:
            result_data = json.loads(response.text)
        except json.JSONDecodeError:
            # Fallback: extract winner from text
            result_data = {
                "winner": "unknown",
                "confidence": 0.0,
                "reasoning": response.text,
                "concerns": ["Failed to parse JSON"]
            }

        if format == "json":
            output = json.dumps(result_data, indent=2)
        else:
            winner = result_data.get("winner", "unknown")
            output = f"Winner: {winner}\n"
            output += f"Confidence: {result_data.get('confidence', 0):.2f}\n"
            output += f"Reasoning: {result_data.get('reasoning', 'N/A')}"

        click.echo(output)

    except Exception as e:
        raise click.ClickException(f"Judgment failed: {e}")


@cli.command()
@click.option(
    "--input", "-i",
    type=click.Path(exists=True),
    required=True,
    help="Input file with source texts (one per line)"
)
@click.option(
    "--output", "-o",
    type=click.Path(),
    required=True,
    help="Output file for translations"
)
@click.option(
    "--provider", "-p",
    type=click.Choice(["anthropic", "openai", "gemini"]),
    required=True,
    help="LLM provider to use"
)
@click.option(
    "--model", "-m",
    help="Model identifier"
)
@click.option(
    "--parallel/--sequential",
    default=True,
    help="Execution mode"
)
@click.option(
    "--format", "-f",
    type=click.Choice(["text", "json", "csv"]),
    default="text",
    help="Output format"
)
def batch(input: str, output: str, provider: str, model: Optional[str],
          parallel: bool, format: str):
    """Batch translate texts from a file.

    Processes each line of the input file as a separate translation.
    """
    provider_instance = _ensure_provider(provider, model)

    # Read input
    input_path = Path(input)
    sources = input_path.read_text(encoding="utf-8").strip().split("\n")

    # Filter empty lines
    sources = [s.strip() for s in sources if s.strip()]

    if not sources:
        raise click.ClickException("No non-empty lines found in input file")

    click.echo(f"Processing {len(sources)} texts...")

    translations = []
    for i, source in enumerate(sources):
        click.echo(f"[{i+1}/{len(sources)}] Processing: {source[:50]}...", err=True)

        prompt = f"Translate the following Japanese text to English:\n\n{source}"
        try:
            response = provider_instance.generate(prompt)
            translations.append({
                "source": source,
                "translation": response.text.strip(),
                "usage": response.usage
            })
        except Exception as e:
            logger.warning(f"Failed to translate line {i+1}: {e}")
            translations.append({
                "source": source,
                "translation": f"[ERROR: {e}]",
                "usage": {}
            })

    # Format output
    if format == "text":
        result = "\n".join(t["translation"] for t in translations)
    elif format == "json":
        result = json.dumps(translations, indent=2, ensure_ascii=False)
    elif format == "csv":
        result = "source,translation,prompt_tokens,completion_tokens\n"
        for t in translations:
            safe_translation = t["translation"].replace('"', '""')
            result += f'"{t["source"]}","{safe_translation}",{t["usage"].get("prompt_tokens", 0)},{t["usage"].get("completion_tokens", 0)}\n'

    # Write output
    Path(output).write_text(result, encoding="utf-8")
    click.echo(f"Translated {len(translations)} texts → {output}")


# For python -m review compatibility
def main():
    """Entry point for python -m review."""
    cli()


if __name__ == "__main__":
    main()
