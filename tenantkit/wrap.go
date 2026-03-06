package tenantkit

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// PlaceholderStyle defines the SQL placeholder style for different databases
type PlaceholderStyle int

const (
	// PlaceholderQuestion uses ? placeholders (MySQL, SQLite)
	PlaceholderQuestion PlaceholderStyle = iota
	// PlaceholderDollar uses $1, $2, ... placeholders (PostgreSQL)
	PlaceholderDollar
	// PlaceholderColon uses :1, :2, ... placeholders (Oracle)
	PlaceholderColon
)

// DB wraps a *sql.DB with automatic tenant filtering
type DB struct {
	db               *sql.DB
	interceptor      *Interceptor
	config           Config
	placeholderStyle PlaceholderStyle
	queryCache       *QueryCache // Query transformation cache
	logger           *slog.Logger
}

// placeholderRegex matches $N style placeholders
var placeholderRegex = regexp.MustCompile(`\$(\d+)`)

// Wrap wraps a *sql.DB with tenant-aware query filtering
//
// Example:
//
//	db := tenantkit.Wrap(sqlDB, tenantkit.Config{
//	    TenantTables: []string{"users", "orders", "products"},
//	    TenantColumn: "tenant_id",
//	})
func Wrap(sqlDB *sql.DB, config Config) (*DB, error) {
	if sqlDB == nil {
		return nil, fmt.Errorf("tenantkit: database cannot be nil")
	}

	interceptor, err := NewInterceptor(config)
	if err != nil {
		return nil, err
	}

	// Detect placeholder style from driver
	placeholderStyle := detectPlaceholderStyle()

	// Set up logger — only log if explicitly configured
	logger := config.Logger

	return &DB{
		db:               sqlDB,
		interceptor:      interceptor,
		config:           config,
		placeholderStyle: placeholderStyle,
		queryCache:       NewQueryCache(1000), // Cache up to 1000 query patterns
		logger:           logger,
	}, nil
}

// detectPlaceholderStyle returns the default placeholder style for SQL queries.
// Defaults to PostgreSQL style ($1) as it's more common in Go.
// Users can override via WrapWithStyle if needed.
func detectPlaceholderStyle() PlaceholderStyle {
	// Try to detect from driver name via a simple ping
	// Default to PostgreSQL style ($1) as it's more common in Go
	// The driver info isn't directly accessible, so we default to PostgreSQL
	// Users can override via WrapWithStyle if needed
	return PlaceholderDollar
}

// WrapWithStyle wraps a *sql.DB with explicit placeholder style
func WrapWithStyle(sqlDB *sql.DB, config Config, style PlaceholderStyle) (*DB, error) {
	db, err := Wrap(sqlDB, config)
	if err != nil {
		return nil, err
	}
	db.placeholderStyle = style
	return db, nil
}

// MustWrap is like Wrap but panics on error
// Useful for initialization code where configuration errors should be fatal
func MustWrap(sqlDB *sql.DB, config Config) *DB {
	db, err := Wrap(sqlDB, config)
	if err != nil {
		panic(err)
	}
	return db
}

// Query executes a query with automatic tenant filtering
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	// Get decision from interceptor
	decision, err := db.interceptor.ShouldFilter(ctx, query)
	if err != nil {
		return nil, err
	}

	// If filtering required, rewrite query
	if decision.RequiresFiltering {
		originalQuery := query
		query, args, err = db.injectTenantFilter(query, decision.TenantID, args)
		if err != nil {
			return nil, fmt.Errorf("tenantkit: failed to inject tenant filter: %w", err)
		}
		// Debug logging
		if db.logger != nil {
			db.logger.Debug("tenantkit: query rewritten",
				"original", originalQuery,
				"modified", query,
				"tenant", decision.TenantID,
			)
		}
	}

	// Execute query
	return db.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row with automatic tenant filtering
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// Get decision from interceptor
	decision, err := db.interceptor.ShouldFilter(ctx, query)
	if err != nil {
		// Return a row that will error on Scan
		return &sql.Row{}
	}

	// If filtering required, rewrite query
	if decision.RequiresFiltering {
		query, args, err = db.injectTenantFilter(query, decision.TenantID, args)
		if err != nil {
			// Return a row that will error on Scan
			return &sql.Row{}
		}
	}

	// Execute query
	return db.db.QueryRowContext(ctx, query, args...)
}

