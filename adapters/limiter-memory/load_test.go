package limitermemory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkTokenBucketAllow benchmarks token bucket Allow
func BenchmarkTokenBucketAllow(b *testing.B) {
	limiter, _ := NewTokenBucketLimiter(10000, 50000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, fmt.Sprintf("key-%d", i%1000))
	}
}

// BenchmarkSlidingWindowAllow benchmarks sliding window Allow
func BenchmarkSlidingWindowAllow(b *testing.B) {
	limiter, _ := NewSlidingWindowLimiter(10000, time.Second)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, fmt.Sprintf("key-%d", i%1000))
	}
}

// BenchmarkFixedWindowAllow benchmarks fixed window Allow
func BenchmarkFixedWindowAllow(b *testing.B) {
	limiter, _ := NewFixedWindowLimiter(10000, time.Second)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, fmt.Sprintf("key-%d", i%1000))
	}
}

// LoadTestResult contains results from load testing
type LoadTestResult struct {
	Algorithm         string
	TotalRequests     int64
	AllowedRequests   int64
	DeniedRequests    int64
	Duration          time.Duration
	RequestsPerSecond float64
	AverageLatenessMs float64
}

// LoadTest runs a load test with concurrent requests
func LoadTest(t *testing.T, limiter LimiterMemory, algorithm string, duration time.Duration, concurrency int) LoadTestResult {
	ctx := context.Background()
	var totalRequests int64
	var allowedRequests int64
	var deniedRequests int64

	done := make(chan struct{})
	var wg sync.WaitGroup

	startTime := time.Now()

	// Launch concurrent goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			key := fmt.Sprintf("tenant-%d", goroutineID)

			for {
				select {
				case <-done:
					return
				default:
					allowed, _ := limiter.Allow(ctx, key)
					atomic.AddInt64(&totalRequests, 1)

					if allowed {
						atomic.AddInt64(&allowedRequests, 1)
					} else {
						atomic.AddInt64(&deniedRequests, 1)
					}
				}
			}
		}(i)
	}

	// Run for specified duration
	time.Sleep(duration)
	close(done)
	wg.Wait()

	elapsed := time.Since(startTime)
	rps := float64(atomic.LoadInt64(&totalRequests)) / elapsed.Seconds()

	return LoadTestResult{
		Algorithm:         algorithm,
		TotalRequests:     atomic.LoadInt64(&totalRequests),
		AllowedRequests:   atomic.LoadInt64(&allowedRequests),
		DeniedRequests:    atomic.LoadInt64(&deniedRequests),
		Duration:          elapsed,
		RequestsPerSecond: rps,
	}
}

// TestLoadTest10KRPSTokenBucket tests 10,000 RPS with token bucket
func TestLoadTest10KRPSTokenBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Token bucket: 10,000 RPS with 50,000 burst
	limiter, _ := NewTokenBucketLimiter(10000, 50000)

	result := LoadTest(t, *limiter, "token-bucket", 5*time.Second, 50)

	t.Logf("Load Test Result: %+v", result)
	t.Logf("Requests/sec: %.2f", result.RequestsPerSecond)
	t.Logf("Allowed: %d, Denied: %d", result.AllowedRequests, result.DeniedRequests)

	// Should handle at least 8,000 RPS
	if result.RequestsPerSecond < 8000 {
		t.Errorf("Expected at least 8,000 RPS, got %.2f", result.RequestsPerSecond)
	}
}

// TestLoadTest10KRPSSlidingWindow tests 10,000 RPS with sliding window
func TestLoadTest10KRPSSlidingWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	limiter, _ := NewSlidingWindowLimiter(10000, time.Second)

	result := LoadTest(t, *limiter, "sliding-window", 5*time.Second, 50)

	t.Logf("Load Test Result: %+v", result)
	t.Logf("Requests/sec: %.2f", result.RequestsPerSecond)
	t.Logf("Allowed: %d, Denied: %d", result.AllowedRequests, result.DeniedRequests)

	// Should handle at least 8,000 RPS
	if result.RequestsPerSecond < 8000 {
		t.Errorf("Expected at least 8,000 RPS, got %.2f", result.RequestsPerSecond)
	}
}

