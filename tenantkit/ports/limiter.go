package ports

import (
	"context"
)

// Limiter is a port interface for per-tenant rate limiting.
// Different implementations can use in-memory stores, Redis, or external services.
type Limiter interface {
	// Allow checks if a request should be allowed.
	// Returns true if allowed, false if rate limit exceeded.
	Allow(ctx context.Context, key string) (bool, error)

	// AllowN checks if N requests should be allowed.
	// Returns true if allowed, false if rate limit exceeded.
	AllowN(ctx context.Context, key string, n int64) (bool, error)

	// Remaining returns the number of remaining requests in the current window.
	Remaining(ctx context.Context, key string) (int64, error)

	// Reset resets the rate limit counter for a key.
	Reset(ctx context.Context, key string) error

	// Health checks if the limiter is available.
	Health(ctx context.Context) error
}

// LimiterConfig contains configuration for rate limiters.
type LimiterConfig struct {
	// RequestsPerSecond is the number of requests allowed per second.
	RequestsPerSecond int64

	// BurstSize is the maximum burst size.
	BurstSize int64

	// WindowDuration is the duration of the rate limit window.
	WindowDuration string // "1s", "1m", "1h", etc.
}