// Exec executes a query without returning any rows with automatic tenant filtering
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// Get decision from interceptor
	decision, err := db.interceptor.ShouldFilter(ctx, query)
	if err != nil {
		return nil, err
	}

	// If filtering required, rewrite query
	if decision.RequiresFiltering {
		query, args, err = db.injectTenantFilter(query, decision.TenantID, args)
		if err != nil {
			return nil, fmt.Errorf("tenantkit: failed to inject tenant filter: %w", err)
		}
	}

	// Execute query
	return db.db.ExecContext(ctx, query, args...)
}

// Close closes the underlying database connection
func (db *DB) Close() error {
	return db.db.Close()
}

// Raw returns the underlying *sql.DB
// Use with caution - bypasses all tenant filtering
func (db *DB) Raw() *sql.DB {
	return db.db
}

// Tx wraps a *sql.Tx with automatic tenant filtering.
// All queries executed through a Tx are tenant-scoped.
type Tx struct {
	tx               *sql.Tx
	interceptor      *Interceptor
	config           Config
	placeholderStyle PlaceholderStyle
	db               *DB // reference to parent DB for helper methods
}

// Begin starts a new tenant-scoped transaction.
// All queries within the transaction are automatically filtered by tenant.
//
// Example:
//
//	ctx := tenantkit.WithTenant(ctx, "acme-corp")
//	tx, err := db.Begin(ctx, nil)
//	if err != nil { ... }
//	defer tx.Rollback()
//
//	tx.Exec(ctx, "INSERT INTO orders (product) VALUES ($1)", "widget")
//	tx.Exec(ctx, "UPDATE inventory SET count = count - 1 WHERE product = $1", "widget")
//	err = tx.Commit()
func (db *DB) Begin(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("tenantkit: failed to begin transaction: %w", err)
	}

	return &Tx{
		tx:               tx,
		interceptor:      db.interceptor,
		config:           db.config,
		placeholderStyle: db.placeholderStyle,
		db:               db,
	}, nil
}

// Query executes a query within the transaction with automatic tenant filtering
func (tx *Tx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	decision, err := tx.interceptor.ShouldFilter(ctx, query)
	if err != nil {
		return nil, err
	}

	if decision.RequiresFiltering {
		query, args, err = tx.db.injectTenantFilter(query, decision.TenantID, args)
		if err != nil {
			return nil, fmt.Errorf("tenantkit: failed to inject tenant filter: %w", err)
		}
	}

	return tx.tx.QueryContext(ctx, query, args...)
}

// QueryRow executes a query within the transaction that returns at most one row
func (tx *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	decision, err := tx.interceptor.ShouldFilter(ctx, query)
	if err != nil {
		return &sql.Row{}
	}

	if decision.RequiresFiltering {
		query, args, err = tx.db.injectTenantFilter(query, decision.TenantID, args)
		if err != nil {
			return &sql.Row{}
		}
	}

	return tx.tx.QueryRowContext(ctx, query, args...)
}

// Exec executes a query within the transaction without returning any rows
func (tx *Tx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	decision, err := tx.interceptor.ShouldFilter(ctx, query)
	if err != nil {
		return nil, err
	}

	if decision.RequiresFiltering {
		query, args, err = tx.db.injectTenantFilter(query, decision.TenantID, args)
		if err != nil {
			return nil, fmt.Errorf("tenantkit: failed to inject tenant filter: %w", err)
		}
	}

	return tx.tx.ExecContext(ctx, query, args...)
}

// Commit commits the transaction
func (tx *Tx) Commit() error {
	return tx.tx.Commit()
}

// Rollback rolls back the transaction
func (tx *Tx) Rollback() error {
	return tx.tx.Rollback()
}

