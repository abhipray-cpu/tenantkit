package httpstd

import (
	"fmt"
	"net/http"
	"strings"
)

// ResolverType defines the strategy for extracting tenant ID from requests
type ResolverType int

const (
	// ResolverTypeSubdomain extracts tenant from subdomain (e.g., tenant.example.com)
	ResolverTypeSubdomain ResolverType = iota
	// ResolverTypeHeader extracts tenant from HTTP header (e.g., X-Tenant-ID)
	ResolverTypeHeader
	// ResolverTypePath extracts tenant from URL path (e.g., /tenants/tenant-id/...)
	ResolverTypePath
	// ResolverTypeJWT extracts tenant from JWT claims
	ResolverTypeJWT
)

// Resolver defines the interface for tenant resolution strategies
type Resolver interface {
	Resolve(r *http.Request) (string, error)
}

// SubdomainResolver extracts tenant ID from the subdomain
// Example: tenant123.example.com -> tenant123
type SubdomainResolver struct {
	// Domain is the root domain (e.g., example.com)
	// The subdomain will be extracted from anything before this domain
	Domain string
}

// NewSubdomainResolver creates a new subdomain-based resolver
func NewSubdomainResolver(domain string) *SubdomainResolver {
	return &SubdomainResolver{
		Domain: domain,
	}
}

// Resolve extracts tenant ID from the request's subdomain
func (r *SubdomainResolver) Resolve(req *http.Request) (string, error) {
	host := req.Host
	if host == "" {
		host = req.Header.Get("Host")
	}

	if host == "" {
		return "", fmt.Errorf("host header not found in request")
	}

	// Remove port if present
	host = strings.Split(host, ":")[0]

	// If the host is just the domain, no subdomain
	if host == r.Domain {
		return "", fmt.Errorf("no tenant subdomain found in host: %s", host)
	}

	// STRICT: Host MUST end with "."+domain (e.g., "tenant.example.com")
	// This prevents suffix attacks like "example.com.fake.com"
	if !strings.HasSuffix(host, "."+r.Domain) {
		return "", fmt.Errorf("host %s does not belong to domain %s", host, r.Domain)
	}

	// Extract subdomain
	fullSubdomain := strings.TrimSuffix(host, "."+r.Domain)
	if fullSubdomain == "" {
		return "", fmt.Errorf("empty subdomain extracted from host: %s", host)
	}

	// STRICT: Only allow single-level subdomains (no nested subdomains like "api.tenant.example.com")
	// The subdomain must not contain additional dots
	if strings.Contains(fullSubdomain, ".") {
		return "", fmt.Errorf("nested subdomains not allowed, host: %s", host)
	}

	return fullSubdomain, nil
}

// HeaderResolver extracts tenant ID from an HTTP header
// Example: X-Tenant-ID: tenant123 -> tenant123
type HeaderResolver struct {
	// HeaderName is the name of the header containing the tenant ID
	HeaderName string
}

// NewHeaderResolver creates a new header-based resolver
func NewHeaderResolver(headerName string) *HeaderResolver {
	if headerName == "" {
		headerName = "X-Tenant-ID"
	}
	return &HeaderResolver{
		HeaderName: headerName,
	}
}

// Resolve extracts tenant ID from the request header
func (r *HeaderResolver) Resolve(req *http.Request) (string, error) {
	tenantID := req.Header.Get(r.HeaderName)
	if tenantID == "" {
		return "", fmt.Errorf("header %s not found or empty in request", r.HeaderName)
	}

	// FIX BUG #22: Validate tenant ID is not whitespace-only after trimming
	trimmed := strings.TrimSpace(tenantID)
	if trimmed == "" {
		return "", fmt.Errorf("header %s contains only whitespace", r.HeaderName)
	}

	return trimmed, nil
}

// PathResolver extracts tenant ID from the URL path
// Example: /tenants/tenant123/... -> tenant123
type PathResolver struct {
	// PathSegment indicates which path segment contains the tenant ID
	// For /tenants/tenant123/users, use PathSegment=1
	PathSegment int
	// Prefix is the path prefix before the tenant ID (e.g., "/tenants")
	Prefix string
}

