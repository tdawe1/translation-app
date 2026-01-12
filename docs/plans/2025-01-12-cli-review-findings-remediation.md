# CLI Review Findings Remediation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all issues identified in the comprehensive code review of the translation worker CLI module, including removing duplicate code, fixing contract violations, and consolidating provider configurations.

**Architecture:** Clean up technical debt by (1) removing unused duplicate `llm/cli.py`, (2) fixing `LMStudioProvider` contract violation, (3) refactoring `cli.py` to use `CliToolProvider` abstraction, (4) consolidating CLI tool command mappings.

**Tech Stack:** Python 3.14, Click 8.x, pytest, asyncio, subprocess

---

## Task 1: Remove Unused Duplicate `llm/cli.py`

**Files:**
- Delete: `backend/cmd/translation-worker/review/llm/cli.py`
- Modify: `backend/cmd/translation-worker/review/llm/__init__.py` (remove import if present)
- Test: `backend/cmd/translation-worker/tests/test_providers/test_cli_provider.py` (verify still passes)

**Why:** `llm/cli.py` is a 210-line duplicate of `llm/cli_provider.py` and is completely unused. The active implementation is `cli_provider.py`.

**Step 1: Verify the file is truly unused**

Search for imports of the file:
```bash
cd /home/thomas/translation-app
grep -r "from.*llm.cli import" --include="*.py" .
grep -r "from review.llm.cli import" --include="*.py" .
grep -r "import.*llm.cli" --include="*.py" .
```

Expected: No results (or only in comments)

**Step 2: Verify cli_provider.py has all needed functionality**

Read both files and compare:
```bash
head -50 backend/cmd/translation-worker/review/llm/cli.py
head -50 backend/cmd/translation-worker/review/llm/cli_provider.py
```

Expected: `cli_provider.py` has the same or better implementation

**Step 3: Run tests to establish baseline**

```bash
cd /home/thomas/translation-app
python -m pytest backend/cmd/translation-worker/tests/test_providers/ -v
```

Expected: All tests pass

**Step 4: Delete the duplicate file**

```bash
rm backend/cmd/translation-worker/review/llm/cli.py
```

**Step 5: Verify tests still pass after deletion**

```bash
python -m pytest backend/cmd/translation-worker/tests/test_providers/ -v
python -m pytest backend/cmd/translation-worker/tests/test_review/ -v
```

Expected: All tests still pass (file was unused)

**Step 6: Check and update llm/__init__.py**

```bash
cat backend/cmd/translation-worker/review/llm/__init__.py
```

If it imports from `.cli`, remove those imports.

**Step 7: Commit**

```bash
git add backend/cmd/translation-worker/review/llm/cli.py
git add backend/cmd/translation-worker/review/llm/__init__.py
git commit -m "refactor(llm): remove unused duplicate llm/cli.py

- llm/cli.py was a 210-line duplicate of cli_provider.py
- The active CLI provider implementation is in cli_provider.py
- No code references llm.cli, confirmed via grep scan
- All tests pass after removal"
```

---

## Task 2: Fix LMStudioProvider Contract Violation

**Files:**
- Modify: `backend/cmd/translation-worker/review/llm/lm_studio.py:100-120`
- Test: `backend/cmd/translation-worker/tests/test_providers/test_lm_studio_provider.py`

**Why:** `LMStudioProvider.generate()` returns `str` instead of `ProviderResponse`, breaking the Liskov Substitution Principle defined in `BaseProvider`.

**Step 1: Read the base contract**

```bash
cat backend/cmd/translation-worker/review/llm/base.py
```

Note: `BaseProvider.generate()` returns `ProviderResponse`

**Step 2: Read the violating implementation**

```bash
sed -n '100,120p' backend/cmd/translation-worker/review/llm/lm_studio.py
```

Current implementation returns `str` instead of `ProviderResponse`.

**Step 3: Read the existing test**

```bash
cat backend/cmd/translation-worker/tests/test_providers/test_lm_studio_provider.py
```

**Step 4: Write failing test for correct return type**

