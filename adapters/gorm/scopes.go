package gormadapter

import (
	"context"

	"github.com/abhipray-cpu/tenantkit/domain"
	"gorm.io/gorm"
)

// SkipTenant is a GORM scope that skips automatic tenant scoping for a query.
// Use this for system-level operations that need to access all tenants' data.
//
// Example:
//
//	db.Scopes(gormadapter.SkipTenant()).Find(&users)
func SkipTenant() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Set("tenantkit:skip", true)
	}
}

// WithTenant is a GORM scope that explicitly sets the tenant context for a query.
// This is useful when you want to query data for a specific tenant without
// modifying the context.
//
// Example:
//
//	db.Scopes(gormadapter.WithTenant("tenant-123")).Find(&users)
func WithTenant(tenantID string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// Create a tenant context
		tid, err := domain.NewTenantID(tenantID)
		if err != nil {
			db.AddError(err)
			return db
		}

		tenantCtx, err := domain.NewContext(tenantID, "system", "explicit-scope")
		if err != nil {
			db.AddError(err)
			return db
		}

		// Create a new context with tenant
		ctx := tenantCtx.ToGoContext(db.Statement.Context)
		if ctx == nil {
			ctx = tenantCtx.ToGoContext(context.Background())
		}

		return db.WithContext(ctx).Where("tenant_id = ?", tid.Value())
	}
}

// ForTenant is similar to WithTenant but takes a domain.TenantID directly.
//
// Example:
//
//	tenantID := domain.MustNewTenantID("tenant-123")
//	db.Scopes(gormadapter.ForTenant(tenantID)).Find(&users)
func ForTenant(tenantID domain.TenantID) func(*gorm.DB) *gorm.DB {
	return WithTenant(tenantID.Value())
}

// WithContext is a helper that wraps a GORM query with a tenant context.
// This is the recommended way to use GORM with TenantKit.
//
// Example:
//
//	tenantCtx, _ := domain.FromGoContext(ctx)
//	db = gormadapter.WithContext(db, tenantCtx)
//	db.Find(&users)
func WithContext(db *gorm.DB, tenantCtx domain.Context) *gorm.DB {
	ctx := tenantCtx.ToGoContext(db.Statement.Context)
	if ctx == nil {
		ctx = tenantCtx.ToGoContext(context.Background())
	}
	return db.WithContext(ctx)
}

// TenantOnly is a scope that ensures a query will only return results for the current tenant.
// It's mainly for documentation purposes as the plugin already enforces this automatically.
//
// Example:
//
//	db.Scopes(gormadapter.TenantOnly()).Find(&users)
func TenantOnly() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// This is a no-op since the plugin already enforces tenant scoping
		// But it makes the intent explicit in the code
		return db
	}
}

// AllTenants is an alias for SkipTenant for better readability.
// Use this when you explicitly want to query across all tenants.
//
// Example:
//
//	db.Scopes(gormadapter.AllTenants()).Find(&users)
func AllTenants() func(*gorm.DB) *gorm.DB {
	return SkipTenant()
}
