package tenantkit

import (
	"context"

	"github.com/abhipray-cpu/tenantkit/domain"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// tenantIDKey is the context key for storing tenant ID
	tenantIDKey contextKey = "tenantkit_tenant_id"
	// bypassKey is the context key for bypass flag
	bypassKey contextKey = "tenantkit_bypass"
)

// WithTenant adds a tenant ID to the context.
// This marks the context as belonging to a specific tenant, enabling
// automatic tenant filtering for queries.
//
// Example:
//
//	ctx := tenantkit.WithTenant(context.Background(), "tenant-123")
//	rows, err := db.Query(ctx, "SELECT * FROM users")
//	// Query automatically filtered: WHERE tenant_id = 'tenant-123'
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// GetTenant retrieves the tenant ID from the context.
// Returns the tenant ID and true if present and non-empty, empty string and false otherwise.
//
// This function checks both the simple string key (set by WithTenant) and the
// domain context key (set by HTTP middleware adapters via domain.ToGoContext),
// ensuring seamless integration between middleware and database layers.
//
// Example:
//
//	tenantID, ok := tenantkit.GetTenant(ctx)
//	if !ok {
//	    return errors.New("tenant context required")
//	}
func GetTenant(ctx context.Context) (string, bool) {
	// Primary: simple string key set by WithTenant()
	tenantID, ok := ctx.Value(tenantIDKey).(string)
	if ok && tenantID != "" {
		return tenantID, true
	}
	// Fallback: domain context set by HTTP middleware adapters
	tc, err := domain.FromGoContext(ctx)
	if err == nil {
		id := tc.TenantID().Value()
		return id, id != ""
	}
	return "", false
}

// WithoutTenantFiltering marks the context to bypass tenant filtering.
// This is an escape hatch for administrative queries that need to access
// data across all tenants.
//
// ⚠️  Use with caution! This bypasses all tenant isolation.
//
// Example:
//
//	// Admin query to get stats across all tenants
//	ctx := tenantkit.WithoutTenantFiltering(context.Background())
//	rows, err := db.Query(ctx, "SELECT tenant_id, COUNT(*) FROM users GROUP BY tenant_id")
func WithoutTenantFiltering(ctx context.Context) context.Context {
	return context.WithValue(ctx, bypassKey, true)
}

// shouldBypass checks if the context has the bypass flag set.
// This is an internal function used by the interceptor.
func shouldBypass(ctx context.Context) bool {
	bypass, ok := ctx.Value(bypassKey).(bool)
	return ok && bypass
}
