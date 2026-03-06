package limitermemory

import (
	"context"
	"log"
	"os"
	"sync"
	"time"
)

// SlidingWindow implements sliding window rate limiting algorithm.
// Counts requests in a sliding time window. More accurate than fixed window
// but uses more memory.
type SlidingWindow struct {
	mu              sync.Mutex
	limit           int64                      // max requests per window
	windowSize      time.Duration              // time window size
	requests        map[string]*SlidingRequest // key -> request history
	lastCleanupTime time.Time
}

// SlidingRequest holds request history for sliding window.
type SlidingRequest struct {
	timestamps []time.Time // timestamps of recent requests
	lastRead   time.Time   // last time we cleaned old timestamps
}

// NewSlidingWindow creates a new sliding window rate limiter.
// limit: maximum requests per window
// windowSize: duration of the sliding window
// ⚠️ WARNING: Not suitable for multi-instance deployments
func NewSlidingWindow(limit int64, windowSize time.Duration) *SlidingWindow {
	// Log warning in production environments
	env := os.Getenv("ENV")
	goEnv := os.Getenv("GO_ENV")
	if env == "production" || goEnv == "production" {
		log.Println("⚠️  WARNING: Using in-memory rate limiter (SlidingWindow) in production. " +
			"This will cause rate limiting inconsistency in multi-instance deployments. " +
			"Consider using a Redis-based rate limiter for production.")
	}

	if limit < 1 {
		limit = 1
	}
	if windowSize < time.Second {
		windowSize = time.Second
	}

	return &SlidingWindow{
		limit:           limit,
		windowSize:      windowSize,
		requests:        make(map[string]*SlidingRequest),
		lastCleanupTime: time.Now(),
	}
}

// Allow checks if one request is allowed.
func (sw *SlidingWindow) Allow(ctx context.Context, key string) (bool, error) {
	return sw.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed.
func (sw *SlidingWindow) AllowN(ctx context.Context, key string, n int64) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	sw.mu.Lock()
	defer sw.mu.Unlock()

	// Cleanup old entries periodically
	if time.Since(sw.lastCleanupTime) > time.Hour {
		sw.cleanup()
		sw.lastCleanupTime = time.Now()
	}

	now := time.Now()

	// Get or create request history
	req, exists := sw.requests[key]
	if !exists {
		req = &SlidingRequest{
			timestamps: make([]time.Time, 0, sw.limit),
			lastRead:   now,
		}
		sw.requests[key] = req
	}

	// Remove timestamps outside the window
	cutoff := now.Add(-sw.windowSize)
	validTimestamps := make([]time.Time, 0, len(req.timestamps))
	for _, ts := range req.timestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	req.timestamps = validTimestamps
	req.lastRead = now

	// Check if we can allow the request
	if int64(len(req.timestamps)) >= sw.limit {
		// At limit, cannot allow more requests
		return false, nil
	}

	if int64(len(req.timestamps))+n > sw.limit {
		// Would exceed limit with n requests
		return false, nil
	}

	// Add request timestamp(s)
	for i := int64(0); i < n; i++ {
		req.timestamps = append(req.timestamps, now)
	}

	return true, nil
}

// Remaining returns the number of remaining requests in the current window.
func (sw *SlidingWindow) Remaining(ctx context.Context, key string) (int64, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	req, exists := sw.requests[key]
	if !exists {
		return sw.limit, nil
	}

	now := time.Now()
	cutoff := now.Add(-sw.windowSize)

	// Count valid timestamps
	count := int64(0)
	for _, ts := range req.timestamps {
		if ts.After(cutoff) {
			count++
		}
	}

	return sw.limit - count, nil
}

// Reset resets the request history for a key.
func (sw *SlidingWindow) Reset(ctx context.Context, key string) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	delete(sw.requests, key)
	return nil
}

// Health checks if the limiter is operational.
func (sw *SlidingWindow) Health(ctx context.Context) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	// Sliding window is always healthy
	return nil
}

// cleanup removes old entries from the requests map.
func (sw *SlidingWindow) cleanup() {
	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)

	for key, req := range sw.requests {
		if req.lastRead.Before(cutoff) {
			delete(sw.requests, key)
		}
	}
}

// Stats returns statistics about the sliding window limiter.
func (sw *SlidingWindow) Stats() map[string]interface{} {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	return map[string]interface{}{
		"limit":         sw.limit,
		"windowSize":    sw.windowSize,
		"activeWindows": len(sw.requests),
		"totalTimestamps": func() int64 {
			var total int64
			for _, req := range sw.requests {
				total += int64(len(req.timestamps))
			}
			return total
		}(),
	}
}
