# Gengo Style Guide Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate the Gengo Japanese-to-English style guide as a configurable system prompt for the translation worker, enabling high-quality US English translations that follow Gengo standards.

**Architecture:** Extract Gengo style guide rules from markdown file, convert to structured JSON/TOML for programmatic access, inject as system prompt for all LLM providers, and enhance existing style checker with Gengo-specific validation rules.

**Tech Stack:** Python 3.12+, TOML (configuration), regex (rule validation), pytest (testing)

---

## Overview

The Gengo style guide (`/home/thomas/kyros-wiki/Japanese-to-English Translation Quick Reference (Gengo Style Guide Alignment).md`) contains comprehensive rules for Japanese-to-English translation. This plan will:

1. **Parse the markdown style guide** into structured rules
2. **Create a system prompt template** that incorporates Gengo rules
3. **Integrate with LLM providers** to inject system prompts
4. **Enhance style checker** with Gengo-specific validation
5. **Add configuration** to enable/disable Gengo style guide
6. **Test** integration with all supported LLM providers

---

## Task 1: Create Style Guide Parser

**Files:**
- Create: `backend/cmd/translation-worker/style_guide/parser.py`
- Test: `backend/cmd/translation-worker/tests/test_style_guide/test_parser.py`

**Step 1: Write failing test**

```python
# tests/test_style_guide/test_parser.py
import pytest
from pathlib import Path

from style_guide.parser import parse_gengo_style_guide

def test_parse_punctuation_rules():
    """Should extract punctuation rules from style guide."""
    guide_path = Path(__file__).parent / "fixtures" / "gengo_style_guide.md"
    rules = parse_gengo_style_guide(guide_path)

    assert "punctuation" in rules
    assert "Use standard English punctuation" in rules["punctuation"]["description"]
    assert "Replace Japanese punctuation" in rules["punctuation"]["rules"][0]

def test_parse_spelling_rules():
    """Should extract US English spelling rules."""
    guide_path = Path(__file__).parent / "fixtures" / "gengo_style_guide.md"
    rules = parse_gengo_style_guide(guide_path)

    assert "spelling" in rules
    assert rules["spelling"]["default"] == "US English"
    assert "color" in rules["spelling"]["examples"]

def test_parse_grammar_rules():
    """Should extract grammar and syntax rules."""
    guide_path = Path(__file__).parent / "fixtures" / "gengo_style_guide.md"
    rules = parse_gengo_style_guide(guide_path)

    assert "grammar" in rules
    assert any("articles" in rule.lower() for rule in rules["grammar"]["rules"])
```

**Step 2: Run test to verify it fails**

Run: `cd backend/cmd/translation-worker && pytest tests/test_style_guide/test_parser.py::test_parse_punctuation_rules -v`
Expected: FAIL with "ModuleNotFoundError: No module named 'style_guide'"

**Step 3: Write minimal implementation**

```python
# style_guide/parser.py
"""Parser for Gengo Japanese-to-English style guide markdown file.

Extracts structured rules from markdown sections into programmatic format
for injection into LLM system prompts and style validation.
"""

import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Dict, List, Optional


@dataclass
class StyleSection:
    """A section of the style guide (e.g., punctuation, spelling)."""

    name: str
    description: str
    rules: List[str] = field(default_factory=list)
    examples: List[Dict[str, str]] = field(default_factory=list)


@dataclass
class ParsedStyleGuide:
    """Parsed style guide with all sections."""

    sections: Dict[str, StyleSection] = field(default_factory=dict)
    raw_content: str = ""


def parse_gengo_style_guide(markdown_path: Path) -> ParsedStyleGuide:
    """Parse Gengo style guide markdown into structured format.

    Args:
        markdown_path: Path to Gengo style guide markdown file

    Returns:
        ParsedStyleGuide with all extracted sections

    Raises:
        FileNotFoundError: If markdown file doesn't exist
    """
    if not markdown_path.exists():
        raise FileNotFoundError(f"Style guide not found: {markdown_path}")

    with open(markdown_path, "r", encoding="utf-8") as f:
        content = f.read()

    guide = ParsedStyleGuide(raw_content=content)

    # Extract sections using regex for markdown headers
    sections = re.split(r"^##\s+(.+)$", content, flags=re.MULTILINE)

    # First element is before first header (intro)
    for i in range(1, len(sections), 2):
        header = sections[i].strip()
        body = sections[i + 1] if i + 1 < len(sections) else ""

        if not header or not body:
            continue

        section = _parse_section(header, body)
        if section:
            guide.sections[section.name] = section

    return guide


def _parse_section(header: str, body: str) -> Optional[StyleSection]:
    """Parse a single section from markdown.

    Args:
        header: Section header text
        body: Section body content

    Returns:
        StyleSection or None if section is empty
    """
    # Extract description (first paragraph)
    description_match = re.search(r"^(.+?)(?:\n|$)", body.strip())
    description = description_match.group(1).strip() if description_match else ""

    # Extract bullet points as rules
    rules = []
    bullet_pattern = r"^\*\s+(.+?)$"
    for match in re.finditer(bullet_pattern, body, re.MULTILINE):
        rule_text = match.group(1).strip()
        if rule_text and len(rule_text) > 5:  # Skip empty or tiny bullets
            rules.append(rule_text)

    # Extract examples (lines with "**" or "*Examples:")
    examples = []
    example_pattern = r'\*\*Examples?:\s*\n(.+?)(?=\n\n|\Z)'
    for match in re.finditer(example_pattern, body, re.MULTILINE | re.DOTALL):
        example_text = match.group(1).strip()
        # Parse "key: value" examples
        for line in example_text.split("\n"):
            if ":" in line:
                key, value = line.split(":", 1)
                examples.append({"key": key.strip(), "value": value.strip()})

    # Skip sections with no rules or examples
    if not rules and not examples:
        return None

    # Determine section name from header
    section_name = header.lower().replace(" ", "_").replace("-", "_")
    if section_name == "must_follow_rules_style_guide_critical":
        section_name = "punctuation_spelling_grammar"

    return StyleSection(
        name=section_name,
        description=description,
        rules=rules,
        examples=examples,
    )
```

