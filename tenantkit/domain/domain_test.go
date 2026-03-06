package domain

import (
	"context"
	"testing"
)

// Test TenantID validation
func TestNewTenantID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid lowercase", "my-tenant", false},
		{"valid with numbers", "tenant-123", false},
		{"valid single char", "a", false},
		{"valid uuid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid with underscores", "org_123", false},
		{"valid with dots", "tenant.prod", false},
		{"valid long", "this-is-a-very-long-tenant-id-with-dashes-and-numbers-123456789012345678901234567890", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 256)), true},
		{"uppercase converted", "My-Tenant", false},
		{"spaces", "my tenant", true},
		{"special chars", "tenant@123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tid, err := NewTenantID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTenantID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tid.Value() != tid.String() {
				t.Errorf("String() = %v, want %v", tid.String(), tid.Value())
			}
		})
	}
}

// Test TenantID equality
func TestTenantIDEqual(t *testing.T) {
	id1, _ := NewTenantID("tenant-1")
	id2, _ := NewTenantID("tenant-1")
	id3, _ := NewTenantID("tenant-2")

	if !id1.Equal(id2) {
		t.Error("Equal IDs should be equal")
	}
	if id1.Equal(id3) {
		t.Error("Different IDs should not be equal")
	}
}

// Test Context creation
func TestNewContext(t *testing.T) {
	ctx, err := NewContext("my-tenant", "user-123", "req-456")
	if err != nil {
		t.Errorf("NewContext() error = %v", err)
	}

	if ctx.TenantID().Value() != "my-tenant" {
		t.Errorf("TenantID() = %v, want my-tenant", ctx.TenantID())
	}
	if ctx.UserID() != "user-123" {
		t.Errorf("UserID() = %v, want user-123", ctx.UserID())
	}
	if ctx.RequestID() != "req-456" {
		t.Errorf("RequestID() = %v, want req-456", ctx.RequestID())
	}
}

// Test Context validation
func TestContextValidation(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  string
		userID    string
		requestID string
		wantErr   bool
	}{
		{"valid", "my-tenant", "user-123", "req-456", false},
		{"empty user ID", "my-tenant", "", "req-456", true},
		{"empty request ID", "my-tenant", "user-123", "", true},
		{"invalid tenant ID", "", "user-123", "req-456", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewContext(tt.tenantID, tt.userID, tt.requestID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test Context WithUser
func TestContextWithUser(t *testing.T) {
	ctx, _ := NewContext("my-tenant", "user-123", "req-456")

	newCtx, err := ctx.WithUser("user-789")
	if err != nil {
		t.Errorf("WithUser() error = %v", err)
	}

	if newCtx.UserID() != "user-789" {
		t.Errorf("UserID() = %v, want user-789", newCtx.UserID())
	}
	if newCtx.TenantID().Value() != "my-tenant" {
		t.Errorf("TenantID() = %v, want my-tenant", newCtx.TenantID())
	}
}

// Test Context with Go context.Context
func TestContextGoContext(t *testing.T) {
	ctx := context.Background()
	tc, _ := NewContext("my-tenant", "user-123", "req-456")

	goCtx := tc.ToGoContext(ctx)

	retrieved, err := FromGoContext(goCtx)
	if err != nil {
		t.Errorf("FromGoContext() error = %v", err)
	}

	if retrieved.TenantID().Value() != tc.TenantID().Value() {
		t.Errorf("TenantID() = %v, want %v", retrieved.TenantID(), tc.TenantID())
	}
	if retrieved.UserID() != tc.UserID() {
		t.Errorf("UserID() = %v, want %v", retrieved.UserID(), tc.UserID())
	}
}

// Test FromGoContext error cases
func TestFromGoContextErrors(t *testing.T) {
	// Test nil context — FromGoContext explicitly handles nil and returns ErrInvalidContext
	var nilCtx context.Context
	_, err := FromGoContext(nilCtx)
	if err == nil {
		t.Error("Expected error for nil context")
	}

	// Context without tenant
	_, err = FromGoContext(context.Background())
	if err == nil {
		t.Error("Expected error for context without tenant")
	}

	// Context with wrong type
	ctx := context.WithValue(context.Background(), TenantContextKey, "not-a-context")
	_, err = FromGoContext(ctx)
	if err == nil {
		t.Error("Expected error for wrong context type")
	}
}

// Test all errors are defined
func TestErrorsDefined(t *testing.T) {
	errors := []error{
		ErrTenantNotFound,
		ErrTenantExists,
		ErrInvalidContext,
		ErrMissingTenantID,
		ErrMissingUserID,
		ErrMissingRequestID,
		ErrUnsafeQuery,
		ErrQueryParseFailed,
		ErrQueryRewriteFailed,
		ErrStorageNotAvailable,
		ErrTransactionFailed,
		ErrCacheNotAvailable,
		ErrQuotaExceeded,
		ErrQuotaNotFound,
		ErrRateLimitExceeded,
		ErrInvalidTenantID,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("All error sentinels must be non-nil")
		}
	}
}
