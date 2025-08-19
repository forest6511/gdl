package backends

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/forest6511/gdl/pkg/storage"
)

// RedisBackend implements storage using Redis for metadata and small objects
// Note: Redis is better suited for metadata storage rather than large binary objects
type RedisBackend struct {
	client *redis.Client
	prefix string
}

// NewRedisBackend creates a new Redis storage backend
func NewRedisBackend() *RedisBackend {
	return &RedisBackend{}
}

// Init initializes the Redis backend with configuration
func (r *RedisBackend) Init(config map[string]interface{}) error {
	// Get Redis connection options
	addr, _ := config["addr"].(string)
	if addr == "" {
		addr = "localhost:6379"
	}

	password, _ := config["password"].(string)

	dbNum := 0
	if db, ok := config["db"].(float64); ok {
		dbNum = int(db)
	} else if db, ok := config["db"].(int); ok {
		dbNum = db
	}

	// Optional prefix for all keys
	if prefix, ok := config["prefix"].(string); ok {
		r.prefix = strings.TrimSuffix(prefix, ":")
	}

	// Create Redis client
	r.client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       dbNum,
	})

	// Test connection
	ctx := context.Background()
	_, err := r.client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return nil
}

// Save stores data to Redis at the specified key
func (r *RedisBackend) Save(ctx context.Context, key string, data io.Reader) error {
	fullKey := r.buildKey(key)

	// Read all data into memory
	// Note: Redis is not ideal for large objects
	dataBytes, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	// Store in Redis
	err = r.client.Set(ctx, fullKey, dataBytes, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to save data to Redis key %s: %w", fullKey, err)
	}

	return nil
}

// Load retrieves data from Redis for the given key
func (r *RedisBackend) Load(ctx context.Context, key string) (io.ReadCloser, error) {
	fullKey := r.buildKey(key)

	// Get data from Redis
	result, err := r.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, storage.ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get data from Redis key %s: %w", fullKey, err)
	}

	// Return as ReadCloser
	return io.NopCloser(strings.NewReader(result)), nil
}

// Delete removes data from Redis for the given key
func (r *RedisBackend) Delete(ctx context.Context, key string) error {
	fullKey := r.buildKey(key)

	// Check if key exists first
	exists, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check key existence in Redis: %w", err)
	}
	if exists == 0 {
		return storage.ErrKeyNotFound
	}

	// Delete from Redis
	err = r.client.Del(ctx, fullKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete data from Redis key %s: %w", fullKey, err)
	}

	return nil
}

// Exists checks if data exists at the given key in Redis
func (r *RedisBackend) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.buildKey(key)

	exists, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence in Redis key %s: %w", fullKey, err)
	}

	return exists > 0, nil
}

// List returns a list of keys with the given prefix
func (r *RedisBackend) List(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := r.buildKey(prefix)
	pattern := fullPrefix + "*"

	// Use SCAN to get keys matching pattern
	var keys []string
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		redisKey := iter.Val()
		// Strip the prefix to get the original key
		key := r.stripPrefix(redisKey)
		keys = append(keys, key)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan Redis keys with pattern %s: %w", pattern, err)
	}

	return keys, nil
}

// Close closes the Redis connection
func (r *RedisBackend) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// buildKey constructs the full Redis key including any configured prefix
func (r *RedisBackend) buildKey(key string) string {
	if r.prefix == "" {
		return key
	}
	return r.prefix + ":" + key
}

// stripPrefix removes the configured prefix from a Redis key to get the original key
func (r *RedisBackend) stripPrefix(redisKey string) string {
	if r.prefix == "" {
		return redisKey
	}

	prefixWithColon := r.prefix + ":"
	if strings.HasPrefix(redisKey, prefixWithColon) {
		return strings.TrimPrefix(redisKey, prefixWithColon)
	}

	return redisKey
}

// SetExpiration sets an expiration time for a key
func (r *RedisBackend) SetExpiration(ctx context.Context, key string, seconds int) error {
	fullKey := r.buildKey(key)

	err := r.client.Expire(ctx, fullKey, time.Duration(seconds)*time.Second).Err()
	if err != nil {
		return fmt.Errorf("failed to set expiration for Redis key %s: %w", fullKey, err)
	}

	return nil
}

// GetTTL returns the time to live for a key
func (r *RedisBackend) GetTTL(ctx context.Context, key string) (int64, error) {
	fullKey := r.buildKey(key)

	ttl, err := r.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for Redis key %s: %w", fullKey, err)
	}

	return int64(ttl.Seconds()), nil
}

// SetMetadata stores metadata for a key (using hash)
func (r *RedisBackend) SetMetadata(ctx context.Context, key string, metadata map[string]interface{}) error {
	metaKey := r.buildKey(key + ":meta")

	// Convert metadata to string map for Redis HSET
	fields := make(map[string]interface{})
	for k, v := range metadata {
		fields[k] = fmt.Sprintf("%v", v)
	}

	err := r.client.HSet(ctx, metaKey, fields).Err()
	if err != nil {
		return fmt.Errorf("failed to set metadata for Redis key %s: %w", metaKey, err)
	}

	return nil
}

// GetMetadata retrieves metadata for a key
func (r *RedisBackend) GetMetadata(ctx context.Context, key string) (map[string]interface{}, error) {
	metaKey := r.buildKey(key + ":meta")

	result, err := r.client.HGetAll(ctx, metaKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for Redis key %s: %w", metaKey, err)
	}

	if len(result) == 0 {
		return nil, storage.ErrKeyNotFound
	}

	// Convert string map to interface{} map
	metadata := make(map[string]interface{})
	for k, v := range result {
		metadata[k] = v
	}

	return metadata, nil
}

// Increment atomically increments a counter key
func (r *RedisBackend) Increment(ctx context.Context, key string) (int64, error) {
	fullKey := r.buildKey(key)

	result, err := r.client.Incr(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment Redis key %s: %w", fullKey, err)
	}

	return result, nil
}

// AddToSet adds a member to a set
func (r *RedisBackend) AddToSet(ctx context.Context, key string, member interface{}) error {
	fullKey := r.buildKey(key)

	err := r.client.SAdd(ctx, fullKey, member).Err()
	if err != nil {
		return fmt.Errorf("failed to add to set Redis key %s: %w", fullKey, err)
	}

	return nil
}

// GetSetMembers returns all members of a set
func (r *RedisBackend) GetSetMembers(ctx context.Context, key string) ([]string, error) {
	fullKey := r.buildKey(key)

	members, err := r.client.SMembers(ctx, fullKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get set members for Redis key %s: %w", fullKey, err)
	}

	return members, nil
}
