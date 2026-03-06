# sqlx Adapter for TenantKit

Multi-tenant database access with automatic tenant isolation for [jmoiron/sqlx](https://github.com/jmoiron/sqlx).

## Features

- **Automatic Tenant Isolation**: All database queries are automatically scoped to the current tenant
- **Full sqlx Support**: Wraps sqlx.DB and sqlx.Tx with complete API compatibility  
- **Transaction Support**: Tenant context propagates through transactions
- **Named Queries**: Full support for sqlx named queries and struct scanning
- **Manual Control**: Skip enforcement when needed (migrations, admin operations)
- **Production-Ready**: Designed for PostgreSQL and MySQL in production

## Installation

```bash
go get github.com/abhipray-cpu/tenantkit/adapters/sqlx
```

**Important**: This is a separate module. Only sqlx users need to import it - GORM users will never download sqlx.

## Quick Start

```go
package main

import (
    "context"
    "log"
    
    "github.com/abhipray-cpu/tenantkit/domain"
    tenantsqlx "github.com/abhipray-cpu/tenantkit/adapters/sqlx"
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq" // PostgreSQL driver
)

type User struct {
    ID       int    `db:"id"`
    Name     string `db:"name"`
    Email    string `db:"email"`
    TenantID string `db:"tenant_id"`
}

func main() {
    // Open database connection
    db, err := tenantsqlx.Connect("postgres", "user=postgres dbname=myapp sslmode=disable", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Create tenant context
    tenantCtx, err := domain.NewContext("tenant-123", "user-456", "req-789")
    if err != nil {
        log.Fatal(err)
    }
    ctx := tenantCtx.ToGoContext(context.Background())
    
    // INSERT - tenant_id automatically added
    _, err = db.ExecContext(ctx, 
        "INSERT INTO users (name, email) VALUES (?, ?)",
        "Alice", "alice@example.com")
    
    // SELECT - automatically filtered by tenant_id
    var users []User
    err = db.SelectContext(ctx, &users, "SELECT * FROM users")
    // Returns only users for tenant-123
    
    // UPDATE - automatically scoped to tenant
    _, err = db.ExecContext(ctx,
        "UPDATE users SET email = ? WHERE name = ?",
        "alice.new@example.com", "Alice")
    // Only updates users for tenant-123
    
    // DELETE - automatically scoped to tenant
    _, err = db.ExecContext(ctx,
        "DELETE FROM users WHERE name = ?", "Alice")
    // Only deletes users for tenant-123
}
```

## Configuration

### Custom Tenant Column

```go
cfg := &tenantsqlx.Config{
    TenantColumn: "org_id", // Use org_id instead of tenant_id
}

db, err := tenantsqlx.Connect("postgres", dsn, cfg)
```

### Skip Tables

Some tables don't need tenant isolation (migrations, system config, etc.):

```go
cfg := &tenantsqlx.Config{
    SkipTables: []string{"migrations", "system_config"},
}

db, err := tenantsqlx.Connect("postgres", dsn, cfg)
```

## Advanced Usage

### Transactions

Tenant context propagates through transactions:

```go
tx, err := db.BeginTxx(ctx, nil)
if err != nil {
    log.Fatal(err)
}
defer tx.Rollback()

// All queries in transaction are tenant-scoped
_, err = tx.ExecContext(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", "Bob", "bob@example.com")
_, err = tx.ExecContext(ctx, "INSERT INTO orders (user_id, total) VALUES (?, ?)", 1, 99.99)

if err := tx.Commit(); err != nil {
    log.Fatal(err)
}
```

### Named Queries

Full support for sqlx named queries:

```go
user := User{
    Name:     "Charlie",
    Email:    "charlie@example.com",
    TenantID: "tenant-123", // Must include tenant_id in struct
}

_, err := db.NamedExecContext(ctx,
    "INSERT INTO users (name, email, tenant_id) VALUES (:name, :email, :tenant_id)",
    user)
```

### Struct Scanning

```go
// Single record
var user User
err := db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = ?", 1)

// Multiple records
var users []User
err := db.SelectContext(ctx, &users, "SELECT * FROM users WHERE active = true")
```

### Skip Tenant Enforcement

For admin operations or migrations:

```go
// Method 1: Skip specific query
skipCtx := tenantsqlx.SkipTenant(ctx)
var allUsers []User
db.SelectContext(skipCtx, &allUsers, "SELECT * FROM users")
// Returns users from ALL tenants

// Method 2: Create DB instance without enforcement
adminDB := db.WithoutTenant()
adminDB.SelectContext(ctx, &allUsers, "SELECT * FROM users")
// Returns users from ALL tenants
```

## How It Works

### Query Rewriting

The adapter automatically rewrites SQL queries:

```go
// Your code:
db.SelectContext(ctx, &users, "SELECT * FROM users WHERE active = true")

// Actual query executed:
// SELECT * FROM users WHERE tenant_id = ? AND active = true
// Args: ["tenant-123"]
```

```go
// Your code:
db.ExecContext(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "alice@example.com")

// Actual query executed:
// INSERT INTO users (name, email, tenant_id) VALUES (?, ?, ?)
// Args: ["Alice", "alice@example.com", "tenant-123"]
```

```go
// Your code:
db.ExecContext(ctx, "UPDATE users SET email = ? WHERE name = ?", "new@example.com", "Alice")

// Actual query executed:
// UPDATE users SET email = ? WHERE tenant_id = ? AND name = ?
// Args: ["new@example.com", "tenant-123", "Alice"]
```

### Tenant Context Extraction

The adapter extracts tenant ID from domain.Context:

```go
tenantCtx, err := domain.NewContext("tenant-123", "user-456", "req-789")
ctx := tenantCtx.ToGoContext(context.Background())

// tenant_id is automatically extracted and used in all queries
```

## Best Practices

### 1. Always Use Context

```go
// ✅ Good - uses context
db.SelectContext(ctx, &users, "SELECT * FROM users")

// ❌ Bad - no context, will fail
db.Select(&users, "SELECT * FROM users")
```

### 2. Include tenant_id in Schema

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    tenant_id TEXT NOT NULL
);

