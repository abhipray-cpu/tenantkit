// Package sqlx provides multi-tenant database access with sqlx.
// This adapter wraps sqlx.DB to enforce tenant isolation on all database operations.
//
// Design Philosophy:
// - Code designed for PostgreSQL/MySQL production use
// - Automatic tenant_id injection for all operations
// - Support for named queries and struct scanning
// - Transaction support with tenant context propagation
// - Manual control via scopes when needed
package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/abhipray-cpu/tenantkit/domain"
	"github.com/jmoiron/sqlx"
)

// DB wraps sqlx.DB with tenant enforcement
type DB struct {
	*sqlx.DB
	tenantColumn string
	skipTables   map[string]bool
}

// Config configures the tenant-aware DB wrapper
type Config struct {
	// TenantColumn is the column name used for tenant isolation (default: "tenant_id")
	TenantColumn string
	// SkipTables is a list of table names that should not have tenant enforcement
	SkipTables []string
}

// New creates a new tenant-aware sqlx.DB wrapper
func New(db *sqlx.DB, cfg *Config) *DB {
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.TenantColumn == "" {
		cfg.TenantColumn = "tenant_id"
	}

	skipTables := make(map[string]bool)
	for _, table := range cfg.SkipTables {
		skipTables[table] = true
	}

	return &DB{
		DB:           db,
		tenantColumn: cfg.TenantColumn,
		skipTables:   skipTables,
	}
}

// Open opens a database connection and returns a tenant-aware wrapper
func Open(driverName, dataSourceName string, cfg *Config) (*DB, error) {
	db, err := sqlx.Open(driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return New(db, cfg), nil
}

// Connect connects to a database and returns a tenant-aware wrapper
func Connect(driverName, dataSourceName string, cfg *Config) (*DB, error) {
	db, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return New(db, cfg), nil
}

// getTenantID extracts tenant ID from context
func (db *DB) getTenantID(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}

	tenantCtx, err := domain.FromGoContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to extract tenant context: %w", err)
	}

	return tenantCtx.TenantID().Value(), nil
}

// ContextKey is a type for storing context values
type ContextKey string

const (
	// SkipTenantKey is used to skip tenant enforcement
	SkipTenantKey ContextKey = "sqlx_skip_tenant"
)

// insertArgAtPosition inserts tenantID at the specified position in args
func insertArgAtPosition(args []interface{}, pos int, tenantID string) []interface{} {
	newArgs := make([]interface{}, len(args)+1)
	copy(newArgs[:pos], args[:pos])
	newArgs[pos] = tenantID
	copy(newArgs[pos+1:], args[pos:])
	return newArgs
}

// shouldSkip checks if tenant enforcement should be skipped for this operation
func (db *DB) shouldSkip(ctx context.Context, query string) bool {
	// Check if context has skip flag
	if skip, ok := ctx.Value(SkipTenantKey).(bool); ok && skip {
		return true
	}

	// Check if query is for a skip table
	queryLower := strings.ToLower(query)
	for table := range db.skipTables {
		if strings.Contains(queryLower, strings.ToLower(table)) {
			return true
		}
	}

	return false
}

