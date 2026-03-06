package metricsprometheus

import (
	"context"
	"fmt"
	"sync"

	"github.com/abhipray-cpu/tenantkit/tenantkit/ports"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Compile-time interface check
var _ ports.Metrics = (*PrometheusMetrics)(nil)

// PrometheusMetrics implements the Metrics port interface using Prometheus
type PrometheusMetrics struct {
	mu         sync.RWMutex
	namespace  string
	subsystem  string
	buckets    []float64
	config     *MetricsConfig
	registerer prometheus.Registerer
	gatherer   prometheus.Gatherer

	// HTTP request metrics
	httpRequestsTotal     prometheus.CounterVec
	httpRequestDurationMs prometheus.HistogramVec
	httpRequestSize       prometheus.HistogramVec
	httpResponseSize      prometheus.HistogramVec

	// Database query metrics
	queriesTotal      prometheus.CounterVec
	queryDurationMs   prometheus.HistogramVec
	queryRowsAffected prometheus.HistogramVec

	// Error metrics
	errorsTotal prometheus.CounterVec

	// Quota metrics
	quotaUsedBytes  prometheus.GaugeVec
	quotaLimitBytes prometheus.GaugeVec
	quotaUsageRatio prometheus.GaugeVec

	// Rate limit metrics
	rateLimitHitsTotal     prometheus.CounterVec
	rateLimitRequestsTotal prometheus.CounterVec

	// Cache metrics
	cacheHitsTotal   prometheus.CounterVec
	cacheMissesTotal prometheus.CounterVec
	cacheHitRatio    prometheus.GaugeVec

	// Cache hit ratio tracking (for calculation)
	cacheStats map[string]*cacheMetricsPair
}

// cacheMetricsPair tracks hits and misses for a specific tenant/cache_type combination
type cacheMetricsPair struct {
	mu     sync.Mutex
	hits   int64
	misses int64
}

// NewPrometheusMetrics creates a new Prometheus metrics collector with default configuration
func NewPrometheusMetrics(namespace, subsystem string) *PrometheusMetrics {
	return NewPrometheusMetricsWithConfig(namespace, subsystem, DefaultMetricsConfig())
}

// NewPrometheusMetricsWithRegistry creates metrics with a custom registry and default configuration
func NewPrometheusMetricsWithRegistry(namespace, subsystem string, reg prometheus.Registerer) *PrometheusMetrics {
	return NewPrometheusMetricsWithRegistryAndConfig(namespace, subsystem, reg, DefaultMetricsConfig())
}

// NewPrometheusMetricsWithConfig creates metrics with custom bucket configuration
func NewPrometheusMetricsWithConfig(namespace, subsystem string, config *MetricsConfig) *PrometheusMetrics {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	pm := &PrometheusMetrics{
		namespace:  namespace,
		subsystem:  subsystem,
		registerer: prometheus.DefaultRegisterer,
		gatherer:   prometheus.DefaultGatherer,
		buckets:    prometheus.DefBuckets, // Default buckets for request duration
		config:     config,
		cacheStats: make(map[string]*cacheMetricsPair),
	}

	pm.initMetrics()
	return pm
}

// NewPrometheusMetricsWithRegistryAndConfig creates metrics with custom registry and bucket configuration
func NewPrometheusMetricsWithRegistryAndConfig(namespace, subsystem string, reg prometheus.Registerer, config *MetricsConfig) *PrometheusMetrics {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	pm := &PrometheusMetrics{
		namespace:  namespace,
		subsystem:  subsystem,
		registerer: reg,
		gatherer:   prometheus.DefaultGatherer,
		buckets:    prometheus.DefBuckets,
		config:     config,
		cacheStats: make(map[string]*cacheMetricsPair),
	}

	pm.initMetrics()
	return pm
}

// initMetrics initializes all Prometheus metrics
func (pm *PrometheusMetrics) initMetrics() {
	factory := promauto.With(pm.registerer)

	// HTTP request metrics
	pm.httpRequestsTotal = *factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests by tenant, method, endpoint, and status",
		},
		[]string{"tenant_id", "method", "endpoint", "status"},
	)

	pm.httpRequestDurationMs = *factory.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "http_request_duration_ms",
			Help:      "HTTP request duration in milliseconds",
			Buckets:   pm.buckets,
		},
		[]string{"tenant_id", "method", "endpoint"},
	)

	pm.httpRequestSize = *factory.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "http_request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   pm.config.HTTPRequestSizeBuckets,
		},
		[]string{"tenant_id", "method", "endpoint"},
	)

	pm.httpResponseSize = *factory.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "http_response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   pm.config.HTTPResponseSizeBuckets,
		},
		[]string{"tenant_id", "method", "endpoint"},
	)

	// Database query metrics
	pm.queriesTotal = *factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "queries_total",
			Help:      "Total database queries by tenant and operation",
		},
		[]string{"tenant_id", "operation"},
	)

	pm.queryDurationMs = *factory.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "query_duration_ms",
			Help:      "Database query duration in milliseconds",
			Buckets:   pm.config.QueryDurationBuckets,
		},
		[]string{"tenant_id", "operation"},
	)

	pm.queryRowsAffected = *factory.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "query_rows_affected",
			Help:      "Database rows affected by query",
			Buckets:   pm.config.QueryRowsBuckets,
		},
		[]string{"tenant_id", "operation"},
	)

	// Error metrics
	pm.errorsTotal = *factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "errors_total",
			Help:      "Total errors by tenant and error type",
		},
		[]string{"tenant_id", "error_type"},
	)

	// Quota metrics
	pm.quotaUsedBytes = *factory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "quota_used_bytes",
			Help:      "Current quota usage in bytes",
		},
		[]string{"tenant_id", "quota_type"},
	)

	pm.quotaLimitBytes = *factory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "quota_limit_bytes",
			Help:      "Quota limit in bytes",
		},
		[]string{"tenant_id", "quota_type"},
	)

	pm.quotaUsageRatio = *factory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "quota_usage_ratio",
			Help:      "Quota usage as ratio (0-1)",
		},
		[]string{"tenant_id", "quota_type"},
	)

	// Rate limit metrics
	pm.rateLimitHitsTotal = *factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "rate_limit_hits_total",
			Help:      "Total rate limit hits by tenant and endpoint",
		},
		[]string{"tenant_id", "endpoint"},
	)

	pm.rateLimitRequestsTotal = *factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "rate_limit_requests_total",
			Help:      "Total requests to rate limited endpoints",
		},
		[]string{"tenant_id", "endpoint"},
	)

	// Cache metrics
	pm.cacheHitsTotal = *factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "cache_hits_total",
			Help:      "Total cache hits by tenant",
		},
		[]string{"tenant_id", "cache_type"},
	)

	pm.cacheMissesTotal = *factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "cache_misses_total",
			Help:      "Total cache misses by tenant",
		},
		[]string{"tenant_id", "cache_type"},
	)

	pm.cacheHitRatio = *factory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: pm.namespace,
			Subsystem: pm.subsystem,
			Name:      "cache_hit_ratio",
			Help:      "Cache hit ratio by tenant",
		},
		[]string{"tenant_id", "cache_type"},
	)
}

