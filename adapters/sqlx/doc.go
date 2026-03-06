package sqlx

// Package sqlx provides a sqlx adapter for tenant-scoped database operations.
//
// It wraps a *sqlx.DB with automatic tenant filtering, supporting named
// queries and struct scanning.
//
// # Usage
//
//	wrappedDB, _ := sqlx.NewDB(sqlx.Config{
//	    DB:           sqlxDB,
//	    TenantColumn: "tenant_id",
//	})
