// Package tenantkit provides transparent, automatic multi-tenant isolation
// for Go applications using database/sql.
//
// TenantKit wraps a standard *sql.DB and transparently rewrites SQL queries
// to include tenant filtering. Application code continues to write normal SQL
// while TenantKit ensures every query is scoped to the correct tenant.
//
// # Quick Start
//
//	wrappedDB, err := tenantkit.Wrap(db, tenantkit.Config{
//	    TenantTables: []string{"users", "orders"},
//	    TenantColumn: "tenant_id",
//	})
//
//	ctx := tenantkit.WithTenant(context.Background(), "acme-corp")
//	rows, err := wrappedDB.Query(ctx, "SELECT * FROM users")
//	// Executed as: SELECT * FROM users WHERE users.tenant_id = $1
//
// # How It Works
//
// The library uses a Two-Rule Decision System:
//
//  1. System queries (DDL, health checks, catalog queries) are automatically bypassed.
//  2. Only tables listed in TenantTables get tenant filtering — other tables pass through.
//
// The query cache (LRU with FNV-1a hashing) avoids re-parsing identical query
// patterns, and sync.Pool optimizations minimize allocations on the hot path.
//
// # Placeholder Styles
//
// Wrap() auto-detects PostgreSQL ($1) style. For MySQL/SQLite (?), use WrapWithStyle:
//
//	wrappedDB, err := tenantkit.WrapWithStyle(db, config, tenantkit.PlaceholderQuestion)
//
// # Transaction Support
//
//	tx, err := wrappedDB.Begin(ctx)
//	tx.Exec(ctx, "INSERT INTO orders ...")
//	tx.Commit()
//
// # Bypassing Tenant Filtering
//
// For admin or cross-tenant operations:
//
//	ctx := tenantkit.WithoutTenantFiltering(context.Background())
//	rows, err := wrappedDB.Query(ctx, "SELECT COUNT(*) FROM users")
//
// # Zero Dependencies
//
// The core package uses only the Go standard library. Adapters for HTTP
// frameworks, Redis, Prometheus, etc. are separate modules so you only
// pull in what you need.
package tenantkit
