package tenantkit

import (
	"context"
	"testing"

	"github.com/abhipray-cpu/tenantkit/domain"
)

// TestWithTenant_AddsTenantToContext verifies tenant ID is stored in context
func TestWithTenant_AddsTenantToContext(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant-123"

	ctx = WithTenant(ctx, tenantID)

	retrieved, ok := GetTenant(ctx)
	if !ok {
		t.Fatal("Expected tenant to be present in context")
	}
	if retrieved != tenantID {
		t.Errorf("Expected tenant ID %q, got %q", tenantID, retrieved)
	}
}

// TestGetTenant_EmptyContext returns false when no tenant
func TestGetTenant_EmptyContext(t *testing.T) {
	ctx := context.Background()

	_, ok := GetTenant(ctx)
	if ok {
		t.Error("Expected no tenant in empty context")
	}
}

// TestGetTenant_EmptyString returns false for empty tenant ID
func TestGetTenant_EmptyString(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenant(ctx, "")

	_, ok := GetTenant(ctx)
	if ok {
		t.Error("Expected false for empty tenant ID")
	}
}

// TestWithTenant_OverridesTenant allows replacing tenant in context
func TestWithTenant_OverridesTenant(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenant(ctx, "tenant-1")
	ctx = WithTenant(ctx, "tenant-2")

	retrieved, ok := GetTenant(ctx)
	if !ok {
		t.Fatal("Expected tenant in context")
	}
	if retrieved != "tenant-2" {
		t.Errorf("Expected tenant-2, got %q", retrieved)
	}
}

// TestWithoutTenantFiltering_MarksBypass verifies bypass flag
func TestWithoutTenantFiltering_MarksBypass(t *testing.T) {
	ctx := context.Background()
	ctx = WithoutTenantFiltering(ctx)

	if !shouldBypass(ctx) {
		t.Error("Expected shouldBypass to return true")
	}
}

// TestShouldBypass_NoBypassFlag returns false when no bypass
func TestShouldBypass_NoBypassFlag(t *testing.T) {
	ctx := context.Background()

	if shouldBypass(ctx) {
		t.Error("Expected shouldBypass to return false for normal context")
	}
}

// TestWithoutTenantFiltering_WithTenant both can coexist
func TestWithoutTenantFiltering_WithTenant(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenant(ctx, "tenant-123")
	ctx = WithoutTenantFiltering(ctx)

	// Bypass should be set
	if !shouldBypass(ctx) {
		t.Error("Expected bypass flag to be set")
	}

	// Tenant should still be accessible
	tenantID, ok := GetTenant(ctx)
	if !ok || tenantID != "tenant-123" {
		t.Error("Expected tenant to still be in context")
	}
}

// TestContextKeys_DoNotCollide ensures our keys don't interfere
func TestContextKeys_DoNotCollide(t *testing.T) {
	ctx := context.Background()

	// Add our values
	ctx = WithTenant(ctx, "tenant-123")
	ctx = WithoutTenantFiltering(ctx)

	// Add some other values with potentially colliding string keys
	ctx = context.WithValue(ctx, "tenant_id", "should-not-interfere")
	ctx = context.WithValue(ctx, "bypass", "should-not-interfere")

	// Our values should still be accessible
	tenantID, ok := GetTenant(ctx)
	if !ok || tenantID != "tenant-123" {
		t.Error("Context key collision detected for tenant_id")
	}

	if !shouldBypass(ctx) {
		t.Error("Context key collision detected for bypass")
	}
}

// TestGetTenant_NilValue handles non-string values gracefully
func TestGetTenant_NilValue(t *testing.T) {
	ctx := context.Background()

	// Manually insert wrong type (shouldn't happen in practice)
	ctx = context.WithValue(ctx, contextKey("tenantkit_tenant_id"), 123)

	_, ok := GetTenant(ctx)
	if ok {
		t.Error("Expected false for non-string tenant value")
	}
}

// TestGetTenant_DomainContextFallback verifies that GetTenant finds the tenant ID
// when it was set via domain.ToGoContext (as HTTP middleware adapters do).
func TestGetTenant_DomainContextFallback(t *testing.T) {
	tc, err := domain.NewContext("tenant-from-middleware", "user-1", "req-1")
	if err != nil {
		t.Fatalf("Failed to create domain context: %v", err)
	}

	ctx := tc.ToGoContext(context.Background())

	tenantID, ok := GetTenant(ctx)
	if !ok {
		t.Fatal("Expected GetTenant to find tenant from domain context fallback")
	}
	if tenantID != "tenant-from-middleware" {
		t.Errorf("Expected tenant ID %q, got %q", "tenant-from-middleware", tenantID)
	}
}

// TestGetTenant_SimpleKeyTakesPrecedence verifies that the simple string key
// (set by WithTenant) takes precedence over the domain context.
func TestGetTenant_SimpleKeyTakesPrecedence(t *testing.T) {
	// Set domain context first
	tc, err := domain.NewContext("domain-tenant", "user-1", "req-1")
	if err != nil {
		t.Fatalf("Failed to create domain context: %v", err)
	}
	ctx := tc.ToGoContext(context.Background())

	// Then set simple key — should take precedence
	ctx = WithTenant(ctx, "simple-tenant")

	tenantID, ok := GetTenant(ctx)
	if !ok {
		t.Fatal("Expected tenant in context")
	}
	if tenantID != "simple-tenant" {
		t.Errorf("Expected simple-tenant to take precedence, got %q", tenantID)
	}
}