// NewPathResolver creates a new path-based resolver
// pathSegment indicates which path segment (0-indexed) contains the tenant ID
// prefix is the path prefix (e.g., "/tenants")
func NewPathResolver(prefix string, pathSegment int) *PathResolver {
	return &PathResolver{
		Prefix:      prefix,
		PathSegment: pathSegment,
	}
}

// Resolve extracts tenant ID from the request path
func (r *PathResolver) Resolve(req *http.Request) (string, error) {
	path := strings.TrimPrefix(req.URL.Path, "/")
	segments := strings.Split(path, "/")

	// Check if prefix matches
	prefixStart := 0
	if r.Prefix != "" {
		expectedPrefix := strings.TrimPrefix(r.Prefix, "/")
		prefixSegments := strings.Split(expectedPrefix, "/")

		// Check if path starts with all prefix segments
		if len(segments) < len(prefixSegments) {
			return "", fmt.Errorf("path does not have enough segments for prefix: %s", r.Prefix)
		}

		for i, prefixSeg := range prefixSegments {
			if segments[i] != prefixSeg {
				return "", fmt.Errorf("path does not start with expected prefix: %s", r.Prefix)
			}
		}
		// Adjust segments to skip prefix
		prefixStart = len(prefixSegments)
	}

	// Calculate the actual segment index
	segmentIndex := prefixStart + r.PathSegment

	// Check if requested segment exists
	if segmentIndex >= len(segments) {
		return "", fmt.Errorf("path segment %d not found in path: %s", r.PathSegment, req.URL.Path)
	}

	tenantID := strings.TrimSpace(segments[segmentIndex])
	if tenantID == "" {
		return "", fmt.Errorf("empty tenant ID at path segment %d in path: %s", r.PathSegment, req.URL.Path)
	}

	return tenantID, nil
}

// JWTResolver extracts tenant ID from JWT claims
// This is a basic implementation that expects a "tenant_id" claim
type JWTResolver struct {
	// ClaimName is the name of the claim containing tenant ID (default: "tenant_id")
	ClaimName string
	// TokenExtractor is a function to extract JWT token from request
	TokenExtractor func(*http.Request) (string, error)
	// ClaimParser is a function to parse JWT and extract the claim
	ClaimParser func(token string, claimName string) (string, error)
}

// NewJWTResolver creates a new JWT-based resolver
func NewJWTResolver(claimName string, tokenExtractor func(*http.Request) (string, error), claimParser func(string, string) (string, error)) *JWTResolver {
	if claimName == "" {
		claimName = "tenant_id"
	}
	return &JWTResolver{
		ClaimName:      claimName,
		TokenExtractor: tokenExtractor,
		ClaimParser:    claimParser,
	}
}

// Resolve extracts tenant ID from JWT claims in the request
func (r *JWTResolver) Resolve(req *http.Request) (string, error) {
	if r.TokenExtractor == nil {
		return "", fmt.Errorf("JWT token extractor not configured")
	}

	if r.ClaimParser == nil {
		return "", fmt.Errorf("JWT claim parser not configured")
	}

	token, err := r.TokenExtractor(req)
	if err != nil {
		return "", fmt.Errorf("failed to extract JWT token: %w", err)
	}

	if token == "" {
		return "", fmt.Errorf("JWT token not found in request")
	}

	tenantID, err := r.ClaimParser(token, r.ClaimName)
	if err != nil {
		return "", fmt.Errorf("failed to extract tenant claim from JWT: %w", err)
	}

	if tenantID == "" {
		return "", fmt.Errorf("tenant ID claim %s is empty in JWT", r.ClaimName)
	}

	return tenantID, nil
}

// ExtractBearerToken extracts the JWT token from the Authorization header
// Standard format: "Bearer <token>"
func ExtractBearerToken(req *http.Request) (string, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Authorization header not found")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid Authorization header format, expected 'Bearer <token>'")
	}

	return parts[1], nil
}

// ChainResolvers tries multiple resolvers in order, returning the first successful tenant ID
func ChainResolvers(resolvers ...Resolver) Resolver {
	return &chainResolver{resolvers: resolvers}
}

type chainResolver struct {
	resolvers []Resolver
}

func (c *chainResolver) Resolve(req *http.Request) (string, error) {
	var lastErr error
	for _, resolver := range c.resolvers {
		if tenantID, err := resolver.Resolve(req); err == nil {
			return tenantID, nil
		} else {
			lastErr = err
		}
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("no resolvers available")
}
