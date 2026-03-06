package tenantkit

import (
	"reflect"
	"sort"
	"testing"
)

// TestTableExtractor_SimpleQueries tests basic single-table queries
func TestTableExtractor_SimpleQueries(t *testing.T) {
	extractor := NewTableExtractor()

	testCases := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "Simple SELECT",
			query:    "SELECT * FROM users",
			expected: []string{"users"},
		},
		{
			name:     "Simple INSERT",
			query:    "INSERT INTO products (name, price) VALUES (?, ?)",
			expected: []string{"products"},
		},
		{
			name:     "Simple UPDATE",
			query:    "UPDATE orders SET status = ? WHERE id = ?",
			expected: []string{"orders"},
		},
		{
			name:     "Simple DELETE",
			query:    "DELETE FROM logs WHERE created_at < ?",
			expected: []string{"logs"},
		},
		{
			name:     "SELECT with WHERE",
			query:    "SELECT id, name FROM customers WHERE active = true",
			expected: []string{"customers"},
		},
		{
			name:     "INSERT with multiple columns",
			query:    "INSERT INTO invoices (tenant_id, amount, status) VALUES (?, ?, ?)",
			expected: []string{"invoices"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tables, err := extractor.ExtractTables(tc.query)
			if err != nil {
				t.Fatalf("ExtractTables error: %v", err)
			}

			sort.Strings(tables)
			sort.Strings(tc.expected)

			if !reflect.DeepEqual(tables, tc.expected) {
				t.Errorf("Expected tables %v, got %v", tc.expected, tables)
			}
		})
	}
}

// TestTableExtractor_JOINs tests queries with multiple tables
func TestTableExtractor_JOINs(t *testing.T) {
	extractor := NewTableExtractor()

	testCases := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "INNER JOIN",
			query:    "SELECT * FROM users INNER JOIN orders ON users.id = orders.user_id",
			expected: []string{"users", "orders"},
		},
		{
			name:     "LEFT JOIN",
			query:    "SELECT u.*, o.* FROM users u LEFT JOIN orders o ON u.id = o.user_id",
			expected: []string{"users", "orders"},
		},
		{
			name:     "Multiple JOINs",
			query:    "SELECT * FROM users u JOIN orders o ON u.id = o.user_id JOIN products p ON o.product_id = p.id",
			expected: []string{"users", "orders", "products"},
		},
		{
			name:     "RIGHT JOIN",
			query:    "SELECT * FROM orders RIGHT JOIN users ON orders.user_id = users.id",
			expected: []string{"orders", "users"},
		},
		{
			name:     "Complex multi-table",
			query:    "SELECT * FROM users JOIN orders ON users.id = orders.user_id JOIN invoices ON orders.id = invoices.order_id",
			expected: []string{"users", "orders", "invoices"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tables, err := extractor.ExtractTables(tc.query)
			if err != nil {
				t.Fatalf("ExtractTables error: %v", err)
			}

			sort.Strings(tables)
			sort.Strings(tc.expected)

			if !reflect.DeepEqual(tables, tc.expected) {
				t.Errorf("Expected tables %v, got %v", tc.expected, tables)
			}
		})
	}
}

// TestTableExtractor_Aliases tests table aliases
func TestTableExtractor_Aliases(t *testing.T) {
	extractor := NewTableExtractor()

	testCases := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "Single alias",
			query:    "SELECT u.* FROM users u",
			expected: []string{"users"},
		},
		{
			name:     "Multiple aliases",
			query:    "SELECT u.*, o.* FROM users u, orders o WHERE u.id = o.user_id",
			expected: []string{"users", "orders"},
		},
		{
			name:     "AS keyword",
			query:    "SELECT * FROM users AS u JOIN orders AS o ON u.id = o.user_id",
			expected: []string{"users", "orders"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tables, err := extractor.ExtractTables(tc.query)
			if err != nil {
				t.Fatalf("ExtractTables error: %v", err)
			}

			sort.Strings(tables)
			sort.Strings(tc.expected)

			if !reflect.DeepEqual(tables, tc.expected) {
				t.Errorf("Expected tables %v, got %v", tc.expected, tables)
			}
		})
	}
}

// TestTableExtractor_SchemaQualified tests schema.table notation
func TestTableExtractor_SchemaQualified(t *testing.T) {
	extractor := NewTableExtractor()

	testCases := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "Simple schema.table",
			query:    "SELECT * FROM public.users",
			expected: []string{"users"},
		},
		{
			name:     "Multiple schemas",
			query:    "SELECT * FROM public.users JOIN archive.orders ON users.id = orders.user_id",
			expected: []string{"users", "orders"},
		},
		// Note: Backticks not supported in Phase 1 (regex-based)
		// Will be handled in Phase 2 with SQL parser integration
		// {
		// 	name:     "Schema with backticks",
		// 	query:    "SELECT * FROM `mydb`.`users`",
		// 	expected: []string{"users"},
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tables, err := extractor.ExtractTables(tc.query)
			if err != nil {
				t.Fatalf("ExtractTables error: %v", err)
			}

			sort.Strings(tables)
			sort.Strings(tc.expected)

			if !reflect.DeepEqual(tables, tc.expected) {
				t.Errorf("Expected tables %v, got %v", tc.expected, tables)
			}
		})
	}
}

