package tenantkit

import (
	"regexp"
	"strings"
)

// TableExtractor extracts table names from SQL queries
type TableExtractor struct {
	// Phase 1: Simple regex-based extraction
	// Phase 2: Will integrate SQL parser for complex cases
}

// NewTableExtractor creates a new table extractor
func NewTableExtractor() *TableExtractor {
	return &TableExtractor{}
}

// ExtractTables extracts all table names from a SQL query
// Returns normalized (lowercase) table names without duplicates
func (te *TableExtractor) ExtractTables(query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	// Normalize query: collapse whitespace, convert to lowercase for pattern matching
	normalized := strings.TrimSpace(query)
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	lower := strings.ToLower(normalized)

	tables := make(map[string]bool)

	// Pattern 1: FROM clause - matches "FROM table", "FROM schema.table", "FROM table alias"
	// Supports: FROM users, FROM public.users, FROM users u, FROM users AS u
	fromPattern := regexp.MustCompile(`\bfrom\s+(?:(\w+)\.)?(\w+)(?:\s+(?:as\s+)?(\w+))?`)
	fromMatches := fromPattern.FindAllStringSubmatch(lower, -1)
	for _, match := range fromMatches {
		// match[1] = schema (optional), match[2] = table name
		tableName := match[2]
		tables[tableName] = true
	}

	// Pattern 2: JOIN clauses - matches all JOIN types (INNER, LEFT, RIGHT, OUTER, CROSS)
	// Supports: JOIN orders, LEFT JOIN public.orders o, RIGHT JOIN orders AS o
	joinPattern := regexp.MustCompile(`\b(?:inner\s+|left\s+|right\s+|outer\s+|cross\s+)?join\s+(?:(\w+)\.)?(\w+)(?:\s+(?:as\s+)?(\w+))?`)
	joinMatches := joinPattern.FindAllStringSubmatch(lower, -1)
	for _, match := range joinMatches {
		// match[1] = schema (optional), match[2] = table name
		tableName := match[2]
		tables[tableName] = true
	}

	// Pattern 3: INSERT INTO
	// Supports: INSERT INTO users, INSERT INTO public.users
	insertPattern := regexp.MustCompile(`\binsert\s+into\s+(?:(\w+)\.)?(\w+)`)
	insertMatches := insertPattern.FindAllStringSubmatch(lower, -1)
	for _, match := range insertMatches {
		// match[1] = schema (optional), match[2] = table name
		tableName := match[2]
		tables[tableName] = true
	}

	// Pattern 4: UPDATE
	// Supports: UPDATE users, UPDATE public.users
	updatePattern := regexp.MustCompile(`\bupdate\s+(?:(\w+)\.)?(\w+)`)
	updateMatches := updatePattern.FindAllStringSubmatch(lower, -1)
	for _, match := range updateMatches {
		// match[1] = schema (optional), match[2] = table name
		tableName := match[2]
		tables[tableName] = true
	}

	// Pattern 5: DELETE FROM
	// Supports: DELETE FROM users, DELETE FROM public.users
	deletePattern := regexp.MustCompile(`\bdelete\s+from\s+(?:(\w+)\.)?(\w+)`)
	deleteMatches := deletePattern.FindAllStringSubmatch(lower, -1)
	for _, match := range deleteMatches {
		// match[1] = schema (optional), match[2] = table name
		tableName := match[2]
		tables[tableName] = true
	}

	// Pattern 6: Comma-separated tables (old-style joins)
	// Example: SELECT * FROM users, orders WHERE users.id = orders.user_id
	// Extract all table names after FROM and before WHERE/ORDER/GROUP/LIMIT
	commaPattern := regexp.MustCompile(`\bfrom\s+((?:\w+(?:\.\w+)?\s*(?:as\s+\w+)?,?\s*)+)(?:\s+where|\s+order|\s+group|\s+limit|$)`)
	commaMatches := commaPattern.FindStringSubmatch(lower)
	if len(commaMatches) > 1 {
		tableList := commaMatches[1]
		// Split by comma and extract table names
		parts := strings.Split(tableList, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			// Extract just the table name (before any alias)
			tablePattern := regexp.MustCompile(`^(?:(\w+)\.)?(\w+)`)
			tableMatch := tablePattern.FindStringSubmatch(part)
			if len(tableMatch) > 2 {
				tableName := tableMatch[2]
				tables[tableName] = true
			}
		}
	}

	// Convert map to slice (deduplicated)
	result := make([]string, 0, len(tables))
	for table := range tables {
		result = append(result, table)
	}

	return result, nil
}
