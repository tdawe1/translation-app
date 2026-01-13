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

logging.basicConfig(level=logging.INFO, format="[%(levelname)s] %(message)s")
logger = logging.getLogger(__name__)

# Short name to internal tool name mapping
# Maps CLI argument values (e.g., "claude") to internal tool names (e.g., "claude_code")
_SHORT_TO_INTERNAL = {
    "claude": "claude_code",
    "codex": "codex",
    "gemini": "gemini_cli",
    "ollama": "ollama",
}

# File size limits for security (prevents DoS via large input files)
MAX_BATCH_FILE_SIZE_MB = 10  # Megabytes
MAX_BATCH_FILE_SIZE = MAX_BATCH_FILE_SIZE_MB * 1024 * 1024  # Bytes

# Error message templates for consistency
ERROR_TEMPLATES = {
    "cli_not_found": "CLI tool '{tool}' not found in PATH.\n\nInstall instructions:\n{install}",
    "mutual_exclusive": "Cannot specify both --{opt1} and --{opt2}. Use {suggestion}.",
    "missing_required": "Must specify either --{opt1} or --{opt2}.",
    "file_too_large": "Input file too large: {size_mb:.1f}MB. Maximum is {max_mb}MB.",
}

# Installation instructions for CLI tools
CLI_INSTALL_INSTRUCTIONS = {
    "claude": "  claude: npm install -g @anthropic-ai/claude-code",
    "codex": "  codex: npm install -g @github-copilot/codex-cli",
    "gemini": "  gemini: npm install -g @google/generative-ai-cli",
    "ollama": "  ollama: curl -fsSL https://ollama.com/install.sh | sh",
}


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

    if tool not in _SHORT_TO_INTERNAL:
        raise click.ClickException(
            f"Unknown CLI tool: {tool}. Use: {list(_SHORT_TO_INTERNAL.keys())}"
        )

    tool_name = _SHORT_TO_INTERNAL[tool]

    # Get base command from DEFAULT_COMMANDS (single source of truth)
    base_command = DEFAULT_COMMANDS.get(tool_name, tool)

    # Check if command is available
    if not shutil.which(base_command):
        hint = CLI_INSTALL_INSTRUCTIONS.get(tool, f"  {tool}: Install {base_command}")
        raise click.ClickException(
            ERROR_TEMPLATES["cli_not_found"].format(tool=base_command, install=hint)
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
    if tool not in _SHORT_TO_INTERNAL:
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
        tool_name = _SHORT_TO_INTERNAL[tool]
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
    "--provider",
    "-p",
    type=click.Choice(["anthropic", "openai", "gemini"]),
    help="LLM provider to use (requires API key)",
)
@click.option(
    "--cli",
    type=click.Choice(["claude", "codex", "gemini", "ollama"]),
    help="Use local CLI tool (no API costs)",
)
@click.option(
    "--model", "-m", help="Model identifier (uses provider default if not specified)"
)
@click.option(
    "--format",
    "-f",
    type=click.Choice(["text", "json", "csv"]),
    default="text",
    help="Output format",
)
@click.option(
    "--output", "-o", type=click.Path(), help="Write output to file instead of stdout"
)
@click.option(
    "--parallel/--sequential",
    default=True,
    # TODO: Implement parallel execution for batch processing
    # Currently reserved for future use - has no effect on execution
    help="Use parallel execution (reserved for future use)",
)
@click.option(
    "--dry-run",
    is_flag=True,
    help="Show the command that would be executed without running it (CLI tools only)",
)
def translate(
    text: str,
    provider: Optional[str],
    cli: Optional[str],
    model: Optional[str],
    format: str,
    output: Optional[str],
    parallel: bool,
    dry_run: bool,
):
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
            ERROR_TEMPLATES["mutual_exclusive"].format(
                opt1="provider",
                opt2="cli",
                suggestion="--cli for local tools or --provider for API-based providers",
            )
        )
    if not provider and not cli:
        raise click.ClickException(
            ERROR_TEMPLATES["missing_required"].format(
                opt1="provider (for API)", opt2="cli (for local tools)"
            )
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
        click.echo(
            f"Would execute: {' '.join(cmd[:2])}..."
        )  # Show truncated for readability
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
            result = json.dumps(
                {
                    "source": text,
                    "translation": translation,
                    "provider": provider_name,
                    "model": model_name,
                    "usage": usage,
                },
                indent=2,
                ensure_ascii=False,
            )
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
    "--provider",
    "-p",
    type=click.Choice(["anthropic", "openai", "gemini"]),
    help="LLM provider for judge (requires API key)",
)
@click.option(
    "--cli",
    type=click.Choice(["claude", "codex", "gemini", "ollama"]),
    help="Use local CLI tool for judge (no API costs)",
)
@click.option("--model", "-m", default="claude-4.5-sonnet", help="Judge model")
@click.option(
    "--format",
    "-f",
    type=click.Choice(["text", "json"]),
    default="text",
    help="Output format",
)
def judge(
    source: str,
    candidate_a: str,
    candidate_b: str,
    provider: Optional[str],
    cli: Optional[str],
    model: str,
    format: str,
):
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
            ERROR_TEMPLATES["mutual_exclusive"].format(
                opt1="provider",
                opt2="cli",
                suggestion="--cli for local tools or --provider for API-based providers",
            )
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
                    "concerns": ["Failed to parse JSON from CLI output"],
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
                    "concerns": ["Failed to parse JSON"],
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
    "--input",
    "-i",
    type=click.Path(exists=True),
    required=True,
    help="Input file with source texts (one per line)",
)
@click.option(
    "--output",
    "-o",
    type=click.Path(),
    required=True,
    help="Output file for translations",
)
@click.option(
    "--provider",
    "-p",
    type=click.Choice(["anthropic", "openai", "gemini"]),
    help="LLM provider to use (requires API key)",
)
@click.option(
    "--cli",
    type=click.Choice(["claude", "codex", "gemini", "ollama"]),
    help="Use local CLI tool (no API costs)",
)
@click.option("--model", "-m", help="Model identifier")
@click.option(
    "--parallel/--sequential",
    default=True,
    # TODO: Implement parallel execution for batch processing
    # Currently reserved for future use - has no effect on execution
    help="Use parallel execution (reserved for future use)",
)
@click.option(
    "--format",
    "-f",
    type=click.Choice(["text", "json", "csv"]),
    default="text",
    help="Output format",
)
def batch(
    input: str,
    output: str,
    provider: Optional[str],
    cli: Optional[str],
    model: Optional[str],
    parallel: bool,
    format: str,
):
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
            ERROR_TEMPLATES["mutual_exclusive"].format(
                opt1="provider",
                opt2="cli",
                suggestion="--cli for local tools or --provider for API-based providers",
            )
        )
    if not provider and not cli:
        raise click.ClickException(
            ERROR_TEMPLATES["missing_required"].format(
                opt1="provider (for API)", opt2="cli (for local tools)"
            )
        )

    # Read input with size check (prevent unbounded memory usage)
    input_path = Path(input)

    # Check file size BEFORE reading (10MB limit)
    file_size = input_path.stat().st_size
    if file_size > MAX_BATCH_FILE_SIZE:
        size_mb = file_size / (1024 * 1024)
        raise click.ClickException(
            ERROR_TEMPLATES["file_too_large"].format(
                size_mb=size_mb, max_mb=MAX_BATCH_FILE_SIZE_MB
            )
        )

    sources = input_path.read_text(encoding="utf-8").strip().split("\n")

    # Filter empty lines
    sources = [s.strip() for s in sources if s.strip()]

    if not sources:
        raise click.ClickException("No non-empty lines found in input file")

    click.echo(f"Processing {len(sources)} texts...")

    translations = []
    for i, source in enumerate(sources):
        click.echo(f"[{i + 1}/{len(sources)}] Processing: {source[:50]}...", err=True)

        try:
            if cli:
                # Use local CLI tool via CLIProvider abstraction
                cli_provider = _get_cli_provider(cli)
                prompt = (
                    f"Translate the following Japanese text to English:\n\n{source}"
                )
                response = cli_provider.generate(prompt)
                usage = response.usage or {}
                usage["tool"] = cli  # Add tool name for compatibility
                translations.append(
                    {
                        "source": source,
                        "translation": response.text.strip(),
                        "usage": usage,
                    }
                )
            else:
                provider_instance = _ensure_provider(provider, model)
                prompt = (
                    f"Translate the following Japanese text to English:\n\n{source}"
                )
                response = provider_instance.generate(prompt)
                translations.append(
                    {
                        "source": source,
                        "translation": response.text.strip(),
                        "usage": response.usage,
                    }
                )
        except Exception as e:
            logger.warning(f"Failed to translate line {i + 1}: {e}")
            translations.append(
                {"source": source, "translation": f"[ERROR: {e}]", "usage": {}}
            )

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