// TestTableExtractor_CaseInsensitive tests case insensitivity
func TestTableExtractor_CaseInsensitive(t *testing.T) {
	extractor := NewTableExtractor()

	testCases := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "Uppercase SELECT",
			query:    "SELECT * FROM USERS",
			expected: []string{"users"},
		},
		{
			name:     "Mixed case",
			query:    "SeLeCt * FrOm UsErS",
			expected: []string{"users"},
		},
		{
			name:     "Uppercase JOIN",
			query:    "SELECT * FROM USERS JOIN ORDERS ON USERS.ID = ORDERS.USER_ID",
			expected: []string{"users", "orders"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tables, err := extractor.ExtractTables(tc.query)
			if err != nil {
				t.Fatalf("ExtractTables error: %v", err)
			}

			sort.Strings(tables)
			sort.Strings(tc.expected)

			if !reflect.DeepEqual(tables, tc.expected) {
				t.Errorf("Expected tables %v, got %v", tc.expected, tables)
			}
		})
	}
}

// TestTableExtractor_WhitespaceHandling tests various whitespace
func TestTableExtractor_WhitespaceHandling(t *testing.T) {
	extractor := NewTableExtractor()

	testCases := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "Extra spaces",
			query:    "SELECT  *  FROM   users",
			expected: []string{"users"},
		},
		{
			name:     "Tabs and newlines",
			query:    "SELECT *\nFROM\tusers\nWHERE active = true",
			expected: []string{"users"},
		},
		{
			name:     "Leading/trailing whitespace",
			query:    "  SELECT * FROM users  ",
			expected: []string{"users"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tables, err := extractor.ExtractTables(tc.query)
			if err != nil {
				t.Fatalf("ExtractTables error: %v", err)
			}

			sort.Strings(tables)
			sort.Strings(tc.expected)

			if !reflect.DeepEqual(tables, tc.expected) {
				t.Errorf("Expected tables %v, got %v", tc.expected, tables)
			}
		})
	}
}

// TestTableExtractor_EdgeCases tests edge cases and error handling
func TestTableExtractor_EdgeCases(t *testing.T) {
	extractor := NewTableExtractor()

	testCases := []struct {
		name        string
		query       string
		expected    []string
		shouldError bool
		note        string
	}{
		{
			name:        "Empty query",
			query:       "",
			expected:    nil,
			shouldError: false,
			note:        "Empty query returns empty list",
		},
		{
			name:        "Whitespace only",
			query:       "   \n\t  ",
			expected:    nil,
			shouldError: false,
			note:        "Whitespace-only returns empty list",
		},
		{
			name:     "Multiple spaces between keywords",
			query:    "SELECT     *     FROM     users",
			expected: []string{"users"},
			note:     "Handles multiple spaces gracefully",
		},
		{
			name:     "Comma-separated tables (old style)",
			query:    "SELECT * FROM users, orders WHERE users.id = orders.user_id",
			expected: []string{"users", "orders"},
			note:     "Handles comma-separated table lists",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tables, err := extractor.ExtractTables(tc.query)

			if tc.shouldError && err == nil {
				t.Fatalf("Expected error but got none")
			}

			if !tc.shouldError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !tc.shouldError {
				sort.Strings(tables)
				if tc.expected != nil {
					sort.Strings(tc.expected)
				}

				if !reflect.DeepEqual(tables, tc.expected) {
					t.Errorf("Expected tables %v, got %v (note: %s)", tc.expected, tables, tc.note)
				}
			}
		})
	}
}

// TestTableExtractor_Duplicates tests deduplication
func TestTableExtractor_Duplicates(t *testing.T) {
	extractor := NewTableExtractor()

	query := "SELECT * FROM users u1 JOIN users u2 ON u1.manager_id = u2.id"
	tables, err := extractor.ExtractTables(query)
	if err != nil {
		t.Fatalf("ExtractTables error: %v", err)
	}

	// Should return only one "users" entry
	if len(tables) != 1 {
		t.Errorf("Expected 1 table (deduplicated), got %d: %v", len(tables), tables)
	}

	if tables[0] != "users" {
		t.Errorf("Expected 'users', got '%s'", tables[0])
	}
}
