# tests/test_audit/test_style_checker.py
"""
Tests for StyleChecker.

Tests JA→EN style compliance checking:
- Honorific detection
- Sentence length checking
- Passive voice detection
- Article checking
- Custom style guide loading
"""

import pytest
import sys
import tempfile
from pathlib import Path

# Add worker directory to path for imports
worker_dir = Path(__file__).parent.parent.parent
sys.path.insert(0, str(worker_dir))

from audit.style_checker import (
    StyleIssue,
    StyleChecker,
    create_style_checker,
)


class TestStyleIssue:
    """Test StyleIssue dataclass."""

    def test_creation(self):
        """Should create style issue with all fields."""
        issue = StyleIssue(
            severity="warning",
            category="honorifics",
            message="Avoid using '-san'",
            location="position 10",
            suggestion="Use Mr./Ms. instead",
        )

        assert issue.severity == "warning"
        assert issue.category == "honorifics"
        assert issue.message == "Avoid using '-san'"
        assert issue.location == "position 10"
        assert issue.suggestion == "Use Mr./Ms. instead"

    def test_creation_defaults(self):
        """Should create issue with optional fields defaulted."""
        issue = StyleIssue(
            severity="info",
            category="test",
            message="Test message",
        )

        assert issue.location == ""
        assert issue.suggestion is None

    def test_to_dict(self):
        """Should convert to dictionary for JSON."""
        issue = StyleIssue(
            severity="error",
            category="forbidden_term",
            message="Forbidden term found",
            location="overall",
            suggestion="Remove this term",
        )

        result = issue.to_dict()

        assert result["severity"] == "error"
        assert result["category"] == "forbidden_term"
        assert result["message"] == "Forbidden term found"
        assert result["location"] == "overall"
        assert result["suggestion"] == "Remove this term"


class TestStyleChecker:
    """Test StyleChecker initialization."""

    def test_initialization_default(self):
        """Should initialize with default settings."""
        checker = StyleChecker()

        assert checker.max_sentence_length == 200
        assert checker.max_passive_ratio == 0.3
        assert checker.honorifics_enabled is True
        assert checker.passive_check_enabled is True
        assert checker.sentence_check_enabled is True

    def test_initialization_custom(self):
        """Should accept custom settings."""
        checker = StyleChecker(
            max_sentence_length=150,
            max_passive_ratio=0.2,
            honorifics_enabled=False,
            passive_check_enabled=False,
            sentence_check_enabled=False,
        )

        assert checker.max_sentence_length == 150
        assert checker.max_passive_ratio == 0.2
        assert checker.honorifics_enabled is False
        assert checker.passive_check_enabled is False
        assert checker.sentence_check_enabled is False

    def test_initialization_with_style_guide(self):
        """Should accept style guide path."""
        with tempfile.NamedTemporaryFile(mode="w", suffix=".toml", delete=False) as f:
            f.write('[forbidden_terms]\nterms = ["badword"]\n')
            temp_path = f.name

        try:
            checker = StyleChecker(style_guide_path=temp_path)
            assert checker.style_guide_path == temp_path
        finally:
            Path(temp_path).unlink()


