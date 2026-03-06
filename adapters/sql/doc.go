package sqladapter

// Package sqladapter provides a database/sql adapter implementing [ports.Storage] and [ports.Enforcer].
//
// It wraps a *sql.DB with tenant-aware storage operations and SQL query
// enforcement for multi-tenant applications.
//
// # Usage
//
//	storage, _ := sqladapter.NewStorage(sqladapter.Config{
//	    DB:           db,
//	    TenantColumn: "tenant_id",
//	})
