package httpecho

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestHeaderResolver(t *testing.T) {
	tests := []struct {
		name        string
		headerName  string
		headerValue string
		expectError bool
		expectID    string
	}{
		{
			name:        "default header name",
			headerName:  "",
			headerValue: "tenant123",
			expectError: false,
			expectID:    "tenant123",
		},
		{
			name:        "custom header name",
			headerName:  "X-Custom-Tenant",
			headerValue: "custom-tenant",
			expectError: false,
			expectID:    "custom-tenant",
		},
		{
			name:        "missing header",
			headerName:  "X-Tenant-ID",
			headerValue: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.headerValue != "" {
				headerName := tt.headerName
				if headerName == "" {
					headerName = "X-Tenant-ID"
				}
				req.Header.Set(headerName, tt.headerValue)
			}

			resolver := &HeaderResolver{HeaderName: tt.headerName}
			tenantID, err := resolver.Resolve(c)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tenantID != tt.expectID {
					t.Errorf("expected %s, got %s", tt.expectID, tenantID)
				}
			}
		})
	}
}

func TestSubdomainResolver(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		baseDomain  string
		expectError bool
		expectID    string
	}{
		{
			name:        "subdomain with base domain",
			host:        "tenant1.example.com",
			baseDomain:  "example.com",
			expectError: false,
			expectID:    "tenant1",
		},
		{
			name:        "subdomain with port",
			host:        "tenant2.example.com:8080",
			baseDomain:  "example.com",
			expectError: false,
			expectID:    "tenant2",
		},
		{
			name:        "no base domain - returns first part",
			host:        "localhost",
			baseDomain:  "",
			expectError: false,
			expectID:    "localhost",
		},
		{
			name:        "no base domain - multi-part host",
			host:        "tenant1.example.com",
			baseDomain:  "",
			expectError: false,
			expectID:    "tenant1",
		},
		{
			name:        "base domain without subdomain",
			host:        "example.com",
			baseDomain:  "example.com",
			expectError: true,
		},
		{
			name:        "wrong base domain",
			host:        "tenant1.other.com",
			baseDomain:  "example.com",
			expectError: true,
		},
		{
			name:        "multi-level subdomain",
			host:        "app.tenant1.example.com",
			baseDomain:  "example.com",
			expectError: false,
			expectID:    "app.tenant1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			resolver := &SubdomainResolver{BaseDomain: tt.baseDomain}
			tenantID, err := resolver.Resolve(c)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tenantID != tt.expectID {
					t.Errorf("expected %s, got %s", tt.expectID, tenantID)
				}
			}
		})
	}
}

func TestPathResolver(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		pathIndex   int
		expectError bool
		expectID    string
	}{
		{
			name:        "index 0",
			path:        "/tenant1/users",
			pathIndex:   0,
			expectError: false,
			expectID:    "tenant1",
		},
		{
			name:        "index 1",
			path:        "/tenants/tenant2/users",
			pathIndex:   1,
			expectError: false,
			expectID:    "tenant2",
		},
		{
			name:        "index out of range",
			path:        "/api/users",
			pathIndex:   5,
			expectError: true,
		},
		{
			name:        "negative index",
			path:        "/api/users",
			pathIndex:   -1,
			expectError: true,
		},
		{
			name:        "trailing slash",
			path:        "/tenant3/api/",
			pathIndex:   0,
			expectError: false,
			expectID:    "tenant3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			resolver := &PathResolver{PathIndex: tt.pathIndex}
			tenantID, err := resolver.Resolve(c)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tenantID != tt.expectID {
					t.Errorf("expected %s, got %s", tt.expectID, tenantID)
				}
			}
		})
	}
}

