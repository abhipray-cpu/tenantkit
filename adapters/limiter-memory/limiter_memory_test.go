package limitermemory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestTokenBucketBasic tests basic token bucket functionality
func TestTokenBucketBasic(t *testing.T) {
	tests := []struct {
		name      string
		rps       float64
		burst     int64
		requests  int64
		wantAllow bool
	}{
		{
			name:      "single request allowed",
			rps:       10,
			burst:     10,
			requests:  1,
			wantAllow: true,
		},
		{
			name:      "multiple requests allowed",
			rps:       10,
			burst:     10,
			requests:  5,
			wantAllow: true,
		},
		{
			name:      "burst allowed",
			rps:       10,
			burst:     10,
			requests:  10,
			wantAllow: true,
		},
		{
			name:      "burst exceeded",
			rps:       10,
			burst:     10,
			requests:  11,
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTokenBucket(tt.rps, tt.burst)
			ctx := context.Background()

			allowed, err := tb.AllowN(ctx, "test-key", tt.requests)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if allowed != tt.wantAllow {
				t.Errorf("got %v, want %v", allowed, tt.wantAllow)
			}
		})
	}
}

// TestTokenBucketRefill tests token refill over time
func TestTokenBucketRefill(t *testing.T) {
	tb := NewTokenBucket(10, 10) // 10 RPS, burst 10
	ctx := context.Background()

	// Use up all tokens
	allowed, err := tb.AllowN(ctx, "test-key", 10)
	if err != nil || !allowed {
		t.Fatalf("initial burst failed")
	}

	// Should be denied
	allowed, err = tb.AllowN(ctx, "test-key", 1)
	if err != nil || allowed {
		t.Fatalf("should have been denied")
	}

	// Wait for tokens to refill (100ms = 1 token at 10 RPS)
	time.Sleep(150 * time.Millisecond)

	// Should now allow 1-2 tokens
	allowed, err = tb.AllowN(ctx, "test-key", 1)
	if err != nil || !allowed {
		t.Fatalf("should have been allowed after refill")
	}
}

// TestTokenBucketRemaining tests Remaining calculation
func TestTokenBucketRemaining(t *testing.T) {
	tb := NewTokenBucket(10, 10)
	ctx := context.Background()

	// Initial should be burst size
	remaining, err := tb.Remaining(ctx, "test-key")
	if err != nil || remaining != 10 {
		t.Fatalf("initial remaining: got %d, want 10", remaining)
	}

	// Use 5 tokens
	tb.AllowN(ctx, "test-key", 5)

	remaining, err = tb.Remaining(ctx, "test-key")
	if err != nil || remaining != 5 {
		t.Fatalf("after use: got %d, want 5", remaining)
	}
}

// TestSlidingWindowBasic tests basic sliding window functionality
func TestSlidingWindowBasic(t *testing.T) {
	tests := []struct {
		name      string
		limit     int64
		requests  int64
		wantAllow bool
	}{
		{
			name:      "single request allowed",
			limit:     10,
			requests:  1,
			wantAllow: true,
		},
		{
			name:      "limit allowed",
			limit:     10,
			requests:  10,
			wantAllow: true,
		},
		{
			name:      "limit exceeded",
			limit:     10,
			requests:  11,
			wantAllow: false,
		},
		{
			name:      "zero requests",
			limit:     10,
			requests:  0,
			wantAllow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw := NewSlidingWindow(tt.limit, time.Second)
			ctx := context.Background()

			allowed, err := sw.AllowN(ctx, "test-key", tt.requests)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if allowed != tt.wantAllow {
				t.Errorf("got %v, want %v", allowed, tt.wantAllow)
			}
		})
	}
}

// TestSlidingWindowWindow tests that requests outside window are not counted
func TestSlidingWindowWindow(t *testing.T) {
	sw := NewSlidingWindow(5, time.Second)
	ctx := context.Background()

	// Use 5 requests
	allowed, _ := sw.AllowN(ctx, "test-key", 5)
	if !allowed {
		t.Fatalf("initial requests should be allowed")
	}

	// Next request should be denied (window still active)
	allowed, _ = sw.AllowN(ctx, "test-key", 1)
	if allowed {
		t.Fatalf("should be denied when limit reached in same window")
	}

	// Reset to clear
	err := sw.Reset(ctx, "test-key")
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}

	// Should now allow 5 again
	allowed, _ = sw.AllowN(ctx, "test-key", 5)
	if !allowed {
		t.Fatalf("should allow requests after reset")
	}
}