**Step 4: Run test to verify it passes**

Run: `cd backend/cmd/translation-worker && pytest tests/test_style_guide/test_parser.py -v`
Expected: PASS

**Step 5: Create test fixture**

Create: `backend/cmd/translation-worker/tests/test_style_guide/fixtures/gengo_style_guide.md`

```markdown
# Test Style Guide

## Punctuation

Use standard English punctuation and spacing.

* Replace Japanese punctuation with English equivalents.

## Spelling

Default to US English.

Examples: color, organize.
```

**Step 6: Run all parser tests**

Run: `cd backend/cmd/translation-worker && pytest tests/test_style_guide/test_parser.py -v`
Expected: All tests PASS

**Step 7: Commit**

```bash
cd backend/cmd/translation-worker
git add style_guide/parser.py tests/test_style_guide/test_parser.py tests/test_style_guide/fixtures/
git commit -m "feat: add Gengo style guide markdown parser"
```

---

## Task 2: Create System Prompt Generator

**Files:**
- Create: `backend/cmd/translation-worker/style_guide/prompt_builder.py`
- Test: `backend/cmd/translation-worker/tests/test_style_guide/test_prompt_builder.py`

**Step 1: Write failing test**

```python
# tests/test_style_guide/test_prompt_builder.py
import pytest

from style_guide.parser import ParsedStyleGuide
from style_guide.prompt_builder import build_system_prompt

def test_build_system_prompt_basic():
    """Should generate system prompt from parsed style guide."""
    guide = ParsedStyleGuide(
        sections={
            "punctuation": StyleSection(
                name="punctuation",
                description="Use standard English punctuation",
                rules=["Replace Japanese punctuation with English equivalents"],
                examples=[],
            ),
        }
    )

    prompt = build_system_prompt(guide)

    assert "punctuation" in prompt.lower()
    assert "japanese punctuation" in prompt.lower()
    assert "english equivalents" in prompt.lower()

def test_build_system_prompt_includes_all_sections():
    """Should include all guide sections."""
    guide = ParsedStyleGuide(
        sections={
            "punctuation": StyleSection(name="punctuation", description="Punctuation rules", rules=[]),
            "spelling": StyleSection(name="spelling", description="Spelling rules", rules=[]),
            "grammar": StyleSection(name="grammar", description="Grammar rules", rules=[]),
        }
    )

    prompt = build_system_prompt(guide)

    assert prompt.count("##") >= 3  # At least 3 section headers
```

**Step 2: Run test to verify it fails**

Run: `cd backend/cmd/translation-worker && pytest tests/test_style_guide/test_prompt_builder.py::test_build_system_prompt_basic -v`
Expected: FAIL with "ModuleNotFoundError: No module named 'style_guide.prompt_builder'"

**Step 3: Write minimal implementation**

```python
# style_guide/prompt_builder.py
"""System prompt generator for Gengo style guide.

Builds comprehensive system prompts from parsed style guide rules
for injection into LLM API calls.
"""

from dataclasses import dataclass
from typing import Optional

from .parser import ParsedStyleGuide, StyleSection


@dataclass
class SystemPromptConfig:
    """Configuration for system prompt generation."""

    include_examples: bool = True
    include_tone: bool = True
    include_formatting: bool = True
    max_section_length: int = 500  # Characters per section


def build_system_prompt(
    guide: ParsedStyleGuide,
    config: Optional[SystemPromptConfig] = None,
) -> str:
    """Build comprehensive system prompt from Gengo style guide.

    Args:
        guide: Parsed style guide with all sections
        config: Optional configuration for prompt generation

    Returns:
        Formatted system prompt string for LLM injection
    """
    cfg = config or SystemPromptConfig()

    prompt_parts = []

    # Core objective
    prompt_parts.append("# Gengo Japanese-to-English Style Guide\n")
    prompt_parts.append(
        "You are a professional Japanese-to-English translator "
        "following Gengo's style guide for natural, high-quality US English.\n"
    )

    # Add each section
    for section_name, section in guide.sections.items():
        section_text = _build_section_prompt(section, cfg)
        if section_text:
            prompt_parts.append(section_text)

    # Quality assurance checklist
    prompt_parts.append("\n## Quality Assurance Checklist\n")
    prompt_parts.append(
        "Before delivering translation, verify:\n"
        "1. Accuracy - No omissions or additions\n"
        "2. Clarity - Natural, idiomatic English (not literal)\n"
        "3. Consistency - Terms, spelling, formatting throughout\n"
        "4. US English - Default to American spelling and conventions\n"
    )

    return "\n".join(prompt_parts)


def _build_section_prompt(section: StyleSection, cfg: SystemPromptConfig) -> str:
    """Build prompt section from StyleSection.

    Args:
        section: Style section to format
        cfg: Configuration for section formatting

    Returns:
        Formatted section string or empty string
    """
    parts = [f"## {section.name.replace('_', ' ').title()}"]

    if section.description:
        parts.append(section.description)

    # Add rules (limit length)
    if section.rules:
        parts.append("\nRules:")
        for rule in section.rules[:10]:  # Limit to 10 rules per section
            if len(rule) <= cfg.max_section_length:
                parts.append(f"- {rule}")
            else:
                # Truncate long rules
                parts.append(f"- {rule[:cfg.max_section_length]}...")

    # Add examples if enabled
    if cfg.include_examples and section.examples:
        parts.append("\nExamples:")
        for ex in section.examples[:5]:  # Limit to 5 examples
            parts.append(f"{ex.get('key', '')}: {ex.get('value', '')}")

    return "\n".join(parts)
```

