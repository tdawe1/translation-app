import json
import logging
import os
import shutil
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

import click

try:
    from .llm import get_provider
    from .llm.cli import CLIProvider

    CLI_AVAILABLE = True
    DEFAULT_COMMANDS = CLIProvider.DEFAULT_COMMANDS if CLIProvider else {}
    _get_provider_func = get_provider
except ImportError:
    CLI_AVAILABLE = False
    CLIProvider = None  # type: ignore
    DEFAULT_COMMANDS = {}  # type: ignore
    _get_provider_func = None  # type: ignore

logger = logging.getLogger(__name__)

_SHORT_TO_INTERNAL = {
    "claude": "claude_code",
    "codex": "codex",
    "gemini": "gemini_cli",
    "ollama": "ollama",
}

MAX_BATCH_FILE_SIZE_MB = 10
MAX_BATCH_FILE_SIZE = MAX_BATCH_FILE_SIZE_MB * 1024 * 1024

ERROR_TEMPLATES = {
    "cli_not_found": "CLI tool '{tool}' not found in PATH.\n\nInstall instructions:\n{install}",
    "mutual_exclusive": "Cannot specify both --{opt1} and --{opt2}. Use {suggestion}.",
    "missing_required": "Must specify either --{opt1} or --{opt2}.",
    "file_too_large": "Input file too large: {size_mb:.1f}MB. Maximum is {max_mb}MB.",
}

CLI_INSTALL_INSTRUCTIONS = {
    "claude": "  claude: npm install -g @anthropic-ai/claude-code",
    "codex": "  codex: npm install -g @github-copilot/codex-cli",
    "gemini": "  gemini: npm install -g @google/generative-ai-cli",
    "ollama": "  ollama: curl -fsSL https://ollama.com/install.sh | sh",
}


def get_cli_available() -> bool:
    return CLI_AVAILABLE


def get_error_templates() -> Dict[str, str]:
    return ERROR_TEMPLATES


def _get_api_key(provider: str) -> Optional[str]:
    env_var = f"{provider.upper()}_API_KEY"
    return os.environ.get(env_var)


def _get_cli_provider(tool: str) -> Any:
    if CLIProvider is None:
        raise click.ClickException("CLIProvider not available")

    from .llm.base import ProviderConfig

    if tool not in _SHORT_TO_INTERNAL:
        raise click.ClickException(
            f"Unknown CLI tool: {tool}. Use: {list(_SHORT_TO_INTERNAL.keys())}"
        )

    tool_name = _SHORT_TO_INTERNAL[tool]
    base_command = DEFAULT_COMMANDS.get(tool_name, tool)

    if not shutil.which(base_command):
        hint = CLI_INSTALL_INSTRUCTIONS.get(tool, f"  {tool}: Install {base_command}")
        raise click.ClickException(
            ERROR_TEMPLATES["cli_not_found"].format(tool=base_command, install=hint)
        )

    config = ProviderConfig(api_key="", timeout=120)

    return CLIProvider(tool_name=tool_name, config=config, command=base_command)


def _build_cli_command(tool: str, prompt: str) -> List[str]:
    if tool not in _SHORT_TO_INTERNAL:
        raise click.ClickException(f"Unknown CLI tool: {tool}")

    if tool == "claude":
        cmd = ["claude", "code", "exec"]
    elif tool == "codex":
        cmd = ["codex", "exec"]
    elif tool == "gemini":
        cmd = ["gemini-cli"]
    elif tool == "ollama":
        cmd = ["ollama", "run", "codellama:latest"]
    else:
        tool_name = _SHORT_TO_INTERNAL[tool]
        base_command = DEFAULT_COMMANDS.get(tool_name, tool)
        cmd = [base_command]

    cmd.append(prompt)
    return cmd


def _ensure_provider(provider: str, model: Optional[str] = None) -> Any:
    if not CLI_AVAILABLE or _get_provider_func is None:
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

        return _get_provider_func(provider, api_key, model, **kwargs)
    except Exception as exc:
        raise click.ClickException(f"Failed to initialize {provider} provider: {exc}")


def validate_provider_cli(provider: Optional[str], cli: Optional[str]) -> None:
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


def translate_text(
    text: str,
    provider: Optional[str],
    cli: Optional[str],
    model: Optional[str],
) -> Tuple[str, str, str, Dict[str, Any]]:
    prompt = f"Translate the following Japanese text to English:\n\n{text}"

    if cli:
        cli_provider = _get_cli_provider(cli)
        response = cli_provider.generate(prompt)
        translation = response.text.strip()
        usage = response.usage or {}
        usage["tool"] = cli
        provider_name = cli
        model_name = response.model or f"{cli}-default"
    else:
        if provider is None:
            raise click.ClickException("Provider name is required when not using CLI")
        provider_instance = _ensure_provider(provider, model)
        response = provider_instance.generate(prompt)
        translation = response.text.strip()
        provider_name = provider or "unknown"
        model_name = response.model
        usage = response.usage

    return translation, provider_name, model_name, usage


