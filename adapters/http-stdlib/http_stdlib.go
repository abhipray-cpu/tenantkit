// Package httpstd provides HTTP middleware adapter for standard library net/http
// Package httpstd provides HTTP middleware and utilities for tenant resolution using only the Go standard library.
// It integrates with the tenantkit domain model to provide automatic tenant context propagation through HTTP requests.
//
// # Features
//
// - Multiple tenant resolution strategies: Subdomain, Header, Path, JWT
// - Chainable resolvers for fallback behavior
// - Middleware for automatic tenant context injection
// - Request context helpers for accessing tenant information
// - Zero external dependencies (stdlib only)
//
// # Usage Example
//
//	package main
//
//	import (
//		"net/http"
//		"github.com/abhipray-cpu/tenantkit/adapters/http-stdlib"
//	)
//
//	func main() {
//		// Create a header-based resolver
//		resolver := httpstd.NewHeaderResolver("X-Tenant-ID")
//
//		// Create middleware
//		mw, err := httpstd.NewMiddleware(httpstd.Config{
//			Resolver: resolver,
//		})
//		if err != nil {
//			panic(err)
//		}
//
//		// Wrap your handler
//		handler := mw.Handler(http.HandlerFunc(myHandler))
//
//		// Start server
//		http.ListenAndServe(":8080", handler)
//	}
//
//	func myHandler(w http.ResponseWriter, r *http.Request) {
//		// Get tenant ID from request
//		tenantID, err := httpstd.GetTenantID(r)
//		if err != nil {
//			http.Error(w, err.Error(), http.StatusInternalServerError)
//			return
//		}
//
//		w.Write([]byte("Tenant: " + tenantID))
//	}
//
// # Resolver Types
//
// - SubdomainResolver: Extracts tenant from URL subdomain (e.g., tenant.example.com)
// - HeaderResolver: Extracts tenant from HTTP header (e.g., X-Tenant-ID)
// - PathResolver: Extracts tenant from URL path (e.g., /tenants/tenant-id/...)
// - JWTResolver: Extracts tenant from JWT claims
//
// Resolvers can be chained using ChainResolvers for fallback behavior.
package httpstd