Create/modify test file:

```python
# File: backend/cmd/translation-worker/tests/test_providers/test_lm_studio_provider.py

from review.llm.lm_studio import LMStudioProvider
from review.llm.base import ProviderResponse
import pytest

def test_generate_returns_provider_response():
    """LMStudioProvider.generate() should return ProviderResponse, not str."""
    provider = LMStudioProvider(base_url="http://localhost:1234", model="test")

    # Mock the HTTP request
    from unittest.mock import patch, MagicMock
    mock_response = MagicMock()
    mock_response.json.return_value = {
        "choices": [{"text": "Test translation"}]
    }
    mock_response.raise_for_status = MagicMock()

    with patch("requests.post", return_value=mock_response):
        result = provider.generate("Translate this")

    # Should return ProviderResponse, not str
    assert isinstance(result, ProviderResponse)
    assert hasattr(result, "text")
    assert hasattr(result, "model")
    assert hasattr(result, "usage")
    assert result.text == "Test translation"
```

**Step 5: Run test to verify it fails**

```bash
cd /home/thomas/translation-app
python -m pytest backend/cmd/translation-worker/tests/test_providers/test_lm_studio_provider.py::test_generate_returns_provider_response -v
```

Expected: FAIL with "str has no attribute 'text'" or similar

**Step 6: Fix the implementation**

Edit `backend/cmd/translation-worker/review/llm/lm_studio.py`:

```python
# Around line 100-120, modify the generate() method

def generate(
    self,
    prompt: str,
    max_tokens: Optional[int] = None,
    temperature: float = 0.0
) -> ProviderResponse:
    """Generate completion using LM Studio API.

    Returns:
        ProviderResponse: Standardized response object
    """
    import time
    from .base import ProviderResponse  # Import the response class

    start = time.time()

    # Build request for LM Studio (OpenAI-compatible format)
    request_body = {
        "model": self._config.model,
        "messages": [{"role": "user", "content": prompt}],
        "temperature": temperature,
    }
    if max_tokens:
        request_body["max_tokens"] = max_tokens

    response = self._http_session.post(
        f"{self._config.base_url}/v1/chat/completions",
        json=request_body,
        timeout=self._config.timeout,
    )
    response.raise_for_status()

    data = response.json()
    latency = int((time.time() - start) * 1000)

    # Extract text from LM Studio response
    text = data["choices"][0]["message"]["content"]

    # Return ProviderResponse instead of str
    return ProviderResponse(
        text=text,
        model=self._config.model,
        usage={
            "prompt_tokens": data.get("usage", {}).get("prompt_tokens", len(prompt)),
            "completion_tokens": data.get("usage", {}).get("completion_tokens", len(text)),
            "total_tokens": data.get("usage", {}).get("total_tokens", len(prompt) + len(text)),
        },
        latency_ms=latency,
        raw_response=data
    )
```

**Step 7: Run test to verify it passes**

```bash
python -m pytest backend/cmd/translation-worker/tests/test_providers/test_lm_studio_provider.py::test_generate_returns_provider_response -v
```

Expected: PASS

**Step 8: Run all LMStudio tests**

```bash
python -m pytest backend/cmd/translation-worker/tests/test_providers/test_lm_studio_provider.py -v
```

Expected: All tests pass

**Step 9: Commit**

```bash
git add backend/cmd/translation-worker/review/llm/lm_studio.py
git add backend/cmd/translation-worker/tests/test_providers/test_lm_studio_provider.py
git commit -m "fix(llm): LMStudioProvider now returns ProviderResponse

- Fixes contract violation with BaseProvider abstract class
- Previously returned str, now returns ProviderResponse with:
  - text: translated content
  - model: model identifier
  - usage: token counts
  - latency_ms: request timing
- Maintains backward compatibility for text access via .text property"
```

---

## Task 3: Refactor `cli.py` to Use `CliToolProvider`

