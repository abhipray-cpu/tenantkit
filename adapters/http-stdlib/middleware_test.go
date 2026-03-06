package httpstd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMiddlewareInitialization tests middleware creation
func TestMiddlewareInitialization(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantError   bool
		description string
	}{
		{
			name: "valid_config",
			config: Config{
				Resolver: NewHeaderResolver("X-Tenant-ID"),
			},
			wantError:   false,
			description: "Should create middleware with valid config",
		},
		{
			name: "missing_resolver",
			config: Config{
				Resolver: nil,
			},
			wantError:   true,
			description: "Should fail when resolver is nil",
		},
		{
			name: "with_error_handler",
			config: Config{
				Resolver: NewHeaderResolver("X-Tenant-ID"),
				OnError: func(w http.ResponseWriter, r *http.Request, err error) {
					http.Error(w, "custom error", http.StatusForbidden)
				},
			},
			wantError:   false,
			description: "Should create middleware with custom error handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMiddleware(tt.config)

			if (err != nil) != tt.wantError {
				t.Errorf("%s: got error=%v, wantError=%v", tt.description, err, tt.wantError)
			}
		})
	}
}

// TestMiddlewareRequestEnrichment tests that middleware enriches requests with tenant context
func TestMiddlewareRequestEnrichment(t *testing.T) {
	resolver := NewHeaderResolver("X-Tenant-ID")
	middleware, err := NewMiddleware(Config{
		Resolver: resolver,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	tests := []struct {
		name        string
		setup       func(*http.Request)
		wantStatus  int
		wantSuccess bool
		description string
	}{
		{
			name: "valid_tenant_header",
			setup: func(r *http.Request) {
				r.Header.Set("X-Tenant-ID", "tenant1")
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
			description: "Should pass request when tenant header present",
		},
		{
			name: "missing_tenant_header",
			setup: func(r *http.Request) {
				// No header set
			},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
			description: "Should return 400 when tenant header missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			tt.setup(req)

			handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("%s: got status=%d, want=%d", tt.description, w.Code, tt.wantStatus)
			}
		})
	}
}

// TestMiddlewareContextAccess tests that tenant context is accessible in handler
func TestMiddlewareContextAccess(t *testing.T) {
	resolver := NewHeaderResolver("X-Tenant-ID")
	middleware, err := NewMiddleware(Config{
		Resolver: resolver,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant1")

	var capturedTenantID string
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, err := GetTenantID(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error: %v", err)))
			return
		}
		capturedTenantID = tenantID
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if capturedTenantID != "tenant1" {
		t.Errorf("Got tenant=%q, want=tenant1", capturedTenantID)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Got status=%d, want=%d", w.Code, http.StatusOK)
	}
}

// TestMiddlewareSkipPaths tests that configured paths skip tenant resolution
func TestMiddlewareSkipPaths(t *testing.T) {
	resolver := NewHeaderResolver("X-Tenant-ID")
	middleware, err := NewMiddleware(Config{
		Resolver:  resolver,
		SkipPaths: []string{"/health", "/readiness"},
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		description string
	}{
		{
			name:        "skip_health",
			path:        "/health",
			wantStatus:  http.StatusOK,
			description: "Should skip tenant resolution for /health",
		},
		{
			name:        "skip_readiness",
			path:        "/readiness",
			wantStatus:  http.StatusOK,
			description: "Should skip tenant resolution for /readiness",
		},
		{
			name:        "normal_path_without_header",
			path:        "/api/data",
			wantStatus:  http.StatusBadRequest,
			description: "Should require tenant for non-skip paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)

			handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("%s: got status=%d, want=%d", tt.description, w.Code, tt.wantStatus)
			}
		})
	}
}

// TestMiddlewareCustomErrorHandler tests custom error handler
func TestMiddlewareCustomErrorHandler(t *testing.T) {
	resolver := NewHeaderResolver("X-Tenant-ID")

	customError := false
	customHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		customError = true
		http.Error(w, "custom error", http.StatusForbidden)
	}

	middleware, err := NewMiddleware(Config{
		Resolver: resolver,
		OnError:  customHandler,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	// No tenant header

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !customError {
		t.Errorf("Custom error handler not called")
	}

	if w.Code != http.StatusForbidden {
		t.Errorf("Got status=%d, want=403", w.Code)
	}
}

// TestMiddlewareHandlerFunc tests the HandlerFunc wrapper
func TestMiddlewareHandlerFunc(t *testing.T) {
	resolver := NewHeaderResolver("X-Tenant-ID")
	middleware, err := NewMiddleware(Config{
		Resolver: resolver,
	})
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant1")

	wrappedHandler := middleware.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, err := GetTenantID(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(tenantID))
	})

	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Got status=%d, want=200", w.Code)
	}

	if w.Body.String() != "tenant1" {
		t.Errorf("Got body=%q, want=tenant1", w.Body.String())
	}
}
