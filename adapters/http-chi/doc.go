package chi

// Package chi provides TenantKit middleware for the Chi router.
//
// It extracts tenant IDs from incoming HTTP requests and injects them into
// the request context for automatic tenant-scoped database queries.
//
// # Usage
//
//	r := chi.NewRouter()
//	r.Use(chi.TenantMiddleware(chi.Config{
//	    Resolver: chi.NewHeaderResolver("X-Tenant-ID"),
//	}))
