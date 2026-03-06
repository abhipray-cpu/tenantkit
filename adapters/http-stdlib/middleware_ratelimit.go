package httpstd

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/abhipray-cpu/tenantkit/tenantkit/ports"
)

// RateLimitConfig holds configuration for rate limiting middleware
type RateLimitConfig struct {
	// Limiter is the rate limiter instance to use
	Limiter ports.Limiter
	// KeyExtractor extracts the rate limit key from the request
	// If nil, defaults to using tenant ID as the key
	KeyExtractor func(r *http.Request) string
	// OnLimitExceeded is called when rate limit is exceeded
	// If not provided, defaults to responding with 429 Too Many Requests
	OnLimitExceeded RateLimitErrorHandler
	// SkipPaths is a list of URL paths that should skip rate limiting
	SkipPaths []string
	// Options holds rate limiting behavior options (limit and reset window)
	// If nil, defaults to DefaultRateLimitOptions()
	Options *RateLimitOptions
}

// RateLimitErrorHandler is a function that handles rate limit exceeded errors
type RateLimitErrorHandler func(w http.ResponseWriter, r *http.Request, remaining int64, resetTime time.Time)

// DefaultRateLimitErrorHandler responds with 429 Too Many Requests
func DefaultRateLimitErrorHandler(w http.ResponseWriter, r *http.Request, remaining int64, resetTime time.Time) {
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
	http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
}

// RateLimitMiddleware wraps an HTTP handler to enforce rate limiting
type RateLimitMiddleware struct {
	config RateLimitConfig
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(config RateLimitConfig) (*RateLimitMiddleware, error) {
	if config.Limiter == nil {
		return nil, fmt.Errorf("limiter is required in rate limit config")
	}

	if config.OnLimitExceeded == nil {
		config.OnLimitExceeded = DefaultRateLimitErrorHandler
	}

	if config.KeyExtractor == nil {
		config.KeyExtractor = func(r *http.Request) string {
			// Default: use tenant ID if available
			if tenantID, err := GetTenantID(r); err == nil && tenantID != "" {
				return tenantID
			}
			// Fallback to request IP
			return r.RemoteAddr
		}
	}

	if config.Options == nil {
		config.Options = DefaultRateLimitOptions()
	}

	return &RateLimitMiddleware{
		config: config,
	}, nil
}

// shouldSkip checks if the path should skip rate limiting
func (m *RateLimitMiddleware) shouldSkip(path string) bool {
	for _, skipPath := range m.config.SkipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// Handler returns an HTTP handler that wraps the provided handler with rate limiting
func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path should be skipped
		if m.shouldSkip(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract rate limit key
		key := m.config.KeyExtractor(r)

		// Check rate limit
		allowed, err := m.config.Limiter.Allow(r.Context(), key)
		if err != nil {
			// On error, log but allow request (fail open)
			fmt.Printf("rate limiter error: %v\n", err)
			next.ServeHTTP(w, r)
			return
		}

		if !allowed {
			// Get remaining and reset time for headers
			remaining, _ := m.config.Limiter.Remaining(r.Context(), key)
			resetTime := time.Now().Add(m.config.Options.ResetWindow)

			m.config.OnLimitExceeded(w, r, remaining, resetTime)
			return
		}

		// Add rate limit headers to response
		remaining, _ := m.config.Limiter.Remaining(r.Context(), key)
		w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(m.config.Options.LimitPerWindow, 10))

		next.ServeHTTP(w, r)
	})
}

// HTTPHandler is a convenience function for net/http compatibility
func (m *RateLimitMiddleware) HTTPHandler(next http.Handler) http.Handler {
	return m.Handler(next)
}
