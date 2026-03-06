package httpstd

import (
	"fmt"
	"net/http"

	"github.com/abhipray-cpu/tenantkit/domain"
)

// Config holds configuration for the HTTP middleware
type Config struct {
	// Resolver is used to extract tenant ID from requests
	Resolver Resolver
	// OnError is called when tenant resolution fails
	// If not provided, defaults to responding with 400 Bad Request
	OnError ErrorHandler
	// SkipPaths is a list of URL paths that should skip tenant resolution
	SkipPaths []string
}

// ErrorHandler is a function that handles errors during tenant resolution
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// DefaultErrorHandler responds with 400 Bad Request for missing tenant
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, fmt.Sprintf("tenant resolution failed: %v", err), http.StatusBadRequest)
}

// Middleware wraps an HTTP handler to enforce tenant resolution
type Middleware struct {
	config Config
}

// NewMiddleware creates a new tenant resolution middleware
func NewMiddleware(config Config) (*Middleware, error) {
	if config.Resolver == nil {
		return nil, fmt.Errorf("resolver is required in middleware config")
	}

	if config.OnError == nil {
		config.OnError = DefaultErrorHandler
	}

	return &Middleware{
		config: config,
	}, nil
}

// Handler returns an HTTP handler that wraps the provided handler with tenant resolution
// The tenant ID is stored in the request context and can be accessed via GetTenantID
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this path should skip tenant resolution
		if m.shouldSkip(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Resolve tenant ID from request
		tenantID, err := m.config.Resolver.Resolve(r)
		if err != nil {
			m.config.OnError(w, r, err)
			return
		}

		// Create tenant context with default user and request IDs
		// These should be extracted from the request or provided separately in a real application
		tenantCtx, err := domain.NewContext(tenantID, "system", "http-request")
		if err != nil {
			m.config.OnError(w, r, fmt.Errorf("failed to create tenant context: %w", err))
			return
		}

		// Store in request context
		goCtx := tenantCtx.ToGoContext(r.Context())
		r = r.WithContext(goCtx)

		// Call next handler with enriched request
		next.ServeHTTP(w, r)
	})
}

// shouldSkip checks if the request path should skip tenant resolution
func (m *Middleware) shouldSkip(path string) bool {
	for _, skipPath := range m.config.SkipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// HandlerFunc wraps a handler function with tenant resolution
func (m *Middleware) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.Handler(next).ServeHTTP(w, r)
	}
}