// injectTenantCondition adds tenant condition to WHERE clause.
// The tenantID parameter is kept for API consistency but not used directly -
// the caller inserts it into args at the returned position.
//
//nolint:revive,unused // tenantID parameter kept for API clarity
func (db *DB) injectTenantCondition(query string, tenantID string) (string, int) {
	_ = tenantID // Parameter kept for API consistency
	queryLower := strings.ToLower(query)
	tenantCondition := fmt.Sprintf("%s = ?", db.tenantColumn)

	// Count existing placeholders before WHERE to determine position
	wherePos := strings.Index(queryLower, "where")
	placeholderPos := 0
	if wherePos != -1 {
		// Count ? before WHERE
		placeholderPos = strings.Count(query[:wherePos], "?")
	}

	// If query has WHERE clause, add AND condition
	if strings.Contains(queryLower, "where") {
		// Find WHERE position
		beforeWhere := query[:wherePos+5] // "WHERE"
		afterWhere := query[wherePos+5:]
		query = beforeWhere + " " + tenantCondition + " AND" + afterWhere
	} else {
		// Add WHERE clause before ORDER BY, LIMIT, etc.
		insertPos := len(query)
		for _, keyword := range []string{"order by", "limit", "offset", "group by", "having"} {
			if pos := strings.Index(queryLower, keyword); pos != -1 && pos < insertPos {
				insertPos = pos
			}
		}
		// Count existing placeholders
		placeholderPos = strings.Count(query, "?")
		query = query[:insertPos] + " WHERE " + tenantCondition + " " + query[insertPos:]
	}

	return query, placeholderPos
}

// QueryContext executes a query with tenant enforcement
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if db.shouldSkip(ctx, query) {
		return db.DB.QueryContext(ctx, query, args...)
	}

	tenantID, err := db.getTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant enforcement failed: %w", err)
	}

	query, pos := db.injectTenantCondition(query, tenantID)
	// Insert tenant_id at the correct position
	newArgs := make([]interface{}, len(args)+1)
	copy(newArgs[:pos], args[:pos])
	newArgs[pos] = tenantID
	copy(newArgs[pos+1:], args[pos:])

	return db.DB.QueryContext(ctx, query, newArgs...)
}

// QueryxContext executes a query and returns an sqlx.Rows with tenant enforcement
func (db *DB) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	if db.shouldSkip(ctx, query) {
		return db.DB.QueryxContext(ctx, query, args...)
	}

	tenantID, err := db.getTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant enforcement failed: %w", err)
	}

	query, pos := db.injectTenantCondition(query, tenantID)
	newArgs := insertArgAtPosition(args, pos, tenantID)

	return db.DB.QueryxContext(ctx, query, newArgs...)
}

// QueryRowxContext executes a query that returns at most one row with tenant enforcement
func (db *DB) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	if db.shouldSkip(ctx, query) {
		return db.DB.QueryRowxContext(ctx, query, args...)
	}

	tenantID, err := db.getTenantID(ctx)
	if err != nil {
		// sqlx.Row doesn't support errors before Scan, so we return a row that will error on Scan
		return db.DB.QueryRowxContext(ctx, "SELECT 1 WHERE 1=0")
	}

	query, pos := db.injectTenantCondition(query, tenantID)
	newArgs := insertArgAtPosition(args, pos, tenantID)

	return db.DB.QueryRowxContext(ctx, query, newArgs...)
}

// GetContext executes a query and scans the result into dest with tenant enforcement
func (db *DB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	if db.shouldSkip(ctx, query) {
		return db.DB.GetContext(ctx, dest, query, args...)
	}

	tenantID, err := db.getTenantID(ctx)
	if err != nil {
		return fmt.Errorf("tenant enforcement failed: %w", err)
	}

	query, pos := db.injectTenantCondition(query, tenantID)
	newArgs := insertArgAtPosition(args, pos, tenantID)

	return db.DB.GetContext(ctx, dest, query, newArgs...)
}

// SelectContext executes a query and scans the results into dest with tenant enforcement
func (db *DB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	if db.shouldSkip(ctx, query) {
		return db.DB.SelectContext(ctx, dest, query, args...)
	}

	tenantID, err := db.getTenantID(ctx)
	if err != nil {
		return fmt.Errorf("tenant enforcement failed: %w", err)
	}

	query, pos := db.injectTenantCondition(query, tenantID)
	newArgs := insertArgAtPosition(args, pos, tenantID)

	return db.DB.SelectContext(ctx, dest, query, newArgs...)
}

