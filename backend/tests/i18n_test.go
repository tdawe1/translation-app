package tests

import (
	"testing"

	appi18n "github.com/tdawe1/translation-app/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

// TestI18nInit tests that i18n bundle initializes without errors
func TestI18nInit(t *testing.T) {
	// Init should succeed with the current JSON format
	err := appi18n.Init()
	require.NoError(t, err, "i18n.Init() should succeed with array-format JSON files")
}

// TestI18nLocalizer tests that localizers can be created and messages retrieved
func TestI18nLocalizer(t *testing.T) {
	// Ensure i18n is initialized
	err := appi18n.Init()
	require.NoError(t, err)

	tests := []struct {
		name     string
		lang     language.Tag
		messageID string
		expected string
	}{
		{
			name:      "English common.loading",
			lang:      appi18n.English,
			messageID: "common.loading",
			expected:  "Loading...",
		},
		{
			name:      "Spanish common.error",
			lang:      appi18n.Spanish,
			messageID: "common.error",
			expected:  "Error",
		},
		{
			name:      "French auth.invalidCredentials",
			lang:      appi18n.French,
			messageID: "auth.invalidCredentials",
			expected:  "E-mail ou mot de passe invalide",
		},
		{
			name:      "German watcher.notConfigured",
			lang:      appi18n.German,
			messageID: "watcher.notConfigured",
			expected:  "Watcher nicht konfiguriert",
		},
		{
			name:      "Japanese common.success",
			lang:      appi18n.Japanese,
			messageID: "common.success",
			expected:  "成功",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := appi18n.GetLocalizedMessage(tt.lang, tt.messageID, nil)
			assert.Equal(t, tt.expected, msg)
		})
	}
}

// TestI18nTemplateData tests messages with template data
func TestI18nTemplateData(t *testing.T) {
	// Ensure i18n is initialized
	err := appi18n.Init()
	require.NoError(t, err)

	// Test validation.minValue with template data
	msg := appi18n.GetLocalizedMessage(
		appi18n.English,
		"validation.minValue",
		map[string]interface{}{"Value": 10},
	)
	assert.Equal(t, "Value must be at least 10", msg)

	// Test server.serverStarted with template data
	msg = appi18n.GetLocalizedMessage(
		appi18n.English,
		"server.serverStarted",
		map[string]interface{}{"Address": ":8000"},
	)
	assert.Equal(t, "Server started on :8000", msg)
}

// TestI18nFallbackToKey tests that unknown message IDs return the key itself
func TestI18nFallbackToKey(t *testing.T) {
	// Ensure i18n is initialized
	err := appi18n.Init()
	require.NoError(t, err)

	unknownKey := "unknown.message.key"
	msg := appi18n.GetLocalizedMessage(appi18n.English, unknownKey, nil)
	assert.Equal(t, unknownKey, msg, "Should return key when message not found")
}

// TestParseLanguageTag tests language tag parsing
func TestParseLanguageTag(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected language.Tag
	}{
		{
			name:     "Empty string defaults to English",
			input:    "",
			expected: language.English,
		},
		{
			name:     "Simple language code",
			input:    "es",
			expected: language.Spanish,
		},
		{
			name:     "Accept-Language header with region",
			input:    "fr-FR,fr;q=0.9,en;q=0.8",
			expected: language.MustParse("fr-FR"),
		},
		{
			name:     "Invalid locale defaults to English",
			input:    "invalid-locale-xyz",
			expected: language.English,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := appi18n.ParseLanguageTag(tt.input)
			assert.Equal(t, tt.expected, tag)
		})
	}
}
