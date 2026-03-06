package limiterredis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/abhipray-cpu/tenantkit/tenantkit/ports"
	"github.com/redis/go-redis/v9"
)

// LimiterRedis is a Redis-based distributed rate limiter using Lua scripts for atomic operations.
// It accepts redis.UniversalClient, which supports standalone, Sentinel, and Cluster deployments.
type LimiterRedis struct {
	client    redis.UniversalClient
	ownsConn  bool // true when we created the client (legacy constructor)
	algorithm string
	limit     int64
	window    time.Duration
}

// Config holds configuration for the Redis rate limiter.
//
// Deprecated fields (RedisAddr, Password, DB) are kept for backward compatibility.
// Prefer using New() with your own redis.UniversalClient for production deployments.
type Config struct {
	// RedisAddr is the Redis server address (e.g., "localhost:6379").
	// Deprecated: Use New() with your own redis.UniversalClient instead.
	RedisAddr string
	// Password is the Redis password (optional).
	// Deprecated: Use New() with your own redis.UniversalClient instead.
	Password string
	// DB is the Redis database number (default: 0).
	// Deprecated: Use New() with your own redis.UniversalClient instead.
	DB int
	// Algorithm is the rate limiting algorithm to use ("token_bucket", "fixed_window", "sliding_window")
	Algorithm string
	// Limit is the maximum requests per window
	Limit int64
	// Window is the time window duration
	Window time.Duration
}

// Lua script for token bucket algorithm (atomic operation)
var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local now = tonumber(ARGV[2])
local window = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local bucket = redis.call('HMGET', key, 'tokens', 'last_update')
local tokens = tonumber(bucket[1])
local last_update = tonumber(bucket[2])

if tokens == nil then
    tokens = limit
    last_update = now
end

local elapsed = now - last_update
local refill = math.floor(elapsed * limit / window)
tokens = math.min(limit, tokens + refill)

if tokens >= requested then
    tokens = tokens - requested
    redis.call('HMSET', key, 'tokens', tokens, 'last_update', now)
    redis.call('EXPIRE', key, window)
    return {1, tokens}
else
    return {0, tokens}
end
`)

// Lua script for fixed window algorithm
var fixedWindowScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local requested = tonumber(ARGV[3])

local current = redis.call('GET', key)
if current == false then
    current = 0
else
    current = tonumber(current)
end

if current + requested <= limit then
    local new_count = redis.call('INCRBY', key, requested)
    if new_count == requested then
        redis.call('EXPIRE', key, window)
    end
    return {1, limit - new_count}
else
    return {0, limit - current}
end
`)

// Lua script for sliding window log algorithm
var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local cutoff = now - window * 1000

redis.call('ZREMRANGEBYSCORE', key, 0, cutoff)

local current = redis.call('ZCARD', key)

if current + requested <= limit then
    for i = 1, requested do
        redis.call('ZADD', key, now, now .. ':' .. i)
    end
    redis.call('EXPIRE', key, window * 2)
    -- FIX: Get count AFTER adding entries, not before
    local new_count = redis.call('ZCARD', key)
    return {1, limit - new_count}
else
    return {0, limit - current}