// RecordRequest records an HTTP request metric
func (pm *PrometheusMetrics) RecordRequest(ctx context.Context, method string, endpoint string, statusCode int, durationMS int64) error {
	tenantID := getTenantID(ctx)

	pm.httpRequestsTotal.WithLabelValues(tenantID, method, endpoint, fmt.Sprintf("%d", statusCode)).Inc()
	pm.httpRequestDurationMs.WithLabelValues(tenantID, method, endpoint).Observe(float64(durationMS))

	return nil
}

// RecordQuery records a database query metric
func (pm *PrometheusMetrics) RecordQuery(ctx context.Context, query string, rowsAffected int64, durationMS int64) error {
	tenantID := getTenantID(ctx)
	operation := extractOperation(query)

	pm.queriesTotal.WithLabelValues(tenantID, operation).Inc()
	pm.queryDurationMs.WithLabelValues(tenantID, operation).Observe(float64(durationMS))
	pm.queryRowsAffected.WithLabelValues(tenantID, operation).Observe(float64(rowsAffected))

	return nil
}

// RecordError records an error metric
func (pm *PrometheusMetrics) RecordError(ctx context.Context, errorType string, message string) error {
	tenantID := getTenantID(ctx)
	pm.errorsTotal.WithLabelValues(tenantID, errorType).Inc()
	return nil
}

