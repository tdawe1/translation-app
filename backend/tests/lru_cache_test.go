package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tdawe1/translation-app/internal/watcher"
)

// TestLRUCache_Eviction verifies that the LRU cache evicts oldest entries when at capacity
// This test prevents P0-2: In-memory seenIDs maps leak memory
func TestLRUCache_Eviction(t *testing.T) {
	cache := watcher.NewLRUCache(3) // Max 3 items

	// Add 3 items
	assert.False(t, cache.Add("job-1"), "First add should return false (not exists)")
	assert.False(t, cache.Add("job-2"), "Second add should return false (not exists)")
	assert.False(t, cache.Add("job-3"), "Third add should return false (not exists)")
	assert.Equal(t, 3, cache.Len(), "Cache should have 3 items")

	// Add 4th item - should evict job-1 (least recently used)
	assert.False(t, cache.Add("job-4"), "Fourth add should return false (not exists)")
	assert.Equal(t, 3, cache.Len(), "Cache should still have 3 items after eviction")

	// job-1 should not exist anymore (was evicted)
	assert.False(t, cache.Add("job-1"), "Evicted item should not exist, add returns false")
}

// TestLRUCache_DuplicateDetection verifies that the cache correctly detects duplicates
func TestLRUCache_DuplicateDetection(t *testing.T) {
	cache := watcher.NewLRUCache(100)

	assert.False(t, cache.Add("job-1"), "First add should return false (not exists)")
	assert.True(t, cache.Add("job-1"), "Duplicate add should return true (already exists)")
	assert.Equal(t, 1, cache.Len(), "Cache should only have 1 item")
}

// TestLRUCache_UpdatesRecency verifies that accessing an item updates its position in LRU
func TestLRUCache_UpdatesRecency(t *testing.T) {
	cache := watcher.NewLRUCache(3)

	// Add items 1, 2, 3
	cache.Add("job-1")
	cache.Add("job-2")
	cache.Add("job-3")

	// Access job-1 (makes it recently used)
	cache.Add("job-1") // Returns true, marks as recently used

	// Add job-4 - should evict job-2 (now least recently used)
	cache.Add("job-4")

	// job-1 should still exist (was accessed)
	assert.True(t, cache.Add("job-1"), "job-1 should still exist after being accessed")

	// job-2 was evicted
	assert.False(t, cache.Add("job-2"), "job-2 should have been evicted")
}

// TestLRUCache_ThreadSafety verifies concurrent access is safe
func TestLRUCache_ThreadSafety(t *testing.T) {
	cache := watcher.NewLRUCache(1000)
	done := make(chan bool)

	// Start 10 goroutines adding items
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				cache.Add(string(rune(n)))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cache should have at most 1000 items
	assert.LessOrEqual(t, cache.Len(), 1000, "LRU cache should not exceed max size")
}
