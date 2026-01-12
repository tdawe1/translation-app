# glossary/matcher.py
"""
Glossary matching with fuzzy and POS-aware support.

Provides matching functionality for finding glossary terms in text,
with support for fuzzy matching and part-of-speech filtering.
"""

from dataclasses import dataclass
from typing import List, Optional, Set, Tuple, Dict
import logging

try:
    from nlp.tokenizer import JapaneseTokenizer, TokenizationResult, Token
    TOKENIZER_AVAILABLE = True
except ImportError:
    TOKENIZER_AVAILABLE = False
    JapaneseTokenizer = None
    TokenizationResult = None
    Token = None

from .loader import Glossary, GlossaryEntry, CompoundTerm

logger = logging.getLogger(__name__)


@dataclass
class GlossaryMatch:
    """A glossary match result.

    Attributes:
        source: The original glossary source term
        target: The glossary target translation
        matched_text: The actual text that was matched
        entry: The glossary entry
        match_type: Type of match ("exact", "variant", "compound", "fuzzy")
        confidence: Match confidence score (0.0 to 1.0)
        pos_match: Whether POS filtering was used
        start_index: Starting position in text (if available)
        end_index: Ending position in text (if available)
    """
    source: str
    target: str
    matched_text: str
    entry: GlossaryEntry
    match_type: str  # "exact", "variant", "compound", "fuzzy"
    confidence: float
    pos_match: bool = False
    start_index: int = -1
    end_index: int = -1

    @property
    def is_exact(self) -> bool:
        return self.match_type == "exact"

    @property
    def is_variant(self) -> bool:
        return self.match_type == "variant"

    @property
    def is_compound(self) -> bool:
        return self.match_type == "compound"

    @property
    def is_fuzzy(self) -> bool:
        return self.match_type == "fuzzy"


