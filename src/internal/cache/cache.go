package cache

import (
	"context"
	"time"
)

// Cache defines the interface for cache operations
type Cache interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string) (interface{}, error)
	
	// Set stores a value in the cache with a TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	
	// Delete removes a value from the cache
	Delete(ctx context.Context, key string) error
	
	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)
	
	// Ping checks if the cache is available
	Ping(ctx context.Context) error
}

// CacheConfig holds configuration for the cache manager
type CacheConfig struct {
	L1TTL     time.Duration
	L2TTL     time.Duration
	L1MaxSize int
	EnableL2  bool
}

// DefaultCacheConfig returns a default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		L1TTL:     5 * time.Minute,
		L2TTL:     30 * time.Minute,
		L1MaxSize: 10000,
		EnableL2:  true,
	}
}