// injectTenantFilter injects tenant filtering into a SQL query.
// Uses the query cache to avoid re-parsing identical query patterns.
func (db *DB) injectTenantFilter(query string, tenantID string, args []interface{}) (string, []interface{}, error) {
	// Normalize query
	query = strings.TrimSpace(query)

	// Try cache first — the cached template stores the transformed query pattern
	if transformed, argCount, isTenantQuery, found := db.queryCache.Get(query); found && isTenantQuery {
		// Cache hit: use the pre-computed template and inject the tenant args
		newArgs := make([]interface{}, 0, len(args)+argCount)

		if db.placeholderStyle == PlaceholderQuestion {
			// For ? style, tenant placeholders are positional. Their position in
			// the rewritten query depends on the query type:
			//   SELECT/DELETE: tenant ? comes first → prepend tenant args
			//   UPDATE: tenant ? comes after SET ? marks → insert in the middle
			upper := strings.ToUpper(query)
			if strings.HasPrefix(upper, "UPDATE") {
				// Count ? in original query before WHERE to find SET arg count
				wherePos := strings.Index(upper, " WHERE ")
				setArgCount := 0
				if wherePos != -1 {
					setArgCount = strings.Count(query[:wherePos], "?")
				} else {
					setArgCount = strings.Count(query, "?")
				}
				newArgs = append(newArgs, args[:setArgCount]...)
				for i := 0; i < argCount; i++ {
					newArgs = append(newArgs, tenantID)
				}
				newArgs = append(newArgs, args[setArgCount:]...)
			} else {
				// SELECT, DELETE: tenant args come first
				for i := 0; i < argCount; i++ {
					newArgs = append(newArgs, tenantID)
				}
				newArgs = append(newArgs, args...)
			}
		} else {
			// For $N style, order doesn't matter
			newArgs = append(newArgs, args...)
			for i := 0; i < argCount; i++ {
				newArgs = append(newArgs, tenantID)
			}
		}
		return transformed, newArgs, nil
	}

	upper := strings.ToUpper(query)

	// Count existing placeholders to know where to start numbering
	existingPlaceholders := db.countPlaceholders(query)

	// Determine query type and inject filter accordingly
	var result string
	var newArgs []interface{}
	var err error

	switch {
	case strings.HasPrefix(upper, "SELECT"):
		result, newArgs, err = db.injectSelectFilter(query, tenantID, args, existingPlaceholders)
	case strings.HasPrefix(upper, "UPDATE"):
		result, newArgs, err = db.injectUpdateFilter(query, tenantID, args, existingPlaceholders)
	case strings.HasPrefix(upper, "DELETE"):
		result, newArgs, err = db.injectDeleteFilter(query, tenantID, args, existingPlaceholders)
	case strings.HasPrefix(upper, "INSERT"):
		result, newArgs, err = db.injectInsertFilter(query, tenantID, args)
	default:
		return query, args, nil
	}

	if err != nil {
		return "", nil, err
	}

	// Cache the transformation: store the tenant arg count (newArgs - args)
	tenantArgCount := len(newArgs) - len(args)
	db.queryCache.Put(query, result, tenantArgCount, true)

	return result, newArgs, err
}

// countPlaceholders counts the number of placeholders in a query
func (db *DB) countPlaceholders(query string) int {
	switch db.placeholderStyle {
	case PlaceholderDollar:
		// Find highest $N placeholder
		matches := placeholderRegex.FindAllStringSubmatch(query, -1)
		maxN := 0
		for _, match := range matches {
			if len(match) > 1 {
				var n int
				fmt.Sscanf(match[1], "%d", &n)
				if n > maxN {
					maxN = n
				}
			}
		}
		return maxN
	case PlaceholderQuestion:
		return strings.Count(query, "?")
	default:
		return 0
	}
}

// placeholder returns the appropriate placeholder for the given position
func (db *DB) placeholder(position int) string {
	switch db.placeholderStyle {
	case PlaceholderDollar:
		return fmt.Sprintf("$%d", position)
	case PlaceholderColon:
		return fmt.Sprintf(":%d", position)
	default:
		return "?"
	}
}

