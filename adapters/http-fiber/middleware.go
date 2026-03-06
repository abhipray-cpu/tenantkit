// Package httpfiber provides Fiber middleware for multi-tenant applications.
package httpfiber

import (
	"fmt"

	"github.com/abhipray-cpu/tenantkit/domain"
	"github.com/gofiber/fiber/v2"
)

type TenantResolver interface {
	Resolve(c *fiber.Ctx) (string, error)
}

type Config struct {
	Resolver     TenantResolver
	ErrorHandler func(c *fiber.Ctx, err error) error
	SkipPaths    []string
	ContextKey   string
}

func Middleware(cfg *Config) fiber.Handler {
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.Resolver == nil {
		cfg.Resolver = &HeaderResolver{HeaderName: "X-Tenant-ID"}
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = func(c *fiber.Ctx, err error) error {
			return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("tenant resolution failed: %v", err)})
		}
	}
	contextKey := cfg.ContextKey
	if contextKey == "" {
		contextKey = "tenant_context"
	}
	skipPaths := make(map[string]bool, len(cfg.SkipPaths))
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *fiber.Ctx) error {
		if skipPaths[c.Path()] {
			return c.Next()
		}

		tenantID, err := cfg.Resolver.Resolve(c)
		if err != nil {
			return cfg.ErrorHandler(c, err)
		}
		if tenantID == "" {
			return cfg.ErrorHandler(c, fmt.Errorf("tenant ID not found"))
		}

		requestID := fmt.Sprintf("%s-req", tenantID)
		tenantCtx, err := domain.NewContext(tenantID, "http-request", requestID)
		if err != nil {
			return cfg.ErrorHandler(c, fmt.Errorf("failed to create tenant context: %w", err))
		}

		c.Locals(contextKey, tenantCtx)
		ctx := tenantCtx.ToGoContext(c.Context())
		c.SetUserContext(ctx)

		return c.Next()
	}
}

func GetTenantContext(c *fiber.Ctx) (domain.Context, error) {
	if val := c.Locals("tenant_context"); val != nil {
		if tenantCtx, ok := val.(domain.Context); ok {
			return tenantCtx, nil
		}
	}
	return domain.FromGoContext(c.UserContext())
}

func GetTenantID(c *fiber.Ctx) (string, error) {
	tenantCtx, err := GetTenantContext(c)
	if err != nil {
		return "", err
	}
	return tenantCtx.TenantID().Value(), nil
}

func MustGetTenantID(c *fiber.Ctx) string {
	tenantID, err := GetTenantID(c)
	if err != nil {
		panic(fmt.Sprintf("tenant context not found: %v", err))
	}
	return tenantID
}

func WithTenantID(c *fiber.Ctx, tenantID string) {
	requestID := fmt.Sprintf("%s-test-req", tenantID)
	tenantCtx, err := domain.NewContext(tenantID, "http-request", requestID)
	if err != nil {
		panic(fmt.Sprintf("failed to create tenant context: %v", err))
	}
	c.Locals("tenant_context", tenantCtx)
	ctx := tenantCtx.ToGoContext(c.UserContext())
	c.SetUserContext(ctx)
}
