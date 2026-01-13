# audit/style_checker.py
"""
Style compliance checker for JA→EN translations.

Checks translations against common style issues:
- Excessive honorific usage (-san, -sama, -kun, etc.)
- Inconsistent terminology
- Improper sentence endings
- Overly long sentences
- Passive voice overuse
- Missing articles

Gengo Style Guide rules (when enabled):
- Oxford comma in lists
- US English spelling (color, organize, favor)
- Currency format (US$1,000, ¥1,000)
- Date format (Month Day, Year)
- Time format (a.m./p.m.)

Uses configurable rules and supports custom style guides.
"""

import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import List, Dict, Optional, Set, Pattern


@dataclass
class StyleIssue:
    """A style compliance issue found during checking.

    Attributes:
        severity: Issue severity - "error", "warning", or "info"
        category: Type of issue (e.g., "honorifics", "sentence_length")
        message: Human-readable description of the issue
        location: Position in text where issue was found
        suggestion: Optional suggested fix
    """

    severity: str  # "error", "warning", "info"
    category: str
    message: str
    location: str = ""
    suggestion: Optional[str] = None

    def to_dict(self) -> Dict:
        """Convert to dictionary for JSON serialization."""
        return {
            "severity": self.severity,
            "category": self.category,
            "message": self.message,
            "location": self.location,
            "suggestion": self.suggestion,
        }