**Step 4: Run test to verify it passes**

Run: `cd backend/cmd/translation-worker && pytest tests/test_style_guide/test_prompt_builder.py -v`
Expected: PASS

**Step 5: Commit**

```bash
cd backend/cmd/translation-worker
git add style_guide/prompt_builder.py tests/test_style_guide/test_prompt_builder.py
git commit -m "feat: add Gengo style guide system prompt builder"
```

---

## Task 3: Integrate with LLM Providers

**Files:**
- Modify: `backend/cmd/translation-worker/review/llm/base.py`
- Modify: `backend/cmd/translation-worker/review/llm/providers.py`
- Test: `backend/cmd/translation-worker/tests/test_review/test_llm_providers.py`

**Step 1: Write failing test**

```python
# tests/test_review/test_llm_providers.py
from pathlib import Path
from unittest.mock import Mock, patch

from review.llm.providers import AnthropicProvider
from style_guide.parser import parse_gengo_style_guide
from style_guide.prompt_builder import build_system_prompt


def test_anthropic_provider_injects_style_guide():
    """Should inject Gengo style guide into system prompt."""
    # Load style guide
    style_guide_path = Path(__file__).parent.parent / "style_guide" / "fixtures" / "gengo_style_guide.md"
    if style_guide_path.exists():
        guide = parse_gengo_style_guide(style_guide_path)
        system_prompt = build_system_prompt(guide)
    else:
        system_prompt = "Test system prompt"

    # Mock API call
    with patch("anthropic.Anthropic") as mock_anthropic:
        mock_client = Mock()
        mock_anthropic.return_value = mock_client

        provider = AnthropicProvider(
            api_key="test-key",
            model="claude-3-5-sonnet-20241022",
            style_guide_system_prompt=system_prompt,  # This will be added
        )

        # Call generate
        response = provider.generate("Translate: こんにちは")

        # Verify system prompt was used
        mock_client.messages.create.assert_called_once()
        call_args = mock_client.messages.create.call_args
        messages = call_args.kwargs.get("messages", [])

        # First message should be system with style guide
        assert len(messages) > 0
        assert messages[0]["role"] == "system"
        assert "Gengo" in messages[0]["content"] or "style guide" in messages[0]["content"]
```

**Step 2: Run test to verify it fails**

Run: `cd backend/cmd/translation-worker && pytest tests/test_review/test_llm_providers.py::test_anthropic_provider_injects_style_guide -v`
Expected: FAIL with "TypeError: AnthropicProvider.__init__() got an unexpected keyword argument 'style_guide_system_prompt'"

**Step 3: Modify BaseProvider to support system prompts**

```python
# review/llm/base.py - Modify __init__ and generate signature
"""Base provider interface and data structures."""
from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Optional, List


@dataclass
class ProviderConfig:
    """Configuration for LLM provider."""

    api_key: str
    base_url: Optional[str] = None
    model: str = "claude-sonnet-4-5-20250929"
    max_tokens: int = 8192
    timeout: int = 120
    system_prompt: Optional[str] = None  # NEW: System prompt injection


@dataclass
class ProviderResponse:
    """Response from LLM provider."""

    text: str
    model: str
    usage: dict
    latency_ms: int
    raw_response: Optional[dict] = None


class BaseProvider(ABC):
    """Abstract base for LLM providers."""

    def __init__(self, config: ProviderConfig):
        self.config = config

    @abstractmethod
    def is_available(self) -> bool:
        """Check if provider is properly configured."""
        pass

    @abstractmethod
    def generate(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0,
    ) -> ProviderResponse:
        """Generate completion from prompt."""
        pass

    @abstractmethod
    async def generate_async(
        self,
        prompt: str,
        max_tokens: Optional[int] = None,
        temperature: float = 0.0,
    ) -> ProviderResponse:
        """Async version of generate."""
        pass

    def _build_messages(self, prompt: str) -> List[dict]:
        """Build message list with optional system prompt.

        Args:
            prompt: User prompt text

        Returns:
            List of message dicts with system prompt if configured
        """
        messages = []

        # Add system prompt if configured
        if self.config.system_prompt:
            messages.append({"role": "system", "content": self.config.system_prompt})

        # Add user prompt
        messages.append({"role": "user", "content": prompt})

        return messages
```

**Step 4: Modify AnthropicProvider to use system prompt**

```python
# review/llm/providers.py - Modify AnthropicProvider.generate() method
# Find the generate() method (around line 95) and modify:

    @retry(...)
    def generate(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion using Anthropic Messages API (2026)."""
        start = time.time()
        client = self._get_client()

        max_tokens = max_tokens or self.config.max_tokens

        # USE _build_messages to get system prompt
        messages = self._build_messages(prompt)

        response = client.messages.create(
            model=self.config.model,
            max_tokens=max_tokens,
            temperature=temperature,
            messages=messages,  # Use messages instead of single user message
        )

        latency = int((time.time() - start) * 1000)

        return ProviderResponse(
            text=response.content[0].text,
            model=response.model,
            usage={
                "prompt_tokens": response.usage.input_tokens,
                "completion_tokens": response.usage.output_tokens,
                "total_tokens": response.usage.input_tokens
                + response.usage.output_tokens,
            },
            latency_ms=latency,
        )
```

**Step 5: Modify OpenAIProvider to use system prompt**

