package metricsprometheus

// MetricsConfig holds configuration for Prometheus metrics
type MetricsConfig struct {
	// HTTPRequestSizeBuckets defines histogram buckets for HTTP request size
	// Values should be in bytes. Example: {100, 1000, 10000, 100000, 1000000}
	HTTPRequestSizeBuckets []float64

	// HTTPResponseSizeBuckets defines histogram buckets for HTTP response size
	// Values should be in bytes. Example: {100, 1000, 10000, 100000, 1000000}
	HTTPResponseSizeBuckets []float64

	// QueryDurationBuckets defines histogram buckets for query execution time
	// Values should be in milliseconds. Example: {1, 5, 10, 50, 100, 500, 1000}
	QueryDurationBuckets []float64

	// QueryRowsBuckets defines histogram buckets for rows affected by queries
	// Example: {1, 10, 100, 1000, 10000, 100000}
	QueryRowsBuckets []float64
}

// DefaultMetricsConfig returns sensible production defaults
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		// HTTP request sizes: 100B to 1MB
		HTTPRequestSizeBuckets: []float64{100, 1000, 10000, 100000, 1000000},

		// HTTP response sizes: 100B to 1MB
		HTTPResponseSizeBuckets: []float64{100, 1000, 10000, 100000, 1000000},

		// Query durations: 1ms to 1 second
		QueryDurationBuckets: []float64{1, 5, 10, 50, 100, 500, 1000},

		// Query rows affected: 1 to 100k
		QueryRowsBuckets: []float64{1, 10, 100, 1000, 10000, 100000},
	}
}

// HighGranularityMetricsConfig returns config optimized for detailed metrics (more buckets)
func HighGranularityMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		// More granular request sizes
		HTTPRequestSizeBuckets: []float64{
			10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000, 5000000,
		},

		// More granular response sizes
		HTTPResponseSizeBuckets: []float64{
			10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000, 5000000,
		},

		// More granular query durations: 1ms to 10 seconds
		QueryDurationBuckets: []float64{
			1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000,
		},

		// More granular row counts
		QueryRowsBuckets: []float64{
			1, 5, 10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000, 500000,
		},
	}
}

// LowGranularityMetricsConfig returns config optimized for minimal overhead (fewer buckets)
func LowGranularityMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		// Minimal request sizes
		HTTPRequestSizeBuckets: []float64{1000, 100000, 1000000},

		// Minimal response sizes
		HTTPResponseSizeBuckets: []float64{1000, 100000, 1000000},

		// Minimal query durations
		QueryDurationBuckets: []float64{10, 100, 1000},

		// Minimal row counts
		QueryRowsBuckets: []float64{100, 10000, 100000},
	}
}
