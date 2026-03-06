package ports

import (
	"context"
)

// Metrics is a port interface for collecting per-tenant metrics.
// Implementations report metrics to various backends (Prometheus, DataDog, etc.).
type Metrics interface {
	// RecordRequest records an HTTP request metric.
	RecordRequest(ctx context.Context, method string, endpoint string, statusCode int, durationMS int64) error

	// RecordQuery records a database query metric.
	RecordQuery(ctx context.Context, query string, rowsAffected int64, durationMS int64) error

	// RecordError records an error metric.
	RecordError(ctx context.Context, errorType string, message string) error

	// RecordQuotaUsage records quota consumption.
	RecordQuotaUsage(ctx context.Context, quotaType string, used int64, limit int64) error

	// RecordRateLimit records rate limit hits.
	RecordRateLimit(ctx context.Context, endpoint string, limited bool) error

	// RecordCacheHit records a cache hit or miss.
	RecordCacheHit(ctx context.Context, key string, hit bool) error
}