```python
# review/llm/providers.py - Modify OpenAIProvider.generate() method
# Find the generate() method (around line 179) and modify:

    @retry(...)
    def generate(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion using OpenAI Chat Completions API (2026)."""
        start = time.time()
        client = self._get_client()

        max_tokens = max_tokens or self.config.max_tokens

        # USE _build_messages to get system prompt
        messages = self._build_messages(prompt)

        response = client.chat.completions.create(
            model=self.config.model,
            max_completion_tokens=max_tokens,
            temperature=temperature,
            messages=messages,  # Use messages instead of single user message
        )

        latency = int((time.time() - start) * 1000)

        return ProviderResponse(
            text=response.choices[0].message.content,
            model=response.model,
            usage={
                "prompt_tokens": response.usage.prompt_tokens,
                "completion_tokens": response.usage.completion_tokens,
                "total_tokens": response.usage.total_tokens,
            },
            latency_ms=latency,
        )
```

**Step 6: Modify GeminiProvider to use system prompt**

```python
# review/llm/providers.py - Modify GeminiProvider.generate() method
# Find the generate() method (around line 275) and modify:

    @retry(...)
    def generate(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        """Generate completion using Vertex AI Gemini API (2026)."""
        import requests

        start = time.time()

        max_tokens = max_tokens or self.config.max_tokens

        # Build request with system prompt
        messages = self._build_messages(prompt)

        # Gemini uses system instruction in generationConfig
        system_content = None
        user_content = prompt
        if len(messages) > 0 and messages[0]["role"] == "system":
            system_content = messages[0]["content"]
            if len(messages) > 1:
                user_content = messages[1]["content"]

        # Gemini request format (2026)
        request_body = {
            "contents": [
                {
                    "role": "user",
                    "parts": [{"text": user_content}],
                }
            ],
            "generationConfig": {
                "temperature": temperature,
                "maxOutputTokens": max_tokens,
            },
        }

        # Add system instruction if provided
        if system_content:
            request_body["generationConfig"]["systemInstruction"] = system_content

        endpoint = self._get_endpoint()
        headers = {
            "Authorization": f"Bearer {self.config.api_key}",
            "Content-Type": "application/json",
        }

        response = requests.post(
            f"{endpoint}:generateContent",
            json=request_body,
            headers=headers,
            timeout=self.config.timeout,
        )
        response.raise_for_status()

        data = response.json()
        latency = int((time.time() - start) * 1000)

        # Parse Gemini response format (2026)
        candidate = data["candidates"][0]
        text = candidate["content"]["parts"][0]["text"]
        usage_metadata = data.get("usageMetadata", {})

        return ProviderResponse(
            text=text,
            model=self.config.model,
            usage={
                "prompt_tokens": usage_metadata.get("promptTokenCount", 0),
                "completion_tokens": usage_metadata.get("candidatesTokenCount", 0),
                "total_tokens": usage_metadata.get("totalTokenCount", 0),
            },
            latency_ms=latency,
        )
```

**Step 7: Update get_provider to support system prompt**

```python
# review/llm/providers.py - Modify get_provider() function signature
# Find get_provider() (around line 337) and modify:

def get_provider(
    provider_name: str,
    api_key: str,
    model: Optional[str] = None,
    system_prompt: Optional[str] = None,  # NEW parameter
    **kwargs
) -> BaseProvider:
    """Factory function to get provider instance.

    Args:
        provider_name: "anthropic", "openai", or "gemini"
        api_key: API key for provider
        model: Optional model override
        system_prompt: Optional system prompt to inject
        **kwargs: Additional provider-specific args (project_id, location for Gemini)

    Returns:
        Configured provider instance

    Raises:
        ValueError: If provider_name is unknown
    """
    models = {
        "anthropic": (AnthropicProvider.DEFAULT_MODEL, AnthropicProvider),
        "openai": (OpenAIProvider.DEFAULT_MODEL, OpenAIProvider),
        "gemini": (GeminiProvider.DEFAULT_MODEL, GeminiProvider),
    }

    if provider_name not in models:
        raise ValueError(
            f"Unknown provider: {provider_name}. Use: {list(models.keys())}"
        )

    default_model, provider_class = models[provider_name]
    model = model or default_model

    # Create config with system prompt
    config = ProviderConfig(api_key=api_key, model=model, system_prompt=system_prompt)

    if provider_name == "gemini":
        return provider_class(api_key=api_key, model=model, **kwargs, config=config)

    return provider_class(api_key=api_key, model=model, config=config)
```

**Step 8: Update provider constructors to accept config**

```python
# review/llm/providers.py - Update all provider __init__ methods
# For AnthropicProvider (line 62):

    def __init__(self, api_key: str, model: str = None, base_url: str = None, config: ProviderConfig = None):
        model = model or self.DEFAULT_MODEL
        config = config or ProviderConfig(api_key=api_key, model=model, base_url=base_url)
        super().__init__(config)  # Pass config to parent
        self._client = None

# For OpenAIProvider (line 152):

    def __init__(self, api_key: str, model: str = None, config: ProviderConfig = None):
        model = model or self.DEFAULT_MODEL
        config = config or ProviderConfig(api_key=api_key, model=model)
        super().__init__(config)  # Pass config to parent
        self._client = None

# For GeminiProvider (line 235):

    def __init__(
        self,
        api_key: str,
        model: str = None,
        project_id: str = None,
        location: str = None,
        config: ProviderConfig = None,
    ):
        model = model or self.DEFAULT_MODEL
        config = config or ProviderConfig(api_key=api_key, model=model)
        super().__init__(config)  # Pass config to parent
        self.project_id = project_id or os.environ.get("GEMINI_PROJECT_ID", "")
        self.location = location or os.environ.get(
            "GEMINI_LOCATION", self.DEFAULT_LOCATION
        )
        self._client = None
```

**Step 9: Run test to verify it passes**

Run: `cd backend/cmd/translation-worker && pytest tests/test_review/test_llm_providers.py::test_anthropic_provider_injects_style_guide -v`
Expected: PASS

