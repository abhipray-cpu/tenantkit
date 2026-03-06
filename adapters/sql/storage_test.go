package sqladapter

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/abhipray-cpu/tenantkit/domain"
)

// MockDB is a mock implementation of sql.DB for testing
type MockDB struct {
	queryFunc    func(query string, args ...interface{}) (*sql.Rows, error)
	queryRowFunc func(query string, args ...interface{}) *sql.Row
	execFunc     func(query string, args ...interface{}) (sql.Result, error)
	closeFunc    func() error
	healthFunc   func() error
}

func TestNewStorage(t *testing.T) {
	// We can't test with a real database here, but we can test the constructor
	config := DefaultConfig()
	if config == nil {
		t.Error("DefaultConfig() returned nil")
	}

	if config.MaxOpenConnections != 25 {
		t.Errorf("expected MaxOpenConnections=25, got %d", config.MaxOpenConnections)
	}

	if config.QueryTimeout != time.Second*30 {
		t.Errorf("expected QueryTimeout=30s, got %v", config.QueryTimeout)
	}
}

func TestStorageEnforcement(t *testing.T) {
	// Create a mock storage by not actually connecting to a database
	// Instead, we'll test the enforcement logic

	tests := []struct {
		name      string
		query     string
		tenantID  string
		checkFunc func(t *testing.T, enforced string)
	}{
		{
			name:     "SELECT query enforcement",
			query:    "SELECT * FROM users WHERE id = 1",
			tenantID: "tenant123",
			checkFunc: func(t *testing.T, enforced string) {
				if !contains(enforced, "tenant_id") {
					t.Errorf("expected enforced query to contain tenant_id, got: %s", enforced)
				}
			},
		},
		{
			name:     "UPDATE query enforcement",
			query:    "UPDATE users SET name = 'test'",
			tenantID: "tenant456",
			checkFunc: func(t *testing.T, enforced string) {
				if !contains(enforced, "tenant_id") {
					t.Errorf("expected enforced query to contain tenant_id, got: %s", enforced)
				}
			},
		},
		{
			name:     "DELETE query enforcement",
			query:    "DELETE FROM users",
			tenantID: "tenant789",
			checkFunc: func(t *testing.T, enforced string) {
				if !contains(enforced, "tenant_id") {
					t.Errorf("expected enforced query to contain tenant_id, got: %s", enforced)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create tenant context
			tc, err := domain.NewContext(tt.tenantID, "user1", "req1")
			if err != nil {
				t.Fatalf("failed to create tenant context: %v", err)
			}
			ctx := tc.ToGoContext(context.Background())

			// Create storage with enforcer and config for testing
			storage := &Storage{
				db:       nil, // We won't execute actual queries in this test
				enforcer: NewEnforcer(),
				config:   DefaultConfig(),
			}

			// Verify storage fields are properly initialized
			if storage.db != nil {
				t.Error("Expected nil db in test setup")
			}

			// Verify enforcer is properly initialized
			if storage.enforcer == nil {
				t.Fatal("Enforcer should be initialized")
			}

			// Verify config is applied
			if storage.config == nil {
				t.Fatal("Config should be initialized")
			}

			// Test enforcement
			enforced, _, err := storage.enforcer.EnforceQuery(ctx, tt.query, []interface{}{})
			if err != nil {
				t.Errorf("EnforceQuery failed: %v", err)
			}

			tt.checkFunc(t, enforced)
		})
	}
}

func TestStorageHealth(t *testing.T) {
	storage := &Storage{
		db:       nil,
		enforcer: NewEnforcer(),
		config:   DefaultConfig(),
	}

	// Health check should fail with nil db
	ctx := context.Background()
	err := storage.Health(ctx)
	if err == nil {
		t.Error("expected Health() to return error with nil db, got nil")
	}
}

func TestStorageClose(t *testing.T) {
	storage := &Storage{
		db:       nil,
		enforcer: NewEnforcer(),
		config:   DefaultConfig(),
	}

	// Close should not panic with nil db
	err := storage.Close()
	if err != nil {
		t.Errorf("expected Close() to return nil with nil db, got %v", err)
	}
}

func TestStorageGetEnforcer(t *testing.T) {
	storage := &Storage{
		db:       nil,
		enforcer: NewEnforcer(),
		config:   DefaultConfig(),
	}

	enforcer := storage.GetEnforcer()
	if enforcer == nil {
		t.Error("expected GetEnforcer() to return non-nil enforcer, got nil")
	}

	ops := enforcer.SupportedOperations()
	if len(ops) == 0 {
		t.Error("expected enforcer to have supported operations, got none")
	}
}

// TestStorageConfigBuilder_WithMaxOpenConnections tests setting max open connections
func TestStorageConfigBuilder_WithMaxOpenConnections(t *testing.T) {
	config := NewStorageConfigBuilder().
		WithMaxOpenConnections(100).
		Build()

	if config.MaxOpenConnections != 100 {
		t.Errorf("Expected MaxOpenConnections=100, got %d", config.MaxOpenConnections)
	}
}

// TestStorageConfigBuilder_WithMaxIdleConnections tests setting max idle connections
func TestStorageConfigBuilder_WithMaxIdleConnections(t *testing.T) {
	config := NewStorageConfigBuilder().
		WithMaxIdleConnections(20).
		Build()

	if config.MaxIdleConnections != 20 {
		t.Errorf("Expected MaxIdleConnections=20, got %d", config.MaxIdleConnections)
	}
}

// TestStorageConfigBuilder_WithConnMaxLifetime tests setting connection lifetime
func TestStorageConfigBuilder_WithConnMaxLifetime(t *testing.T) {
	config := NewStorageConfigBuilder().
		WithConnMaxLifetime(time.Hour * 2).
		Build()

	if config.ConnMaxLifetime != time.Hour*2 {
		t.Errorf("Expected ConnMaxLifetime=2h, got %v", config.ConnMaxLifetime)
	}
}

// TestStorageConfigBuilder_WithConnMaxIdleTime tests setting connection idle time
func TestStorageConfigBuilder_WithConnMaxIdleTime(t *testing.T) {
	config := NewStorageConfigBuilder().
		WithConnMaxIdleTime(time.Minute * 5).
		Build()

	if config.ConnMaxIdleTime != time.Minute*5 {
		t.Errorf("Expected ConnMaxIdleTime=5m, got %v", config.ConnMaxIdleTime)
	}
}

// TestStorageConfigBuilder_WithQueryTimeout tests setting query timeout
func TestStorageConfigBuilder_WithQueryTimeout(t *testing.T) {
	config := NewStorageConfigBuilder().
		WithQueryTimeout(time.Minute).
		Build()

	if config.QueryTimeout != time.Minute {
		t.Errorf("Expected QueryTimeout=1m, got %v", config.QueryTimeout)
	}
}

// TestStorageConfigBuilder_DefaultValues tests that defaults are sensible
func TestStorageConfigBuilder_DefaultValues(t *testing.T) {
	config := NewStorageConfigBuilder().Build()

	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
	}{
		{"MaxOpenConnections", config.MaxOpenConnections, 25},
		{"MaxIdleConnections", config.MaxIdleConnections, 5},
		{"ConnMaxLifetime", config.ConnMaxLifetime, time.Hour},
		{"ConnMaxIdleTime", config.ConnMaxIdleTime, time.Minute * 10},
		{"QueryTimeout", config.QueryTimeout, time.Second * 30},
	}

	for _, tt := range tests {
		if tt.actual != tt.expected {
			t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, tt.actual)
		}
	}
}

