package sqladapter

import (
	"strings"
	"testing"
	"time"
)

// TestStorageConfigBuilder_DoesNotPanicOnInvalidValues ensures builder doesn't panic,
// but instead returns errors on Build()
func TestStorageConfigBuilder_DoesNotPanicOnInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		buildFunc func() (*Config, error)
		wantErr   string
	}{
		{
			name: "negative MaxOpenConnections",
			buildFunc: func() (*Config, error) {
				return NewStorageConfigBuilder().
					WithMaxOpenConnections(-1).
					BuildWithValidation()
			},
			wantErr: "MaxOpenConnections must be > 0",
		},
		{
			name: "zero MaxOpenConnections",
			buildFunc: func() (*Config, error) {
				return NewStorageConfigBuilder().
					WithMaxOpenConnections(0).
					BuildWithValidation()
			},
			wantErr: "MaxOpenConnections must be > 0",
		},
		{
			name: "negative MaxIdleConnections",
			buildFunc: func() (*Config, error) {
				return NewStorageConfigBuilder().
					WithMaxIdleConnections(-1).
					BuildWithValidation()
			},
			wantErr: "MaxIdleConnections must be >= 0",
		},
		{
			name: "negative ConnMaxLifetime",
			buildFunc: func() (*Config, error) {
				return NewStorageConfigBuilder().
					WithConnMaxLifetime(-1 * time.Second).
					BuildWithValidation()
			},
			wantErr: "ConnMaxLifetime must be > 0",
		},
		{
			name: "zero ConnMaxLifetime",
			buildFunc: func() (*Config, error) {
				return NewStorageConfigBuilder().
					WithConnMaxLifetime(0).
					BuildWithValidation()
			},
			wantErr: "ConnMaxLifetime must be > 0",
		},
		{
			name: "negative ConnMaxIdleTime",
			buildFunc: func() (*Config, error) {
				return NewStorageConfigBuilder().
					WithConnMaxIdleTime(-1 * time.Second).
					BuildWithValidation()
			},
			wantErr: "ConnMaxIdleTime must be >= 0",
		},
		{
			name: "negative QueryTimeout",
			buildFunc: func() (*Config, error) {
				return NewStorageConfigBuilder().
					WithQueryTimeout(-1 * time.Second).
					BuildWithValidation()
			},
			wantErr: "QueryTimeout must be > 0",
		},
		{
			name: "zero QueryTimeout",
			buildFunc: func() (*Config, error) {
				return NewStorageConfigBuilder().
					WithQueryTimeout(0).
					BuildWithValidation()
			},
			wantErr: "QueryTimeout must be > 0",
		},
		{
			name: "multiple validation errors",
			buildFunc: func() (*Config, error) {
				return NewStorageConfigBuilder().
					WithMaxOpenConnections(-1).
					WithMaxIdleConnections(-1).
					WithQueryTimeout(0).
					BuildWithValidation()
			},
			wantErr: "MaxOpenConnections", // Should accumulate all errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure builder methods don't panic
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Builder panicked: %v - Builders should NOT panic", r)
				}
			}()

			// Build should return error
			cfg, err := tt.buildFunc()

			if err == nil {
				t.Fatal("BuildWithValidation() should return error for invalid config")
			}

			if cfg != nil {
				t.Fatal("BuildWithValidation() should return nil config on error")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Error should contain '%s', got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestStorageConfigBuilder_ValidConfig ensures valid configs work
func TestStorageConfigBuilder_ValidConfig(t *testing.T) {
	cfg, err := NewStorageConfigBuilder().
		WithMaxOpenConnections(50).
		WithMaxIdleConnections(10).
		WithConnMaxLifetime(2 * time.Hour).
		WithConnMaxIdleTime(30 * time.Minute).
		WithQueryTimeout(60 * time.Second).
		BuildWithValidation()

	if err != nil {
		t.Fatalf("Valid config should not error: %v", err)
	}

	if cfg == nil {
		t.Fatal("Valid config should return non-nil config")
	}

	// Verify values
	if cfg.MaxOpenConnections != 50 {
		t.Errorf("Expected MaxOpenConnections=50, got %d", cfg.MaxOpenConnections)
	}
	if cfg.MaxIdleConnections != 10 {
		t.Errorf("Expected MaxIdleConnections=10, got %d", cfg.MaxIdleConnections)
	}
}

// TestStorageConfigBuilder_BackwardCompatible ensures old Build() still works
func TestStorageConfigBuilder_BackwardCompatible(t *testing.T) {
	// Old pattern should still work without validation
	cfg := NewStorageConfigBuilder().
		WithMaxOpenConnections(50).
		Build()

	if cfg == nil {
		t.Fatal("Build() should return config for backward compatibility")
	}

	if cfg.MaxOpenConnections != 50 {
		t.Errorf("Expected MaxOpenConnections=50, got %d", cfg.MaxOpenConnections)
	}
}

// TestStorageConfigBuilder_AutoAdjustsIdleConnections ensures auto-adjustment works
func TestStorageConfigBuilder_AutoAdjustsIdleConnections(t *testing.T) {
	cfg, err := NewStorageConfigBuilder().
		WithMaxOpenConnections(10).
		WithMaxIdleConnections(20). // More than max open
		BuildWithValidation()

	if err != nil {
		t.Fatalf("Auto-adjustment should not cause error: %v", err)
	}

	// Should auto-adjust to MaxOpenConnections / 2
	if cfg.MaxIdleConnections != 5 {
		t.Errorf("Expected MaxIdleConnections to be auto-adjusted to 5, got %d", cfg.MaxIdleConnections)
	}
}

// TestStorageConfigBuilder_ErrorAccumulation ensures all errors are collected
func TestStorageConfigBuilder_ErrorAccumulation(t *testing.T) {
	_, err := NewStorageConfigBuilder().
		WithMaxOpenConnections(-1).         // Error 1
		WithMaxIdleConnections(-1).         // Error 2
		WithConnMaxLifetime(0).             // Error 3
		WithQueryTimeout(-5 * time.Second). // Error 4
		BuildWithValidation()

	if err == nil {
		t.Fatal("BuildWithValidation() should return error")
	}

	errMsg := err.Error()

	// Should contain all 4 errors
	expectedErrors := []string{
		"MaxOpenConnections",
		"MaxIdleConnections",
		"ConnMaxLifetime",
		"QueryTimeout",
	}

	for _, expected := range expectedErrors {
		if !strings.Contains(errMsg, expected) {
			t.Errorf("Error message should contain '%s', got: %s", expected, errMsg)
		}
	}
}