**Step 10: Commit**

```bash
cd backend/cmd/translation-worker
git add review/llm/base.py review/llm/providers.py tests/test_review/test_llm_providers.py
git commit -m "feat: add system prompt injection for Gengo style guide"
```

---

## Task 4: Add Configuration Support

**Files:**
- Modify: `backend/cmd/translation-worker/main.py`
- Create: `backend/cmd/translation-worker/config.example.toml`
- Test: `backend/cmd/translation-worker/tests/test_main.py`

**Step 1: Write failing test**

```python
# tests/test_main.py
import os
from pathlib import Path
import tempfile

from main import load_config, validate_config


def test_config_with_style_guide_enabled():
    """Should validate config with style guide enabled."""
    with tempfile.NamedTemporaryFile(mode="w", suffix=".toml", delete=False) as f:
        f.write("""
[worker]
id = "test-worker"
max_concurrent = 1

[translation]
default_provider = "anthropic"
default_model = "claude-sonnet-4-5-20250929"

[style_guide]
enabled = true
path = "/path/to/gengo_style_guide.md"
""")
        f.flush()

        errors = validate_config(load_config(f.name))
        assert len(errors) == 0

def test_config_with_style_guide_disabled():
    """Should validate config with style guide disabled."""
    with tempfile.NamedTemporaryFile(mode="w", suffix=".toml", delete=False) as f:
        f.write("""
[worker]
id = "test-worker"
max_concurrent = 1

[translation]
default_provider = "anthropic"
default_model = "claude-sonnet-4-5-20250929"

[style_guide]
enabled = false
""")
        f.flush()

        errors = validate_config(load_config(f.name))
        assert len(errors) == 0
```

**Step 2: Run test to verify it fails**

Run: `cd backend/cmd/translation-worker && pytest tests/test_main.py::test_config_with_style_guide_enabled -v`
Expected: FAIL (test may not exist yet)

**Step 3: Modify validate_config to accept style_guide section**

```python
# main.py - Modify validate_config() function
# Add to required_sections check (around line 60):

    required_sections = ["worker", "translation"]
    # style_guide is optional

# Add validation for style_guide section (around line 88, after job_queue validation):

    # Validate style_guide section if enabled
    if "style_guide" in cfg and cfg["style_guide"].get("enabled", False):
        if "path" not in cfg["style_guide"]:
            errors.append("style_guide.path required when style_guide.enabled=true")
        else:
            guide_path = Path(cfg["style_guide"]["path"])
            if not guide_path.exists():
                errors.append(f"style_guide.path file not found: {guide_path}")
```

**Step 4: Create example config with style guide**

Create: `backend/cmd/translation-worker/config.example.toml`

```toml
# Translation Worker Configuration Example
# Copy this file to config.toml and customize for your setup

[worker]
id = "worker-1"
max_concurrent = 3
heartbeat_interval = "30s"

[translation]
default_provider = "anthropic"
default_model = "claude-sonnet-4-5-20250929"

[style_guide]
# Enable Gengo style guide injection
enabled = true
# Path to Gengo style guide markdown file
path = "/home/thomas/kyros-wiki/Japanese-to-English Translation Quick Reference (Gengo Style Guide Alignment).md"

[cache.redis]
host = "localhost"
port = 6379
db = 0
# password = "your-password"  # Optional

[job_queue]
enabled = true
backend = "redis"
max_concurrent = 3
poll_interval = "1s"
```

**Step 5: Modify main() to load style guide**

```python
# main.py - Modify main() function
# Add imports at top:
from pathlib import Path
from style_guide.parser import parse_gengo_style_guide
from style_guide.prompt_builder import build_system_prompt
from review.llm.providers import get_provider

# In main() after loading config (around line 242):

        # Load configuration
        config = load_config()

        # Validate configuration
        validation_errors = validate_config(config)
        if validation_errors:
            print("Configuration errors:", file=sys.stderr)
            for error in validation_errors:
                print(f"  - {error}", file=sys.stderr)
            sys.exit(1)

        # Load Gengo style guide if enabled
        system_prompt = None
        style_guide_enabled = config.get("style_guide", {}).get("enabled", False)
        if style_guide_enabled:
            style_guide_path = Path(config["style_guide"]["path"])
            if style_guide_path.exists():
                try:
                    guide = parse_gengo_style_guide(style_guide_path)
                    system_prompt = build_system_prompt(guide)
                    print(f"Loaded Gengo style guide: {style_guide_path}")
                    print(f"  Sections: {len(guide.sections)}")
                except Exception as e:
                    print(f"Warning: Failed to load style guide: {e}", file=sys.stderr)
            else:
                print(f"Warning: Style guide not found: {style_guide_path}", file=sys.stderr)

        # Display worker info
        worker_id = config.get("worker", {}).get("id", "unspecified")
        provider = config.get("translation", {}).get("default_provider")
        model = config.get("translation", {}).get("default_model")

        print(f"Translation Worker v1.0.0 starting...")
        print(f"  Worker ID: {worker_id}")
        print(f"  Translation Backend: {provider}/{model}")
        print(f"  Style Guide: {'enabled' if style_guide_enabled else 'disabled'}")
        print(f"  Mode: hybrid (folder watch + Redis job queue)")

        # Pass system_prompt to providers when creating them
        # (This would be done in job processing, not here)
```

**Step 6: Run test to verify it passes**

Run: `cd backend/cmd/translation-worker && pytest tests/test_main.py::test_config_with_style_guide_enabled -v`
Expected: PASS

**Step 7: Commit**

```bash
cd backend/cmd/translation-worker
git add main.py config.example.toml tests/test_main.py
git commit -m "feat: add style guide configuration support"
```

---

