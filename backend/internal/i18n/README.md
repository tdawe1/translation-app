# i18n Package - Embedded Translation Files

## Overview

The i18n package provides internationalization support for the GengoWatcher SaaS application. Translation files are **embedded directly into the Go binary** using Go's `embed` package, ensuring they are always available regardless of the deployment environment or execution context.

## Architecture

### Embedded File System

Translation files are embedded at compile time using the `//go:embed` directive:

```go
//go:embed translations/*/*.json
var i18nFS embed.FS
```

This approach provides several benefits:

1. **Deployment Independence**: Translation files are always available, regardless of where the binary is executed
2. **No File Path Issues**: Eliminates brittle relative path dependencies
3. **Self-Contained Binary**: The server binary contains everything it needs to run
4. **Atomic Deployments**: Files and code are versioned together as a single unit
5. **Performance**: No disk I/O required to read translation files at runtime

### Supported Languages

The package currently supports five languages:

- English (en)
- Spanish (es)
- French (fr)
- German (de)
- Japanese (ja)

Each language has its translation file at `translations/{lang}/active.{lang}.json`.

## Usage

### Initialization

The i18n package must be initialized before use, typically during application startup:

```go
import appi18n "github.com/tdawe1/translation-app/internal/i18n"

func main() {
    if err := appi18n.Init(); err != nil {
        log.Fatal("Failed to initialize i18n:", err)
    }
    // ... rest of application setup
}
```

### Getting Localized Messages

```go
import (
    appi18n "github.com/tdawe1/translation-app/internal/i18n"
    "golang.org/x/text/language"
)

// Simple message lookup
msg := appi18n.GetLocalizedMessage(appi18n.English, "common.loading", nil)
// Returns: "Loading..."

// With template data
templateData := map[string]interface{}{
    "Name": "John",
    "Count": 5,
}
msg := appi18n.GetLocalizedMessage(appi18n.Spanish, "greeting.hello", templateData)
```

### Parsing Language Tags

The package provides a helper to parse Accept-Language headers or locale strings:

```go
// From HTTP header
acceptLang := r.Header.Get("Accept-Language")
tag := appi18n.ParseLanguageTag(acceptLang)

// From locale string
tag := appi18n.ParseLanguageTag("es-MX")
```

### Direct Localizer Access

For more control, you can get a localizer instance:

```go
localizer := appi18n.Localizer(appi18n.French)
msg, err := localizer.Localize(&i18n.LocalizeConfig{
    MessageID: "auth.invalidCredentials",
})
```

## File Structure

```
internal/i18n/
├── localizer.go           # Main package implementation
├── localizer_test.go      # Comprehensive test suite
└── translations/          # Translation files (embedded)
    ├── en/
    │   └── active.en.json
    ├── es/
    │   └── active.es.json
    ├── fr/
    │   └── active.fr.json
    ├── de/
    │   └── active.de.json
    └── ja/
        └── active.ja.json
```

## Translation File Format

Translation files use the format expected by `go-i18n/v2`:

```json
{
  "common": {
    "loading": "Loading...",
    "error": "Error",
    "success": "Success"
  },
  "auth": {
    "invalidCredentials": "Invalid email or password",
    "userExists": "User already exists"
  }
}
```

## Testing

The package includes comprehensive tests that verify:

1. **Initialization**: Bundle is properly initialized
2. **Localizer Creation**: Localizers work for all supported languages
3. **Message Retrieval**: Messages are correctly translated
4. **Embedded Files**: All translation files are properly embedded
5. **Language Tag Parsing**: Accept-Language headers are correctly parsed

Run tests with:

```bash
cd backend
go test ./internal/i18n/... -v
```

## Adding New Languages

To add a new language:

1. Create a new translation file: `translations/{lang}/active.{lang}.json`
2. Add the language constant to `localizer.go`:
   ```go
   var (
       // ... existing languages
       Portuguese = language.Portuguese
   )
   ```
3. Add the language to the initialization loop:
   ```go
   for _, lang := range []string{"en", "es", "fr", "de", "ja", "pt"} {
       // ...
   }
   ```
4. Add test cases to `localizer_test.go`

## Implementation Details

### Thread Safety

The package uses `sync.Once` to ensure the bundle is initialized exactly once, making it safe for concurrent use:

```go
var (
    bundle     *i18n.Bundle
    bundleOnce sync.Once
)

func Init() error {
    var initErr error
    bundleOnce.Do(func() {
        // ... initialization code
    })
    return initErr
}
```

### Error Handling

- If a message key is not found, the function returns the key itself as a fallback
- If initialization fails, the error is returned to the caller
- Missing translation files cause initialization to fail early

### Performance

- Translation files are embedded at compile time (zero runtime overhead)
- Bundle initialization happens once via `sync.Once`
- No disk I/O required during normal operation
- Localizer instances are lightweight and can be created on-demand

## Migration from File-Based Loading

If your code previously used file-based loading (e.g., `os.ReadFile`), the migration to embedded files is transparent:

**Before:**
```go
data, err := os.ReadFile(fmt.Sprintf("i18n/%s/active.%s.json", lang, lang))
```

**After:**
```go
// The package handles this internally now
data, err := i18nFS.ReadFile(fmt.Sprintf("translations/%s/active.%s.json", lang, lang))
```

No changes are required in consuming code - the API remains the same.
