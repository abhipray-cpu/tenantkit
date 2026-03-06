package httpstd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abhipray-cpu/tenantkit/adapters/limiter-memory"
)

// MockLimiter is a mock rate limiter for testing
type MockLimiter struct {
	allowCount   int
	allowUntil   int // Allow requests up to this count, then deny
	remainingVal int64
}

func (m *MockLimiter) Allow(ctx context.Context, key string) (bool, error) {
	m.allowCount++
	if m.allowCount > m.allowUntil {
		return false, nil
	}
	return true, nil
}

func (m *MockLimiter) AllowN(ctx context.Context, key string, n int64) (bool, error) {
	return m.Allow(ctx, key)
}

func (m *MockLimiter) Remaining(ctx context.Context, key string) (int64, error) {
	return m.remainingVal, nil
}

func (m *MockLimiter) Reset(ctx context.Context, key string) error {
	m.allowCount = 0
	return nil
}

func (m *MockLimiter) Health(ctx context.Context) error {
	return nil
}

func (m *MockLimiter) Stats() map[string]interface{} {
	return map[string]interface{}{
		"type": "mock",
	}
}

// TestNewRateLimitMiddleware tests creating a new rate limit middleware
func TestNewRateLimitMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		config  RateLimitConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: RateLimitConfig{
				Limiter: &MockLimiter{},
			},
			wantErr: false,
		},
		{
			name: "no limiter",
			config: RateLimitConfig{
				Limiter: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRateLimitMiddleware(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err=%v, want err=%v", err, tt.wantErr)
			}
		})
	}
}

// TestRateLimitAllowed tests that allowed requests pass through
func TestRateLimitAllowed(t *testing.T) {
	limiter := &MockLimiter{
		allowUntil:   1000, // Allow many requests
		remainingVal: 100,
	}
	middleware, _ := NewRateLimitMiddleware(RateLimitConfig{
		Limiter: limiter,
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handler(next)
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check for rate limit headers
	if w.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("missing X-RateLimit-Remaining header")
	}
}

// TestRateLimitDenied tests that denied requests are blocked with 429
func TestRateLimitDenied(t *testing.T) {
	limiter := &MockLimiter{
		allowUntil:   0,
		remainingVal: 0,
	}
	middleware, _ := NewRateLimitMiddleware(RateLimitConfig{
		Limiter: limiter,
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handler(next)
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}
}

// TestRateLimitSkipPaths tests that skip paths bypass rate limiting
func TestRateLimitSkipPaths(t *testing.T) {
	limiter := &MockLimiter{
		allowUntil:   0,
		remainingVal: 0,
	}
	middleware, _ := NewRateLimitMiddleware(RateLimitConfig{
		Limiter:   limiter,
		SkipPaths: []string{"/health", "/metrics"},
	})

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handler(next)

	// Test that /health is skipped
	r := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if !called {
		t.Error("next handler should be called for skipped paths")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for skipped path, got %d", w.Code)
	}
}

// TestRateLimitKeyExtractor tests custom key extraction
func TestRateLimitKeyExtractor(t *testing.T) {
	limiter := &MockLimiter{remainingVal: 100}
	var extractedKey string

	middleware, _ := NewRateLimitMiddleware(RateLimitConfig{
		Limiter: limiter,
		KeyExtractor: func(r *http.Request) string {
			extractedKey = "custom-key"
			return extractedKey
		},
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handler(next)
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if extractedKey != "custom-key" {
		t.Errorf("key extractor not called or returned wrong value: %s", extractedKey)
	}
}

// TestRateLimitWithRealLimiter tests rate limiting with a real in-memory limiter
func TestRateLimitWithRealLimiter(t *testing.T) {
	limiter, _ := limitermemory.NewTokenBucketLimiter(2, 5) // 2 RPS, burst of 5

	middleware, _ := NewRateLimitMiddleware(RateLimitConfig{
		Limiter: limiter,
		KeyExtractor: func(r *http.Request) string {
			return "test-key"
		},
	})

	successCount := 0
	deniedCount := 0

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		successCount++
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handler(next)

	// Make 6 requests - should allow 5, deny 1
	for i := 0; i < 6; i++ {
		r := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code == http.StatusTooManyRequests {
			deniedCount++
		}
	}

	if successCount != 5 {
		t.Errorf("expected 5 allowed requests, got %d", successCount)
	}

	if deniedCount != 1 {
		t.Errorf("expected 1 denied request, got %d", deniedCount)
	}
}
