"""CLI for translation worker review module.

Provides direct command-line access to translation and judge
functionality without requiring a web server.

Usage:
    # Use local CLI tools (no API costs!)
    python -m review translate "こんにちは" --cli claude
    python -m review translate "こんにちは" --cli codex
    python -m review translate "こんにちは" --cli gemini

    # Or use API providers
    python -m review translate "こんにちは" --provider anthropic
"""
import json
import logging
import os
import shutil
from pathlib import Path
from typing import List, Optional

import click

# Try importing optional dependencies
try:
    from .llm import get_provider
    from .llm.cli import CLIProvider
    from .multimodel import MultiModelTranslator
    from .judge import TranslationJudge
    from .models import TranslationCandidate
    CLI_AVAILABLE = True
    # Access DEFAULT_COMMANDS through the class
    DEFAULT_COMMANDS = CLIProvider.DEFAULT_COMMANDS if CLIProvider else {}
except ImportError:
    CLI_AVAILABLE = False
    CLIProvider = None  # type: ignore
    DEFAULT_COMMANDS = {}  # type: ignore

logging.basicConfig(
    level=logging.INFO,
    format="[%(levelname)s] %(message)s"
)
logger = logging.getLogger(__name__)


def _get_api_key(provider: str) -> Optional[str]:
    """Get API key from environment variable."""
    env_var = f"{provider.upper()}_API_KEY"
    return os.environ.get(env_var)


def _get_cli_provider(tool: str) -> "CLIProvider":
    """Get CLIProvider instance for a CLI tool name.

    Maps CLI tool names ("claude", "codex", "gemini", "ollama")
    to CLIProvider instances.

    Uses DEFAULT_COMMANDS from review/llm/cli.py as the single source
    of truth for tool name to base_command mappings.

    Args:
        tool: CLI tool name (short form like "claude", "codex")

    Returns:
        Configured CLIProvider instance

    Raises:
        click.ClickException: If tool is unknown or not available
    """
    if CLIProvider is None:
        raise click.ClickException("CLIProvider not available")

    from .llm.base import ProviderConfig

    # Map short CLI names to internal tool names used by DEFAULT_COMMANDS
    # This is the ONLY place where short names map to internal names
    SHORT_TO_INTERNAL = {
        "claude": "claude_code",
        "codex": "codex",
        "gemini": "gemini_cli",
        "ollama": "ollama",
    }

    if tool not in SHORT_TO_INTERNAL:
        raise click.ClickException(
            f"Unknown CLI tool: {tool}. Use: {list(SHORT_TO_INTERNAL.keys())}"
        )

    tool_name = SHORT_TO_INTERNAL[tool]

    # Get base command from DEFAULT_COMMANDS (single source of truth)
    base_command = DEFAULT_COMMANDS.get(tool_name, tool)

    # Check if command is available
    if not shutil.which(base_command):
        install_hints = {
            "claude": "npm install -g @anthropic-ai/claude-code",
            "codex": "npm install -g @github-copilot/codex-cli",
            "gemini": "npm install -g @google/generative-ai-cli",
            "ollama": "curl -fsSL https://ollama.com/install.sh | sh",
        }
        hint = install_hints.get(tool, f"Install {base_command}")
        raise click.ClickException(
            f"CLI tool '{base_command}' not found in PATH.\nInstall it first:\n  - {tool}: {hint}"
        )

    # Create ProviderConfig with dummy API key (CLI tools don't need it)
    config = ProviderConfig(api_key="", timeout=120)

    return CLIProvider(tool_name=tool_name, config=config, command=base_command)


