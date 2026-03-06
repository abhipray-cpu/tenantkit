package httpstd

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// TestDefaultRateLimitOptions verifies default configuration values
func TestDefaultRateLimitOptions(t *testing.T) {
	opts := DefaultRateLimitOptions()

	if opts == nil {
		t.Fatal("DefaultRateLimitOptions returned nil")
	}

	if opts.LimitPerWindow != 100 {
		t.Errorf("Expected LimitPerWindow=100, got %d", opts.LimitPerWindow)
	}

	if opts.ResetWindow != 1*time.Minute {
		t.Errorf("Expected ResetWindow=1m, got %v", opts.ResetWindow)
	}
}

// TestStrictRateLimitOptions verifies strict configuration
func TestStrictRateLimitOptions(t *testing.T) {
	opts := StrictRateLimitOptions()

	if opts == nil {
		t.Fatal("StrictRateLimitOptions returned nil")
	}

	if opts.LimitPerWindow != 30 {
		t.Errorf("Expected LimitPerWindow=30, got %d", opts.LimitPerWindow)
	}

	if opts.LimitPerWindow >= DefaultRateLimitOptions().LimitPerWindow {
		t.Error("Strict options should be more restrictive than default")
	}
}

// TestGenerousRateLimitOptions verifies generous configuration
func TestGenerousRateLimitOptions(t *testing.T) {
	opts := GenerousRateLimitOptions()

	if opts == nil {
		t.Fatal("GenerousRateLimitOptions returned nil")
	}

	if opts.LimitPerWindow != 500 {
		t.Errorf("Expected LimitPerWindow=500, got %d", opts.LimitPerWindow)
	}

	if opts.LimitPerWindow <= DefaultRateLimitOptions().LimitPerWindow {
		t.Error("Generous options should be more permissive than default")
	}
}

// TestPerSecondRateLimitOptions verifies per-second configuration
func TestPerSecondRateLimitOptions(t *testing.T) {
	opts := PerSecondRateLimitOptions()

	if opts == nil {
		t.Fatal("PerSecondRateLimitOptions returned nil")
	}

	if opts.ResetWindow != 1*time.Second {
		t.Errorf("Expected ResetWindow=1s, got %v", opts.ResetWindow)
	}

	if opts.ResetWindow >= DefaultRateLimitOptions().ResetWindow {
		t.Error("Per-second window should be shorter than per-minute")
	}
}

// TestCustomRateLimitOptions verifies custom configuration creation
func TestCustomRateLimitOptions(t *testing.T) {
	limit := int64(50)
	window := 30 * time.Second

	opts := CustomRateLimitOptions(limit, window)

	if opts == nil {
		t.Fatal("CustomRateLimitOptions returned nil")
	}

	if opts.LimitPerWindow != limit {
		t.Errorf("Expected LimitPerWindow=%d, got %d", limit, opts.LimitPerWindow)
	}

	if opts.ResetWindow != window {
		t.Errorf("Expected ResetWindow=%v, got %v", window, opts.ResetWindow)
	}
}

// TestCustomRateLimitOptions_InvalidLimit verifies invalid limit is corrected
func TestCustomRateLimitOptions_InvalidLimit(t *testing.T) {
	opts := CustomRateLimitOptions(-1, 1*time.Minute)

	if opts.LimitPerWindow <= 0 {
		t.Error("Invalid limit should be corrected to default")
	}
}

// TestCustomRateLimitOptions_InvalidWindow verifies invalid window is corrected
func TestCustomRateLimitOptions_InvalidWindow(t *testing.T) {
	opts := CustomRateLimitOptions(100, -1*time.Second)

	if opts.ResetWindow <= 0 {
		t.Error("Invalid window should be corrected to default")
	}
}

// TestRateLimitMiddleware_UsesConfigOptions verifies middleware uses config options
func TestRateLimitMiddleware_UsesConfigOptions(t *testing.T) {
	mockLimiter := &MockLimiter{allowUntil: 100, remainingVal: 99}
	opts := CustomRateLimitOptions(50, 30*time.Second)

	config := RateLimitConfig{
		Limiter: mockLimiter,
		Options: opts,
	}

	mw, err := NewRateLimitMiddleware(config)
	if err != nil {
		t.Fatalf("NewRateLimitMiddleware failed: %v", err)
	}

	// Create test handler
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify X-RateLimit-Limit header uses configured limit
	limitHeader := w.Header().Get("X-RateLimit-Limit")
	if limitHeader != strconv.FormatInt(opts.LimitPerWindow, 10) {
		t.Errorf("Expected X-RateLimit-Limit=%d, got %s", opts.LimitPerWindow, limitHeader)
	}
}