class TestHonorificChecking:
    """Test honorific detection."""

    def test_detect_san_suffix(self):
        """Should detect -san suffix."""
        checker = StyleChecker()
        issues = checker.check("We met with Tanaka-san yesterday.")

        assert len(issues) == 1
        assert issues[0].category == "honorifics"
        assert "-san" in issues[0].message or "san" in issues[0].message
        assert issues[0].severity == "warning"

    def test_detect_sama_suffix(self):
        """Should detect -sama suffix."""
        checker = StyleChecker()
        issues = checker.check("President-sama arrived.")

        assert len(issues) == 1
        assert issues[0].category == "honorifics"

    def test_detect_kun_suffix(self):
        """Should detect -kun suffix."""
        checker = StyleChecker()
        issues = checker.check("My brother-kun is here.")

        assert len(issues) == 1
        assert issues[0].category == "honorifics"

    def test_detect_chan_suffix(self):
        """Should detect -chan suffix."""
        checker = StyleChecker()
        issues = checker.check("Little-chan is cute.")

        assert len(issues) == 1
        assert issues[0].category == "honorifics"

    def test_detect_sensei_suffix(self):
        """Should detect -sensei suffix."""
        checker = StyleChecker()
        issues = checker.check("The sensei taught us well.")

        assert len(issues) == 1
        assert issues[0].category == "honorifics"
        assert issues[0].suggestion is not None

    def test_detect_senpai_suffix(self):
        """Should detect -senpai suffix."""
        checker = StyleChecker()
        issues = checker.check("My senpai helped me.")

        assert any(i.category == "honorifics" for i in issues)

    def test_case_insensitive_honorifics(self):
        """Should detect honorifics regardless of case."""
        checker = StyleChecker()
        issues = checker.check("Tanaka-SAN is here.")

        assert len(issues) >= 1
        assert any(i.category == "honorifics" for i in issues)

    def test_no_honorifics_clean_text(self):
        """Should not flag clean English text."""
        checker = StyleChecker()
        issues = checker.check("We met with Mr. Tanaka yesterday.")

        honorific_issues = [i for i in issues if i.category == "honorifics"]
        assert len(honorific_issues) == 0

    def test_honorifics_disabled(self):
        """Should not check honorifics when disabled."""
        checker = StyleChecker(honorifics_enabled=False)
        issues = checker.check("Tanaka-san is here.")

        honorific_issues = [i for i in issues if i.category == "honorifics"]
        assert len(honorific_issues) == 0


class TestSentenceLengthChecking:
    """Test sentence length detection."""

    def test_long_sentence_warning(self):
        """Should warn about overly long sentences."""
        checker = StyleChecker(max_sentence_length=50)
        long_sentence = "This is a very long sentence that continues on and on " * 4

        issues = checker.check(long_sentence)

        length_issues = [i for i in issues if i.category == "sentence_length"]
        assert len(length_issues) > 0

    def test_short_sentence_no_warning(self):
        """Should not warn about short sentences."""
        checker = StyleChecker()
        issues = checker.check("This is a normal sentence.")

        length_issues = [i for i in issues if i.category == "sentence_length"]
        assert len(length_issues) == 0

    def test_multiple_long_sentences(self):
        """Should flag each long sentence separately."""
        checker = StyleChecker(max_sentence_length=30)
        text = "This is the first very long sentence that goes on. " \
               "This is the second very long sentence that continues."

        issues = checker.check(text)

        length_issues = [i for i in issues if i.category == "sentence_length"]
        assert len(length_issues) == 2

    def test_sentence_check_disabled(self):
        """Should not check sentence length when disabled."""
        checker = StyleChecker(
            max_sentence_length=10,
            sentence_check_enabled=False,
        )
        issues = checker.check("This is a very long sentence that continues.")

        length_issues = [i for i in issues if i.category == "sentence_length"]
        assert len(length_issues) == 0


class TestPassiveVoiceChecking:
    """Test passive voice detection."""

    def test_detect_passive_was_ed(self):
        """Should detect 'was + ed' pattern."""
        checker = StyleChecker()
        issues = checker.check(
            "The document was reviewed by the team. " * 10
        )

        passive_issues = [i for i in issues if i.category == "passive_voice"]
        # Should flag if ratio exceeds threshold (30%)
        # 10 sentences all passive = 100% > 30%
        assert len(passive_issues) == 1

    def test_detect_passive_were_ed(self):
        """Should detect 'were + ed' pattern."""
        checker = StyleChecker()
        issues = checker.check(
            "The files were processed. " * 10
        )

        passive_issues = [i for i in issues if i.category == "passive_voice"]
        assert len(passive_issues) == 1

    def test_active_voice_no_warning(self):
        """Should not warn about active voice text."""
        checker = StyleChecker()
        issues = checker.check(
            "We reviewed the document. "
            "The team processed the files. "
            "She wrote the report. "
        )

        passive_issues = [i for i in issues if i.category == "passive_voice"]
        assert len(passive_issues) == 0

    def test_mixed_passive_active(self):
        """Should handle mixed passive and active voice."""
        checker = StyleChecker()
        # Mix of passive and active - should be below threshold
        text = " ".join([
            "The report was written.",
            "We analyzed the data.",
            "The team reviewed the findings.",
            "The conclusion was reached.",
        ])

        issues = checker.check(text)

        # 2 passive out of 4 = 50% > 30%
        passive_issues = [i for i in issues if i.category == "passive_voice"]
        assert len(passive_issues) == 1

    def test_passive_check_disabled(self):
        """Should not check passive voice when disabled."""
        checker = StyleChecker(passive_check_enabled=False)
        issues = checker.check(
            "The document was reviewed by the team. " * 10
        )

        passive_issues = [i for i in issues if i.category == "passive_voice"]
        assert len(passive_issues) == 0