// injectSelectFilter injects tenant filter into SELECT queries
func (db *DB) injectSelectFilter(query string, tenantID string, args []interface{}, existingPlaceholders int) (string, []interface{}, error) {
	upper := strings.ToUpper(query)

	// Extract ALL tables from the query (main + JOINs)
	allTables := db.extractAllTables(query)

	// Filter to only tenant tables that we should add filtering for
	tablesToFilter := db.filterTenantTables(allTables)

	// If no tables to filter, return original query
	if len(tablesToFilter) == 0 {
		return query, args, nil
	}

	// SECURITY: Always add tenant filter even if tenant_id already present in WHERE
	// This prevents malicious queries like "WHERE tenant_id = 'other-tenant'" from bypassing isolation

	// Build the tenant filter conditions for all tables
	// Use string builder pool to reduce allocations
	sb := getStringBuilder()
	defer putStringBuilder(sb)

	// Pre-allocate args slice with known capacity
	argsPtr := getArgsSlice()
	defer putArgsSlice(argsPtr)
	newArgs := *argsPtr

	// For ? style placeholders, tenant args must be prepended because the
	// tenant condition is injected BEFORE the original WHERE conditions.
	// For $N style, order doesn't matter since placeholders are numbered.
	var tenantArgs []interface{}

	nextPlaceholder := existingPlaceholders + 1

	for i, table := range tablesToFilter {
		if i > 0 {
			sb.WriteString(" AND ")
		}

		// Build qualified column name
		if table.Alias != "" {
			sb.WriteString(table.Alias)
		} else {
			sb.WriteString(table.Name)
		}
		sb.WriteByte('.')
		sb.WriteString(db.config.TenantColumn)
		sb.WriteString(" = ")
		sb.WriteString(db.placeholder(nextPlaceholder))

		tenantArgs = append(tenantArgs, tenantID)
		nextPlaceholder++
	}

	if db.placeholderStyle == PlaceholderQuestion {
		// ? placeholders are positional — tenant filter appears first in the query
		newArgs = append(newArgs, tenantArgs...)
		newArgs = append(newArgs, args...)
	} else {
		// $N placeholders are numbered — order doesn't matter
		newArgs = append(newArgs, args...)
		newArgs = append(newArgs, tenantArgs...)
	}

	tenantFilter := sb.String()

	// Keywords that mark the end of WHERE conditions
	endKeywords := []string{" ORDER BY ", " GROUP BY ", " LIMIT ", " HAVING ", " UNION ", " EXCEPT ", " INTERSECT ", " FOR UPDATE", " FOR SHARE"}

	// Find WHERE clause position
	wherePos := strings.Index(upper, " WHERE ")

	if wherePos != -1 {
		// Has WHERE clause - need to find where conditions end
		afterWhere := query[wherePos+7:]
		afterWhereUpper := upper[wherePos+7:]

		// Find the end of WHERE conditions
		whereEndPos := len(afterWhere)
		for _, keyword := range endKeywords {
			if pos := strings.Index(afterWhereUpper, keyword); pos != -1 && pos < whereEndPos {
				whereEndPos = pos
			}
		}

		// Split into: before WHERE, WHERE conditions, after (ORDER BY, LIMIT, etc.)
		before := query[:wherePos+7] // Include " WHERE "
		conditions := afterWhere[:whereEndPos]
		after := afterWhere[whereEndPos:]

		// Reuse string builder for final query
		sb.Reset()
		sb.WriteString(before)
		sb.WriteString(tenantFilter)
		sb.WriteString(" AND (")
		sb.WriteString(strings.TrimSpace(conditions))
		sb.WriteByte(')')
		sb.WriteString(after)

		return sb.String(), newArgs, nil
	}

	// No WHERE clause - find insertion point
	insertPos := len(query)

	for _, keyword := range endKeywords {
		if pos := strings.Index(upper, keyword); pos != -1 && pos < insertPos {
			insertPos = pos
		}
	}

	// Inject WHERE clause
	before := strings.TrimSpace(query[:insertPos])
	after := query[insertPos:]

	// Reuse string builder for final query
	sb.Reset()
	sb.WriteString(before)
	sb.WriteString(" WHERE ")
	sb.WriteString(tenantFilter)
	sb.WriteString(after)

	return sb.String(), newArgs, nil
}

// TableRef represents a table reference in a query
type TableRef struct {
	Name  string // Original table name
	Alias string // Alias if present (empty if no alias)
}

