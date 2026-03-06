package sqladapter

import (
	"testing"
	"time"
)

// TestDefaultHealthCheckConfig verifies default configuration values
func TestDefaultHealthCheckConfig(t *testing.T) {
	config := DefaultHealthCheckConfig()

	if config == nil {
		t.Fatal("DefaultHealthCheckConfig returned nil")
	}

	if config.Timeout != 5*time.Second {
		t.Errorf("Expected Timeout=5s, got %v", config.Timeout)
	}

	if config.Interval != 30*time.Second {
		t.Errorf("Expected Interval=30s, got %v", config.Interval)
	}
}

// TestFastHealthCheckConfig verifies fast configuration
func TestFastHealthCheckConfig(t *testing.T) {
	config := FastHealthCheckConfig()

	if config == nil {
		t.Fatal("FastHealthCheckConfig returned nil")
	}

	if config.Timeout != 1*time.Second {
		t.Errorf("Expected Timeout=1s, got %v", config.Timeout)
	}

	if config.Interval != 5*time.Second {
		t.Errorf("Expected Interval=5s, got %v", config.Interval)
	}

	// Fast should be faster than default
	if config.Timeout >= DefaultHealthCheckConfig().Timeout {
		t.Error("Fast config timeout should be less than default")
	}
}

// TestRelaxedHealthCheckConfig verifies relaxed configuration
func TestRelaxedHealthCheckConfig(t *testing.T) {
	config := RelaxedHealthCheckConfig()

	if config == nil {
		t.Fatal("RelaxedHealthCheckConfig returned nil")
	}

	if config.Timeout != 10*time.Second {
		t.Errorf("Expected Timeout=10s, got %v", config.Timeout)
	}

	if config.Interval != 60*time.Second {
		t.Errorf("Expected Interval=60s, got %v", config.Interval)
	}

	// Relaxed should be slower than default
	if config.Timeout <= DefaultHealthCheckConfig().Timeout {
		t.Error("Relaxed config timeout should be greater than default")
	}
}

// TestCustomHealthCheckConfig verifies custom configuration creation
func TestCustomHealthCheckConfig(t *testing.T) {
	customTimeout := 2 * time.Second
	customInterval := 15 * time.Second

	config := CustomHealthCheckConfig(customTimeout, customInterval)

	if config == nil {
		t.Fatal("CustomHealthCheckConfig returned nil")
	}

	if config.Timeout != customTimeout {
		t.Errorf("Expected Timeout=%v, got %v", customTimeout, config.Timeout)
	}

	if config.Interval != customInterval {
		t.Errorf("Expected Interval=%v, got %v", customInterval, config.Interval)
	}
}

// TestCustomHealthCheckConfig_InvalidTimeout verifies invalid timeout is corrected
func TestCustomHealthCheckConfig_InvalidTimeout(t *testing.T) {
	config := CustomHealthCheckConfig(-1*time.Second, 30*time.Second)

	if config.Timeout <= 0 {
		t.Error("Invalid timeout should be corrected to default")
	}

	if config.Timeout != 5*time.Second {
		t.Errorf("Expected corrected timeout=5s, got %v", config.Timeout)
	}
}

// TestCustomHealthCheckConfig_InvalidInterval verifies invalid interval is corrected
func TestCustomHealthCheckConfig_InvalidInterval(t *testing.T) {
	config := CustomHealthCheckConfig(5*time.Second, -1*time.Second)

	if config.Interval <= 0 {
		t.Error("Invalid interval should be corrected to default")
	}

	if config.Interval != 30*time.Second {
		t.Errorf("Expected corrected interval=30s, got %v", config.Interval)
	}
}

// TestStorageConfigBuilder_WithHealthCheckConfig verifies builder integration
func TestStorageConfigBuilder_WithHealthCheckConfig(t *testing.T) {
	hc := FastHealthCheckConfig()

	config := NewStorageConfigBuilder().
		WithHealthCheckConfig(hc).
		Build()

	if config.HealthCheckConfig == nil {
		t.Fatal("HealthCheckConfig should be set")
	}

	if config.HealthCheckConfig.Timeout != hc.Timeout {
		t.Error("Health check config not applied correctly")
	}
}

