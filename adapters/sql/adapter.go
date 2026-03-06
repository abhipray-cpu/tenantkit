// Package sqladapter provides SQL storage adapter for multi-tenant applications.
// It implements the ports.Storage and ports.Enforcer interfaces by wrapping
// database/sql.DB and automatically enforcing tenant isolation on all queries.
//
// Usage:
//
//	import (
//	    "database/sql"
//	    sqladapter "github.com/abhipray-cpu/tenantkit/adapters/sql"
//	    _ "github.com/lib/pq"  // User chooses their driver
//	)
//
//	db, err := sql.Open("postgres", dsn)
//	storage := sqladapter.New(db)
//	defer storage.Close()
//
// All queries executed through storage will automatically have tenant_id
// filtering injected, ensuring complete tenant isolation at the database level.
package sqladapter
