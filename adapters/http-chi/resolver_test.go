package chi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeaderResolver(t *testing.T) {
	resolver := NewHeaderResolver("X-Tenant-ID")

	tests := []struct {
		name      string
		header    string
		value     string
		wantID    string
		wantError bool
	}{
		{"valid header", "X-Tenant-ID", "tenant-123", "tenant-123", false},
		{"missing header", "X-Tenant-ID", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.value != "" {
				req.Header.Set(tt.header, tt.value)
			}

			gotID, err := resolver.Resolve(req)
			if (err != nil) != tt.wantError {
				t.Errorf("expected error=%v, got error=%v", tt.wantError, err != nil)
			}
			if gotID != tt.wantID {
				t.Errorf("expected ID '%s', got '%s'", tt.wantID, gotID)
			}
		})
	}
}

func TestSubdomainResolver(t *testing.T) {
	tests := []struct {
		name       string
		baseDomain string
		host       string
		wantID     string
		wantError  bool
	}{
		{"valid subdomain", "example.com", "tenant1.example.com", "tenant1", false},
		{"with port", "example.com", "tenant1.example.com:8080", "tenant1", false},
		{"no subdomain", "example.com", "example.com", "", true},
		{"localhost no base", "", "localhost", "localhost", false},
		{"localhost with base", "example.com", "localhost", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewSubdomainResolver(tt.baseDomain)
			req := httptest.NewRequest("GET", "/test", nil)
			req.Host = tt.host

			gotID, err := resolver.Resolve(req)
			if (err != nil) != tt.wantError {
				t.Errorf("expected error=%v, got error=%v (err: %v)", tt.wantError, err != nil, err)
			}
			if gotID != tt.wantID {
				t.Errorf("expected ID '%s', got '%s'", tt.wantID, gotID)
			}
		})
	}
}

func TestPathResolver(t *testing.T) {
	resolver := NewPathResolver("/api/", 0)

	tests := []struct {
		name      string
		path      string
		wantID    string
		wantError bool
	}{
		{"valid path", "/api/tenant1/users", "tenant1", false},
		{"different tenant", "/api/tenant2/orders", "tenant2", false},
		{"wrong prefix", "/v1/tenant1/users", "", true},
		{"no tenant segment", "/api/", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)

			gotID, err := resolver.Resolve(req)
			if (err != nil) != tt.wantError {
				t.Errorf("expected error=%v, got error=%v (err: %v)", tt.wantError, err != nil, err)
			}
			if gotID != tt.wantID {
				t.Errorf("expected ID '%s', got '%s'", tt.wantID, gotID)
			}
		})
	}
}

func TestQueryParamResolver(t *testing.T) {
	resolver := NewQueryParamResolver("tenant")

	tests := []struct {
		name      string
		url       string
		wantID    string
		wantError bool
	}{
		{"valid param", "/api/users?tenant=tenant1", "tenant1", false},
		{"missing param", "/api/users", "", true},
		{"empty param", "/api/users?tenant=", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)

			gotID, err := resolver.Resolve(req)
			if (err != nil) != tt.wantError {
				t.Errorf("expected error=%v, got error=%v (err: %v)", tt.wantError, err != nil, err)
			}
			if gotID != tt.wantID {
				t.Errorf("expected ID '%s', got '%s'", tt.wantID, gotID)
			}
		})
	}
}

func TestChainResolver(t *testing.T) {
	chain := NewChainResolver(
		NewHeaderResolver("X-Tenant-ID"),
		NewQueryParamResolver("tenant"),
		NewStaticResolver("fallback-tenant"),
	)

	tests := []struct {
		name   string
		setup  func(*http.Request)
		wantID string
	}{
		{
			"header resolver succeeds",
			func(r *http.Request) {
				r.Header.Set("X-Tenant-ID", "header-tenant")
			},
			"header-tenant",
		},
		{
			"query param resolver succeeds",
			func(r *http.Request) {
				// No header, but has query param
			},
			"query-tenant",
		},
		{
			"fallback to static",
			func(r *http.Request) {
				// No header, no query param
			},
			"fallback-tenant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.name == "query param resolver succeeds" {
				req = httptest.NewRequest("GET", "/test?tenant=query-tenant", nil)
			} else {
				req = httptest.NewRequest("GET", "/test", nil)
			}

			tt.setup(req)

			gotID, err := chain.Resolve(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotID != tt.wantID {
				t.Errorf("expected ID '%s', got '%s'", tt.wantID, gotID)
			}
		})
	}
}

func TestStaticResolver(t *testing.T) {
	resolver := NewStaticResolver("static-tenant")

	req := httptest.NewRequest("GET", "/test", nil)
	gotID, err := resolver.Resolve(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotID != "static-tenant" {
		t.Errorf("expected ID 'static-tenant', got '%s'", gotID)
	}
}
