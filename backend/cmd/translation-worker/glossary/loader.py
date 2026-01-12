# glossary/loader.py
"""
Glossary loading from JSON files and dictionaries.

Provides dataclasses for glossary entries and loading functions.
"""

from dataclasses import dataclass, field
from typing import List, Dict, Any, Optional
import json
import logging

logger = logging.getLogger(__name__)


@dataclass
class GlossaryEntry:
    """A single glossary entry.

    Attributes:
        source: Source term (typically Japanese)
        target: Target translation
        part_of_speech: POS tag for POS-aware matching
        context: Usage context or domain
        variants: Alternative forms of the source term
        forbidden_translations: Translations to avoid
        notes: Additional notes or instructions
    """
    source: str
    target: str
    part_of_speech: str = ""
    context: str = ""
    variants: List[str] = field(default_factory=list)
    forbidden_translations: List[str] = field(default_factory=list)
    notes: str = ""

    def matches_variant(self, text: str) -> bool:
        """Check if text matches this entry or any variant."""
        if text == self.source:
            return True
        return text in self.variants

    def is_forbidden(self, translation: str) -> bool:
        """Check if translation is in forbidden list."""
        return translation in self.forbidden_translations


@dataclass
class CompoundTerm:
    """A compound term (multi-word expression).

    Attributes:
        source: Multi-word source term
        target: Target translation
        part_of_speech: POS tag (usually "compound")
        context: Usage context
    """
    source: str
    target: str
    part_of_speech: str = "compound"
    context: str = ""


@dataclass
class Glossary:
    """Loaded glossary with entries and metadata.

    Attributes:
        entries: List of glossary entries
        compound_terms: List of compound terms
        name: Glossary name/identifier
        version: Glossary version
        source_language: Source language code
        target_language: Target language code
        metadata: Additional metadata
    """
    entries: List[GlossaryEntry] = field(default_factory=list)
    compound_terms: List[CompoundTerm] = field(default_factory=list)
    name: str = "default"
    version: str = "1.0"
    source_language: str = "ja"
    target_language: str = "en"
    metadata: Dict[str, Any] = field(default_factory=dict)

    def __post_init__(self):
        """Initialize defaults for list fields."""
        # Ensure defaults are applied if empty lists were passed
        if self.entries is None:
            self.entries = []
        if self.compound_terms is None:
            self.compound_terms = []

    def get_entry(self, source: str) -> Optional[GlossaryEntry]:
        """Find entry by source term."""
        for entry in self.entries:
            if entry.source == source:
                return entry
        return None

    def find_entries_by_pos(self, pos: str) -> List[GlossaryEntry]:
        """Find all entries with a specific POS tag."""
        return [e for e in self.entries if pos in e.part_of_speech]

    def get_all_sources(self) -> List[str]:
        """Get all source terms including variants."""
        sources = [e.source for e in self.entries]
        for entry in self.entries:
            sources.extend(entry.variants)
        return sources


def load_glossary_from_dict(data: Dict[str, Any]) -> Glossary:
    """Load glossary from dictionary (parsed JSON).

    Args:
        data: Dictionary with glossary data

    Returns:
        Glossary object

    Expected format:
    {
        "name": "my-glossary",
        "version": "1.0",
        "source_language": "ja",
        "target_language": "en",
        "entries": [
            {
                "source": "顧客",
                "target": "customer",
                "part_of_speech": "noun",
                "variants": ["お客様", "お客さま"],
                "forbidden_translations": ["client"],
                "context": "business",
                "notes": "Use 'customer' not 'client'"
            }
        ],
        "compound_terms": [
            {
                "source": "顧客満足度",
                "target": "customer satisfaction",
                "context": "business"
            }
        ]
    }
    """
    entries = [
        GlossaryEntry(
            source=e["source"],
            target=e["target"],
            part_of_speech=e.get("part_of_speech", ""),
            context=e.get("context", ""),
            variants=e.get("variants", []),
            forbidden_translations=e.get("forbidden_translations", []),
            notes=e.get("notes", "")
        )
        for e in data.get("entries", [])
    ]

    compound_terms = [
        CompoundTerm(
            source=c["source"],
            target=c["target"],
            part_of_speech=c.get("part_of_speech", "compound"),
            context=c.get("context", "")
        )
        for c in data.get("compound_terms", [])
    ]

    return Glossary(
        entries=entries,
        compound_terms=compound_terms,
        name=data.get("name", "default"),
        version=data.get("version", "1.0"),
        source_language=data.get("source_language", "ja"),
        target_language=data.get("target_language", "en"),
        metadata=data.get("metadata", {})
    )


def load_glossary_from_file(filepath: str) -> Glossary:
    """Load glossary from JSON file.

    Args:
        filepath: Path to glossary JSON file

    Returns:
        Glossary object

    Raises:
        FileNotFoundError: If file doesn't exist
        json.JSONDecodeError: If file contains invalid JSON
    """
    with open(filepath, "r", encoding="utf-8") as f:
        data = json.load(f)
    return load_glossary_from_dict(data)


def load_glossary_from_files(filepaths: List[str]) -> Glossary:
    """Load and merge multiple glossary files.

    Later files override earlier ones for matching source terms.

    Args:
        filepaths: List of paths to glossary JSON files

    Returns:
        Merged Glossary object
    """
    merged_entries: Dict[str, GlossaryEntry] = {}
    merged_compounds: Dict[str, CompoundTerm] = {}
    metadata: Dict[str, Any] = {}

    for filepath in filepaths:
        try:
            glos = load_glossary_from_file(filepath)

            # Merge entries (later wins)
            for entry in glos.entries:
                merged_entries[entry.source] = entry

            # Merge compounds (later wins)
            for compound in glos.compound_terms:
                merged_compounds[compound.source] = compound

            # Collect metadata
            metadata[filepath] = {
                "name": glos.name,
                "version": glos.version,
                "entry_count": len(glos.entries)
            }

        except (FileNotFoundError, json.JSONDecodeError) as e:
            logger.warning(f"Failed to load glossary from {filepath}: {e}")

    return Glossary(
        entries=list(merged_entries.values()),
        compound_terms=list(merged_compounds.values()),
        name="merged",
        metadata=metadata
    )


def create_empty_glossary(name: str = "empty") -> Glossary:
    """Create an empty glossary.

    Args:
        name: Name for the glossary

    Returns:
        Empty Glossary object
    """
    return Glossary(
        entries=[],
        compound_terms=[],
        name=name
    )
