package sqladapter

import (
	"context"
	"testing"
)

// TestQueryRow_DoesNotPanicOnError ensures QueryRow returns error, not panic
// This is PHASE 1.1 - Critical fix to prevent service crashes
func TestQueryRow_DoesNotPanicOnError(t *testing.T) {
	// Create storage with nil enforcer to trigger error
	storage := &Storage{
		db:       nil, // Will cause error
		config:   DefaultConfig(),
		enforcer: nil, // This will cause enforcement to fail
	}

	ctx := context.Background()

	// This should NOT panic - must return error gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("QueryRow panicked: %v - This violates non-invasive principle", r)
		}
	}()

	// Call should return error, not panic
	// TODO: This will fail because QueryRowWithError doesn't exist yet
	row, err := storage.QueryRowWithError(ctx, "SELECT 1")

	if err == nil {
		t.Fatal("Expected error for nil enforcer, got nil")
	}

	// Verify error is actionable
	if row != nil {
		t.Fatal("Expected nil row on error")
	}
}

// TestQueryRow_BackwardCompatible ensures old API still works
func TestQueryRow_BackwardCompatible(t *testing.T) {
	t.Skip("TODO: Implement after storage setup helper is created")

	// This test will verify that existing code using QueryRow still works
	// Old API should still work but with proper error handling
}

// TestQueryRow_ReturnsGracefulError ensures errors don't crash service
func TestQueryRow_ReturnsGracefulError(t *testing.T) {
	t.Skip("TODO: Implement after storage setup helper is created")

	// This will test that invalid queries return errors gracefully
}

// TestQueryRowWithError_NewAPIWorks tests the new recommended API
func TestQueryRowWithError_NewAPIWorks(t *testing.T) {
	t.Skip("TODO: Implement after storage setup helper is created")

	// This will test the new explicit error-returning API
}
