# Gin Middleware Adapter

Multi-tenant middleware for [Gin](https://gin-gonic.com/) web framework.

## Installation

```bash
go get github.com/abhipray-cpu/tenantkit/adapters/http-gin
```

## Quick Start

```go
package main

import (
	"github.com/abhipray-cpu/tenantkit/adapters/http-gin"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	
	// Add tenant middleware
	r.Use(httpgin.Middleware(&httpgin.Config{
		Resolver: &httpgin.HeaderResolver{},
	}))
	
	r.GET("/users", func(c *gin.Context) {
		tenantID, err := httpgin.GetTenantID(c)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"tenant": tenantID})
	})
	
	r.Run(":8080")
}
```

## Resolvers

- **HeaderResolver**: Extract from HTTP header (default: X-Tenant-ID)
- **SubdomainResolver**: Extract from subdomain (tenant1.example.com → tenant1)
- **PathResolver**: Extract from URL path segment
- **ParamResolver**: Extract from Gin URL parameter
- **QueryParamResolver**: Extract from query parameter
- **ChainResolver**: Try multiple strategies with fallback
- **StaticResolver**: Fixed tenant ID (testing)

## Configuration

```go
httpgin.Middleware(&httpgin.Config{
	Resolver:  &httpgin.HeaderResolver{},
	ErrorHandler: func(c *gin.Context, err error) {
		c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
	},
	SkipPaths: []string{"/health", "/metrics"},
})
```

## Helper Functions

```go
// Get tenant ID (returns error if not found)
tenantID, err := httpgin.GetTenantID(c)

// Get tenant ID or panic
tenantID := httpgin.MustGetTenantID(c)

// Set tenant ID for testing
httpgin.WithTenantID(c, "test-tenant")
```

## Module Independence

This adapter ONLY depends on Gin. Gin users never download Chi, Echo, Fiber, etc.

## License

MIT License
