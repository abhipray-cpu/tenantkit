package gormadapter

import (
	"context"
	"testing"

	"github.com/abhipray-cpu/tenantkit/domain"
	"gorm.io/gorm"
)

// TestTenantPlugin tests the basic plugin functionality
func TestTenantPlugin(t *testing.T) {
	plugin := NewTenantPlugin(nil)

	if plugin == nil {
		t.Fatal("NewTenantPlugin returned nil")
	}

	if plugin.Name() != "tenantkit:gorm" {
		t.Errorf("Expected plugin name 'tenantkit:gorm', got %s", plugin.Name())
	}

	if plugin.tenantColumn != "tenant_id" {
		t.Errorf("Expected default column 'tenant_id', got %s", plugin.tenantColumn)
	}
}

// TestTenantPluginCustomConfig tests custom configuration
func TestTenantPluginCustomConfig(t *testing.T) {
	config := &PluginConfig{
		TenantColumn: "org_id",
		SkipTables:   []string{"migrations", "system_config"},
	}

	plugin := NewTenantPlugin(config)

	if plugin.tenantColumn != "org_id" {
		t.Errorf("Expected column 'org_id', got %s", plugin.tenantColumn)
	}

	if !plugin.skipTables["migrations"] {
		t.Error("Expected 'migrations' to be in skip list")
	}

	if !plugin.skipTables["system_config"] {
		t.Error("Expected 'system_config' to be in skip list")
	}
}

// TestShouldSkip tests the skip logic
func TestShouldSkip(t *testing.T) {
	config := &PluginConfig{
		SkipTables: []string{"migrations"},
	}
	plugin := NewTenantPlugin(config)

	// Create a mock DB with Statement properly initialized
	db := &gorm.DB{
		Statement: &gorm.Statement{
			Table: "users",
		},
	}

	// Should not skip by default
	if plugin.shouldSkip(db) {
		t.Error("Should not skip without skip flag")
	}

	// Should skip when flag is set
	db.Statement.Settings.Store("tenantkit:skip", true)
	if !plugin.shouldSkip(db) {
		t.Error("Should skip when flag is set")
	}

	// Should skip for tables in skip list
	db = &gorm.DB{
		Statement: &gorm.Statement{
			Table: "migrations",
		},
	}
	if !plugin.shouldSkip(db) {
		t.Error("Should skip for tables in skip list")
	}
}

// TestGetTenantID tests tenant ID extraction
func TestGetTenantID(t *testing.T) {
	plugin := NewTenantPlugin(nil)

	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "nil context",
			ctx:     nil,
			wantErr: true,
		},
		{
			name:    "empty context",
			ctx:     context.Background(),
			wantErr: true,
		},
		{
			name: "valid context",
			ctx: func() context.Context {
				tenantCtx, _ := domain.NewContext("tenant-123", "user-1", "req-1")
				return tenantCtx.ToGoContext(context.Background())
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantID, err := plugin.getTenantID(tt.ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("getTenantID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tenantID != "tenant-123" {
				t.Errorf("getTenantID() = %v, want tenant-123", tenantID)
			}
		})
	}
}

// TestSkipTenantScope tests the SkipTenant scope
func TestSkipTenantScope(t *testing.T) {
	db := &gorm.DB{
		Statement: &gorm.Statement{},
	}

	// Apply SkipTenant scope
	db = SkipTenant()(db)

	// Check if skip flag is set
	skip, ok := db.Statement.Settings.Load("tenantkit:skip")
	if !ok {
		t.Error("SkipTenant scope did not set skip flag")
	}

	if !skip.(bool) {
		t.Error("SkipTenant scope set skip flag to false")
	}
}

// TestWithTenantScope tests the WithTenant scope
func TestWithTenantScope(t *testing.T) {
	// This test would require a real DB connection
	// For now, just test that the scope function is created
	scope := WithTenant("tenant-123")
	if scope == nil {
		t.Error("WithTenant returned nil scope")
	}
}

// TestForTenantScope tests the ForTenant scope
func TestForTenantScope(t *testing.T) {
	tenantID, err := domain.NewTenantID("tenant-456")
	if err != nil {
		t.Fatalf("Failed to create tenant ID: %v", err)
	}

	scope := ForTenant(tenantID)
	if scope == nil {
		t.Error("ForTenant returned nil scope")
	}
}

// TestTenantOnlyScope tests the TenantOnly scope
func TestTenantOnlyScope(t *testing.T) {
	db := &gorm.DB{
		Statement: &gorm.Statement{},
	}

	// Apply TenantOnly scope
	result := TenantOnly()(db)

	// Should return the same DB (no-op)
	if result != db {
		t.Error("TenantOnly should return the same DB")
	}
}

// TestAllTenantsScope tests the AllTenants scope
func TestAllTenantsScope(t *testing.T) {
	db := &gorm.DB{
		Statement: &gorm.Statement{},
	}

	// Apply AllTenants scope
	db = AllTenants()(db)

	// Check if skip flag is set
	skip, ok := db.Statement.Settings.Load("tenantkit:skip")
	if !ok {
		t.Error("AllTenants scope did not set skip flag")
	}

	if !skip.(bool) {
		t.Error("AllTenants scope set skip flag to false")
	}
}

// TestCallbackHelper tests the callback helper
func TestCallbackHelper(t *testing.T) {
	helper := NewCallbackHelper("")
	if helper.tenantColumn != "tenant_id" {
		t.Errorf("Expected default column 'tenant_id', got %s", helper.tenantColumn)
	}

	helper = NewCallbackHelper("custom_column")
	if helper.tenantColumn != "custom_column" {
		t.Errorf("Expected column 'custom_column', got %s", helper.tenantColumn)
	}
}

// TestCleanCallbackName tests the callback name generator
func TestCleanCallbackName(t *testing.T) {
	name := CleanCallbackName("tenantkit", "before_create")
	expected := "tenantkit:before_create"

	if name != expected {
		t.Errorf("Expected %s, got %s", expected, name)
	}
}

// Integration test helper type
type User struct {
	ID       uint   `gorm:"primaryKey"`
	TenantID string `gorm:"column:tenant_id;not null"`
	Name     string
	Email    string
}

func (User) TableName() string {
	return "users"
}

// Note: Full integration tests would require a real database connection.
// These tests focus on unit testing the plugin logic.