// TestStorageConfigBuilder_DefaultHealthCheckConfig verifies defaults are used
func TestStorageConfigBuilder_DefaultHealthCheckConfig(t *testing.T) {
	config := NewStorageConfigBuilder().Build()

	if config.HealthCheckConfig == nil {
		t.Fatal("HealthCheckConfig should be initialized")
	}

	defaultHC := DefaultHealthCheckConfig()
	if config.HealthCheckConfig.Timeout != defaultHC.Timeout {
		t.Error("Should use default health check timeout")
	}
}

// TestHealthCheckConfig_Boundaries verifies configuration boundaries
func TestHealthCheckConfig_Boundaries(t *testing.T) {
	tests := []struct {
		name     string
		timeout  time.Duration
		interval time.Duration
	}{
		{"Very short timeout", 100 * time.Millisecond, 1 * time.Second},
		{"Very long timeout", 30 * time.Second, 5 * time.Minute},
		{"Sub-second interval", 5 * time.Second, 500 * time.Millisecond},
		{"Multi-minute interval", 10 * time.Second, 10 * time.Minute},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := CustomHealthCheckConfig(test.timeout, test.interval)
			if config.Timeout != test.timeout {
				t.Errorf("Expected timeout %v, got %v", test.timeout, config.Timeout)
			}
			if config.Interval != test.interval {
				t.Errorf("Expected interval %v, got %v", test.interval, config.Interval)
			}
		})
	}
}

// TestHealthCheckConfig_Isolation verifies configurations don't interfere
func TestHealthCheckConfig_Isolation(t *testing.T) {
	defaultConfig := DefaultHealthCheckConfig()
	fastConfig := FastHealthCheckConfig()
	relaxedConfig := RelaxedHealthCheckConfig()

	// Verify they have different values
	if defaultConfig.Timeout == fastConfig.Timeout {
		t.Error("Default and fast configs should differ")
	}

	if defaultConfig.Timeout == relaxedConfig.Timeout {
		t.Error("Default and relaxed configs should differ")
	}

	if fastConfig.Timeout == relaxedConfig.Timeout {
		t.Error("Fast and relaxed configs should differ")
	}
}

// TestHealthCheckConfig_WithNilInterval verifies interval defaults are applied
func TestHealthCheckConfig_WithNilInterval(t *testing.T) {
	config := CustomHealthCheckConfig(2*time.Second, 0)

	if config.Interval != 30*time.Second {
		t.Errorf("Expected default interval when 0 provided, got %v", config.Interval)
	}
}

// TestHealthCheckConfig_RealWorldScenarios verifies production scenarios
func TestHealthCheckConfig_RealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		name     string
		config   *HealthCheckConfig
		purpose  string
	}{
		{"Default", DefaultHealthCheckConfig(), "General production use"},
		{"Fast HA", FastHealthCheckConfig(), "High-availability deployments"},
		{"Relaxed Stable", RelaxedHealthCheckConfig(), "Stable internal environments"},
		{"Custom", CustomHealthCheckConfig(3*time.Second, 20*time.Second), "Fine-tuned deployments"},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if scenario.config == nil {
				t.Fatalf("%s config is nil", scenario.name)
			}

			if scenario.config.Timeout <= 0 {
				t.Errorf("%s: invalid timeout %v", scenario.name, scenario.config.Timeout)
			}

			if scenario.config.Interval <= 0 {
				t.Errorf("%s: invalid interval %v", scenario.name, scenario.config.Interval)
			}

			// Interval should typically be >= timeout for practical health checks
			if scenario.config.Interval < scenario.config.Timeout {
				t.Logf("%s: interval (%v) < timeout (%v) - may cause rapid retries", 
					scenario.name, scenario.config.Interval, scenario.config.Timeout)
			}
		})
	}
}
