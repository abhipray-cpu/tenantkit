package quota

// Package quota provides an in-memory quota manager implementing [ports.QuotaManager].
//
// It enforces per-tenant resource quotas to prevent noisy-neighbor problems
// in multi-tenant systems. Quotas track usage against configurable limits
// for different resource types (queries, rows, storage, connections, etc.).
//
// Suitable for single-instance deployments. For distributed systems,
// use the quota-redis adapter instead.
//
// # Usage
//
//	qm := quota.NewInMemoryQuotaManager()
//	qm.SetLimit(ctx, "tenant-1", "queries", 1000)
//
//	allowed, _ := qm.CheckQuota(ctx, "tenant-1", "queries", 1)
