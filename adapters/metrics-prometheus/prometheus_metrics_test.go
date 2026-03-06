package metricsprometheus

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPrometheusMetrics(t *testing.T) {
	pm := NewPrometheusMetrics("test", "subsystem")
	if pm == nil {
		t.Fatal("PrometheusMetrics should not be nil")
	}

	if pm.namespace != "test" {
		t.Errorf("Expected namespace 'test', got '%s'", pm.namespace)
	}

	if pm.subsystem != "subsystem" {
		t.Errorf("Expected subsystem 'subsystem', got '%s'", pm.subsystem)
	}
}

func TestNewPrometheusMetricsWithRegistry(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMetricsWithRegistry("test", "subsystem", reg)

	if pm == nil {
		t.Fatal("PrometheusMetrics should not be nil")
	}

	if pm.registerer != reg {
		t.Error("Custom registry not set")
	}
}

func TestRecordRequest(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMetricsWithRegistry("test", "metrics", reg)
	ctx := context.Background()

	err := pm.RecordRequest(ctx, "GET", "/api/users", 200, 150)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = pm.RecordRequest(ctx, "POST", "/api/users", 201, 250)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = pm.RecordRequest(ctx, "GET", "/api/users", 404, 50)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRecordQuery(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMetricsWithRegistry("test", "metrics", reg)
	ctx := context.Background()

	tests := []struct {
		query    string
		rows     int64
		duration int64
		expectOp string
	}{
		{"SELECT * FROM users WHERE id = ?", 1, 10, "select"},
		{"INSERT INTO users (name) VALUES (?)", 1, 25, "insert"},
		{"UPDATE users SET name = ? WHERE id = ?", 1, 15, "update"},
		{"DELETE FROM users WHERE id = ?", 1, 10, "delete"},
	}

	for _, tt := range tests {
		err := pm.RecordQuery(ctx, tt.query, tt.rows, tt.duration)
		if err != nil {
			t.Errorf("Expected no error for %s, got %v", tt.expectOp, err)
		}
	}
}

func TestRecordError(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMetricsWithRegistry("test", "metrics", reg)
	ctx := context.Background()

	err := pm.RecordError(ctx, "validation_error", "Invalid input")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = pm.RecordError(ctx, "database_error", "Connection timeout")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRecordQuotaUsage(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMetricsWithRegistry("test", "metrics", reg)
	ctx := context.Background()

	tests := []struct {
		quotaType string
		used      int64
		limit     int64
	}{
		{"api_requests_daily", 2500, 5000},
		{"database_rows", 50000, 1000000},
		{"storage_bytes", 536870912, 1073741824}, // 512MB / 1GB
	}

	for _, tt := range tests {
		err := pm.RecordQuotaUsage(ctx, tt.quotaType, tt.used, tt.limit)
		if err != nil {
			t.Errorf("Expected no error for %s, got %v", tt.quotaType, err)
		}
	}
}

func TestRecordQuotaUsage_ZeroLimit(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMetricsWithRegistry("test", "metrics", reg)
	ctx := context.Background()

	// Should not panic with zero limit
	err := pm.RecordQuotaUsage(ctx, "test_quota", 0, 0)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRecordRateLimit(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMetricsWithRegistry("test", "metrics", reg)
	ctx := context.Background()

	// Record allowed requests
	err := pm.RecordRateLimit(ctx, "/api/users", false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = pm.RecordRateLimit(ctx, "/api/users", false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Record limited requests
	err = pm.RecordRateLimit(ctx, "/api/users", true)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRecordCacheHit(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMetricsWithRegistry("test", "metrics", reg)
	ctx := context.Background()

	// Record cache hits
	err := pm.RecordCacheHit(ctx, "tenant:config:123", true)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = pm.RecordCacheHit(ctx, "tenant:config:456", true)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Record cache misses
	err = pm.RecordCacheHit(ctx, "tenant:config:789", false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestMultipleTenants(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMetricsWithRegistry("test", "metrics", reg)

	ctx1 := context.WithValue(context.Background(), "tenant_id", "tenant-1")
	ctx2 := context.WithValue(context.Background(), "tenant_id", "tenant-2")

	err := pm.RecordRequest(ctx1, "GET", "/api/test", 200, 100)
	if err != nil {
		t.Errorf("Expected no error for tenant-1, got %v", err)
	}

	err = pm.RecordRequest(ctx2, "GET", "/api/test", 200, 150)
	if err != nil {
		t.Errorf("Expected no error for tenant-2, got %v", err)
	}

	err = pm.RecordQuotaUsage(ctx1, "api_requests_daily", 1000, 5000)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = pm.RecordQuotaUsage(ctx2, "api_requests_daily", 2000, 5000)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExtractOperation(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"SELECT * FROM users", "select"},
		{"SELECT id, name FROM users WHERE id = ?", "select"},
		{"INSERT INTO users (name) VALUES (?)", "insert"},
		{"UPDATE users SET name = ? WHERE id = ?", "update"},
		{"DELETE FROM users WHERE id = ?", "delete"},
		{"UNKNOWN OPERATION", "other"},
		{"select lowercase", "select"}, // Should handle uppercase conversion
	}

	for _, tt := range tests {
		result := extractOperation(tt.query)
		if result != tt.expected {
			t.Errorf("extractOperation(%q) = %q, expected %q", tt.query, result, tt.expected)
		}
	}
}

func TestExtractCacheType(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"tenant:config:123", "config"},
		{"tenant:user:456", "user"},
		{"tenant:settings:789", "settings"},
		{"single", "default"},
		{"", "default"},
	}

	for _, tt := range tests {
		result := extractCacheType(tt.key)
		if result != tt.expected {
			t.Errorf("extractCacheType(%q) = %q, expected %q", tt.key, result, tt.expected)
		}
	}
}

func TestGetTenantID_WithContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "tenant_id", "test-tenant")
	tenantID := getTenantID(ctx)

	if tenantID != "test-tenant" {
		t.Errorf("Expected 'test-tenant', got %q", tenantID)
	}
}

func TestGetTenantID_WithoutContext(t *testing.T) {
	ctx := context.Background()
	tenantID := getTenantID(ctx)

	if tenantID != "unknown" {
		t.Errorf("Expected 'unknown', got %q", tenantID)
	}
}

func TestStringUtilities(t *testing.T) {
	// Test toUpperASCII
	if toUpperASCII("hello") != "HELLO" {
		t.Error("toUpperASCII failed")
	}

	if toUpperASCII("HeLLo") != "HELLO" {
		t.Error("toUpperASCII mixed case failed")
	}

	if toUpperASCII("123!@#") != "123!@#" {
		t.Error("toUpperASCII with special chars failed")
	}

	// Test startsWith
	if !startsWith("hello world", "hello") {
		t.Error("startsWith failed")
	}

	if startsWith("hello world", "world") {
		t.Error("startsWith false positive")
	}

	// Test splitString
	parts := splitString("a:b:c", ":")
	if len(parts) != 3 || parts[0] != "a" || parts[1] != "b" || parts[2] != "c" {
		t.Error("splitString failed")
	}
}

func TestConcurrentRecording(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMetricsWithRegistry("test", "metrics", reg)

	// Record metrics concurrently
	done := make(chan bool, 10)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		go func(index int) {
			endpoint := "/api/test"
			pm.RecordRequest(ctx, "GET", endpoint, 200, 100)
			pm.RecordQuery(ctx, "SELECT * FROM users", 1, 10)
			pm.RecordError(ctx, "test_error", "test message")
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMetricsNamespace(t *testing.T) {
	pm := NewPrometheusMetrics("myapp", "core")

	if pm.namespace != "myapp" {
		t.Errorf("Expected namespace 'myapp', got %q", pm.namespace)
	}

	if pm.subsystem != "core" {
		t.Errorf("Expected subsystem 'core', got %q", pm.subsystem)
	}
}