**Files:**
- Modify: `backend/cmd/translation-worker/review/cli.py:53-114` (remove `_translate_with_cli` function)
- Modify: `backend/cmd/translation-worker/review/cli.py:260-263` (use CliToolProvider)
- Modify: `backend/cmd/translation-worker/review/cli.py:381-392` (use CliToolProvider for judge)
- Modify: `backend/cmd/translation-worker/review/cli.py:506-512` (use CliToolProvider for batch)
- Test: `backend/cmd/translation-worker/tests/test_review/test_cli.py`

**Why:** The CLI layer has direct subprocess calls via `_translate_with_cli()`, bypassing the provider abstraction. This creates code duplication and makes testing harder.

**Step 1: Read CliToolProvider interface**

```bash
cat backend/cmd/translation-worker/review/llm/cli_provider.py
```

Note: `CliToolProvider` has a `generate()` method that takes a prompt.

**Step 2: Read current cli.py implementation**

```bash
sed -n '53,114p' backend/cmd/translation-worker/review/cli.py
```

**Step 3: Write test for refactored behavior**

```python
# File: backend/cmd/translation-worker/tests/test_review/test_cli.py

from unittest.mock import patch, MagicMock

def test_translate_uses_cli_tool_provider():
    """translate command should use CliToolProvider, not direct subprocess."""
    # Mock CliToolProvider instead of subprocess
    mock_provider = MagicMock()
    mock_provider.generate.return_value = MagicMock(
        text="Hello World",
        model="claude-default",
        usage={"tool": "claude"}
    )

    with patch("review.cli.CliToolProvider", return_value=mock_provider):
        result = runner.invoke(translate, ["--cli", "claude", "こんにちは"])

    assert result.exit_code == 0
    assert "Hello World" in result.output
```

**Step 4: Run test to verify it fails**

```bash
python -m pytest backend/cmd/translation-worker/tests/test_review/test_cli.py::test_translate_uses_cli_tool_provider -v
```

Expected: FAIL (CliToolProvider not imported/used)

**Step 5: Refactor cli.py to use CliToolProvider**

Edit `backend/cmd/translation-worker/review/cli.py`:

```python
# Add import at top of file
try:
    from .llm import get_provider
    from .llm.cli_provider import CliToolProvider  # ADD THIS
    from .multimodel import MultiModelTranslator
    from .judge import TranslationJudge
    from .models import TranslationCandidate
    CLI_AVAILABLE = True
except ImportError:
    CLI_AVAILABLE = False

# REMOVE the entire _translate_with_cli function (lines 53-114)
# DELETE THIS FUNCTION:

# def _translate_with_cli(text: str, cli_tool: str) -> tuple[str, dict]:
#     """Translate using local CLI tool (no API costs)."""
#     ... ENTIRE FUNCTION DELETED ...

# UPDATE the translate function to use CliToolProvider (around line 260-263):
# Find this code in translate():
    try:
        if cli:
            # OLD: translation, usage = _translate_with_cli(text, cli)
            # NEW:
            cli_provider = CliToolProvider(tool=cli)
            response = cli_provider.generate(f"Translate the following Japanese text to English:\n\n{text}")
            translation = response.text
            usage = response.usage
            provider_name = usage.get("tool", cli)
            model_name = response.model
        else:
            # ... existing API provider code ...
```

**Step 6: Update judge command to use CliToolProvider**

Edit `backend/cmd/translation-worker/review/cli.py` (around line 379-392):

```python
# In judge() function, find:
    try:
        if cli:
            # OLD: judgment, usage = _translate_with_cli(prompt, cli)
            # NEW:
            cli_provider = CliToolProvider(tool=cli)
            response = cli_provider.generate(prompt)
            judgment = response.text

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
```

**Step 7: Update batch command to use CliToolProvider**

Edit `backend/cmd/translation-worker/review/cli.py` (around line 505-512):

```python
# In batch() function, find:
        try:
            if cli:
                # OLD: translation, usage = _translate_with_cli(source, cli)
                # NEW:
                cli_provider = CliToolProvider(tool=cli)
                response = cli_provider.generate(f"Translate the following Japanese text to English:\n\n{source}")
                translation = response.text
                translations.append({
                    "source": source,
                    "translation": translation,
                    "usage": response.usage
                })
```

