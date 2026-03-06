package sqladapter

import (
	"time"
)

// HealthCheckConfig holds configuration for database health checks
type HealthCheckConfig struct {
	// Timeout is the maximum duration to wait for a health check ping
	// Default: 5 seconds
	Timeout time.Duration

	// Interval is the recommended interval between health checks
	// This is advisory and used by callers to schedule checks
	// Default: 30 seconds
	Interval time.Duration
}

// DefaultHealthCheckConfig returns sensible defaults for health checking
func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Timeout:  5 * time.Second,
		Interval: 30 * time.Second,
	}
}

// FastHealthCheckConfig returns aggressive health check settings
// Useful for high-availability setups where rapid failure detection is critical
func FastHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Timeout:  1 * time.Second,
		Interval: 5 * time.Second,
	}
}

// RelaxedHealthCheckConfig returns conservative health check settings
// Useful for stable environments or to reduce database load
func RelaxedHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Timeout:  10 * time.Second,
		Interval: 60 * time.Second,
	}
}

// CustomHealthCheckConfig creates health check config with custom settings
func CustomHealthCheckConfig(timeout, interval time.Duration) *HealthCheckConfig {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &HealthCheckConfig{
		Timeout:  timeout,
		Interval: interval,
	}
}
