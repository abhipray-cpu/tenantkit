package limitermemory

import (
	"context"
	"log"
	"os"
	"sync"
	"time"
)

// TokenBucket implements token bucket rate limiting algorithm.
// Tokens are added at a constant rate (rps), up to a maximum burst size.
// Each request consumes 1 token. Requests are allowed if tokens are available.
type TokenBucket struct {
	mu              sync.Mutex
	rps             float64              // requests per second
	burstSize       int64                // maximum tokens available
	tokens          map[string]TokenData // key -> token data
	windowDuration  time.Duration
	lastCleanupTime time.Time
}

// TokenData holds token bucket state for a key.
type TokenData struct {
	tokens    float64   // current tokens available
	lastRefil time.Time // last time tokens were refilled
}

// NewTokenBucket creates a new token bucket rate limiter.
// rps: requests per second
// burstSize: maximum tokens (can be less than rps for burst limiting)
// ⚠️ WARNING: Not suitable for multi-instance deployments
func NewTokenBucket(rps float64, burstSize int64) *TokenBucket {
	// Log warning in production environments
	env := os.Getenv("ENV")
	goEnv := os.Getenv("GO_ENV")
	if env == "production" || goEnv == "production" {
		log.Println("⚠️  WARNING: Using in-memory rate limiter in production. " +
			"This will cause rate limiting inconsistency in multi-instance deployments. " +
			"Consider using a Redis-based rate limiter for production.")
	}

	if rps <= 0 {
		rps = 1.0
	}
	if burstSize < 1 {
		burstSize = int64(rps)
	}

	return &TokenBucket{
		rps:             rps,
		burstSize:       burstSize,
		tokens:          make(map[string]TokenData),
		windowDuration:  time.Hour, // cleanup window
		lastCleanupTime: time.Now(),
	}
}

// Allow checks if one token is available.
func (tb *TokenBucket) Allow(ctx context.Context, key string) (bool, error) {
	return tb.AllowN(ctx, key, 1)
}

// AllowN checks if n tokens are available.
func (tb *TokenBucket) AllowN(ctx context.Context, key string, n int64) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Cleanup old entries periodically
	if time.Since(tb.lastCleanupTime) > tb.windowDuration {
		tb.cleanup()
		tb.lastCleanupTime = time.Now()
	}

	// Get current token data
	data, exists := tb.tokens[key]
	if !exists {
		data = TokenData{
			tokens:    float64(tb.burstSize),
			lastRefil: time.Now(),
		}
	}

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(data.lastRefil).Seconds()
	data.tokens += elapsed * tb.rps

	// Cap tokens at burst size
	if data.tokens > float64(tb.burstSize) {
		data.tokens = float64(tb.burstSize)
	}

	// Check if we have enough tokens
	if data.tokens >= float64(n) {
		data.tokens -= float64(n)
		data.lastRefil = now
		tb.tokens[key] = data
		return true, nil
	}

	// Not enough tokens, update last refill time
	data.lastRefil = now
	tb.tokens[key] = data
	return false, nil
}

// Remaining returns the number of remaining tokens.
func (tb *TokenBucket) Remaining(ctx context.Context, key string) (int64, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	data, exists := tb.tokens[key]
	if !exists {
		return tb.burstSize, nil
	}

	// Recalculate tokens
	now := time.Now()
	elapsed := now.Sub(data.lastRefil).Seconds()
	tokens := data.tokens + elapsed*tb.rps

	if tokens > float64(tb.burstSize) {
		tokens = float64(tb.burstSize)
	}

	return int64(tokens), nil
}

// Reset resets the token count for a key.
func (tb *TokenBucket) Reset(ctx context.Context, key string) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	delete(tb.tokens, key)
	return nil
}

// Health checks if the limiter is operational.
func (tb *TokenBucket) Health(ctx context.Context) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	// Token bucket is always healthy
	return nil
}

// cleanup removes old entries from the tokens map.
// Entries are considered old if they haven't been used recently.
func (tb *TokenBucket) cleanup() {
	now := time.Now()
	cutoffTime := now.Add(-1 * time.Hour)

	for key, data := range tb.tokens {
		if data.lastRefil.Before(cutoffTime) {
			delete(tb.tokens, key)
		}
	}
}

// Stats returns statistics about the token bucket limiter.
func (tb *TokenBucket) Stats() map[string]interface{} {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	return map[string]interface{}{
		"rps":            tb.rps,
		"burstSize":      tb.burstSize,
		"activeBuckets":  len(tb.tokens),
		"windowDuration": tb.windowDuration,
	}
}
