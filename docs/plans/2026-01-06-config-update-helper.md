# Config Update Helper Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate 43 lines of manual field-by-field config updates in `handlers/watcher.go` by creating a generic `ApplyPartialUpdate()` helper function.

**Architecture:** Create a reflection-based helper that updates only non-zero/non-nil fields from a request struct onto a model, using struct tags for field mapping.

**Tech Stack:** Go 1.25, reflection package, existing GORM models

**Documentation:** `docs/api/watcher-endpoints.md`

---

## Overview

Current problem in `handlers/watcher.go:121-163`:
```go
updates := make(map[string]interface{})

if req.RSSFeedURL != "" {
    updates["RSSFeedURL"] = req.RSSFeedURL
}
if req.WebSocketEnabled != nil {
    updates["WebSocketEnabled"] = *req.WebSocketEnabled
}
if req.GengoUserID != "" {
    updates["GengoUserID"] = req.GengoUserID
}
// ... 12 more fields
```

This is error-prone and doesn't scale. We'll create a generic helper.

---

## Task 1: Create Generic Update Helper

**Files:**
- Create: `backend/internal/handlers/update.go`

**Step 1: Write failing test**

Create: `backend/internal/handlers/update_test.go`

```go
package handlers

import (
    "testing"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "github.com/tdawe1/translation-app/internal/models"
)

// Mock struct for testing
type TestRequest struct {
    StringField  string
    IntField     *int
    BoolField    *bool
    FloatField   *float64
    IgnoreField  string // Should not update if empty
}

type TestModel struct {
    StringField  string
    IntField     int
    BoolField    bool
    FloatField   float64
    IgnoreField  string
    UpdatedAt    time.Time
}

func TestApplyPartialUpdate_AllFields(t *testing.T) {
    req := TestRequest{
        StringField: "new value",
        IntField:    intPtr(42),
        BoolField:   boolPtr(true),
        FloatField:  float64Ptr(3.14),
    }

    model := TestModel{
        StringField: "old value",
        IntField:    10,
        BoolField:   false,
        FloatField:  1.5,
    }

    updates := ApplyPartialUpdate(req)

    // Verify all fields are included
    if updates["StringField"] != "new value" {
        t.Errorf("Expected StringField update, got %v", updates["StringField"])
    }
    if updates["IntField"] != 42 {
        t.Errorf("Expected IntField update, got %v", updates["IntField"])
    }
    if updates["BoolField"] != true {
        t.Errorf("Expected BoolField update, got %v", updates["BoolField"])
    }
    if updates["FloatField"] != 3.14 {
        t.Errorf("Expected FloatField update, got %v", updates["FloatField"])
    }
}

func TestApplyPartialUpdate_OnlyProvidedFields(t *testing.T) {
    req := TestRequest{
        StringField: "updated",
        // IntField, BoolField, FloatField not provided (nil)
    }

    updates := ApplyPartialUpdate(req)

    // Only StringField should be in updates
    if len(updates) != 1 {
        t.Errorf("Expected 1 update, got %d", len(updates))
    }
    if updates["StringField"] != "updated" {
        t.Errorf("Expected StringField update, got %v", updates["StringField"])
    }
}

func TestApplyPartialUpdate_SkipEmptyString(t *testing.T) {
    req := TestRequest{
        StringField: "", // Empty string should be skipped
        IntField:    intPtr(10),
    }

    updates := ApplyPartialUpdate(req)

    // Empty StringField should not be in updates
    if _, exists := updates["StringField"]; exists {
        t.Error("Empty string should be skipped")
    }
    if updates["IntField"] != 10 {
        t.Errorf("Expected IntField update, got %v", updates["IntField"])
    }
}

// Helper functions
func intPtr(i int) *int     { return &i }
func boolPtr(b bool) *bool   { return &b }
func float64Ptr(f float64) *float64 { return &f }
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/handlers/... -v -run TestApplyPartialUpdate`

Expected: FAIL with "undefined: ApplyPartialUpdate"

**Step 3: Implement ApplyPartialUpdate**

Create: `backend/internal/handlers/update.go`

