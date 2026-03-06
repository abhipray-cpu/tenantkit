package httpstd

import (
	"testing"
	"time"
)

// TestDefaultRequestTimeoutConfig verifies default HTTP request timeouts
func TestDefaultRequestTimeoutConfig(t *testing.T) {
	config := DefaultRequestTimeoutConfig()

	if config == nil {
		t.Fatal("DefaultRequestTimeoutConfig returned nil")
	}

	if config.ReadTimeout != 30*time.Second {
		t.Errorf("Expected ReadTimeout=30s, got %v", config.ReadTimeout)
	}

	if config.WriteTimeout != 30*time.Second {
		t.Errorf("Expected WriteTimeout=30s, got %v", config.WriteTimeout)
	}

	if config.IdleTimeout != 120*time.Second {
		t.Errorf("Expected IdleTimeout=120s, got %v", config.IdleTimeout)
	}

	if config.RequestContextTimeout != 60*time.Second {
		t.Errorf("Expected RequestContextTimeout=60s, got %v", config.RequestContextTimeout)
	}
}

// TestFastRequestTimeoutConfig verifies fast request timeout configuration
func TestFastRequestTimeoutConfig(t *testing.T) {
	config := FastRequestTimeoutConfig()

	if config == nil {
		t.Fatal("FastRequestTimeoutConfig returned nil")
	}

	if config.ReadTimeout != 10*time.Second {
		t.Errorf("Expected ReadTimeout=10s, got %v", config.ReadTimeout)
	}

	if config.RequestContextTimeout != 15*time.Second {
		t.Errorf("Expected RequestContextTimeout=15s, got %v", config.RequestContextTimeout)
	}

	// Fast should be faster than default
	if config.ReadTimeout >= DefaultRequestTimeoutConfig().ReadTimeout {
		t.Error("Fast config should have shorter timeouts than default")
	}
}

// TestRelaxedRequestTimeoutConfig verifies relaxed request timeout configuration
func TestRelaxedRequestTimeoutConfig(t *testing.T) {
	config := RelaxedRequestTimeoutConfig()

	if config == nil {
		t.Fatal("RelaxedRequestTimeoutConfig returned nil")
	}

	if config.ReadTimeout != 300*time.Second {
		t.Errorf("Expected ReadTimeout=300s, got %v", config.ReadTimeout)
	}

	// Relaxed should be longer than default
	if config.ReadTimeout <= DefaultRequestTimeoutConfig().ReadTimeout {
		t.Error("Relaxed config should have longer timeouts than default")
	}
}

// TestCustomRequestTimeoutConfig verifies custom request timeout creation
func TestCustomRequestTimeoutConfig(t *testing.T) {
	customRead := 15 * time.Second
	customWrite := 20 * time.Second
	customIdle := 45 * time.Second
	customCtx := 25 * time.Second

	config := CustomRequestTimeoutConfig(customRead, customWrite, customIdle, customCtx)

	if config == nil {
		t.Fatal("CustomRequestTimeoutConfig returned nil")
	}

	if config.ReadTimeout != customRead {
		t.Errorf("Expected ReadTimeout=%v, got %v", customRead, config.ReadTimeout)
	}

	if config.WriteTimeout != customWrite {
		t.Errorf("Expected WriteTimeout=%v, got %v", customWrite, config.WriteTimeout)
	}
}

// TestCustomRequestTimeoutConfig_InvalidTimeouts verifies invalid timeouts are corrected
func TestCustomRequestTimeoutConfig_InvalidTimeouts(t *testing.T) {
	config := CustomRequestTimeoutConfig(-1*time.Second, -1*time.Second, -1*time.Second, -1*time.Second)

	if config.ReadTimeout <= 0 {
		t.Error("Invalid ReadTimeout should be corrected to default")
	}

	if config.WriteTimeout <= 0 {
		t.Error("Invalid WriteTimeout should be corrected to default")
	}

	if config.IdleTimeout <= 0 {
		t.Error("Invalid IdleTimeout should be corrected to default")
	}

	if config.RequestContextTimeout <= 0 {
		t.Error("Invalid RequestContextTimeout should be corrected to default")
	}
}

// TestDefaultTimeoutConfig verifies default system-wide timeout configuration
func TestDefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()

	if config == nil {
		t.Fatal("DefaultTimeoutConfig returned nil")
	}

	if config.RequestTimeout == nil {
		t.Fatal("RequestTimeout should be initialized")
	}

	if config.DefaultContextTimeout != 30*time.Second {
		t.Errorf("Expected DefaultContextTimeout=30s, got %v", config.DefaultContextTimeout)
	}

	if config.DatabaseQueryTimeout != 30*time.Second {
		t.Errorf("Expected DatabaseQueryTimeout=30s, got %v", config.DatabaseQueryTimeout)
	}

	if config.BackgroundTaskTimeout != 5*time.Minute {
		t.Errorf("Expected BackgroundTaskTimeout=5m, got %v", config.BackgroundTaskTimeout)
	}
}

