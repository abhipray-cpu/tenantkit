//go:build integration

package sqlx

import (
	"context"
	"testing"

	"github.com/abhipray-cpu/tenantkit/domain"
	_ "github.com/mattn/go-sqlite3"
)

type TestUser struct {
	ID       int    `db:"id"`
	Name     string `db:"name"`
	Email    string `db:"email"`
	TenantID string `db:"tenant_id"`
}

func setupTestDB(t *testing.T) *DB {
	tdb, err := Connect("sqlite3", ":memory:", nil)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Create test table
	schema := `
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		tenant_id TEXT NOT NULL
	);
	CREATE INDEX idx_users_tenant_id ON users(tenant_id);
	`

	if _, err := tdb.DB.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return tdb
}

func createTenantContext(t *testing.T, tenantID string) context.Context {
	tc, err := domain.NewContext(tenantID, "user1", "req1")
	if err != nil {
		t.Fatalf("failed to create tenant context: %v", err)
	}
	return tc.ToGoContext(context.Background())
}

func TestInteg_BasicCRUD(t *testing.T) {
	tdb := setupTestDB(t)
	defer tdb.Close()

	ctx1 := createTenantContext(t, "t1")
	ctx2 := createTenantContext(t, "t2")

	// Insert users for tenant1
	_, err := tdb.ExecContext(ctx1, "INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "alice@t1.com")
	if err != nil {
		t.Fatalf("failed to insert user for t1: %v", err)
	}

	// Insert users for tenant2
	_, err = tdb.ExecContext(ctx2, "INSERT INTO users (name, email) VALUES (?, ?)", "Bob", "bob@t2.com")
	if err != nil {
		t.Fatalf("failed to insert user for t2: %v", err)
	}

	// Query tenant1 - should only see Alice
	var users []TestUser
	err = tdb.SelectContext(ctx1, &users, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("failed to select users for t1: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("expected 1 user for t1, got %d", len(users))
	}
	if len(users) > 0 && users[0].Name != "Alice" {
		t.Errorf("expected Alice, got %s", users[0].Name)
	}

	// Query tenant2 - should only see Bob
	users = []TestUser{}
	err = tdb.SelectContext(ctx2, &users, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("failed to select users for t2: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("expected 1 user for t2, got %d", len(users))
	}
	if len(users) > 0 && users[0].Name != "Bob" {
		t.Errorf("expected Bob, got %s", users[0].Name)
	}

	// Update tenant1 user
	_, err = tdb.ExecContext(ctx1, "UPDATE users SET email = ? WHERE name = ?", "alice.new@t1.com", "Alice")
	if err != nil {
		t.Fatalf("failed to update user for t1: %v", err)
	}

	// Verify update
	var user TestUser
	err = tdb.GetContext(ctx1, &user, "SELECT * FROM users WHERE name = ?", "Alice")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if user.Email != "alice.new@t1.com" {
		t.Errorf("expected alice.new@t1.com, got %s", user.Email)
	}

	// Delete tenant1 user
	_, err = tdb.ExecContext(ctx1, "DELETE FROM users WHERE name = ?", "Alice")
	if err != nil {
		t.Fatalf("failed to delete user for t1: %v", err)
	}

	// Verify deletion
	users = []TestUser{}
	err = tdb.SelectContext(ctx1, &users, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("failed to select users for t1: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users for t1, got %d", len(users))
	}

	// Verify tenant2 user still exists
	users = []TestUser{}
	err = tdb.SelectContext(ctx2, &users, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("failed to select users for t2: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user for t2, got %d", len(users))
	}
}

func TestInteg_Transactions(t *testing.T) {
	tdb := setupTestDB(t)
	defer tdb.Close()

	ctx := createTenantContext(t, "t1")

	// Test commit
	tx, err := tdb.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "alice@t1.com")
	if err != nil {
		t.Fatalf("failed to insert in transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Verify commit
	var users []TestUser
	err = tdb.SelectContext(ctx, &users, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("failed to select users: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user after commit, got %d", len(users))
	}

	// Test rollback
	tx, err = tdb.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", "Bob", "bob@t1.com")
	if err != nil {
		t.Fatalf("failed to insert in transaction: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}

	// Verify rollback
	users = []TestUser{}
	err = tdb.SelectContext(ctx, &users, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("failed to select users: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user after rollback (Bob should not exist), got %d", len(users))
	}
}

func TestInteg_SkipTables(t *testing.T) {
	cfg := &Config{
		SkipTables: []string{"migrations"},
	}
	tdb, err := Connect("sqlite3", ":memory:", cfg)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer tdb.Close()

	// Create migrations table
	schema := `
	CREATE TABLE migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);
	`
	if _, err := tdb.DB.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	ctx := createTenantContext(t, "t1")

	// Insert into migrations table - should work without tenant_id column
	_, err = tdb.ExecContext(ctx, "INSERT INTO migrations (name) VALUES (?)", "001_init")
	if err != nil {
		t.Fatalf("failed to insert into migrations: %v", err)
	}

	// Query migrations table - should work without tenant filtering
	var count int
	err = tdb.DB.GetContext(ctx, &count, "SELECT COUNT(*) FROM migrations")
	if err != nil {
		t.Fatalf("failed to count migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 migration, got %d", count)
	}
}

func TestInteg_SkipTenantContext(t *testing.T) {
	tdb := setupTestDB(t)
	defer tdb.Close()

	ctx1 := createTenantContext(t, "t1")
	ctx2 := createTenantContext(t, "t2")

	// Insert users for both tenants
	_, err := tdb.ExecContext(ctx1, "INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "alice@t1.com")
	if err != nil {
		t.Fatalf("failed to insert user for t1: %v", err)
	}

	_, err = tdb.ExecContext(ctx2, "INSERT INTO users (name, email) VALUES (?, ?)", "Bob", "bob@t2.com")
	if err != nil {
		t.Fatalf("failed to insert user for t2: %v", err)
	}

	// Query with SkipTenant - should see all users
	skipCtx := SkipTenant(ctx1)
	var users []TestUser
	err = tdb.SelectContext(skipCtx, &users, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("failed to select users with skip: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users with skip, got %d", len(users))
	}
}

func TestInteg_WithoutTenant(t *testing.T) {
	tdb := setupTestDB(t)
	defer tdb.Close()

	ctx1 := createTenantContext(t, "t1")
	ctx2 := createTenantContext(t, "t2")

	// Insert users for both tenants
	_, err := tdb.ExecContext(ctx1, "INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "alice@t1.com")
	if err != nil {
		t.Fatalf("failed to insert user for t1: %v", err)
	}

	_, err = tdb.ExecContext(ctx2, "INSERT INTO users (name, email) VALUES (?, ?)", "Bob", "bob@t2.com")
	if err != nil {
		t.Fatalf("failed to insert user for t2: %v", err)
	}

	// Query with WithoutTenant - should see all users
	noTenantDB := tdb.WithoutTenant()
	var users []TestUser
	err = noTenantDB.SelectContext(ctx1, &users, "SELECT * FROM users")
	if err != nil {
		t.Fatalf("failed to select users without tenant: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users without tenant enforcement, got %d", len(users))
	}
}

func TestInteg_Errors(t *testing.T) {
	tdb := setupTestDB(t)
	defer tdb.Close()

	// Query without tenant context - should fail
	_, err := tdb.ExecContext(context.Background(), "INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "alice@test.com")
	if err == nil {
		t.Error("expected error when querying without tenant context")
	}

	// Select without tenant context - should fail
	var users []TestUser
	err = tdb.SelectContext(context.Background(), &users, "SELECT * FROM users")
	if err == nil {
		t.Error("expected error when selecting without tenant context")
	}

	// Get without tenant context - should fail
	var user TestUser
	err = tdb.GetContext(context.Background(), &user, "SELECT * FROM users WHERE id = ?", 1)
	if err == nil {
		t.Error("expected error when getting without tenant context")
	}
}