func TestParamResolver(t *testing.T) {
	tests := []struct {
		name        string
		paramName   string
		paramValue  string
		expectError bool
		expectID    string
	}{
		{
			name:        "default param name",
			paramName:   "",
			paramValue:  "param-tenant",
			expectError: false,
			expectID:    "param-tenant",
		},
		{
			name:        "custom param name",
			paramName:   "tid",
			paramValue:  "custom-tenant",
			expectError: false,
			expectID:    "custom-tenant",
		},
		{
			name:        "missing param",
			paramName:   "tenantID",
			paramValue:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set param value
			paramName := tt.paramName
			if paramName == "" {
				paramName = "tenantID"
			}
			if tt.paramValue != "" {
				c.SetParamNames(paramName)
				c.SetParamValues(tt.paramValue)
			}

			resolver := &ParamResolver{ParamName: tt.paramName}
			tenantID, err := resolver.Resolve(c)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tenantID != tt.expectID {
					t.Errorf("expected %s, got %s", tt.expectID, tenantID)
				}
			}
		})
	}
}

func TestQueryParamResolver(t *testing.T) {
	tests := []struct {
		name        string
		paramName   string
		paramValue  string
		expectError bool
		expectID    string
	}{
		{
			name:        "default param name",
			paramName:   "",
			paramValue:  "query-tenant",
			expectError: false,
			expectID:    "query-tenant",
		},
		{
			name:        "custom param name",
			paramName:   "tid",
			paramValue:  "custom-tenant",
			expectError: false,
			expectID:    "custom-tenant",
		},
		{
			name:        "missing param",
			paramName:   "tenant",
			paramValue:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()

			url := "/"
			if tt.paramValue != "" {
				paramName := tt.paramName
				if paramName == "" {
					paramName = "tenant"
				}
				url += "?" + paramName + "=" + tt.paramValue
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			resolver := &QueryParamResolver{ParamName: tt.paramName}
			tenantID, err := resolver.Resolve(c)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tenantID != tt.expectID {
					t.Errorf("expected %s, got %s", tt.expectID, tenantID)
				}
			}
		})
	}
}

func TestChainResolver(t *testing.T) {
	tests := []struct {
		name        string
		resolvers   []TenantResolver
		setupCtx    func(echo.Context)
		expectError bool
		expectID    string
	}{
		{
			name: "first resolver succeeds",
			resolvers: []TenantResolver{
				&HeaderResolver{},
				&StaticResolver{TenantID: "fallback"},
			},
			setupCtx: func(c echo.Context) {
				c.Request().Header.Set("X-Tenant-ID", "header-tenant")
			},
			expectError: false,
			expectID:    "header-tenant",
		},
		{
			name: "fallback to second resolver",
			resolvers: []TenantResolver{
				&HeaderResolver{},
				&StaticResolver{TenantID: "fallback-tenant"},
			},
			setupCtx: func(c echo.Context) {
				// No header, should fallback
			},
			expectError: false,
			expectID:    "fallback-tenant",
		},
		{
			name: "all resolvers fail",
			resolvers: []TenantResolver{
				&HeaderResolver{},
				&QueryParamResolver{},
			},
			setupCtx: func(c echo.Context) {
				// No header, no query param
			},
			expectError: true,
		},
		{
			name:        "no resolvers",
			resolvers:   []TenantResolver{},
			setupCtx:    func(c echo.Context) {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupCtx(c)

			resolver := &ChainResolver{Resolvers: tt.resolvers}
			tenantID, err := resolver.Resolve(c)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tenantID != tt.expectID {
					t.Errorf("expected %s, got %s", tt.expectID, tenantID)
				}
			}
		})
	}
}

func TestStaticResolver(t *testing.T) {
	tests := []struct {
		name        string
		tenantID    string
		expectError bool
	}{
		{
			name:        "valid static tenant",
			tenantID:    "static-tenant",
			expectError: false,
		},
		{
			name:        "empty tenant ID",
			tenantID:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			resolver := &StaticResolver{TenantID: tt.tenantID}
			tenantID, err := resolver.Resolve(c)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tenantID != tt.tenantID {
					t.Errorf("expected %s, got %s", tt.tenantID, tenantID)
				}
			}
		})
	}
}
