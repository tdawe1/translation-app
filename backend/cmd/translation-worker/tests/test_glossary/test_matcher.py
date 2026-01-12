# tests/test_glossary/test_matcher.py
"""
Unit tests for glossary matcher.

Tests exact matching, variant matching, fuzzy matching,
and POS-aware matching.
"""

import pytest
import sys
import tempfile
from pathlib import Path

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from glossary.loader import (
    GlossaryEntry,
    CompoundTerm,
    Glossary,
    load_glossary_from_dict,
    load_glossary_from_file,
    create_empty_glossary,
)
from glossary.matcher import (
    GlossaryMatch,
    GlossaryMatcher,
)


class TestGlossaryEntry:
    """Test GlossaryEntry dataclass."""

    def test_entry_creation(self):
        """Should create entry with all fields."""
        entry = GlossaryEntry(
            source="顧客",
            target="customer",
            part_of_speech="noun",
            variants=["お客様"],
            forbidden_translations=["client"]
        )
        assert entry.source == "顧客"
        assert entry.target == "customer"
        assert entry.part_of_speech == "noun"
        assert entry.variants == ["お客様"]

    def test_matches_variant_exact(self):
        """Should match exact source."""
        entry = GlossaryEntry(
            source="顧客",
            target="customer",
            variants=["お客様"]
        )
        assert entry.matches_variant("顧客") is True

    def test_matches_variant_list(self):
        """Should match variant."""
        entry = GlossaryEntry(
            source="顧客",
            target="customer",
            variants=["お客様", "お客さま"]
        )
        assert entry.matches_variant("お客様") is True
        assert entry.matches_variant("お客さま") is True

    def test_is_forbidden(self):
        """Should check forbidden translations."""
        entry = GlossaryEntry(
            source="顧客",
            target="customer",
            forbidden_translations=["client", "patron"]
        )
        assert entry.is_forbidden("client") is True
        assert entry.is_forbidden("customer") is False


class TestCompoundTerm:
    """Test CompoundTerm dataclass."""

    def test_compound_creation(self):
        """Should create compound term."""
        term = CompoundTerm(
            source="顧客満足度",
            target="customer satisfaction",
            context="business"
        )
        assert term.source == "顧客満足度"
        assert term.target == "customer satisfaction"


class TestGlossary:
    """Test Glossary dataclass."""

    def test_glossary_creation(self):
        """Should create glossary with entries."""
        entries = [
            GlossaryEntry(source="顧客", target="customer"),
            GlossaryEntry(source="満足", target="satisfaction")
        ]
        glossary = Glossary(entries=entries, name="test")

        assert len(glossary.entries) == 2
        assert glossary.name == "test"

    def test_get_entry(self):
        """Should find entry by source."""
        entry = GlossaryEntry(source="顧客", target="customer")
        glossary = Glossary(entries=[entry])

        found = glossary.get_entry("顧客")
        assert found is not None
        assert found.target == "customer"

    def test_get_entry_not_found(self):
        """Should return None for missing entry."""
        glossary = Glossary(entries=[])
        assert glossary.get_entry("missing") is None

    def test_find_entries_by_pos(self):
        """Should find entries by POS tag."""
        entries = [
            GlossaryEntry(source="顧客", target="customer", part_of_speech="noun"),
            GlossaryEntry(source="調査", target="survey", part_of_speech="verb"),
            GlossaryEntry(source="する", target="do", part_of_speech="verb")
        ]
        glossary = Glossary(entries=entries)

        verbs = glossary.find_entries_by_pos("verb")
        assert len(verbs) == 2

    def test_get_all_sources(self):
        """Should get all sources including variants."""
        entries = [
            GlossaryEntry(
                source="顧客",
                target="customer",
                variants=["お客様", "お客さま"]
            )
        ]
        glossary = Glossary(entries=entries)

        sources = glossary.get_all_sources()
        assert "顧客" in sources
        assert "お客様" in sources
        assert "お客さま" in sources


