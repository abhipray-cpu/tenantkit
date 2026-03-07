# TenantKit

<p align="center">
  <img src="icon.png" alt="TenantKit" width="480" />
</p>

[![Go Reference](https://pkg.go.dev/badge/github.com/abhipray-cpu/tenantkit.svg)](https://pkg.go.dev/github.com/abhipray-cpu/tenantkit)
[![CI](https://github.com/abhipray-cpu/tenantkit/actions/workflows/ci.yml/badge.svg)](https://github.com/abhipray-cpu/tenantkit/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/abhipray-cpu/tenantkit)](https://goreportcard.com/report/github.com/abhipray-cpu/tenantkit)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![codecov](https://codecov.io/gh/abhipray-cpu/tenantkit/branch/main/graph/badge.svg)](https://codecov.io/gh/abhipray-cpu/tenantkit)

**TenantKit** is a transparent multi-tenancy library for Go that provides automatic tenant isolation at the database level. It wraps the standard `database/sql` package and transparently injects tenant conditions into all SQL queries — no ORM required.

## Features

- �� **Automatic Tenant Isolation** — All queries are automatically filtered by tenant
- ⚡ **High Performance** — LRU query cache with `sync.Pool` optimizations
- 🔄 **Transparent** — Works with existing `database/sql` code
- 🎯 **Zero Dependencies** — Core library has no external dependencies
- 📊 **Rate Limiting** — Per-tenant rate limiting (memory or Redis-backed)
- 🛡️ **Quota Management** — Noisy-neighbor protection with configurable limits
- �� **Extensible** — Adapters for Gin, Echo, Chi, Fiber, and net/http
- 🧪 **Well Tested** — Comprehensive test suite with race detection

## Quick Start

### Installation

```bash
go get github.com/abhipray-cpu/tenantkit/tenantkit
```

### Basic Usage (PostgreSQL)

```go
package main

import (
    "context"
    "database/sql"
    "log"

    "github.com/abhipray-cpu/tenantkit/tenantkit"
    _ "github.com/lib/pq"
)

func main() {
    // Open your regular database connection
    db, err := sql.Open("postgres", "postgres://localhost/myapp?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Wrap it with TenantKit — tell it which tables have tenant_id
    wrappedDB, err := tenantkit.Wrap(db, tenantkit.Config{
        TenantTables: []string{"users", "orders", "products"},
        TenantColumn: "tenant_id",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create a context with tenant information
    ctx := tenantkit.WithTenant(context.Background(), "acme-corp")

    // All queries are now automatically filtered by tenant!
    // This query:  SELECT * FROM users WHERE name = $1
    // Becomes:     SELECT * FROM users WHERE users.tenant_id = $2 AND (name = $1)
    rows, err := wrappedDB.Query(ctx, "SELECT * FROM users WHERE name = $1", "John")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    // Process rows as usual...
}
```

### MySQL / SQLite

For databases that use `?` placeholders, use `WrapWithStyle`:

```go
wrappedDB, err := tenantkit.WrapWithStyle(db, tenantkit.Config{
    TenantTables: []string{"users", "orders"},
    TenantColumn: "tenant_id",
}, tenantkit.PlaceholderQuestion)
```

### Transactions

```go
tx, err := wrappedDB.Begin(ctx, nil)
if err != nil {
    log.Fatal(err)
}

// All transaction queries are tenant-filtered
_, err = tx.Exec(ctx, "INSERT INTO orders (product, qty) VALUES ($1, $2)", "Widget", 5)
if err != nil {
    tx.Rollback()
    log.Fatal(err)
}

tx.Commit()
```

### Admin / Cross-Tenant Queries

Use `WithoutTenantFiltering` to bypass tenant isolation for admin operations:

```go
ctx := tenantkit.WithoutTenantFiltering(context.Background())
rows, err := wrappedDB.Query(ctx, "SELECT COUNT(*) FROM users")
```

## How It Works

TenantKit intercepts all database operations and automatically modifies SQL queries to include tenant filtering:

| Operation | Original Query | Transformed Query |
|-----------|---------------|-------------------|
| SELECT | `SELECT * FROM users` | `SELECT * FROM users WHERE users.tenant_id = $1` |
| INSERT | `INSERT INTO users (name) VALUES ($1)` | `INSERT INTO users (tenant_id, name) VALUES ($1, $2)` |
| UPDATE | `UPDATE users SET name = $1` | `UPDATE users SET name = $1 WHERE tenant_id = $2` |
| DELETE | `DELETE FROM users WHERE id = $1` | `DELETE FROM users WHERE tenant_id = $2 AND (id = $1)` |

**Two-Rule Decision System:**
1. **System queries** (DDL, health checks, `pg_catalog`, etc.) are automatically bypassed
2. Only tables listed in `TenantTables` get filtering — other tables pass through unchanged

## Documentation

- 🏗️ [Architecture](./docs/architecture.md) — C4 diagrams and system design
- 📖 [Usage Guide](./docs/usage.md) — Complete developer reference

## Adapters

TenantKit provides adapters for popular Go frameworks. Each adapter is a separate Go module — you only pull in the dependencies you need.

### HTTP Middleware

| Adapter | Package |
|---------|---------|
| stdlib (net/http) | `github.com/abhipray-cpu/tenantkit/adapters/http-stdlib` |
| Gin | `github.com/abhipray-cpu/tenantkit/adapters/http-gin` |
| Echo | `github.com/abhipray-cpu/tenantkit/adapters/http-echo` |
| Chi | `github.com/abhipray-cpu/tenantkit/adapters/http-chi` |
| Fiber | `github.com/abhipray-cpu/tenantkit/adapters/http-fiber` |

### Database

| Adapter | Package |
|---------|---------|
| database/sql | `github.com/abhipray-cpu/tenantkit/adapters/sql` |
| GORM | `github.com/abhipray-cpu/tenantkit/adapters/gorm` |
| sqlx | `github.com/abhipray-cpu/tenantkit/adapters/sqlx` |
| PostgreSQL (RLS) | `github.com/abhipray-cpu/tenantkit/adapters/postgres` |

### Rate Limiting & Quotas

| Adapter | Package |
|---------|---------|
| Rate Limiter (Memory) | `github.com/abhipray-cpu/tenantkit/adapters/limiter-memory` |
| Rate Limiter (Redis) | `github.com/abhipray-cpu/tenantkit/adapters/limiter-redis` |
| Quota Manager (Memory) | `github.com/abhipray-cpu/tenantkit/adapters/quota-memory` |
| Quota Manager (Redis) | `github.com/abhipray-cpu/tenantkit/adapters/quota-redis` |

### Metrics

| Adapter | Package |
|---------|---------|
| No-Op | `github.com/abhipray-cpu/tenantkit/adapters/metrics-noop` |
| Prometheus | `github.com/abhipray-cpu/tenantkit/adapters/metrics-prometheus` |

## Performance

| Operation | Latency | Throughput |
|-----------|---------|------------|
| Query cache hit | ~20ns | 59M ops/s |
| Query transformation | ~1μs | 1M ops/s |
| Rate limit check (memory) | ~100ns | 10M ops/s |
| Context tenant extraction | ~10ns | 100M ops/s |

### Memory Optimization

- **60-70% reduction** in allocations per query using `sync.Pool`
- **LRU cache** with configurable entry limit prevents unbounded memory growth
- **Thread-safe** with minimal lock contention

## Requirements

- Go 1.21 or later
- Database with a `tenant_id` column (or custom column name) on tenant-scoped tables

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## Security

For security vulnerabilities, please see our [Security Policy](SECURITY.md).

## License

TenantKit is released under the [MIT License](LICENSE).

## Author

**Abhipray Puttanarasaiah**  
Email: dumkaabhipray@gmail.com  
GitHub: [@abhipray-cpu](https://github.com/abhipray-cpu)

---

Made with ❤️ for the Go community
