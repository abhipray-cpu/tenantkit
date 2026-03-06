package ports

// Package ports defines the interfaces (ports) for TenantKit's hexagonal architecture.
//
// Adapters implement these interfaces to provide pluggable backends for
// rate limiting, quota management, metrics, storage, and tenant resolution.
//
// # Available Ports
//
//   - [Limiter] — Per-tenant rate limiting (token bucket, sliding window, etc.)
//   - [QuotaManager] — Resource quota enforcement for noisy-neighbor protection
//   - [Metrics] — Observability (query counts, latencies, cache hit rates)
//   - [Storage] — Tenant metadata persistence
//   - [StorageTransaction] — Transactional storage operations
//   - [Enforcer] — SQL query rewriting enforcement
//   - [Resolver] — Tenant ID resolution from HTTP requests
//
// # Adapter Modules
//
// Each adapter is a separate Go module so applications only pull in the
// dependencies they actually need:
//
//	go get github.com/abhipray-cpu/tenantkit/adapters/limiter-redis
//	go get github.com/abhipray-cpu/tenantkit/adapters/quota-redis
//	go get github.com/abhipray-cpu/tenantkit/adapters/metrics-prometheus