def format_translation_output(
    text: str,
    translation: str,
    provider_name: str,
    model_name: str,
    usage: Dict[str, Any],
    format: str,
) -> str:
    if format == "text":
        return translation
    if format == "json":
        return json.dumps(
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
    if format == "csv":
        return (
            "source,translation,provider,model\n"
            f'"{text}","{translation}","{provider_name}","{model_name}"'
        )
    return translation


def build_cli_dry_run(tool: str, prompt: str) -> str:
    cmd = _build_cli_command(tool, prompt)
    short = f"Would execute: {' '.join(cmd[:2])}..."
    full = f"Full command: {' '.join(repr(c) if ' ' in c else c for c in cmd)}"
    return f"{short}\n{full}"


def read_text_file(path: str) -> str:
    return Path(path).read_text(encoding="utf-8").strip()


def validate_batch_file_size(path: Path) -> None:
    file_size = path.stat().st_size
    if file_size > MAX_BATCH_FILE_SIZE:
        size_mb = file_size / (1024 * 1024)
        raise click.ClickException(
            ERROR_TEMPLATES["file_too_large"].format(
                size_mb=size_mb, max_mb=MAX_BATCH_FILE_SIZE_MB
            )
        )


def batch_translate(
    sources: List[str],
    provider: Optional[str],
    cli: Optional[str],
    model: Optional[str],
) -> List[Dict[str, Any]]:
    translations = []
    for index, source in enumerate(sources):
        logger.info("Processing %d/%d", index + 1, len(sources))
        try:
            if cli:
                cli_provider = _get_cli_provider(cli)
                prompt = (
                    f"Translate the following Japanese text to English:\n\n{source}"
                )
                response = cli_provider.generate(prompt)
                usage = response.usage or {}
                usage["tool"] = cli
                translations.append(
                    {
                        "source": source,
                        "translation": response.text.strip(),
                        "usage": usage,
                    }
                )
            else:
                if provider is None:
                    raise click.ClickException(
                        "Provider name required when not using CLI"
                    )
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
        except Exception as exc:
            logger.warning("Failed to translate line %d: %s", index + 1, exc)
            translations.append(
                {"source": source, "translation": f"[ERROR: {exc}]", "usage": {}}
            )
    return translations


def format_batch_output(translations: List[Dict[str, Any]], format: str) -> str:
    if format == "text":
        return "\n".join(item["translation"] for item in translations)
    if format == "json":
        return json.dumps(translations, indent=2, ensure_ascii=False)
    if format == "csv":
        result = "source,translation,prompt_tokens,completion_tokens\n"
        for item in translations:
            safe_source = item["source"].replace('"', '""')
            safe_translation = item["translation"].replace('"', '""')
            result += (
                f'"{safe_source}","{safe_translation}",'
                f"{item['usage'].get('prompt_tokens', 0)},"
                f"{item['usage'].get('completion_tokens', 0)}\n"
            )
        return result
    return "\n".join(item["translation"] for item in translations)


def judge_translation(
    source_text: str,
    text_a: str,
    text_b: str,
    provider: Optional[str],
    cli: Optional[str],
    model: str,
) -> Tuple[Dict[str, Any], str]:
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
    \"winner\": \"translation_a\" | \"translation_b\" | \"tie\",
    \"confidence\": 0.0-1.0,
    \"reasoning\": \"Brief explanation\",
    \"concerns\": [\"list any concerns found\"]
}}

Evaluate both translations and output JSON."""

    if cli:
        cli_provider = _get_cli_provider(cli)
        response = cli_provider.generate(prompt)
        judgment = response.text.strip()
        provider_name = cli
        try:
            result_data = json.loads(judgment)
        except json.JSONDecodeError:
            result_data = {
                "winner": "unknown",
                "confidence": 0.0,
                "reasoning": judgment,
                "concerns": ["Failed to parse JSON from CLI output"],
            }
    else:
        provider_instance = _ensure_provider(provider or "anthropic", model)
        response = provider_instance.generate(prompt)
        provider_name = provider or "anthropic"
        try:
            result_data = json.loads(response.text)
        except json.JSONDecodeError:
            result_data = {
                "winner": "unknown",
                "confidence": 0.0,
                "reasoning": response.text,
                "concerns": ["Failed to parse JSON"],
            }

    return result_data, provider_name


def format_judge_output(result_data: Dict[str, Any], format: str) -> str:
    if format == "json":
        return json.dumps(result_data, indent=2)

    winner = result_data.get("winner", "unknown")
    return (
        f"Winner: {winner}\n"
        f"Confidence: {result_data.get('confidence', 0):.2f}\n"
        f"Reasoning: {result_data.get('reasoning', 'N/A')}"
    )


def run_simple_document_workflow(
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
            if provider_name is None:
                return (idx, seg, None, "No provider configured")
            provider_instance = _ensure_provider(provider_name, model)
            response = provider_instance.generate(prompt)
            return (idx, seg, response.text.strip(), None)
        except Exception as exc:
            return (idx, seg, None, str(exc))

    all_issues: List[Dict[str, Any]] = []
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


def run_review_document_workflow(
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
    all_issues: List[Dict[str, Any]] = []

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
    click.echo("REVIEW WORKFLOW COMPLETE")
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
