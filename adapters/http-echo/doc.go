package httpecho

// Package httpecho provides TenantKit middleware for the Echo web framework.
//
// It extracts tenant IDs from incoming HTTP requests and injects them into
// the request context for automatic tenant-scoped database queries.
//
// # Usage
//
//	e := echo.New()
//	e.Use(httpecho.TenantMiddleware(httpecho.Config{
//	    Resolver: httpecho.NewHeaderResolver("X-Tenant-ID"),
//	}))