**Step 8: Remove CLI_TOOLS dictionary** (no longer needed)

```bash
# Delete these lines from cli.py:
# Lines 44-50 approximately:
# CLI_TOOLS = {
#     "claude": ["claude", "code", "exec"],
#     "codex": ["codex", "exec"],
#     "gemini": ["gemini-cli"],
#     "ollama": ["ollama", "run"],
# }
```

**Step 9: Update dry-run to use CliToolProvider**

Edit `backend/cmd/translation-worker/review/cli.py` (around line 240-254):

```python
    # Handle dry-run for CLI tools
    if dry_run:
        if not cli:
            raise click.ClickException(
                "--dry-run only works with --cli (local CLI tools). "
                "For API providers, the request goes to an external service."
            )
        # Use CliToolProvider to show the command
        from .llm.cli_provider import CliToolProvider
        provider = CliToolProvider(tool=cli)
        cmd = provider._build_command(f"Translate the following Japanese text to English:\n\n{text}")
        click.echo(f"Would execute: {' '.join(cmd[:2])}...")
        click.echo(f"Full command: {' '.join(repr(c) if ' ' in c else c for c in cmd)}")
        return
```

**Step 10: Run all tests**

```bash
python -m pytest backend/cmd/translation-worker/tests/test_review/test_cli.py -v
```

Expected: All 17 tests pass

**Step 11: Update CLI_TOOLS reference if any**

```bash
grep -n "CLI_TOOLS" backend/cmd/translation-worker/review/cli.py
```

If any remain, update them to use CliToolProvider.

**Step 12: Commit**

```bash
git add backend/cmd/translation-worker/review/cli.py
git add backend/cmd/translation-worker/tests/test_review/test_cli.py
git commit -m "refactor(cli): use CliToolProvider instead of direct subprocess

- Removes _translate_with_cli() helper function
- translate, judge, batch commands now use CliToolProvider abstraction
- Eliminates code duplication between CLI layer and provider layer
- Dry-run now uses CliToolProvider._build_command() for consistency
- All 17 CLI tests continue to pass"
```

---

## Task 4: Consolidate CLI Tool Command Mappings

**Files:**
- Modify: `backend/cmd/translation-worker/review/llm/cli_provider.py:34-39`
- Optionally create shared constants file

**Why:** `CLI_TOOLS` in cli.py and `CLI_COMMANDS` in cli_provider.py are duplicate mappings. After Task 3 removes `CLI_TOOLS`, we still need to ensure the mapping is in a single place.

**Step 1: Verify CLI_COMMANDS is the single source**

```bash
grep -n "CLI_TOOLS\|CLI_COMMANDS" backend/cmd/translation-worker/review/*.py
```

Expected: Only `CLI_COMMANDS` remains (in cli_provider.py)

**Step 2: Verify all tools are represented**

```bash
sed -n '34,39p' backend/cmd/translation-worker/review/llm/cli_provider.py
```

Should show:
```python
CLI_COMMANDS = {
    "claude": ["claude", "code", "exec"],
    "codex": ["codex", "exec"],
    "gemini": ["gemini-cli", "generate"],
    "ollama": ["ollama", "run"],
}
```

**Step 3: Add docstring explaining the mapping**

Edit `backend/cmd/translation-worker/review/llm/cli_provider.py`:

```python
class CliToolProvider(BaseProvider):
    """Provider that shells out to local CLI tools.

    Uses subprocess to call CLI tools directly, avoiding API costs
    for development and internal use.

    Supported Tools:
    - claude: claude code exec "prompt" (Claude Code CLI)
    - codex: codex exec "prompt" (GitHub Copilot Codex CLI)
    - gemini: gemini-cli "prompt" (Gemini CLI)
    - ollama: ollama run model "prompt" (Ollama)
    """

    # CLI command configurations - SINGLE SOURCE OF TRUTH
    # When adding new tools, update this mapping and tests/test_cli_tools.py
    CLI_COMMANDS = {
```

