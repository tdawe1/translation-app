package i18n

import (
	"sync"
	"testing"

	"golang.org/x/text/language"
)

func TestInit(t *testing.T) {
	// Reset bundle for testing
	bundle = nil
	bundleOnce = sync.Once{}

	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	if bundle == nil {
		t.Fatal("bundle should not be nil after Init()")
	}

	// Test idempotency - calling Init() again should not cause issues
	err = Init()
	if err != nil {
		t.Fatalf("Init() should be idempotent: %v", err)
	}
}

func TestLocalizer_Initialized(t *testing.T) {
	// Ensure bundle is initialized
	bundle = nil
	bundleOnce = sync.Once{}
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Test that Localizer works when initialized
	loc := Localizer(language.English)
	if loc == nil {
		t.Fatal("Localizer() returned nil")
	}
}

func TestLocalizer_NotInitialized(t *testing.T) {
	// Reset bundle to simulate uninitialized state
	bundle = nil
	bundleOnce = sync.Once{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Localizer() should panic when bundle is not initialized")
		} else {
			// Verify panic message
			msg, ok := r.(string)
			if !ok {
				t.Fatalf("Expected panic with string message, got %T", r)
			}
			expectedMsg := "i18n bundle not initialized: ensure Init() is called at application startup"
			if msg != expectedMsg {
				t.Fatalf("Expected panic message %q, got %q", expectedMsg, msg)
			}
		}

		// Re-initialize for other tests
		Init()
	}()

	// This should panic
	Localizer(language.English)
}

func TestLocalizer_ConcurrentAccess(t *testing.T) {
	// Ensure bundle is initialized
	bundle = nil
	bundleOnce = sync.Once{}
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Test concurrent access to Localizer
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			loc := Localizer(language.English)
			if loc == nil {
				t.Error("Localizer() returned nil in concurrent access")
			}
		}()
	}
	wg.Wait()
}

func TestGetLocalizedMessage(t *testing.T) {
	// Ensure bundle is initialized
	bundle = nil
	bundleOnce = sync.Once{}
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	tests := []struct {
		name         string
		tag          language.Tag
		key          string
		templateData map[string]interface{}
	}{
		{
			name:         "English message",
			tag:          language.English,
			key:          "test_key",
			templateData: nil,
		},
		{
			name:         "Spanish message",
			tag:          language.Spanish,
			key:          "test_key",
			templateData: nil,
		},
		{
			name: "Message with template data",
			tag:  language.English,
			key:  "test_key",
			templateData: map[string]interface{}{
				"name": "Test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := GetLocalizedMessage(tt.tag, tt.key, tt.templateData)
			// Should return the key if translation not found
			if msg == "" {
				t.Error("GetLocalizedMessage() returned empty string")
			}
		})
	}
}

func TestParseLanguageTag(t *testing.T) {
	tests := []struct {
		name     string
		locale   string
		expected language.Tag
	}{
		{
			name:     "Empty string defaults to English",
			locale:   "",
			expected: language.English,
		},
		{
			name:     "Valid English locale",
			locale:   "en",
			expected: language.English,
		},
		{
			name:     "Valid Spanish locale",
			locale:   "es",
			expected: language.Spanish,
		},
		{
			name:     "Valid French locale",
			locale:   "fr",
			expected: language.French,
		},
		{
			name:     "Valid German locale",
			locale:   "de",
			expected: language.German,
		},
		{
			name:     "Valid Japanese locale",
			locale:   "ja",
			expected: language.Japanese,
		},
		{
			name:     "Invalid locale defaults to English",
			locale:   "invalid",
			expected: language.English,
		},
		{
			name:     "Complex locale string",
			locale:   "en-US",
			expected: language.AmericanEnglish,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLanguageTag(tt.locale)
			if result != tt.expected {
				t.Errorf("ParseLanguageTag(%q) = %v, want %v", tt.locale, result, tt.expected)
			}
		})
	}
}
