package gormadapter

import (
	"fmt"

	"gorm.io/gorm"
)

// CallbackHelper provides helper functions for working with GORM callbacks and tenant isolation.
type CallbackHelper struct {
	tenantColumn string
}

// NewCallbackHelper creates a new callback helper.
func NewCallbackHelper(tenantColumn string) *CallbackHelper {
	if tenantColumn == "" {
		tenantColumn = "tenant_id"
	}
	return &CallbackHelper{
		tenantColumn: tenantColumn,
	}
}

// EnsureTenantColumn checks if a table has the tenant_id column and adds it if missing.
// This is useful during migrations.
func (h *CallbackHelper) EnsureTenantColumn(db *gorm.DB, tableName string, columnType string) error {
	if columnType == "" {
		columnType = "VARCHAR(255)"
	}

	// Check if column exists
	hasColumn := db.Migrator().HasColumn(tableName, h.tenantColumn)
	if hasColumn {
		return nil
	}

	// Add column
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, h.tenantColumn, columnType)
	return db.Exec(sql).Error
}

// AddTenantIndex adds an index on the tenant_id column for better query performance.
func (h *CallbackHelper) AddTenantIndex(db *gorm.DB, tableName string) error {
	indexName := fmt.Sprintf("idx_%s_tenant_id", tableName)
	sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)", indexName, tableName, h.tenantColumn)
	return db.Exec(sql).Error
}

// MigrateTenantColumn ensures the tenant_id column exists and is indexed.
func (h *CallbackHelper) MigrateTenantColumn(db *gorm.DB, models ...interface{}) error {
	for _, model := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			return fmt.Errorf("failed to parse model: %w", err)
		}

		tableName := stmt.Table

		// Ensure column exists
		if err := h.EnsureTenantColumn(db, tableName, "VARCHAR(255) NOT NULL"); err != nil {
			return fmt.Errorf("failed to ensure tenant column for %s: %w", tableName, err)
		}

		// Add index
		if err := h.AddTenantIndex(db, tableName); err != nil {
			return fmt.Errorf("failed to add tenant index for %s: %w", tableName, err)
		}
	}

	return nil
}

// AutoMigrate is a wrapper around GORM's AutoMigrate that also ensures tenant columns.
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	// First run standard GORM AutoMigrate
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}

	// Then ensure tenant columns and indexes
	helper := NewCallbackHelper("tenant_id")
	return helper.MigrateTenantColumn(db, models...)
}

// WithoutTenantCallbacks creates a new DB session with tenant callbacks temporarily disabled.
// This is useful for migrations or admin operations.
func WithoutTenantCallbacks(db *gorm.DB) *gorm.DB {
	return db.Session(&gorm.Session{
		SkipHooks: true,
	}).Scopes(SkipTenant())
}

// CleanCallbackName returns a clean callback name for registration.
func CleanCallbackName(prefix, operation string) string {
	return fmt.Sprintf("%s:%s", prefix, operation)
}