class GlossaryMatcher:
    """Matches text against glossary with exact, fuzzy, and POS-aware matching.

    Example:
        >>> from glossary.loader import load_glossary_from_dict
        >>> data = {
        ...     "entries": [
        ...         {"source": "顧客", "target": "customer", "part_of_speech": "noun"}
        ...     ]
        ... }
        >>> glossary = load_glossary_from_dict(data)
        >>> matcher = GlossaryMatcher(glossary)
        >>> matches = matcher.match("顧客について")
        >>> matches[0].target
        'customer'
    """

    # POS tags that should typically be skipped (function words)
    SKIP_POS: Set[str] = {
        "助詞",      # Particle
        "助動詞",    # Auxiliary verb
        "記号",      # Symbol/punctuation
        "接続詞",    # Conjunction
        "接頭辞",    # Prefix
        "接尾辞",    # Suffix
        "particle",  # English tag
        "conjunction",
        "punctuation"
    }

    # Default confidence scores by match type
    CONFIDENCE = {
        "exact": 1.0,
        "variant": 0.95,
        "compound": 0.98,
        "fuzzy": 0.8  # Multiplied by similarity ratio
    }

    def __init__(
        self,
        glossary: Glossary,
        fuzzy_threshold: float = 0.85,
        enable_pos_matching: bool = True,
        enable_fuzzy: bool = True
    ):
        """Initialize the glossary matcher.

        Args:
            glossary: The glossary to match against
            fuzzy_threshold: Minimum similarity ratio for fuzzy matches (0-1)
            enable_pos_matching: Whether to use POS-aware matching
            enable_fuzzy: Whether to enable fuzzy matching
        """
        self.glossary = glossary
        self.fuzzy_threshold = fuzzy_threshold
        self.enable_pos_matching = enable_pos_matching and TOKENIZER_AVAILABLE
        self.enable_fuzzy = enable_fuzzy

        # Initialize tokenizer if POS matching is enabled
        self.tokenizer: Optional[JapaneseTokenizer] = None
        if self.enable_pos_matching:
            try:
                self.tokenizer = JapaneseTokenizer()
            except RuntimeError as e:
                logger.warning(f"Could not initialize tokenizer: {e}")
                self.enable_pos_matching = False

        # Build lookup indices
        self._build_indices()

    def _build_indices(self):
        """Build lookup indices for efficient matching."""
        # Build exact source -> entry index
        self.exact_index: Dict[str, GlossaryEntry] = {
            e.source: e for e in self.glossary.entries
        }

        # Build variant -> entry index
        self.variant_index: Dict[str, GlossaryEntry] = {}
        for entry in self.glossary.entries:
            for variant in entry.variants:
                self.variant_index[variant] = entry

        # Build compound term index (sorted by length desc for longest match)
        self.compound_index: List[CompoundTerm] = sorted(
            self.glossary.compound_terms,
            key=lambda c: len(c.source),
            reverse=True
        )

    def match(
        self,
        text: str,
        context: Optional[dict] = None
    ) -> List[GlossaryMatch]:
        """Find all glossary matches in text.

        Args:
            text: Text to search for glossary terms
            context: Optional context dict with domain, tone, etc.

        Returns:
            List of matches sorted by confidence (descending)
        """
        if not text:
            return []

        matches: List[GlossaryMatch] = []

        if self.enable_pos_matching and self.tokenizer:
            matches = self._match_with_pos(text)
        else:
            matches = self._match_without_pos(text)

        # Apply context filtering if provided
        if context and "domain" in context:
            matches = self._filter_by_context(matches, context["domain"])

        # Sort by confidence descending
        matches.sort(key=lambda m: m.confidence, reverse=True)

        return matches

    def _match_with_pos(self, text: str) -> List[GlossaryMatch]:
        """Match using POS-aware tokenization."""
        matches: List[GlossaryMatch] = []
        seen_sources: Set[str] = set()

        try:
            result: TokenizationResult = self.tokenizer.tokenize(text)
        except Exception as e:
            logger.warning(f"Tokenization failed, falling back to simple match: {e}")
            return self._match_without_pos(text)

        # First pass: exact and variant matches with POS filtering
        for token in result.tokens:
            # Skip function words (particles, conjunctions, etc.)
            if self._should_skip_pos(token.pos):
                continue

            # Check for exact match
            if token.text in self.exact_index:
                entry = self.exact_index[token.text]
                if entry.source not in seen_sources:
                    matches.append(GlossaryMatch(
                        source=entry.source,
                        target=entry.target,
                        matched_text=token.text,
                        entry=entry,
                        match_type="exact",
                        confidence=self.CONFIDENCE["exact"],
                        pos_match=True
                    ))
                    seen_sources.add(entry.source)

            # Check for variant match
            if token.text in self.variant_index:
                entry = self.variant_index[token.text]
                if entry.source not in seen_sources:
                    matches.append(GlossaryMatch(
                        source=entry.source,
                        target=entry.target,
                        matched_text=token.text,
                        entry=entry,
                        match_type="variant",
                        confidence=self.CONFIDENCE["variant"],
                        pos_match=True
                    ))
                    seen_sources.add(entry.source)

        # Second pass: compound term matching
        for compound in self.compound_index:
            if compound.source in text:
                # Create a pseudo-entry for compound
                pseudo_entry = GlossaryEntry(
                    source=compound.source,
                    target=compound.target,
                    part_of_speech="compound",
                    context=compound.context
                )
                if compound.source not in seen_sources:
                    # Find position in text
                    start_idx = text.find(compound.source)
                    matches.append(GlossaryMatch(
                        source=compound.source,
                        target=compound.target,
                        matched_text=compound.source,
                        entry=pseudo_entry,
                        match_type="compound",
                        confidence=self.CONFIDENCE["compound"],
                        pos_match=True,
                        start_index=start_idx,
                        end_index=start_idx + len(compound.source)
                    ))
                    seen_sources.add(compound.source)

        # Third pass: fuzzy matching for remaining unmatched entries
        if self.enable_fuzzy:
            matches.extend(self._fuzzy_match_tokens(text, result.tokens, seen_sources))

        return matches

    def _match_without_pos(self, text: str) -> List[GlossaryMatch]:
        """Match without POS tagging (simpler, faster)."""
        matches: List[GlossaryMatch] = []
        seen_sources: Set[str] = set()

        # Exact matches (check longest first)
        sorted_sources = sorted(
            self.exact_index.keys(),
            key=len,
            reverse=True
        )

        for source in sorted_sources:
            if source in text and source not in seen_sources:
                entry = self.exact_index[source]
                start_idx = text.find(source)
                matches.append(GlossaryMatch(
                    source=source,
                    target=entry.target,
                    matched_text=source,
                    entry=entry,
                    match_type="exact",
                    confidence=self.CONFIDENCE["exact"],
                    pos_match=False,
                    start_index=start_idx,
                    end_index=start_idx + len(source)
                ))
                seen_sources.add(source)

        # Variant matches
        for variant, entry in self.variant_index.items():
            if variant in text and entry.source not in seen_sources:
                start_idx = text.find(variant)
                matches.append(GlossaryMatch(
                    source=entry.source,
                    target=entry.target,
                    matched_text=variant,
                    entry=entry,
                    match_type="variant",
                    confidence=self.CONFIDENCE["variant"],
                    pos_match=False,
                    start_index=start_idx,
                    end_index=start_idx + len(variant)
                ))
                seen_sources.add(entry.source)

        # Compound term matching
        for compound in self.compound_index:
            if compound.source in text and compound.source not in seen_sources:
                pseudo_entry = GlossaryEntry(
                    source=compound.source,
                    target=compound.target,
                    part_of_speech="compound",
                    context=compound.context
                )
                start_idx = text.find(compound.source)
                matches.append(GlossaryMatch(
                    source=compound.source,
                    target=compound.target,
                    matched_text=compound.source,
                    entry=pseudo_entry,
                    match_type="compound",
                    confidence=self.CONFIDENCE["compound"],
                    pos_match=False,
                    start_index=start_idx,
                    end_index=start_idx + len(compound.source)
                ))
                seen_sources.add(compound.source)

        # Fuzzy matching
        if self.enable_fuzzy:
            matches.extend(self._fuzzy_match_text(text, seen_sources))

        return matches

    def _should_skip_pos(self, pos: str) -> bool:
        """Check if POS tag should be skipped (function word)."""
        if not pos:
            return False
        # Check if any skip POS tag is contained in the given pos
        for skip_tag in self.SKIP_POS:
            if skip_tag in pos:
                return True
        return False

    def _fuzzy_match_tokens(
        self,
        text: str,
        tokens: List[Token],
        seen_sources: Set[str]
    ) -> List[GlossaryMatch]:
        """Apply fuzzy matching to tokens not already matched."""
        matches: List[GlossaryMatch] = []

        for token in tokens:
            # Skip if we already have this source
            if token.text in seen_sources:
                continue

            for entry in self.glossary.entries:
                if entry.source in seen_sources:
                    continue

                # Check similarity using simple character overlap
                similarity = self._calculate_similarity(token.text, entry.source)
                if similarity >= self.fuzzy_threshold:
                    confidence = similarity * self.CONFIDENCE["fuzzy"]
                    matches.append(GlossaryMatch(
                        source=entry.source,
                        target=entry.target,
                        matched_text=token.text,
                        entry=entry,
                        match_type="fuzzy",
                        confidence=confidence,
                        pos_match=True
                    ))
                    seen_sources.add(entry.source)
                    break

        return matches

    def _fuzzy_match_text(self, text: str, seen_sources: Set[str]) -> List[GlossaryMatch]:
        """Apply fuzzy matching for text without tokenization."""
        matches: List[GlossaryMatch] = []

        for entry in self.glossary.entries:
            if entry.source in seen_sources:
                continue

            # Only consider fuzzy match if source is somewhat similar to text
            if len(entry.source) > len(text) * 2 or len(text) > len(entry.source) * 2:
                continue

            # Check for substring match first
            if entry.source in text or text in entry.source:
                similarity = self._calculate_similarity(entry.source, text)
                if similarity >= self.fuzzy_threshold:
                    confidence = similarity * self.CONFIDENCE["fuzzy"]
                    matches.append(GlossaryMatch(
                        source=entry.source,
                        target=entry.target,
                        matched_text=entry.source,
                        entry=entry,
                        match_type="fuzzy",
                        confidence=confidence,
                        pos_match=False
                    ))
                    seen_sources.add(entry.source)

        return matches

    def _calculate_similarity(self, s1: str, s2: str) -> float:
        """Calculate similarity ratio between two strings.

        Uses Levenshtein-like character-based similarity.
        Falls back to simple Jaccard-like similarity if Levenshtein unavailable.
        """
        # Try to use python-Levenshtein if available
        try:
            import Levenshtein
            return Levenshtein.ratio(s1, s2)
        except ImportError:
            # Fallback to simple character overlap similarity
            return self._character_similarity(s1, s2)

    def _character_similarity(self, s1: str, s2: str) -> float:
        """Calculate character-based similarity (Jaccard-like)."""
        set1 = set(s1)
        set2 = set(s2)

        if not set1 and not set2:
            return 1.0
        if not set1 or not set2:
            return 0.0

        intersection = len(set1 & set2)
        union = len(set1 | set2)

        return intersection / union if union > 0 else 0.0

    def _filter_by_context(
        self,
        matches: List[GlossaryMatch],
        domain: str
    ) -> List[GlossaryMatch]:
        """Filter matches by domain/context.

        Args:
            matches: List of matches to filter
            domain: Domain to filter by

        Returns:
            Filtered list of matches
        """
        if not domain:
            return matches

        # Prefer matches with matching context
        domain_lower = domain.lower()
        filtered = []

        # First add matches with matching context
        for match in matches:
            if (match.entry.context and
                domain_lower in match.entry.context.lower()):
                filtered.append(match)

        # Then add matches without context context (lower priority)
        for match in matches:
            if not match.entry.context and match not in filtered:
                filtered.append(match)

        return filtered

    def inject_into_prompt(self, matches: List[GlossaryMatch]) -> str:
        """Generate glossary section for system prompt.

        Args:
            matches: List of matches to format

        Returns:
            Formatted string for LLM prompt
        """
        if not matches:
            return ""

        lines = ["## Glossary Terms", ""]
        for match in matches[:20]:  # Limit to top 20 matches
            line = f"- {match.source} → {match.target}"
            if match.entry.context:
                line += f" ({match.entry.context})"
            if match.entry.notes:
                line += f" [Note: {match.entry.notes}]"
            lines.append(line)

        # Add forbidden translations warning
        forbidden = [
            m for m in matches
            if m.entry.forbidden_translations
        ]
        if forbidden:
            lines.append("")
            lines.append("### Forbidden Translations:")
            for match in forbidden:
                for forbidden_trans in match.entry.forbidden_translations:
                    lines.append(f"- Do NOT use '{forbidden_trans}' for {match.source}")

        return "\n".join(lines)

    def get_glossary_summary(self) -> dict:
        """Get summary statistics about the loaded glossary.

        Returns:
            Dict with entry counts and statistics
        """
        return {
            "total_entries": len(self.glossary.entries),
            "total_compounds": len(self.glossary.compound_terms),
            "source_language": self.glossary.source_language,
            "target_language": self.glossary.target_language,
            "glossary_name": self.glossary.name,
            "pos_matching_enabled": self.enable_pos_matching,
            "fuzzy_matching_enabled": self.enable_fuzzy,
            "fuzzy_threshold": self.fuzzy_threshold
        }
