package tenantkit

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupSimpleDB creates an in-memory SQLite database for testing
func setupSimpleDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create test tables
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO users (id, tenant_id, name) VALUES
		(1, 'tenant-1', 'Alice'),
		(2, 'tenant-1', 'Bob'),
		(3, 'tenant-2', 'Charlie')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	return db
}

func TestWrapBasic(t *testing.T) {
	sqlDB := setupSimpleDB(t)
	defer sqlDB.Close()

	config := Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	}

	db, err := WrapWithStyle(sqlDB, config, PlaceholderQuestion)
	if err != nil {
		t.Fatalf("Wrap failed: %v", err)
	}

	// Test Query with tenant filtering
	ctx := WithTenant(context.Background(), "tenant-1")
	rows, err := db.Query(ctx, "SELECT name FROM users ORDER BY id")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		names = append(names, name)
	}

	// Should only return tenant-1's users
	expected := []string{"Alice", "Bob"}
	if len(names) != len(expected) {
		t.Errorf("Expected %d users, got %d", len(expected), len(names))
	}
	for i, name := range names {
		if i < len(expected) && name != expected[i] {
			t.Errorf("Expected user %d to be %s, got %s", i, expected[i], name)
		}
	}
}

func TestWrapWithWhere(t *testing.T) {
	sqlDB := setupSimpleDB(t)
	defer sqlDB.Close()

	config := Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	}

	db, err := WrapWithStyle(sqlDB, config, PlaceholderQuestion)
	if err != nil {
		t.Fatalf("Wrap failed: %v", err)
	}

	// Test Query with existing WHERE clause
	ctx := WithTenant(context.Background(), "tenant-1")
	rows, err := db.Query(ctx, "SELECT name FROM users WHERE name = ?", "Alice")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		count++
		if name != "Alice" {
			t.Errorf("Expected Alice, got %s", name)
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}
}

func TestWrapUpdate(t *testing.T) {
	sqlDB := setupSimpleDB(t)
	defer sqlDB.Close()

	config := Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	}

	db, err := WrapWithStyle(sqlDB, config, PlaceholderQuestion)
	if err != nil {
		t.Fatalf("Wrap failed: %v", err)
	}

	// Test UPDATE with WHERE clause
	ctx := WithTenant(context.Background(), "tenant-1")
	result, err := db.Exec(ctx, "UPDATE users SET name = ? WHERE name = ?", "NewAlice", "Alice")
	if err != nil {
		t.Fatalf("UPDATE failed: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected failed: %v", err)
	}

	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}
}

func TestWrapMissingTenant(t *testing.T) {
	sqlDB := setupSimpleDB(t)
	defer sqlDB.Close()

	config := Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	}

	db, err := WrapWithStyle(sqlDB, config, PlaceholderQuestion)
	if err != nil {
		t.Fatalf("Wrap failed: %v", err)
	}

	// Test Query without tenant context - should error
	ctx := context.Background() // No tenant
	_, err = db.Query(ctx, "SELECT name FROM users")
	if err == nil {
		t.Fatal("Expected error when querying without tenant context")
	}

	// Should contain "tenant" in error message
	if err != ErrMissingTenant {
		t.Logf("Got error: %v", err)
	}
}

func TestWrapBypass(t *testing.T) {
	sqlDB := setupSimpleDB(t)
	defer sqlDB.Close()

	config := Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	}

	db, err := WrapWithStyle(sqlDB, config, PlaceholderQuestion)
	if err != nil {
		t.Fatalf("Wrap failed: %v", err)
	}

	// Test Query with bypass context
	ctx := WithoutTenantFiltering(context.Background())
	rows, err := db.Query(ctx, "SELECT COUNT(*) FROM users")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
	}

	// Should return all 3 users
	if count != 3 {
		t.Errorf("Expected 3 users, got %d", count)
	}
}
