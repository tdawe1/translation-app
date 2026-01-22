package main

import (
	"fmt"
	"os"

	appi18n "github.com/tdawe1/translation-app/internal/i18n"
	"golang.org/x/text/language"
)

func main() {
	// Initialize i18n
	if err := appi18n.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize i18n: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== i18n Embedded Files Demo ===")
	fmt.Println()

	// Test all supported languages
	languages := []struct {
		name string
		tag  language.Tag
	}{
		{"English", appi18n.English},
		{"Spanish", appi18n.Spanish},
		{"French", appi18n.French},
		{"German", appi18n.German},
		{"Japanese", appi18n.Japanese},
	}

	for _, lang := range languages {
		msg := appi18n.GetLocalizedMessage(lang.tag, "common.loading", nil)
		fmt.Printf("%s: %s\n", lang.name, msg)
	}

	fmt.Println()
	fmt.Println("✓ All translation files successfully embedded!")
	fmt.Println("✓ This binary can run from any directory without external translation files.")
}
