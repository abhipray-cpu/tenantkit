package httpfiber

import (
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestMiddleware(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware(&Config{Resolver: &HeaderResolver{}}))
	app.Get("/test", func(c *fiber.Ctx) error {
		id, _ := GetTenantID(c)
		return c.SendString(id)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	
	if string(body) != "t1" {
		t.Errorf("expected t1, got %s", body)
	}
}

func TestSkipPaths(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware(&Config{
		Resolver:  &HeaderResolver{},
		SkipPaths: []string{"/health"},
	}))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	
	req := httptest.NewRequest("GET", "/health", nil)
	resp, _ := app.Test(req)
	
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestResolvers(t *testing.T) {
	tests := []struct {
		name     string
		resolver TenantResolver
		url      string
		header   map[string]string
		want     string
	}{
		{
			"header",
			&HeaderResolver{},
			"/test",
			map[string]string{"X-Tenant-ID": "h1"},
			"h1",
		},
		{
			"query",
			&QueryParamResolver{},
			"/test?tenant=q1",
			nil,
			"q1",
		},
		{
			"static",
			&StaticResolver{TenantID: "s1"},
			"/test",
			nil,
			"s1",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(Middleware(&Config{Resolver: tt.resolver}))
			app.Get("/test", func(c *fiber.Ctx) error {
				id, err := GetTenantID(c)
				if err != nil {
					return c.SendStatus(500)
				}
				return c.SendString(id)
			})
			
			req := httptest.NewRequest("GET", tt.url, nil)
			for k, v := range tt.header {
				req.Header.Set(k, v)
			}
			resp, _ := app.Test(req)
			body, _ := io.ReadAll(resp.Body)
			
			if string(body) != tt.want {
				t.Errorf("expected %s, got %s", tt.want, body)
			}
		})
	}
}

func TestChainResolver(t *testing.T) {
	app := fiber.New()
	app.Use(Middleware(&Config{
		Resolver: &ChainResolver{
			Resolvers: []TenantResolver{
				&HeaderResolver{},
				&StaticResolver{TenantID: "fallback"},
			},
		},
	}))
	app.Get("/test", func(c *fiber.Ctx) error {
		id, _ := GetTenantID(c)
		return c.SendString(id)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	
	if string(body) != "fallback" {
		t.Errorf("expected fallback, got %s", body)
	}
}

func TestHelpers(t *testing.T) {
	app := fiber.New()
	
	t.Run("WithTenantID", func(t *testing.T) {
		app.Get("/with", func(c *fiber.Ctx) error {
			WithTenantID(c, "test")
			id, _ := GetTenantID(c)
			return c.SendString(id)
		})
		
		req := httptest.NewRequest("GET", "/with", nil)
		resp, _ := app.Test(req)
		body, _ := io.ReadAll(resp.Body)
		
		if string(body) != "test" {
			t.Errorf("expected test, got %s", body)
		}
	})
	
	t.Run("MustGetTenantID", func(t *testing.T) {
		app.Get("/must", func(c *fiber.Ctx) error {
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic")
				}
			}()
			MustGetTenantID(c)
			return nil
		})
		
		req := httptest.NewRequest("GET", "/must", nil)
		app.Test(req)
	})
}

type mockResolver struct {
	id  string
	err error
}

func (m *mockResolver) Resolve(c *fiber.Ctx) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.id, nil
}

func TestCustomResolver(t *testing.T) {
	tests := []struct {
		name   string
		r      TenantResolver
		expect int
	}{
		{"success", &mockResolver{id: "mock"}, 200},
		{"error", &mockResolver{err: errors.New("fail")}, 400},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(Middleware(&Config{Resolver: tt.r}))
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("ok")
			})
			
			req := httptest.NewRequest("GET", "/test", nil)
			resp, _ := app.Test(req)
			
			if resp.StatusCode != tt.expect {
				t.Errorf("expected %d, got %d", tt.expect, resp.StatusCode)
			}
		})
	}
}