// TestFastTimeoutConfig verifies fast system-wide timeout configuration
func TestFastTimeoutConfig(t *testing.T) {
	config := FastTimeoutConfig()

	if config == nil {
		t.Fatal("FastTimeoutConfig returned nil")
	}

	if config.DefaultContextTimeout != 10*time.Second {
		t.Errorf("Expected DefaultContextTimeout=10s, got %v", config.DefaultContextTimeout)
	}

	if config.DatabaseQueryTimeout != 10*time.Second {
		t.Errorf("Expected DatabaseQueryTimeout=10s, got %v", config.DatabaseQueryTimeout)
	}

	// Fast should be faster than default
	if config.DefaultContextTimeout >= DefaultTimeoutConfig().DefaultContextTimeout {
		t.Error("Fast config should have shorter timeouts than default")
	}
}

// TestRelaxedTimeoutConfig verifies relaxed system-wide timeout configuration
func TestRelaxedTimeoutConfig(t *testing.T) {
	config := RelaxedTimeoutConfig()

	if config == nil {
		t.Fatal("RelaxedTimeoutConfig returned nil")
	}

	if config.DefaultContextTimeout != 120*time.Second {
		t.Errorf("Expected DefaultContextTimeout=120s, got %v", config.DefaultContextTimeout)
	}

	// Relaxed should be longer than default
	if config.DefaultContextTimeout <= DefaultTimeoutConfig().DefaultContextTimeout {
		t.Error("Relaxed config should have longer timeouts than default")
	}
}

// TestCustomTimeoutConfig verifies custom system-wide timeout configuration
func TestCustomTimeoutConfig(t *testing.T) {
	requestCfg := FastRequestTimeoutConfig()
	customCtx := 45 * time.Second
	customDB := 25 * time.Second
	customBG := 10 * time.Minute

	config := CustomTimeoutConfig(requestCfg, customCtx, customDB, customBG)

	if config == nil {
		t.Fatal("CustomTimeoutConfig returned nil")
	}

	if config.RequestTimeout != requestCfg {
		t.Error("RequestTimeout not set correctly")
	}

	if config.DefaultContextTimeout != customCtx {
		t.Errorf("Expected DefaultContextTimeout=%v, got %v", customCtx, config.DefaultContextTimeout)
	}

	if config.DatabaseQueryTimeout != customDB {
		t.Errorf("Expected DatabaseQueryTimeout=%v, got %v", customDB, config.DatabaseQueryTimeout)
	}

	if config.BackgroundTaskTimeout != customBG {
		t.Errorf("Expected BackgroundTaskTimeout=%v, got %v", customBG, config.BackgroundTaskTimeout)
	}
}

// TestCustomTimeoutConfig_InvalidTimeouts verifies invalid system timeouts are corrected
func TestCustomTimeoutConfig_InvalidTimeouts(t *testing.T) {
	config := CustomTimeoutConfig(nil, -1*time.Second, -1*time.Second, -1*time.Second)

	if config.RequestTimeout == nil {
		t.Error("RequestTimeout should default when nil")
	}

	if config.DefaultContextTimeout <= 0 {
		t.Error("Invalid DefaultContextTimeout should be corrected")
	}

	if config.DatabaseQueryTimeout <= 0 {
		t.Error("Invalid DatabaseQueryTimeout should be corrected")
	}

	if config.BackgroundTaskTimeout <= 0 {
		t.Error("Invalid BackgroundTaskTimeout should be corrected")
	}
}

// TestTimeoutConfig_Boundaries verifies timeout boundaries
func TestTimeoutConfig_Boundaries(t *testing.T) {
	tests := []struct {
		name         string
		readTimeout  time.Duration
		writeTimeout time.Duration
		idleTimeout  time.Duration
		ctxTimeout   time.Duration
	}{
		{"Very short", 100 * time.Millisecond, 100 * time.Millisecond, 100 * time.Millisecond, 100 * time.Millisecond},
		{"Very long", 10 * time.Minute, 10 * time.Minute, 30 * time.Minute, 30 * time.Minute},
		{"Mixed", 5 * time.Second, 15 * time.Second, 1 * time.Minute, 2 * time.Minute},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := CustomRequestTimeoutConfig(test.readTimeout, test.writeTimeout, test.idleTimeout, test.ctxTimeout)

			if config.ReadTimeout != test.readTimeout {
				t.Errorf("Expected ReadTimeout %v, got %v", test.readTimeout, config.ReadTimeout)
			}

			if config.WriteTimeout != test.writeTimeout {
				t.Errorf("Expected WriteTimeout %v, got %v", test.writeTimeout, config.WriteTimeout)
			}
		})
	}
}