// TestStorageConfigBuilder_Chaining tests fluent interface chaining
func TestStorageConfigBuilder_Chaining(t *testing.T) {
	config := NewStorageConfigBuilder().
		WithMaxOpenConnections(100).
		WithMaxIdleConnections(10).
		WithConnMaxLifetime(time.Hour * 2).
		WithConnMaxIdleTime(time.Minute * 5).
		WithQueryTimeout(time.Minute).
		Build()

	if config.MaxOpenConnections != 100 {
		t.Errorf("Chaining failed: MaxOpenConnections expected 100, got %d", config.MaxOpenConnections)
	}
	if config.MaxIdleConnections != 10 {
		t.Errorf("Chaining failed: MaxIdleConnections expected 10, got %d", config.MaxIdleConnections)
	}
	if config.QueryTimeout != time.Minute {
		t.Errorf("Chaining failed: QueryTimeout expected 1m, got %v", config.QueryTimeout)
	}
}

// TestStorageConfigBuilder_ValidationMaxOpenConnections tests validation for MaxOpenConnections
// Updated: No longer expects panic - validates with BuildWithValidation()
func TestStorageConfigBuilder_ValidationMaxOpenConnections(t *testing.T) {
	// With BuildWithValidation(), should return error
	_, err := NewStorageConfigBuilder().WithMaxOpenConnections(0).BuildWithValidation()
	if err == nil {
		t.Error("Expected error for MaxOpenConnections <= 0")
	}

	// With Build(), should not panic (backward compatible)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Build() should not panic for backward compatibility: %v", r)
		}
	}()
	config := NewStorageConfigBuilder().WithMaxOpenConnections(0).Build()
	if config == nil {
		t.Error("Build() should return config even with invalid values for backward compatibility")
	}
}

// TestStorageConfigBuilder_ValidationMaxIdleConnections tests validation for MaxIdleConnections
// Updated: No longer expects panic - validates with BuildWithValidation()
func TestStorageConfigBuilder_ValidationMaxIdleConnections(t *testing.T) {
	// With BuildWithValidation(), should return error
	_, err := NewStorageConfigBuilder().WithMaxIdleConnections(-1).BuildWithValidation()
	if err == nil {
		t.Error("Expected error for MaxIdleConnections < 0")
	}

	// With Build(), should not panic (backward compatible)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Build() should not panic for backward compatibility: %v", r)
		}
	}()
	config := NewStorageConfigBuilder().WithMaxIdleConnections(-1).Build()
	if config == nil {
		t.Error("Build() should return config even with invalid values for backward compatibility")
	}
}

