package httpecho

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abhipray-cpu/tenantkit/domain"
	"github.com/labstack/echo/v4"
)

func TestMiddleware_HeaderResolver(t *testing.T) {
	e := echo.New()

	cfg := &Config{
		Resolver: &HeaderResolver{HeaderName: "X-Tenant-ID"},
	}

	e.Use(Middleware(cfg))

	e.GET("/test", func(c echo.Context) error {
		tenantID, _ := GetTenantID(c)
		return c.String(http.StatusOK, tenantID)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant123")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "tenant123" {
		t.Errorf("expected tenant123, got %s", rec.Body.String())
	}
}

func TestMiddleware_MissingTenant(t *testing.T) {
	e := echo.New()

	cfg := &Config{
		Resolver: &HeaderResolver{HeaderName: "X-Tenant-ID"},
	}

	e.Use(Middleware(cfg))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No tenant header
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestMiddleware_CustomErrorHandler(t *testing.T) {
	e := echo.New()

	customErrorCalled := false
	cfg := &Config{
		Resolver: &HeaderResolver{HeaderName: "X-Tenant-ID"},
		ErrorHandler: func(c echo.Context, err error) error {
			customErrorCalled = true
			return echo.NewHTTPError(http.StatusUnauthorized, "custom error")
		},
	}

	e.Use(Middleware(cfg))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if !customErrorCalled {
		t.Error("custom error handler was not called")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestMiddleware_SkipPaths(t *testing.T) {
	e := echo.New()

	cfg := &Config{
		Resolver:  &HeaderResolver{HeaderName: "X-Tenant-ID"},
		SkipPaths: []string{"/health", "/metrics"},
	}

	e.Use(Middleware(cfg))

	e.GET("/health", func(c echo.Context) error {
		tenantID, _ := GetTenantID(c)
		if tenantID != "" {
			t.Error("tenant ID should be empty for skipped paths")
		}
		return c.String(http.StatusOK, "healthy")
	})

	e.GET("/api", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	tests := []struct {
		name           string
		path           string
		expectStatus   int
		expectTenantID bool
	}{
		{
			name:           "skipped path - health",
			path:           "/health",
			expectStatus:   http.StatusOK,
			expectTenantID: false,
		},
		{
			name:         "non-skipped path without tenant",
			path:         "/api",
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			if rec.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, rec.Code)
			}
		})
	}
}

func TestGetTenantContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := e.NewContext(req, nil)

	// No tenant context - should return error
	_, err := GetTenantContext(ctx)
	if err == nil {
		t.Error("expected error when no tenant context")
	}

	// Set tenant context
	expectedCtx, err := domain.NewContext("tenant123", "user1", "req1")
	if err != nil {
		t.Fatalf("failed to create context: %v", err)
	}
	ctx.Set("tenant_context", expectedCtx)

	tenantCtx, err := GetTenantContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tenantCtx.TenantID().Value() != "tenant123" {
		t.Errorf("expected tenant123, got %s", tenantCtx.TenantID().Value())
	}
}

func TestGetTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := e.NewContext(req, nil)

	// No tenant context - should return error
	_, err := GetTenantID(ctx)
	if err == nil {
		t.Error("expected error when no tenant context")
	}

	// Set tenant context
	tenantCtx, err := domain.NewContext("tenant456", "user1", "req1")
	if err != nil {
		t.Fatalf("failed to create context: %v", err)
	}
	ctx.Set("tenant_context", tenantCtx)

	tenantID, err := GetTenantID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantID != "tenant456" {
		t.Errorf("expected tenant456, got %s", tenantID)
	}
}

func TestWithTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := e.NewContext(req, nil)

	// Set tenant ID
	ctx = WithTenantID(ctx, "tenant789")

	tenantID, err := GetTenantID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantID != "tenant789" {
		t.Errorf("expected tenant789, got %s", tenantID)
	}

	// Check request context as well
	tenantCtx, err := GetTenantContext(ctx)
	if err != nil {
		t.Fatalf("failed to get tenant context: %v", err)
	}

	if tenantCtx.TenantID().Value() != "tenant789" {
		t.Errorf("expected tenant789 in request context, got %s", tenantCtx.TenantID().Value())
	}
}

func TestMustGetTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := e.NewContext(req, nil)

	// Should panic when no tenant context
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when tenant context missing")
		}
	}()

	MustGetTenantID(ctx)
}

func TestMustGetTenantID_Success(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := e.NewContext(req, nil)

	tenantCtx, err := domain.NewContext("tenant999", "user1", "req1")
	if err != nil {
		t.Fatalf("failed to create context: %v", err)
	}
	ctx.Set("tenant_context", tenantCtx)

	tenantID := MustGetTenantID(ctx)
	if tenantID != "tenant999" {
		t.Errorf("expected tenant999, got %s", tenantID)
	}
}

func TestMiddleware_Integration(t *testing.T) {
	tests := []struct {
		name         string
		resolver     TenantResolver
		setupRequest func(*http.Request)
		expectStatus int
		expectTenant string
	}{
		{
			name:     "header resolver",
			resolver: &HeaderResolver{},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Tenant-ID", "header-tenant")
			},
			expectStatus: http.StatusOK,
			expectTenant: "header-tenant",
		},
		{
			name:     "query param resolver",
			resolver: &QueryParamResolver{ParamName: "tenant"},
			setupRequest: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("tenant", "query-tenant")
				r.URL.RawQuery = q.Encode()
			},
			expectStatus: http.StatusOK,
			expectTenant: "query-tenant",
		},
		{
			name:     "static resolver",
			resolver: &StaticResolver{TenantID: "static-tenant"},
			setupRequest: func(r *http.Request) {
				// No setup needed
			},
			expectStatus: http.StatusOK,
			expectTenant: "static-tenant",
		},
		{
			name: "chain resolver - header fallback to static",
			resolver: &ChainResolver{
				Resolvers: []TenantResolver{
					&HeaderResolver{},
					&StaticResolver{TenantID: "fallback-tenant"},
				},
			},
			setupRequest: func(r *http.Request) {
				// No header, should fallback to static
			},
			expectStatus: http.StatusOK,
			expectTenant: "fallback-tenant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			cfg := &Config{
				Resolver: tt.resolver,
			}

			e.Use(Middleware(cfg))

			e.GET("/test", func(c echo.Context) error {
				tenantID, _ := GetTenantID(c)
				return c.String(http.StatusOK, tenantID)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			if rec.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, rec.Code)
			}

			if tt.expectStatus == http.StatusOK {
				if rec.Body.String() != tt.expectTenant {
					t.Errorf("expected tenant %s, got %s", tt.expectTenant, rec.Body.String())
				}
			}
		})
	}
}

func TestMiddleware_DefaultConfig(t *testing.T) {
	e := echo.New()

	// Use middleware with nil config (should use defaults)
	e.Use(Middleware(nil))

	e.GET("/test", func(c echo.Context) error {
		tenantID, _ := GetTenantID(c)
		return c.String(http.StatusOK, tenantID)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "default-tenant")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "default-tenant" {
		t.Errorf("expected default-tenant, got %s", rec.Body.String())
	}
}

// Mock resolver for testing custom resolvers
type mockResolver struct {
	tenantID string
	err      error
}

func (m *mockResolver) Resolve(c echo.Context) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.tenantID, nil
}

func TestMiddleware_CustomResolver(t *testing.T) {
	tests := []struct {
		name         string
		resolver     TenantResolver
		expectStatus int
		expectTenant string
	}{
		{
			name:         "successful resolution",
			resolver:     &mockResolver{tenantID: "mock-tenant"},
			expectStatus: http.StatusOK,
			expectTenant: "mock-tenant",
		},
		{
			name:         "resolution error",
			resolver:     &mockResolver{err: errors.New("mock error")},
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			cfg := &Config{
				Resolver: tt.resolver,
			}

			e.Use(Middleware(cfg))

			e.GET("/test", func(c echo.Context) error {
				tenantID, _ := GetTenantID(c)
				return c.String(http.StatusOK, tenantID)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			if rec.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, rec.Code)
			}

			if tt.expectStatus == http.StatusOK {
				if rec.Body.String() != tt.expectTenant {
					t.Errorf("expected tenant %s, got %s", tt.expectTenant, rec.Body.String())
				}
			}
		})
	}
}
