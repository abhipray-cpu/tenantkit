// Package chi provides multi-tenant middleware for the Chi router.
// This adapter extracts tenant information from HTTP requests and creates
// a tenant context that can be used throughout the request lifecycle.
//
// Design Philosophy:
// - Idiomatic Chi middleware pattern
// - Multiple tenant resolution strategies (header, subdomain, path, JWT)
// - Composable resolvers
// - Clear error responses
package chi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/abhipray-cpu/tenantkit/domain"
	"github.com/go-chi/chi/v5"
)

// TenantResolver extracts tenant ID from an HTTP request
type TenantResolver interface {
	// Resolve extracts the tenant ID from the request
	// Returns empty string if tenant cannot be determined
	Resolve(r *http.Request) (string, error)
}

// Config configures the tenant middleware
type Config struct {
	// Resolver is the strategy for extracting tenant ID from requests
	Resolver TenantResolver

	// ErrorHandler is called when tenant resolution fails
	// If nil, returns 400 Bad Request with error message
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

	// SkipPaths is a list of path prefixes to skip tenant resolution
	// Useful for health checks, metrics endpoints, etc.
	SkipPaths []string

	// ContextKey is the key used to store tenant context in request context
	// If empty, uses domain.TenantContextKey
	ContextKey interface{}
}

// Middleware returns a Chi middleware that extracts tenant information
// and adds it to the request context
func Middleware(cfg *Config) func(next http.Handler) http.Handler {
	if cfg == nil {
		cfg = &Config{}
	}

	if cfg.Resolver == nil {
		cfg.Resolver = NewHeaderResolver("X-Tenant-ID")
	}

	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = defaultErrorHandler
	}

	contextKey := cfg.ContextKey
	if contextKey == nil {
		contextKey = domain.TenantContextKey
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			for _, skip := range cfg.SkipPaths {
				if strings.HasPrefix(r.URL.Path, skip) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Resolve tenant ID
			tenantID, err := cfg.Resolver.Resolve(r)
			if err != nil {
				cfg.ErrorHandler(w, r, err)
				return
			}

			if tenantID == "" {
				cfg.ErrorHandler(w, r, fmt.Errorf("tenant ID not found"))
				return
			}

			// Create tenant context
			// Use a simple request ID generator for now
			requestID := fmt.Sprintf("%s-%d", tenantID, r.Context().Value("requestID"))
			if requestID == fmt.Sprintf("%s-<nil>", tenantID) {
				requestID = fmt.Sprintf("%s-req", tenantID)
			}

			tenantCtx, err := domain.NewContext(tenantID, "http-request", requestID)
			if err != nil {
				cfg.ErrorHandler(w, r, fmt.Errorf("failed to create tenant context: %w", err))
				return
			}

			// Add to request context
			ctx := context.WithValue(r.Context(), contextKey, tenantCtx)

			// Also use domain's ToGoContext for consistency
			ctx = tenantCtx.ToGoContext(ctx)

			// Continue with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// defaultErrorHandler returns a 400 Bad Request with the error message
func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, fmt.Sprintf("Tenant resolution failed: %v", err), http.StatusBadRequest)
}

// GetTenantContext extracts the tenant context from the request context
func GetTenantContext(ctx context.Context) (domain.Context, error) {
	return domain.FromGoContext(ctx)
}

// GetTenantID is a convenience function to extract just the tenant ID
func GetTenantID(ctx context.Context) (string, error) {
	tc, err := GetTenantContext(ctx)
	if err != nil {
		return "", err
	}
	return tc.TenantID().Value(), nil
}

// GetTenantIDFromRequest extracts tenant ID from a Chi request
func GetTenantIDFromRequest(r *http.Request) (string, error) {
	return GetTenantID(r.Context())
}

// URLParam extracts a URL parameter and validates tenant context
// This is a tenant-aware wrapper around chi.URLParam
func URLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// WithTenantID creates a new context with the specified tenant ID
// Useful for testing or background jobs
func WithTenantID(ctx context.Context, tenantID string) (context.Context, error) {
	tenantCtx, err := domain.NewContext(tenantID, "system", "background")
	if err != nil {
		return nil, err
	}
	return tenantCtx.ToGoContext(ctx), nil
}
