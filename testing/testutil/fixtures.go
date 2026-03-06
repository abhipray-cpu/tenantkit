// Package testutil provides testing utilities for TenantKit
// This file contains test fixtures and mock data generators
package testutil

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// TestTenant represents a tenant for testing
type TestTenant struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Status string                 `json:"status"`
	Tier   string                 `json:"tier"`
	Config map[string]interface{} `json:"config"`
	Quotas map[string]TestQuota   `json:"quotas,omitempty"`
}

// TestQuota represents a quota for testing
type TestQuota struct {
	Limit  int64  `json:"limit"`
	Period string `json:"period"`
	Used   int64  `json:"used"`
}

// FixtureLoader loads test fixtures from JSON files
type FixtureLoader struct {
	basePath string
}

// NewFixtureLoader creates a new fixture loader
func NewFixtureLoader(basePath string) *FixtureLoader {
	return &FixtureLoader{basePath: basePath}
}

// LoadTenants loads tenant fixtures from a JSON file
func (fl *FixtureLoader) LoadTenants(filename string) ([]TestTenant, error) {
	path := filepath.Join(fl.basePath, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixture file: %w", err)
	}

	var result struct {
		Tenants []TestTenant `json:"tenants"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse fixture: %w", err)
	}

	return result.Tenants, nil
}

// LoadJSON loads any JSON fixture file
func (fl *FixtureLoader) LoadJSON(filename string, v interface{}) error {
	path := filepath.Join(fl.basePath, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read fixture file: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse fixture: %w", err)
	}

	return nil
}

// MockDataGenerator generates mock data for testing
type MockDataGenerator struct {
	rng *rand.Rand
}

// NewMockDataGenerator creates a new mock data generator
func NewMockDataGenerator() *MockDataGenerator {
	return &MockDataGenerator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewMockDataGeneratorWithSeed creates a seeded generator for reproducibility
func NewMockDataGeneratorWithSeed(seed int64) *MockDataGenerator {
	return &MockDataGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// GenerateTenantID generates a random tenant ID
func (g *MockDataGenerator) GenerateTenantID() string {
	return fmt.Sprintf("tenant-%08x", g.rng.Uint32())
}

// GenerateUserID generates a random user ID
func (g *MockDataGenerator) GenerateUserID() string {
	return fmt.Sprintf("user-%08x", g.rng.Uint32())
}

// GenerateTenant generates a test tenant
func (g *MockDataGenerator) GenerateTenant(opts ...TenantOption) TestTenant {
	tenant := TestTenant{
		ID:     g.GenerateTenantID(),
		Name:   fmt.Sprintf("Test Tenant %d", g.rng.Intn(10000)),
		Status: "active",
		Tier:   "standard",
		Config: map[string]interface{}{
			"max_requests_per_minute": 1000,
			"max_storage_bytes":       1073741824,
			"max_connections":         50,
		},
		Quotas: map[string]TestQuota{
			"api_requests": {
				Limit:  100000,
				Period: "monthly",
				Used:   0,
			},
		},
	}

	for _, opt := range opts {
		opt(&tenant)
	}

	return tenant
}

// TenantOption is a function that modifies a TestTenant
type TenantOption func(*TestTenant)

// WithTenantID sets the tenant ID
func WithTenantID(id string) TenantOption {
	return func(t *TestTenant) {
		t.ID = id
	}
}

// WithTenantName sets the tenant name
func WithTenantName(name string) TenantOption {
	return func(t *TestTenant) {
		t.Name = name
	}
}

// WithTenantStatus sets the tenant status
func WithTenantStatus(status string) TenantOption {
	return func(t *TestTenant) {
		t.Status = status
	}
}

// WithTenantTier sets the tenant tier
func WithTenantTier(tier string) TenantOption {
	return func(t *TestTenant) {
		t.Tier = tier
	}
}

// WithTenantQuota adds or updates a quota
func WithTenantQuota(name string, quota TestQuota) TenantOption {
	return func(t *TestTenant) {
		if t.Quotas == nil {
			t.Quotas = make(map[string]TestQuota)
		}
		t.Quotas[name] = quota
	}
}

// WithTenantConfig sets a config value
func WithTenantConfig(key string, value interface{}) TenantOption {
	return func(t *TestTenant) {
		if t.Config == nil {
			t.Config = make(map[string]interface{})
		}
		t.Config[key] = value
	}
}

// GenerateTenants generates multiple test tenants
func (g *MockDataGenerator) GenerateTenants(count int, opts ...TenantOption) []TestTenant {
	tenants := make([]TestTenant, count)
	for i := 0; i < count; i++ {
		tenants[i] = g.GenerateTenant(opts...)
	}
	return tenants
}

// GenerateRandomString generates a random string of the specified length
func (g *MockDataGenerator) GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[g.rng.Intn(len(charset))]
	}
	return string(result)
}

// GenerateEmail generates a random email address
func (g *MockDataGenerator) GenerateEmail() string {
	return fmt.Sprintf("user%d@example.com", g.rng.Intn(100000))
}

// Predefined test tenants
var (
	// EnterpriseTenant is a high-tier tenant for testing
	EnterpriseTenant = TestTenant{
		ID:     "enterprise-tenant",
		Name:   "Enterprise Corp",
		Status: "active",
		Tier:   "enterprise",
		Config: map[string]interface{}{
			"max_requests_per_minute": 10000,
			"max_storage_bytes":       int64(10 * 1024 * 1024 * 1024), // 10GB
			"max_connections":         100,
		},
		Quotas: map[string]TestQuota{
			"api_requests": {Limit: 1000000, Period: "monthly", Used: 0},
			"storage":      {Limit: 10737418240, Period: "none", Used: 0},
		},
	}

	// StarterTenant is a low-tier tenant for testing
	StarterTenant = TestTenant{
		ID:     "starter-tenant",
		Name:   "Startup Inc",
		Status: "active",
		Tier:   "starter",
		Config: map[string]interface{}{
			"max_requests_per_minute": 100,
			"max_storage_bytes":       int64(1 * 1024 * 1024 * 1024), // 1GB
			"max_connections":         10,
		},
		Quotas: map[string]TestQuota{
			"api_requests": {Limit: 10000, Period: "monthly", Used: 0},
			"storage":      {Limit: 1073741824, Period: "none", Used: 0},
		},
	}

	// SuspendedTenant is a suspended tenant for testing
	SuspendedTenant = TestTenant{
		ID:     "suspended-tenant",
		Name:   "Suspended Co",
		Status: "suspended",
		Tier:   "standard",
		Config: map[string]interface{}{
			"max_requests_per_minute": 0,
		},
	}

	// NoisyTenant is for noisy neighbor testing
	NoisyTenant = TestTenant{
		ID:     "noisy-tenant",
		Name:   "Noisy Neighbor",
		Status: "active",
		Tier:   "starter",
		Config: map[string]interface{}{
			"max_requests_per_minute": 100,
		},
		Quotas: map[string]TestQuota{
			"api_requests": {Limit: 1000, Period: "hourly", Used: 0},
		},
	}

	// VictimTenant is for noisy neighbor testing (the affected tenant)
	VictimTenant = TestTenant{
		ID:     "victim-tenant",
		Name:   "Normal User",
		Status: "active",
		Tier:   "professional",
		Config: map[string]interface{}{
			"max_requests_per_minute": 1000,
		},
		Quotas: map[string]TestQuota{
			"api_requests": {Limit: 100000, Period: "monthly", Used: 0},
		},
	}
)
