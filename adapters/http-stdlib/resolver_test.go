package httpstd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSubdomainResolver tests the subdomain resolver
func TestSubdomainResolver(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		host        string
		wantTenant  string
		wantError   bool
		description string
	}{
		{
			name:        "valid_subdomain",
			domain:      "example.com",
			host:        "tenant1.example.com",
			wantTenant:  "tenant1",
			wantError:   false,
			description: "Should extract tenant from subdomain",
		},
		{
			name:        "multi_level_subdomain",
			domain:      "example.com",
			host:        "api.tenant1.example.com",
			wantTenant:  "",
			wantError:   true,
			description: "Should reject nested subdomains",
		},
		{
			name:        "no_subdomain",
			domain:      "example.com",
			host:        "example.com",
			wantTenant:  "",
			wantError:   true,
			description: "Should fail when no subdomain present",
		},
		{
			name:        "wrong_domain",
			domain:      "example.com",
			host:        "tenant1.wrong.com",
			wantTenant:  "",
			wantError:   true,
			description: "Should fail for wrong domain",
		},
		{
			name:        "with_port",
			domain:      "example.com",
			host:        "tenant1.example.com:8080",
			wantTenant:  "tenant1",
			wantError:   false,
			description: "Should extract tenant ignoring port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewSubdomainResolver(tt.domain)
			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host

			got, err := resolver.Resolve(req)

			if (err != nil) != tt.wantError {
				t.Errorf("%s: got error=%v, wantError=%v", tt.description, err, tt.wantError)
			}

			if got != tt.wantTenant {
				t.Errorf("%s: got tenant=%q, want=%q", tt.description, got, tt.wantTenant)
			}
		})
	}
}

// TestHeaderResolver tests the header resolver
func TestHeaderResolver(t *testing.T) {
	tests := []struct {
		name        string
		headerName  string
		headerValue string
		wantTenant  string
		wantError   bool
		description string
	}{
		{
			name:        "valid_header",
			headerName:  "X-Tenant-ID",
			headerValue: "tenant1",
			wantTenant:  "tenant1",
			wantError:   false,
			description: "Should extract tenant from header",
		},
		{
			name:        "custom_header",
			headerName:  "X-Custom-Tenant",
			headerValue: "custom-tenant",
			wantTenant:  "custom-tenant",
			wantError:   false,
			description: "Should support custom header names",
		},
		{
			name:        "missing_header",
			headerName:  "X-Tenant-ID",
			headerValue: "",
			wantTenant:  "",
			wantError:   true,
			description: "Should fail when header missing",
		},
		{
			name:        "header_with_whitespace",
			headerName:  "X-Tenant-ID",
			headerValue: "  tenant1  ",
			wantTenant:  "tenant1",
			wantError:   false,
			description: "Should trim whitespace from header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewHeaderResolver(tt.headerName)
			req := httptest.NewRequest("GET", "/", nil)
			if tt.headerValue != "" {
				req.Header.Set(tt.headerName, tt.headerValue)
			}

			got, err := resolver.Resolve(req)

			if (err != nil) != tt.wantError {
				t.Errorf("%s: got error=%v, wantError=%v", tt.description, err, tt.wantError)
			}

			if got != tt.wantTenant {
				t.Errorf("%s: got tenant=%q, want=%q", tt.description, got, tt.wantTenant)
			}
		})
	}
}

// TestPathResolver tests the path resolver
func TestPathResolver(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		segment     int
		path        string
		wantTenant  string
		wantError   bool
		description string
	}{
		{
			name:        "standard_path",
			prefix:      "/tenants",
			segment:     0,
			path:        "/tenants/tenant1/users",
			wantTenant:  "tenant1",
			wantError:   false,
			description: "Should extract tenant from standard path",
		},
		{
			name:        "nested_path",
			prefix:      "/api/v1/tenants",
			segment:     0,
			path:        "/api/v1/tenants/tenant1/data",
			wantTenant:  "tenant1",
			wantError:   false,
			description: "Should handle nested paths",
		},
		{
			name:        "tenant_at_different_segment",
			prefix:      "/api",
			segment:     1,
			path:        "/api/users/tenant1/info",
			wantTenant:  "tenant1",
			wantError:   false,
			description: "Should extract tenant from specified segment",
		},
		{
			name:        "missing_prefix",
			prefix:      "/tenants",
			segment:     0,
			path:        "/wrong/tenant1/users",
			wantTenant:  "",
			wantError:   true,
			description: "Should fail when prefix doesn't match",
		},
		{
			name:        "missing_segment",
			prefix:      "/tenants",
			segment:     5,
			path:        "/tenants/tenant1",
			wantTenant:  "",
			wantError:   true,
			description: "Should fail when segment index out of range",
		},
		{
			name:        "no_prefix",
			prefix:      "",
			segment:     0,
			path:        "/tenant1/users",
			wantTenant:  "tenant1",
			wantError:   false,
			description: "Should work without prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewPathResolver(tt.prefix, tt.segment)
			req := httptest.NewRequest("GET", tt.path, nil)

			got, err := resolver.Resolve(req)

			if (err != nil) != tt.wantError {
				t.Errorf("%s: got error=%v, wantError=%v", tt.description, err, tt.wantError)
			}

			if got != tt.wantTenant {
				t.Errorf("%s: got tenant=%q, want=%q", tt.description, got, tt.wantTenant)
			}
		})
	}
}

