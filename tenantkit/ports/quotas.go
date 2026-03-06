package ports

import (
	"context"
)

// QuotaManager is a port interface for managing per-tenant resource quotas.
// Quotas can be based on API requests, data volume, or custom metrics.
type QuotaManager interface {
	// CheckQuota checks if the current usage is within quota limits.
	// Returns true if within limits, false if quota exceeded.
	CheckQuota(ctx context.Context, quotaType string, amount int64) (bool, error)

	// ConsumeQuota consumes quota and returns remaining quota.
	// Returns an error if quota would be exceeded.
	ConsumeQuota(ctx context.Context, quotaType string, amount int64) (int64, error)

	// GetUsage returns the current usage and limit for a quota type.
	GetUsage(ctx context.Context, quotaType string) (used int64, limit int64, err error)

	// ResetQuota resets the quota for a type (usually daily/monthly reset).
	ResetQuota(ctx context.Context, quotaType string) error

	// SetLimit sets the quota limit for a type.
	SetLimit(ctx context.Context, quotaType string, limit int64) error
}
