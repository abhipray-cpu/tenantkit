package tenantkit

import "strings"

// SystemQueryDetector detects DDL, DCL, health checks, and system queries
// that should bypass tenant filtering.
type SystemQueryDetector struct {
	// prefixes are SQL command prefixes that indicate system queries
	prefixes []string

	// exactMatch are complete queries that should be bypassed
	exactMatch []string

	// substrings are schema/catalog names that indicate system queries
	substrings []string
}

// NewSystemQueryDetector creates a new system query detector with default patterns.
// The detector identifies:
// - DDL: ALTER, CREATE, DROP, TRUNCATE, etc.
// - DCL: GRANT, REVOKE
// - Health checks: SELECT 1, SELECT NOW(), etc.
// - System schemas: information_schema, pg_catalog, mysql.*, sys.*, etc.
// - Maintenance: VACUUM, ANALYZE, REINDEX
// - Session commands: SET, SHOW, DESCRIBE, EXPLAIN
func NewSystemQueryDetector() *SystemQueryDetector {
	return &SystemQueryDetector{
		prefixes: []string{
			// DDL (Data Definition Language)
			"ALTER ",
			"CREATE ",
			"DROP ",
			"TRUNCATE ",

			// DCL (Data Control Language)
			"GRANT ",
			"REVOKE ",

			// Information/Metadata queries
			"SHOW ",
			"DESCRIBE ",
			"EXPLAIN ",

			// Session configuration
			"SET ",

			// Maintenance operations
			"VACUUM ",
			"ANALYZE ",
			"REINDEX ",
		},
		exactMatch: []string{
			// Health check patterns (exact matches for performance)
			"SELECT 1",
			"SELECT NOW()",
			"SELECT VERSION()",
			"SELECT CURRENT_TIMESTAMP",
		},
		substrings: []string{
			// PostgreSQL system catalogs
			"pg_catalog",
			"pg_tables",
			"pg_stat",

			// MySQL system schemas
			"mysql.",
			"mysql.user",
			"mysql.db",

			// ANSI SQL information schema
			"information_schema",

			// SQL Server system schemas
			"sys.",
			"sys.databases",

			// Performance schema
			"performance_schema",
		},
	}
}

// IsSystemQuery determines if a query should bypass tenant filtering.
// It checks:
// 1. Exact matches (fast path for health checks)
// 2. SQL command prefixes (DDL, DCL, maintenance, etc.)
// 3. System schema/catalog references
//
// The detection is case-insensitive and handles leading/trailing whitespace.
func (d *SystemQueryDetector) IsSystemQuery(query string) bool {
	if query == "" {
		return false
	}

	// Normalize: trim whitespace and convert to uppercase for comparison
	normalized := strings.TrimSpace(query)
	upper := strings.ToUpper(normalized)
	lower := strings.ToLower(normalized)

	// Fast path: Check exact matches (health checks)
	for _, exact := range d.exactMatch {
		if upper == exact {
			return true
		}
	}

	// Check command prefixes (DDL, DCL, maintenance, etc.)
	for _, prefix := range d.prefixes {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}

	// Check for system schema/catalog references
	for _, substr := range d.substrings {
		if strings.Contains(lower, substr) {
			return true
		}
	}

	return false
}
