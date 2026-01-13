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
    """A section of style guide (e.g., punctuation, spelling)."""

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

    sections = re.split(r"^##\s+(.+)$", content, flags=re.MULTILINE)

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
    description_match = re.search(r"^(.+?)(?:\n|$)", body.strip())
    description = description_match.group(1).strip() if description_match else ""

    rules = []
    bullet_pattern = r"^\*\s+(.+?)$"
    for match in re.finditer(bullet_pattern, body, re.MULTILINE):
        rule_text = match.group(1).strip()
        if rule_text and len(rule_text) > 5:
            rules.append(rule_text)

    examples = []
    example_pattern = r"\*\*Examples?:\s*\n(.+?)(?=\n\n|\Z)"
    for match in re.finditer(example_pattern, body, re.MULTILINE | re.DOTALL):
        example_text = match.group(1).strip()
        for line in example_text.split("\n"):
            if ":" in line:
                key, value = line.split(":", 1)
                examples.append({"key": key.strip(), "value": value.strip()})

    if not rules and not examples:
        return None

    section_name = header.lower().replace(" ", "_").replace("-", "_")
    if section_name == "must_follow_rules_style_guide_critical":
        section_name = "punctuation_spelling_grammar"

    return StyleSection(
        name=section_name,
        description=description,
        rules=rules,
        examples=examples,
    )
