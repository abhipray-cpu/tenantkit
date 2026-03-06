package quotaredis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/abhipray-cpu/tenantkit/domain"
	"github.com/abhipray-cpu/tenantkit/tenantkit/ports"
	"github.com/redis/go-redis/v9"
)

// Compile-time interface check
var _ ports.QuotaManager = (*RedisQuotaManager)(nil)

// Config holds Redis quota manager configuration
type Config struct {
	// Prefix for Redis keys
	Prefix string

	// Limits maps quota types to their limits
	Limits map[string]int64

	// ResetTTL is the TTL for quota keys (e.g., 24h for daily quotas)
	ResetTTL time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Prefix:   "quota:",
		Limits:   make(map[string]int64),
		ResetTTL: 24 * time.Hour,
	}
}

// RedisQuotaManager implements ports.QuotaManager with Redis backend.
// This is the recommended adapter for production multi-instance deployments.
// All operations are atomic via Lua scripts, ensuring consistency across
// distributed service instances sharing the same Redis.
//
// It accepts redis.UniversalClient, supporting standalone, Sentinel, and Cluster deployments.
type RedisQuotaManager struct {
	client   redis.UniversalClient
	ownsConn bool // true when client was created internally
	config   Config
}

// NewRedisQuotaManager creates a new Redis quota manager.
// The client parameter accepts redis.UniversalClient, which supports
// standalone Redis, Sentinel, and Cluster deployments.
//
// Example (standalone):
//
//	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	qm, err := quotaredis.NewRedisQuotaManager(client, quotaredis.DefaultConfig())
//
// Example (cluster):
//
//	client := redis.NewClusterClient(&redis.ClusterOptions{
//	    Addrs: []string{"node1:6379", "node2:6379", "node3:6379"},
//	})
//	qm, err := quotaredis.NewRedisQuotaManager(client, quotaredis.DefaultConfig())
func NewRedisQuotaManager(client redis.UniversalClient, config Config) (*RedisQuotaManager, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}

	if config.Prefix == "" {
		config.Prefix = "quota:"
	}

	if config.ResetTTL == 0 {
		config.ResetTTL = 24 * time.Hour
	}

	return &RedisQuotaManager{
		client: client,
		config: config,
	}, nil
}

// Lua script for atomic quota check (read-only, no side effects)
// Returns: 1 if within limits, 0 if would exceed
var checkQuotaScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local amount = tonumber(ARGV[2])

local current = redis.call('GET', key)
if current == false then
    current = 0
else
    current = tonumber(current)
end

if current + amount > limit then
    return 0
end
return 1
`)

// Lua script for atomic quota consumption
// Returns: [allowed (0/1), remaining]
var consumeQuotaScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local amount = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])

-- Get current usage
local current = redis.call('GET', key)
if current == false then
    current = 0
else
    current = tonumber(current)
end

-- Check if consumption would exceed limit
if current + amount > limit then
    return {0, limit - current}  -- denied, remaining
end

-- Atomic increment
local new_usage = redis.call('INCRBY', key, amount)

-- Set TTL on first usage
if new_usage == amount then
    redis.call('EXPIRE', key, ttl)
end

return {1, limit - new_usage}  -- allowed, remaining
`)

// Lua script for atomic limit update
// Updates the limit key and returns the new limit
var setLimitScript = redis.NewScript(`
local limit_key = KEYS[1]
local new_limit = tonumber(ARGV[1])

redis.call('SET', limit_key, new_limit)
return new_limit
`)

// CheckQuota checks if the current usage is within quota limits.
// This is a read-only check that does NOT consume quota.
// Safe to call from multiple instances concurrently.
func (qm *RedisQuotaManager) CheckQuota(ctx context.Context, quotaType string, amount int64) (bool, error) {
	if amount < 0 {
		return false, fmt.Errorf("amount must be non-negative")
	}

	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		return false, err
	}

	limit, err := qm.resolveLimit(ctx, tenantID, quotaType)
	if err != nil {
		return false, err
	}

	key := qm.buildUsageKey(tenantID, quotaType)

	result, err := checkQuotaScript.Run(ctx, qm.client,
		[]string{key},
		limit, amount,
	).Int64()

	if err != nil {
		return false, fmt.Errorf("redis error: %w", err)
	}

	return result == 1, nil
}