def _build_cli_command(tool: str, prompt: str) -> List[str]:
    """Build the full command for a CLI tool (for dry-run).

    Uses DEFAULT_COMMANDS from review/llm/cli.py as the single source
    of truth for tool name to base_command mappings.

    Args:
        tool: CLI tool name (short form like "claude", "codex")
        prompt: Prompt text

    Returns:
        Full command as list of strings
    """
    # Short to internal name mapping (must match _get_cli_provider)
    SHORT_TO_INTERNAL = {
        "claude": "claude_code",
        "codex": "codex",
        "gemini": "gemini_cli",
        "ollama": "ollama",
    }

    if tool not in SHORT_TO_INTERNAL:
        raise click.ClickException(f"Unknown CLI tool: {tool}")

    # Build command based on tool type
    # Note: These are full commands with subcommands, different from base command
    if tool == "claude":
        cmd = ["claude", "code", "exec"]
    elif tool == "codex":
        cmd = ["codex", "exec"]
    elif tool == "gemini":
        cmd = ["gemini-cli"]
    elif tool == "ollama":
        cmd = ["ollama", "run", "codellama:latest"]
    else:
        # Fallback: use base command from DEFAULT_COMMANDS
        tool_name = SHORT_TO_INTERNAL[tool]
        base_command = DEFAULT_COMMANDS.get(tool_name, tool)
        cmd = [base_command]

    cmd.append(prompt)
    return cmd


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
    help="LLM provider to use (requires API key)"
)
@click.option(
    "--cli",
    type=click.Choice(["claude", "codex", "gemini", "ollama"]),
    help="Use local CLI tool (no API costs)"
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
@click.option(
    "--dry-run",
    is_flag=True,
    help="Show the command that would be executed without running it (CLI tools only)"
)
def translate(text: str, provider: Optional[str], cli: Optional[str], model: Optional[str], format: str,
             output: Optional[str], parallel: bool, dry_run: bool):
    """Translate Japanese text to English.

    Examples:

        # Use local CLI tool (no API costs!)
        translate "こんにちは" --cli claude
        translate "こんにちは" --cli codex
        translate "こんにちは" --cli gemini

        # Or use API providers (requires API key)
        translate "こんにちは" --provider anthropic --format json

        # Dry-run to see what command would be executed
        translate "こんにちは" --cli claude --dry-run
    """
    # Validate mutual exclusivity
    if provider and cli:
        raise click.ClickException(
            "Cannot specify both --provider and --cli. "
            "Use --cli for local tools or --provider for API-based providers."
        )
    if not provider and not cli:
        raise click.ClickException(
            "Must specify either --provider (for API) or --cli (for local tools). "
            "Use --cli claude/codex/gemini/ollama for local tools with no API costs."
        )

    # Handle dry-run for CLI tools
    if dry_run:
        if not cli:
            raise click.ClickException(
                "--dry-run only works with --cli (local CLI tools). "
                "For API providers, the request goes to an external service."
            )
        # Build and display the command that would be run
        prompt = f"Translate the following Japanese text to English:\n\n{text}"
        cmd = _build_cli_command(cli, prompt)
        click.echo(f"Would execute: {' '.join(cmd[:2])}...")  # Show truncated for readability
        click.echo(f"Full command: {' '.join(repr(c) if ' ' in c else c for c in cmd)}")
        return

    # Build prompt
    prompt = f"Translate the following Japanese text to English:\n\n{text}"

    try:
        if cli:
            # Use local CLI tool (no API costs) via CLIProvider abstraction
            cli_provider = _get_cli_provider(cli)
            response = cli_provider.generate(prompt)
            translation = response.text.strip()
            usage = response.usage or {}
            usage["tool"] = cli  # Add tool name for compatibility
            provider_name = cli
            model_name = response.model or f"{cli}-default"
        else:
            # Use API provider
            provider_instance = _ensure_provider(provider, model)
            response = provider_instance.generate(prompt)
            translation = response.text.strip()
            provider_name = provider
            model_name = response.model
            usage = response.usage

        # Format output
        if format == "text":
            result = translation
        elif format == "json":
            result = json.dumps({
                "source": text,
                "translation": translation,
                "provider": provider_name,
                "model": model_name,
                "usage": usage,
            }, indent=2, ensure_ascii=False)
        elif format == "csv":
            result = f"source,translation,provider,model\n"
            result += f'"{text}","{translation}","{provider_name}","{model_name}"'

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
    help="LLM provider for judge (requires API key)"
)
@click.option(
    "--cli",
    type=click.Choice(["claude", "codex", "gemini", "ollama"]),
    help="Use local CLI tool for judge (no API costs)"
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
def judge(source: str, candidate_a: str, candidate_b: str, provider: Optional[str],
          cli: Optional[str], model: str, format: str):
    """Judge which translation is better.

    Compare two translations and select the winner using LLM evaluation.

    Examples:

        # Use local CLI tool (no API costs!)
        judge source.txt translation_a.txt translation_b.txt --cli claude

        # Or use API provider
        judge source.txt translation_a.txt translation_b.txt --provider anthropic
    """
    # Validate mutual exclusivity and set defaults
    if provider and cli:
        raise click.ClickException(
            "Cannot specify both --provider and --cli. "
            "Use --cli for local tools or --provider for API-based providers."
        )
    if not provider and not cli:
        # Default to API provider for backward compatibility
        # (judge was originally API-only, prefer API over CLI for consistency)
        provider = "anthropic"

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
        if cli:
            # Use local CLI tool for judgment via CLIProvider abstraction
            cli_provider = _get_cli_provider(cli)
            response = cli_provider.generate(prompt)
            judgment = response.text.strip()

            # Parse JSON from CLI output
            try:
                result_data = json.loads(judgment)
            except json.JSONDecodeError:
                result_data = {
                    "winner": "unknown",
                    "confidence": 0.0,
                    "reasoning": judgment,
                    "concerns": ["Failed to parse JSON from CLI output"]
                }
            provider_name = cli
        else:
            # Use API provider
            provider_instance = _ensure_provider(provider, model)
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
            provider_name = provider

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
    help="LLM provider to use (requires API key)"
)
@click.option(
    "--cli",
    type=click.Choice(["claude", "codex", "gemini", "ollama"]),
    help="Use local CLI tool (no API costs)"
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
def batch(input: str, output: str, provider: Optional[str], cli: Optional[str],
          model: Optional[str], parallel: bool, format: str):
    """Batch translate texts from a file.

    Processes each line of the input file as a separate translation.

    Examples:

        # Use local CLI tool (no API costs!)
        batch --input sources.txt --output translations.txt --cli claude

        # Or use API provider
        batch --input sources.txt --output translations.txt --provider anthropic
    """
    # Validate mutual exclusivity
    if provider and cli:
        raise click.ClickException(
            "Cannot specify both --provider and --cli. "
            "Use --cli for local tools or --provider for API-based providers."
        )
    if not provider and not cli:
        raise click.ClickException(
            "Must specify either --provider (for API) or --cli (for local tools)."
        )

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

        try:
            if cli:
                # Use local CLI tool via CLIProvider abstraction
                cli_provider = _get_cli_provider(cli)
                prompt = f"Translate the following Japanese text to English:\n\n{source}"
                response = cli_provider.generate(prompt)
                usage = response.usage or {}
                usage["tool"] = cli  # Add tool name for compatibility
                translations.append({
                    "source": source,
                    "translation": response.text.strip(),
                    "usage": usage
                })
            else:
                provider_instance = _ensure_provider(provider, model)
                prompt = f"Translate the following Japanese text to English:\n\n{source}"
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
            # Proper CSV escaping: double any quotes within fields
            safe_source = t["source"].replace('"', '""')
            safe_translation = t["translation"].replace('"', '""')
            result += f'"{safe_source}","{safe_translation}",{t["usage"].get("prompt_tokens", 0)},{t["usage"].get("completion_tokens", 0)}\n'

    # Write output
    Path(output).write_text(result, encoding="utf-8")
    click.echo(f"Translated {len(translations)} texts → {output}")


# For python -m review compatibility
def main():
    """Entry point for python -m review."""
    cli()


if __name__ == "__main__":
    main()