// TestJWTResolver tests the JWT resolver
func TestJWTResolver(t *testing.T) {
	// Mock token extractor
	tokenExtractor := func(r *http.Request) (string, error) {
		return ExtractBearerToken(r)
	}

	// Mock claim parser - parses simple format: "tenant_id:value"
	claimParser := func(token string, claimName string) (string, error) {
		if token == "invalid" {
			return "", fmt.Errorf("invalid token")
		}
		if strings.HasPrefix(token, claimName+":") {
			return strings.TrimPrefix(token, claimName+":"), nil
		}
		return "", fmt.Errorf("claim %s not found in token", claimName)
	}

	tests := []struct {
		name        string
		authHeader  string
		wantTenant  string
		wantError   bool
		description string
	}{
		{
			name:        "valid_jwt",
			authHeader:  "Bearer tenant_id:tenant1",
			wantTenant:  "tenant1",
			wantError:   false,
			description: "Should extract tenant from JWT",
		},
		{
			name:        "missing_auth_header",
			authHeader:  "",
			wantTenant:  "",
			wantError:   true,
			description: "Should fail when Authorization header missing",
		},
		{
			name:        "invalid_bearer_format",
			authHeader:  "Bearer",
			wantTenant:  "",
			wantError:   true,
			description: "Should fail with invalid Bearer format",
		},
		{
			name:        "invalid_token",
			authHeader:  "Bearer invalid",
			wantTenant:  "",
			wantError:   true,
			description: "Should fail with invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewJWTResolver("tenant_id", tokenExtractor, claimParser)
			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			got, err := resolver.Resolve(req)

			if (err != nil) != tt.wantError {
				t.Errorf("%s: got error=%v, wantError=%v", tt.description, err, tt.wantError)
			}

			if got != tt.wantTenant {
				t.Errorf("%s: got tenant=%q, want=%q", tt.description, got, tt.wantTenant)
			}
		})
	}
}

// TestExtractBearerToken tests the bearer token extraction
func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		wantToken   string
		wantError   bool
		description string
	}{
		{
			name:        "valid_bearer",
			authHeader:  "Bearer mytoken123",
			wantToken:   "mytoken123",
			wantError:   false,
			description: "Should extract bearer token",
		},
		{
			name:        "missing_header",
			authHeader:  "",
			wantToken:   "",
			wantError:   true,
			description: "Should fail when header missing",
		},
		{
			name:        "invalid_format",
			authHeader:  "Basic dXNlcjpwYXNz",
			wantToken:   "",
			wantError:   true,
			description: "Should fail with non-Bearer auth",
		},
		{
			name:        "malformed_bearer",
			authHeader:  "Bearer",
			wantToken:   "",
			wantError:   true,
			description: "Should fail with incomplete Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			got, err := ExtractBearerToken(req)

			if (err != nil) != tt.wantError {
				t.Errorf("%s: got error=%v, wantError=%v", tt.description, err, tt.wantError)
			}

			if got != tt.wantToken {
				t.Errorf("%s: got token=%q, want=%q", tt.description, got, tt.wantToken)
			}
		})
	}
}

// TestChainResolvers tests the chained resolver
func TestChainResolvers(t *testing.T) {
	headerResolver := NewHeaderResolver("X-Tenant-ID")
	pathResolver := NewPathResolver("/tenants", 0)

	tests := []struct {
		name        string
		setup       func(*http.Request)
		wantTenant  string
		wantError   bool
		description string
	}{
		{
			name: "first_resolver_success",
			setup: func(r *http.Request) {
				r.Header.Set("X-Tenant-ID", "tenant1")
			},
			wantTenant:  "tenant1",
			wantError:   false,
			description: "Should use first successful resolver",
		},
		{
			name: "second_resolver_fallback",
			setup: func(r *http.Request) {
				r.URL.Path = "/tenants/tenant2/users"
			},
			wantTenant:  "tenant2",
			wantError:   false,
			description: "Should fallback to second resolver",
		},
		{
			name: "both_fail",
			setup: func(r *http.Request) {
				// No header, no path
			},
			wantTenant:  "",
			wantError:   true,
			description: "Should fail when all resolvers fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := ChainResolvers(headerResolver, pathResolver)
			req := httptest.NewRequest("GET", "/", nil)
			tt.setup(req)

			got, err := resolver.Resolve(req)

			if (err != nil) != tt.wantError {
				t.Errorf("%s: got error=%v, wantError=%v", tt.description, err, tt.wantError)
			}

			if got != tt.wantTenant {
				t.Errorf("%s: got tenant=%q, want=%q", tt.description, got, tt.wantTenant)
			}
		})
	}
}
