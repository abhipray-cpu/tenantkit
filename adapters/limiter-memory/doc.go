package limitermemory

// Package limitermemory provides an in-memory rate limiter implementing [ports.Limiter].
//
// It supports three rate limiting algorithms:
//   - Token Bucket: smooth rate with burst capacity
//   - Sliding Window: precise rate counting over a sliding time window
//   - Fixed Window: simple counter reset at fixed intervals
//
// Suitable for single-instance deployments. For distributed systems,
// use the limiter-redis adapter instead.
//
// # Usage
//
//	limiter, _ := limitermemory.New(limitermemory.Config{
//	    Rate:      100,
//	    Period:    time.Second,
//	    Algorithm: limitermemory.TokenBucket,
//	})
//
//	allowed, _ := limiter.Allow(ctx, "tenant-1")