## Task 5: Enhance Style Checker with Gengo Rules

**Files:**
- Modify: `backend/cmd/translation-worker/audit/style_checker.py`
- Test: `backend/cmd/translation-worker/tests/test_audit/test_style_checker.py`

**Step 1: Write failing test**

```python
# tests/test_audit/test_style_checker.py
from audit.style_checker import StyleChecker, StyleIssue


def test_gengo_oxford_comma_rule():
    """Should flag missing Oxford comma."""
    checker = StyleChecker()

    translation = "I like apples bananas and oranges."
    issues = checker.check(translation)

    # Should flag for potential missing Oxford comma in lists
    oxford_issues = [i for i in issues if "oxford" in i.message.lower()]
    assert len(oxford_issues) > 0

def test_gengo_us_spelling_rule():
    """Should prefer US English spelling."""
    checker = StyleChecker()

    translation = "The colour of the sky is blue."
    issues = checker.check(translation)

    # Should flag UK spelling
    spelling_issues = [i for i in issues if i.category == "spelling"]
    assert len(spelling_issues) > 0
    assert "color" in spelling_issues[0].suggestion or "colour" in spelling_issues[0].message

def test_gengo_currency_format():
    """Should validate currency formatting."""
    checker = StyleChecker()

    translation = "The cost is 1000 yen."
    issues = checker.check(translation)

    # Should recommend ¥ symbol
    currency_issues = [i for i in issues if "currency" in i.category]
    assert len(currency_issues) > 0
```

**Step 2: Run test to verify it fails**

Run: `cd backend/cmd/translation-worker && pytest tests/test_audit/test_style_checker.py::test_gengo_oxford_comma_rule -v`
Expected: FAIL (no Oxford comma rule exists)

**Step 3: Add Gengo-specific rules to StyleChecker**

```python
# audit/style_checker.py - Modify DEFAULT_HONORIFICS_PATTERNS and add new patterns
# Add to class attributes (around line 100):

    # Gengo-specific patterns
    OXFORD_COMMA_PATTERNS = [
        (r"\b(\w+)\s+(,\s*\w+\s+and\s+\w+)", "Consider using Oxford comma in lists"),
    ]

    # US vs UK spelling patterns
    SPELLING_PATTERNS = [
        (r"\bcolour\b", "Prefer US spelling: 'color'"),
        (r"\bcentre\b", "Prefer US spelling: 'center'"),
        (r"\borganise\b", "Prefer US spelling: 'organize'"),
        (r"\bprogramme\b", "Prefer US spelling: 'program'"),
    ]

    # Currency formatting
    CURRENCY_PATTERNS = [
        (r"\b\d+\s+(?:yen|dollars|pounds)\b", "Use currency symbol (¥, $, £)"),
        (r"\bus\s+\$\d+", "Use US$ format for currency"),
    ]

    # Number formatting
    NUMBER_PATTERNS = [
        (r"\b(?:[a-z]+\s+){1,2}\d+\b", "Use numerals for 10+"),
    ]

# Modify _load_style_guide() to load Gengo rules (around line 326):
    def _load_style_guide(self, path: str) -> None:
        """Load custom style guide from file.

        Expected format (TOML or simple key=value):
        [forbidden_terms]
        terms = ["term1", "term2"]

        [preferred_terms]
        "source_term" = "preferred_translation"

        [gengo_rules]
        oxford_comma = true
        us_spelling = true
        currency_format = true
        """
        try:
            guide_path = Path(path)
            if not guide_path.exists():
                return

            # Try to load as TOML
            try:
                import tomli
                with open(guide_path, "rb") as f:
                    self.custom_rules = tomli.load(f)

                # Extract forbidden terms
                if "forbidden_terms" in self.custom_rules:
                    self.forbidden_terms = set(
                        self.custom_rules["forbidden_terms"].get("terms", [])
                    )

                # Extract preferred terms
                if "preferred_terms" in self.custom_rules:
                    self.preferred_terms = self.custom_rules["preferred_terms"]

                # Extract Gengo-specific rule flags
                self._gengo_oxford_comma = self.custom_rules.get("gengo_rules", {}).get("oxford_comma", False)
                self._gengo_us_spelling = self.custom_rules.get("gengo_rules", {}).get("us_spelling", False)
                self._gengo_currency_format = self.custom_rules.get("gengo_rules", {}).get("currency_format", False)

            except ImportError:
                # tomli not installed, fall back to simple parsing
                self._parse_simple_style_guide(guide_path)
            except Exception:
                # TOML parsing failed (invalid TOML), try simple format
                self._parse_simple_style_guide(guide_path)

        except Exception:
            # If loading fails, continue with defaults
            pass
```

**Step 4: Add Gengo check methods**

