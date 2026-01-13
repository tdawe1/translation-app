from dataclasses import dataclass
from typing import Optional

from .parser import ParsedStyleGuide, StyleSection


@dataclass
class SystemPromptConfig:
    include_examples: bool = True
    include_tone: bool = True
    include_formatting: bool = True
    max_section_length: int = 500


def build_system_prompt(
    guide: ParsedStyleGuide,
    config: Optional[SystemPromptConfig] = None,
) -> str:
    cfg = config or SystemPromptConfig()

    prompt_parts = []

    prompt_parts.append("# Gengo Japanese-to-English Style Guide\n")
    prompt_parts.append(
        "You are a professional Japanese-to-English translator "
        "following Gengo's style guide for natural, high-quality US English.\n"
    )

    for section_name, section in guide.sections.items():
        section_text = _build_section_prompt(section, cfg)
        if section_text:
            prompt_parts.append(section_text)

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
    parts = [f"## {section.name.replace('_', ' ').title()}"]

    if section.description:
        parts.append(section.description)

    if section.rules:
        parts.append("\nRules:")
        for rule in section.rules[:10]:
            if len(rule) <= cfg.max_section_length:
                parts.append(f"- {rule}")
            else:
                parts.append(f"- {rule[: cfg.max_section_length]}...")

    if cfg.include_examples and section.examples:
        parts.append("\nExamples:")
        for ex in section.examples[:5]:
            parts.append(f"{ex.get('key', '')}: {ex.get('value', '')}")

    return "\n".join(parts)