// ExecContext executes a query with tenant enforcement
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if db.shouldSkip(ctx, query) {
		return db.DB.ExecContext(ctx, query, args...)
	}

	tenantID, err := db.getTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant enforcement failed: %w", err)
	}

	// For INSERT queries, we need to add tenant_id to the column list
	queryLower := strings.ToLower(query)
	if strings.HasPrefix(strings.TrimSpace(queryLower), "insert") {
		query = db.injectTenantIntoInsert(query, tenantID)
		// Append tenant_id to args
		args = append(args, tenantID)
		return db.DB.ExecContext(ctx, query, args...)
	}

	// For UPDATE/DELETE queries, add WHERE condition
	query, pos := db.injectTenantCondition(query, tenantID)
	newArgs := insertArgAtPosition(args, pos, tenantID)

	return db.DB.ExecContext(ctx, query, newArgs...)
}

// injectTenantIntoInsert adds tenant_id to INSERT statement.
// The tenantID parameter is kept for API consistency but not used directly -
// the caller appends it to args after modifying the query.
//
//nolint:revive,unused // tenantID parameter kept for API clarity
func (db *DB) injectTenantIntoInsert(query string, tenantID string) string {
	_ = tenantID // Parameter kept for API consistency
	queryLower := strings.ToLower(query)

	// Find the VALUES position
	valuesPos := strings.Index(queryLower, "values")
	if valuesPos == -1 {
		return query
	}

	// Find column list (between INSERT and VALUES)
	insertPos := strings.Index(queryLower, "insert into")
	if insertPos == -1 {
		return query
	}

	// Extract table name and column list
	betweenInsertAndValues := query[insertPos+11 : valuesPos]
	parts := strings.SplitN(strings.TrimSpace(betweenInsertAndValues), "(", 2)
	if len(parts) != 2 {
		return query
	}

	tableName := strings.TrimSpace(parts[0])
	columnList := strings.TrimSuffix(strings.TrimSpace(parts[1]), ")")

	// Add tenant column to column list
	newColumnList := columnList + ", " + db.tenantColumn

	// Find the value placeholders
	afterValues := query[valuesPos+6:]
	afterValues = strings.TrimSpace(afterValues)
	if !strings.HasPrefix(afterValues, "(") {
		return query
	}

	// Count placeholders
	placeholderEnd := strings.Index(afterValues, ")")
	if placeholderEnd == -1 {
		return query
	}

	valuePlaceholders := afterValues[1:placeholderEnd]
	newValuePlaceholders := valuePlaceholders + ", ?"

	// Reconstruct query
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)%s",
		tableName,
		newColumnList,
		newValuePlaceholders,
		afterValues[placeholderEnd+1:])
}

// NamedExecContext executes a named query with tenant enforcement
func (db *DB) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	if db.shouldSkip(ctx, query) {
		return db.DB.NamedExecContext(ctx, query, arg)
	}

	// For named queries, users should include tenant_id in their structs/maps
	// We validate that tenant context exists but don't inject automatically
	_, err := db.getTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("tenant enforcement failed: %w", err)
	}

	return db.DB.NamedExecContext(ctx, query, arg)
}

// BeginTxx starts a transaction with tenant context
func (db *DB) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTxx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &Tx{
		Tx:           tx,
		db:           db,
		ctx:          ctx,
		tenantColumn: db.tenantColumn,
		skipTables:   db.skipTables,
	}, nil
}

// WithoutTenant returns a new DB instance that skips tenant enforcement
func (db *DB) WithoutTenant() *DB {
	newDB := *db
	newDB.skipTables = map[string]bool{"*": true}
	return &newDB
}

// SkipTenant adds a skip_tenant flag to the context
func SkipTenant(ctx context.Context) context.Context {
	return context.WithValue(ctx, SkipTenantKey, true)
}

// WithTenant creates a new context with the specified tenant ID
func WithTenant(ctx context.Context, tenantID string) (context.Context, error) {
	tc, err := domain.NewContext(tenantID, "system", "sqlx-adapter")
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant context: %w", err)
	}
	return tc.ToGoContext(ctx), nil
}
