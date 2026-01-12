package watcher

import (
	"container/list"
	"sync"
)

// LRUCache is a thread-safe LRU (Least Recently Used) cache with max size.
// P0-2 FIX: Prevents unbounded memory growth from seenIDs maps.
type LRUCache struct {
	mu     sync.Mutex
	maxLen int
	ll     *list.List
	cache  map[string]*list.Element
}

// cacheEntry represents an entry in the LRU cache
type cacheEntry struct {
	key string
}

// NewLRUCache creates a new LRU cache with specified max size.
// When the cache exceeds max size, the least recently used entry is evicted.
func NewLRUCache(maxSize int) *LRUCache {
	return &LRUCache{
		maxLen: maxSize,
		ll:     list.New(),
		cache:  make(map[string]*list.Element),
	}
}

// Add marks a key as seen, returns true if was already present.
// If the key was not present, it's added and marked as most recently used.
// If the cache exceeds max size, the least recently used entry is evicted.
func (c *LRUCache) Add(key string) (exists bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ele, hit := c.cache[key]; hit {
		// Key exists: move to front (mark as recently used)
		c.ll.MoveToFront(ele)
		return true
	}

	// Add new entry at front
	ele := c.ll.PushFront(&cacheEntry{key: key})
	c.cache[key] = ele

	// Evict oldest if at capacity
	if c.ll.Len() > c.maxLen {
		oldest := c.ll.Back()
		if oldest != nil {
			c.ll.Remove(oldest)
			entry := oldest.Value.(*cacheEntry)
			delete(c.cache, entry.key)
		}
	}

	return false
}

// Len returns the current size of the cache
func (c *LRUCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}
