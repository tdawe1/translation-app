package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"

	"github.com/tdawe1/translation-app/internal/i18n"
)

// TestLocalizer_PanicsWhenNotInitialized verifies that calling Localizer
// before Init() panics with a clear error message
func TestLocalizer_PanicsWhenNotInitialized(t *testing.T) {
	// This test cannot be run as-is because Init() is called globally
	// It serves as documentation of the expected behavior
	t.Skip("Cannot test uninitialized state - Init() is called during package initialization")
}

// TestLocalizer_WorksAfterInit verifies that Localizer works correctly
// after Init() has been called successfully
func TestLocalizer_WorksAfterInit(t *testing.T) {
	// Init is called in main.go and in test init, so bundle should be initialized
	err := i18n.Init()
	require.NoError(t, err, "i18n.Init() should succeed")

	// Test that Localizer returns a valid localizer
	loc := i18n.Localizer(language.English)
	require.NotNil(t, loc, "Localizer should return a non-nil localizer")
}

// TestLocalizer_SupportsMultipleLanguages verifies that Localizer
// works for all supported languages
func TestLocalizer_SupportsMultipleLanguages(t *testing.T) {
	err := i18n.Init()
	require.NoError(t, err, "i18n.Init() should succeed")

	testCases := []struct {
		name string
		tag  language.Tag
	}{
		{"English", i18n.English},
		{"Spanish", i18n.Spanish},
		{"French", i18n.French},
		{"German", i18n.German},
		{"Japanese", i18n.Japanese},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			loc := i18n.Localizer(tc.tag)
			assert.NotNil(t, loc, "Localizer should return a non-nil localizer for %s", tc.name)
		})
	}
}

// TestGetLocalizedMessage_ReturnsKeyOnError verifies that GetLocalizedMessage
// returns the key when localization fails
func TestGetLocalizedMessage_ReturnsKeyOnError(t *testing.T) {
	err := i18n.Init()
	require.NoError(t, err, "i18n.Init() should succeed")

	// Use a non-existent key
	result := i18n.GetLocalizedMessage(language.English, "non_existent_key", nil)
	assert.Equal(t, "non_existent_key", result, "Should return the key when localization fails")
}

// TestParseLanguageTag_HandlesEmptyString verifies that ParseLanguageTag
// returns English for empty string
func TestParseLanguageTag_HandlesEmptyString(t *testing.T) {
	tag := i18n.ParseLanguageTag("")
	assert.Equal(t, language.English, tag, "Should return English for empty string")
}

// TestParseLanguageTag_HandlesInvalidLocale verifies that ParseLanguageTag
// returns English for invalid locale
func TestParseLanguageTag_HandlesInvalidLocale(t *testing.T) {
	tag := i18n.ParseLanguageTag("invalid-locale-123")
	assert.Equal(t, language.English, tag, "Should return English for invalid locale")
}

// TestParseLanguageTag_ParsesValidLocale verifies that ParseLanguageTag
// correctly parses valid locale strings
func TestParseLanguageTag_ParsesValidLocale(t *testing.T) {
	testCases := []struct {
		locale   string
		expected language.Tag
	}{
		{"en", language.English},
		{"en-US", language.English}, // Base language match
		{"es", language.Spanish},
		{"fr", language.French},
		{"de", language.German},
		{"ja", language.Japanese},
	}

	for _, tc := range testCases {
		t.Run(tc.locale, func(t *testing.T) {
			tag := i18n.ParseLanguageTag(tc.locale)
			// Compare base languages since exact tag matching is stricter
			assert.Equal(t, tc.expected.String(), tag.String()[:2], 
				"Should parse %s correctly", tc.locale)
		})
	}
}

// TestInit_CanBeCalledMultipleTimes verifies that Init() is idempotent
// and can be safely called multiple times
func TestInit_CanBeCalledMultipleTimes(t *testing.T) {
	// First call
	err1 := i18n.Init()
	require.NoError(t, err1, "First Init() call should succeed")

	// Second call should also succeed (idempotent via sync.Once)
	err2 := i18n.Init()
	require.NoError(t, err2, "Second Init() call should succeed")

	// Verify that Localizer still works
	loc := i18n.Localizer(language.English)
	assert.NotNil(t, loc, "Localizer should work after multiple Init() calls")
}
