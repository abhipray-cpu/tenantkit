package httpstd

import (
	"time"
)

// RequestTimeoutConfig holds configuration for HTTP request-level timeouts
type RequestTimeoutConfig struct {
	// ReadTimeout is the maximum duration before timing out a read from the client
	// Default: 30 seconds
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out a write to the client
	// Default: 30 seconds
	WriteTimeout time.Duration

	// IdleTimeout is the maximum duration an idle connection is kept alive
	// Default: 120 seconds
	IdleTimeout time.Duration

	// RequestContextTimeout is the timeout for the entire request context
	// Default: 60 seconds
	RequestContextTimeout time.Duration
}

// DefaultRequestTimeoutConfig returns sensible defaults for HTTP request handling
func DefaultRequestTimeoutConfig() *RequestTimeoutConfig {
	return &RequestTimeoutConfig{
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           120 * time.Second,
		RequestContextTimeout: 60 * time.Second,
	}
}

// FastRequestTimeoutConfig returns aggressive timeout settings
// Suitable for microservices where quick failure detection is important
func FastRequestTimeoutConfig() *RequestTimeoutConfig {
	return &RequestTimeoutConfig{
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           30 * time.Second,
		RequestContextTimeout: 15 * time.Second,
	}
}

// RelaxedRequestTimeoutConfig returns conservative timeout settings
// Suitable for file uploads, long-running operations, or high-latency networks
func RelaxedRequestTimeoutConfig() *RequestTimeoutConfig {
	return &RequestTimeoutConfig{
		ReadTimeout:           300 * time.Second,  // 5 minutes
		WriteTimeout:          300 * time.Second,  // 5 minutes
		IdleTimeout:           600 * time.Second,  // 10 minutes
		RequestContextTimeout: 600 * time.Second,  // 10 minutes
	}
}

// CustomRequestTimeoutConfig creates request timeout config with custom settings
func CustomRequestTimeoutConfig(readTimeout, writeTimeout, idleTimeout, contextTimeout time.Duration) *RequestTimeoutConfig {
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	if writeTimeout <= 0 {
		writeTimeout = 30 * time.Second
	}
	if idleTimeout <= 0 {
		idleTimeout = 120 * time.Second
	}
	if contextTimeout <= 0 {
		contextTimeout = 60 * time.Second
	}
	return &RequestTimeoutConfig{
		ReadTimeout:           readTimeout,
		WriteTimeout:          writeTimeout,
		IdleTimeout:           idleTimeout,
		RequestContextTimeout: contextTimeout,
	}
}

// TimeoutConfig holds all timeout configurations for the system
type TimeoutConfig struct {
	// RequestTimeout configures HTTP request-level timeouts
	RequestTimeout *RequestTimeoutConfig

	// DefaultContextTimeout is the default timeout for operations without explicit timeout
	// Default: 30 seconds
	DefaultContextTimeout time.Duration

	// DatabaseQueryTimeout is used when not specified by storage adapter
	// Default: 30 seconds
	DatabaseQueryTimeout time.Duration

	// BackgroundTaskTimeout is the timeout for background operations
	// Default: 5 minutes
	BackgroundTaskTimeout time.Duration
}

// DefaultTimeoutConfig returns sensible defaults for all system timeouts
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		RequestTimeout:        DefaultRequestTimeoutConfig(),
		DefaultContextTimeout: 30 * time.Second,
		DatabaseQueryTimeout:  30 * time.Second,
		BackgroundTaskTimeout: 5 * time.Minute,
	}
}

// FastTimeoutConfig returns aggressive timeout settings across all operations
func FastTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		RequestTimeout:        FastRequestTimeoutConfig(),
		DefaultContextTimeout: 10 * time.Second,
		DatabaseQueryTimeout:  10 * time.Second,
		BackgroundTaskTimeout: 1 * time.Minute,
	}
}

// RelaxedTimeoutConfig returns conservative timeout settings
func RelaxedTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		RequestTimeout:        RelaxedRequestTimeoutConfig(),
		DefaultContextTimeout: 120 * time.Second,
		DatabaseQueryTimeout:  120 * time.Second,
		BackgroundTaskTimeout: 30 * time.Minute,
	}
}

// CustomTimeoutConfig creates a fully customized timeout configuration
func CustomTimeoutConfig(
	requestTimeoutConfig *RequestTimeoutConfig,
	defaultContextTimeout time.Duration,
	databaseQueryTimeout time.Duration,
	backgroundTaskTimeout time.Duration,
) *TimeoutConfig {
	if requestTimeoutConfig == nil {
		requestTimeoutConfig = DefaultRequestTimeoutConfig()
	}
	if defaultContextTimeout <= 0 {
		defaultContextTimeout = 30 * time.Second
	}
	if databaseQueryTimeout <= 0 {
		databaseQueryTimeout = 30 * time.Second
	}
	if backgroundTaskTimeout <= 0 {
		backgroundTaskTimeout = 5 * time.Minute
	}
	return &TimeoutConfig{
		RequestTimeout:        requestTimeoutConfig,
		DefaultContextTimeout: defaultContextTimeout,
		DatabaseQueryTimeout:  databaseQueryTimeout,
		BackgroundTaskTimeout: backgroundTaskTimeout,
	}
}
