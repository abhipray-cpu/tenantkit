# Usage Guide

This comprehensive guide covers all aspects of using TenantKit in your Go applications.

## Table of Contents

- [Core Concepts](#core-concepts)
- [Installation](#installation)
- [Configuration](#configuration)
- [Context Management](#context-management)
- [Database Operations](#database-operations)
- [Transactions](#transactions)
- [Bypassing Tenant Filtering](#bypassing-tenant-filtering)
- [Placeholder Styles](#placeholder-styles)
- [Rate Limiting](#rate-limiting)
- [Quota Management](#quota-management)
- [Error Handling](#error-handling)
- [Logging](#logging)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Core Concepts

### Tenant Isolation Model

TenantKit uses **row-level multi-tenancy** where all tenants share the same database and tables, but each row is associated with a specific tenant via a `tenant_id` column.

```
┌─────────────────────────────────────────────────────┐
│                    users table                       │
├────────────┬─────────────┬─────────┬───────────────┤
│ tenant_id  │ id          │ name    │ email         │
├────────────┼─────────────┼─────────┼───────────────┤
│ acme-corp  │ 1           │ Alice   │ alice@acme    │
│ acme-corp  │ 2           │ Bob     │ bob@acme      │
│ beta-inc   │ 3           │ Carol   │ carol@beta    │
│ beta-inc   │ 4           │ Dave    │ dave@beta     │
└────────────┴─────────────┴─────────┴───────────────┘
```

When tenant `acme-corp` queries users, they only see Alice and Bob.

### How It Works

TenantKit wraps `*sql.DB` and transparently rewrites every SQL query to include a `WHERE tenant_id = ?` clause. Your application code stays clean — no manual tenant filtering required.

```
Your query:    SELECT * FROM users WHERE active = true
TenantKit:     SELECT * FROM users WHERE tenant_id = $1 AND active = true
```

### Database Schema Requirements

Your tables must include a `tenant_id` column:

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Recommended: Create an index for efficient filtering
CREATE INDEX idx_users_tenant_id ON users(tenant_id);

-- Recommended: Composite index for common query patterns
CREATE INDEX idx_users_tenant_id_email ON users(tenant_id, email);
```

---

## Installation

```bash
go get github.com/abhipray-cpu/tenantkit/tenantkit
```

---

## Configuration

### Config Struct

```go
type Config struct {
    // TenantTables is the list of tables that require tenant filtering.
    // Only queries touching these tables will be transformed.
    TenantTables []string

    // TenantColumn is the column name for tenant ID (default: "tenant_id").
    TenantColumn string

    // Logger is an optional structured logger. If nil, logging is disabled.
    Logger *slog.Logger
}
```

### Example Configurations

#### Minimal Configuration

```go
import "github.com/abhipray-cpu/tenantkit/tenantkit"

db, _ := sql.Open("postgres", connStr)

wrappedDB, err := tenantkit.Wrap(db, tenantkit.Config{
    TenantTables: []string{"users", "orders", "products"},
})
```

#### Full Configuration

```go
wrappedDB, err := tenantkit.Wrap(db, tenantkit.Config{
    TenantTables: []string{"users", "orders", "products"},
    TenantColumn: "org_id",              // Custom column name
    Logger:       slog.Default(),        // Enable structured logging
})
if err != nil {
    log.Fatal(err)
}
defer wrappedDB.Close()
```

#### Must-Wrap (Panics on Error)

```go
wrappedDB := tenantkit.MustWrap(db, tenantkit.Config{
    TenantTables: []string{"users", "orders"},
})
```

---

## Context Management

### Setting Tenant Context

Every database operation requires a tenant context:

```go
import "github.com/abhipray-cpu/tenantkit/tenantkit"

// Set tenant in context
ctx := tenantkit.WithTenant(context.Background(), "acme-corp")

// Use with database operations
rows, err := wrappedDB.Query(ctx, "SELECT * FROM users")
// Executes: SELECT * FROM users WHERE tenant_id = $1  [args: "acme-corp"]
```

### Retrieving Tenant from Context

```go
tenantID, ok := tenantkit.GetTenant(ctx)
if !ok {
    return fmt.Errorf("tenant context required")
}
```

### HTTP Middleware Integration

```go
// Using stdlib middleware
import httpstdlib "github.com/abhipray-cpu/tenantkit/adapters/http-stdlib"

mw := httpstdlib.TenantMiddleware(httpstdlib.Config{
    HeaderName: "X-Tenant-ID",
    Required:   true,
})

http.Handle("/api/", mw(yourHandler))
```

```go
// Using Gin middleware
import httpgin "github.com/abhipray-cpu/tenantkit/adapters/http-gin"

router := gin.Default()
router.Use(httpgin.TenantMiddleware(httpgin.Config{
    HeaderName: "X-Tenant-ID",
}))
```

Other supported frameworks: Echo, Chi, Fiber.

---

## Database Operations

### Query Operations

```go
// Single row query
var name string
err := wrappedDB.QueryRow(ctx,
    "SELECT name FROM users WHERE id = $1", userID,
).Scan(&name)

// Multiple rows query
rows, err := wrappedDB.Query(ctx,
    "SELECT id, name, email FROM users WHERE active = $1", true,
)
if err != nil {
    return err
}
defer rows.Close()

for rows.Next() {
    var id int
    var name, email string
    if err := rows.Scan(&id, &name, &email); err != nil {
        return err
    }
    // Process row...
}
```

### Insert Operations

TenantKit automatically adds the `tenant_id` column and value to INSERT queries:

```go
// tenant_id is injected automatically
result, err := wrappedDB.Exec(ctx,
    "INSERT INTO users (name, email) VALUES ($1, $2)",
    "Alice", "alice@example.com",
)
// Executes: INSERT INTO users (tenant_id, name, email) VALUES ($1, $2, $3)
```

> **Note:** INSERT queries must use explicit column lists. `INSERT INTO users VALUES (...)` is not supported.

### Update Operations

```go
result, err := wrappedDB.Exec(ctx,
    "UPDATE users SET name = $1 WHERE id = $2",
    "Alice Updated", userID,
)
// Executes: UPDATE users SET name = $1 WHERE tenant_id = $2 AND id = $3

rowsAffected, _ := result.RowsAffected()
fmt.Printf("Updated %d rows\n", rowsAffected)
```

### Delete Operations

```go
result, err := wrappedDB.Exec(ctx,
    "DELETE FROM users WHERE id = $1",
    userID,
)
// Executes: DELETE FROM users WHERE tenant_id = $1 AND id = $2
```

---

## Transactions

Transactions maintain tenant isolation throughout their lifecycle:

```go
// Begin a tenant-aware transaction
tx, err := wrappedDB.Begin(ctx, nil)
if err != nil {
    return err
}

// Use defer for cleanup
defer func() {
    if p := recover(); p != nil {
        tx.Rollback()
        panic(p)
    }
}()

// All operations within the transaction are tenant-filtered
_, err = tx.Exec(ctx,
    "INSERT INTO orders (product_id, quantity) VALUES ($1, $2)",
    productID, quantity,
)
if err != nil {
    tx.Rollback()
    return err
}

_, err = tx.Exec(ctx,
    "UPDATE inventory SET stock = stock - $1 WHERE product_id = $2",
    quantity, productID,
)
if err != nil {
    tx.Rollback()
    return err
}

// Commit the transaction
return tx.Commit()
```

### Transaction Helper Pattern

```go
func WithTx(ctx context.Context, db *tenantkit.DB, fn func(*tenantkit.Tx) error) error {
    tx, err := db.Begin(ctx, nil)
    if err != nil {
        return err
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)
        }
    }()

    if err := fn(tx); err != nil {
        tx.Rollback()
        return err
    }

    return tx.Commit()
}

// Usage
err := WithTx(ctx, wrappedDB, func(tx *tenantkit.Tx) error {
    _, err := tx.Exec(ctx, "INSERT INTO orders ...")
    return err
})
```

---

## Bypassing Tenant Filtering

For administrative or cross-tenant queries, use `WithoutTenantFiltering`:

```go
// ⚠️  Use with extreme caution — bypasses all tenant isolation!
adminCtx := tenantkit.WithoutTenantFiltering(context.Background())

// Count users across ALL tenants
rows, err := wrappedDB.Query(adminCtx,
    "SELECT tenant_id, COUNT(*) FROM users GROUP BY tenant_id",
)
```

---

## Placeholder Styles

TenantKit auto-detects placeholder style from the database driver:

| Driver | Style | Example |
|--------|-------|---------|
| PostgreSQL (`pgx`, `lib/pq`) | `$1, $2, $3` | Auto-detected |
| SQLite, MySQL | `?, ?, ?` | Auto-detected or use `WrapWithStyle` |

For explicit control:

```go
// SQLite / MySQL
wrappedDB, err := tenantkit.WrapWithStyle(db, tenantkit.Config{
    TenantTables: []string{"users"},
}, tenantkit.PlaceholderQuestion)

// PostgreSQL (default)
wrappedDB, err := tenantkit.WrapWithStyle(db, tenantkit.Config{
    TenantTables: []string{"users"},
}, tenantkit.PlaceholderDollar)
```

---

## Rate Limiting

TenantKit provides rate limiting adapters through the `ports.Limiter` interface.

### In-Memory Rate Limiter

Best for single-instance deployments. Supports token bucket, sliding window, and fixed window algorithms:

```go
import limitermemory "github.com/abhipray-cpu/tenantkit/adapters/limiter-memory"

// Token bucket: 100 requests/sec with burst of 10
limiter, err := limitermemory.NewTokenBucketLimiter(100, 10)

// Sliding window: 1000 requests per minute
limiter, err := limitermemory.NewSlidingWindowLimiter(1000, time.Minute)

// Fixed window: 500 requests per minute
limiter, err := limitermemory.NewFixedWindowLimiter(500, time.Minute)
```

### Redis Rate Limiter

Best for distributed deployments. Supports standalone, Sentinel, and Cluster Redis:

```go
import limiterredis "github.com/abhipray-cpu/tenantkit/adapters/limiter-redis"
import "github.com/redis/go-redis/v9"

// Standalone Redis
rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

limiter, err := limiterredis.New(rdb, limiterredis.Config{
    Algorithm: "token_bucket", // or "fixed_window", "sliding_window"
    Limit:     100,
    Window:    time.Minute,
})
```

```go
// Redis Cluster
rdb := redis.NewClusterClient(&redis.ClusterOptions{
    Addrs: []string{"node1:6379", "node2:6379", "node3:6379"},
})
limiter, err := limiterredis.New(rdb, limiterredis.Config{...})
```

### Using the Limiter

```go
allowed, err := limiter.Allow(ctx, tenantID)
if err != nil {
    log.Printf("rate limiter error: %v", err)
} else if !allowed {
    http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
    return
}

// Check remaining quota
remaining, _ := limiter.Remaining(ctx, tenantID)
w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
```

---

## Quota Management

TenantKit provides quota management adapters through the `ports.QuotaManager` interface. This prevents the **noisy neighbor problem** where one tenant consumes disproportionate resources.

### In-Memory Quota Manager

Best for development and single-instance deployments:

```go
import quotamemory "github.com/abhipray-cpu/tenantkit/adapters/quota-memory"

qm := quotamemory.NewInMemoryQuotaManager()

// Set limits
ctx := tenantkit.WithTenant(context.Background(), "acme-corp")
qm.SetLimit(ctx, "api_requests", 10000)
qm.SetLimit(ctx, "storage_bytes", 1*1024*1024*1024) // 1 GB
```

### Redis Quota Manager

Best for production distributed deployments. Uses Lua scripts for atomicity.
Supports standalone, Sentinel, and Cluster Redis:

```go
import quotaredis "github.com/abhipray-cpu/tenantkit/adapters/quota-redis"
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

config := quotaredis.DefaultConfig()
config.Limits = map[string]int64{
    "api_requests": 10000,
    "storage_bytes": 1 * 1024 * 1024 * 1024, // 1 GB
}

qm, err := quotaredis.NewRedisQuotaManager(rdb, config)
```

### Checking and Consuming Quotas

```go
// Check if within quota
allowed, err := qm.CheckQuota(ctx, "api_requests", 1)
if !allowed {
    return fmt.Errorf("quota exceeded")
}

// Consume quota and get remaining
remaining, err := qm.ConsumeQuota(ctx, "api_requests", 1)
fmt.Printf("Remaining: %d\n", remaining)

// Get current usage
used, limit, err := qm.GetUsage(ctx, "api_requests")
fmt.Printf("Used: %d / %d\n", used, limit)

// Reset quota (e.g., at billing period start)
qm.ResetQuota(ctx, "api_requests")
```

---

## Error Handling

### Sentinel Errors

```go
import "github.com/abhipray-cpu/tenantkit/tenantkit"

if err != nil {
    switch {
    case errors.Is(err, tenantkit.ErrMissingTenant):
        // Tenant not set in context
        return fmt.Errorf("tenant context required")

    case errors.Is(err, tenantkit.ErrInvalidTenant):
        // Tenant ID format is invalid
        return fmt.Errorf("invalid tenant ID")

    case errors.Is(err, tenantkit.ErrQueryParsing):
        // Query could not be parsed for tenant injection
        return fmt.Errorf("unsupported query format")

    default:
        return fmt.Errorf("database error: %w", err)
    }
}
```

### TenantError Type

For detailed error context:

```go
var tenantErr *tenantkit.TenantError
if errors.As(err, &tenantErr) {
    log.Printf("Query: %s, Tables: %v, Error: %v",
        tenantErr.Query, tenantErr.Tables, tenantErr.Err)
}
```

---

## Logging

TenantKit uses `log/slog` for structured logging. Logging is **disabled by default** — the library never writes to stdout unless you explicitly configure a logger.

```go
// Enable logging with default logger
wrappedDB, _ := tenantkit.Wrap(db, tenantkit.Config{
    TenantTables: []string{"users"},
    Logger:       slog.Default(),
})

// Use a custom logger
logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

wrappedDB, _ := tenantkit.Wrap(db, tenantkit.Config{
    TenantTables: []string{"users"},
    Logger:       logger,
})
```

---

## Best Practices

### 1. Always Set Tenant Context Early

```go
// Good: Set tenant in middleware
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := r.Header.Get("X-Tenant-ID")
        if tenantID == "" {
            http.Error(w, "tenant required", http.StatusBadRequest)
            return
        }
        ctx := tenantkit.WithTenant(r.Context(), tenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 2. Use Indexes on tenant_id

```sql
-- Always index tenant_id
CREATE INDEX idx_tablename_tenant_id ON tablename(tenant_id);

-- Consider composite indexes for common queries
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);
```

### 3. List All Tenant Tables

```go
// Good: Explicit list of ALL tables with tenant_id
wrappedDB, _ := tenantkit.Wrap(db, tenantkit.Config{
    TenantTables: []string{
        "users", "orders", "products",
        "invoices", "payments", "audit_log",
    },
})

// Bad: Missing tables = no tenant filtering for those tables!
wrappedDB, _ := tenantkit.Wrap(db, tenantkit.Config{
    TenantTables: []string{"users"}, // ⚠️  "orders" queries won't be filtered!
})
```

### 4. Use Explicit Column Lists in INSERT

```go
// Good: Explicit columns
wrappedDB.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)", name, email)

// Bad: No column list — TenantKit can't inject tenant_id
wrappedDB.Exec(ctx, "INSERT INTO users VALUES ($1, $2, $3)", id, name, email)
```

### 5. Access Raw DB When Needed

```go
// For operations that shouldn't be tenant-filtered
rawDB := wrappedDB.Raw()
rawDB.SetMaxOpenConns(100)
rawDB.SetMaxIdleConns(10)
rawDB.SetConnMaxLifetime(30 * time.Minute)
```

---

## Troubleshooting

### Query Not Being Transformed

**Problem**: Queries aren't being filtered by tenant.

**Checklist**:
1. Is the table listed in `TenantTables`?
2. Is `WithTenant()` called before the query?
3. Is the table name spelled correctly (case-sensitive)?

```go
// Debug: Check if tenant is set
tenantID, ok := tenantkit.GetTenant(ctx)
if !ok {
    log.Println("WARNING: No tenant in context!")
}
```

### "TenantTables cannot be empty" Error

**Problem**: `Wrap()` returns an error.

**Solution**: Provide at least one table:

```go
wrappedDB, err := tenantkit.Wrap(db, tenantkit.Config{
    TenantTables: []string{"users"}, // Required!
})
```

### INSERT Fails with "explicit column lists" Error

**Problem**: INSERT queries fail.

**Solution**: Always use explicit column lists:

```go
// Works
"INSERT INTO users (name, email) VALUES ($1, $2)"

// Fails
"INSERT INTO users VALUES ($1, $2, $3)"
```

### Slow Query Performance

**Problem**: Queries are slower than expected.

**Solution**: Ensure proper indexing:

```sql
-- Check if index exists
SELECT indexname FROM pg_indexes WHERE tablename = 'your_table';

-- Add index if missing
CREATE INDEX IF NOT EXISTS idx_your_table_tenant_id ON your_table(tenant_id);
```

TenantKit includes a built-in LRU query cache (1000 entries) that caches transformed queries, so the same query pattern is only parsed once.

---

## Next Steps

- [Architecture Guide](./architecture.md) — Understand the internals, C4 diagrams, and security model
- [README](../README.md) — Quick start and adapter reference
