package tenantkit

import (
	"testing"
)

// TestSystemQueryDetector_DDL verifies DDL queries are detected
func TestSystemQueryDetector_DDL(t *testing.T) {
	detector := NewSystemQueryDetector()

	ddlQueries := []string{
		"ALTER TABLE users ADD COLUMN email VARCHAR(255)",
		"CREATE TABLE new_table (id INT)",
		"DROP TABLE old_table",
		"TRUNCATE TABLE logs",
		"CREATE INDEX idx_users_email ON users(email)",
		"DROP INDEX idx_users_email",
		"CREATE UNIQUE INDEX idx_unique ON users(email)",
		"REINDEX TABLE users",
	}

	for _, query := range ddlQueries {
		t.Run(query, func(t *testing.T) {
			if !detector.IsSystemQuery(query) {
				t.Errorf("Expected DDL query to be detected as system query: %s", query)
			}
		})
	}
}

// TestSystemQueryDetector_DCL verifies DCL queries are detected
func TestSystemQueryDetector_DCL(t *testing.T) {
	detector := NewSystemQueryDetector()

	dclQueries := []string{
		"GRANT SELECT ON users TO readonly_user",
		"REVOKE INSERT ON orders FROM app_user",
		"GRANT ALL PRIVILEGES ON database TO admin",
	}

	for _, query := range dclQueries {
		t.Run(query, func(t *testing.T) {
			if !detector.IsSystemQuery(query) {
				t.Errorf("Expected DCL query to be detected as system query: %s", query)
			}
		})
	}
}

// TestSystemQueryDetector_HealthChecks verifies health check queries are detected
func TestSystemQueryDetector_HealthChecks(t *testing.T) {
	detector := NewSystemQueryDetector()

	healthQueries := []string{
		"SELECT 1",
		"select 1",     // Case insensitive
		"  SELECT 1  ", // With whitespace
		"SELECT NOW()",
		"SELECT VERSION()",
		"SELECT CURRENT_TIMESTAMP",
	}

	for _, query := range healthQueries {
		t.Run(query, func(t *testing.T) {
			if !detector.IsSystemQuery(query) {
				t.Errorf("Expected health check to be detected: %s", query)
			}
		})
	}
}

// TestSystemQueryDetector_SystemSchemas verifies system schema queries are detected
func TestSystemQueryDetector_SystemSchemas(t *testing.T) {
	detector := NewSystemQueryDetector()

	schemaQueries := []string{
		"SELECT * FROM information_schema.tables",
		"SELECT * FROM pg_catalog.pg_tables",
		"SELECT * FROM mysql.user",
		"SELECT * FROM sys.databases",
		"SELECT * FROM performance_schema.events",
		"SHOW TABLES",
		"SHOW DATABASES",
		"DESCRIBE users",
		"EXPLAIN SELECT * FROM users",
	}

	for _, query := range schemaQueries {
		t.Run(query, func(t *testing.T) {
			if !detector.IsSystemQuery(query) {
				t.Errorf("Expected system schema query to be detected: %s", query)
			}
		})
	}
}

// TestSystemQueryDetector_Maintenance verifies maintenance queries are detected
func TestSystemQueryDetector_Maintenance(t *testing.T) {
	detector := NewSystemQueryDetector()

	maintenanceQueries := []string{
		"VACUUM users",
		"VACUUM FULL users",
		"ANALYZE users",
		"ANALYZE TABLE users",
	}

	for _, query := range maintenanceQueries {
		t.Run(query, func(t *testing.T) {
			if !detector.IsSystemQuery(query) {
				t.Errorf("Expected maintenance query to be detected: %s", query)
			}
		})
	}
}

// TestSystemQueryDetector_SessionCommands verifies session commands are detected
func TestSystemQueryDetector_SessionCommands(t *testing.T) {
	detector := NewSystemQueryDetector()

	sessionQueries := []string{
		"SET timezone = 'UTC'",
		"SET NAMES utf8",
		"SET autocommit = 1",
	}

	for _, query := range sessionQueries {
		t.Run(query, func(t *testing.T) {
			if !detector.IsSystemQuery(query) {
				t.Errorf("Expected session command to be detected: %s", query)
			}
		})
	}
}

