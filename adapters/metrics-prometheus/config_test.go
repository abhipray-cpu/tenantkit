package metricsprometheus

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// TestDefaultMetricsConfig verifies default configuration values
func TestDefaultMetricsConfig(t *testing.T) {
	config := DefaultMetricsConfig()

	if config == nil {
		t.Fatal("DefaultMetricsConfig returned nil")
	}

	if len(config.HTTPRequestSizeBuckets) == 0 {
		t.Error("HTTPRequestSizeBuckets is empty")
	}

	if len(config.HTTPResponseSizeBuckets) == 0 {
		t.Error("HTTPResponseSizeBuckets is empty")
	}

	if len(config.QueryDurationBuckets) == 0 {
		t.Error("QueryDurationBuckets is empty")
	}

	if len(config.QueryRowsBuckets) == 0 {
		t.Error("QueryRowsBuckets is empty")
	}

	// Verify buckets are in ascending order
	for i := 0; i < len(config.HTTPRequestSizeBuckets)-1; i++ {
		if config.HTTPRequestSizeBuckets[i] >= config.HTTPRequestSizeBuckets[i+1] {
			t.Errorf("HTTPRequestSizeBuckets not in ascending order at index %d", i)
		}
	}
}

// TestHighGranularityMetricsConfig verifies high granularity configuration
func TestHighGranularityMetricsConfig(t *testing.T) {
	config := HighGranularityMetricsConfig()

	if config == nil {
		t.Fatal("HighGranularityMetricsConfig returned nil")
	}

	// High granularity should have more buckets than defaults
	defaultConfig := DefaultMetricsConfig()
	if len(config.HTTPRequestSizeBuckets) <= len(defaultConfig.HTTPRequestSizeBuckets) {
		t.Error("HighGranularityMetricsConfig should have more buckets than default")
	}

	if len(config.QueryDurationBuckets) <= len(defaultConfig.QueryDurationBuckets) {
		t.Error("HighGranularityMetricsConfig query duration should have more buckets")
	}
}

// TestLowGranularityMetricsConfig verifies low granularity configuration
func TestLowGranularityMetricsConfig(t *testing.T) {
	config := LowGranularityMetricsConfig()

	if config == nil {
		t.Fatal("LowGranularityMetricsConfig returned nil")
	}

	// Low granularity should have fewer buckets than defaults
	defaultConfig := DefaultMetricsConfig()
	if len(config.HTTPRequestSizeBuckets) >= len(defaultConfig.HTTPRequestSizeBuckets) {
		t.Error("LowGranularityMetricsConfig should have fewer buckets than default")
	}

	if len(config.QueryDurationBuckets) >= len(defaultConfig.QueryDurationBuckets) {
		t.Error("LowGranularityMetricsConfig query duration should have fewer buckets")
	}
}

// TestNewPrometheusMetricsWithConfig creates metrics with custom config
func TestNewPrometheusMetricsWithConfig(t *testing.T) {
	config := &MetricsConfig{
		HTTPRequestSizeBuckets:  []float64{100, 1000, 10000},
		HTTPResponseSizeBuckets: []float64{100, 1000, 10000},
		QueryDurationBuckets:    []float64{1, 10, 100},
		QueryRowsBuckets:        []float64{10, 100, 1000},
	}

	registry := prometheus.NewRegistry()
	metrics := NewPrometheusMetricsWithRegistryAndConfig("test", "metrics", registry, config)

	if metrics == nil {
		t.Fatal("NewPrometheusMetricsWithConfig returned nil")
	}

	if metrics.config != config {
		t.Error("MetricsConfig not set correctly")
	}

	if metrics.namespace != "test" {
		t.Errorf("Expected namespace 'test', got '%s'", metrics.namespace)
	}

	if metrics.subsystem != "metrics" {
		t.Errorf("Expected subsystem 'metrics', got '%s'", metrics.subsystem)
	}
}

// TestNewPrometheusMetricsWithConfig_NilConfig uses defaults when nil config provided
func TestNewPrometheusMetricsWithConfig_NilConfig(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewPrometheusMetricsWithRegistryAndConfig("test", "metrics", registry, nil)

	if metrics == nil {
		t.Fatal("NewPrometheusMetricsWithConfig returned nil")
	}

	if metrics.config == nil {
		t.Fatal("Config should be set to default, not nil")
	}

	// Should have default bucket counts
	defaultConfig := DefaultMetricsConfig()
	if len(metrics.config.HTTPRequestSizeBuckets) != len(defaultConfig.HTTPRequestSizeBuckets) {
		t.Error("Should use default buckets when nil config provided")
	}
}

// TestNewPrometheusMetrics_UsesDefaultConfig verifies NewPrometheusMetrics uses default configuration
func TestNewPrometheusMetrics_UsesDefaultConfig(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewPrometheusMetricsWithRegistry("test", "metrics", registry)

	if metrics == nil {
		t.Fatal("NewPrometheusMetrics returned nil")
	}

	if metrics.config == nil {
		t.Fatal("Config should be initialized")
	}

	// Verify default configuration is used
	defaultConfig := DefaultMetricsConfig()
	if len(metrics.config.HTTPRequestSizeBuckets) != len(defaultConfig.HTTPRequestSizeBuckets) {
		t.Error("Should use default configuration")
	}
}