// ConsumeQuota atomically consumes quota
func (qm *RedisQuotaManager) ConsumeQuota(ctx context.Context, quotaType string, amount int64) (int64, error) {
	if amount < 0 {
		return 0, fmt.Errorf("amount must be non-negative")
	}

	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		return 0, err
	}

	limit, err := qm.resolveLimit(ctx, tenantID, quotaType)
	if err != nil {
		return 0, err
	}

	key := qm.buildUsageKey(tenantID, quotaType)
	ttlSeconds := int64(qm.config.ResetTTL.Seconds())

	result, err := consumeQuotaScript.Run(ctx, qm.client,
		[]string{key},
		limit, amount, ttlSeconds,
	).Slice()

	if err != nil {
		return 0, fmt.Errorf("redis error: %w", err)
	}

	allowed := result[0].(int64)
	remaining := result[1].(int64)

	if allowed == 0 {
		return remaining, fmt.Errorf("quota exceeded for %s: limit %d", quotaType, limit)
	}

	return remaining, nil
}

// GetUsage returns current usage and limit
func (qm *RedisQuotaManager) GetUsage(ctx context.Context, quotaType string) (int64, int64, error) {
	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		return 0, 0, err
	}

	limit, err := qm.resolveLimit(ctx, tenantID, quotaType)
	if err != nil {
		return 0, 0, err
	}

	key := qm.buildUsageKey(tenantID, quotaType)

	// Get current usage from Redis
	result, err := qm.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key doesn't exist - no usage yet
		return 0, limit, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("redis error: %w", err)
	}

	used, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse usage: %w", err)
	}

	return used, limit, nil
}

// ResetQuota resets quota for a tenant/type
func (qm *RedisQuotaManager) ResetQuota(ctx context.Context, quotaType string) error {
	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	key := qm.buildUsageKey(tenantID, quotaType)
	return qm.client.Del(ctx, key).Err()
}

// SetLimit sets or updates the quota limit for a specific tenant and quota type.
// The limit is stored in Redis so it is shared across all service instances.
func (qm *RedisQuotaManager) SetLimit(ctx context.Context, quotaType string, limit int64) error {
	if limit < 0 {
		return fmt.Errorf("limit must be non-negative")
	}

	tenantID, err := getTenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	limitKey := qm.buildLimitKey(tenantID, quotaType)

	_, err = setLimitScript.Run(ctx, qm.client,
		[]string{limitKey},
		limit,
	).Result()

	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}

	return nil
}

// resolveLimit resolves the effective limit for a tenant+quotaType.
// Priority: per-tenant Redis limit > config default limit > error
func (qm *RedisQuotaManager) resolveLimit(ctx context.Context, tenantID, quotaType string) (int64, error) {
	// 1. Check for per-tenant override in Redis
	limitKey := qm.buildLimitKey(tenantID, quotaType)
	result, err := qm.client.Get(ctx, limitKey).Result()
	if err == nil {
		limit, parseErr := strconv.ParseInt(result, 10, 64)
		if parseErr == nil {
			return limit, nil
		}
	}

	// 2. Fall back to config default
	if limit, ok := qm.config.Limits[quotaType]; ok {
		return limit, nil
	}

	return 0, fmt.Errorf("unknown quota type: %s", quotaType)
}

// Health checks Redis connectivity
func (qm *RedisQuotaManager) Health(ctx context.Context) error {
	return qm.client.Ping(ctx).Err()
}

// Close releases resources
func (qm *RedisQuotaManager) Close() error {
	return qm.client.Close()
}

// buildUsageKey constructs the Redis key for tenant quota usage
func (qm *RedisQuotaManager) buildUsageKey(tenantID, quotaType string) string {
	return fmt.Sprintf("%s%s:%s", qm.config.Prefix, tenantID, quotaType)
}

// buildLimitKey constructs the Redis key for tenant quota limits
func (qm *RedisQuotaManager) buildLimitKey(tenantID, quotaType string) string {
	return fmt.Sprintf("%slimit:%s:%s", qm.config.Prefix, tenantID, quotaType)
}

// getTenantIDFromContext extracts tenant ID from context.
// Supports both the domain context (set by HTTP middleware adapters)
// and the simple string key (set by tenantkit.WithTenant).
func getTenantIDFromContext(ctx context.Context) (string, error) {
	// Try domain context first (set by HTTP middleware adapters)
	tc, err := domain.FromGoContext(ctx)
	if err == nil {
		id := tc.TenantID().Value()
		if id != "" {
			return id, nil
		}
	}

	return "", fmt.Errorf("tenant context required")
}
