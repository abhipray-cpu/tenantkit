package ports

import (
	"net/http"
)

// Resolver is a port interface for extracting tenant ID from HTTP requests.
// Different implementations can resolve tenant ID from different sources:
// - Subdomain: tenant.example.com
// - Header: X-Tenant-ID: tenant-id
// - Path: /tenants/tenant-id/...
// - JWT claims: claims.tenant_id
type Resolver interface {
	// Resolve extracts the tenant ID from an HTTP request.
	// Returns the tenant ID string or an error if resolution fails.
	Resolve(r *http.Request) (string, error)
}