```python
# audit/style_checker.py - Add new check methods
# Add after _check_terminology_consistency() (around line 324):

    def _check_oxford_comma(self, text: str) -> List[StyleIssue]:
        """Check for Oxford comma usage in lists."""
        issues = []

        if not getattr(self, '_gengo_oxford_comma', False):
            return issues

        for pattern, message in self.OXFORD_COMMA_PATTERNS:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                issues.append(StyleIssue(
                    severity="warning",
                    category="punctuation",
                    message=message,
                    location=f"position {match.start()}",
                    suggestion="Add comma before 'and' in lists of 3+ items",
                ))

        return issues

    def _check_us_spelling(self, text: str) -> List[StyleIssue]:
        """Check for UK vs US spelling."""
        issues = []

        if not getattr(self, '_gengo_us_spelling', False):
            return issues

        for pattern, suggestion in self.SPELLING_PATTERNS:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                issues.append(StyleIssue(
                    severity="warning",
                    category="spelling",
                    message=f"UK spelling detected: {match.group()}",
                    location=f"position {match.start()}",
                    suggestion=suggestion,
                ))

        return issues

    def _check_currency_format(self, text: str) -> List[StyleIssue]:
        """Check currency formatting."""
        issues = []

        if not getattr(self, '_gengo_currency_format', False):
            return issues

        for pattern, message in self.CURRENCY_PATTERNS:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                issues.append(StyleIssue(
                    severity="info",
                    category="formatting",
                    message=message,
                    location=f"position {match.start()}",
                ))

        return issues

# Modify check() method to call new checks (around line 176):
    def check(
        self,
        translation: str,
        source: Optional[str] = None,
    ) -> List[StyleIssue]:
        """Check translation against style guide.

        Args:
            translation: The translated text to check
            source: Optional source Japanese text for consistency checks

        Returns:
            List of StyleIssue objects found
        """
        issues = []

        # Check honorifics
        if self.honorifics_enabled:
            issues.extend(self._check_honorifics(translation))

        # Check sentence length
        if self.sentence_check_enabled:
            issues.extend(self._check_sentence_length(translation))

        # Check passive voice
        if self.passive_check_enabled:
            issues.extend(self._check_passive_voice(translation))

        # Check articles
        issues.extend(self._check_articles(translation))

        # Check Gengo-specific rules
        issues.extend(self._check_oxford_comma(translation))
        issues.extend(self._check_us_spelling(translation))
        issues.extend(self._check_currency_format(translation))

        # Check custom rules
        if self.custom_rules:
            issues.extend(self._check_custom_rules(translation))

        # Check terminology consistency if source provided
        if source and self.preferred_terms:
            issues.extend(self._check_terminology_consistency(translation, source))

        return issues
```

**Step 5: Initialize Gengo flags in __init__**

```python
# audit/style_checker.py - Modify __init__ method (around line 103):
    def __init__(
        self,
        style_guide_path: Optional[str] = None,
        max_sentence_length: int = DEFAULT_MAX_SENTENCE_LENGTH,
        max_passive_ratio: float = DEFAULT_MAX_PASSIVE_RATIO,
        honorifics_enabled: bool = True,
        passive_check_enabled: bool = True,
        sentence_check_enabled: bool = True,
    ):
        """Initialize the style checker."""
        self.style_guide_path = style_guide_path
        self.max_sentence_length = max_sentence_length
        self.max_passive_ratio = max_passive_ratio
        self.honorifics_enabled = honorifics_enabled
        self.passive_check_enabled = passive_check_enabled
        self.sentence_check_enabled = sentence_check_enabled

        # Initialize Gengo rule flags
        self._gengo_oxford_comma = False
        self._gengo_us_spelling = False
        self._gengo_currency_format = False

        # Load custom rules if style guide provided
        self.custom_rules: Dict = {}
        self.forbidden_terms: Set[str] = set()
        self.preferred_terms: Dict[str, str] = {}

        if style_guide_path:
            self._load_style_guide(style_guide_path)
```

**Step 6: Run tests to verify they pass**

Run: `cd backend/cmd/translation-worker && pytest tests/test_audit/test_style_checker.py -v`
Expected: All tests PASS

**Step 7: Commit**

```bash
cd backend/cmd/translation-worker
git add audit/style_checker.py tests/test_audit/test_style_checker.py
git commit -m "feat: add Gengo-specific style checking rules"
```

---

## Task 6: Create Integration Test

**Files:**
- Create: `backend/cmd/translation-worker/tests/test_integration/test_gengo_integration.py`
- Create: `backend/cmd/translation-worker/tests/test_integration/fixtures/sample_ja.txt`

**Step 1: Write failing integration test**

```python
# tests/test_integration/test_gengo_integration.py
import pytest
from pathlib import Path

from review.llm.providers import AnthropicProvider
from style_guide.parser import parse_gengo_style_guide
from style_guide.prompt_builder import build_system_prompt


@pytest.mark.integration
def test_end_to_end_translation_with_gengo_style():
    """Test complete translation flow with Gengo style guide injected."""
    # Load style guide
    style_guide_path = Path(__file__).parent.parent.parent / "style_guide" / "fixtures" / "gengo_style_guide.md"
    if not style_guide_path.exists():
        pytest.skip("Gengo style guide fixture not found")

    guide = parse_gengo_style_guide(style_guide_path)
    system_prompt = build_system_prompt(guide)

    # Create provider with system prompt
    provider = AnthropicProvider(
        api_key="test-key",
        model="claude-3-5-sonnet-20241022",
        config=ProviderConfig(api_key="test-key", model="claude-3-5-sonnet-20241022", system_prompt=system_prompt),
    )

    # Translate sample text
    sample_text = "こんにちは、お元気ですか。"
    response = provider.generate(sample_text)

    # Verify response
    assert response.text
    assert len(response.text) > 0
    assert "Hello" in response.text or "Hi" in response.text  # Should be in English

    # Verify system prompt was used (would need to mock client in real test)
    # This is a placeholder for actual integration testing

@pytest.mark.integration
def test_all_providers_support_system_prompt():
    """Verify all providers can accept system prompt."""
    from review.llm.providers import OpenAIProvider, GeminiProvider

    system_prompt = "Test system prompt"

    # Anthropic
    anthropic = AnthropicProvider(
        api_key="test-key",
        config=ProviderConfig(api_key="test-key", system_prompt=system_prompt),
    )
    assert anthropic.config.system_prompt == system_prompt

    # OpenAI
    openai = OpenAIProvider(
        api_key="test-key",
        config=ProviderConfig(api_key="test-key", system_prompt=system_prompt),
    )
    assert openai.config.system_prompt == system_prompt

    # Gemini
    gemini = GeminiProvider(
        api_key="test-key",
        project_id="test-project",
        config=ProviderConfig(api_key="test-key", system_prompt=system_prompt),
    )
    assert gemini.config.system_prompt == system_prompt
```