// TestRateLimitMiddleware_DefaultOptions verifies middleware uses default options when none provided
func TestRateLimitMiddleware_DefaultOptions(t *testing.T) {
	mockLimiter := &MockLimiter{allowUntil: 100, remainingVal: 99}

	config := RateLimitConfig{
		Limiter: mockLimiter,
		// Options not set - should default
	}

	mw, err := NewRateLimitMiddleware(config)
	if err != nil {
		t.Fatalf("NewRateLimitMiddleware failed: %v", err)
	}

	if mw.config.Options == nil {
		t.Fatal("Options should be initialized to defaults")
	}

	if mw.config.Options.LimitPerWindow != DefaultRateLimitOptions().LimitPerWindow {
		t.Error("Should use default limit when not specified")
	}
}

// TestRateLimitMiddleware_ResetTimeUseConfigWindow verifies reset time uses config window
func TestRateLimitMiddleware_ResetTimeUseConfigWindow(t *testing.T) {
	mockLimiter := &MockLimiter{allowUntil: 0} // Trigger rate limit exceeded
	customWindow := 45 * time.Second
	opts := CustomRateLimitOptions(100, customWindow)

	config := RateLimitConfig{
		Limiter: mockLimiter,
		Options: opts,
		OnLimitExceeded: func(w http.ResponseWriter, r *http.Request, remaining int64, resetTime time.Time) {
			// Verify reset time is approximately now + custom window
			delta := time.Until(resetTime)
			if delta < customWindow-100*time.Millisecond || delta > customWindow+100*time.Millisecond {
				t.Errorf("Reset time should be now + %v, got delta=%v", customWindow, delta)
			}
		},
	}

	mw, err := NewRateLimitMiddleware(config)
	if err != nil {
		t.Fatalf("NewRateLimitMiddleware failed: %v", err)
	}

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

// TestRateLimitMiddleware_MultipleOptions verifies different preset options
func TestRateLimitMiddleware_MultipleOptions(t *testing.T) {
	testCases := []struct {
		name string
		opts *RateLimitOptions
	}{
		{"Default", DefaultRateLimitOptions()},
		{"Strict", StrictRateLimitOptions()},
		{"Generous", GenerousRateLimitOptions()},
		{"PerSecond", PerSecondRateLimitOptions()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockLimiter := &MockLimiter{allowUntil: 100, remainingVal: 99}
			config := RateLimitConfig{
				Limiter: mockLimiter,
				Options: tc.opts,
			}

			mw, err := NewRateLimitMiddleware(config)
			if err != nil {
				t.Fatalf("NewRateLimitMiddleware failed: %v", err)
			}

			if mw.config.Options.LimitPerWindow != tc.opts.LimitPerWindow {
				t.Errorf("Option %s: limit mismatch", tc.name)
			}

			if mw.config.Options.ResetWindow != tc.opts.ResetWindow {
				t.Errorf("Option %s: window mismatch", tc.name)
			}
		})
	}
}

// TestRateLimitOptions_Boundaries verifies option boundaries
func TestRateLimitOptions_Boundaries(t *testing.T) {
	tests := []struct {
		name         string
		limit        int64
		window       time.Duration
		expectLimit  int64
		expectWindow time.Duration
	}{
		{"Very small limit", 1, 1 * time.Second, 1, 1 * time.Second},
		{"Very large limit", 10000, 1 * time.Hour, 10000, 1 * time.Hour},
		{"Sub-second window", 1, 100 * time.Millisecond, 1, 100 * time.Millisecond},
		{"Multi-hour window", 1000, 24 * time.Hour, 1000, 24 * time.Hour},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := CustomRateLimitOptions(test.limit, test.window)
			if opts.LimitPerWindow != test.expectLimit {
				t.Errorf("Expected limit %d, got %d", test.expectLimit, opts.LimitPerWindow)
			}
			if opts.ResetWindow != test.expectWindow {
				t.Errorf("Expected window %v, got %v", test.expectWindow, opts.ResetWindow)
			}
		})
	}
}

// TestRateLimitMiddleware_ConfigurationIsolation verifies configurations don't interfere
func TestRateLimitMiddleware_ConfigurationIsolation(t *testing.T) {
	limiter1 := &MockLimiter{allowUntil: 100, remainingVal: 99}
	limiter2 := &MockLimiter{allowUntil: 100, remainingVal: 99}

	opts1 := CustomRateLimitOptions(30, 30*time.Second)
	opts2 := CustomRateLimitOptions(500, 1*time.Minute)

	config1 := RateLimitConfig{
		Limiter: limiter1,
		Options: opts1,
	}

	config2 := RateLimitConfig{
		Limiter: limiter2,
		Options: opts2,
	}

	mw1, _ := NewRateLimitMiddleware(config1)
	mw2, _ := NewRateLimitMiddleware(config2)

	// Verify each middleware has its own configuration
	if mw1.config.Options.LimitPerWindow == mw2.config.Options.LimitPerWindow {
		t.Error("Middlewares should have independent configurations")
	}

	if mw1.config.Options.ResetWindow == mw2.config.Options.ResetWindow {
		// They both use different windows
		if opts1.ResetWindow == opts2.ResetWindow {
			t.Error("Test setup issue: should have different windows")
		}
	}
}