// extractAllTables extracts all table references from a query (FROM + JOINs)
func (db *DB) extractAllTables(query string) []TableRef {
	var tables []TableRef
	upper := strings.ToUpper(query)

	// Find FROM clause
	fromPos := strings.Index(upper, " FROM ")
	if fromPos == -1 {
		return tables
	}

	// Find where the table references end (before WHERE)
	wherePos := strings.Index(upper, " WHERE ")
	endPos := len(query)
	if wherePos != -1 {
		endPos = wherePos
	}

	// Also check for other end markers
	endMarkers := []string{" ORDER BY ", " GROUP BY ", " LIMIT ", " HAVING ", " UNION "}
	for _, marker := range endMarkers {
		if pos := strings.Index(upper[fromPos:], marker); pos != -1 {
			markerPos := fromPos + pos
			if markerPos < endPos {
				endPos = markerPos
			}
		}
	}

	// Extract the FROM...JOIN section
	fromSection := query[fromPos+6 : endPos]
	fromSectionUpper := upper[fromPos+6 : endPos]

	// Split by JOIN keywords to get individual table references
	joinKeywords := []string{" LEFT OUTER JOIN ", " RIGHT OUTER JOIN ", " FULL OUTER JOIN ", " LEFT JOIN ", " RIGHT JOIN ", " INNER JOIN ", " CROSS JOIN ", " NATURAL JOIN ", " JOIN "}

	// Start with the main table (before first JOIN)
	firstJoinPos := len(fromSection)
	for _, jk := range joinKeywords {
		if pos := strings.Index(fromSectionUpper, jk); pos != -1 && pos < firstJoinPos {
			firstJoinPos = pos
		}
	}

	// Extract main table
	mainTableRef := strings.TrimSpace(fromSection[:firstJoinPos])
	if mainTable := db.parseTableRef(mainTableRef); mainTable.Name != "" {
		tables = append(tables, mainTable)
	}

	// Extract joined tables
	remaining := fromSection
	remainingUpper := fromSectionUpper

	for {
		// Find next JOIN
		nextJoinKeyword := ""
		nextJoinPos := len(remaining)

		for _, jk := range joinKeywords {
			if pos := strings.Index(remainingUpper, jk); pos != -1 && pos < nextJoinPos {
				nextJoinPos = pos
				nextJoinKeyword = jk
			}
		}

		if nextJoinKeyword == "" {
			break // No more JOINs
		}

		// Move past the JOIN keyword
		afterJoin := remaining[nextJoinPos+len(nextJoinKeyword):]
		afterJoinUpper := remainingUpper[nextJoinPos+len(nextJoinKeyword):]

		// Find where this table reference ends (at ON clause or next JOIN)
		onPos := strings.Index(afterJoinUpper, " ON ")
		tableEndPos := len(afterJoin)
		if onPos != -1 {
			tableEndPos = onPos
		}

		// Also check for next JOIN
		for _, jk := range joinKeywords {
			if pos := strings.Index(afterJoinUpper, jk); pos != -1 && pos < tableEndPos {
				tableEndPos = pos
			}
		}

		// Extract this joined table
		joinedTableRef := strings.TrimSpace(afterJoin[:tableEndPos])
		if joinedTable := db.parseTableRef(joinedTableRef); joinedTable.Name != "" {
			tables = append(tables, joinedTable)
		}

		// Move to remainder (after ON clause if present)
		if onPos != -1 {
			// Find end of ON condition (next JOIN or end of section)
			onConditionEnd := len(afterJoin)
			for _, jk := range joinKeywords {
				if pos := strings.Index(afterJoinUpper[onPos:], jk); pos != -1 && onPos+pos < onConditionEnd {
					onConditionEnd = onPos + pos
				}
			}
			remaining = afterJoin[onConditionEnd:]
			remainingUpper = afterJoinUpper[onConditionEnd:]
		} else {
			remaining = afterJoin[tableEndPos:]
			remainingUpper = afterJoinUpper[tableEndPos:]
		}
	}

	return tables
}

// parseTableRef parses a table reference string into TableRef
// Handles: "table", "table alias", "table AS alias", "schema.table alias"
func (db *DB) parseTableRef(ref string) TableRef {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return TableRef{}
	}

	// Remove any trailing comma
	ref = strings.TrimSuffix(ref, ",")

	// Split by whitespace
	parts := strings.Fields(ref)
	if len(parts) == 0 {
		return TableRef{}
	}

	tableName := parts[0]
	var alias string

	if len(parts) >= 2 {
		if strings.ToUpper(parts[1]) == "AS" && len(parts) >= 3 {
			alias = parts[2]
		} else {
			alias = parts[1]
		}
	}

	return TableRef{Name: tableName, Alias: alias}
}

