package httpstd

import (
	"time"
)

// RateLimitOptions holds configurable options for rate limiting behavior
type RateLimitOptions struct {
	// LimitPerWindow is the maximum number of requests allowed per window
	// This is what gets returned in the X-RateLimit-Limit header
	LimitPerWindow int64
	// ResetWindow is the duration after which the rate limit counter resets
	// Default: 1 minute
	ResetWindow time.Duration
}

// DefaultRateLimitOptions returns sensible defaults for rate limiting
func DefaultRateLimitOptions() *RateLimitOptions {
	return &RateLimitOptions{
		LimitPerWindow: 100,
		ResetWindow:    1 * time.Minute,
	}
}

// StrictRateLimitOptions returns conservative rate limiting options
// Useful for high-security scenarios or protecting against abuse
func StrictRateLimitOptions() *RateLimitOptions {
	return &RateLimitOptions{
		LimitPerWindow: 30,
		ResetWindow:    1 * time.Minute,
	}
}

// GenerousRateLimitOptions returns permissive rate limiting options
// Useful for internal APIs or trusted clients
func GenerousRateLimitOptions() *RateLimitOptions {
	return &RateLimitOptions{
		LimitPerWindow: 500,
		ResetWindow:    1 * time.Minute,
	}
}

// PerSecondRateLimitOptions returns options for per-second rate limiting
// Much shorter reset window for fine-grained control
func PerSecondRateLimitOptions() *RateLimitOptions {
	return &RateLimitOptions{
		LimitPerWindow: 10,
		ResetWindow:    1 * time.Second,
	}
}

// CustomRateLimitOptions creates options with custom limit and reset window
func CustomRateLimitOptions(limitPerWindow int64, resetWindow time.Duration) *RateLimitOptions {
	if limitPerWindow <= 0 {
		limitPerWindow = 100
	}
	if resetWindow <= 0 {
		resetWindow = 1 * time.Minute
	}
	return &RateLimitOptions{
		LimitPerWindow: limitPerWindow,
		ResetWindow:    resetWindow,
	}
}