class TestLoadGlossaryFromDict:
    """Test loading glossary from dict."""

    def test_load_basic_glossary(self):
        """Should load glossary from dict."""
        data = {
            "name": "test-glossary",
            "version": "1.0",
            "entries": [
                {
                    "source": "顧客",
                    "target": "customer",
                    "part_of_speech": "noun"
                }
            ]
        }
        glossary = load_glossary_from_dict(data)

        assert glossary.name == "test-glossary"
        assert glossary.version == "1.0"
        assert len(glossary.entries) == 1
        assert glossary.entries[0].source == "顧客"

    def test_load_with_variants(self):
        """Should load variants from dict."""
        data = {
            "entries": [
                {
                    "source": "顧客",
                    "target": "customer",
                    "variants": ["お客様", "お客さま"]
                }
            ]
        }
        glossary = load_glossary_from_dict(data)

        entry = glossary.entries[0]
        assert entry.variants == ["お客様", "お客さま"]

    def test_load_with_compound_terms(self):
        """Should load compound terms."""
        data = {
            "entries": [],
            "compound_terms": [
                {
                    "source": "顧客満足度",
                    "target": "customer satisfaction"
                }
            ]
        }
        glossary = load_glossary_from_dict(data)

        assert len(glossary.compound_terms) == 1
        assert glossary.compound_terms[0].source == "顧客満足度"

    def test_load_with_forbidden_translations(self):
        """Should load forbidden translations."""
        data = {
            "entries": [
                {
                    "source": "顧客",
                    "target": "customer",
                    "forbidden_translations": ["client", "patron"]
                }
            ]
        }
        glossary = load_glossary_from_dict(data)

        entry = glossary.entries[0]
        assert entry.forbidden_translations == ["client", "patron"]

    def test_load_defaults(self):
        """Should apply default values."""
        data = {"entries": []}
        glossary = load_glossary_from_dict(data)

        assert glossary.name == "default"
        assert glossary.version == "1.0"
        assert glossary.source_language == "ja"
        assert glossary.target_language == "en"


class TestLoadGlossaryFromFile:
    """Test loading glossary from file."""

    def test_load_from_json_file(self):
        """Should load glossary from JSON file."""
        with tempfile.NamedTemporaryFile(
            mode='w',
            suffix='.json',
            delete=False,
            encoding='utf-8'
        ) as f:
            f.write('''
            {
                "name": "file-test",
                "entries": [
                    {
                        "source": "テスト",
                        "target": "test"
                    }
                ]
            }
            ''')
            temp_path = f.name

        try:
            glossary = load_glossary_from_file(temp_path)
            assert glossary.name == "file-test"
            assert len(glossary.entries) == 1
            assert glossary.entries[0].source == "テスト"
        finally:
            Path(temp_path).unlink(missing_ok=True)


class TestCreateEmptyGlossary:
    """Test empty glossary creation."""

    def test_create_empty(self):
        """Should create empty glossary."""
        glossary = create_empty_glossary("test-empty")

        assert glossary.name == "test-empty"
        assert len(glossary.entries) == 0
        assert len(glossary.compound_terms) == 0


class TestGlossaryMatch:
    """Test GlossaryMatch dataclass."""

    def test_match_creation(self):
        """Should create match result."""
        entry = GlossaryEntry(source="顧客", target="customer")
        match = GlossaryMatch(
            source="顧客",
            target="customer",
            matched_text="顧客",
            entry=entry,
            match_type="exact",
            confidence=1.0
        )

        assert match.is_exact is True
        assert match.is_variant is False
        assert match.is_compound is False
        assert match.is_fuzzy is False

    def test_match_type_properties(self):
        """Should correctly identify match types."""
        entry = GlossaryEntry(source="x", target="y")

        exact = GlossaryMatch(
            source="x", target="y", matched_text="x", entry=entry,
            match_type="exact", confidence=1.0
        )
        assert exact.is_exact

        variant = GlossaryMatch(
            source="x", target="y", matched_text="z", entry=entry,
            match_type="variant", confidence=0.95
        )
        assert variant.is_variant

        compound = GlossaryMatch(
            source="x", target="y", matched_text="x", entry=entry,
            match_type="compound", confidence=0.98
        )
        assert compound.is_compound

        fuzzy = GlossaryMatch(
            source="x", target="y", matched_text="z", entry=entry,
            match_type="fuzzy", confidence=0.8
        )
        assert fuzzy.is_fuzzy


