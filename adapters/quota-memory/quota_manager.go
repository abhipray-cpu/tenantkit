package quota

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abhipray-cpu/tenantkit/domain"
	"github.com/abhipray-cpu/tenantkit/tenantkit/ports"
)

// Compile-time interface check
var _ ports.QuotaManager = (*InMemoryQuotaManager)(nil)

// InMemoryQuotaManager is an in-memory implementation of QuotaManager port
type InMemoryQuotaManager struct {
	mu      sync.RWMutex
	limits  map[string]int64       // quotaType -> limit
	usage   map[string]*QuotaUsage // tenant:quotaType -> usage
	resets  map[string]*QuotaResetPolicy
	lastRun map[string]time.Time
}

// QuotaUsage tracks current usage of a quota
type QuotaUsage struct {
	used   atomic.Int64
	limit  atomic.Int64
	period string
	start  time.Time
}

// QuotaResetPolicy defines how/when a quota resets
type QuotaResetPolicy struct {
	period  time.Duration
	autoRun bool
	nextRun time.Time
}

// NewInMemoryQuotaManager creates a new in-memory quota manager
// ⚠️ WARNING: Not suitable for multi-instance deployments
func NewInMemoryQuotaManager() *InMemoryQuotaManager {
	// Log warning in production environments
	env := os.Getenv("ENV")
	goEnv := os.Getenv("GO_ENV")
	if env == "production" || goEnv == "production" {
		log.Println("⚠️  WARNING: Using in-memory quota manager in production. " +
			"This will cause quota enforcement inconsistency in multi-instance deployments. " +
			"Consider using github.com/abhipray-cpu/tenantkit/adapters/quota-redis instead.")
	}

	return &InMemoryQuotaManager{
		limits: map[string]int64{
			"api_requests_monthly": 100000,
			"api_requests_daily":   5000,
			"database_rows":        1000000,
			"storage_bytes":        1073741824,
		},
		usage:   make(map[string]*QuotaUsage),
		resets:  make(map[string]*QuotaResetPolicy),
		lastRun: make(map[string]time.Time),
	}
}

