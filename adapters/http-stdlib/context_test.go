package httpstd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abhipray-cpu/tenantkit/domain"
)

// TestGetTenantID tests the GetTenantID helper
func TestGetTenantID(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*http.Request) *http.Request
		wantTenant  string
		wantError   bool
		description string
	}{
		{
			name: "valid_context",
			setup: func(r *http.Request) *http.Request {
				tenantCtx, _ := domain.NewContext("tenant1", "user1", "req1")
				goCtx := tenantCtx.ToGoContext(r.Context())
				return r.WithContext(goCtx)
			},
			wantTenant:  "tenant1",
			wantError:   false,
			description: "Should extract tenant ID from context",
		},
		{
			name: "missing_context",
			setup: func(r *http.Request) *http.Request {
				return r
			},
			wantTenant:  "",
			wantError:   true,
			description: "Should fail when context missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req = tt.setup(req)

			got, err := GetTenantID(req)

			if (err != nil) != tt.wantError {
				t.Errorf("%s: got error=%v, wantError=%v", tt.description, err, tt.wantError)
			}

			if got != tt.wantTenant {
				t.Errorf("%s: got tenant=%q, want=%q", tt.description, got, tt.wantTenant)
			}
		})
	}
}

// TestGetTenantContext tests the GetTenantContext helper
func TestGetTenantContext(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*http.Request) *http.Request
		wantTenant  string
		wantUser    string
		wantError   bool
		description string
	}{
		{
			name: "valid_context",
			setup: func(r *http.Request) *http.Request {
				tenantCtx, _ := domain.NewContext("tenant1", "user1", "req1")
				goCtx := tenantCtx.ToGoContext(r.Context())
				return r.WithContext(goCtx)
			},
			wantTenant:  "tenant1",
			wantUser:    "user1",
			wantError:   false,
			description: "Should extract full tenant context",
		},
		{
			name: "missing_context",
			setup: func(r *http.Request) *http.Request {
				return r
			},
			wantTenant:  "",
			wantUser:    "",
			wantError:   true,
			description: "Should fail when context missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req = tt.setup(req)

			ctx, err := GetTenantContext(req)

			if (err != nil) != tt.wantError {
				t.Errorf("%s: got error=%v, wantError=%v", tt.description, err, tt.wantError)
			}

			if !tt.wantError {
				if ctx.TenantID().Value() != tt.wantTenant {
					t.Errorf("%s: got tenant=%q, want=%q", tt.description, ctx.TenantID().Value(), tt.wantTenant)
				}
				if ctx.UserID() != tt.wantUser {
					t.Errorf("%s: got user=%q, want=%q", tt.description, ctx.UserID(), tt.wantUser)
				}
			}
		})
	}
}

// TestMustGetTenantID tests the MustGetTenantID helper
func TestMustGetTenantID(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*http.Request) *http.Request
		wantTenant  string
		shouldPanic bool
		description string
	}{
		{
			name: "valid_context",
			setup: func(r *http.Request) *http.Request {
				tenantCtx, _ := domain.NewContext("tenant1", "user1", "req1")
				goCtx := tenantCtx.ToGoContext(r.Context())
				return r.WithContext(goCtx)
			},
			wantTenant:  "tenant1",
			shouldPanic: false,
			description: "Should extract tenant ID without panic",
		},
		{
			name: "missing_context",
			setup: func(r *http.Request) *http.Request {
				return r
			},
			wantTenant:  "",
			shouldPanic: true,
			description: "Should panic when context missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req = tt.setup(req)

			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("%s: unexpected panic: %v", tt.description, r)
					}
				} else if tt.shouldPanic {
					t.Errorf("%s: expected panic but got none", tt.description)
				}
			}()

			got := MustGetTenantID(req)

			if !tt.shouldPanic && got != tt.wantTenant {
				t.Errorf("%s: got tenant=%q, want=%q", tt.description, got, tt.wantTenant)
			}
		})
	}
}

// TestMustGetTenantContext tests the MustGetTenantContext helper
func TestMustGetTenantContext(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*http.Request) *http.Request
		wantTenant  string
		shouldPanic bool
		description string
	}{
		{
			name: "valid_context",
			setup: func(r *http.Request) *http.Request {
				tenantCtx, _ := domain.NewContext("tenant1", "user1", "req1")
				goCtx := tenantCtx.ToGoContext(r.Context())
				return r.WithContext(goCtx)
			},
			wantTenant:  "tenant1",
			shouldPanic: false,
			description: "Should extract context without panic",
		},
		{
			name: "missing_context",
			setup: func(r *http.Request) *http.Request {
				return r
			},
			wantTenant:  "",
			shouldPanic: true,
			description: "Should panic when context missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req = tt.setup(req)

			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("%s: unexpected panic: %v", tt.description, r)
					}
				} else if tt.shouldPanic {
					t.Errorf("%s: expected panic but got none", tt.description)
				}
			}()

			ctx := MustGetTenantContext(req)

			if !tt.shouldPanic && ctx.TenantID().Value() != tt.wantTenant {
				t.Errorf("%s: got tenant=%q, want=%q", tt.description, ctx.TenantID().Value(), tt.wantTenant)
			}
		})
	}
}

// TestWithTenantID tests the WithTenantID helper
func TestWithTenantID(t *testing.T) {
	tests := []struct {
		name        string
		tenantID    string
		wantError   bool
		description string
	}{
		{
			name:        "valid_tenant",
			tenantID:    "tenant1",
			wantError:   false,
			description: "Should create request with tenant context",
		},
		{
			name:        "empty_tenant",
			tenantID:    "",
			wantError:   true,
			description: "Should fail with empty tenant ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)

			newReq, err := WithTenantID(req, tt.tenantID)

			if (err != nil) != tt.wantError {
				t.Errorf("%s: got error=%v, wantError=%v", tt.description, err, tt.wantError)
			}

			if !tt.wantError {
				tenantID, err := GetTenantID(newReq)
				if err != nil {
					t.Errorf("%s: failed to extract tenant: %v", tt.description, err)
				}
				if tenantID != tt.tenantID {
					t.Errorf("%s: got tenant=%q, want=%q", tt.description, tenantID, tt.tenantID)
				}
			}
		})
	}
}
