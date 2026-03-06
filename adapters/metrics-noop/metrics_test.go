package noop

import (
	"context"
	"testing"
)

func TestNoOpMetrics_RecordRequest(t *testing.T) {
	metrics := NewNoOpMetrics()
	err := metrics.RecordRequest(context.Background(), "GET", "/api/users", 200, 100)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestNoOpMetrics_RecordQuery(t *testing.T) {
	metrics := NewNoOpMetrics()
	err := metrics.RecordQuery(context.Background(), "SELECT * FROM users", 10, 50)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestNoOpMetrics_RecordError(t *testing.T) {
	metrics := NewNoOpMetrics()
	err := metrics.RecordError(context.Background(), "database_error", "connection timeout")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestNoOpMetrics_RecordQuotaUsage(t *testing.T) {
	metrics := NewNoOpMetrics()
	err := metrics.RecordQuotaUsage(context.Background(), "api_requests", 500, 1000)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestNoOpMetrics_RecordRateLimit(t *testing.T) {
	metrics := NewNoOpMetrics()
	err := metrics.RecordRateLimit(context.Background(), "/api/endpoint", true)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestNoOpMetrics_RecordCacheHit(t *testing.T) {
	metrics := NewNoOpMetrics()
	err := metrics.RecordCacheHit(context.Background(), "user:123", true)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	err = metrics.RecordCacheHit(context.Background(), "user:456", false)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestNoOpMetrics_AllMethodsReturnNil(t *testing.T) {
	metrics := NewNoOpMetrics()
	ctx := context.Background()

	// Test that all methods consistently return nil
	tests := []struct {
		name string
		fn   func() error
	}{
		{"RecordRequest", func() error {
			return metrics.RecordRequest(ctx, "POST", "/test", 201, 150)
		}},
		{"RecordQuery", func() error {
			return metrics.RecordQuery(ctx, "INSERT INTO test", 1, 25)
		}},
		{"RecordError", func() error {
			return metrics.RecordError(ctx, "test_error", "test message")
		}},
		{"RecordQuotaUsage", func() error {
			return metrics.RecordQuotaUsage(ctx, "storage", 1024, 2048)
		}},
		{"RecordRateLimit", func() error {
			return metrics.RecordRateLimit(ctx, "/api/test", false)
		}},
		{"RecordCacheHit", func() error {
			return metrics.RecordCacheHit(ctx, "key", true)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err != nil {
				t.Errorf("%s returned error: %v, want nil", tt.name, err)
			}
		})
	}
}

func TestNewNoOpMetrics(t *testing.T) {
	metrics := NewNoOpMetrics()
	if metrics == nil {
		t.Error("NewNoOpMetrics returned nil")
	}
}
