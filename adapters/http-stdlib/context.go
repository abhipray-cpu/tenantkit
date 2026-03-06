package httpstd

import (
	"fmt"
	"net/http"

	"github.com/abhipray-cpu/tenantkit/domain"
)

// GetTenantID extracts the tenant ID from the request context
// Returns an error if the tenant context is not found
func GetTenantID(r *http.Request) (string, error) {
	tenantCtx, err := domain.FromGoContext(r.Context())
	if err != nil {
		return "", fmt.Errorf("tenant context not found in request: %w", err)
	}
	return tenantCtx.TenantID().Value(), nil
}

// GetTenantContext extracts the full tenant context from the request
// Returns an error if the tenant context is not found
func GetTenantContext(r *http.Request) (domain.Context, error) {
	tenantCtx, err := domain.FromGoContext(r.Context())
	if err != nil {
		return domain.Context{}, fmt.Errorf("tenant context not found in request: %w", err)
	}
	return tenantCtx, nil
}

// MustGetTenantID extracts the tenant ID from the request context
// Panics if the tenant context is not found
func MustGetTenantID(r *http.Request) string {
	tenantID, err := GetTenantID(r)
	if err != nil {
		panic(err)
	}
	return tenantID
}

// MustGetTenantContext extracts the full tenant context from the request
// Panics if the tenant context is not found
func MustGetTenantContext(r *http.Request) domain.Context {
	tenantCtx, err := GetTenantContext(r)
	if err != nil {
		panic(err)
	}
	return tenantCtx
}

// WithTenantID returns a new request with the tenant context set
// Useful for creating new requests with the same tenant
func WithTenantID(r *http.Request, tenantID string) (*http.Request, error) {
	tenantCtx, err := domain.NewContext(tenantID, "system", "http-context")
	if err != nil {
		return nil, err
	}

	goCtx := tenantCtx.ToGoContext(r.Context())
	return r.WithContext(goCtx), nil
}
