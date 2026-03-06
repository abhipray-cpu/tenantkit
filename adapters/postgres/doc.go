package postgres

// Package postgres provides PostgreSQL-specific features for TenantKit,
// including Row-Level Security (RLS) policy management.
//
// RLS provides a database-level defense-in-depth layer on top of TenantKit's
// query rewriting. Even if a query bypasses TenantKit, PostgreSQL will enforce
// tenant isolation at the database level.
//
// # Usage
//
//	rls := postgres.NewRLSManager(postgres.RLSConfig{
//	    TenantColumn: "tenant_id",
//	    SessionVar:   "app.tenant_id",
//	})
//
//	rls.EnableRLS(ctx, db, "users")
