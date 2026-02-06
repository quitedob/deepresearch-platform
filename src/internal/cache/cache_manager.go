package cache

import (
	"context"
	"errors"
	"time"
)

// CacheManager implements a two-tier cache strategy (L1 memory + L2 Redis)
type CacheManager struct {
	l1Cache Cache
	l2Cache Cache
	config  CacheConfig
}

// NewCacheManager creates a new cache manager with two-tier fallback
func NewCacheManager(l1Cache Cache, l2Cache Cache, config CacheConfig) *CacheManager {
	return &CacheManager{
		l1Cache: l1Cache,
		l2Cache: l2Cache,
		config:  config,
	}
}

// Get retrieves a value from the cache with two-tier fallback
// First checks L1 (memory), then L2 (Redis) if L1 misses
func (c *CacheManager) Get(ctx context.Context, key string) (interface{}, error) {
	// Try L1 cache first
	value, err := c.l1Cache.Get(ctx, key)
	if err == nil {
		return value, nil
	}

	// If L1 miss and not a "not found" error, return the error
	if !errors.Is(err, ErrCacheKeyNotFound) && !errors.Is(err, ErrCacheKeyExpired) {
		return nil, err
	}

	// If L2 is disabled, return the L1 error
	if !c.config.EnableL2 || c.l2Cache == nil {
		return nil, err
	}

	// Try L2 cache
	value, err = c.l2Cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// Populate L1 cache with the value from L2
	_ = c.l1Cache.Set(ctx, key, value, c.config.L1TTL)

	return value, nil
}

// Set stores a value in both cache tiers
func (c *CacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Set in L1 cache
	if err := c.l1Cache.Set(ctx, key, value, c.config.L1TTL); err != nil {
		return err
	}

	// Set in L2 cache if enabled
	if c.config.EnableL2 && c.l2Cache != nil {
		if err := c.l2Cache.Set(ctx, key, value, c.config.L2TTL); err != nil {
			// L2 failure is not critical, log but don't fail
			// In production, this should be logged
			return nil
		}
	}

	return nil
}

// Delete removes a value from both cache tiers
func (c *CacheManager) Delete(ctx context.Context, key string) error {
	// Delete from L1
	if err := c.l1Cache.Delete(ctx, key); err != nil {
		return err
	}

	// Delete from L2 if enabled
	if c.config.EnableL2 && c.l2Cache != nil {
		if err := c.l2Cache.Delete(ctx, key); err != nil {
			// L2 failure is not critical
			return nil
		}
	}

	return nil
}

// Exists checks if a key exists in either cache tier
func (c *CacheManager) Exists(ctx context.Context, key string) (bool, error) {
	// Check L1 first
	exists, err := c.l1Cache.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	// Check L2 if enabled
	if c.config.EnableL2 && c.l2Cache != nil {
		return c.l2Cache.Exists(ctx, key)
	}

	return false, nil
}

// Ping checks if the cache is available
func (c *CacheManager) Ping(ctx context.Context) error {
	// Check L1 cache
	if err := c.l1Cache.Ping(ctx); err != nil {
		return err
	}

	// Check L2 cache if enabled
	if c.config.EnableL2 && c.l2Cache != nil {
		return c.l2Cache.Ping(ctx)
	}

	return nil
}