// TestSystemQueryDetector_NotSystemQueries verifies normal queries are NOT detected
func TestSystemQueryDetector_NotSystemQueries(t *testing.T) {
	detector := NewSystemQueryDetector()

	normalQueries := []string{
		"SELECT * FROM users",
		"SELECT * FROM users WHERE id = 1",
		"INSERT INTO orders (name) VALUES (?)",
		"UPDATE products SET price = ? WHERE id = ?",
		"DELETE FROM logs WHERE created_at < ?",
		"SELECT COUNT(*) FROM invoices",
		"SELECT users.*, orders.* FROM users JOIN orders ON users.id = orders.user_id",
	}

	for _, query := range normalQueries {
		t.Run(query, func(t *testing.T) {
			if detector.IsSystemQuery(query) {
				t.Errorf("Normal query should NOT be detected as system query: %s", query)
			}
		})
	}
}

// TestSystemQueryDetector_CaseInsensitive verifies detection is case-insensitive
func TestSystemQueryDetector_CaseInsensitive(t *testing.T) {
	detector := NewSystemQueryDetector()

	testCases := []struct {
		query    string
		expected bool
	}{
		{"ALTER TABLE users ADD col INT", true},
		{"alter table users add col int", true},
		{"AlTeR tAbLe users ADD col INT", true},
		{"SELECT 1", true},
		{"select 1", true},
		{"SeLeCt 1", true},
		{"SELECT * FROM users", false},
		{"select * from users", false},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			result := detector.IsSystemQuery(tc.query)
			if result != tc.expected {
				t.Errorf("Query %q: expected %v, got %v", tc.query, tc.expected, result)
			}
		})
	}
}

// TestSystemQueryDetector_WhitespaceHandling verifies whitespace is handled correctly
func TestSystemQueryDetector_WhitespaceHandling(t *testing.T) {
	detector := NewSystemQueryDetector()

	testCases := []string{
		"  SELECT 1  ",
		"\nSELECT 1\n",
		"\t\tALTER TABLE users\t\t",
		"   CREATE TABLE test   ",
	}

	for _, query := range testCases {
		t.Run(query, func(t *testing.T) {
			if !detector.IsSystemQuery(query) {
				t.Errorf("Query with whitespace should be detected: %q", query)
			}
		})
	}
}

// TestSystemQueryDetector_EdgeCases verifies edge cases are handled
func TestSystemQueryDetector_EdgeCases(t *testing.T) {
	detector := NewSystemQueryDetector()

	testCases := []struct {
		name     string
		query    string
		expected bool
		note     string
	}{
		{"Empty query", "", false, ""},
		{"Only whitespace", "   ", false, ""},
		{
			"SELECT with system schema in WHERE",
			"SELECT * FROM users WHERE schema = 'information_schema'",
			true,
			"Conservative: substring match catches this (acceptable false positive)",
		},
		{
			"Comment before DDL",
			"-- comment\nALTER TABLE users ADD col INT",
			false,
			"Comments before ALTER not handled (acceptable limitation)",
		},
		{"Multiline DDL", "ALTER TABLE users\nADD COLUMN email VARCHAR(255)", true, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.IsSystemQuery(tc.query)
			if result != tc.expected {
				if tc.note != "" {
					t.Logf("Note: %s", tc.note)
				}
				t.Errorf("Expected %v for %q, got %v", tc.expected, tc.query, result)
			}
		})
	}
}

// TestSystemQueryDetector_PostgreSQLSpecific verifies PostgreSQL-specific queries
func TestSystemQueryDetector_PostgreSQLSpecific(t *testing.T) {
	detector := NewSystemQueryDetector()

	pgQueries := []string{
		"SELECT * FROM pg_catalog.pg_stat_activity",
		"SELECT * FROM pg_tables",
		"REINDEX INDEX idx_users",
		"VACUUM ANALYZE users",
	}

	for _, query := range pgQueries {
		t.Run(query, func(t *testing.T) {
			if !detector.IsSystemQuery(query) {
				t.Errorf("Expected PostgreSQL system query to be detected: %s", query)
			}
		})
	}
}

// TestSystemQueryDetector_MySQLSpecific verifies MySQL-specific queries
func TestSystemQueryDetector_MySQLSpecific(t *testing.T) {
	detector := NewSystemQueryDetector()

	mysqlQueries := []string{
		"SELECT * FROM mysql.user",
		"SELECT * FROM mysql.db",
		"SHOW TABLES FROM database",
		"SHOW DATABASES",
		"DESCRIBE users",
		"SHOW CREATE TABLE users",
	}

	for _, query := range mysqlQueries {
		t.Run(query, func(t *testing.T) {
			if !detector.IsSystemQuery(query) {
				t.Errorf("Expected MySQL system query to be detected: %s", query)
			}
		})
	}
}
