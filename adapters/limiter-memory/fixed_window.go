package limitermemory

import (
	"context"
	"log"
	"os"
	"sync"
	"time"
)

// FixedWindow implements fixed window rate limiting algorithm.
// Divides time into fixed windows and counts requests per window.
// Simplest implementation, fastest execution.
type FixedWindow struct {
	mu              sync.Mutex
	limit           int64                  // max requests per window
	windowSize      time.Duration          // duration of each window
	windows         map[string]*WindowData // key -> window data
	lastCleanupTime time.Time
}

// WindowData holds state for a fixed window.
type WindowData struct {
	count      int64
	windowEnd  time.Time
	lastAccess time.Time
}

// NewFixedWindow creates a new fixed window rate limiter.
// limit: maximum requests per window
// windowSize: duration of each window
// ⚠️ WARNING: Not suitable for multi-instance deployments
func NewFixedWindow(limit int64, windowSize time.Duration) *FixedWindow {
	// Log warning in production environments
	env := os.Getenv("ENV")
	goEnv := os.Getenv("GO_ENV")
	if env == "production" || goEnv == "production" {
		log.Println("⚠️  WARNING: Using in-memory rate limiter (FixedWindow) in production. " +
			"This will cause rate limiting inconsistency in multi-instance deployments. " +
			"Consider using a Redis-based rate limiter for production.")
	}

	if limit < 1 {
		limit = 1
	}
	if windowSize < time.Second {
		windowSize = time.Second
	}

	return &FixedWindow{
		limit:           limit,
		windowSize:      windowSize,
		windows:         make(map[string]*WindowData),
		lastCleanupTime: time.Now(),
	}
}

// Allow checks if one request is allowed.
func (fw *FixedWindow) Allow(ctx context.Context, key string) (bool, error) {
	return fw.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed.
func (fw *FixedWindow) AllowN(ctx context.Context, key string, n int64) (bool, error) {
	if n <= 0 {
		return true, nil
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Cleanup old entries periodically
	if time.Since(fw.lastCleanupTime) > time.Hour {
		fw.cleanup()
		fw.lastCleanupTime = time.Now()
	}

	now := time.Now()

	// Get or create window
	wd, exists := fw.windows[key]
	if !exists {
		wd = &WindowData{
			count:      0,
			windowEnd:  now.Add(fw.windowSize),
			lastAccess: now,
		}
		fw.windows[key] = wd
	}

	// Check if window has expired
	if now.After(wd.windowEnd) {
		// Start new window
		wd.count = 0
		wd.windowEnd = now.Add(fw.windowSize)
	}

	// Update last access time
	wd.lastAccess = now

	// Check if we can allow the request
	if wd.count >= fw.limit {
		return false, nil
	}

	if wd.count+n > fw.limit {
		return false, nil
	}

	// Increment count
	wd.count += n
	return true, nil
}

// Remaining returns the number of remaining requests in the current window.
func (fw *FixedWindow) Remaining(ctx context.Context, key string) (int64, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	wd, exists := fw.windows[key]
	if !exists {
		return fw.limit, nil
	}

	now := time.Now()

	// Check if window has expired
	if now.After(wd.windowEnd) {
		return fw.limit, nil
	}

	return fw.limit - wd.count, nil
}

// Reset resets the window for a key.
func (fw *FixedWindow) Reset(ctx context.Context, key string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	delete(fw.windows, key)
	return nil
}

// Health checks if the limiter is operational.
func (fw *FixedWindow) Health(ctx context.Context) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	// Fixed window is always healthy
	return nil
}

// cleanup removes old entries from the windows map.
func (fw *FixedWindow) cleanup() {
	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)

	for key, wd := range fw.windows {
		if wd.lastAccess.Before(cutoff) {
			delete(fw.windows, key)
		}
	}
}

// Stats returns statistics about the fixed window limiter.
func (fw *FixedWindow) Stats() map[string]interface{} {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	return map[string]interface{}{
		"limit":         fw.limit,
		"windowSize":    fw.windowSize,
		"activeWindows": len(fw.windows),
	}
}
