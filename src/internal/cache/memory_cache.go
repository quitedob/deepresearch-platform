package cache

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrCacheKeyNotFound is returned when a key is not found in the cache
	ErrCacheKeyNotFound = errors.New("cache key not found")
	// ErrCacheKeyExpired is returned when a key has expired
	ErrCacheKeyExpired = errors.New("cache key expired")
)

// cacheEntry represents a single cache entry with expiration
type cacheEntry struct {
	key       string
	value     interface{}
	expiresAt time.Time
}

// MemoryCache implements an in-memory LRU cache
type MemoryCache struct {
	maxSize int
	mu      sync.RWMutex
	items   map[string]*list.Element
	lru     *list.List
}

// NewMemoryCache creates a new in-memory cache with LRU eviction
func NewMemoryCache(maxSize int) *MemoryCache {
	m := &MemoryCache{
		maxSize: maxSize,
		items:   make(map[string]*list.Element),
		lru:     list.New(),
	}

	// 启动后台清理协程，定期淘汰过期条目
	go m.cleanupExpired()

	return m
}

// cleanupExpired 定期清理过期缓存条目（防止只写不读的 key 永久驻留内存）
func (m *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		var toDelete []string
		for key, elem := range m.items {
			entry := elem.Value.(*cacheEntry)
			if now.After(entry.expiresAt) {
				toDelete = append(toDelete, key)
			}
		}
		for _, key := range toDelete {
			if elem, ok := m.items[key]; ok {
				m.lru.Remove(elem)
				delete(m.items, key)
			}
		}
		m.mu.Unlock()
	}
}

// Get retrieves a value from the memory cache
func (m *MemoryCache) Get(ctx context.Context, key string) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	elem, exists := m.items[key]
	if !exists {
		return nil, ErrCacheKeyNotFound
	}

	entry := elem.Value.(*cacheEntry)

	// Check if the entry has expired
	if time.Now().After(entry.expiresAt) {
		// Remove expired entry
		m.lru.Remove(elem)
		delete(m.items, key)
		return nil, ErrCacheKeyExpired
	}

	// Move to front (most recently used)
	m.lru.MoveToFront(elem)

	return entry.value, nil
}

// Set stores a value in the memory cache with a TTL
func (m *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	expiresAt := time.Now().Add(ttl)

	// If key already exists, update it
	if elem, exists := m.items[key]; exists {
		entry := elem.Value.(*cacheEntry)
		entry.value = value
		entry.expiresAt = expiresAt
		m.lru.MoveToFront(elem)
		return nil
	}

	// Create new entry
	entry := &cacheEntry{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
	}

	// Add to front of LRU list
	elem := m.lru.PushFront(entry)
	m.items[key] = elem

	// Evict least recently used if over capacity
	if m.lru.Len() > m.maxSize {
		m.evictOldest()
	}

	return nil
}

// Delete removes a value from the memory cache
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	elem, exists := m.items[key]
	if !exists {
		return nil // Not an error if key doesn't exist
	}

	m.lru.Remove(elem)
	delete(m.items, key)

	return nil
}

// Exists checks if a key exists in the memory cache
func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	elem, exists := m.items[key]
	if !exists {
		return false, nil
	}

	entry := elem.Value.(*cacheEntry)

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return false, nil
	}

	return true, nil
}

// evictOldest removes the least recently used entry
// Must be called with lock held
func (m *MemoryCache) evictOldest() {
	elem := m.lru.Back()
	if elem != nil {
		entry := elem.Value.(*cacheEntry)
		m.lru.Remove(elem)
		delete(m.items, entry.key)
	}
}

// Size returns the current number of items in the cache
func (m *MemoryCache) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lru.Len()
}

// Clear removes all entries from the cache
func (m *MemoryCache) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]*list.Element)
	m.lru = list.New()
}

// Ping checks if the memory cache is available (always returns nil)
func (m *MemoryCache) Ping(ctx context.Context) error {
	return nil
}