CREATE INDEX idx_users_tenant_id ON users(tenant_id);
```

### 3. Named Queries Require tenant_id

For named queries, include `tenant_id` in your struct:

```go
type User struct {
    ID       int    `db:"id"`
    Name     string `db:"name"`
    Email    string `db:"email"`
    TenantID string `db:"tenant_id"` // Required for named queries
}
```

### 4. Use Skip Carefully

Only skip tenant enforcement for:
- Database migrations
- System-wide configuration
- Admin operations viewing all tenants

```go
// ✅ Good - skip for migrations
skipCtx := tenantsqlx.SkipTenant(ctx)
db.ExecContext(skipCtx, "CREATE TABLE migrations (...)")

// ❌ Bad - never skip for regular queries
skipCtx := tenantsqlx.SkipTenant(ctx)
db.SelectContext(skipCtx, &users, "SELECT * FROM users") // Exposes all tenants!
```

## Limitations

### 1. Simple SQL Only

The query rewriter handles standard SQL patterns:
- SELECT with WHERE, ORDER BY, LIMIT, etc.
- INSERT with column list
- UPDATE with SET and WHERE
- DELETE with WHERE

Complex SQL may require manual tenant handling:

```go
// ✅ Works
db.QueryContext(ctx, "SELECT * FROM users WHERE active = true")

// ⚠️ May not work correctly
db.QueryContext(ctx, "SELECT * FROM (SELECT * FROM users) AS subquery")
// Use SkipTenant and add tenant_id manually for complex queries
```

### 2. Named Query Injection

Named queries (`NamedExecContext`) require you to include `tenant_id` in the struct or map. The adapter validates the tenant context but doesn't auto-inject for named queries.

### 3. Raw SQL Functions

SQLite-specific or database-specific functions may not be portable:

```go
// ✅ Portable
db.QueryContext(ctx, "SELECT * FROM users WHERE created_at > ?", time.Now())

// ⚠️ SQLite-specific
db.QueryContext(ctx, "SELECT * FROM users WHERE datetime(created_at) > datetime('now')")
```

## Testing

The adapter includes comprehensive tests:

```bash
# Unit tests (fast, no database required)
go test -v

# Integration tests (requires database)
go test -v -tags=integration
```

## Performance

### Overhead

- Query rewriting: < 0.1ms per query
- Tenant context extraction: < 0.01ms per query
- Total overhead: < 0.15ms per query

### Indexes

Always create indexes on `tenant_id` columns:

```sql
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_orders_tenant_id ON orders(tenant_id);
```

### Connection Pooling

The adapter doesn't interfere with sqlx connection pooling:

```go
db, err := tenantsqlx.Connect("postgres", dsn, nil)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5 * time.Minute)
```

## Database Support

Designed for production use with:
- **PostgreSQL** (recommended)
- **MySQL/MariaDB**
- **SQLite** (for testing only - has concurrency limitations)

## Troubleshooting

### Error: "context does not contain tenant information"

```go
// ❌ Wrong - using stdlib context
ctx := context.Background()
db.QueryContext(ctx, "SELECT * FROM users") // ERROR

// ✅ Correct - using domain.Context
tenantCtx, _ := domain.NewContext("tenant-123", "user-456", "req-789")
ctx := tenantCtx.ToGoContext(context.Background())
db.QueryContext(ctx, "SELECT * FROM users") // OK
```

### Error: "not enough args to execute query"

This usually means the query rewriter failed. Use `SkipTenant` and add `tenant_id` manually:

```go
skipCtx := tenantsqlx.SkipTenant(ctx)
tenantCtx, _ := domain.FromGoContext(ctx)
tenantID := tenantCtx.TenantID().Value()

db.QueryContext(skipCtx, 
    "SELECT * FROM complex_query WHERE tenant_id = ?", 
    tenantID)
```

## Examples

See [examples/](../../examples/) directory for complete applications:
- Basic CRUD operations
- Multi-tenant REST API
- Transaction handling
- Admin operations

## License

MIT License - see LICENSE file for details