@cli.command()
@click.argument("input_file", type=click.Path(exists=True))
@click.option(
    "--output",
    "-o",
    type=click.Path(),
    help="Output file path (default: input_translated.ext)",
)
@click.option(
    "--provider",
    "-p",
    type=click.Choice(["anthropic", "openai", "gemini"]),
    multiple=True,
    help="LLM provider(s) to use. Specify twice for review mode.",
)
@click.option(
    "--cli",
    multiple=True,
    type=click.Choice(["claude", "codex", "gemini", "ollama"]),
    help="CLI tool(s) to use. Specify twice for review mode.",
)
@click.option("--model", "-m", help="Model identifier")
@click.option(
    "--style-guide",
    "-s",
    type=click.Path(exists=True),
    help="Path to Gengo style guide markdown file",
)
@click.option(
    "--check-style/--no-check-style",
    default=True,
    help="Run style checker on translations",
)
@click.option(
    "--review/--no-review",
    default=False,
    help="Enable full review workflow: multi-model + judge + flagging + CSV export",
)
@click.option(
    "--csv-output",
    type=click.Path(),
    help="Directory for bilingual CSV export (default: same as output)",
)
@click.option(
    "--format",
    "-f",
    type=click.Choice(["text", "json"]),
    default="text",
    help="Progress output format",
)
def document(
    input_file: str,
    output: Optional[str],
    provider: tuple,
    cli: tuple,
    model: Optional[str],
    style_guide: Optional[str],
    check_style: bool,
    review: bool,
    csv_output: Optional[str],
    format: str,
):
    """Translate a document file (xlsx, docx, pptx).

    Parses the document, translates each segment, optionally validates
    against a style guide, and renders the translated document.

    Use --review for full workflow: multi-model translation, judge evaluation,
    flagging low-confidence segments, and CSV export for human review.

    Examples:

        # Simple translation with one CLI tool
        document input.xlsx --cli gemini

        # Full review workflow with two models
        document input.xlsx --cli claude --cli gemini --review

        # With style guide
        document input.docx --cli claude -s style_guide.md -o output.docx

        # Review mode with API providers
        document input.pptx --provider anthropic --provider openai --review
    """
    import sys
    from pathlib import Path as PathlibPath

    worker_dir = PathlibPath(__file__).parent.parent
    sys.path.insert(0, str(worker_dir))

    providers_list = list(provider) if provider else []
    cli_list = list(cli) if cli else []

    if providers_list and cli_list:
        raise click.ClickException(
            ERROR_TEMPLATES["mutual_exclusive"].format(
                opt1="provider",
                opt2="cli",
                suggestion="--cli for local tools or --provider for API-based providers",
            )
        )
    if not providers_list and not cli_list:
        raise click.ClickException(
            ERROR_TEMPLATES["missing_required"].format(
                opt1="provider (for API)", opt2="cli (for local tools)"
            )
        )

    if review and len(cli_list) < 2 and len(providers_list) < 2:
        raise click.ClickException(
            "Review mode requires at least 2 providers/CLI tools. "
            "Use --cli twice (e.g., --cli claude --cli gemini) or --provider twice."
        )

    input_path = Path(input_file)
    ext = input_path.suffix.lower()

    supported_extensions = {".xlsx", ".docx", ".pptx"}
    if ext not in supported_extensions:
        raise click.ClickException(
            f"Unsupported file type: {ext}. Supported: {', '.join(supported_extensions)}"
        )

    if not output:
        output = str(input_path.with_stem(input_path.stem + "_translated"))

    try:
        from parsers.xlsx_parser import XLSXParser
        from parsers.docx_parser import DOCXParser
        from parsers.pptx_parser import PPTXParser
    except ImportError as e:
        raise click.ClickException(f"Parser not available: {e}")

    parsers_map = {
        ".xlsx": XLSXParser,
        ".docx": DOCXParser,
        ".pptx": PPTXParser,
    }

    parser = parsers_map[ext]()
    click.echo(f"Parsing {input_file}...")
    doc = parser.parse(str(input_path))
    click.echo(f"Found {len(doc.segments)} segments to translate")

    system_prompt = None
    if style_guide:
        try:
            from style_guide.parser import parse_gengo_style_guide
            from style_guide.prompt_builder import build_system_prompt

            guide = parse_gengo_style_guide(Path(style_guide))
            system_prompt = build_system_prompt(guide)
            click.echo(f"Loaded style guide: {len(guide.sections)} sections")
        except ImportError:
            click.echo("Warning: style_guide module not available", err=True)
        except Exception as e:
            click.echo(f"Warning: Failed to load style guide: {e}", err=True)

    style_checker = None
    if check_style:
        try:
            from audit.style_checker import StyleChecker

            style_checker = StyleChecker(gengo_rules_enabled=True)
        except ImportError:
            click.echo("Warning: style_checker not available", err=True)

    segments_to_translate = []
    for i, segment in enumerate(doc.segments):
        if not segment.text or not segment.text.strip():
            continue
        if not any("\u3040" <= c <= "\u9fff" for c in segment.text):
            continue
        segments_to_translate.append((i, segment))

    click.echo(f"Translating {len(segments_to_translate)} Japanese segments...")

    if review:
        _run_review_workflow(
            doc,
            segments_to_translate,
            cli_list,
            providers_list,
            model,
            system_prompt,
            style_checker,
            output,
            csv_output,
            input_path,
            parser,
            format,
        )
    else:
        _run_simple_workflow(
            doc,
            segments_to_translate,
            cli_list,
            providers_list,
            model,
            system_prompt,
            style_checker,
            output,
            input_path,
            parser,
            format,
        )