class TestGlossaryMatcher:
    """Test GlossaryMatcher functionality."""

    def test_matcher_initialization(self):
        """Should initialize with glossary."""
        glossary = Glossary(entries=[
            GlossaryEntry(source="顧客", target="customer")
        ])
        matcher = GlossaryMatcher(glossary)

        assert matcher.glossary == glossary
        assert matcher.fuzzy_threshold == 0.85

    def test_exact_match(self):
        """Should find exact match in text."""
        glossary = Glossary(entries=[
            GlossaryEntry(source="顧客", target="customer", part_of_speech="noun")
        ])
        matcher = GlossaryMatcher(glossary)

        matches = matcher.match("顧客について")

        assert len(matches) > 0
        assert matches[0].target == "customer"
        assert matches[0].is_exact

    def test_variant_match(self):
        """Should match variants."""
        glossary = Glossary(entries=[
            GlossaryEntry(
                source="顧客",
                target="customer",
                variants=["お客様", "お客さま"]
            )
        ])
        matcher = GlossaryMatcher(glossary)

        matches = matcher.match("お客様について")

        assert len(matches) > 0
        assert matches[0].target == "customer"
        assert matches[0].is_variant

    def test_multiple_variants(self):
        """Should match all variants."""
        glossary = Glossary(entries=[
            GlossaryEntry(
                source="顧客",
                target="customer",
                variants=["お客様", "お客さま"]
            )
        ])
        matcher = GlossaryMatcher(glossary)

        # Test each variant
        for variant in ["顧客", "お客様", "お客さま"]:
            matches = matcher.match(f"{variant}について")
            assert len(matches) > 0, f"Failed to match variant: {variant}"
            assert matches[0].target == "customer"

    def test_compound_match(self):
        """Should match compound terms."""
        glossary = Glossary(
            entries=[],
            compound_terms=[
                CompoundTerm(source="顧客満足度", target="customer satisfaction")
            ]
        )
        matcher = GlossaryMatcher(glossary)

        matches = matcher.match("顧客満足度を調査します")

        assert len(matches) > 0
        assert matches[0].target == "customer satisfaction"
        assert matches[0].is_compound

    def test_no_match_empty_text(self):
        """Should handle empty text."""
        glossary = Glossary(entries=[
            GlossaryEntry(source="顧客", target="customer")
        ])
        matcher = GlossaryMatcher(glossary)

        matches = matcher.match("")
        assert matches == []

    def test_confidence_sorting(self):
        """Should sort matches by confidence."""
        glossary = Glossary(entries=[
            GlossaryEntry(source="A", target="a"),
            GlossaryEntry(source="B", target="b")
        ])
        matcher = GlossaryMatcher(glossary)

        # Create a situation with multiple matches
        matches = matcher.match("AとB")

        # Check sorted descending
        for i in range(len(matches) - 1):
            assert matches[i].confidence >= matches[i+1].confidence

    def test_fuzzy_threshold(self):
        """Should respect fuzzy threshold."""
        glossary = Glossary(entries=[
            GlossaryEntry(source="見積書", target="quotation")
        ])
        matcher = GlossaryMatcher(glossary, fuzzy_threshold=0.9)

        # With high threshold, should still match exact
        matches = matcher.match("見積書です")
        assert len(matches) > 0

    def test_character_similarity_fallback(self):
        """Should use character similarity when Levenshtein unavailable."""
        glossary = Glossary(entries=[
            GlossaryEntry(source="ABC", target="xyz")
        ])
        matcher = GlossaryMatcher(glossary)

        # Should use internal similarity function
        sim = matcher._character_similarity("ABC", "ABD")
        assert 0 < sim < 1

    def test_get_glossary_summary(self):
        """Should return glossary summary."""
        glossary = Glossary(
            entries=[
                GlossaryEntry(source="A", target="a"),
                GlossaryEntry(source="B", target="b")
            ],
            compound_terms=[
                CompoundTerm(source="AB", target="ab")
            ],
            name="test",
            source_language="ja",
            target_language="en"
        )
        matcher = GlossaryMatcher(glossary, fuzzy_threshold=0.9)

        summary = matcher.get_glossary_summary()
        assert summary["total_entries"] == 2
        assert summary["total_compounds"] == 1
        assert summary["glossary_name"] == "test"
        assert summary["fuzzy_threshold"] == 0.9

    def test_inject_into_prompt(self):
        """Should format matches for prompt."""
        glossary = Glossary(entries=[
            GlossaryEntry(
                source="顧客",
                target="customer",
                context="business",
                notes="Use this for people"
            )
        ])
        matcher = GlossaryMatcher(glossary)

        matches = matcher.match("顧客")
        prompt_section = matcher.inject_into_prompt(matches)

        assert "Glossary Terms" in prompt_section
        assert "顧客 → customer" in prompt_section
        assert "business" in prompt_section

    def test_inject_forbidden_translations(self):
        """Should include forbidden translations in prompt."""
        glossary = Glossary(entries=[
            GlossaryEntry(
                source="顧客",
                target="customer",
                forbidden_translations=["client", "patron"]
            )
        ])
        matcher = GlossaryMatcher(glossary)

        matches = matcher.match("顧客")
        prompt_section = matcher.inject_into_prompt(matches)

        assert "Forbidden Translations" in prompt_section
        assert "client" in prompt_section

    def test_disable_fuzzy_matching(self):
        """Should allow disabling fuzzy matching."""
        glossary = Glossary(entries=[
            GlossaryEntry(source="顧客", target="customer")
        ])
        matcher = GlossaryMatcher(glossary, enable_fuzzy=False)

        summary = matcher.get_glossary_summary()
        assert summary["fuzzy_matching_enabled"] is False


