"""CLI provider that shells out to local LLM CLI tools.

Supports:
- claude code exec "prompt" - Claude Code CLI
- codex exec "prompt" - GitHub Copilot Codex CLI
- gemini-cli "prompt" - Gemini CLI
"""
import json
import logging
import subprocess
import time
from typing import Optional

from .base import BaseProvider, ProviderConfig, ProviderResponse

logger = logging.getLogger(__name__)


class CliToolProvider(BaseProvider):
    """Provider that shells out to local CLI tools.

    Uses subprocess to call CLI tools directly, avoiding API costs
    for development and internal use.

    Example:
        provider = CliToolProvider(
            tool="claude",  # or "codex", "gemini"
            model="claude-sonnet-4-5-20250929"
        )
        response = provider.generate("Translate: こんにちは")
    """

    # CLI command configurations
    CLI_COMMANDS = {
        "claude": ["claude", "code", "exec"],
        "codex": ["codex", "exec"],
        "gemini": ["gemini-cli", "generate"],
        "ollama": ["ollama", "run"],
    }

    def __init__(self, tool: str, model: str = None, base_url: str = None):
        """Initialize CLI tool provider.

        Args:
            tool: CLI tool name ("claude", "codex", "gemini", "ollama")
            model: Model identifier (for tools that support it)
            base_url: Not used for CLI tools (kept for interface compatibility)
        """
        if tool not in self.CLI_COMMANDS:
            raise ValueError(
                f"Unknown CLI tool: {tool}. "
                f"Supported: {list(self.CLI_COMMANDS.keys())}"
            )

        # Use placeholder API key since CLI tools don't need it
        config = ProviderConfig(
            api_key="cli-tool",  # Placeholder
            model=model or self._get_default_model(tool),
            base_url=base_url,
        )
        super().__init__(config)
        self.tool = tool

    def _get_default_model(self, tool: str) -> str:
        """Get default model for CLI tool."""
        defaults = {
            "claude": "claude-sonnet-4-5-20250929",
            "codex": "gpt-4",
            "gemini": "gemini-2.0-flash-exp",
            "ollama": "codellama:latest",
        }
        return defaults.get(tool, "default")

    def is_available(self) -> bool:
        """Check if CLI tool is available on PATH."""
        cmd_name = self.CLI_COMMANDS[self.tool][0]
        try:
            result = subprocess.run(
                [cmd_name, "--help"],
                capture_output=True,
                timeout=5,
            )
            return result.returncode == 0 or "not recognized" not in result.stderr.decode()
        except FileNotFoundError:
            return False
        except Exception:
            # If --help fails, try running the command directly
            try:
                result = subprocess.run(
                    [cmd_name],
                    capture_output=True,
                    timeout=5,
                    input=b"",
                )
                return True
            except Exception:
                return False

    def generate(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion using CLI tool.

        Args:
            prompt: The prompt to send to the CLI tool
            max_tokens: Not supported by most CLI tools (ignored)
            temperature: Not supported by most CLI tools (ignored)

        Returns:
            ProviderResponse with translated text
        """
        start = time.time()

        cmd = self._build_command(prompt)

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=self.config.timeout,
                check=False,
            )

            latency = int((time.time() - start) * 1000)

            if result.returncode != 0:
                error_msg = result.stderr.strip() or result.stdout.strip()
                raise RuntimeError(
                    f"{self.tool} CLI failed: {error_msg}"
                )

            # Extract response text (stdout is the output)
            text = result.stdout.strip()

            return ProviderResponse(
                text=text,
                model=self.config.model,
                usage={
                    "prompt_tokens": len(prompt),  # Approximate
                    "completion_tokens": len(text),
                    "total_tokens": len(prompt) + len(text),
                },
                latency_ms=latency,
                raw_response={"stderr": result.stderr, "returncode": result.returncode}
            )

        except subprocess.TimeoutExpired:
            raise RuntimeError(f"{self.tool} CLI timed out after {self.config.timeout}s")

    def _build_command(self, prompt: str) -> list[str]:
        """Build command list for subprocess.

        Args:
            prompt: The prompt to translate

        Returns:
            List of command arguments
        """
        base_cmd = list(self.CLI_COMMANDS[self.tool])

        if self.tool == "ollama":
            # ollama run MODEL "prompt"
            return base_cmd + [self.config.model, prompt]

        # Most tools take the prompt as the final argument
        # Some tools may need the prompt passed via stdin
        return base_cmd + [prompt]

    async def generate_async(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0
    ) -> ProviderResponse:
        """Async version using asyncio.to_thread."""
        import asyncio

        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            None,
            lambda: self.generate(prompt, max_tokens, temperature)
        )


def get_cli_provider(tool: str, model: Optional[str] = None) -> CliToolProvider:
    """Factory function to create CLI tool provider.

    Args:
        tool: CLI tool name ("claude", "codex", "gemini", "ollama")
        model: Optional model identifier

    Returns:
        Configured CliToolProvider instance

    Raises:
        ValueError: If tool is unknown

    Examples:
        >>> provider = get_cli_provider("claude")
        >>> response = provider.generate("Translate: こんにちは")

        >>> provider = get_cli_provider("ollama", model="codellama:latest")
        >>> response = provider.generate("Translate: こんにちは")
    """
    return CliToolProvider(tool=tool, model=model)
