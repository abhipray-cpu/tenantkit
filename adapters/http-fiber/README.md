# Fiber HTTP Adapter for TenantKit

Multi-tenancy middleware adapter for the [Fiber](https://github.com/gofiber/fiber) web framework.

## Installation

```bash
go get github.com/abhipray-cpu/tenantkit/adapters/http-fiber
```

## Quick Start

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    httpfiber "github.com/abhipray-cpu/tenantkit/adapters/http-fiber"
)

func main() {
    app := fiber.New()
    
    // Add tenant middleware
    app.Use(httpfiber.Middleware(&httpfiber.Config{
        Resolver: &httpfiber.HeaderResolver{},
    }))
    
    app.Get("/api/data", func(c *fiber.Ctx) error {
        tenantID, _ := httpfiber.GetTenantID(c)
        return c.JSON(fiber.Map{"tenant": tenantID})
    })
    
    app.Listen(":3000")
}
```

## Tenant Resolvers

- **HeaderResolver** - Extract from HTTP header (default: `X-Tenant-ID`)
- **SubdomainResolver** - Extract from subdomain (e.g., `tenant1.example.com`)
- **PathResolver** - Extract from URL path segment by index
- **ParamResolver** - Extract from Fiber route parameter
- **QueryParamResolver** - Extract from query parameter
- **ChainResolver** - Try multiple resolvers with fallback
- **StaticResolver** - Fixed tenant ID (testing/development)

## Configuration

```go
config := &httpfiber.Config{
    Resolver: &httpfiber.HeaderResolver{
        HeaderName: "X-Custom-Tenant",
    },
    SkipPaths: []string{"/health", "/metrics"},
    ErrorHandler: func(c *fiber.Ctx, err error) error {
        return c.Status(400).JSON(fiber.Map{"error": err.Error()})
    },
}

app.Use(httpfiber.Middleware(config))
```

## Helper Functions

```go
// Get tenant ID (returns error if not found)
tenantID, err := httpfiber.GetTenantID(c)

// Get tenant ID (panics if not found)
tenantID := httpfiber.MustGetTenantID(c)

// Get full tenant context
ctx, err := httpfiber.GetTenantContext(c)

// Set tenant ID (useful for testing)
httpfiber.WithTenantID(c, "test-tenant")
```

## Module Independence

This adapter has **zero dependencies** on other TenantKit HTTP adapters. Choose only the framework you need:

```bash
# Fiber only - no Chi/Echo/Gin dependencies
go get github.com/abhipray-cpu/tenantkit/adapters/http-fiber
```

## License

MIT