class TestGlossaryMatcherPOSMatching:
    """Test POS-aware matching (when tokenizer available)."""

    def test_skip_pos_initialization(self):
        """Should initialize without POS if tokenizer unavailable."""
        glossary = Glossary(entries=[
            GlossaryEntry(source="test", target="test")
        ])
        matcher = GlossaryMatcher(
            glossary,
            enable_pos_matching=True
        )
        # Should not crash, just fall back to simple matching
        matches = matcher.match("test")
        assert isinstance(matches, list)

    def test_should_skip_pos(self):
        """Should correctly identify skip POS tags."""
        glossary = Glossary(entries=[])
        matcher = GlossaryMatcher(glossary)

        # Japanese particles
        assert matcher._should_skip_pos("助詞") is True
        assert matcher._should_skip_pos("助動詞") is True
        assert matcher._should_skip_pos("記号") is True

        # Content words
        assert matcher._should_skip_pos("名詞") is False
        assert matcher._should_skip_pos("動詞") is False


class TestGlossaryMatcherContextFiltering:
    """Test context-based filtering."""

    def test_filter_by_domain(self):
        """Should filter matches by domain."""
        glossary = Glossary(entries=[
            GlossaryEntry(
                source="顧客",
                target="customer",
                context="business"
            ),
            GlossaryEntry(
                source="患者",
                target="patient",
                context="medical"
            )
        ])
        matcher = GlossaryMatcher(glossary)

        matches = matcher.match("顧客と患者")
        business_matches = matcher._filter_by_context(matches, "business")

        # Should prefer business-context entries
        assert all(
            m.entry.context == "business" or not m.entry.context
            for m in business_matches
        )

    def test_filter_empty_context(self):
        """Should handle empty context gracefully."""
        glossary = Glossary(entries=[
            GlossaryEntry(source="test", target="test")
        ])
        matcher = GlossaryMatcher(glossary)

        matches = matcher.match("test")
        filtered = matcher._filter_by_context(matches, None)

        # Should return all matches when context is None
        assert filtered == matches