```go
package handlers

import (
    "reflect"
    "strings"
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

// ApplyPartialUpdate creates a map of fields to update from a request struct.
// Only includes non-zero values:
//   - strings: non-empty
//   - pointers: non-nil
//   - time.Time: non-zero
//   - uuid.UUID: non-zero
//   - slices/maps: non-nil and non-empty
//
// Field names are converted from CamelCase to snake_case for database columns.
func ApplyPartialUpdate(req interface{}) map[string]interface{} {
    updates := make(map[string]interface{})

    val := reflect.ValueOf(req)
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }

    // Handle if it's still a pointer (e.g., **T)
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }

    typ := val.Type()

    for i := 0; i < val.NumField(); i++ {
        field := val.Field(i)
        fieldType := typ.Field(i)

        // Skip unexported fields
        if !field.IsExported() {
            continue
        }

        // Get the field name (convert CamelCase to snake_case)
        fieldName := camelToSnake(fieldType.Name)

        // Skip based on field type and value
        if shouldSkipField(field) {
            continue
        }

        // Add to updates
        updates[fieldName] = field.Interface()
    }

    return updates
}

// shouldSkipField determines if a field should be skipped based on its value
func shouldSkipField(field reflect.Value) bool {
    kind := field.Kind()

    switch kind {
    case reflect.String:
        return field.String() == ""

    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return field.Int() == 0

    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        return field.Uint() == 0

    case reflect.Float32, reflect.Float64:
        return field.Float() == 0

    case reflect.Bool:
        // Always include bool fields if explicitly set
        return false

    case reflect.Struct:
        // Handle special struct types
        if field.Type() == reflect.TypeOf(time.Time{}) {
            return field.Interface().(time.Time).IsZero()
        }
        if field.Type() == reflect.TypeOf(uuid.UUID{}) {
            return field.Interface().(uuid.UUID) == uuid.Nil
        }
        // For other structs, check if it's a zero value
        return reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface())

    case reflect.Ptr, reflect.Interface:
        return field.IsNil()

    case reflect.Slice, reflect.Map:
        return field.IsNil() || field.Len() == 0

    default:
        return false
    }
}

// camelToSnake converts CamelCase to snake_case
func camelToSnake(s string) string {
    var result []rune
    for i, r := range s {
        if i > 0 && r >= 'A' && r <= 'Z' {
            result = append(result, '_')
        }
        result = append(result, r)
    }
    return strings.ToLower(string(result))
}

// ApplyModelUpdates applies updates to a model using GORM
func ApplyModelUpdates(db *gorm.DB, model interface{}, updates map[string]interface{}) error {
    return db.Model(model).Updates(updates).Error
}
```

**Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/handlers/... -v -run TestApplyPartialUpdate`

Expected: PASS

**Step 5: Add edge case tests**

Add to `backend/internal/handlers/update_test.go`:

```go
func TestCamelToSnake(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"RSSFeedURL", "rss_feed_url"},
        {"WebSocketEnabled", "websocket_enabled"},
        {"GengoUserID", "gengo_user_id"},
        {"ID", "id"},
        {"MinReward", "min_reward"},
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            if got := camelToSnake(tt.input); got != tt.expected {
                t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, got, tt.expected)
            }
        })
    }
}

