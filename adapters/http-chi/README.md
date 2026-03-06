# Chi Adapter for TenantKit

Multi-tenant HTTP middleware for the [Chi router](https://github.com/go-chi/chi).

## Features

- **Multiple Resolution Strategies**: Header, subdomain, path, query param, URL param, or custom
- **Composable Resolvers**: Chain multiple strategies with fallback
- **Idiomatic Chi**: Follows Chi middleware patterns
- **Skip Paths**: Exclude health checks, metrics, etc.
- **Custom Error Handling**: Full control over error responses
- **Zero Dependencies**: Only Chi and core domain

## Installation

```bash
go get github.com/abhipray-cpu/tenantkit/adapters/http-chi
```

**Important**: Separate module - Chi users never download Echo/Gin/Fiber.

## Quick Start

```go
package main

import (
    "net/http"
    
    tenantchi "github.com/abhipray-cpu/tenantkit/adapters/http-chi"
    "github.com/go-chi/chi/v5"
)

func main() {
    r := chi.NewRouter()
    
    // Add tenant middleware
    r.Use(tenantchi.Middleware(&tenantchi.Config{
        Resolver: tenantchi.NewHeaderResolver("X-Tenant-ID"),
    }))
    
    r.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
        // Extract tenant from context
        tenantID, _ := tenantchi.GetTenantIDFromRequest(r)
        
        // Your handler logic
        w.Write([]byte("Tenant: " + tenantID))
    })
    
    http.ListenAndServe(":8080", r)
}
```

## Tenant Resolution Strategies

### 1. Header Resolver (Default)

Extract tenant from HTTP header:

```go
r.Use(tenantchi.Middleware(&tenantchi.Config{
    Resolver: tenantchi.NewHeaderResolver("X-Tenant-ID"),
}))

// Request: curl -H "X-Tenant-ID: tenant-123" http://localhost:8080/api/users
```

### 2. Subdomain Resolver

Extract tenant from subdomain:

```go
r.Use(tenantchi.Middleware(&tenantchi.Config{
    Resolver: tenantchi.NewSubdomainResolver("example.com"),
}))

// Request: curl http://tenant-123.example.com/api/users
```

### 3. URL Parameter Resolver

Extract tenant from Chi URL parameter:

```go
r.Route("/tenants/{tenantID}", func(r chi.Router) {
    r.Use(tenantchi.Middleware(&tenantchi.Config{
        Resolver: tenantchi.NewURLParamResolver("tenantID"),
    }))
    
    r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
        tenantID := tenantchi.URLParam(r, "tenantID")
        // ...
    })
})

// Request: curl http://localhost:8080/tenants/tenant-123/users
```

### 4. Path Resolver

Extract tenant from URL path segment:

```go
r.Use(tenantchi.Middleware(&tenantchi.Config{
    Resolver: tenantchi.NewPathResolver("/api/", 0),
}))

// Request: curl http://localhost:8080/api/tenant-123/users
// Extracts "tenant-123" from segment 0 after "/api/"
```

### 5. Query Parameter Resolver

Extract tenant from query parameter:

```go
r.Use(tenantchi.Middleware(&tenantchi.Config{
    Resolver: tenantchi.NewQueryParamResolver("tenant"),
}))

// Request: curl http://localhost:8080/api/users?tenant=tenant-123
```

### 6. Chain Resolver (Multiple Strategies)

Try multiple resolvers in order:

```go
r.Use(tenantchi.Middleware(&tenantchi.Config{
    Resolver: tenantchi.NewChainResolver(
        tenantchi.NewHeaderResolver("X-Tenant-ID"),
        tenantchi.NewQueryParamResolver("tenant"),
        tenantchi.NewSubdomainResolver("example.com"),
    ),
}))

// Tries header first, then query param, then subdomain
```

### 7. Static Resolver (Testing)

Always return the same tenant ID:

```go
r.Use(tenantchi.Middleware(&tenantchi.Config{
    Resolver: tenantchi.NewStaticResolver("test-tenant"),
}))

// Useful for testing or single-tenant deployments
```

## Configuration

### Skip Paths

Exclude certain paths from tenant resolution:

```go
r.Use(tenantchi.Middleware(&tenantchi.Config{
    Resolver: tenantchi.NewHeaderResolver("X-Tenant-ID"),
    SkipPaths: []string{"/health", "/metrics", "/public"},
}))

// /health and /metrics don't require tenant header
```

### Custom Error Handler

Handle tenant resolution errors:

```go
r.Use(tenantchi.Middleware(&tenantchi.Config{
    Resolver: tenantchi.NewHeaderResolver("X-Tenant-ID"),
    ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
        w.WriteHeader(http.StatusUnauthorized)
        json.NewEncoder(w).Encode(map[string]string{
            "error": "Tenant required",
            "details": err.Error(),
        })
    },
}))
```

## Extracting Tenant Information

### From Request

```go
r.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
    // Method 1: Get tenant ID only
    tenantID, err := tenantchi.GetTenantIDFromRequest(r)
    
    // Method 2: Get full context
    tenantCtx, err := tenantchi.GetTenantContext(r.Context())
    tenantID = tenantCtx.TenantID().Value()
    userID = tenantCtx.UserID()
    requestID = tenantCtx.RequestID()
})
```

### From Context

```go
func someFunction(ctx context.Context) {
    tenantID, err := tenantchi.GetTenantID(ctx)
    // ...
}
```

## Advanced Usage

### Per-Route Tenant Resolution

Different strategies for different routes:

```go
r := chi.NewRouter()

// Admin routes use header
r.Route("/admin", func(r chi.Router) {
    r.Use(tenantchi.Middleware(&tenantchi.Config{
        Resolver: tenantchi.NewHeaderResolver("X-Admin-Tenant"),
    }))
    r.Get("/stats", adminStatsHandler)
})

// Public API uses URL parameter
r.Route("/api/v1/tenants/{tenantID}", func(r chi.Router) {
    r.Use(tenantchi.Middleware(&tenantchi.Config{
        Resolver: tenantchi.NewURLParamResolver("tenantID"),
    }))
    r.Get("/users", usersHandler)
})
```

### Custom Resolver

Implement your own resolution logic:

```go
type CustomResolver struct{}

func (c *CustomResolver) Resolve(r *http.Request) (string, error) {
    // Your custom logic here
    // e.g., extract from JWT, database lookup, etc.
    return "custom-tenant", nil
}

r.Use(tenantchi.Middleware(&tenantchi.Config{
    Resolver: &CustomResolver{},
}))
```

## Integration with Database Adapters

Use with GORM or sqlx adapters:

```go
import (
    tenantchi "github.com/abhipray-cpu/tenantkit/adapters/http-chi"
    tenantgorm "github.com/abhipray-cpu/tenantkit/adapters/gorm"
)

r.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
    // Tenant context automatically in request context
    ctx := r.Context()
    
    // Database queries automatically scoped to tenant
    var users []User
    db.WithContext(ctx).Find(&users)
    // SELECT * FROM users WHERE tenant_id = ?
    
    json.NewEncoder(w).Encode(users)
})
```

## Testing

Create tenant context for tests:

```go
func TestHandler(t *testing.T) {
    ctx, _ := tenantchi.WithTenantID(context.Background(), "test-tenant")
    req := httptest.NewRequest("GET", "/api/users", nil)
    req = req.WithContext(ctx)
    
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)
    
    // Assert response
}
```

## Performance

- Middleware overhead: < 0.5ms per request
- Resolver execution: < 0.1ms
- Context propagation: < 0.01ms

## Best Practices

1. **Use Chain Resolver for Flexibility**
   ```go
   NewChainResolver(
       NewHeaderResolver("X-Tenant-ID"),
       NewQueryParamResolver("tenant"),
   )
   ```

2. **Skip Non-Tenant Routes**
   ```go
   SkipPaths: []string{"/health", "/metrics", "/docs"}
   ```

3. **Custom Error Responses**
   ```go
   ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
       // Log error, return JSON, etc.
   }
   ```

4. **Extract Tenant Early**
   ```go
   tenantID, _ := tenantchi.GetTenantIDFromRequest(r)
   // Use tenantID throughout handler
   ```

## Complete Example

```go
package main

import (
    "encoding/json"
    "net/http"
    
    tenantchi "github.com/abhipray-cpu/tenantkit/adapters/http-chi"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

func main() {
    r := chi.NewRouter()
    
    // Standard middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    
    // Tenant middleware with multiple strategies
    r.Use(tenantchi.Middleware(&tenantchi.Config{
        Resolver: tenantchi.NewChainResolver(
            tenantchi.NewHeaderResolver("X-Tenant-ID"),
            tenantchi.NewQueryParamResolver("tenant"),
        ),
        SkipPaths: []string{"/health", "/metrics"},
        ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]string{
                "error": "Tenant identification required",
            })
        },
    }))
    
    // Health check (skipped)
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("OK"))
    })
    
    // Tenant-scoped routes
    r.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
        tenantID, _ := tenantchi.GetTenantIDFromRequest(r)
        
        // Your business logic
        users := getUsersForTenant(tenantID)
        
        json.NewEncoder(w).Encode(users)
    })
    
    http.ListenAndServe(":8080", r)
}
```

## Troubleshooting

### Error: "Tenant resolution failed"

Check that your requests include the expected tenant identifier:
- Header: `X-Tenant-ID: tenant-123`
- Subdomain: `tenant-123.example.com`
- Query param: `?tenant=tenant-123`

### Error: "context does not contain tenant information"

Ensure the middleware is added before routes that need tenant context:
```go
r.Use(tenantchi.Middleware(...)) // Before routes
r.Get("/api/users", handler)      // After middleware
```

## License

MIT License
