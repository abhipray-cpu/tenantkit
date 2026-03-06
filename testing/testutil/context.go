// Package testutil provides testing utilities for TenantKit
// These utilities support TDD approach and make testing easier
package testutil

import (
	"context"
	"testing"
	"time"
)

// TestTenantID is a well-known tenant ID for testing
const TestTenantID = "test-tenant-001"

// TestUserID is a well-known user ID for testing
const TestUserID = "test-user-001"

// ContextKey types for tenant context
type contextKey string

const (
	TenantIDKey contextKey = "tenant_id"
	UserIDKey   contextKey = "user_id"
)

// WithTestTenant creates a context with a test tenant ID
func WithTestTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantIDKey, tenantID)
}

// WithTestUser creates a context with a test user ID
func WithTestUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// NewTestContext creates a context with default test tenant and user
func NewTestContext() context.Context {
	ctx := context.Background()
	ctx = WithTestTenant(ctx, TestTenantID)
	ctx = WithTestUser(ctx, TestUserID)
	return ctx
}

// NewTestContextWithTimeout creates a test context with a timeout
func NewTestContextWithTimeout(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx := NewTestContext()
	return context.WithTimeout(ctx, timeout)
}

// NewTestContextWithCancel creates a test context with a cancel function
func NewTestContextWithCancel(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx := NewTestContext()
	return context.WithCancel(ctx)
}

// GetTenantFromContext extracts tenant ID from context
func GetTenantFromContext(ctx context.Context) string {
	if v := ctx.Value(TenantIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// GetUserFromContext extracts user ID from context
func GetUserFromContext(ctx context.Context) string {
	if v := ctx.Value(UserIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// TestConfig holds configuration for tests
type TestConfig struct {
	PostgresURL string
	MySQLURL    string
	RedisURL    string
	Timeout     time.Duration
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() TestConfig {
	return TestConfig{
		PostgresURL: "postgres://tenantkit:tenantkit_secret@localhost:5432/tenantkit?sslmode=disable",
		MySQLURL:    "mysql://tenantkit:tenantkit_secret@localhost:3306/tenantkit",
		RedisURL:    "redis://localhost:6379",
		Timeout:     30 * time.Second,
	}
}

// SkipIfShort skips the test if running in short mode
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
}

// SkipIfIntegration skips if running unit tests only
func SkipIfIntegration(t *testing.T) {
	t.Helper()
	// TODO: Check for INTEGRATION_TEST env var
}

// Retry runs a function with retries and backoff
func Retry(t *testing.T, attempts int, delay time.Duration, fn func() error) error {
	t.Helper()

	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			t.Logf("attempt %d failed: %v, retrying in %v", i+1, err, delay)
			time.Sleep(delay)
			delay *= 2 // exponential backoff
		}
	}

	return lastErr
}

// Eventually waits for a condition to become true
func Eventually(t *testing.T, timeout time.Duration, condition func() bool, msgAndArgs ...interface{}) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if condition() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("condition not met within timeout", msgAndArgs)
		}
	}
}

// Never asserts that a condition is never true during the specified duration
func Never(t *testing.T, duration time.Duration, condition func() bool, msgAndArgs ...interface{}) {
	t.Helper()

	deadline := time.Now().Add(duration)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if condition() {
			t.Fatal("condition became true when it should never be", msgAndArgs)
		}
		if time.Now().After(deadline) {
			return // Success - condition was never true
		}
	}
}