def _run_simple_workflow(
    doc,
    segments_to_translate,
    cli_list,
    providers_list,
    model,
    system_prompt,
    style_checker,
    output,
    input_path,
    parser,
    format,
):
    from concurrent.futures import ThreadPoolExecutor

    cli_tool = cli_list[0] if cli_list else None
    provider_name = providers_list[0] if providers_list else None

    base_prompt = """Translate the following Japanese text to natural, fluent US English.
Preserve any formatting, numbers, and proper nouns.
IMPORTANT: Use natural phrasing. Avoid em-dashes without spaces.

Japanese text:
"""

    def translate_segment(args):
        idx, seg = args
        if system_prompt:
            prompt = (
                system_prompt
                + "\n\n"
                + base_prompt
                + seg.text
                + "\n\nEnglish translation:"
            )
        else:
            prompt = base_prompt + seg.text + "\n\nEnglish translation:"

        try:
            if cli_tool:
                cli_provider = _get_cli_provider(cli_tool)
                response = cli_provider.generate(prompt)
                return (idx, seg, response.text.strip(), None)
            else:
                provider_instance = _ensure_provider(provider_name, model)
                response = provider_instance.generate(prompt)
                return (idx, seg, response.text.strip(), None)
        except Exception as e:
            return (idx, seg, None, str(e))

    all_issues = []
    completed = 0

    with ThreadPoolExecutor(max_workers=5) as executor:
        futures = list(executor.map(translate_segment, segments_to_translate))

    for idx, seg, translation, error in futures:
        completed += 1
        if error:
            click.echo(
                f"[{completed}/{len(segments_to_translate)}] ERROR: {error}", err=True
            )
            seg.text = f"[TRANSLATION ERROR: {error}]"
        else:
            click.echo(
                f"[{completed}/{len(segments_to_translate)}] Done: {seg.text[:30]}...",
                err=True,
            )
            seg.text = translation

            if style_checker:
                issues = style_checker.check(translation)
                for issue in issues:
                    all_issues.append(
                        {
                            "segment": seg.id,
                            "issue": issue.message,
                            "severity": issue.severity,
                            "suggestion": getattr(issue, "suggestion", None),
                        }
                    )

    click.echo(f"Rendering translated document to {output}...")
    parser.render(doc, output, template_path=str(input_path))

    click.echo(f"\nCompleted: {len(doc.segments)} segments translated")
    click.echo(f"Output: {output}")

    if all_issues:
        click.echo(f"\nStyle issues found: {len(all_issues)}")
        if format == "json":
            click.echo(json.dumps(all_issues, indent=2, ensure_ascii=False))
        else:
            for issue in all_issues[:10]:
                click.echo(
                    f"  [{issue['severity']}] {issue['segment']}: {issue['issue']}"
                )
            if len(all_issues) > 10:
                click.echo(f"  ... and {len(all_issues) - 10} more issues")


