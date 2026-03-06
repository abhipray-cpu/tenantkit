// Package httpgin provides Gin middleware for multi-tenant applications.
// This adapter enables automatic tenant resolution and context injection for Gin framework.
package httpgin

import (
	"fmt"
	"net/http"

	"github.com/abhipray-cpu/tenantkit/domain"
	"github.com/gin-gonic/gin"
)

// TenantResolver defines the interface for extracting tenant ID from requests.
type TenantResolver interface {
	Resolve(c *gin.Context) (string, error)
}

// Config holds the middleware configuration.
type Config struct {
	// Resolver is used to extract the tenant ID from the request.
	// If nil, HeaderResolver with default header name is used.
	Resolver TenantResolver

	// ErrorHandler is called when tenant resolution fails.
	// If nil, default handler aborts with 400 Bad Request.
	ErrorHandler func(c *gin.Context, err error)

	// SkipPaths contains URL paths that should skip tenant resolution.
	// Useful for health checks, metrics endpoints, etc.
	SkipPaths []string

	// ContextKey is the key used to store tenant context in Gin context.
	// If not set, defaults to "tenant_context".
	ContextKey string
}

// Middleware returns a Gin middleware that resolves tenant ID and injects it into the context.
func Middleware(cfg *Config) gin.HandlerFunc {
	if cfg == nil {
		cfg = &Config{}
	}

	// Default resolver
	if cfg.Resolver == nil {
		cfg.Resolver = &HeaderResolver{HeaderName: "X-Tenant-ID"}
	}

	// Default error handler
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = func(c *gin.Context, err error) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("tenant resolution failed: %v", err),
			})
		}
	}

	// Default context key
	contextKey := cfg.ContextKey
	if contextKey == "" {
		contextKey = "tenant_context"
	}

	// Build skip paths map for O(1) lookup
	skipPaths := make(map[string]bool, len(cfg.SkipPaths))
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip tenant resolution for specified paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Resolve tenant ID
		tenantID, err := cfg.Resolver.Resolve(c)
		if err != nil {
			cfg.ErrorHandler(c, err)
			return
		}

		if tenantID == "" {
			cfg.ErrorHandler(c, fmt.Errorf("tenant ID not found"))
			return
		}

		// Create tenant context
		requestID := fmt.Sprintf("%s-req", tenantID)

		tenantCtx, err := domain.NewContext(tenantID, "http-request", requestID)
		if err != nil {
			cfg.ErrorHandler(c, fmt.Errorf("failed to create tenant context: %w", err))
			return
		}

		// Store in Gin context using Set
		c.Set(contextKey, tenantCtx)

		// Also inject into request context for compatibility with other libraries
		ctx := tenantCtx.ToGoContext(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// GetTenantContext extracts tenant context from Gin context.
// Returns error if tenant context is not found or invalid.
func GetTenantContext(c *gin.Context) (domain.Context, error) {
	// Try Gin context first
	if val, exists := c.Get("tenant_context"); exists {
		if tenantCtx, ok := val.(domain.Context); ok {
			return tenantCtx, nil
		}
	}

	// Fallback to request context using domain's FromGoContext
	return domain.FromGoContext(c.Request.Context())
}

// GetTenantID extracts tenant ID from Gin context.
// Returns error if tenant context is not found.
func GetTenantID(c *gin.Context) (string, error) {
	tenantCtx, err := GetTenantContext(c)
	if err != nil {
		return "", err
	}
	return tenantCtx.TenantID().Value(), nil
}

// MustGetTenantID extracts tenant ID from Gin context or panics if not found.
// Use this only in handlers where tenant context is guaranteed to be present.
func MustGetTenantID(c *gin.Context) string {
	tenantID, err := GetTenantID(c)
	if err != nil {
		panic(fmt.Sprintf("tenant context not found in Gin context: %v", err))
	}
	return tenantID
}

// WithTenantID creates a new Gin context with the given tenant ID.
// This is useful for testing or manual context creation.
func WithTenantID(c *gin.Context, tenantID string) {
	requestID := fmt.Sprintf("%s-test-req", tenantID)
	tenantCtx, err := domain.NewContext(tenantID, "http-request", requestID)
	if err != nil {
		panic(fmt.Sprintf("failed to create tenant context: %v", err))
	}

	c.Set("tenant_context", tenantCtx)

	// Also update request context
	ctx := tenantCtx.ToGoContext(c.Request.Context())
	c.Request = c.Request.WithContext(ctx)
}