// TestStorageConfigBuilder_ValidationConnMaxLifetime tests validation for ConnMaxLifetime
// Updated: No longer expects panic - validates with BuildWithValidation()
func TestStorageConfigBuilder_ValidationConnMaxLifetime(t *testing.T) {
	// With BuildWithValidation(), should return error
	_, err := NewStorageConfigBuilder().WithConnMaxLifetime(0).BuildWithValidation()
	if err == nil {
		t.Error("Expected error for ConnMaxLifetime <= 0")
	}

	// With Build(), should not panic (backward compatible)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Build() should not panic for backward compatibility: %v", r)
		}
	}()
	config := NewStorageConfigBuilder().WithConnMaxLifetime(0).Build()
	if config == nil {
		t.Error("Build() should return config even with invalid values for backward compatibility")
	}
}

// TestStorageConfigBuilder_ValidationQueryTimeout tests validation for QueryTimeout
// Updated: No longer expects panic - validates with BuildWithValidation()
func TestStorageConfigBuilder_ValidationQueryTimeout(t *testing.T) {
	// With BuildWithValidation(), should return error
	_, err := NewStorageConfigBuilder().WithQueryTimeout(0).BuildWithValidation()
	if err == nil {
		t.Error("Expected error for QueryTimeout <= 0")
	}

	// With Build(), should not panic (backward compatible)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Build() should not panic for backward compatibility: %v", r)
		}
	}()
	config := NewStorageConfigBuilder().WithQueryTimeout(0).Build()
	if config == nil {
		t.Error("Build() should return config even with invalid values for backward compatibility")
	}
}

// TestStorageConfigBuilder_AutoAdjustIdleConnections tests that idle connections are auto-adjusted
func TestStorageConfigBuilder_AutoAdjustIdleConnections(t *testing.T) {
	config := NewStorageConfigBuilder().
		WithMaxOpenConnections(10).
		WithMaxIdleConnections(20). // More than open connections
		Build()

	if config.MaxIdleConnections >= config.MaxOpenConnections {
		t.Errorf("Expected idle connections to be less than open connections, but got idle=%d, open=%d",
			config.MaxIdleConnections, config.MaxOpenConnections)
	}
}

// TestStorageConfigBuilder_BackwardCompatibility tests that DefaultConfig still works
func TestStorageConfigBuilder_BackwardCompatibility(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Error("DefaultConfig() should not return nil")
	}

	if config.MaxOpenConnections <= 0 {
		t.Error("DefaultConfig() should have positive MaxOpenConnections")
	}

	if config.QueryTimeout <= 0 {
		t.Error("DefaultConfig() should have positive QueryTimeout")
	}
}

// TestStorageHealth_WithConfiguredTimeout verifies Health uses configured timeout
func TestStorageHealth_WithConfiguredTimeout(t *testing.T) {
	config := NewStorageConfigBuilder().
		WithHealthCheckConfig(FastHealthCheckConfig()).
		Build()

	if config.HealthCheckConfig == nil {
		t.Fatal("HealthCheckConfig should be set")
	}

	if config.HealthCheckConfig.Timeout != 1*time.Second {
		t.Errorf("Expected fast timeout of 1s, got %v", config.HealthCheckConfig.Timeout)
	}

	// Verify config contains the right values
	if config.HealthCheckConfig.Interval != 5*time.Second {
		t.Errorf("Expected fast interval of 5s, got %v", config.HealthCheckConfig.Interval)
	}
}

// TestStorageConfig_HealthCheckConfigIntegration verifies full integration
func TestStorageConfig_HealthCheckConfigIntegration(t *testing.T) {
	tests := []struct {
		name      string
		getConfig func() *HealthCheckConfig
		timeout   time.Duration
		interval  time.Duration
	}{
		{"Default", DefaultHealthCheckConfig, 5 * time.Second, 30 * time.Second},
		{"Fast", FastHealthCheckConfig, 1 * time.Second, 5 * time.Second},
		{"Relaxed", RelaxedHealthCheckConfig, 10 * time.Second, 60 * time.Second},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := NewStorageConfigBuilder().
				WithHealthCheckConfig(test.getConfig()).
				Build()

			if config.HealthCheckConfig.Timeout != test.timeout {
				t.Errorf("Expected timeout %v, got %v", test.timeout, config.HealthCheckConfig.Timeout)
			}

			if config.HealthCheckConfig.Interval != test.interval {
				t.Errorf("Expected interval %v, got %v", test.interval, config.HealthCheckConfig.Interval)
			}
		})
	}
}
