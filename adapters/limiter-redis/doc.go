package limiterredis

// Package limiterredis provides a Redis-backed rate limiter implementing [ports.Limiter].
//
// It uses atomic Redis operations for distributed rate limiting across
// multiple application instances. Suitable for production distributed systems.
//
// # Usage
//
//	limiter, _ := limiterredis.New(limiterredis.Config{
//	    RedisAddr: "localhost:6379",
//	    Rate:      100,
//	    Period:    time.Second,
//	})
//
//	allowed, _ := limiter.Allow(ctx, "tenant-1")