// TestBucketOrdering verifies all bucket configurations are in ascending order
func TestBucketOrdering(t *testing.T) {
	configs := []*MetricsConfig{
		DefaultMetricsConfig(),
		HighGranularityMetricsConfig(),
		LowGranularityMetricsConfig(),
	}

	for idx, config := range configs {
		// Check HTTPRequestSizeBuckets
		for i := 0; i < len(config.HTTPRequestSizeBuckets)-1; i++ {
			if config.HTTPRequestSizeBuckets[i] >= config.HTTPRequestSizeBuckets[i+1] {
				t.Errorf("Config %d: HTTPRequestSizeBuckets not in ascending order", idx)
			}
		}

		// Check HTTPResponseSizeBuckets
		for i := 0; i < len(config.HTTPResponseSizeBuckets)-1; i++ {
			if config.HTTPResponseSizeBuckets[i] >= config.HTTPResponseSizeBuckets[i+1] {
				t.Errorf("Config %d: HTTPResponseSizeBuckets not in ascending order", idx)
			}
		}

		// Check QueryDurationBuckets
		for i := 0; i < len(config.QueryDurationBuckets)-1; i++ {
			if config.QueryDurationBuckets[i] >= config.QueryDurationBuckets[i+1] {
				t.Errorf("Config %d: QueryDurationBuckets not in ascending order", idx)
			}
		}

		// Check QueryRowsBuckets
		for i := 0; i < len(config.QueryRowsBuckets)-1; i++ {
			if config.QueryRowsBuckets[i] >= config.QueryRowsBuckets[i+1] {
				t.Errorf("Config %d: QueryRowsBuckets not in ascending order", idx)
			}
		}
	}
}

// TestMetricsInitialization verifies metrics are properly initialized with config
func TestMetricsInitialization(t *testing.T) {
	config := HighGranularityMetricsConfig()
	registry := prometheus.NewRegistry()
	metrics := NewPrometheusMetricsWithRegistryAndConfig("test", "metrics", registry, config)

	// Verify metrics are created (no panic)
	if metrics == nil {
		t.Fatal("Metrics should be created successfully")
	}

	// Verify all metric fields are initialized
	if metrics.httpRequestsTotal == (prometheus.CounterVec{}) {
		t.Error("httpRequestsTotal not initialized")
	}

	if metrics.httpRequestDurationMs == (prometheus.HistogramVec{}) {
		t.Error("httpRequestDurationMs not initialized")
	}

	if metrics.httpRequestSize == (prometheus.HistogramVec{}) {
		t.Error("httpRequestSize not initialized")
	}

	if metrics.httpResponseSize == (prometheus.HistogramVec{}) {
		t.Error("httpResponseSize not initialized")
	}

	if metrics.queriesTotal == (prometheus.CounterVec{}) {
		t.Error("queriesTotal not initialized")
	}

	if metrics.queryDurationMs == (prometheus.HistogramVec{}) {
		t.Error("queryDurationMs not initialized")
	}

	if metrics.queryRowsAffected == (prometheus.HistogramVec{}) {
		t.Error("queryRowsAffected not initialized")
	}
}

// TestConfigBackwardCompatibility verifies old constructors still work
func TestConfigBackwardCompatibility(t *testing.T) {
	reg1 := prometheus.NewRegistry()
	reg2 := prometheus.NewRegistry()

	// Old constructor should still work
	metrics1 := NewPrometheusMetricsWithRegistry("test", "metrics", reg1)
	if metrics1 == nil {
		t.Fatal("NewPrometheusMetrics should still work")
	}

	// New constructor should produce same result
	metrics2 := NewPrometheusMetricsWithRegistryAndConfig("test", "metrics", reg2, DefaultMetricsConfig())
	if metrics2 == nil {
		t.Fatal("NewPrometheusMetricsWithConfig should work")
	}

	// Both should have same configuration
	if len(metrics1.config.HTTPRequestSizeBuckets) != len(metrics2.config.HTTPRequestSizeBuckets) {
		t.Error("Backward compatibility broken: different bucket counts")
	}
}

// TestBucketBoundaries verifies reasonable bucket boundaries for production
func TestBucketBoundaries(t *testing.T) {
	config := DefaultMetricsConfig()

	// HTTP request sizes should start at reasonable minimum
	if config.HTTPRequestSizeBuckets[0] < 10 {
		t.Error("First HTTP request bucket too low (< 10 bytes)")
	}

	if config.HTTPRequestSizeBuckets[len(config.HTTPRequestSizeBuckets)-1] < 1000000 {
		t.Error("Last HTTP request bucket too low (< 1MB)")
	}

	// Query durations should be reasonable
	if config.QueryDurationBuckets[0] < 1 {
		t.Error("First query duration bucket too low (< 1ms)")
	}

	if config.QueryDurationBuckets[len(config.QueryDurationBuckets)-1] < 100 {
		t.Error("Last query duration bucket too low (< 100ms)")
	}
}

// TestConfigCustomization verifies users can create custom configurations
func TestConfigCustomization(t *testing.T) {
	customConfig := &MetricsConfig{
		HTTPRequestSizeBuckets:  []float64{500, 5000, 50000},
		HTTPResponseSizeBuckets: []float64{500, 5000, 50000},
		QueryDurationBuckets:    []float64{5, 50, 500},
		QueryRowsBuckets:        []float64{50, 500, 5000},
	}

	metrics := NewPrometheusMetricsWithConfig("custom", "app", customConfig)

	if metrics == nil {
		t.Fatal("Custom config should work")
	}

	// Verify custom values are used
	if len(metrics.config.HTTPRequestSizeBuckets) != 3 {
		t.Error("Custom bucket count not respected")
	}

	if metrics.config.HTTPRequestSizeBuckets[0] != 500 {
		t.Error("Custom bucket values not set correctly")
	}
}