func TestApplyPartialUpdate_WithRealWatcherConfig(t *testing.T) {
    req := struct {
        RSSFeedURL     string
        MinReward       *float64
        AutoAcceptEnabled *bool
    }{
        RSSFeedURL:     "https://example.com/feed",
        MinReward:       float64Ptr(5.50),
        AutoAcceptEnabled: boolPtr(true),
    }

    updates := ApplyPartialUpdate(req)

    expectedFields := []string{"rss_feed_url", "min_reward", "auto_accept_enabled"}
    if len(updates) != len(expectedFields) {
        t.Errorf("Expected %d fields, got %d", len(expectedFields), len(updates))
    }

    for _, field := range expectedFields {
        if _, exists := updates[field]; !exists {
            t.Errorf("Missing field: %s", field)
        }
    }
}
```

**Step 6: Run all tests**

Run: `cd backend && go test ./internal/handlers/... -v -run TestApplyPartialUpdate`

Expected: All PASS

**Step 7: Commit**

```bash
cd backend
git add internal/handlers/update.go internal/handlers/update_test.go
git commit -m "feat(handlers): add ApplyPartialUpdate helper for config updates"
```

---

## Task 2: Refactor UpdateConfig to Use ApplyPartialUpdate

**Files:**
- Modify: `backend/internal/handlers/watcher.go:85-182`

**Step 1: Write test for UpdateConfig refactoring**

Create test in `backend/internal/handlers/watcher_test.go`:

```go
func TestWatcherHandler_UpdateConfig_PartialUpdate(t *testing.T) {
    // Integration test to verify UpdateConfig uses new helper
    t.Skip("Integration test - add after refactoring")
}
```

**Step 2: Run test (should skip)**

Run: `cd backend && go test ./internal/handlers/... -v -run TestWatcherHandler_UpdateConfig`

Expected: PASS (skipped)

**Step 3: Refactor UpdateConfig function**

Edit: `backend/internal/handlers/watcher.go:85-182`

Replace the entire UpdateConfig function:

```go
// UpdateConfig updates the user's watcher configuration
func (h *WatcherHandler) UpdateConfig(c *fiber.Ctx) error {
    return middleware.RequireAuth(h.updateConfigLogic)(c)
}

// updateConfigLogic contains the actual UpdateConfig logic after auth is verified
func (h *WatcherHandler) updateConfigLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
    var req UpdateConfigRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
            "code":  "INVALID_REQUEST",
        })
    }

    // Load existing config
    var config models.WatcherConfig
    if err := h.db.Where("user_id = ?", userUUID).First(&config).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Watcher config not found",
            "code":  "CONFIG_NOT_FOUND",
        })
    }

    // Apply partial updates using helper
    updates := ApplyPartialUpdate(req)

    // Special handling for IncludedLanguagePairs (JSON field)
    if req.IncludedLanguagePairs != nil {
        jsonPairs, _ := json.Marshal(req.IncludedLanguagePairs)
        updates["IncludedLanguagePairs"] = string(jsonPairs)
    }

    // Apply updates to database
    if err := h.db.Model(&config).Updates(updates).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to update config",
            "code":  "UPDATE_ERROR",
        })
    }

    // Reload config
    if err := h.db.Where("user_id = ?", userUUID).First(&config).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to reload config",
            "code":  "RELOAD_ERROR",
        })
    }

    return c.JSON(configToResponse(&config))
}
```

**Step 4: Add missing imports**

Edit: `backend/internal/handlers/watcher.go:1-14`

Ensure these imports are present:
```go
    "encoding/json"
    "github.com/google/uuid"
```

**Step 5: Run tests**

Run: `cd backend && go test ./internal/handlers/... -v -run TestWatcherHandler`

Expected: PASS

**Step 6: Manual smoke test**

Run: `cd backend && go run cmd/server/main.go`

Test with curl:
```bash
curl -X PUT http://localhost:8080/api/v1/watcher/config \
  -H "Content-Type: application/json" \
  -H "Cookie: session_token=<valid-token>" \
  -d '{"min_reward": 5.00, "auto_accept_enabled": true}'
```

Expected: Config updates successfully

**Step 7: Commit**

```bash
cd backend
git add internal/handlers/watcher.go
git commit -m "refactor(watcher): use ApplyPartialUpdate helper - reduces from 43 to 10 lines"
```

---

## Task 3: Apply Pattern to Other Config Updates

**Files:**
- Check for similar patterns in other handlers

**Step 1: Search for similar manual update patterns**

Run: `cd backend && grep -r "updates\[\"" internal/handlers/ | grep -v "update.go"`

Expected: Minimal results (most config updates should be in watcher.go)

**Step 2: If other files exist, apply same pattern**

For each file found:
1. Create UpdateRequest struct if doesn't exist
2. Use ApplyPartialUpdate helper
3. Test and commit

**Step 3: Verify no more manual updates**

Run: `cd backend && grep -r "updates\[" internal/handlers/ | wc -l`

Expected: Count should be minimal (only in update.go itself)

**Step 4: Final commit if changes made**

```bash
cd backend
git add -A
git commit -m "refactor(handlers): apply ApplyPartialUpdate pattern to all config updates"
```

---

## Task 4: Update Documentation

**Files:**
- Modify: `docs/api/watcher-endpoints.md`

**Step 1: Document the update pattern**

Add to `docs/api/watcher-endpoints.md`:

```markdown
## Partial Updates