// TestFixedWindowBasic tests basic fixed window functionality
func TestFixedWindowBasic(t *testing.T) {
	tests := []struct {
		name      string
		limit     int64
		requests  int64
		wantAllow bool
	}{
		{
			name:      "single request allowed",
			limit:     10,
			requests:  1,
			wantAllow: true,
		},
		{
			name:      "limit allowed",
			limit:     10,
			requests:  10,
			wantAllow: true,
		},
		{
			name:      "limit exceeded",
			limit:     10,
			requests:  11,
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw := NewFixedWindow(tt.limit, time.Second)
			ctx := context.Background()

			allowed, err := fw.AllowN(ctx, "test-key", tt.requests)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if allowed != tt.wantAllow {
				t.Errorf("got %v, want %v", allowed, tt.wantAllow)
			}
		})
	}
}

// TestFixedWindowReset tests that window resets properly
func TestFixedWindowReset(t *testing.T) {
	fw := NewFixedWindow(5, time.Second)
	ctx := context.Background()

	// Use all 5 tokens
	allowed, _ := fw.AllowN(ctx, "test-key", 5)
	if !allowed {
		t.Fatalf("initial tokens should be allowed")
	}

	// Should be denied
	allowed, _ = fw.AllowN(ctx, "test-key", 1)
	if allowed {
		t.Fatalf("should be denied when limit reached")
	}

	// Reset
	err := fw.Reset(ctx, "test-key")
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}

	// Should now allow 5 more
	allowed, _ = fw.AllowN(ctx, "test-key", 5)
	if !allowed {
		t.Fatalf("should allow requests after reset")
	}
}

// TestLimiterMemoryTokenBucket tests token bucket limiter
func TestLimiterMemoryTokenBucket(t *testing.T) {
	limiter, err := NewTokenBucketLimiter(10, 10)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// Should allow 10 requests
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx, "test-key")
		if err != nil || !allowed {
			t.Fatalf("request %d failed", i)
		}
	}

	// 11th request should be denied
	allowed, err := limiter.Allow(ctx, "test-key")
	if err != nil || allowed {
		t.Fatalf("11th request should be denied")
	}
}

// TestLimiterMemorySlidingWindow tests sliding window limiter
func TestLimiterMemorySlidingWindow(t *testing.T) {
	limiter, err := NewSlidingWindowLimiter(10, time.Second)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// Should allow 10 requests
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx, "test-key")
		if err != nil || !allowed {
			t.Fatalf("request %d failed", i)
		}
	}

	// 11th request should be denied
	allowed, err := limiter.Allow(ctx, "test-key")
	if err != nil || allowed {
		t.Fatalf("11th request should be denied")
	}
}

// TestLimiterMemoryFixedWindow tests fixed window limiter
func TestLimiterMemoryFixedWindow(t *testing.T) {
	limiter, err := NewFixedWindowLimiter(10, time.Second)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// Should allow 10 requests
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx, "test-key")
		if err != nil || !allowed {
			t.Fatalf("request %d failed", i)
		}
	}

	// 11th request should be denied
	allowed, err := limiter.Allow(ctx, "test-key")
	if err != nil || allowed {
		t.Fatalf("11th request should be denied")
	}
}

// TestReset tests the Reset functionality
func TestReset(t *testing.T) {
	limiter, _ := NewTokenBucketLimiter(10, 10)
	ctx := context.Background()

	// Use up all tokens
	limiter.AllowN(ctx, "test-key", 10)

	// Should be denied
	allowed, _ := limiter.Allow(ctx, "test-key")
	if allowed {
		t.Fatalf("should be denied")
	}

	// Reset
	err := limiter.Reset(ctx, "test-key")
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}

	// Should now be allowed
	allowed, _ = limiter.Allow(ctx, "test-key")
	if !allowed {
		t.Fatalf("should be allowed after reset")
	}
}

// TestHealth tests Health check
func TestHealth(t *testing.T) {
	limiter, _ := NewTokenBucketLimiter(10, 10)
	ctx := context.Background()

	err := limiter.Health(ctx)
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
}

