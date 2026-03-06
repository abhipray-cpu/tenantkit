package limitermemory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/abhipray-cpu/tenantkit/tenantkit/ports"
)

// Algorithm specifies which rate limiting algorithm to use
type Algorithm string

const (
	// AlgorithmTokenBucket uses token bucket algorithm
	AlgorithmTokenBucket Algorithm = "token_bucket"
	// AlgorithmSlidingWindow uses sliding window algorithm
	AlgorithmSlidingWindow Algorithm = "sliding_window"
	// AlgorithmFixedWindow uses fixed window algorithm
	AlgorithmFixedWindow Algorithm = "fixed_window"
)

// LimiterMemory is an in-memory rate limiter adapter.
// It supports three algorithms: token bucket, sliding window, and fixed window.
type LimiterMemory struct {
	limiter Limiter
	algo    Algorithm
}

// Limiter is the interface for rate limiting implementations
type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	AllowN(ctx context.Context, key string, n int64) (bool, error)
	Remaining(ctx context.Context, key string) (int64, error)
	Reset(ctx context.Context, key string) error
	Health(ctx context.Context) error
	Stats() map[string]interface{}
}

// Config holds configuration for the in-memory rate limiter
type Config struct {
	// Algorithm specifies which algorithm to use
	Algorithm Algorithm
	// RequestsPerSecond (for token bucket)
	RequestsPerSecond float64
	// BurstSize (for token bucket)
	BurstSize int64
	// Limit (for sliding window and fixed window)
	Limit int64
	// WindowSize (for sliding window and fixed window)
	WindowSize time.Duration
}

// NewLimiterMemory creates a new in-memory rate limiter.
func NewLimiterMemory(config Config) (*LimiterMemory, error) {
	if config.Algorithm == "" {
		config.Algorithm = AlgorithmTokenBucket
	}

	var limiter Limiter

	switch config.Algorithm {
	case AlgorithmTokenBucket:
		rps := config.RequestsPerSecond
		if rps < 0 {
			return nil, fmt.Errorf("RequestsPerSecond must be non-negative, got %v", rps)
		}
		if rps == 0 {
			rps = 10.0 // default: 10 RPS
		}
		burst := config.BurstSize
		if burst <= 0 {
			burst = int64(rps * 2) // default: 2x RPS
		}
		limiter = NewTokenBucket(rps, burst)

	case AlgorithmSlidingWindow:
		limit := config.Limit
		if limit <= 0 {
			limit = 10
		}
		windowSize := config.WindowSize
		if windowSize <= 0 {
			windowSize = time.Second
		}
		limiter = NewSlidingWindow(limit, windowSize)

	case AlgorithmFixedWindow:
		limit := config.Limit
		if limit <= 0 {
			limit = 10
		}
		windowSize := config.WindowSize
		if windowSize <= 0 {
			windowSize = time.Second
		}
		limiter = NewFixedWindow(limit, windowSize)

	default:
		return nil, fmt.Errorf("unknown algorithm: %v", config.Algorithm)
	}

	return &LimiterMemory{
		limiter: limiter,
		algo:    config.Algorithm,
	}, nil
}

// NewTokenBucketLimiter creates a token bucket rate limiter.
func NewTokenBucketLimiter(rps float64, burstSize int64) (*LimiterMemory, error) {
	return NewLimiterMemory(Config{
		Algorithm:         AlgorithmTokenBucket,
		RequestsPerSecond: rps,
		BurstSize:         burstSize,
	})
}

// NewSlidingWindowLimiter creates a sliding window rate limiter.
func NewSlidingWindowLimiter(limit int64, windowSize time.Duration) (*LimiterMemory, error) {
	return NewLimiterMemory(Config{
		Algorithm:  AlgorithmSlidingWindow,
		Limit:      limit,
		WindowSize: windowSize,
	})
}

// NewFixedWindowLimiter creates a fixed window rate limiter.
func NewFixedWindowLimiter(limit int64, windowSize time.Duration) (*LimiterMemory, error) {
	return NewLimiterMemory(Config{
		Algorithm:  AlgorithmFixedWindow,
		Limit:      limit,
		WindowSize: windowSize,
	})
}

// Allow checks if one request is allowed (implements ports.Limiter).
func (l *LimiterMemory) Allow(ctx context.Context, key string) (bool, error) {
	// FIX BUG #20, #21: Validate tenant ID (key) is not empty or whitespace
	if err := validateTenantKey(key); err != nil {
		return false, err
	}
	return l.limiter.Allow(ctx, key)
}

// AllowN checks if n requests are allowed (implements ports.Limiter).
func (l *LimiterMemory) AllowN(ctx context.Context, key string, n int64) (bool, error) {
	// FIX BUG #20, #21: Validate tenant ID (key) is not empty or whitespace
	if err := validateTenantKey(key); err != nil {
		return false, err
	}
	return l.limiter.AllowN(ctx, key, n)
}

// Remaining returns remaining requests (implements ports.Limiter).
func (l *LimiterMemory) Remaining(ctx context.Context, key string) (int64, error) {
	return l.limiter.Remaining(ctx, key)
}

// Reset resets the limiter for a key (implements ports.Limiter).
func (l *LimiterMemory) Reset(ctx context.Context, key string) error {
	return l.limiter.Reset(ctx, key)
}

// Health checks if the limiter is healthy (implements ports.Limiter).
func (l *LimiterMemory) Health(ctx context.Context) error {
	return l.limiter.Health(ctx)
}

// Stats returns statistics about the limiter.
func (l *LimiterMemory) Stats() map[string]interface{} {
	stats := l.limiter.Stats()
	stats["algorithm"] = string(l.algo)
	return stats
}

// validateTenantKey ensures tenant ID (key) is not empty or whitespace only
func validateTenantKey(key string) error {
	if key == "" {
		return fmt.Errorf("tenant ID cannot be empty")
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("tenant ID cannot be whitespace only")
	}
	return nil
}

// Verify that LimiterMemory implements ports.Limiter
var _ ports.Limiter = (*LimiterMemory)(nil)