// filterTenantTables filters the list of tables to only include tenant tables
func (db *DB) filterTenantTables(tables []TableRef) []TableRef {
	var result []TableRef

	for _, table := range tables {
		// Check if this table is in our tenant tables list
		tableName := strings.ToLower(table.Name)
		// Handle schema-qualified names (schema.table)
		if dotPos := strings.LastIndex(tableName, "."); dotPos != -1 {
			tableName = tableName[dotPos+1:]
		}

		for _, tenantTable := range db.config.TenantTables {
			if strings.ToLower(tenantTable) == tableName {
				result = append(result, table)
				break
			}
		}
	}

	return result
}

// injectUpdateFilter injects tenant filter into UPDATE queries
func (db *DB) injectUpdateFilter(query string, tenantID string, args []interface{}, existingPlaceholders int) (string, []interface{}, error) {
	upper := strings.ToUpper(query)

	// SECURITY: Always add tenant filter even if tenant_id already present in WHERE
	// This prevents malicious queries like "WHERE tenant_id = 'other-tenant'" from bypassing isolation

	nextPlaceholder := db.placeholder(existingPlaceholders + 1)

	// Use string builder pool
	sb := getStringBuilder()
	defer putStringBuilder(sb)

	// For ? style placeholders, the tenant condition is prepended in the WHERE
	// clause, so the tenant arg must be inserted between the SET args and the
	// original WHERE args. For $N style, order doesn't matter.
	buildArgs := func() []interface{} {
		argsPtr := getArgsSlice()
		defer putArgsSlice(argsPtr)
		out := *argsPtr

		if db.placeholderStyle == PlaceholderQuestion {
			// Count ? placeholders before WHERE to determine SET arg count
			wherePos := strings.Index(upper, " WHERE ")
			setArgCount := 0
			if wherePos != -1 {
				setArgCount = strings.Count(query[:wherePos], "?")
			} else {
				setArgCount = strings.Count(query, "?")
			}
			// SET args, then tenant, then WHERE args
			out = append(out, args[:setArgCount]...)
			out = append(out, tenantID)
			out = append(out, args[setArgCount:]...)
		} else {
			out = append(out, args...)
			out = append(out, tenantID)
		}
		return out
	}

	// Find WHERE clause position
	wherePos := strings.Index(upper, " WHERE ")

	// Keywords that mark the end of WHERE conditions for UPDATE
	endKeywords := []string{" RETURNING "}

	if wherePos != -1 {
		// Has WHERE clause - find where conditions end
		afterWhere := query[wherePos+7:]
		afterWhereUpper := upper[wherePos+7:]

		whereEndPos := len(afterWhere)
		for _, keyword := range endKeywords {
			if pos := strings.Index(afterWhereUpper, keyword); pos != -1 && pos < whereEndPos {
				whereEndPos = pos
			}
		}

		before := query[:wherePos+7]
		conditions := afterWhere[:whereEndPos]
		after := afterWhere[whereEndPos:]

		sb.WriteString(before)
		sb.WriteString(db.config.TenantColumn)
		sb.WriteString(" = ")
		sb.WriteString(nextPlaceholder)
		sb.WriteString(" AND (")
		sb.WriteString(strings.TrimSpace(conditions))
		sb.WriteByte(')')
		sb.WriteString(after)

		return sb.String(), buildArgs(), nil
	}

	// No WHERE clause - add one at the end (before RETURNING if present)
	returningPos := strings.Index(upper, " RETURNING ")
	if returningPos != -1 {
		before := query[:returningPos]
		after := query[returningPos:]

		sb.WriteString(before)
		sb.WriteString(" WHERE ")
		sb.WriteString(db.config.TenantColumn)
		sb.WriteString(" = ")
		sb.WriteString(nextPlaceholder)
		sb.WriteString(after)

		return sb.String(), buildArgs(), nil
	}

	sb.WriteString(query)
	sb.WriteString(" WHERE ")
	sb.WriteString(db.config.TenantColumn)
	sb.WriteString(" = ")
	sb.WriteString(nextPlaceholder)

	return sb.String(), buildArgs(), nil
}