// TestMultipleTenants tests that different keys are isolated
func TestMultipleTenants(t *testing.T) {
	limiter, _ := NewTokenBucketLimiter(10, 10)
	ctx := context.Background()

	// Tenant 1 uses 10 tokens
	for i := 0; i < 10; i++ {
		limiter.Allow(ctx, "tenant-1")
	}

	// Tenant 2 should still have tokens
	allowed, _ := limiter.Allow(ctx, "tenant-2")
	if !allowed {
		t.Fatalf("tenant-2 should have tokens")
	}

	// Tenant 1 should be denied
	allowed, _ = limiter.Allow(ctx, "tenant-1")
	if allowed {
		t.Fatalf("tenant-1 should be denied")
	}
}

// TestConcurrency tests concurrent access to the limiter
func TestConcurrency(t *testing.T) {
	limiter, _ := NewTokenBucketLimiter(500, 500)
	ctx := context.Background()

	var allowed int64
	var denied int64
	var wg sync.WaitGroup

	// 50 goroutines making 100 requests each
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				allow, _ := limiter.Allow(ctx, "concurrent-key")
				if allow {
					atomic.AddInt64(&allowed, 1)
				} else {
					atomic.AddInt64(&denied, 1)
				}
			}
		}()
	}

	wg.Wait()

	// Should allow up to 500, deny the rest (with some tolerance for timing)
	total := allowed + denied
	if total != 5000 {
		t.Errorf("total requests: got %d, want 5000", total)
	}

	// Allow some tolerance due to timing
	if allowed < 450 || allowed > 550 {
		t.Errorf("allowed: got %d, expected around 500", allowed)
	}
}

// TestStats tests Stats functionality
func TestStats(t *testing.T) {
	limiter, _ := NewTokenBucketLimiter(10, 10)
	ctx := context.Background()

	// Make some requests
	limiter.Allow(ctx, "key-1")
	limiter.Allow(ctx, "key-2")
	limiter.Allow(ctx, "key-3")

	stats := limiter.Stats()

	if stats["algorithm"] != "token_bucket" {
		t.Errorf("algorithm: got %v, want token_bucket", stats["algorithm"])
	}

	if stats["rps"] != 10.0 {
		t.Errorf("rps: got %v, want 10.0", stats["rps"])
	}

	if stats["burstSize"] != int64(10) {
		t.Errorf("burstSize: got %v, want 10", stats["burstSize"])
	}

	activeBuckets := stats["activeBuckets"].(int)
	if activeBuckets < 1 {
		t.Errorf("activeBuckets: got %d, want >= 1", activeBuckets)
	}
}

// TestRemaining tests the Remaining functionality
func TestRemaining(t *testing.T) {
	limiter, _ := NewTokenBucketLimiter(10, 10)
	ctx := context.Background()

	// Initially should have all tokens
	remaining, _ := limiter.Remaining(ctx, "test-key")
	if remaining != 10 {
		t.Errorf("initial remaining: got %d, want 10", remaining)
	}

	// Use 3 tokens
	limiter.AllowN(ctx, "test-key", 3)

	remaining, _ = limiter.Remaining(ctx, "test-key")
	if remaining != 7 {
		t.Errorf("after use: got %d, want 7", remaining)
	}
}

// BenchmarkTokenBucket benchmarks token bucket performance
func BenchmarkTokenBucket(b *testing.B) {
	tb := NewTokenBucket(10000, 10000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow(ctx, fmt.Sprintf("bench-key-%d", i%1000))
	}
}

// BenchmarkSlidingWindow benchmarks sliding window performance
func BenchmarkSlidingWindow(b *testing.B) {
	sw := NewSlidingWindow(10000, time.Second)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sw.Allow(ctx, fmt.Sprintf("bench-key-%d", i%1000))
	}
}

// BenchmarkFixedWindow benchmarks fixed window performance
func BenchmarkFixedWindow(b *testing.B) {
	fw := NewFixedWindow(10000, time.Second)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fw.Allow(ctx, fmt.Sprintf("bench-key-%d", i%1000))
	}
}

// BenchmarkConcurrent benchmarks concurrent access
func BenchmarkConcurrent(b *testing.B) {
	limiter, _ := NewTokenBucketLimiter(100000, 100000)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			limiter.Allow(ctx, fmt.Sprintf("bench-key-%d", i%1000))
			i++
		}
	})
}
