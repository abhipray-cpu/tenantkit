package tenantkit_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/abhipray-cpu/tenantkit/tenantkit"
	_ "github.com/mattn/go-sqlite3"
)

func Example_basic() {
	// Open a database connection
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create a tenant-scoped table
	db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, tenant_id TEXT, name TEXT)")
	db.Exec("INSERT INTO users (tenant_id, name) VALUES ('acme', 'Alice'), ('acme', 'Bob'), ('other', 'Charlie')")

	// Wrap with TenantKit
	wrappedDB, err := tenantkit.WrapWithStyle(db, tenantkit.Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	}, tenantkit.PlaceholderQuestion)
	if err != nil {
		log.Fatal(err)
	}

	// Query with tenant context — only sees tenant's own data
	ctx := tenantkit.WithTenant(context.Background(), "acme")
	rows, err := wrappedDB.Query(ctx, "SELECT name FROM users ORDER BY name")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Println(name)
	}
	// Output:
	// Alice
	// Bob
}

func Example_bypass() {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()
	db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, tenant_id TEXT, name TEXT)")
	db.Exec("INSERT INTO users (tenant_id, name) VALUES ('a', 'Alice'), ('b', 'Bob')")

	wrappedDB, _ := tenantkit.WrapWithStyle(db, tenantkit.Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	}, tenantkit.PlaceholderQuestion)

	// Admin query — bypass tenant filtering
	ctx := tenantkit.WithoutTenantFiltering(context.Background())
	var count int
	wrappedDB.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	fmt.Println(count)
	// Output:
	// 2
}

func Example_transaction() {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()
	db.Exec("CREATE TABLE orders (id INTEGER PRIMARY KEY, tenant_id TEXT, product TEXT)")

	wrappedDB, _ := tenantkit.WrapWithStyle(db, tenantkit.Config{
		TenantTables: []string{"orders"},
		TenantColumn: "tenant_id",
	}, tenantkit.PlaceholderQuestion)

	ctx := tenantkit.WithTenant(context.Background(), "acme")

	// Begin a transaction — all operations are tenant-scoped
	tx, err := wrappedDB.Begin(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	tx.Exec(ctx, "INSERT INTO orders (product) VALUES (?)", "Widget")
	tx.Exec(ctx, "INSERT INTO orders (product) VALUES (?)", "Gadget")
	tx.Commit()

	// Verify
	rows, _ := wrappedDB.Query(ctx, "SELECT product FROM orders ORDER BY product")
	defer rows.Close()
	for rows.Next() {
		var product string
		rows.Scan(&product)
		fmt.Println(product)
	}
	// Output:
	// Gadget
	// Widget
}