class StyleChecker:
    """Checks translation against style guide for JA→EN translations.

    Detects common issues that arise when translating from Japanese to English:
    - Excessive use of Japanese honorifics (-san, -sama, etc.)
    - Run-on sentences
    - Passive voice overuse
    - Missing definite articles
    - Subject-verb disagreement
    - Terminology inconsistency

    Example:
        >>> checker = StyleChecker()
        >>> issues = checker.check("We are honored to meet you, Tanaka-san.")
        >>> len(issues)
        1
        >>> issues[0].category
        'honorifics'
    """

    # Default honorific patterns to check
    # Matches both with hyphen (-san) and without (san as standalone word)
    DEFAULT_HONORIFICS_PATTERNS = [
        (r"(?<!\w)(-?san)(?!\w)", "Avoid using '-san' suffix in English"),
        (r"(?<!\w)(-?sama)(?!\w)", "Avoid using '-sama' suffix in English"),
        (r"(?<!\w)(-?kun)(?!\w)", "Avoid using '-kun' suffix in English"),
        (r"(?<!\w)(-?chan)(?!\w)", "Avoid using '-chan' suffix in English"),
        (
            r"(?<!\w)(-?sensei)(?!\w)",
            "Consider translating 'sensei' to 'teacher' or 'Mr./Ms.'",
        ),
        (
            r"(?<!\w)(-?senpai)(?!\w)",
            "Consider translating 'senpai' to 'senior' or 'mentor'",
        ),
    ]

    # Passive voice indicators
    # Regular "-ed" forms + common irregular past participles
    PASSIVE_PATTERNS = [
        r"\bwas\b \w+ed\b",
        r"\bwere\b \w+ed\b",
        r"\bbeen\b \w+ed\b",
        # Common irregular passive constructions
        r"\b(?:was|were|been)\s+(?:written|reached|made|taken|found|seen|done|held|kept|spent|got|put|set|let|begun|forgotten|frozen|hidden|ridden|shaken|shewn|shot|shown|shut|slept|spent|spread|stuck|stung|sung|sung|sunk|swept|swum|taught|told|thought|thrown|torn|understood|worn|woven|wrapped|written)\b",
        r"\bby\s+\w+\s+(after|in|at|on)\b",
    ]

    # Missing article patterns (simplified)
    ARTICLE_PATTERNS = [
        (r"\bis\s+[aeiou]\b", "Consider using 'an' instead of 'is'"),
        (r"\bare\s+[aeiou]\b", "Consider using 'an' instead of 'are'"),
    ]

    # UK English spellings to flag (Gengo requires US English)
    # Format: (UK spelling pattern, US spelling replacement)
    UK_SPELLING_PATTERNS = [
        (r"\b(colour)s?\b", "color"),
        (r"\b(favour)s?\b", "favor"),
        (r"\b(honour)s?\b", "honor"),
        (r"\b(labour)s?\b", "labor"),
        (r"\b(neighbour)s?\b", "neighbor"),
        (r"\b(behaviour)s?\b", "behavior"),
        (r"\b(organise)d?\b", "organize"),
        (r"\b(recognise)d?\b", "recognize"),
        (r"\b(realise)d?\b", "realize"),
        (r"\b(apologise)d?\b", "apologize"),
        (r"\b(specialise)d?\b", "specialize"),
        (r"\b(analyse)d?\b", "analyze"),
        (r"\b(centre)s?\b", "center"),
        (r"\b(metre)s?\b", "meter"),
        (r"\b(litre)s?\b", "liter"),
        (r"\b(theatre)s?\b", "theater"),
        (r"\b(defence)\b", "defense"),
        (r"\b(offence)\b", "offense"),
        (r"\b(licence)\b", "license"),
        (r"\b(practise)\b", "practice"),
        (r"\b(travelling)\b", "traveling"),
        (r"\b(cancelled)\b", "canceled"),
        (r"\b(modelling)\b", "modeling"),
        (r"\b(grey)\b", "gray"),
        (r"\b(cheque)s?\b", "check"),
        (r"\b(programme)s?\b", "program"),
        (r"\b(catalogue)s?\b", "catalog"),
        (r"\b(dialogue)s?\b", "dialog"),
    ]

    # Oxford comma pattern: matches "X, Y and Z" without comma before "and"
    # This is a simplified pattern that catches common cases
    OXFORD_COMMA_PATTERN = r"\b\w+,\s+\w+\s+and\s+\w+"

    # Currency patterns that violate Gengo style (should be US$1,000 not "1000 dollars")
    # Matches patterns like "1000 dollars", "1,000 yen", etc.
    CURRENCY_PATTERNS = [
        (
            r"\b\d{1,3}(?:,\d{3})*\s+(?:dollars?|USD)\b",
            "Use US$ symbol format (e.g., US$1,000)",
        ),
        (r"\b\d{1,3}(?:,\d{3})*\s+(?:yen|JPY)\b", "Use ¥ symbol format (e.g., ¥1,000)"),
        (
            r"\b\d{1,3}(?:,\d{3})*\s+(?:euros?|EUR)\b",
            "Use € symbol format (e.g., €1,000)",
        ),
        (
            r"\b\d{1,3}(?:,\d{3})*\s+(?:pounds?|GBP)\b",
            "Use £ symbol format (e.g., £1,000)",
        ),
    ]

    # Date format patterns (UK format to flag - should be US format: Month Day, Year)
    # Matches patterns like "21 September 2025" or "21/09/2025"
    UK_DATE_PATTERNS = [
        (
            r"\b\d{1,2}\s+(?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{4}\b",
            "Use US date format (e.g., September 21, 2025)",
        ),
        (
            r"\b\d{1,2}/\d{1,2}/\d{2,4}\b",
            "Ambiguous date format; use US format (e.g., September 21, 2025)",
        ),
    ]

    # Time format patterns (should be a.m./p.m. not AM/PM)
    TIME_FORMAT_PATTERNS = [
        (r"\b\d{1,2}:\d{2}\s*(?:AM|PM)\b", "Use lowercase a.m./p.m. (e.g., 3:00 p.m.)"),
        (r"\b\d{1,2}\s*(?:AM|PM)\b", "Use lowercase a.m./p.m. (e.g., 3 p.m.)"),
    ]

    # AI-style hyphen patterns (em-dashes without spaces are a giveaway)
    # Human translators use spaced en-dashes ( - ) or avoid hyphens entirely
    # AI models often use em-dashes (—) or (–) without spaces
    AI_HYPHEN_PATTERNS = [
        # Em-dash without spaces (AI giveaway)
        (
            r"\w—\w",
            "Avoid em-dash without spaces (AI giveaway). Use ' - ' with spaces or rephrase.",
        ),
        # En-dash without spaces
        (r"\w–\w", "Avoid en-dash without spaces. Use ' - ' with spaces or rephrase."),
        # Double hyphen (another AI pattern)
        (r"\w--\w", "Avoid double hyphen. Use ' - ' with spaces or rephrase."),
    ]

    # Default thresholds
    DEFAULT_MAX_SENTENCE_LENGTH = 200
    DEFAULT_MAX_PASSIVE_RATIO = 0.3  # 30% of sentences

    def __init__(
        self,
        style_guide_path: Optional[str] = None,
        max_sentence_length: int = DEFAULT_MAX_SENTENCE_LENGTH,
        max_passive_ratio: float = DEFAULT_MAX_PASSIVE_RATIO,
        honorifics_enabled: bool = True,
        passive_check_enabled: bool = True,
        sentence_check_enabled: bool = True,
        gengo_rules_enabled: bool = False,
    ):
        """Initialize the style checker.

        Args:
            style_guide_path: Optional path to custom style guide file
            max_sentence_length: Maximum characters before warning
            max_passive_ratio: Maximum ratio of passive voice sentences
            honorifics_enabled: Whether to check honorific usage
            passive_check_enabled: Whether to check passive voice
            sentence_check_enabled: Whether to check sentence length
            gengo_rules_enabled: Whether to enable Gengo style guide checks
        """
        self.style_guide_path = style_guide_path
        self.max_sentence_length = max_sentence_length
        self.max_passive_ratio = max_passive_ratio
        self.honorifics_enabled = honorifics_enabled
        self.passive_check_enabled = passive_check_enabled
        self.sentence_check_enabled = sentence_check_enabled
        self.gengo_rules_enabled = gengo_rules_enabled

        # Load custom rules if style guide provided
        self.custom_rules: Dict = {}
        self.forbidden_terms: Set[str] = set()
        self.preferred_terms: Dict[str, str] = {}

        if style_guide_path:
            self._load_style_guide(style_guide_path)

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
        if self.gengo_rules_enabled:
            issues.extend(self._check_gengo_rules(translation))

        # Check custom rules
        if self.custom_rules:
            issues.extend(self._check_custom_rules(translation))

        # Check terminology consistency if source provided
        if source and self.preferred_terms:
            issues.extend(self._check_terminology_consistency(translation, source))

        return issues

    def _check_honorifics(self, text: str) -> List[StyleIssue]:
        """Check for excessive honorific usage in English translation.

        Japanese honorifics like -san, -sama, -kun, -chan are typically
        not retained in formal English translations.
        """
        issues = []

        for pattern, message in self.DEFAULT_HONORIFICS_PATTERNS:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                issues.append(
                    StyleIssue(
                        severity="warning",
                        category="honorifics",
                        message=message,
                        location=f"position {match.start()}",
                        suggestion=f"Consider removing '{match.group()}' or using Mr./Ms.",
                    )
                )

        return issues

    def _check_sentence_length(self, text: str) -> List[StyleIssue]:
        """Check for overly long sentences.

        Long sentences can be difficult to read and may indicate
        run-on sentence structures common in Japanese but less
        appropriate in English.
        """
        issues = []

        # Split by sentence terminators
        sentences = re.split(r"[.!?]+", text)

        for i, sentence in enumerate(sentences):
            stripped = sentence.strip()
            if len(stripped) > self.max_sentence_length:
                issues.append(
                    StyleIssue(
                        severity="warning",
                        category="sentence_length",
                        message=f"Very long sentence detected ({len(stripped)} chars > {self.max_sentence_length})",
                        location=f"sentence {i + 1}",
                        suggestion="Consider splitting into multiple sentences",
                    )
                )

        return issues

    def _check_passive_voice(self, text: str) -> List[StyleIssue]:
        """Check for excessive passive voice usage.

        Passive voice is more common in Japanese but should be used
        sparingly in English for clearer, more direct writing.
        """
        issues = []

        # Count passive constructions
        passive_count = 0
        total_sentences = 0

        sentences = re.split(r"[.!?]+", text)
        for sentence in sentences:
            stripped = sentence.strip()
            if len(stripped) > 10:  # Only check non-fragment sentences
                total_sentences += 1
                for pattern in self.PASSIVE_PATTERNS:
                    if re.search(pattern, stripped, re.IGNORECASE):
                        passive_count += 1
                        break

        # Only flag if ratio exceeds threshold
        if total_sentences > 0:
            passive_ratio = passive_count / total_sentences
            if passive_ratio > self.max_passive_ratio:
                issues.append(
                    StyleIssue(
                        severity="info",
                        category="passive_voice",
                        message=f"High passive voice ratio ({passive_ratio:.1%} > {self.max_passive_ratio:.1%})",
                        location="overall",
                        suggestion="Consider using active voice for clearer communication",
                    )
                )

        return issues

    def _check_articles(self, text: str) -> List[StyleIssue]:
        """Check for potential article issues.

        Japanese lacks articles (a, an, the), which can lead to
        missing or incorrect articles in English translations.
        """
        issues = []

        # This is a simplified check - a full implementation would
        # use NLP to properly detect article issues
        for pattern, message in self.ARTICLE_PATTERNS:
            if re.search(pattern, text):
                issues.append(
                    StyleIssue(
                        severity="info",
                        category="articles",
                        message=message,
                        location="overall",
                    )
                )

        return issues

    def _check_gengo_rules(self, text: str) -> List[StyleIssue]:
        """Check Gengo style guide specific rules."""
        issues = []

        issues.extend(self._check_uk_spelling(text))
        issues.extend(self._check_oxford_comma(text))
        issues.extend(self._check_currency_format(text))
        issues.extend(self._check_date_format(text))
        issues.extend(self._check_time_format(text))
        issues.extend(self._check_ai_hyphens(text))

        return issues

    def _check_ai_hyphens(self, text: str) -> List[StyleIssue]:
        """Check for AI-style hyphen usage (em-dashes without spaces)."""
        issues = []

        for pattern, message in self.AI_HYPHEN_PATTERNS:
            matches = re.finditer(pattern, text)
            for match in matches:
                issues.append(
                    StyleIssue(
                        severity="error",
                        category="ai_hyphen",
                        message=message,
                        location=f"position {match.start()}",
                        suggestion="Use ' - ' with spaces on both sides, or rephrase to avoid hyphens",
                    )
                )

        return issues

    def _check_uk_spelling(self, text: str) -> List[StyleIssue]:
        """Check for UK English spellings (Gengo requires US English)."""
        issues = []
        text_lower = text.lower()

        for pattern, us_spelling in self.UK_SPELLING_PATTERNS:
            matches = re.finditer(pattern, text_lower)
            for match in matches:
                issues.append(
                    StyleIssue(
                        severity="warning",
                        category="uk_spelling",
                        message=f"Use US English spelling: '{us_spelling}'",
                        location=f"position {match.start()}",
                        suggestion=f"Replace '{match.group()}' with '{us_spelling}'",
                    )
                )

        return issues

    def _check_oxford_comma(self, text: str) -> List[StyleIssue]:
        """Check for missing Oxford comma in lists."""
        issues = []

        matches = re.finditer(self.OXFORD_COMMA_PATTERN, text, re.IGNORECASE)
        for match in matches:
            matched_text = match.group()
            if ", and " not in matched_text.lower():
                issues.append(
                    StyleIssue(
                        severity="info",
                        category="oxford_comma",
                        message="Consider using Oxford comma before 'and' in list",
                        location=f"position {match.start()}",
                        suggestion="Add comma before 'and' (e.g., 'apples, oranges, and bananas')",
                    )
                )

        return issues

    def _check_currency_format(self, text: str) -> List[StyleIssue]:
        """Check for incorrect currency format (should use symbol prefix)."""
        issues = []

        for pattern, message in self.CURRENCY_PATTERNS:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                issues.append(
                    StyleIssue(
                        severity="warning",
                        category="currency_format",
                        message=message,
                        location=f"position {match.start()}",
                        suggestion=f"Replace '{match.group()}' with symbol format",
                    )
                )

        return issues

    def _check_date_format(self, text: str) -> List[StyleIssue]:
        """Check for non-US date formats."""
        issues = []

        for pattern, message in self.UK_DATE_PATTERNS:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                issues.append(
                    StyleIssue(
                        severity="warning",
                        category="date_format",
                        message=message,
                        location=f"position {match.start()}",
                        suggestion="Use Month Day, Year format",
                    )
                )

        return issues

    def _check_time_format(self, text: str) -> List[StyleIssue]:
        """Check for incorrect time format (should be a.m./p.m.)."""
        issues = []

        for pattern, message in self.TIME_FORMAT_PATTERNS:
            matches = re.finditer(pattern, text)
            for match in matches:
                issues.append(
                    StyleIssue(
                        severity="warning",
                        category="time_format",
                        message=message,
                        location=f"position {match.start()}",
                        suggestion="Use lowercase a.m./p.m.",
                    )
                )

        return issues

    def _check_custom_rules(self, text: str) -> List[StyleIssue]:
        """Check against custom style guide rules."""
        issues = []

        # Check forbidden terms
        for term in self.forbidden_terms:
            if term.lower() in text.lower():
                issues.append(
                    StyleIssue(
                        severity="error",
                        category="forbidden_term",
                        message=f"Forbidden term '{term}' found",
                        location="overall",
                    )
                )

        return issues

    def _check_terminology_consistency(
        self,
        translation: str,
        source: str,
    ) -> List[StyleIssue]:
        """Check for terminology consistency.

        Ensures that terms are translated consistently across
        the document according to the preferred terms mapping.
        """
        issues = []

        for source_term, preferred_translation in self.preferred_terms.items():
            # Check if source term appears in source text
            if source_term not in source:
                continue

            # Check if preferred translation is used
            if preferred_translation.lower() not in translation.lower():
                issues.append(
                    StyleIssue(
                        severity="warning",
                        category="terminology",
                        message=f"Consider using '{preferred_translation}' for '{source_term}'",
                        location="overall",
                        suggestion=f"Replace with preferred term: {preferred_translation}",
                    )
                )

        return issues

    def _load_style_guide(self, path: str) -> None:
        """Load custom style guide from file.

        Expected format (TOML or simple key=value):
        [forbidden_terms]
        terms = ["term1", "term2"]

        [preferred_terms]
        "source_term" = "preferred_translation"
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

            except ImportError:
                # tomli not installed, fall back to simple parsing
                self._parse_simple_style_guide(guide_path)
            except Exception:
                # TOML parsing failed (invalid TOML), try simple format
                self._parse_simple_style_guide(guide_path)

        except Exception:
            # If loading fails, continue with defaults
            pass

    def _parse_simple_style_guide(self, path: Path) -> None:
        """Parse simple key=value style guide format."""
        with open(path, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#"):
                    continue

                if "=" in line:
                    key, value = line.split("=", 1)
                    key = key.strip()
                    value = value.strip()

                    if key == "forbidden":
                        self.forbidden_terms.add(value)
                    elif key == "preferred":
                        # Format: preferred = source|translation
                        if "|" in value:
                            source_term, translation = value.split("|", 1)
                            self.preferred_terms[source_term.strip()] = (
                                translation.strip()
                            )


def create_style_checker(
    style_guide_path: Optional[str] = None,
    max_sentence_length: int = StyleChecker.DEFAULT_MAX_SENTENCE_LENGTH,
    max_passive_ratio: float = StyleChecker.DEFAULT_MAX_PASSIVE_RATIO,
    gengo_rules_enabled: bool = False,
) -> StyleChecker:
    """Factory function to create a configured style checker.

    Args:
        style_guide_path: Optional path to custom style guide
        max_sentence_length: Maximum sentence length before warning
        max_passive_ratio: Maximum passive voice ratio
        gengo_rules_enabled: Whether to enable Gengo style guide checks

    Returns:
        Configured StyleChecker instance
    """
    return StyleChecker(
        style_guide_path=style_guide_path,
        max_sentence_length=max_sentence_length,
        max_passive_ratio=max_passive_ratio,
        gengo_rules_enabled=gengo_rules_enabled,
    )
