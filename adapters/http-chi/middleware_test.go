package chi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abhipray-cpu/tenantkit/domain"
	chirouter "github.com/go-chi/chi/v5"
)

func TestMiddleware_HeaderResolver(t *testing.T) {
	cfg := &Config{
		Resolver: NewHeaderResolver("X-Tenant-ID"),
	}

	middleware := Middleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, err := GetTenantIDFromRequest(r)
		if err != nil {
			t.Fatalf("failed to get tenant ID: %v", err)
		}
		if tenantID != "test-tenant" {
			t.Errorf("expected tenant ID 'test-tenant', got '%s'", tenantID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestMiddleware_MissingTenant(t *testing.T) {
	cfg := &Config{
		Resolver: NewHeaderResolver("X-Tenant-ID"),
	}

	middleware := Middleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when tenant is missing")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	// No X-Tenant-ID header
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestMiddleware_CustomErrorHandler(t *testing.T) {
	var errorHandlerCalled bool

	cfg := &Config{
		Resolver: NewHeaderResolver("X-Tenant-ID"),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			errorHandlerCalled = true
			w.WriteHeader(http.StatusUnauthorized)
		},
	}

	middleware := Middleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !errorHandlerCalled {
		t.Error("custom error handler was not called")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestMiddleware_SkipPaths(t *testing.T) {
	cfg := &Config{
		Resolver:  NewHeaderResolver("X-Tenant-ID"),
		SkipPaths: []string{"/health", "/metrics"},
	}

	middleware := Middleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should be called even without tenant header for skip paths
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name string
		path string
		code int
	}{
		{"skip health", "/health", http.StatusOK},
		{"skip metrics", "/metrics/prometheus", http.StatusOK},
		{"require tenant", "/api/users", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			// No tenant header
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.code {
				t.Errorf("expected status %d, got %d", tt.code, rec.Code)
			}
		})
	}
}

func TestGetTenantContext(t *testing.T) {
	tenantCtx, err := domain.NewContext("t123", "u456", "r789")
	if err != nil {
		t.Fatalf("failed to create context: %v", err)
	}

	ctx := tenantCtx.ToGoContext(context.Background())

	retrieved, err := GetTenantContext(ctx)
	if err != nil {
		t.Fatalf("failed to get tenant context: %v", err)
	}

	if retrieved.TenantID().Value() != "t123" {
		t.Errorf("expected tenant ID 't123', got '%s'", retrieved.TenantID().Value())
	}
}

func TestGetTenantID(t *testing.T) {
	tenantCtx, err := domain.NewContext("t123", "u456", "r789")
	if err != nil {
		t.Fatalf("failed to create context: %v", err)
	}

	ctx := tenantCtx.ToGoContext(context.Background())

	tenantID, err := GetTenantID(ctx)
	if err != nil {
		t.Fatalf("failed to get tenant ID: %v", err)
	}

	if tenantID != "t123" {
		t.Errorf("expected tenant ID 't123', got '%s'", tenantID)
	}
}

func TestWithTenantID(t *testing.T) {
	ctx, err := WithTenantID(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("failed to create context with tenant ID: %v", err)
	}

	tenantID, err := GetTenantID(ctx)
	if err != nil {
		t.Fatalf("failed to get tenant ID: %v", err)
	}

	if tenantID != "test-tenant" {
		t.Errorf("expected tenant ID 'test-tenant', got '%s'", tenantID)
	}
}

func TestURLParam(t *testing.T) {
	r := chirouter.NewRouter()

	r.Route("/tenants/{tenantID}", func(r chirouter.Router) {
		r.Use(Middleware(&Config{
			Resolver: NewURLParamResolver("tenantID"),
		}))

		r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
			tenantID := URLParam(r, "tenantID")
			if tenantID != "test-tenant" {
				t.Errorf("expected tenant ID 'test-tenant', got '%s'", tenantID)
			}

			// Verify it matches context
			ctxTenantID, err := GetTenantIDFromRequest(r)
			if err != nil {
				t.Fatalf("failed to get tenant from context: %v", err)
			}

			if ctxTenantID != tenantID {
				t.Errorf("context tenant '%s' doesn't match URL param '%s'", ctxTenantID, tenantID)
			}

			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest("GET", "/tenants/test-tenant/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestMiddleware_Integration(t *testing.T) {
	r := chirouter.NewRouter()

	// Add tenant middleware
	r.Use(Middleware(&Config{
		Resolver: NewHeaderResolver("X-Tenant-ID"),
	}))

	// Add test routes
	r.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
		tenantID, err := GetTenantIDFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-Response-Tenant", tenantID)
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		tenantID   string
		expectCode int
	}{
		{"valid tenant", "tenant-123", http.StatusOK},
		{"different tenant", "tenant-456", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/users", nil)
			req.Header.Set("X-Tenant-ID", tt.tenantID)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectCode {
				t.Errorf("expected status %d, got %d", tt.expectCode, rec.Code)
			}

			if rec.Code == http.StatusOK {
				responseTenant := rec.Header().Get("X-Response-Tenant")
				if responseTenant != tt.tenantID {
					t.Errorf("expected response tenant '%s', got '%s'", tt.tenantID, responseTenant)
				}
			}
		})
	}
}
