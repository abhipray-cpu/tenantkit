// Basic example demonstrating TenantKit with SQLite.
//
// Run: go run main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/abhipray-cpu/tenantkit/tenantkit"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 1. Open a regular database connection
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 2. Create a multi-tenant table
	_, err = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_id TEXT NOT NULL,
		name TEXT NOT NULL,
		email TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Wrap with TenantKit
	wrappedDB, err := tenantkit.WrapWithStyle(db, tenantkit.Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	}, tenantkit.PlaceholderQuestion)
	if err != nil {
		log.Fatal(err)
	}

	// 4. Insert data for different tenants
	ctx1 := tenantkit.WithTenant(context.Background(), "acme-corp")
	ctx2 := tenantkit.WithTenant(context.Background(), "globex-inc")

	wrappedDB.Exec(ctx1, "INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "alice@acme.com")
	wrappedDB.Exec(ctx1, "INSERT INTO users (name, email) VALUES (?, ?)", "Bob", "bob@acme.com")
	wrappedDB.Exec(ctx2, "INSERT INTO users (name, email) VALUES (?, ?)", "Charlie", "charlie@globex.com")

	// 5. Query as tenant 1 — only sees their own data
	fmt.Println("=== Acme Corp users ===")
	rows, err := wrappedDB.Query(ctx1, "SELECT name, email FROM users ORDER BY name")
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var name, email string
		rows.Scan(&name, &email)
		fmt.Printf("  %s <%s>\n", name, email)
	}
	rows.Close()

	// 6. Query as tenant 2 — isolated from tenant 1
	fmt.Println("=== Globex Inc users ===")
	rows, err = wrappedDB.Query(ctx2, "SELECT name, email FROM users ORDER BY name")
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var name, email string
		rows.Scan(&name, &email)
		fmt.Printf("  %s <%s>\n", name, email)
	}
	rows.Close()

	// 7. Admin query — bypass filtering to see all data
	fmt.Println("=== All users (admin) ===")
	adminCtx := tenantkit.WithoutTenantFiltering(context.Background())
	var count int
	wrappedDB.QueryRow(adminCtx, "SELECT COUNT(*) FROM users").Scan(&count)
	fmt.Printf("  Total: %d users\n", count)
}