// RecordQuotaUsage records quota consumption
func (pm *PrometheusMetrics) RecordQuotaUsage(ctx context.Context, quotaType string, used int64, limit int64) error {
	tenantID := getTenantID(ctx)

	pm.quotaUsedBytes.WithLabelValues(tenantID, quotaType).Set(float64(used))
	pm.quotaLimitBytes.WithLabelValues(tenantID, quotaType).Set(float64(limit))

	if limit > 0 {
		ratio := float64(used) / float64(limit)
		pm.quotaUsageRatio.WithLabelValues(tenantID, quotaType).Set(ratio)
	}

	return nil
}

// RecordRateLimit records rate limit hits
func (pm *PrometheusMetrics) RecordRateLimit(ctx context.Context, endpoint string, limited bool) error {
	tenantID := getTenantID(ctx)

	pm.rateLimitRequestsTotal.WithLabelValues(tenantID, endpoint).Inc()
	if limited {
		pm.rateLimitHitsTotal.WithLabelValues(tenantID, endpoint).Inc()
	}

	return nil
}

// RecordCacheHit records a cache hit or miss
func (pm *PrometheusMetrics) RecordCacheHit(ctx context.Context, key string, hit bool) error {
	tenantID := getTenantID(ctx)
	cacheType := extractCacheType(key)

	if hit {
		pm.cacheHitsTotal.WithLabelValues(tenantID, cacheType).Inc()
	} else {
		pm.cacheMissesTotal.WithLabelValues(tenantID, cacheType).Inc()
	}

	// Track cache statistics for ratio calculation
	statsKey := tenantID + ":" + cacheType

	pm.mu.Lock()
	stats, exists := pm.cacheStats[statsKey]
	if !exists {
		stats = &cacheMetricsPair{}
		pm.cacheStats[statsKey] = stats
	}
	pm.mu.Unlock()

	// Update counters thread-safely
	stats.mu.Lock()
	if hit {
		stats.hits++
	} else {
		stats.misses++
	}
	total := stats.hits + stats.misses
	stats.mu.Unlock()

	// Calculate and record cache hit ratio
	var ratio float64
	if total > 0 {
		ratio = float64(stats.hits) / float64(total)
	}
	pm.cacheHitRatio.WithLabelValues(tenantID, cacheType).Set(ratio)

	return nil
}

// Helper functions

func getTenantID(ctx context.Context) string {
	// Try to get tenant from context (similar to other adapters)
	if tenantID, ok := ctx.Value("tenant_id").(string); ok {
		return tenantID
	}
	return "unknown"
}

func extractOperation(query string) string {
	// Simple extraction of SQL operation type
	upperQuery := toUpperASCII(query)
	switch {
	case startsWith(upperQuery, "SELECT"):
		return "select"
	case startsWith(upperQuery, "INSERT"):
		return "insert"
	case startsWith(upperQuery, "UPDATE"):
		return "update"
	case startsWith(upperQuery, "DELETE"):
		return "delete"
	default:
		return "other"
	}
}

func extractCacheType(key string) string {
	// Extract cache type from key prefix
	// e.g., "tenant:config:123" -> "config"
	parts := splitString(key, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "default"
}

// String utility functions (no external dependencies)

func toUpperASCII(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		} else {
			b[i] = c
		}
	}
	return string(b)
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func splitString(s, sep string) []string {
	if len(sep) == 0 {
		return []string{s}
	}
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}
