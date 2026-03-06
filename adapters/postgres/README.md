# PostgreSQL RLS Adapter

This adapter provides **PostgreSQL-specific Row-Level Security (RLS)** features for multi-tenancy. It is a specialized adapter focused on PostgreSQL's native RLS capabilities.

## Purpose

PostgreSQL offers built-in Row-Level Security (RLS), which is the most secure and performant way to implement multi-tenancy at the database level. This adapter helps you configure and manage RLS policies for your tenants.

**Note**: For general PostgreSQL database operations (CRUD, transactions, etc.), use the standard `adapters/sql` adapter with a PostgreSQL driver. This adapter is specifically for RLS setup and management.

## What is Row-Level Security?

Row-Level Security (RLS) is a PostgreSQL feature that restricts which rows can be accessed or modified based on the current database user or session variables. It works at the database engine level, providing:

- **Security**: Enforced by PostgreSQL, cannot be bypassed
- **Performance**: Native database filtering, no application overhead
- **Simplicity**: Set once, applies to all queries automatically

## Features

- ✅ Enable/disable RLS on tables
- ✅ Create tenant-scoped RLS policies
- ✅ Automatic policy naming and management
- ✅ Support for all DML operations (SELECT, INSERT, UPDATE, DELETE)
- ✅ Policy testing and verification
- ✅ Comprehensive test coverage

## Installation

```bash
go get github.com/abhipray-cpu/tenantkit/adapters/postgres
```

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "log"
    
    "github.com/jackc/pgx/v5"
    "github.com/abhipray-cpu/tenantkit/adapters/postgres"
)

func main() {
    // Connect to PostgreSQL
    conn, err := pgx.Connect(context.Background(), "postgres://user:pass@localhost/db")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close(context.Background())
    
    // Create RLS manager
    rlsConfig := postgres.RLSConfig{
        TenantIDColumn: "tenant_id", // Column name in your tables
        RLSPolicyName:  "tenant_rls", // Policy name prefix
    }
    
    rlsManager := postgres.NewRLSManager(conn, rlsConfig)
    
    // Enable RLS on a table
    ctx := context.Background()
    err = rlsManager.EnableRLS(ctx, "users")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("RLS enabled successfully!")
}
```

### Setting Up Tenant Context

```go
// Set the tenant context for the session
tenantID := "tenant-123"
err := rlsManager.SetTenantContext(ctx, tenantID)
if err != nil {
    log.Fatal(err)
}

// Now all queries in this session are automatically scoped to tenant-123
rows, err := conn.Query(ctx, "SELECT * FROM users")
// Only returns users for tenant-123, enforced by PostgreSQL
```

### Enabling RLS on Multiple Tables

```go
tables := []string{"users", "orders", "products", "invoices"}

for _, table := range tables {
    err := rlsManager.EnableRLS(ctx, table)
    if err != nil {
        log.Fatalf("Failed to enable RLS on %s: %v", table, err)
    }
    log.Printf("RLS enabled on table: %s", table)
}
```

### Testing RLS Policies

```go
// Verify that the policy works correctly
tenantID := "tenant-456"
isWorking, err := rlsManager.TestPolicy(ctx, "users", tenantID)
if err != nil {
    log.Fatal(err)
}

if !isWorking {
    log.Fatal("RLS policy is not working correctly!")
}

log.Println("RLS policy verified successfully!")
```

### Disabling RLS (for maintenance)

```go
// Disable RLS temporarily (e.g., for migrations)
err := rlsManager.DisableRLS(ctx, "users")
if err != nil {
    log.Fatal(err)
}

// Do maintenance work...

// Re-enable RLS
err = rlsManager.EnableRLS(ctx, "users")
if err != nil {
    log.Fatal(err)
}
```

## Configuration

### RLSConfig Options

```go
type RLSConfig struct {
    // TenantIDColumn is the name of the tenant_id column in your tables
    // Default: "tenant_id"
    TenantIDColumn string
    
    // RLSPolicyName is the prefix for RLS policy names
    // Default: "tenant_rls"
    // Actual policy names will be: {prefix}_{table_name}
    RLSPolicyName string
}
```

## How It Works

When you enable RLS on a table:

1. **Enable RLS**: `ALTER TABLE {table} ENABLE ROW LEVEL SECURITY`
2. **Create Policy**: Creates policies for SELECT, INSERT, UPDATE, DELETE
3. **Tenant Filtering**: All policies check `tenant_id = current_setting('app.tenant_id')`

Example policy created:

```sql
CREATE POLICY tenant_rls_users_select ON users
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::text);

CREATE POLICY tenant_rls_users_insert ON users
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::text);

-- Similar for UPDATE and DELETE
```

## Database Schema Requirements

Your tables must have a `tenant_id` column (or whatever you configure):

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    tenant_id TEXT NOT NULL,  -- Required for RLS
    name TEXT,
    email TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_tenant_id ON users(tenant_id);
```

