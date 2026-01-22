package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	//go:embed translations/*/*.json
	i18nFS     embed.FS
	bundle     *i18n.Bundle
	bundleOnce sync.Once
	English    = language.English
	Spanish    = language.Spanish
	French     = language.French
	German     = language.German
	Japanese   = language.Japanese
)

// Init initializes i18n bundle
func Init() error {
	var initErr error
	bundleOnce.Do(func() {
		bundle = i18n.NewBundle(language.English)
		bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

		for _, lang := range []string{"en", "es", "fr", "de", "ja"} {
			path := fmt.Sprintf("translations/%s/active.%s.json", lang, lang)
			data, err := i18nFS.ReadFile(path)
			if err != nil {
				initErr = fmt.Errorf("failed to read embedded translation file %s: %w", path, err)
				return
			}
			if _, err := bundle.ParseMessageFileBytes(data, path); err != nil {
				initErr = fmt.Errorf("failed to parse translation file %s: %w", path, err)
				return
			}
		}
	})
	return initErr
}

// Localizer returns a localizer for given language tag.
// Panics if Init() has not been called successfully.
// This is a programming error and should be caught during development.
func Localizer(tag language.Tag) *i18n.Localizer {
	if bundle == nil {
		panic("i18n: Localizer called before Init() - call i18n.Init() at application startup")
	}
	return i18n.NewLocalizer(bundle, tag.String())
}

// GetLocalizedMessage returns a localized message with given key and template data
func GetLocalizedMessage(tag language.Tag, key string, templateData map[string]interface{}) string {
	loc := Localizer(tag)
	msg, err := loc.Localize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: templateData,
	})
	if err != nil {
		return key
	}
	return msg
}

// ParseLanguageTag parses an Accept-Language header or locale string into a language tag
func ParseLanguageTag(locale string) language.Tag {
	if locale == "" {
		return language.English
	}

	tag, err := language.Parse(locale)
	if err != nil {
		return language.English
	}
	return tag
}
