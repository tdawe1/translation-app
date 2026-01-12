# glossary/__init__.py
"""
Glossary system for translation consistency.

Provides loading and matching functionality for glossaries.
"""

from .loader import (
    GlossaryEntry,
    CompoundTerm,
    Glossary,
    load_glossary_from_dict,
    load_glossary_from_file,
    load_glossary_from_files,
    create_empty_glossary,
)

from .matcher import (
    GlossaryMatch,
    GlossaryMatcher,
)

__all__ = [
    "GlossaryEntry",
    "CompoundTerm",
    "Glossary",
    "GlossaryMatch",
    "GlossaryMatcher",
    "load_glossary_from_dict",
    "load_glossary_from_file",
    "load_glossary_from_files",
    "create_empty_glossary",
]