**Step 4: Write test for command mapping**

```python
# File: backend/cmd/translation-worker/tests/test_providers/test_cli_tools.py

def test_all_cli_tools_have_commands():
    """Every supported CLI tool must have a command mapping."""
    from review.llm.cli_provider import CliToolProvider

    expected_tools = ["claude", "codex", "gemini", "ollama"]

    for tool in expected_tools:
        assert tool in CliToolProvider.CLI_COMMANDS, f"{tool} missing from CLI_COMMANDS"
        assert len(CliToolProvider.CLI_COMMANDS[tool]) > 0, f"{tool} has empty command list"
```

**Step 5: Run test**

```bash
python -m pytest backend/cmd/translation-worker/tests/test_providers/test_cli_tools.py::test_all_cli_tools_have_commands -v
```

**Step 6: Commit**

```bash
git add backend/cmd/translation-worker/review/llm/cli_provider.py
git add backend/cmd/translation-worker/tests/test_providers/test_cli_tools.py
git commit -m "refactor(llm): consolidate CLI command mappings as single source

- CLI_COMMANDS in cli_provider.py is now the only mapping
- Removed duplicate CLI_TOOLS from cli.py
- Added comprehensive docstring listing all supported tools
- Added test to verify all tools have command mappings"
```

---

## Task 5: Add Type Hint for Unused `parallel` Parameter

**Files:**
- Modify: `backend/cmd/translation-worker/review/cli.py:201-203` and `:453-455`

**Why:** The `--parallel` flag exists but has no effect. At minimum, document its reserved status.

**Step 1: Add TODO comment for parallel flag**

Edit `backend/cmd/translation-worker/review/cli.py`:

```python
@click.option(
    "--parallel/--sequential",
    default=True,
    # TODO: Implement parallel execution for batch processing
    # Currently reserved for future use - has no effect on execution
    help="Use parallel execution (reserved for future use)"
)
```

Do this for both occurrences (translate and batch commands).

**Step 2: Commit**

```bash
git add backend/cmd/translation-worker/review/cli.py
git commit -m "docs(cli): mark --parallel flag as reserved

- Flag currently has no effect on execution
- Documented as reserved for future parallel batch processing
- Implementation will use asyncio.gather() for concurrent CLI calls"
```

---

## Task 6: Add File Size Limits for Batch Processing

**Files:**
- Modify: `backend/cmd/translation-worker/review/cli.py:488-496`

**Why:** Security consideration - prevent unbounded memory usage from large input files.

**Step 1: Write test for file size limit**

```python
# File: backend/cmd/translation-worker/tests/test_review/test_cli.py

from click.testing import CliRunner

def test_batch_rejects_oversized_files():
    """Should reject input files exceeding size limit."""
    runner = CliRunner()
    with runner.isolated_filesystem():
        # Create file exceeding limit (10MB = 10,485,760 bytes)
        large_text = "x" * (11 * 1024 * 1024)  # 11MB
        with open("huge.txt", "w") as f:
            f.write(large_text)

        result = runner.invoke(batch, [
            "--input", "huge.txt",
            "--output", "out.txt",
            "--cli", "claude"
        ])

        assert result.exit_code != 0
        assert "too large" in result.output.lower() or "size limit" in result.output.lower()
```

**Step 2: Run test to verify it fails**

```bash
python -m pytest backend/cmd/translation-worker/tests/test_review/test_cli.py::test_batch_rejects_oversized_files -v
```

Expected: FAIL (no size limit currently)

**Step 3: Implement file size check**

Edit `backend/cmd/translation-worker/review/cli.py`:

```python
# In batch() function, after reading input (around line 490)

def batch(...):
    """..."""

    # Read input with size check
    input_path = Path(input)

    # Check file size before reading (10MB limit)
    MAX_FILE_SIZE = 10 * 1024 * 1024  # 10MB
    file_size = input_path.stat().st_size
    if file_size > MAX_FILE_SIZE:
        size_mb = file_size / (1024 * 1024)
        raise click.ClickException(
            f"Input file too large: {size_mb:.1f}MB. "
            f"Maximum size is {MAX_FILE_SIZE / (1024 * 1024)}MB."
        )

    sources = input_path.read_text(encoding="utf-8").strip().split("\n")
    # ... rest of function
```

**Step 4: Run test to verify it passes**

```bash
python -m pytest backend/cmd/translation-worker/tests/test_review/test_cli.py::test_batch_rejects_oversized_files -v
```

**Step 5: Run all batch tests**

```bash
python -m pytest backend/cmd/translation-worker/tests/test_review/test_cli.py -k batch -v
```

**Step 6: Commit**

```bash
git add backend/cmd/translation-worker/review/cli.py
git add backend/cmd/translation-worker/tests/test_review/test_cli.py
git commit -m "feat(cli): add 10MB file size limit for batch processing

- Prevents unbounded memory usage from large input files
- Clear error message when limit exceeded
- Security improvement for DoS prevention"
```

---

## Task 7: Standardize Error Message Format

**Files:**
- Modify: Various error messages in `cli.py`

**Why:** Consistent error messaging improves UX and debugging.

**Step 1: Create error message templates**

Edit `backend/cmd/translation-worker/review/cli.py` (add at top after imports):

```python
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
```

**Step 2: Update _translate_with_cli error to use template**

Note: This function will be removed in Task 3, but for consistency:

```python
# Old format:
raise click.ClickException(
    f"CLI tool '{cmd_name}' not found in PATH.\n"
    f"Install it first:\n"
    f"  - claude: npm install -g @anthropic-ai/claude-code\n"
    ...
)

# New format (if function still exists):
install_cmds = "\n".join(CLI_INSTALL_INSTRUCTIONS.values())
raise click.ClickException(
    ERROR_TEMPLATES["cli_not_found"].format(
        tool=cli_tool,
        install=install_cmds
    )
)
```

**Step 3: Update mutual exclusivity errors**

```python
# Old format:
raise click.ClickException(
    "Cannot specify both --provider and --cli. "
    "Use --cli for local tools or --provider for API-based providers."
)

# New format:
raise click.ClickException(
    ERROR_TEMPLATES["mutual_exclusive"].format(
        opt1="provider",
        opt2="cli",
        suggestion="--cli for local tools or --provider for API-based providers"
    )
)
```

**Step 4: Run tests to ensure no breakage**

```bash
python -m pytest backend/cmd/translation-worker/tests/test_review/test_cli.py -v
```

**Step 5: Commit**

```bash
git add backend/cmd/translation-worker/review/cli.py
git commit -m "refactor(cli): standardize error message format

- Added ERROR_TEMPLATES for consistent messaging
- Added CLI_INSTALL_INSTRUCTIONS for tool-specific help
- Improves debugging and user experience"
```

---

## Task 8: Update Module Documentation

**Files:**
- Create: `backend/cmd/translation-worker/review/README.md`

**Why:** CLI help text is good but a separate README provides better discoverability and examples.

**Step 1: Create CLI README**

Create `backend/cmd/translation-worker/review/README.md`:

```markdown
# Translation Worker CLI

Command-line interface for Japanese-to-English translation with LLM provider support.

## Quick Start

```bash
# Using local CLI tools (no API costs!)
python -m review translate "こんにちは" --cli claude

# Using API providers (requires API key)
python -m review translate "こんにちは" --provider anthropic

# Batch processing
python -m review batch --input sources.txt --output translations.txt --cli claude

# Compare translations
python -m review judge source.txt trans_a.txt trans_b.txt --cli claude
```

## Installation

```bash
# For API providers
pip install anthropic openai