**Step 2: Run test to verify it fails**

Run: `cd backend/cmd/translation-worker && pytest tests/test_integration/test_gengo_integration.py::test_end_to_end_translation_with_gengo_style -v`
Expected: FAIL (integration test needs mocking or fixture)

**Step 3: Create sample Japanese text fixture**

Create: `backend/cmd/translation-worker/tests/test_integration/fixtures/sample_ja.txt`

```text
こんにちは、お元気ですか。
本日は日本銀行の会議に参加します。
```

**Step 4: Run integration tests**

Run: `cd backend/cmd/translation-worker && pytest tests/test_integration/test_gengo_integration.py -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
cd backend/cmd/translation-worker
git add tests/test_integration/ tests/test_integration/fixtures/
git commit -m "test: add Gengo style guide integration tests"
```

---

## Task 7: Update Documentation

**Files:**
- Modify: `backend/cmd/translation-worker/README.md`
- Create: `backend/cmd/translation-worker/docs/GENGO_STYLE_GUIDE.md`

**Step 1: Create dedicated style guide documentation**

Create: `backend/cmd/translation-worker/docs/GENGO_STYLE_GUIDE.md`

```markdown
# Gengo Style Guide Integration

The translation worker supports Gengo's Japanese-to-English style guide for producing high-quality, consistent translations.

## Enabling the Style Guide

Add the `[style_guide]` section to your `config.toml`:

```toml
[style_guide]
enabled = true
path = "/path/to/Japanese-to-English Translation Quick Reference (Gengo Style Guide Alignment).md"
```

## What Gets Enforced

When the style guide is enabled:

1. **System Prompt Injection**: All LLM providers receive Gengo rules as system instructions
2. **US English Spelling**: Enforced by style checker (color vs colour, etc.)
3. **Punctuation Rules**: Oxford commas, proper quotes, no Japanese punctuation
4. **Number Formatting**: Numerals for 10+, spelled out zero-nine
5. **Currency Format**: ¥, US$ symbols preferred
6. **Honorific Removal**: No -san, -sama in English translations
7. **Tone Matching**: Business, marketing, informal tones preserved
8. **Active Voice Preference**: Passive voice flagged if >30%

## Configuration Options

The style guide can be customized via a TOML configuration file:

```toml
[gengo_rules]
oxford_comma = true          # Enforce Oxford comma in lists
us_spelling = true           # Enforce US English spelling
currency_format = true        # Enforce currency symbol format
```

## Provider Support

All LLM providers support system prompt injection:
- ✅ Anthropic Claude
- ✅ OpenAI GPT
- ✅ Google Gemini

## Example Translation

**Without Style Guide:**
```
Source: こんにちは、お元気ですか。
Output: Hello, Tanaka-san. Are you healthy today?
```

**With Gengo Style Guide:**
```
Source: こんにちは、お元気ですか。
Output: Hello, Mr. Tanaka. How are you today?
```

The style guide ensures:
- Proper English titles (Mr./Ms.) instead of honorifics
- Natural phrasing (not literal "healthy today")
- US English conventions

## Testing

Run tests to verify style guide integration:

```bash
pytest tests/test_style_guide/
pytest tests/test_integration/test_gengo_integration.py
```
```

**Step 2: Update main README with style guide section**

Modify: `backend/cmd/translation-worker/README.md`
Add section after "Architecture":

```markdown
## Gengo Style Guide

The worker can inject Gengo's Japanese-to-English style guide rules into LLM system prompts for consistent, high-quality translations.

See [docs/GENGO_STYLE_GUIDE.md](docs/GENGO_STYLE_GUIDE.md) for details.
```

**Step 3: Commit**

```bash
cd backend/cmd/translation-worker
git add README.md docs/GENGO_STYLE_GUIDE.md
git commit -m "docs: add Gengo style guide integration documentation"
```

---

## Task 8: Final Verification

**Files:**
- Run: All tests in test suite
- Run: Integration tests
- Run: LSP diagnostics

**Step 1: Run all tests**

```bash
cd backend/cmd/translation-worker

# Run all tests
pytest tests/ -v

# Run with coverage
pytest --cov=. --cov-report=html
```

Expected: All tests PASS

**Step 2: Run LSP diagnostics**

```bash
cd backend/cmd/translation-worker

# Run Python type checking (if mypy is installed)
mypy style_guide/ review/llm/ audit/

# Or use pyright
pyright style_guide/ review/llm/ audit/
```

Expected: No type errors

**Step 3: Verify config example works**

```bash
cd backend/cmd/translation-worker

# Test loading example config
python -c "from main import load_config, validate_config; cfg = load_config('config.example.toml'); errors = validate_config(cfg); print(f'Errors: {errors}')"
```

Expected: No configuration errors

**Step 4: Final commit**

```bash
cd backend/cmd/translation-worker
git add .
git commit -m "feat: complete Gengo style guide integration"
```

---

## Summary

This plan implements:

✅ **Markdown Parser**: Extracts Gengo rules from markdown file
✅ **System Prompt Builder**: Formats rules for LLM injection
✅ **Provider Integration**: All LLM providers support system prompts
✅ **Configuration**: TOML-based enable/disable with path to style guide
✅ **Enhanced Style Checker**: Gengo-specific validation rules
✅ **Tests**: Unit, integration, and end-to-end coverage
✅ **Documentation**: Complete setup and usage guide

**Estimated Time**: 4-6 hours (8 tasks × 30-45 minutes each)

**Files Modified/Created**: 15 files
- 6 new modules (parser, prompt_builder)
- 3 modified modules (providers, base, style_checker, main)
- 4 test files
- 2 documentation files

---

**For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.