When sending update requests, only include the fields you want to change.
Omitted fields will not be modified.

Example - updating only min_reward and auto_accept:
\`\`\`json
{
  "min_reward": 5.00,
  "auto_accept_enabled": true
}
\`\`\`

Empty string values are ignored. To explicitly clear a string field, send null
(except where noted).

### Adding New Config Fields

When adding new configuration fields to `UpdateConfigRequest`:

1. Add the field to the struct with `json` tag
2. Add the corresponding database column via migration
3. The `ApplyPartialUpdate` helper will automatically include it in updates
4. No manual mapping code needed
\`\`\`
```

**Step 2: Run documentation build**

Run: `cd docs && mkdocs build 2>/dev/null || echo "MkDocs not configured - skipping"`

**Step 3: Commit**

```bash
git add docs/api/watcher-endpoints.md
git commit -m "docs(api): document partial update pattern and field addition guide"
```

---

## Task 5: Final Verification

**Step 1: Run full test suite**

Run: `cd backend && go test ./... -v -run TestApplyPartialUpdate`

Expected: All PASS

**Step 2: Test with real-world scenarios**

Create test in `backend/internal/handlers/watcher_integration_test.go`:

```go
//go:build integration

package handlers

import (
    "testing"
    "github.com/gofiber/fiber/v2"
)

func TestUpdateConfig_Integration(t *testing.T) {
    // This test requires a running database
    t.Skip("Integration test - requires database")

    // Test various update scenarios
    scenarios := []struct {
        name   string
        req    UpdateConfigRequest
        expect map[string]interface{}
    }{
        {
            name: "single field",
            req: UpdateConfigRequest{
                MinReward: float64Ptr(5.00),
            },
            expect: map[string]interface{}{"min_reward": 5.00},
        },
        {
            name: "multiple fields",
            req: UpdateConfigRequest{
                MinReward:       float64Ptr(5.00),
                MaxReward:       float64Ptr(10.00),
                AutoAcceptEnabled: boolPtr(true),
            },
            expect: map[string]interface{}{
                "min_reward": 5.00,
                "max_reward": 10.00,
                "auto_accept_enabled": true,
            },
        },
        {
            name: "empty request",
            req: UpdateConfigRequest{},
            expect: map[string]interface{}{},
        },
    }

    for _, tt := range scenarios {
        t.Run(tt.name, func(t *testing.T) {
            updates := ApplyPartialUpdate(tt.req)
            // Verify updates match expectations
            for k, v := range tt.expect {
                if updates[k] != v {
                    t.Errorf("Field %s: expected %v, got %v", k, v, updates[k])
                }
            }
        })
    }
}
```

**Step 3: Build verification**

Run: `cd backend && go build ./cmd/server`

Expected: Build succeeds

**Step 4: Final commit**

```bash
cd backend
git add -A
git commit -m "feat(config): complete config update helper implementation"
```

---

## Success Criteria

- [ ] `ApplyPartialUpdate()` function created and tested
- [ ] `UpdateConfig` refactored from 43 to ~10 lines
- [ ] Zero manual field-by-field update patterns remaining
- [ ] Documentation updated with new pattern
- [ ] All tests passing

---

## Next Steps

After completing this plan:

1. **Split Email Handler** - Refactor 504-line god file
2. **Add Handler Tests** - Increase test coverage to 60%
3. **Repository Pattern** - Abstract database access layer

---

**Estimated Time**: 4 hours
**Lines Removed**: ~50
**New Code**: ~120 lines (with comprehensive tests)
