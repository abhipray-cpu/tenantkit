package httpgin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMiddleware_HeaderResolver(t *testing.T) {
	r := gin.New()
	
	r.Use(Middleware(&Config{
		Resolver: &HeaderResolver{HeaderName: "X-Tenant-ID"},
	}))
	
	r.GET("/test", func(c *gin.Context) {
		tenantID, err := GetTenantID(c)
		if err != nil {
			c.String(500, err.Error())
			return
		}
		c.String(200, tenantID)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant123")
	w := httptest.NewRecorder()
	
	r.ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "tenant123" {
		t.Errorf("expected tenant123, got %s", w.Body.String())
	}
}

func TestMiddleware_MissingTenant(t *testing.T) {
	r := gin.New()
	
	r.Use(Middleware(&Config{
		Resolver: &HeaderResolver{},
	}))
	
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	r.ServeHTTP(w, req)
	
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestMiddleware_CustomErrorHandler(t *testing.T) {
	r := gin.New()
	
	called := false
	r.Use(Middleware(&Config{
		Resolver: &HeaderResolver{},
		ErrorHandler: func(c *gin.Context, err error) {
			called = true
			c.AbortWithStatusJSON(401, gin.H{"error": "custom"})
		},
	}))
	
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	r.ServeHTTP(w, req)
	
	if !called {
		t.Error("custom error handler not called")
	}
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_SkipPaths(t *testing.T) {
	r := gin.New()
	
	r.Use(Middleware(&Config{
		Resolver:  &HeaderResolver{},
		SkipPaths: []string{"/health", "/metrics"},
	}))
	
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "healthy")
	})
	
	r.GET("/api", func(c *gin.Context) {
		c.String(200, "ok")
	})
	
	tests := []struct {
		name   string
		path   string
		expect int
	}{
		{"skip health", "/health", 200},
		{"require tenant", "/api", 400},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			if w.Code != tt.expect {
				t.Errorf("expected %d, got %d", tt.expect, w.Code)
			}
		})
	}
}

func TestGetTenantID(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	
	_, err := GetTenantID(c)
	if err == nil {
		t.Error("expected error when no tenant")
	}
}

func TestWithTenantID(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	
	WithTenantID(c, "tenant789")
	
	tenantID, err := GetTenantID(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantID != "tenant789" {
		t.Errorf("expected tenant789, got %s", tenantID)
	}
}

func TestMustGetTenantID_Panic(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	
	MustGetTenantID(c)
}

func TestMustGetTenantID_Success(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	
	WithTenantID(c, "tenant999")
	
	tenantID := MustGetTenantID(c)
	if tenantID != "tenant999" {
		t.Errorf("expected tenant999, got %s", tenantID)
	}
}

func TestMiddleware_Integration(t *testing.T) {
	tests := []struct {
		name     string
		resolver TenantResolver
		setup    func(*http.Request)
		expect   int
		tenant   string
	}{
		{
			name:     "header",
			resolver: &HeaderResolver{},
			setup: func(r *http.Request) {
				r.Header.Set("X-Tenant-ID", "h-tenant")
			},
			expect: 200,
			tenant: "h-tenant",
		},
		{
			name:     "query",
			resolver: &QueryParamResolver{},
			setup: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("tenant", "q-tenant")
				r.URL.RawQuery = q.Encode()
			},
			expect: 200,
			tenant: "q-tenant",
		},
		{
			name:     "static",
			resolver: &StaticResolver{TenantID: "static"},
			setup:    func(r *http.Request) {},
			expect:   200,
			tenant:   "static",
		},
		{
			name: "chain",
			resolver: &ChainResolver{
				Resolvers: []TenantResolver{
					&HeaderResolver{},
					&StaticResolver{TenantID: "fallback"},
				},
			},
			setup:  func(r *http.Request) {},
			expect: 200,
			tenant: "fallback",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(Middleware(&Config{Resolver: tt.resolver}))
			r.GET("/test", func(c *gin.Context) {
				id, _ := GetTenantID(c)
				c.String(200, id)
			})
			
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setup(req)
			w := httptest.NewRecorder()
			
			r.ServeHTTP(w, req)
			
			if w.Code != tt.expect {
				t.Errorf("expected %d, got %d", tt.expect, w.Code)
			}
			if tt.expect == 200 && w.Body.String() != tt.tenant {
				t.Errorf("expected %s, got %s", tt.tenant, w.Body.String())
			}
		})
	}
}

type mockResolver struct {
	id  string
	err error
}

func (m *mockResolver) Resolve(c *gin.Context) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.id, nil
}

func TestMiddleware_CustomResolver(t *testing.T) {
	tests := []struct {
		name     string
		resolver TenantResolver
		expect   int
	}{
		{"success", &mockResolver{id: "mock"}, 200},
		{"error", &mockResolver{err: errors.New("fail")}, 400},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(Middleware(&Config{Resolver: tt.resolver}))
			r.GET("/test", func(c *gin.Context) {
				c.String(200, "ok")
			})
			
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			if w.Code != tt.expect {
				t.Errorf("expected %d, got %d", tt.expect, w.Code)
			}
		})
	}
}
