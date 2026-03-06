package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/abhipray-cpu/tenantkit/domain"
	"github.com/jmoiron/sqlx"
)

// Tx wraps sqlx.Tx with tenant enforcement
type Tx struct {
	*sqlx.Tx
	db           *DB
	ctx          context.Context
	tenantColumn string
	skipTables   map[string]bool
}

// getTenantID extracts tenant ID from the transaction's context
func (tx *Tx) getTenantID() (string, error) {
	if tx.ctx == nil {
		return "", fmt.Errorf("context is nil")
	}

	tenantCtx, err := domain.FromGoContext(tx.ctx)
	if err != nil {
		return "", fmt.Errorf("failed to extract tenant context: %w", err)
	}

	return tenantCtx.TenantID().Value(), nil
}

// shouldSkip checks if tenant enforcement should be skipped for this operation
func (tx *Tx) shouldSkip(query string) bool {
	// Check if context has skip flag
	if skip, ok := tx.ctx.Value(SkipTenantKey).(bool); ok && skip {
		return true
	}

	// Check if query is for a skip table
	queryLower := strings.ToLower(query)
	for table := range tx.skipTables {
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
func (tx *Tx) injectTenantCondition(query string, tenantID string) (string, int) {
	_ = tenantID // Parameter kept for API consistency
	queryLower := strings.ToLower(query)
	tenantCondition := fmt.Sprintf("%s = ?", tx.tenantColumn)

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

// injectTenantIntoInsert adds tenant_id to INSERT statement.
// The tenantID parameter is kept for API consistency but not used directly -
// the caller appends it to args after modifying the query.
//
//nolint:revive,unused // tenantID parameter kept for API clarity
func (tx *Tx) injectTenantIntoInsert(query string, tenantID string) string {
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
	newColumnList := columnList + ", " + tx.tenantColumn

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

// QueryContext executes a query with tenant enforcement
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if tx.shouldSkip(query) {
		return tx.Tx.QueryContext(ctx, query, args...)
	}

	tenantID, err := tx.getTenantID()
	if err != nil {
		return nil, fmt.Errorf("tenant enforcement failed: %w", err)
	}

	query, pos := tx.injectTenantCondition(query, tenantID)
	newArgs := insertArgAtPosition(args, pos, tenantID)

	return tx.Tx.QueryContext(ctx, query, newArgs...)
}

// QueryxContext executes a query and returns an sqlx.Rows with tenant enforcement
func (tx *Tx) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	if tx.shouldSkip(query) {
		return tx.Tx.QueryxContext(ctx, query, args...)
	}

	tenantID, err := tx.getTenantID()
	if err != nil {
		return nil, fmt.Errorf("tenant enforcement failed: %w", err)
	}

	query, pos := tx.injectTenantCondition(query, tenantID)
	newArgs := insertArgAtPosition(args, pos, tenantID)

	return tx.Tx.QueryxContext(ctx, query, newArgs...)
}

// QueryRowxContext executes a query that returns at most one row with tenant enforcement
func (tx *Tx) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	if tx.shouldSkip(query) {
		return tx.Tx.QueryRowxContext(ctx, query, args...)
	}

	tenantID, err := tx.getTenantID()
	if err != nil {
		// sqlx.Row doesn't support errors before Scan
		return tx.Tx.QueryRowxContext(ctx, "SELECT 1 WHERE 1=0")
	}

	query, pos := tx.injectTenantCondition(query, tenantID)
	newArgs := insertArgAtPosition(args, pos, tenantID)

	return tx.Tx.QueryRowxContext(ctx, query, newArgs...)
}

// GetContext executes a query and scans the result into dest with tenant enforcement
func (tx *Tx) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	if tx.shouldSkip(query) {
		return tx.Tx.GetContext(ctx, dest, query, args...)
	}

	tenantID, err := tx.getTenantID()
	if err != nil {
		return fmt.Errorf("tenant enforcement failed: %w", err)
	}

	query, pos := tx.injectTenantCondition(query, tenantID)
	newArgs := insertArgAtPosition(args, pos, tenantID)

	return tx.Tx.GetContext(ctx, dest, query, newArgs...)
}

// SelectContext executes a query and scans the results into dest with tenant enforcement
func (tx *Tx) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	if tx.shouldSkip(query) {
		return tx.Tx.SelectContext(ctx, dest, query, args...)
	}

	tenantID, err := tx.getTenantID()
	if err != nil {
		return fmt.Errorf("tenant enforcement failed: %w", err)
	}

	query, pos := tx.injectTenantCondition(query, tenantID)
	newArgs := insertArgAtPosition(args, pos, tenantID)

	return tx.Tx.SelectContext(ctx, dest, query, newArgs...)
}

// ExecContext executes a query with tenant enforcement
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if tx.shouldSkip(query) {
		return tx.Tx.ExecContext(ctx, query, args...)
	}

	tenantID, err := tx.getTenantID()
	if err != nil {
		return nil, fmt.Errorf("tenant enforcement failed: %w", err)
	}

	// For INSERT queries, we need to add tenant_id to the column list
	queryLower := strings.ToLower(query)
	if strings.HasPrefix(strings.TrimSpace(queryLower), "insert") {
		query = tx.injectTenantIntoInsert(query, tenantID)
		// Append tenant_id to args
		args = append(args, tenantID)
		return tx.Tx.ExecContext(ctx, query, args...)
	}

	// For UPDATE/DELETE queries, add WHERE condition
	query, pos := tx.injectTenantCondition(query, tenantID)
	newArgs := insertArgAtPosition(args, pos, tenantID)

	return tx.Tx.ExecContext(ctx, query, newArgs...)
}

// NamedExecContext executes a named query with tenant enforcement
func (tx *Tx) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	if tx.shouldSkip(query) {
		return tx.Tx.NamedExecContext(ctx, query, arg)
	}

	// Validate tenant context exists
	_, err := tx.getTenantID()
	if err != nil {
		return nil, fmt.Errorf("tenant enforcement failed: %w", err)
	}

	return tx.Tx.NamedExecContext(ctx, query, arg)
}
