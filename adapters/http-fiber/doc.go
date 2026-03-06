package httpfiber

// Package httpfiber provides TenantKit middleware for the Fiber web framework.
//
// It extracts tenant IDs from incoming HTTP requests and injects them into
// the request context for automatic tenant-scoped database queries.
//
// # Usage
//
//	app := fiber.New()
//	app.Use(httpfiber.TenantMiddleware(httpfiber.Config{
//	    Resolver: httpfiber.NewHeaderResolver("X-Tenant-ID"),
//	}))