def _run_review_workflow(
    doc,
    segments_to_translate,
    cli_list,
    providers_list,
    model,
    system_prompt,
    style_checker,
    output,
    csv_output,
    input_path,
    parser,
    format,
):
    from .multimodel import MultiModelTranslator
    from .judge import TranslationJudge
    from .flagging import FlaggingEngine
    from .exporter import BilingualCSVExporter
    from .models import TranslationJob, TranslationSegment, ReviewConfig

    click.echo(f"REVIEW MODE: Using {len(cli_list) + len(providers_list)} providers")

    llm_providers = []
    for cli_tool in cli_list:
        llm_providers.append(_get_cli_provider(cli_tool))
    for prov_name in providers_list:
        llm_providers.append(_ensure_provider(prov_name, model))

    translator = MultiModelTranslator(
        providers=llm_providers,
        system_prompt=system_prompt,
        parallel=True,
    )

    judge_provider = llm_providers[0] if llm_providers else None
    judge = TranslationJudge(provider=judge_provider, enabled=True)

    flagger = FlaggingEngine()
    config = ReviewConfig.for_project_type("routine")

    csv_dir = csv_output or str(input_path.parent)
    exporter = BilingualCSVExporter(output_dir=csv_dir)

    job = TranslationJob(
        id=f"doc_{input_path.stem[:8]}",
        source_file=str(input_path),
        target_file=output,
        project_type="routine",
    )

    total = len(segments_to_translate)
    flagged_count = 0
    all_issues = []

    for idx, (orig_idx, segment) in enumerate(segments_to_translate):
        click.echo(f"[{idx + 1}/{total}] Translating: {segment.text[:40]}...")

        candidates = translator.translate(segment.text)

        if len(candidates) < 2:
            click.echo(f"  Warning: Only {len(candidates)} candidate(s)", err=True)

        judge_result = judge.judge(
            segment_id=segment.id or f"seg_{orig_idx}",
            source=segment.text,
            candidates=candidates,
        )

        winner_text = ""
        if judge_result.winner == "model_a" and len(candidates) > 0:
            winner_text = candidates[0].text
        elif judge_result.winner == "model_b" and len(candidates) > 1:
            winner_text = candidates[1].text
        elif candidates:
            winner_text = candidates[0].text

        trans_segment = TranslationSegment(
            id=segment.id or f"seg_{orig_idx}",
            job_id=job.id,
            source=segment.text,
            target=winner_text,
            judge_winner=judge_result.winner,
            judge_confidence=judge_result.confidence,
            judge_reasoning=judge_result.reasoning,
            model_a_output=candidates[0].text if len(candidates) > 0 else "",
            model_b_output=candidates[1].text if len(candidates) > 1 else "",
        )

        flagger.flag_segment(trans_segment, judge_result, config)
        if trans_segment.is_flagged:
            flagged_count += 1
            click.echo(f"  FLAGGED: {trans_segment.flag_reason}", err=True)

        job.segments.append(trans_segment)

        segment.text = winner_text

        if style_checker:
            issues = style_checker.check(winner_text)
            for issue in issues:
                all_issues.append(
                    {
                        "segment": trans_segment.id,
                        "issue": issue.message,
                        "severity": issue.severity,
                    }
                )

        click.echo(
            f"  Winner: {judge_result.winner} (conf={judge_result.confidence:.2f})"
        )

    job.update_metrics()

    csv_path = exporter.export_job(job)
    click.echo(f"\nBilingual CSV exported: {csv_path}")

    click.echo(f"Rendering translated document to {output}...")
    parser.render(doc, output, template_path=str(input_path))

    click.echo(f"\n{'=' * 60}")
    click.echo(f"REVIEW WORKFLOW COMPLETE")
    click.echo(f"{'=' * 60}")
    click.echo(f"Segments translated: {total}")
    click.echo(f"Flagged for review:  {flagged_count}")
    click.echo(f"Overall score:       {job.overall_score:.2f}")
    click.echo(f"Output document:     {output}")
    click.echo(f"Bilingual CSV:       {csv_path}")

    if flagged_count > 0:
        click.echo(
            f"\nWARNING: {flagged_count} segments need human review before final use!"
        )
        click.echo("Check the CSV file for flagged segments and their alternatives.")

    if all_issues:
        click.echo(f"\nStyle issues found: {len(all_issues)}")
        for issue in all_issues[:5]:
            click.echo(f"  [{issue['severity']}] {issue['segment']}: {issue['issue']}")
        if len(all_issues) > 5:
            click.echo(f"  ... and {len(all_issues) - 5} more")


def main():
    cli()


if __name__ == "__main__":
    main()
