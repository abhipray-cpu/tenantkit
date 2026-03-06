package httpgin

// Package httpgin provides TenantKit middleware for the Gin web framework.
//
// It extracts tenant IDs from incoming HTTP requests and injects them into
// the request context for automatic tenant-scoped database queries.
//
// # Usage
//
//	r := gin.Default()
//	r.Use(httpgin.TenantMiddleware(httpgin.Config{
//	    Resolver: httpgin.NewHeaderResolver("X-Tenant-ID"),
//	}))
