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
from pathlib import Path
from typing import Optional

import click
import yaml

from .cli_helpers import (
    batch_translate,
    build_cli_dry_run,
    format_batch_output,
    format_judge_output,
    format_translation_output,
    get_cli_available,
    get_error_templates,
    judge_translation,
    read_text_file,
    run_review_document_workflow,
    run_simple_document_workflow,
    translate_text,
    validate_batch_file_size,
    validate_provider_cli,
)

logging.basicConfig(level=logging.INFO, format="[%(levelname)s] %(message)s")
logger = logging.getLogger(__name__)


def load_config(config_path: str) -> dict:
    """Load configuration from YAML or JSON file."""
    path = Path(config_path)
    if not path.exists():
        raise FileNotFoundError(f"Config file not found: {config_path}")

    with open(path) as f:
        if path.suffix in (".yaml", ".yml"):
            return yaml.safe_load(f) or {}
        elif path.suffix == ".json":
            return json.load(f) or {}
        else:
            raise ValueError(f"Unsupported config format: {path.suffix}")


ERROR_TEMPLATES = get_error_templates()
CLI_AVAILABLE = get_cli_available()


@click.group()
@click.version_option(version="1.0.0")
@click.option(
    "--config",
    "-c",
    type=click.Path(exists=True),
    help="Path to configuration file (YAML or JSON)",
)
@click.pass_context
def cli(ctx: click.Context, config: Optional[str]):
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
    ctx.ensure_object(dict)
    if config:
        try:
            ctx.obj["config"] = load_config(config)
        except (FileNotFoundError, ValueError) as exc:
            raise click.ClickException(f"Failed to load config: {exc}")
    else:
        ctx.obj["config"] = {}


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
    help="Use parallel execution (reserved for future use)",
)
@click.option(
    "--dry-run",
    is_flag=True,
    help="Show the command that would be executed without running it (CLI tools only)",
)
@click.pass_context
def translate(
    ctx: click.Context,
    text: str,
    provider: Optional[str],
    cli: Optional[str],
    model: Optional[str],
    format: str,
    output: Optional[str],
    parallel: bool,
    dry_run: bool,
):
    """Translate Japanese text to English."""
    config = ctx.obj.get("config", {})

    provider = provider or config.get("provider")
    cli = cli or config.get("cli")
    model = model or config.get("model")
    format = format or config.get("format", "text")

    validate_provider_cli(provider, cli)

    if dry_run:
        if not cli:
            raise click.ClickException(
                "--dry-run only works with --cli (local CLI tools). "
                "For API providers, the request goes to an external service."
            )
        prompt = f"Translate the following Japanese text to English:\n\n{text}"
        click.echo(build_cli_dry_run(cli, prompt))
        return

    translation, provider_name, model_name, usage = translate_text(
        text, provider, cli, model
    )
    result = format_translation_output(
        text, translation, provider_name, model_name, usage, format
    )

    if output:
        Path(output).write_text(result, encoding="utf-8")
        click.echo(f"Translation written to {output}")
        return

    click.echo(result)


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
@click.pass_context
def judge(
    ctx: click.Context,
    source: str,
    candidate_a: str,
    candidate_b: str,
    provider: Optional[str],
    cli: Optional[str],
    model: str,
    format: str,
):
    """Judge which translation is better."""
    config = ctx.obj.get("config", {})

    provider = provider or config.get("provider")
    cli = cli or config.get("cli")
    model = model or config.get("model", "claude-4.5-sonnet")
    format = format or config.get("format", "text")

    if provider and cli:
        raise click.ClickException(
            ERROR_TEMPLATES["mutual_exclusive"].format(
                opt1="provider",
                opt2="cli",
                suggestion="--cli for local tools or --provider for API-based providers",
            )
        )
    if not provider and not cli:
        provider = "anthropic"

    source_text = read_text_file(source)
    text_a = read_text_file(candidate_a)
    text_b = read_text_file(candidate_b)

    result_data, _provider_name = judge_translation(
        source_text, text_a, text_b, provider, cli, model
    )
    click.echo(format_judge_output(result_data, format))


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
    help="Use parallel execution (reserved for future use)",
)
@click.option(
    "--format",
    "-f",
    type=click.Choice(["text", "json", "csv"]),
    default="text",
    help="Output format",
)
@click.pass_context
def batch(
    ctx: click.Context,
    input: str,
    output: str,
    provider: Optional[str],
    cli: Optional[str],
    model: Optional[str],
    parallel: bool,
    format: str,
):
    """Batch translate texts from a file."""
    config = ctx.obj.get("config", {})

    provider = provider or config.get("provider")
    cli = cli or config.get("cli")
    model = model or config.get("model")
    format = format or config.get("format", "text")

    validate_provider_cli(provider, cli)

    input_path = Path(input)
    validate_batch_file_size(input_path)

    sources = input_path.read_text(encoding="utf-8").strip().split("\n")
    sources = [s.strip() for s in sources if s.strip()]

    if not sources:
        raise click.ClickException("No non-empty lines found in input file")

    click.echo(f"Processing {len(sources)} texts...")
    translations = batch_translate(sources, provider, cli, model)

    result = format_batch_output(translations, format)
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
@click.pass_context
def document(
    ctx: click.Context,
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
    """Translate a document file (xlsx, docx, pptx)."""
    config = ctx.obj.get("config", {})

    model = model or config.get("model")
    style_guide = style_guide or config.get("style_guide")
    format = format or config.get("format", "text")

    if config.get("check_style") is not None:
        check_style = config["check_style"]
    if config.get("review") is not None:
        review = config["review"]

    if not CLI_AVAILABLE:
        raise click.ClickException(
            "LLM integration not available. Install dependencies:\n"
            "  pip install anthropic openai requests"
        )

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
        providers_list = config.get("providers", [])
        cli_list = config.get("cli", [])

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
    except ImportError as exc:
        raise click.ClickException(f"Parser not available: {exc}")

    parsers_map = {
        ".xlsx": XLSXParser,
        ".docx": DOCXParser,
        ".pptx": PPTXParser,
    }

    parser = parsers_map[ext]()
    click.echo(f"Parsing {input_file}...")
    doc = parser.parse(str(input_file))
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
        except Exception as exc:
            click.echo(f"Warning: Failed to load style guide: {exc}", err=True)

    style_checker = None
    if check_style:
        try:
            from audit.style_checker import StyleChecker

            style_checker = StyleChecker(gengo_rules_enabled=True)
        except ImportError:
            click.echo("Warning: style_checker not available", err=True)

    segments_to_translate = []
    for idx, segment in enumerate(doc.segments):
        if not segment.text or not segment.text.strip():
            continue
        if not any("\u3040" <= c <= "\u9fff" for c in segment.text):
            continue
        segments_to_translate.append((idx, segment))

    click.echo(f"Translating {len(segments_to_translate)} Japanese segments...")

    if review:
        run_review_document_workflow(
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
        return

    run_simple_document_workflow(
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
    """Translate a document file (xlsx, docx, pptx)."""
    if not CLI_AVAILABLE:
        raise click.ClickException(
            "LLM integration not available. Install dependencies:\n"
            "  pip install anthropic openai requests"
        )

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
    except ImportError as exc:
        raise click.ClickException(f"Parser not available: {exc}")

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
        except Exception as exc:
            click.echo(f"Warning: Failed to load style guide: {exc}", err=True)

    style_checker = None
    if check_style:
        try:
            from audit.style_checker import StyleChecker

            style_checker = StyleChecker(gengo_rules_enabled=True)
        except ImportError:
            click.echo("Warning: style_checker not available", err=True)

    segments_to_translate = []
    for idx, segment in enumerate(doc.segments):
        if not segment.text or not segment.text.strip():
            continue
        if not any("\u3040" <= c <= "\u9fff" for c in segment.text):
            continue
        segments_to_translate.append((idx, segment))

    click.echo(f"Translating {len(segments_to_translate)} Japanese segments...")

    if review:
        run_review_document_workflow(
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
        return

    run_simple_document_workflow(
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


def main():
    cli()


if __name__ == "__main__":
    main()
