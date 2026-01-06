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

		// Skip unexported fields (they can't be interfaced)
		if !field.CanInterface() {
			continue
		}

		// Get the field name (convert CamelCase to snake_case)
		fieldName := camelToSnake(fieldType.Name)

		// Skip based on field type and value
		if shouldSkipField(field) {
			continue
		}

		// Add to updates (dereference pointers if needed)
		updates[fieldName] = getValueForUpdate(field)
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

	case reflect.Ptr:
		return field.IsNil()

	case reflect.Interface:
		return field.IsNil()

	case reflect.Slice, reflect.Map:
		return field.IsNil() || field.Len() == 0

	default:
		return false
	}
}

// getValueForUpdate returns the value to store in updates map.
// For pointers, this dereferences them to get the underlying value.
func getValueForUpdate(field reflect.Value) interface{} {
	if field.Kind() == reflect.Ptr && !field.IsNil() {
		return field.Elem().Interface()
	}
	return field.Interface()
}

// camelToSnake converts CamelCase to snake_case
// Handles consecutive capitals (e.g., RSSFeedURL -> rss_feed_url)
// and acronyms (e.g., ID -> id, HTTP -> http)
func camelToSnake(s string) string {
	var result []rune
	for i, r := range s {
		// Insert underscore before capital if:
		// 1. Not at the start, AND
		// 2. Previous char was lowercase, OR next char is lowercase
		// This handles: "FeedURL" -> "feed_url", "RSSFeed" -> "rss_feed"
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := s[i-1]
			next := byte(0)
			if i < len(s)-1 {
				next = s[i+1]
			}
			// Add underscore if: prev is lowercase OR next is lowercase
			if prev >= 'a' && prev <= 'z' || (next >= 'a' && next <= 'z') {
				result = append(result, '_')
			}
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// ApplyModelUpdates applies updates to a model using GORM
func ApplyModelUpdates(db *gorm.DB, model interface{}, updates map[string]interface{}) error {
	return db.Model(model).Updates(updates).Error
}
