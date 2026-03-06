// Package noop provides no-op metrics adapter for testing
package noop

import (
	"context"

	"github.com/abhipray-cpu/tenantkit/tenantkit/ports"
)

// Compile-time interface check
var _ ports.Metrics = (*NoOpMetrics)(nil)

// NoOpMetrics is a no-op implementation of the Metrics port interface.
// All methods return nil without performing any actions.
// This is useful for testing scenarios where metrics collection is not needed.
type NoOpMetrics struct{}

// NewNoOpMetrics creates a new no-op metrics instance
func NewNoOpMetrics() *NoOpMetrics {
	return &NoOpMetrics{}
}

// RecordRequest does nothing and returns nil
func (n *NoOpMetrics) RecordRequest(ctx context.Context, method string, endpoint string, statusCode int, durationMS int64) error {
	return nil
}

// RecordQuery does nothing and returns nil
func (n *NoOpMetrics) RecordQuery(ctx context.Context, query string, rowsAffected int64, durationMS int64) error {
	return nil
}

// RecordError does nothing and returns nil
func (n *NoOpMetrics) RecordError(ctx context.Context, errorType string, message string) error {
	return nil
}

// RecordQuotaUsage does nothing and returns nil
func (n *NoOpMetrics) RecordQuotaUsage(ctx context.Context, quotaType string, used int64, limit int64) error {
	return nil
}

// RecordRateLimit does nothing and returns nil
func (n *NoOpMetrics) RecordRateLimit(ctx context.Context, endpoint string, limited bool) error {
	return nil
}

// RecordCacheHit does nothing and returns nil
func (n *NoOpMetrics) RecordCacheHit(ctx context.Context, key string, hit bool) error {
	return nil
}
