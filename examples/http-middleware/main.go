// Gin middleware example demonstrating HTTP tenant extraction with TenantKit.
//
// Run: go run main.go
// Test: curl -H "X-Tenant-ID: acme-corp" http://localhost:8080/users
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/abhipray-cpu/tenantkit/tenantkit"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Set up database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, tenant_id TEXT, name TEXT)`)
	db.Exec(`INSERT INTO users (tenant_id, name) VALUES ('acme-corp', 'Alice'), ('acme-corp', 'Bob'), ('globex', 'Charlie')`)

	wrappedDB, err := tenantkit.WrapWithStyle(db, tenantkit.Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	}, tenantkit.PlaceholderQuestion)
	if err != nil {
		log.Fatal(err)
	}

	// Simple middleware that extracts tenant from X-Tenant-ID header
	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := r.Header.Get("X-Tenant-ID")
			if tenantID == "" {
				http.Error(w, "missing X-Tenant-ID header", http.StatusBadRequest)
				return
			}
			ctx := tenantkit.WithTenant(r.Context(), tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Handler that queries tenant-scoped data
	usersHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rows, err := wrappedDB.Query(r.Context(), "SELECT name FROM users ORDER BY name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		w.Header().Set("Content-Type", "text/plain")
		for rows.Next() {
			var name string
			rows.Scan(&name)
			w.Write([]byte(name + "\n"))
		}
	})

	// Admin handler — bypasses tenant filtering
	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := tenantkit.WithoutTenantFiltering(context.Background())
		var count int
		wrappedDB.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Total users: " + string(rune(count+'0')) + "\n"))
	})

	mux := http.NewServeMux()
	mux.Handle("/users", tenantMiddleware(usersHandler))
	mux.Handle("/admin/count", adminHandler)

	log.Println("Server starting on :8080")
	log.Println("Try: curl -H 'X-Tenant-ID: acme-corp' http://localhost:8080/users")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
