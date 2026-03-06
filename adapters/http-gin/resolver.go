package httpgin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// Common errors for tenant resolution
var (
	ErrTenantNotFound = errors.New("tenant ID not found")
	ErrInvalidTenant  = errors.New("invalid tenant ID")
)

// HeaderResolver extracts tenant ID from HTTP header.
type HeaderResolver struct {
	// HeaderName is the name of the header containing the tenant ID.
	// Defaults to "X-Tenant-ID".
	HeaderName string
}

// Resolve extracts tenant ID from the specified header.
func (h *HeaderResolver) Resolve(c *gin.Context) (string, error) {
	headerName := h.HeaderName
	if headerName == "" {
		headerName = "X-Tenant-ID"
	}

	tenantID := c.GetHeader(headerName)
	if tenantID == "" {
		return "", fmt.Errorf("%w: header %s is empty", ErrTenantNotFound, headerName)
	}

	return tenantID, nil
}

// SubdomainResolver extracts tenant ID from subdomain.
// Example: tenant1.example.com -> tenant1
type SubdomainResolver struct {
	// BaseDomain is the base domain to strip from the host.
	// Example: "example.com" will extract "tenant1" from "tenant1.example.com"
	// If empty, returns the first part of the host.
	BaseDomain string
}

// Resolve extracts tenant ID from subdomain.
func (s *SubdomainResolver) Resolve(c *gin.Context) (string, error) {
	host := c.Request.Host

	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// If base domain is specified, extract subdomain
	if s.BaseDomain != "" {
		// Check if host ends with ".BaseDomain"
		suffix := "." + s.BaseDomain
		if !strings.HasSuffix(host, suffix) {
			return "", fmt.Errorf("%w: host %s does not end with base domain %s", ErrTenantNotFound, host, s.BaseDomain)
		}

		// Extract subdomain
		subdomain := strings.TrimSuffix(host, suffix)
		if subdomain == "" {
			return "", fmt.Errorf("%w: no subdomain found in host %s", ErrTenantNotFound, host)
		}

		return subdomain, nil
	}

	// No base domain - return first part of host
	parts := strings.Split(host, ".")
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("%w: cannot extract subdomain from host %s", ErrTenantNotFound, host)
	}

	return parts[0], nil
}

// PathResolver extracts tenant ID from URL path.
// Example: /tenants/tenant1/users -> tenant1
type PathResolver struct {
	// PathIndex is the zero-based index of the path segment containing tenant ID.
	// Example: For "/tenants/:tenantID/users", PathIndex should be 1.
	PathIndex int
}

// Resolve extracts tenant ID from URL path at the specified index.
func (p *PathResolver) Resolve(c *gin.Context) (string, error) {
	path := c.Request.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if p.PathIndex < 0 || p.PathIndex >= len(parts) {
		return "", fmt.Errorf("%w: path index %d out of range for path %s", ErrTenantNotFound, p.PathIndex, path)
	}

	tenantID := parts[p.PathIndex]
	if tenantID == "" {
		return "", fmt.Errorf("%w: path segment at index %d is empty", ErrTenantNotFound, p.PathIndex)
	}

	return tenantID, nil
}

// ParamResolver extracts tenant ID from Gin URL parameter.
// Example: Route "/tenants/:tenantID/users" -> c.Param("tenantID")
type ParamResolver struct {
	// ParamName is the name of the URL parameter containing tenant ID.
	// Example: "tenantID" for route "/tenants/:tenantID/users"
	ParamName string
}

// Resolve extracts tenant ID from Gin URL parameter.
func (p *ParamResolver) Resolve(c *gin.Context) (string, error) {
	paramName := p.ParamName
	if paramName == "" {
		paramName = "tenantID"
	}

	tenantID := c.Param(paramName)
	if tenantID == "" {
		return "", fmt.Errorf("%w: parameter %s is empty", ErrTenantNotFound, paramName)
	}

	return tenantID, nil
}

// QueryParamResolver extracts tenant ID from query parameter.
// Example: /api/users?tenant=tenant1 -> tenant1
type QueryParamResolver struct {
	// ParamName is the name of the query parameter containing tenant ID.
	// Defaults to "tenant".
	ParamName string
}

// Resolve extracts tenant ID from query parameter.
func (q *QueryParamResolver) Resolve(c *gin.Context) (string, error) {
	paramName := q.ParamName
	if paramName == "" {
		paramName = "tenant"
	}

	tenantID := c.Query(paramName)
	if tenantID == "" {
		return "", fmt.Errorf("%w: query parameter %s is empty", ErrTenantNotFound, paramName)
	}

	return tenantID, nil
}

// ChainResolver tries multiple resolvers in order until one succeeds.
// This is useful for supporting multiple tenant resolution strategies.
type ChainResolver struct {
	// Resolvers is the list of resolvers to try in order.
	Resolvers []TenantResolver
}

// Resolve tries each resolver in order until one succeeds.
func (ch *ChainResolver) Resolve(c *gin.Context) (string, error) {
	if len(ch.Resolvers) == 0 {
		return "", fmt.Errorf("%w: no resolvers configured", ErrTenantNotFound)
	}

	var lastErr error
	for _, resolver := range ch.Resolvers {
		tenantID, err := resolver.Resolve(c)
		if err == nil {
			return tenantID, nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("%w: all resolvers failed, last error: %v", ErrTenantNotFound, lastErr)
}

// StaticResolver always returns a fixed tenant ID.
// This is useful for testing or single-tenant deployments.
type StaticResolver struct {
	// TenantID is the tenant ID to always return.
	TenantID string
}

// Resolve returns the static tenant ID.
func (s *StaticResolver) Resolve(c *gin.Context) (string, error) {
	if s.TenantID == "" {
		return "", fmt.Errorf("%w: static tenant ID is empty", ErrTenantNotFound)
	}
	return s.TenantID, nil
}