end
`)

// New creates a new Redis-based rate limiter with the given client.
// This is the recommended constructor for production use. It accepts
// redis.UniversalClient, supporting standalone Redis, Sentinel, and Cluster.
//
// Example (standalone):
//
//	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	limiter, err := limiterredis.New(client, limiterredis.Config{
//	    Algorithm: "token_bucket",
//	    Limit:     100,
//	    Window:    time.Minute,
//	})
//
// Example (cluster):
//
//	client := redis.NewClusterClient(&redis.ClusterOptions{
//	    Addrs: []string{"node1:6379", "node2:6379", "node3:6379"},
//	})
//	limiter, err := limiterredis.New(client, limiterredis.Config{...})
//
// Example (sentinel):
//
//	client := redis.NewFailoverClient(&redis.FailoverOptions{
//	    MasterName:    "mymaster",
//	    SentinelAddrs: []string{"sentinel1:26379", "sentinel2:26379"},
//	})
//	limiter, err := limiterredis.New(client, limiterredis.Config{...})
func New(client redis.UniversalClient, config Config) (*LimiterRedis, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}

	if config.Limit <= 0 {
		return nil, fmt.Errorf("limit must be positive")
	}

	if config.Window <= 0 {
		return nil, fmt.Errorf("window duration must be positive")
	}

	algorithm := config.Algorithm
	if algorithm == "" {
		algorithm = "token_bucket"
	}

	switch algorithm {
	case "token_bucket", "fixed_window", "sliding_window":
		// Valid
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s (use token_bucket, fixed_window, or sliding_window)", algorithm)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &LimiterRedis{
		client:    client,
		ownsConn:  false,
		algorithm: algorithm,
		limit:     config.Limit,
		window:    config.Window,
	}, nil
}

// NewLimiterRedis creates a new Redis-based rate limiter with a new Redis connection.
//
// Deprecated: Use New() with your own redis.UniversalClient for production use.
// This constructor creates a standalone redis.Client internally, which doesn't
// support Sentinel or Cluster deployments. It is kept for backward compatibility.
func NewLimiterRedis(config Config) (*LimiterRedis, error) {
	if config.RedisAddr == "" {
		return nil, fmt.Errorf("redis address is required")
	}

	if config.Limit <= 0 {
		return nil, fmt.Errorf("limit must be positive")
	}

	if config.Window <= 0 {
		return nil, fmt.Errorf("window duration must be positive")
	}

	algorithm := config.Algorithm
	if algorithm == "" {
		algorithm = "token_bucket"
	}

	// Validate algorithm
	switch algorithm {
	case "token_bucket", "fixed_window", "sliding_window":
		// Valid
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s (use token_bucket, fixed_window, or sliding_window)", algorithm)
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", config.RedisAddr, err)
	}

	return &LimiterRedis{
		client:    client,
		ownsConn:  true,
		algorithm: algorithm,
		limit:     config.Limit,
		window:    config.Window,
	}, nil
}

// Allow checks if one request is allowed (implements ports.Limiter).
func (lr *LimiterRedis) Allow(ctx context.Context, key string) (bool, error) {
	return lr.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed (implements ports.Limiter).
func (lr *LimiterRedis) AllowN(ctx context.Context, key string, n int64) (bool, error) {
	if n <= 0 {
		return false, fmt.Errorf("n must be positive")
	}

	var result []interface{}
	var err error

	switch lr.algorithm {
	case "token_bucket":
		now := time.Now().Unix()
		windowSeconds := int64(lr.window.Seconds())
		result, err = tokenBucketScript.Run(ctx, lr.client, []string{key}, lr.limit, now, windowSeconds, n).Slice()

	case "fixed_window":
		windowSeconds := int64(lr.window.Seconds())
		result, err = fixedWindowScript.Run(ctx, lr.client, []string{key}, lr.limit, windowSeconds, n).Slice()

	case "sliding_window":
		now := time.Now().UnixMilli()
		windowSeconds := int64(lr.window.Seconds())
		result, err = slidingWindowScript.Run(ctx, lr.client, []string{key}, lr.limit, windowSeconds, now, n).Slice()

	default:
		return false, fmt.Errorf("unknown algorithm: %s", lr.algorithm)
	}

	if err != nil {
		return false, fmt.Errorf("redis rate limit check failed: %w", err)
	}

	if len(result) < 1 {
		return false, fmt.Errorf("unexpected redis response")
	}

	allowed, ok := result[0].(int64)
	if !ok {
		return false, fmt.Errorf("unexpected redis response type")
	}

	return allowed == 1, nil
}

// Remaining returns the number of remaining requests (implements ports.Limiter).
func (lr *LimiterRedis) Remaining(ctx context.Context, key string) (int64, error) {
	switch lr.algorithm {
	case "token_bucket":
		result, err := lr.client.HMGet(ctx, key, "tokens").Result()
		if err != nil {
			if err == redis.Nil {
				return lr.limit, nil
			}
			return 0, fmt.Errorf("failed to get remaining tokens: %w", err)
		}
		if len(result) == 0 || result[0] == nil {
			return lr.limit, nil
		}
		tokensStr, ok := result[0].(string)
		if !ok {
			return lr.limit, nil
		}
		tokens, err := strconv.ParseInt(tokensStr, 10, 64)
		if err != nil {
			return lr.limit, nil
		}
		return tokens, nil

	case "fixed_window":
		current, err := lr.client.Get(ctx, key).Int64()
		if err != nil {
			if err == redis.Nil {
				return lr.limit, nil
			}
			return 0, fmt.Errorf("failed to get current count: %w", err)
		}
		remaining := lr.limit - current
		if remaining < 0 {
			remaining = 0
		}
		return remaining, nil

	case "sliding_window":
		now := time.Now().UnixMilli()
		cutoff := now - int64(lr.window.Seconds())*1000

		// Remove old entries
		if err := lr.client.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(cutoff, 10)).Err(); err != nil {
			return 0, fmt.Errorf("failed to remove old entries: %w", err)
		}

		current, err := lr.client.ZCard(ctx, key).Result()
		if err != nil {
			return 0, fmt.Errorf("failed to get current count: %w", err)
		}

		remaining := lr.limit - current
		if remaining < 0 {
			remaining = 0
		}
		return remaining, nil

	default:
		return 0, fmt.Errorf("unknown algorithm: %s", lr.algorithm)
	}
}

// Reset resets the limiter for a key (implements ports.Limiter).
func (lr *LimiterRedis) Reset(ctx context.Context, key string) error {
	if err := lr.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to reset key: %w", err)
	}
	return nil
}

// Health checks if Redis is accessible (implements ports.Limiter).
func (lr *LimiterRedis) Health(ctx context.Context) error {
	if err := lr.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}
	return nil
}

// Stats returns statistics about the limiter.
func (lr *LimiterRedis) Stats() map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	info := lr.client.Info(ctx, "stats").Val()

	return map[string]interface{}{
		"backend":   "redis",
		"algorithm": lr.algorithm,
		"limit":     lr.limit,
		"window":    lr.window.String(),
		"healthy":   lr.Health(ctx) == nil,
		"info":      info,
	}
}

// Close closes the Redis connection if it was created by this limiter.
// If the client was provided externally via New(), Close is a no-op
// (the caller is responsible for closing their own client).
func (lr *LimiterRedis) Close() error {
	if lr.ownsConn {
		return lr.client.Close()
	}
	return nil
}

// Verify that LimiterRedis implements ports.Limiter
var _ ports.Limiter = (*LimiterRedis)(nil)