// injectDeleteFilter injects tenant filter into DELETE queries
func (db *DB) injectDeleteFilter(query string, tenantID string, args []interface{}, existingPlaceholders int) (string, []interface{}, error) {
	upper := strings.ToUpper(query)

	// SECURITY: Always add tenant filter even if tenant_id already present in WHERE
	// This prevents malicious queries like "WHERE tenant_id = 'other-tenant'" from bypassing isolation

	nextPlaceholder := db.placeholder(existingPlaceholders + 1)

	// Use string builder pool
	sb := getStringBuilder()
	defer putStringBuilder(sb)

	// Use args pool
	argsPtr := getArgsSlice()
	defer putArgsSlice(argsPtr)
	newArgs := *argsPtr

	// For ? style placeholders, tenant arg must be prepended because the
	// tenant condition is injected BEFORE the original WHERE conditions.
	// For $N style, order doesn't matter since placeholders are numbered.
	if db.placeholderStyle == PlaceholderQuestion {
		newArgs = append(newArgs, tenantID)
		newArgs = append(newArgs, args...)
	} else {
		newArgs = append(newArgs, args...)
		newArgs = append(newArgs, tenantID)
	}

	// Find WHERE clause position
	wherePos := strings.Index(upper, " WHERE ")

	// Keywords that mark the end of WHERE conditions for DELETE
	endKeywords := []string{" RETURNING "}

	if wherePos != -1 {
		// Has WHERE clause - find where conditions end
		afterWhere := query[wherePos+7:]
		afterWhereUpper := upper[wherePos+7:]

		whereEndPos := len(afterWhere)
		for _, keyword := range endKeywords {
			if pos := strings.Index(afterWhereUpper, keyword); pos != -1 && pos < whereEndPos {
				whereEndPos = pos
			}
		}

		before := query[:wherePos+7]
		conditions := afterWhere[:whereEndPos]
		after := afterWhere[whereEndPos:]

		sb.WriteString(before)
		sb.WriteString(db.config.TenantColumn)
		sb.WriteString(" = ")
		sb.WriteString(nextPlaceholder)
		sb.WriteString(" AND (")
		sb.WriteString(strings.TrimSpace(conditions))
		sb.WriteByte(')')
		sb.WriteString(after)

		return sb.String(), newArgs, nil
	}

	// No WHERE clause - add one at the end (before RETURNING if present)
	returningPos := strings.Index(upper, " RETURNING ")
	if returningPos != -1 {
		before := query[:returningPos]
		after := query[returningPos:]

		sb.WriteString(before)
		sb.WriteString(" WHERE ")
		sb.WriteString(db.config.TenantColumn)
		sb.WriteString(" = ")
		sb.WriteString(nextPlaceholder)
		sb.WriteString(after)

		return sb.String(), newArgs, nil
	}

	sb.WriteString(query)
	sb.WriteString(" WHERE ")
	sb.WriteString(db.config.TenantColumn)
	sb.WriteString(" = ")
	sb.WriteString(nextPlaceholder)

	return sb.String(), newArgs, nil
}