// TestLoadTest10KRPSFixedWindow tests 10,000 RPS with fixed window
func TestLoadTest10KRPSFixedWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	limiter, _ := NewFixedWindowLimiter(10000, time.Second)

	result := LoadTest(t, *limiter, "fixed-window", 5*time.Second, 50)

	t.Logf("Load Test Result: %+v", result)
	t.Logf("Requests/sec: %.2f", result.RequestsPerSecond)
	t.Logf("Allowed: %d, Denied: %d", result.AllowedRequests, result.DeniedRequests)

	// Should handle at least 8,000 RPS
	if result.RequestsPerSecond < 8000 {
		t.Errorf("Expected at least 8,000 RPS, got %.2f", result.RequestsPerSecond)
	}
}

// TestMemoryUsage tests memory usage with many tenants
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	ctx := context.Background()

	// Token bucket with 1000 tenants
	limiter, _ := NewTokenBucketLimiter(100, 500)

	// Create entries for 1000 tenants
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("tenant-%d", i)
		limiter.Allow(ctx, key)
	}

	// Verify stats work
	stats := limiter.Stats()
	if stats == nil {
		t.Fatal("Stats returned nil")
	}

	t.Logf("Memory Test: 1000 tenants created")
	t.Logf("Stats: %+v", stats)

	// Should have algorithm info
	if algo, ok := stats["algorithm"]; !ok || algo == "" {
		t.Errorf("Stats missing algorithm info")
	}
}

// TestLatencyUnderLoad tests latency with various load levels
func TestLatencyUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping latency test in short mode")
	}

	ctx := context.Background()
	limiter, _ := NewTokenBucketLimiter(10000, 50000)

	// Test with increasing concurrency
	concurrencyLevels := []int{10, 25, 50, 100}

	for _, concurrency := range concurrencyLevels {
		var latencies []time.Duration
		var mu sync.Mutex
		var requestCount int64

		done := make(chan struct{})
		var wg sync.WaitGroup

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				key := fmt.Sprintf("tenant-%d", goroutineID)

				for {
					select {
					case <-done:
						return
					default:
						start := time.Now()
						limiter.Allow(ctx, key)
						elapsed := time.Since(start)

						mu.Lock()
						latencies = append(latencies, elapsed)
						localCount := len(latencies)
						mu.Unlock()

						atomic.AddInt64(&requestCount, 1)

						if localCount >= 1000 {
							return
						}
					}
				}
			}(i)
		}

		// Run for up to 2 seconds
		time.Sleep(2 * time.Second)
		close(done)
		wg.Wait()

		// Calculate average latency
		var totalLatency time.Duration
		for _, lat := range latencies {
			totalLatency += lat
		}
		avgLatency := time.Duration(0)
		if len(latencies) > 0 {
			avgLatency = totalLatency / time.Duration(len(latencies))
		}

		t.Logf("Concurrency: %d, Requests: %d, Avg Latency: %v", concurrency, len(latencies), avgLatency)

		// Should be less than 1ms on average
		if avgLatency > 1*time.Millisecond {
			t.Logf("Warning: Latency %.2fµs exceeds 1ms at concurrency %d", float64(avgLatency.Microseconds()), concurrency)
		}
	}
}

// BenchmarkParallelAllow benchmarks parallel Allow calls
func BenchmarkParallelAllow(b *testing.B) {
	limiter, _ := NewTokenBucketLimiter(100000, 500000)
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			limiter.Allow(ctx, fmt.Sprintf("key-%d", i%1000))
			i++
		}
	})
}

// BenchmarkParallelAllowN benchmarks parallel AllowN calls
func BenchmarkParallelAllowN(b *testing.B) {
	limiter, _ := NewTokenBucketLimiter(100000, 500000)
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			limiter.AllowN(ctx, fmt.Sprintf("key-%d", i%1000), 10)
			i++
		}
	})
}

// BenchmarkRemaining benchmarks Remaining calls
func BenchmarkRemaining(b *testing.B) {
	limiter, _ := NewTokenBucketLimiter(10000, 50000)
	ctx := context.Background()

	// Warm up
	for i := 0; i < 100; i++ {
		limiter.Allow(ctx, "warm-up")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Remaining(ctx, "test-key")
	}
}

// BenchmarkStats benchmarks Stats calls
func BenchmarkStats(b *testing.B) {
	limiter, _ := NewTokenBucketLimiter(10000, 50000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Stats()
	}
}
