package chi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// HeaderResolver extracts tenant ID from an HTTP header
type HeaderResolver struct {
	HeaderName string
}

// NewHeaderResolver creates a resolver that extracts tenant ID from a header
func NewHeaderResolver(headerName string) *HeaderResolver {
	return &HeaderResolver{
		HeaderName: headerName,
	}
}

// Resolve extracts tenant ID from the specified header
func (h *HeaderResolver) Resolve(r *http.Request) (string, error) {
	tenantID := r.Header.Get(h.HeaderName)
	if tenantID == "" {
		return "", fmt.Errorf("header %s not found", h.HeaderName)
	}
	return tenantID, nil
}

// SubdomainResolver extracts tenant ID from subdomain
// Example: tenant1.example.com -> tenant1
type SubdomainResolver struct {
	// BaseDomain is the base domain to strip (e.g., "example.com")
	BaseDomain string
}

// NewSubdomainResolver creates a resolver that extracts tenant ID from subdomain
func NewSubdomainResolver(baseDomain string) *SubdomainResolver {
	return &SubdomainResolver{
		BaseDomain: baseDomain,
	}
}

// Resolve extracts tenant ID from subdomain
func (s *SubdomainResolver) Resolve(r *http.Request) (string, error) {
	host := r.Host

	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Split into parts
	parts := strings.Split(host, ".")

	// If we have a base domain, check if host ends with it
	if s.BaseDomain != "" {
		// Remove base domain from consideration
		baseParts := strings.Split(s.BaseDomain, ".")
		if len(parts) <= len(baseParts) {
			return "", fmt.Errorf("no subdomain found in host: %s", r.Host)
		}

		// Check if host ends with base domain
		expectedSuffix := "." + s.BaseDomain
		if !strings.HasSuffix(host, expectedSuffix) {
			return "", fmt.Errorf("host %s does not end with base domain %s", host, s.BaseDomain)
		}

		// Get subdomain part (everything before base domain)
		subdomain := strings.TrimSuffix(host, expectedSuffix)
		if subdomain == "" {
			return "", fmt.Errorf("empty subdomain in host: %s", r.Host)
		}

		return subdomain, nil
	}

	// No base domain configured - treat first part as tenant
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("no subdomain found in host: %s", r.Host)
	}

	return parts[0], nil
}

// PathResolver extracts tenant ID from URL path
// Example: /tenants/tenant1/users -> tenant1
type PathResolver struct {
	// PathPrefix is the prefix before tenant ID (e.g., "/tenants/")
	PathPrefix string
	// PathSegment is the 0-based segment index (e.g., 1 for /tenants/{tenant}/...)
	PathSegment int
}

// NewPathResolver creates a resolver that extracts tenant ID from URL path
// segment is 0-based index (e.g., for /api/tenants/{tenant}/users, segment would be 2)
func NewPathResolver(pathPrefix string, segment int) *PathResolver {
	return &PathResolver{
		PathPrefix:  pathPrefix,
		PathSegment: segment,
	}
}

// Resolve extracts tenant ID from URL path segment
func (p *PathResolver) Resolve(r *http.Request) (string, error) {
	path := r.URL.Path

	// Remove prefix if specified
	if p.PathPrefix != "" {
		if !strings.HasPrefix(path, p.PathPrefix) {
			return "", fmt.Errorf("path does not start with %s", p.PathPrefix)
		}
		path = strings.TrimPrefix(path, p.PathPrefix)
	}

	// Split into segments
	segments := strings.Split(strings.Trim(path, "/"), "/")

	if p.PathSegment >= len(segments) {
		return "", fmt.Errorf("path segment %d not found in %s", p.PathSegment, r.URL.Path)
	}

	tenantID := segments[p.PathSegment]
	if tenantID == "" {
		return "", fmt.Errorf("empty tenant ID in path segment %d", p.PathSegment)
	}

	return tenantID, nil
}

// URLParamResolver extracts tenant ID from Chi URL parameter
// Example: chi.Route("/tenants/{tenantID}", ...) -> extracts {tenantID}
type URLParamResolver struct {
	ParamName string
}

// NewURLParamResolver creates a resolver that extracts tenant ID from Chi URL parameter
func NewURLParamResolver(paramName string) *URLParamResolver {
	return &URLParamResolver{
		ParamName: paramName,
	}
}

// Resolve extracts tenant ID from Chi URL parameter
func (u *URLParamResolver) Resolve(r *http.Request) (string, error) {
	tenantID := chi.URLParam(r, u.ParamName)
	if tenantID == "" {
		return "", fmt.Errorf("URL parameter %s not found", u.ParamName)
	}
	return tenantID, nil
}

// QueryParamResolver extracts tenant ID from query parameter
// Example: /api/users?tenant=tenant1 -> tenant1
type QueryParamResolver struct {
	ParamName string
}

// NewQueryParamResolver creates a resolver that extracts tenant ID from query parameter
func NewQueryParamResolver(paramName string) *QueryParamResolver {
	return &QueryParamResolver{
		ParamName: paramName,
	}
}

// Resolve extracts tenant ID from query parameter
func (q *QueryParamResolver) Resolve(r *http.Request) (string, error) {
	tenantID := r.URL.Query().Get(q.ParamName)
	if tenantID == "" {
		return "", fmt.Errorf("query parameter %s not found", q.ParamName)
	}
	return tenantID, nil
}

// ChainResolver tries multiple resolvers in order until one succeeds
type ChainResolver struct {
	Resolvers []TenantResolver
}

// NewChainResolver creates a resolver that tries multiple strategies
func NewChainResolver(resolvers ...TenantResolver) *ChainResolver {
	return &ChainResolver{
		Resolvers: resolvers,
	}
}

// Resolve tries each resolver in order until one succeeds
func (c *ChainResolver) Resolve(r *http.Request) (string, error) {
	var errors []string

	for i, resolver := range c.Resolvers {
		tenantID, err := resolver.Resolve(r)
		if err == nil && tenantID != "" {
			return tenantID, nil
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("resolver %d: %v", i, err))
		}
	}

	if len(errors) > 0 {
		return "", fmt.Errorf("all resolvers failed: %s", strings.Join(errors, "; "))
	}

	return "", fmt.Errorf("no resolver found tenant ID")
}

// StaticResolver always returns a fixed tenant ID
// Useful for testing or single-tenant deployments
type StaticResolver struct {
	TenantID string
}

// NewStaticResolver creates a resolver that always returns the same tenant ID
func NewStaticResolver(tenantID string) *StaticResolver {
	return &StaticResolver{
		TenantID: tenantID,
	}
}

// Resolve always returns the configured tenant ID
func (s *StaticResolver) Resolve(r *http.Request) (string, error) {
	return s.TenantID, nil
}
