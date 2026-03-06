package quotaredis

// Package quotaredis provides a Redis-backed quota manager implementing [ports.QuotaManager].
//
// It uses atomic Lua scripts for distributed quota enforcement across
// multiple application instances, ensuring consistent noisy-neighbor
// protection in horizontally-scaled deployments.
//
// All operations (CheckQuota, SetLimit, GetUsage, ResetQuota) are
// atomic at the Redis level, preventing race conditions.
//
// # Usage
//
//	qm, _ := quotaredis.New(quotaredis.Config{
//	    RedisAddr: "localhost:6379",
//	    KeyPrefix: "tenantkit:quota:",
//	})
//
//	qm.SetLimit(ctx, "tenant-1", "queries", 1000)
//	allowed, _ := qm.CheckQuota(ctx, "tenant-1", "queries", 1)
