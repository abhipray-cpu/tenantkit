package httpfiber

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

var (
	ErrTenantNotFound = errors.New("tenant ID not found")
	ErrInvalidTenant  = errors.New("invalid tenant ID")
)

type HeaderResolver struct {
	HeaderName string
}

func (h *HeaderResolver) Resolve(c *fiber.Ctx) (string, error) {
	headerName := h.HeaderName
	if headerName == "" {
		headerName = "X-Tenant-ID"
	}
	tenantID := c.Get(headerName)
	if tenantID == "" {
		return "", fmt.Errorf("%w: header %s is empty", ErrTenantNotFound, headerName)
	}
	return tenantID, nil
}

type SubdomainResolver struct {
	BaseDomain string
}

func (s *SubdomainResolver) Resolve(c *fiber.Ctx) (string, error) {
	host := c.Hostname()
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	if s.BaseDomain != "" {
		suffix := "." + s.BaseDomain
		if !strings.HasSuffix(host, suffix) {
			return "", fmt.Errorf("%w: host %s does not end with base domain %s", ErrTenantNotFound, host, s.BaseDomain)
		}
		subdomain := strings.TrimSuffix(host, suffix)
		if subdomain == "" {
			return "", fmt.Errorf("%w: no subdomain found in host %s", ErrTenantNotFound, host)
		}
		return subdomain, nil
	}
	parts := strings.Split(host, ".")
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("%w: cannot extract subdomain from host %s", ErrTenantNotFound, host)
	}
	return parts[0], nil
}

type PathResolver struct {
	PathIndex int
}

func (p *PathResolver) Resolve(c *fiber.Ctx) (string, error) {
	path := c.Path()
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if p.PathIndex < 0 || p.PathIndex >= len(parts) {
		return "", fmt.Errorf("%w: path index %d out of range for path %s", ErrTenantNotFound, p.PathIndex, path)
	}
	tenantID := parts[p.PathIndex]
	if tenantID == "" {
		return "", fmt.Errorf("%w: path segment at index %d is empty", ErrTenantNotFound, p.PathIndex)
	}
	return tenantID, nil
}

type ParamResolver struct {
	ParamName string
}

func (p *ParamResolver) Resolve(c *fiber.Ctx) (string, error) {
	paramName := p.ParamName
	if paramName == "" {
		paramName = "tenantID"
	}
	tenantID := c.Params(paramName)
	if tenantID == "" {
		return "", fmt.Errorf("%w: parameter %s is empty", ErrTenantNotFound, paramName)
	}
	return tenantID, nil
}

type QueryParamResolver struct {
	ParamName string
}

func (q *QueryParamResolver) Resolve(c *fiber.Ctx) (string, error) {
	paramName := q.ParamName
	if paramName == "" {
		paramName = "tenant"
	}
	tenantID := c.Query(paramName)
	if tenantID == "" {
		return "", fmt.Errorf("%w: query parameter %s is empty", ErrTenantNotFound, paramName)
	}
	return tenantID, nil
}

type ChainResolver struct {
	Resolvers []TenantResolver
}

func (ch *ChainResolver) Resolve(c *fiber.Ctx) (string, error) {
	if len(ch.Resolvers) == 0 {
		return "", fmt.Errorf("%w: no resolvers configured", ErrTenantNotFound)
	}
	var lastErr error
	for _, resolver := range ch.Resolvers {
		tenantID, err := resolver.Resolve(c)
		if err == nil {
			return tenantID, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("%w: all resolvers failed, last error: %v", ErrTenantNotFound, lastErr)
}

type StaticResolver struct {
	TenantID string
}

func (s *StaticResolver) Resolve(c *fiber.Ctx) (string, error) {
	if s.TenantID == "" {
		return "", fmt.Errorf("%w: static tenant ID is empty", ErrTenantNotFound)
	}
	return s.TenantID, nil
}
