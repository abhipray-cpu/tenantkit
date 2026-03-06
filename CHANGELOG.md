# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-03-06

### Added
- Core `Wrap()` and `WrapWithStyle()` API for transparent tenant-scoped SQL rewriting
- Transaction support via `Begin()` and `BeginTx()` on wrapped DB
- Context-based tenant management: `WithTenant()`, `GetTenant()`, `WithoutTenantFiltering()`
- Two-Rule Decision System: system query bypass + tenant table filtering
- LRU query cache with FNV-1a hashing and `sync.Pool` optimizations
- `PlaceholderDollar` (PostgreSQL), `PlaceholderQuestion` (MySQL/SQLite), `PlaceholderColon` (Oracle) support
- Structured logging via `log/slog` (configurable via `Config.Logger`)
- HTTP middleware adapters: Gin, Echo, Chi, Fiber, net/http
- Database adapters: database/sql, GORM, sqlx, PostgreSQL RLS
- Rate limiting adapters: in-memory (token bucket, sliding window, fixed window) and Redis
- Quota management adapters: in-memory and Redis (Lua-scripted atomic operations)
- Metrics adapters: Prometheus and no-op
- Port/adapter architecture with 7 port interfaces
- Comprehensive test suite with `-race` detection
- CI/CD with GitHub Actions (lint, test, security scan, release)
- Documentation: architecture (C4 diagrams), usage guide
- Open source scaffolding: LICENSE (MIT), CONTRIBUTING.md, SECURITY.md, issue templates

### Security
- Tenant filter always injected even if `tenant_id` appears in user's WHERE clause (prevents bypass)
- System query detection prevents accidental filtering of DDL, health checks, and catalog queries