// injectInsertFilter injects tenant value into INSERT queries
func (db *DB) injectInsertFilter(query string, tenantID string, args []interface{}) (string, []interface{}, error) {
	// For INSERT, we need to add tenant_id to the column list and values
	// This preserves the original query structure while adding tenant filtering

	upper := strings.ToUpper(query)

	// Pattern: INSERT INTO table (col1, col2) VALUES ($1, $2)
	insertPos := strings.Index(upper, "INSERT INTO ")
	if insertPos == -1 {
		return query, args, fmt.Errorf("invalid INSERT query")
	}

	// Find the table name and column list
	valuesPos := strings.Index(upper, " VALUES ")
	if valuesPos == -1 {
		// No explicit column list - we need the table structure
		// For now, return an error - user should use explicit columns
		return query, args, fmt.Errorf("INSERT queries must use explicit column lists for tenant filtering")
	}

	// Extract parts
	tablePart := query[insertPos+12 : valuesPos] // "table (col1, col2)"
	valuesPart := query[valuesPos+8:]            // "($1, $2)"

	// Find opening paren in table part
	parenPos := strings.Index(tablePart, "(")
	if parenPos == -1 {
		// No column list
		return query, args, fmt.Errorf("INSERT queries must use explicit column lists for tenant filtering")
	}

	tableName := strings.TrimSpace(tablePart[:parenPos])
	columnList := tablePart[parenPos:] // "(col1, col2)"

	// Check if tenant column is already in the column list
	// This allows users to explicitly include tenant_id in their queries
	columnListUpper := strings.ToUpper(columnList)
	tenantColumnUpper := strings.ToUpper(db.config.TenantColumn)

	// Check for tenant column in the list (must be whole word match)
	if db.columnExistsInList(columnListUpper, tenantColumnUpper) {
		// Tenant column already present - don't modify the query
		// The user is responsible for providing the correct tenant value
		return query, args, nil
	}

	// Inject tenant column at the beginning
	newColumnList := strings.Replace(columnList, "(", fmt.Sprintf("(%s, ", db.config.TenantColumn), 1)

	// For INSERT, we need to:
	// 1. Add tenant_id as $1
	// 2. Renumber all existing placeholders ($1 -> $2, $2 -> $3, etc.)

	// First, renumber existing placeholders in the values part
	newValuesPart := db.renumberPlaceholders(valuesPart, 1)

	// Then add tenant placeholder at the beginning
	valuesParenPos := strings.Index(newValuesPart, "(")
	if valuesParenPos == -1 {
		return query, args, fmt.Errorf("invalid INSERT VALUES clause")
	}

	tenantPlaceholder := db.placeholder(1)
	newValuesPart = strings.Replace(newValuesPart, "(", fmt.Sprintf("(%s, ", tenantPlaceholder), 1)

	// Handle RETURNING clause if present
	returningClause := ""
	returningPos := strings.Index(strings.ToUpper(newValuesPart), " RETURNING ")
	if returningPos != -1 {
		returningClause = newValuesPart[returningPos:]
		newValuesPart = newValuesPart[:returningPos]
	}

	// Handle ON CONFLICT clause if present
	onConflictClause := ""
	onConflictPos := strings.Index(strings.ToUpper(newValuesPart), " ON CONFLICT ")
	if onConflictPos != -1 {
		onConflictClause = newValuesPart[onConflictPos:]
		newValuesPart = newValuesPart[:onConflictPos]
	}

	// Construct new query
	newQuery := fmt.Sprintf("INSERT INTO %s %s VALUES %s%s%s", tableName, newColumnList, newValuesPart, onConflictClause, returningClause)

	// Prepend tenant ID to args
	newArgs := append([]interface{}{tenantID}, args...)

	return newQuery, newArgs, nil
}

// columnExistsInList checks if a column name exists in a column list string
// It handles proper word boundary matching to avoid false positives
func (db *DB) columnExistsInList(columnList, columnName string) bool {
	// Simple check: look for the column name surrounded by non-alphanumeric chars
	// This handles cases like "(tenant_id, other)" and "(other, tenant_id)"

	// Remove parentheses for simpler matching
	list := strings.Trim(columnList, "()")

	// Split by comma and check each column
	columns := strings.Split(list, ",")
	for _, col := range columns {
		col = strings.TrimSpace(col)
		if col == columnName {
			return true
		}
	}
	return false
}

// renumberPlaceholders shifts all $N placeholders by the given offset
func (db *DB) renumberPlaceholders(query string, offset int) string {
	if db.placeholderStyle != PlaceholderDollar {
		return query // Only needed for PostgreSQL-style placeholders
	}

	// Replace from highest to lowest to avoid double-replacement
	// First, find all placeholders and their positions
	matches := placeholderRegex.FindAllStringSubmatchIndex(query, -1)
	if len(matches) == 0 {
		return query
	}

	// Process in reverse order to avoid position shifts
	result := query
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		if len(match) >= 4 {
			numStart := match[2]
			numEnd := match[3]
			var n int
			fmt.Sscanf(query[numStart:numEnd], "%d", &n)
			newPlaceholder := fmt.Sprintf("$%d", n+offset)
			result = result[:match[0]] + newPlaceholder + result[match[1]:]
		}
	}

	return result
}

// ClearQueryCache clears the query transformation cache
// Useful for testing or when you want to force cache refresh
func (db *DB) ClearQueryCache() {
	if db.queryCache != nil {
		db.queryCache.Clear()
	}
}

// QueryCacheStats returns statistics about the query cache
// Returns hits, misses, current size, and hit rate percentage
func (db *DB) QueryCacheStats() (hits, misses uint64, size int, hitRate float64) {
	if db.queryCache != nil {
		return db.queryCache.Stats()
	}
	return 0, 0, 0, 0.0
}