**Important**: Always index the `tenant_id` column for performance!

## Complete Example: Multi-Tenant Application

```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "github.com/jackc/pgx/v5"
    "github.com/abhipray-cpu/tenantkit/adapters/postgres"
    "github.com/abhipray-cpu/tenantkit/tenantkit/domain"
)

func main() {
    // Setup PostgreSQL connection
    conn, _ := pgx.Connect(context.Background(), "postgres://localhost/myapp")
    defer conn.Close(context.Background())
    
    // Setup RLS
    rlsManager := postgres.NewRLSManager(conn, postgres.RLSConfig{})
    
    // Enable RLS on all tables (one-time setup)
    tables := []string{"users", "posts", "comments"}
    for _, table := range tables {
        rlsManager.EnableRLS(context.Background(), table)
    }
    
    // HTTP handler
    http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
        // Extract tenant from request (using tenantkit middleware)
        ctx := r.Context()
        tenantCtx, _ := domain.FromGoContext(ctx)
        tenantID := tenantCtx.TenantID().Value()
        
        // Set tenant context in database session
        rlsManager.SetTenantContext(ctx, tenantID)
        
        // Query users - automatically scoped to tenant!
        rows, _ := conn.Query(ctx, "SELECT * FROM users")
        defer rows.Close()
        
        // Process rows...
        // All users returned belong to the tenant automatically
    })
    
    http.ListenAndServe(":8080", nil)
}
```

## Integration with TenantKit

This adapter works seamlessly with other TenantKit components:

```go
import (
    "github.com/abhipray-cpu/tenantkit/tenantkit"
    httpstd "github.com/abhipray-cpu/tenantkit/adapters/http-stdlib"
    "github.com/abhipray-cpu/tenantkit/adapters/postgres"
    "github.com/abhipray-cpu/tenantkit/adapters/sql"
)

// 1. Use http-stdlib for tenant resolution
resolver := httpstd.NewSubdomainResolver()
middleware := httpstd.NewMiddleware(resolver)

// 2. Use sql adapter for general database operations
storage := sqladapter.NewStorage(db, enforcer)

// 3. Use postgres adapter for RLS setup
rlsManager := postgres.NewRLSManager(conn, postgres.RLSConfig{})
```

## Performance Considerations

### Pros of RLS:
- ✅ **Database-level security**: Cannot be bypassed by application bugs
- ✅ **Zero application overhead**: Filtering happens in PostgreSQL
- ✅ **Automatic**: Works with any query, no code changes needed
- ✅ **Fast**: Native PostgreSQL index usage

### Best Practices:
- Always index `tenant_id` columns
- Use connection pooling with tenant context
- Test policies thoroughly before production
- Monitor query performance (EXPLAIN ANALYZE)

### When to Use RLS:
- ✅ High security requirements
- ✅ PostgreSQL is your database
- ✅ Simple tenant isolation model
- ✅ Performance is critical

### When NOT to Use RLS:
- ❌ Using MySQL, SQLite, or other databases
- ❌ Complex tenant hierarchies
- ❌ Frequent policy changes
- ❌ Need flexibility in tenant filtering

## Testing

Run the test suite:

```bash
cd adapters/postgres
go test -v
```

Tests cover:
- RLS enabling/disabling
- Policy creation
- Tenant context setting
- Policy verification
- Error handling

## Comparison with Other Approaches

| Approach | Security | Performance | Flexibility | Complexity |
|----------|----------|-------------|-------------|------------|
| **RLS (this adapter)** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Query Rewriting** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Application Logic** | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |

## Limitations

- **PostgreSQL Only**: RLS is a PostgreSQL feature, won't work with MySQL, SQLite, etc.
- **Session-Based**: Requires setting tenant context per connection/transaction
- **Policy Complexity**: Complex policies can impact query planning
- **No Cross-Tenant Queries**: RLS enforces strict isolation

## FAQ

**Q: Can I use this with the `sql` adapter?**  
A: Yes! Use `sql` adapter for queries and this adapter for RLS setup.

**Q: Does this work with connection pooling?**  
A: Yes, but set tenant context at the start of each request/transaction.

**Q: Can I customize the policies?**  
A: Currently creates standard policies. For custom policies, use SQL directly.

**Q: What if I need cross-tenant queries (e.g., admin)?**  
A: Disable RLS for that session or use a superuser connection.

**Q: How do I migrate existing data?**  
A: Add `tenant_id` column, backfill data, then enable RLS.

## Related Documentation

- [TenantKit Architecture](../../docs/architecture.md)
- [SQL Adapter](../sql/README.md) - For general database operations
- [Query Enforcement](../../docs/query-enforcement.md) - Alternative approach
- [PostgreSQL RLS Documentation](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)

## Support

For issues or questions:
- GitHub Issues: https://github.com/abhipray-cpu/tenantkit/issues
- Documentation: https://github.com/abhipray-cpu/tenantkit/docs

## License

Same as TenantKit core library.
