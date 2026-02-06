package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCache implements a Redis-based cache
type RedisCache struct {
	client *redis.Client
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// NewRedisCache creates a new Redis cache with connection pooling
func NewRedisCache(config RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         config.Host + ":" + config.Port,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{
		client: client,
	}, nil
}

// Get retrieves a value from Redis cache
func (r *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, ErrCacheKeyNotFound
	}
	if err != nil {
		return nil, err
	}

	// Deserialize the value
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		// If deserialization fails, return the raw string
		return val, nil
	}

	return result, nil
}

// Set stores a value in Redis cache with a TTL
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Serialize the value
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

// Delete removes a value from Redis cache
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists checks if a key exists in Redis cache
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Ping checks if the Redis cache is available
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}