// TestTimeoutConfig_Isolation verifies configurations don't interfere
func TestTimeoutConfig_Isolation(t *testing.T) {
	defaultCfg := DefaultTimeoutConfig()
	fastCfg := FastTimeoutConfig()
	relaxedCfg := RelaxedTimeoutConfig()

	// Verify they have different values
	if defaultCfg.DefaultContextTimeout == fastCfg.DefaultContextTimeout {
		t.Error("Default and fast configs should differ")
	}

	if defaultCfg.DefaultContextTimeout == relaxedCfg.DefaultContextTimeout {
		t.Error("Default and relaxed configs should differ")
	}

	if fastCfg.DefaultContextTimeout == relaxedCfg.DefaultContextTimeout {
		t.Error("Fast and relaxed configs should differ")
	}
}

// TestTimeoutConfig_RealWorldScenarios verifies production scenarios
func TestTimeoutConfig_RealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		name    string
		config  *TimeoutConfig
		purpose string
	}{
		{"Default", DefaultTimeoutConfig(), "General production use"},
		{"Fast", FastTimeoutConfig(), "Microservices / quick failure detection"},
		{"Relaxed", RelaxedTimeoutConfig(), "File uploads / long operations"},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if scenario.config == nil {
				t.Fatalf("%s config is nil", scenario.name)
			}

			if scenario.config.DefaultContextTimeout <= 0 {
				t.Errorf("%s: invalid DefaultContextTimeout %v", scenario.name, scenario.config.DefaultContextTimeout)
			}

			if scenario.config.RequestTimeout == nil {
				t.Errorf("%s: RequestTimeout is nil", scenario.name)
			}

			// Request context timeout should typically be >= longest request timeout
			longestRequestTimeout := scenario.config.RequestTimeout.ReadTimeout
			if scenario.config.RequestTimeout.WriteTimeout > longestRequestTimeout {
				longestRequestTimeout = scenario.config.RequestTimeout.WriteTimeout
			}
			if scenario.config.RequestTimeout.IdleTimeout > longestRequestTimeout {
				longestRequestTimeout = scenario.config.RequestTimeout.IdleTimeout
			}

			if scenario.config.RequestTimeout.RequestContextTimeout < longestRequestTimeout {
				t.Logf("%s: RequestContextTimeout (%v) should be >= longest request timeout (%v)",
					scenario.name, scenario.config.RequestTimeout.RequestContextTimeout, longestRequestTimeout)
			}
		})
	}
}

// TestRequestTimeoutConfig_TimeoutRelationships verifies sensible timeout relationships
func TestRequestTimeoutConfig_TimeoutRelationships(t *testing.T) {
	configs := []struct {
		name   string
		config *RequestTimeoutConfig
	}{
		{"Default", DefaultRequestTimeoutConfig()},
		{"Fast", FastRequestTimeoutConfig()},
		{"Relaxed", RelaxedRequestTimeoutConfig()},
	}

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			// All timeouts should be positive
			if tc.config.ReadTimeout <= 0 {
				t.Errorf("%s: ReadTimeout should be positive, got %v", tc.name, tc.config.ReadTimeout)
			}

			if tc.config.WriteTimeout <= 0 {
				t.Errorf("%s: WriteTimeout should be positive, got %v", tc.name, tc.config.WriteTimeout)
			}

			if tc.config.IdleTimeout <= 0 {
				t.Errorf("%s: IdleTimeout should be positive, got %v", tc.name, tc.config.IdleTimeout)
			}

			if tc.config.RequestContextTimeout <= 0 {
				t.Errorf("%s: RequestContextTimeout should be positive, got %v", tc.name, tc.config.RequestContextTimeout)
			}

			// Context timeout should typically be >= read/write timeouts
			// (client must have time to send/receive data before context timeout)
			if tc.config.RequestContextTimeout < tc.config.ReadTimeout {
				t.Logf("%s: context timeout (%v) < read timeout (%v) - may cause early cancellation",
					tc.name, tc.config.RequestContextTimeout, tc.config.ReadTimeout)
			}
		})
	}
}