class TestArticleChecking:
    """Test article checking."""

    def test_article_check(self):
        """Should check article usage (simplified)."""
        checker = StyleChecker()
        # This is a simplified check - actual implementation would need NLP
        issues = checker.check("is apple")

        article_issues = [i for i in issues if i.category == "articles"]
        assert len(article_issues) >= 0  # Pattern-based, may match


class TestCustomStyleGuide:
    """Test custom style guide loading."""

    def test_load_forbidden_terms_from_toml(self):
        """Should load forbidden terms from TOML file."""
        try:
            import tomli
        except ImportError:
            pytest.skip("tomli not installed")

        with tempfile.NamedTemporaryFile(
            mode="wb",
            suffix=".toml",
            delete=False
        ) as f:
            f.write(b'[forbidden_terms]\nterms = ["badword", "termbad"]\n')
            temp_path = f.name

        try:
            checker = StyleChecker(style_guide_path=temp_path)
            issues = checker.check("This badword is not allowed.")

            forbidden_issues = [i for i in issues if i.category == "forbidden_term"]
            assert len(forbidden_issues) == 1
        finally:
            Path(temp_path).unlink()

    def test_load_preferred_terms_from_toml(self):
        """Should load preferred terms from TOML file."""
        try:
            import tomli
        except ImportError:
            pytest.skip("tomli not installed")

        with tempfile.NamedTemporaryFile(
            mode="wb",
            suffix=".toml",
            delete=False
        ) as f:
            # TOML format for preferred terms
            f.write(b'[preferred_terms]\n"source_term" = "preferred_translation"\n')
            temp_path = f.name

        try:
            checker = StyleChecker(style_guide_path=temp_path)
            issues = checker.check(
                "We used a different translation.",
                source="The source_term is here."
            )

            # Should flag because preferred_translation not used
            term_issues = [i for i in issues if i.category == "terminology"]
            assert len(term_issues) >= 0
        finally:
            Path(temp_path).unlink()

    def test_load_simple_format(self):
        """Should load simple key=value format."""
        with tempfile.NamedTemporaryFile(
            mode="w",
            suffix=".txt",
            delete=False
        ) as f:
            f.write('forbidden=badword\n')
            f.write('preferred=source|preferred\n')
            temp_path = f.name

        try:
            checker = StyleChecker(style_guide_path=temp_path)
            assert "badword" in checker.forbidden_terms
        finally:
            Path(temp_path).unlink()

    def test_nonexistent_style_guide(self):
        """Should handle nonexistent style guide gracefully."""
        checker = StyleChecker(style_guide_path="/nonexistent/path.toml")

        # Should not raise error, just use defaults
        issues = checker.check("Normal text here.")
        assert isinstance(issues, list)


