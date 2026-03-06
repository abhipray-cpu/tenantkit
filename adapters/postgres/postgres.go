// Package postgres provides PostgreSQL-specific multi-tenancy features.
//
// This adapter focuses on PostgreSQL's Row-Level Security (RLS) capabilities,
// which provide database-level tenant isolation. For general PostgreSQL database
// operations (CRUD, transactions, etc.), use the standard adapters/sql adapter
// with a PostgreSQL driver (github.com/lib/pq or github.com/jackc/pgx).
//
// # Row-Level Security
//
// RLS is a PostgreSQL feature that restricts which rows can be accessed or
// modified based on session variables. It provides:
//   - Database-level security (cannot be bypassed by application bugs)
//   - Zero application overhead (filtering happens in PostgreSQL)
//   - Automatic tenant isolation (no query rewriting needed)
//
// See rls.go for the RLS implementation and README.md for usage examples.
//
// # Usage
//
// Basic setup:
//
//	conn, _ := pgx.Connect(context.Background(), "postgres://localhost/myapp")
//	rlsManager := postgres.NewRLSManager(conn, postgres.RLSConfig{})
//	rlsManager.EnableRLS(ctx, "users")
//	rlsManager.SetTenantContext(ctx, "tenant-123")
//
// See rls.go for full API documentation.
package postgres
