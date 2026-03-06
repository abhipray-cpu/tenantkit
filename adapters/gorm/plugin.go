package gormadapter

import (
	"context"
	"fmt"
	"reflect"

	"github.com/abhipray-cpu/tenantkit/domain"
	"gorm.io/gorm"
)

// TenantPlugin is a GORM plugin that automatically adds tenant_id to all queries.
// It implements the gorm.Plugin interface and registers callbacks for all CRUD operations.
type TenantPlugin struct {
	tenantColumn string
	skipTables   map[string]bool
}

// PluginConfig holds configuration for the TenantPlugin.
type PluginConfig struct {
	// TenantColumn is the name of the tenant_id column (default: "tenant_id")
	TenantColumn string

	// SkipTables is a list of table names that should NOT be tenant-scoped
	// (e.g., migrations, system tables)
	SkipTables []string
}

// NewTenantPlugin creates a new GORM tenant plugin with the given configuration.
func NewTenantPlugin(config *PluginConfig) *TenantPlugin {
	if config == nil {
		config = &PluginConfig{}
	}

	if config.TenantColumn == "" {
		config.TenantColumn = "tenant_id"
	}

	skipTables := make(map[string]bool)
	for _, table := range config.SkipTables {
		skipTables[table] = true
	}

	return &TenantPlugin{
		tenantColumn: config.TenantColumn,
		skipTables:   skipTables,
	}
}

// Name returns the plugin name.
func (p *TenantPlugin) Name() string {
	return "tenantkit:gorm"
}

// Initialize registers the plugin callbacks with GORM.
func (p *TenantPlugin) Initialize(db *gorm.DB) error {
	// Register callbacks for all CRUD operations
	if err := p.registerCreateCallbacks(db); err != nil {
		return fmt.Errorf("failed to register create callbacks: %w", err)
	}

	if err := p.registerQueryCallbacks(db); err != nil {
		return fmt.Errorf("failed to register query callbacks: %w", err)
	}

	if err := p.registerUpdateCallbacks(db); err != nil {
		return fmt.Errorf("failed to register update callbacks: %w", err)
	}

	if err := p.registerDeleteCallbacks(db); err != nil {
		return fmt.Errorf("failed to register delete callbacks: %w", err)
	}

	return nil
}

// registerCreateCallbacks registers callbacks for CREATE operations.
func (p *TenantPlugin) registerCreateCallbacks(db *gorm.DB) error {
	return db.Callback().Create().Before("gorm:create").Register("tenantkit:before_create", p.beforeCreate)
}

// registerQueryCallbacks registers callbacks for QUERY operations.
func (p *TenantPlugin) registerQueryCallbacks(db *gorm.DB) error {
	return db.Callback().Query().Before("gorm:query").Register("tenantkit:before_query", p.beforeQuery)
}

// registerUpdateCallbacks registers callbacks for UPDATE operations.
func (p *TenantPlugin) registerUpdateCallbacks(db *gorm.DB) error {
	return db.Callback().Update().Before("gorm:update").Register("tenantkit:before_update", p.beforeUpdate)
}

// registerDeleteCallbacks registers callbacks for DELETE operations.
func (p *TenantPlugin) registerDeleteCallbacks(db *gorm.DB) error {
	return db.Callback().Delete().Before("gorm:delete").Register("tenantkit:before_delete", p.beforeDelete)
}

// beforeCreate is called before creating a record.
// It automatically sets the tenant_id field.
func (p *TenantPlugin) beforeCreate(db *gorm.DB) {
	if p.shouldSkip(db) {
		return
	}

	tenantID, err := p.getTenantID(db.Statement.Context)
	if err != nil {
		db.AddError(fmt.Errorf("tenant context required for create: %w", err))
		return
	}

	// Use GORM's ReflectValue to properly handle both single and batch inserts
	reflectValue := db.Statement.ReflectValue

	switch reflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		// Batch insert - set tenant_id on all items
		for i := 0; i < reflectValue.Len(); i++ {
			item := reflectValue.Index(i)
			if item.Kind() == reflect.Ptr {
				item = item.Elem()
			}
			if item.Kind() == reflect.Struct {
				tenantField := item.FieldByName("TenantID")
				if tenantField.IsValid() && tenantField.CanSet() {
					tenantField.SetString(tenantID)
				}
			}
		}
	case reflect.Struct:
		// Single insert - use SetColumn for better GORM integration
		db.Statement.SetColumn(p.tenantColumn, tenantID)
	case reflect.Ptr:
		// Pointer to struct
		if reflectValue.Elem().Kind() == reflect.Struct {
			db.Statement.SetColumn(p.tenantColumn, tenantID)
		}
	}
}

// beforeQuery is called before executing a query.
// It automatically adds WHERE tenant_id = ? to the query.
func (p *TenantPlugin) beforeQuery(db *gorm.DB) {
	if p.shouldSkip(db) {
		return
	}

	tenantID, err := p.getTenantID(db.Statement.Context)
	if err != nil {
		db.AddError(fmt.Errorf("tenant context required for query: %w", err))
		return
	}

	// Add WHERE clause for tenant_id
	db.Statement.Where(fmt.Sprintf("%s = ?", p.tenantColumn), tenantID)
}

// beforeUpdate is called before updating a record.
// It automatically adds WHERE tenant_id = ? to the update query.
func (p *TenantPlugin) beforeUpdate(db *gorm.DB) {
	if p.shouldSkip(db) {
		return
	}

	tenantID, err := p.getTenantID(db.Statement.Context)
	if err != nil {
		db.AddError(fmt.Errorf("tenant context required for update: %w", err))
		return
	}

	// Add WHERE clause for tenant_id
	db.Statement.Where(fmt.Sprintf("%s = ?", p.tenantColumn), tenantID)
}

// beforeDelete is called before deleting a record.
// It automatically adds WHERE tenant_id = ? to the delete query.
func (p *TenantPlugin) beforeDelete(db *gorm.DB) {
	if p.shouldSkip(db) {
		return
	}

	tenantID, err := p.getTenantID(db.Statement.Context)
	if err != nil {
		db.AddError(fmt.Errorf("tenant context required for delete: %w", err))
		return
	}

	// Add WHERE clause for tenant_id
	db.Statement.Where(fmt.Sprintf("%s = ?", p.tenantColumn), tenantID)
}

// getTenantID extracts the tenant ID from the context.
func (p *TenantPlugin) getTenantID(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}

	tenantCtx, err := domain.FromGoContext(ctx)
	if err != nil {
		return "", err
	}

	return tenantCtx.TenantID().Value(), nil
}

// shouldSkip checks if the current operation should skip tenant scoping.
func (p *TenantPlugin) shouldSkip(db *gorm.DB) bool {
	// Skip if explicitly requested via SkipTenant scope
	if skip, ok := db.Statement.Settings.Load("tenantkit:skip"); ok && skip.(bool) {
		return true
	}

	// Skip if table is in the skip list
	if p.skipTables[db.Statement.Table] {
		return true
	}

	return false
}

// Ensure TenantPlugin implements gorm.Plugin
var _ gorm.Plugin = (*TenantPlugin)(nil)