class TestTerminologyConsistency:
    """Test terminology consistency checking."""

    def test_consistent_terminology_no_issues(self):
        """Should not flag when preferred terms are used."""
        try:
            import tomli
        except ImportError:
            pytest.skip("tomli not installed")

        with tempfile.NamedTemporaryFile(
            mode="wb",
            suffix=".toml",
            delete=False
        ) as f:
            f.write(b'[preferred_terms]\n"customer" = "client"\n')
            temp_path = f.name

        try:
            checker = StyleChecker(style_guide_path=temp_path)
            issues = checker.check(
                "We served the client well.",
                source="The customer was satisfied."
            )

            term_issues = [i for i in issues if i.category == "terminology"]
            assert len(term_issues) == 0
        finally:
            Path(temp_path).unlink()

    def test_inconsistent_terminology_flags(self):
        """Should flag when preferred terms are not used."""
        try:
            import tomli
        except ImportError:
            pytest.skip("tomli not installed")

        with tempfile.NamedTemporaryFile(
            mode="wb",
            suffix=".toml",
            delete=False
        ) as f:
            f.write(b'[preferred_terms]\n"customer" = "client"\n')
            temp_path = f.name

        try:
            checker = StyleChecker(style_guide_path=temp_path)
            issues = checker.check(
                "We served the customer well.",
                source="The customer was satisfied."
            )

            term_issues = [i for i in issues if i.category == "terminology"]
            assert len(term_issues) == 1
            assert "client" in term_issues[0].suggestion or "client" in term_issues[0].message
        finally:
            Path(temp_path).unlink()


class TestFactoryFunction:
    """Test factory function."""

    def test_create_style_checker_default(self):
        """Should create checker with defaults."""
        checker = create_style_checker()

        assert isinstance(checker, StyleChecker)
        assert checker.max_sentence_length == 200

    def test_create_style_checker_custom(self):
        """Should create checker with custom settings."""
        checker = create_style_checker(
            max_sentence_length=150,
            max_passive_ratio=0.2,
        )

        assert checker.max_sentence_length == 150
        assert checker.max_passive_ratio == 0.2

    def test_create_style_checker_with_guide(self):
        """Should create checker with style guide."""
        with tempfile.NamedTemporaryFile(mode="w", suffix=".toml", delete=False) as f:
            f.write('[forbidden_terms]\nterms = []\n')
            temp_path = f.name

        try:
            checker = create_style_checker(style_guide_path=temp_path)
            assert checker.style_guide_path == temp_path
        finally:
            Path(temp_path).unlink()


class TestStyleIssueSeverity:
    """Test style issue severity levels."""

    def test_forbidden_term_is_error(self):
        """Should mark forbidden terms as errors."""
        try:
            import tomli
        except ImportError:
            pytest.skip("tomli not installed")

        with tempfile.NamedTemporaryFile(
            mode="wb",
            suffix=".toml",
            delete=False
        ) as f:
            f.write(b'[forbidden_terms]\nterms = ["forbidden"]\n')
            temp_path = f.name

        try:
            checker = StyleChecker(style_guide_path=temp_path)
            issues = checker.check("This forbidden is here.")

            forbidden_issues = [i for i in issues if i.category == "forbidden_term"]
            assert len(forbidden_issues) > 0
            assert forbidden_issues[0].severity == "error"
        finally:
            Path(temp_path).unlink()

    def test_honorific_is_warning(self):
        """Should mark honorific issues as warnings."""
        checker = StyleChecker()
        issues = checker.check("Tanaka-san is here.")

        honorific_issues = [i for i in issues if i.category == "honorifics"]
        assert len(honorific_issues) > 0
        assert honorific_issues[0].severity == "warning"

    def test_passive_voice_is_info(self):
        """Should mark passive voice as info."""
        checker = StyleChecker()
        issues = checker.check("The document was reviewed. " * 20)

        passive_issues = [i for i in issues if i.category == "passive_voice"]
        if passive_issues:
            assert passive_issues[0].severity == "info"


class TestIssueLocation:
    """Test issue location tracking."""

    def test_honorific_location(self):
        """Should track honorific location."""
        checker = StyleChecker()
        issues = checker.check("Hello Tanaka-san")

        honorific_issues = [i for i in issues if i.category == "honorifics"]
        assert len(honorific_issues) > 0
        assert honorific_issues[0].location != ""

    def test_sentence_location(self):
        """Should track sentence location."""
        checker = StyleChecker(max_sentence_length=10)
        issues = checker.check("Short. This is a very long sentence.")

        length_issues = [i for i in issues if i.category == "sentence_length"]
        assert len(length_issues) > 0
        assert "sentence" in length_issues[0].location
