package httpstd

// Package httpstdlib provides TenantKit middleware for Go's standard net/http package.
//
// It extracts tenant IDs from incoming HTTP requests (headers, subdomains, JWT,
// URL paths) and injects them into the request context using [tenantkit.WithTenant].
//
// # Usage
//
//	resolver := httpstdlib.NewHeaderResolver("X-Tenant-ID")
//	mw := httpstdlib.NewMiddleware(httpstdlib.Config{Resolver: resolver})
//
//	http.Handle("/api/", mw.Handler(apiHandler))
