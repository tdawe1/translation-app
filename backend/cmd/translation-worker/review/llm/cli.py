"""CLI tool provider for translation."""

import asyncio
import json
import logging
import shutil
import subprocess
import time
from typing import Optional

from tenacity import (
    retry,
    stop_after_attempt,
    wait_exponential,
    retry_if_exception_type,
)

from .base import BaseProvider, ProviderConfig, ProviderResponse

logger = logging.getLogger(__name__)


class CLIProvider(BaseProvider):
    """Provider for CLI-based translation tools.

    SINGLE SOURCE OF TRUTH for CLI tool command mappings.
    All CLI tool configurations must be defined here.

    Supported Tools:
        Tool Name (internal)    Base Command    Description
        --------------------    -------------    ----------------------------
        claude_code             claude           Claude Code CLI (@anthropic-ai/claude-code)
        gemini_cli              gemini-cli       Gemini CLI (@google/generative-ai-cli)
        codex                   codex            GitHub Copilot Codex CLI (@github-copilot/codex-cli)
        ollama                  ollama           Ollama CLI (local models)

    How to Add a New CLI Tool:
        1. Add an entry to DEFAULT_COMMANDS below: {"tool_name": "base_command"}
        2. Update get_cli_provider() VALID_TOOLS list if needed
        3. Add install hint in review/cli.py install_hints dict
        4. Update tests in tests/test_review/test_cli.py

    Usage:
        provider = CLIProvider(tool_name="claude_code", config=config)
        response = provider.generate("Translate this text")

    The CLI tool mapping is used by:
        - CLIProvider.__init__(): resolves base_command from tool_name
        - review/cli.py: imports DEFAULT_COMMANDS for dry-run and validation

    Wraps command-line translation tools via subprocess calls.
    Returns structured results matching TranslationProvider protocol.
    """

    # SINGLE SOURCE OF TRUTH: CLI tool command configurations
    # When adding new tools, update this dictionary and document above
    DEFAULT_COMMANDS = {
        "claude_code": "claude",      # claude code exec "prompt"
        "gemini_cli": "gemini-cli",   # gemini-cli "prompt"
        "codex": "codex",             # codex exec "prompt"
        "ollama": "ollama",           # ollama run model "prompt"
    }

    def __init__(
        self, tool_name: str, config: ProviderConfig, command: Optional[str] = None
    ):
        super().__init__(config)
        self.tool_name = tool_name
        self.command = command or self.DEFAULT_COMMANDS.get(tool_name, tool_name)

    def is_available(self) -> bool:
        """Check if CLI tool is installed."""
        if not self.tool_name or self.tool_name.strip() == "":
            raise ValueError("tool name is required")

        command_path = shutil.which(self.command)
        if command_path is None:
            raise ValueError(f"CLI tool '{self.command}' not found in PATH")

        logger.info(f"CLI tool found: {command_path}")
        return True

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=60),
        retry=retry_if_exception_type(
            (subprocess.TimeoutExpired, subprocess.CalledProcessError),
        ),
    )
    def generate(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0,
    ) -> ProviderResponse:
        """Generate completion using CLI tool."""
        start = time.time()

        timeout = self.config.timeout

        cmd = self._build_command(prompt, max_tokens, temperature)

        logger.debug(f"Running CLI command: {self.command}")

        try:
            result = subprocess.run(
                cmd, capture_output=True, text=True, timeout=timeout, check=True
            )

            latency = int((time.time() - start) * 1000)

            return self._parse_response(result, latency)

        except subprocess.TimeoutExpired as e:
            logger.error(f"CLI tool timeout after {timeout}s: {e}")
            raise
        except subprocess.CalledProcessError as e:
            logger.error(f"CLI tool failed with exit code {e.returncode}: {e.stderr}")
            raise RuntimeError(f"CLI tool failed: {e.stderr}") from e

    def _build_command(
        self, prompt: str, max_tokens: Optional[int], temperature: float
    ) -> list:
        """Build command list for subprocess."""
        cmd = [self.command]

        if max_tokens:
            cmd.extend(["--max-tokens", str(max_tokens)])

        if temperature != 0.0:
            cmd.extend(["--temperature", str(temperature)])

        if self.config.api_key:
            cmd.extend(["--api-key", self.config.api_key])

        cmd.append(prompt)

        return cmd

    def _parse_response(
        self, result: subprocess.CompletedProcess, latency: int
    ) -> ProviderResponse:
        """Parse CLI output into ProviderResponse."""
        output_text = result.stdout.strip()

        try:
            data = json.loads(output_text)

            text = data.get("text", output_text)

            raw_response = {"stderr": result.stderr, "returncode": result.returncode}

            if "confidence" in data:
                raw_response["confidence"] = data["confidence"]

            usage = data.get(
                "usage",
                {
                    "prompt_tokens": 0,
                    "completion_tokens": 0,
                    "total_tokens": 0,
                },
            )

            return ProviderResponse(
                text=text,
                model=self.tool_name,
                usage=usage,
                latency_ms=latency,
                raw_response=raw_response,
            )

        except json.JSONDecodeError:
            return ProviderResponse(
                text=output_text,
                model=self.tool_name,
                usage={
                    "prompt_tokens": 0,
                    "completion_tokens": 0,
                    "total_tokens": 0,
                },
                latency_ms=latency,
                raw_response={
                    "stderr": result.stderr,
                    "returncode": result.returncode,
                },
            )

    async def generate_async(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0,
    ) -> ProviderResponse:
        """Async version using asyncio.to_thread."""
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            None, lambda: self.generate(prompt, max_tokens, temperature)
        )


def get_cli_provider(tool_name: str, api_key: str, **kwargs) -> CLIProvider:
    """Factory function to get CLI provider instance.

    Uses DEFAULT_COMMANDS as the single source of truth for valid tool names.

    Args:
        tool_name: One of the keys from CLIProvider.DEFAULT_COMMANDS
                   ("claude_code", "gemini_cli", "codex", "ollama")
        api_key: API key for CLI tool (may be empty for local CLI tools)
        **kwargs: Additional config (command, timeout, etc.)

    Returns:
        Configured CLIProvider instance

    Raises:
        ValueError: If tool_name is unknown (not in DEFAULT_COMMANDS)
    """
    # Use DEFAULT_COMMANDS as single source of truth for valid tools
    valid_tools = list(CLIProvider.DEFAULT_COMMANDS.keys())

    if tool_name not in valid_tools:
        raise ValueError(f"Unknown CLI tool: {tool_name}. Use: {valid_tools}")

    command = kwargs.pop("command", None)
    timeout = kwargs.pop("timeout", 120)

    config = ProviderConfig(api_key=api_key, timeout=timeout)

    return CLIProvider(tool_name=tool_name, config=config, command=command)
