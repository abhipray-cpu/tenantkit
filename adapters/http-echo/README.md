# Echo Middleware Adapter

Multi-tenant middleware for [Echo](https://echo.labstack.com/) web framework.

## Installation

```bash
go get github.com/abhipray-cpu/tenantkit/adapters/http-echo
```

## Quick Start

```go
package main

import (
	"github.com/abhipray-cpu/tenantkit/adapters/http-echo"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	
	// Add tenant middleware
	e.Use(httpecho.Middleware(&httpecho.Config{
		Resolver: &httpecho.HeaderResolver{},
	}))
	
	e.GET("/users", func(c echo.Context) error {
		tenantID, err := httpecho.GetTenantID(c)
		if err != nil {
			return err
		}
		// Use tenantID to filter users...
		return c.JSON(200, map[string]string{"tenant": tenantID})
	})
	
	e.Start(":8080")
}
```

## Tenant Resolution Strategies

### 1. HeaderResolver (Default)
Extract tenant ID from HTTP header.

```go
httpecho.Middleware(&httpecho.Config{
	Resolver: &httpecho.HeaderResolver{
		HeaderName: "X-Tenant-ID", // default
	},
})
```

### 2. SubdomainResolver
Extract from subdomain (tenant1.example.com → tenant1).

```go
httpecho.Middleware(&httpecho.Config{
	Resolver: &httpecho.SubdomainResolver{
		BaseDomain: "example.com",
	},
})
```

### 3. PathResolver
Extract from URL path segment.

```go
httpecho.Middleware(&httpecho.Config{
	Resolver: &httpecho.PathResolver{
		PathIndex: 1, // /tenants/:tenantID/...
	},
})
```

### 4. ParamResolver
Extract from Echo URL parameter.

```go
e.GET("/tenants/:tenantID/users", handler)

httpecho.Middleware(&httpecho.Config{
	Resolver: &httpecho.ParamResolver{
		ParamName: "tenantID",
	},
})
```

### 5. QueryParamResolver
Extract from query parameter.

```go
httpecho.Middleware(&httpecho.Config{
	Resolver: &httpecho.QueryParamResolver{
		ParamName: "tenant", // ?tenant=tenant1
	},
})
```

### 6. ChainResolver
Try multiple strategies with fallback.

```go
httpecho.Middleware(&httpecho.Config{
	Resolver: &httpecho.ChainResolver{
		Resolvers: []httpecho.TenantResolver{
			&httpecho.HeaderResolver{},
			&httpecho.QueryParamResolver{},
			&httpecho.StaticResolver{TenantID: "default"},
		},
	},
})
```

### 7. StaticResolver
Always return fixed tenant ID (testing/development).

```go
httpecho.Middleware(&httpecho.Config{
	Resolver: &httpecho.StaticResolver{
		TenantID: "test-tenant",
	},
})
```

## Configuration

```go
httpecho.Middleware(&httpecho.Config{
	Resolver: &httpecho.HeaderResolver{},
	
	// Custom error handler
	ErrorHandler: func(c echo.Context, err error) error {
		return echo.NewHTTPError(401, "Unauthorized")
	},
	
	// Skip tenant resolution for these paths
	SkipPaths: []string{"/health", "/metrics"},
	
	// Custom context key
	ContextKey: "my_tenant_context",
})
```

## Helper Functions

```go
// Get tenant context (returns error if not found)
tenantCtx, err := httpecho.GetTenantContext(c)

// Get tenant ID (returns error if not found)
tenantID, err := httpecho.GetTenantID(c)

// Get tenant ID or panic
tenantID := httpecho.MustGetTenantID(c)

// Set tenant ID for testing
c = httpecho.WithTenantID(c, "test-tenant")
```

## Integration with Database Adapters

```go
import (
	"github.com/abhipray-cpu/tenantkit/adapters/http-echo"
	tenantsqlx "github.com/abhipray-cpu/tenantkit/adapters/sqlx"
)

func handler(c echo.Context) error {
	tenantCtx, _ := httpecho.GetTenantContext(c)
	
	// Database automatically filters by tenant
	ctx := tenantCtx.ToGoContext(c.Request().Context())
	users, err := db.QueryContext(ctx, "SELECT * FROM users")
	// Only returns users for this tenant
	
	return c.JSON(200, users)
}
```

## Testing

```go
func TestHandler(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	
	// Set tenant for testing
	c = httpecho.WithTenantID(c, "test-tenant")
	
	// Test your handler
	err := yourHandler(c)
	// ...
}
```

## Best Practices

1. **Always check errors**: `GetTenantID()` returns an error if tenant context is missing
2. **Use MustGetTenantID() carefully**: Only in handlers where tenant is guaranteed
3. **Skip health checks**: Add `/health`, `/metrics` to SkipPaths
4. **Use ChainResolver**: Provide fallback strategies for flexibility
5. **Production**: Use HeaderResolver or SubdomainResolver, not StaticResolver

## Module Independence

This adapter ONLY depends on Echo. Echo users never download Gin, Fiber, Chi, etc.

```bash
$ go mod graph | grep tenantkit
github.com/abhipray-cpu/tenantkit/adapters/http-echo github.com/labstack/echo/v4@v4.15.0
# No other framework dependencies!
```

## Performance

- Middleware overhead: ~0.5ms per request
- Resolver execution: ~0.1ms
- Total impact: < 1ms

## License

MIT License - see LICENSE file for details