// CheckQuota verifies if usage would exceed the limit
func (q *InMemoryQuotaManager) CheckQuota(ctx context.Context, quotaType string, amount int64) (bool, error) {
	if amount < 0 {
		return false, fmt.Errorf("amount must be non-negative")
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	tenantID := getTenantIDFromContext(ctx)
	key := fmt.Sprintf("%s:%s", tenantID, quotaType)

	// Check if quota type is registered
	limit, limitExists := q.limits[quotaType]
	if !limitExists {
		return false, fmt.Errorf("unknown quota type: %s", quotaType)
	}

	usage, exists := q.usage[key]
	if !exists {
		// No usage yet, check if amount fits within limit
		return amount <= limit, nil
	}

	currentUsed := usage.used.Load()
	usageLimit := usage.limit.Load()

	return currentUsed+amount <= usageLimit, nil
}

// ConsumeQuota consumes quota and returns remaining
func (q *InMemoryQuotaManager) ConsumeQuota(ctx context.Context, quotaType string, amount int64) (int64, error) {
	if amount < 0 {
		return 0, fmt.Errorf("amount must be non-negative")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	tenantID := getTenantIDFromContext(ctx)
	key := fmt.Sprintf("%s:%s", tenantID, quotaType)

	usage, exists := q.usage[key]
	if !exists {
		// Get the limit from q.limits
		limit, limitExists := q.limits[quotaType]

		// If limit not set, use unlimited (MaxInt64)
		if !limitExists {
			limit = 9223372036854775807 // math.MaxInt64
		}
		// If limit is 0, it was explicitly set to 0, so we keep it

		usage = &QuotaUsage{
			period: getPeriodForQuotaType(quotaType),
			start:  time.Now(),
		}
		usage.limit.Store(limit)
		usage.used.Store(0)

		q.usage[key] = usage
	} else {
		// Check if period has elapsed and reset if needed
		if shouldResetQuota(usage) {
			usage.used.Store(0)
			usage.start = time.Now()
		}
	}

	currentUsed := usage.used.Load()
	limit := usage.limit.Load()

	if currentUsed+amount > limit {
		return 0, domain.ErrQuotaExceeded
	}

	newUsed := usage.used.Add(amount)
	remaining := limit - newUsed

	return remaining, nil
}

// GetUsage returns current usage and limit
func (q *InMemoryQuotaManager) GetUsage(ctx context.Context, quotaType string) (used int64, limit int64, err error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	tenantID := getTenantIDFromContext(ctx)
	key := fmt.Sprintf("%s:%s", tenantID, quotaType)

	usage, exists := q.usage[key]
	if !exists {
		return 0, q.limits[quotaType], nil
	}

	return usage.used.Load(), usage.limit.Load(), nil
}

// ResetQuota resets a quota to zero
func (q *InMemoryQuotaManager) ResetQuota(ctx context.Context, quotaType string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	tenantID := getTenantIDFromContext(ctx)
	key := fmt.Sprintf("%s:%s", tenantID, quotaType)

	usage, exists := q.usage[key]
	if !exists {
		return nil
	}

	usage.used.Store(0)
	usage.start = time.Now()
	q.lastRun[quotaType] = time.Now()

	return nil
}

// SetLimit sets a new quota limit
func (q *InMemoryQuotaManager) SetLimit(ctx context.Context, quotaType string, limit int64) error {
	if limit <= 0 {
		return fmt.Errorf("limit must be positive")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	tenantID := getTenantIDFromContext(ctx)
	key := fmt.Sprintf("%s:%s", tenantID, quotaType)

	q.limits[quotaType] = limit

	if usage, exists := q.usage[key]; exists {
		usage.limit.Store(limit)
	}

	return nil
}

func getTenantIDFromContext(ctx context.Context) string {
	tenantCtx, err := domain.FromGoContext(ctx)
	if err == nil {
		return tenantCtx.TenantID().Value()
	}
	return "default-tenant"
}

func shouldResetQuota(usage *QuotaUsage) bool {
	elapsed := time.Since(usage.start)
	switch usage.period {
	case "short":
		// 100ms period for testing
		return elapsed >= 100*time.Millisecond
	case "daily":
		// 24 hours
		return elapsed >= 24*time.Hour
	case "monthly":
		// 30 days (approximate)
		return elapsed >= 30*24*time.Hour
	default:
		// No reset for unknown periods
		return false
	}
}

func getPeriodForQuotaType(quotaType string) string {
	switch quotaType {
	case "api_requests_daily":
		return "daily"
	case "api_requests_monthly":
		return "monthly"
	default:
		// For unknown/unspecified quota types, use a short period (100ms)
		// to allow for testing and default behavior
		return "short"
	}
}

// BulkResetQuotas resets all quotas for a tenant
func (q *InMemoryQuotaManager) BulkResetQuotas(ctx context.Context, quotaTypes []string) error {
	for _, quotaType := range quotaTypes {
		if err := q.ResetQuota(ctx, quotaType); err != nil {
			return err
		}
	}
	return nil
}

// GetAllQuotas returns all quota usage for a tenant
func (q *InMemoryQuotaManager) GetAllQuotas(ctx context.Context) (map[string][2]int64, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	tenantID := getTenantIDFromContext(ctx)
	result := make(map[string][2]int64)

	for quotaType, limit := range q.limits {
		key := fmt.Sprintf("%s:%s", tenantID, quotaType)
		usage, exists := q.usage[key]
		if exists {
			result[quotaType] = [2]int64{usage.used.Load(), limit}
		} else {
			result[quotaType] = [2]int64{0, limit}
		}
	}

	return result, nil
}

// QuotaStats holds statistics about quota system
type QuotaStats struct {
	TotalQuotaTypes int
	ActiveQuotas    int
	TotalUsage      int64
}

// GetStats returns statistics about quota usage
func (q *InMemoryQuotaManager) GetStats() QuotaStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := QuotaStats{
		TotalQuotaTypes: len(q.limits),
		ActiveQuotas:    len(q.usage),
	}

	for _, usage := range q.usage {
		stats.TotalUsage += usage.used.Load()
	}

	return stats
}
