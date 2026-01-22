package i18n

import (
	"testing"

	"golang.org/x/text/language"
)

func TestInit(t *testing.T) {
	// Test that initialization works
	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Verify bundle is initialized
	if bundle == nil {
		t.Fatal("bundle should not be nil after Init()")
	}
}

func TestLocalizer(t *testing.T) {
	// Initialize first
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	tests := []struct {
		name string
		tag  language.Tag
	}{
		{"English", English},
		{"Spanish", Spanish},
		{"French", French},
		{"German", German},
		{"Japanese", Japanese},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localizer := Localizer(tt.tag)
			if localizer == nil {
				t.Errorf("Localizer(%v) returned nil", tt.tag)
			}
		})
	}
}

func TestGetLocalizedMessage(t *testing.T) {
	// Initialize first
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	tests := []struct {
		name     string
		tag      language.Tag
		key      string
		wantText string
	}{
		{
			name:     "English loading message",
			tag:      English,
			key:      "common.loading",
			wantText: "Loading...",
		},
		{
			name:     "Spanish loading message",
			tag:      Spanish,
			key:      "common.loading",
			wantText: "Cargando...",
		},
		{
			name:     "French loading message",
			tag:      French,
			key:      "common.loading",
			wantText: "Chargement...",
		},
		{
			name:     "German loading message",
			tag:      German,
			key:      "common.loading",
			wantText: "Laden...",
		},
		{
			name:     "Japanese loading message",
			tag:      Japanese,
			key:      "common.loading",
			wantText: "読み込み中...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := GetLocalizedMessage(tt.tag, tt.key, nil)
			if msg != tt.wantText {
				t.Errorf("GetLocalizedMessage(%v, %q) = %q, want %q", tt.tag, tt.key, msg, tt.wantText)
			}
		})
	}
}

func TestGetLocalizedMessageWithTemplateData(t *testing.T) {
	// Initialize first
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Test with a key that uses template data (if exists in translations)
	// For now, test that missing key returns the key itself
	msg := GetLocalizedMessage(English, "nonexistent.key", nil)
	if msg != "nonexistent.key" {
		t.Errorf("GetLocalizedMessage with missing key should return the key, got %q", msg)
	}
}

func TestParseLanguageTag(t *testing.T) {
	tests := []struct {
		name   string
		locale string
		want   language.Tag
	}{
		{
			name:   "Empty string defaults to English",
			locale: "",
			want:   English,
		},
		{
			name:   "English",
			locale: "en",
			want:   English,
		},
		{
			name:   "Spanish",
			locale: "es",
			want:   Spanish,
		},
		{
			name:   "French",
			locale: "fr",
			want:   French,
		},
		{
			name:   "German",
			locale: "de",
			want:   German,
		},
		{
			name:   "Japanese",
			locale: "ja",
			want:   Japanese,
		},
		{
			name:   "Invalid locale defaults to English",
			locale: "invalid-locale",
			want:   English,
		},
		{
			name:   "Accept-Language header format",
			locale: "en-US",
			want:   language.MustParse("en-US"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLanguageTag(tt.locale)
			if got.String() != tt.want.String() {
				t.Errorf("ParseLanguageTag(%q) = %v, want %v", tt.locale, got, tt.want)
			}
		})
	}
}

func TestEmbeddedFilesExist(t *testing.T) {
	// Test that all expected translation files are embedded
	languages := []string{"en", "es", "fr", "de", "ja"}

	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			path := "translations/" + lang + "/active." + lang + ".json"
			data, err := i18nFS.ReadFile(path)
			if err != nil {
				t.Errorf("Failed to read embedded file %s: %v", path, err)
			}
			if len(data) == 0 {
				t.Errorf("Embedded file %s is empty", path)
			}
		})
	}
}
