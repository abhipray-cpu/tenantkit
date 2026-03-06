package metricsprometheus

// Package metricsprometheus provides a Prometheus metrics adapter implementing [ports.Metrics].
//
// It records per-tenant query counts, latencies, error rates, and cache
// hit/miss ratios as Prometheus metrics, enabling multi-tenant observability
// dashboards.
//
// # Usage
//
//	metrics := metricsprometheus.New(metricsprometheus.MetricsConfig{
//	    Namespace: "myapp",
//	    Subsystem: "tenantkit",
//	})
//
//	metrics.RecordQuery(ctx, "tenant-1", "SELECT", 1.5*time.Millisecond, nil)