# For CLI tools (no API costs!)
npm install -g @anthropic-ai/claude-code
npm install -g @github-copilot/codex-cli
npm install -g @google/generative-ai-cli
```

## Commands

### translate

Translate Japanese text to English.

```bash
python -m review translate "こんにちは" --cli claude
python -m review translate "こんにちは" --cli codex
python -m review translate "こんにちは" --provider anthropic --format json
```

### batch

Batch translate from file (one line per source).

```bash
python -m review batch --input sources.txt --output translations.txt --cli claude --format csv
```

### judge

Compare two translations and select winner.

```bash
python -m review judge source.txt translation_a.txt translation_b.txt --cli claude
```

## Options

| Option | Description |
|--------|-------------|
| `--provider` | Use API provider (anthropic, openai, gemini) |
| `--cli` | Use local CLI tool (claude, codex, gemini, ollama) |
| `--model` | Specify model identifier |
| `--format` | Output format: text, json, csv |
| `--output` | Write to file instead of stdout |
| `--dry-run` | Show command without executing (CLI only) |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `OPENAI_API_KEY` | OpenAI API key |
| `GEMINI_API_KEY` | Google Gemini API key |
| `GEMINI_PROJECT_ID` | Google Cloud project ID |
| `GEMINI_LOCATION` | Gemini region (default: us-central1) |
```

**Step 2: Update main __init__.py to reference README**

Edit `backend/cmd/translation-worker/review/__init__.py`:

```python
"""
Bilingual translation review workflow system.

... existing content ...

## CLI Usage

For command-line interface documentation, see README.md in this directory.
"""
```

**Step 3: Commit**

```bash
git add backend/cmd/translation-worker/review/README.md
git add backend/cmd/translation-worker/review/__init__.py
git commit -m "docs(cli): add comprehensive CLI README

- Quick start guide with examples
- Installation instructions for all tools
- Command reference with options
- Environment variable documentation"
```

---

## Task 9: Final Verification

**Files:**
- All modified files

**Step 1: Run full test suite**

```bash
cd /home/thomas/translation-app
python -m pytest backend/cmd/translation-worker/tests/ -v --tb=short
```

Expected: All tests pass

**Step 2: Run CLI tests specifically**

```bash
python -m pytest backend/cmd/translation-worker/tests/test_review/test_cli.py -v
```

Expected: All tests pass (count may be higher due to new tests)

**Step 3: Verify imports work**

```bash
cd backend/cmd/translation-worker
python -c "from review import cli; print('CLI import OK')"
python -c "from review.llm.cli_provider import CliToolProvider; print('CliToolProvider import OK')"
python -c "from review.llm.lm_studio import LMStudioProvider; print('LMStudioProvider import OK')"
```

Expected: All imports succeed

**Step 4: Check for remaining TODOs**

```bash
grep -rn "TODO\|FIXME\|XXX" backend/cmd/translation-worker/review/ --include="*.py"
```

Document any remaining TODOs in project notes.

**Step 5: Create summary commit**

```bash
git add -A
git commit -m "docs: complete CLI review findings remediation

All high and medium priority items from comprehensive review addressed:
- Removed unused llm/cli.py duplicate
- Fixed LMStudioProvider contract violation
- Refactored cli.py to use CliToolProvider abstraction
- Consolidated CLI command mappings
- Added file size limits for batch processing
- Standardized error message format
- Added comprehensive CLI README

Test Results:
- All CLI tests passing
- All provider tests passing
- No breaking changes to public API
- 100% backward compatible"
```

---

## Summary

This plan addresses **all suggestions** from the comprehensive code review:

1. ✅ Remove unused `llm/cli.py` (Task 1)
2. ✅ Fix `LMStudioProvider` contract violation (Task 2)
3. ✅ Refactor `cli.py` to use `CliToolProvider` (Task 3)
4. ✅ Consolidate CLI command mappings (Task 4)
5. ✅ Document `--parallel` flag as reserved (Task 5)
6. ✅ Add file size limits for security (Task 6)
7. ✅ Standardize error message format (Task 7)
8. ✅ Add comprehensive CLI documentation (Task 8)

**Estimated Time**: 3-4 hours for all tasks
**Test Coverage**: Maintained or improved
**Breaking Changes**: None (backward compatible)
