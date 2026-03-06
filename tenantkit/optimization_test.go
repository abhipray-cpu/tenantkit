package tenantkit

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestQueryCache_BasicOperations(t *testing.T) {
	cache := NewQueryCache(10)

	// Test cache miss
	_, _, _, found := cache.Get("SELECT * FROM users")
	if found {
		t.Error("Expected cache miss for new query")
	}

	// Test cache put and get
	cache.Put("SELECT * FROM users", "SELECT * FROM users WHERE tenant_id = $1", 1, true)
	transformed, argCount, isTenantQuery, found := cache.Get("SELECT * FROM users")
	if !found {
		t.Error("Expected cache hit after Put")
	}
	if transformed != "SELECT * FROM users WHERE tenant_id = $1" {
		t.Errorf("Got wrong transformed query: %s", transformed)
	}
	if argCount != 1 {
		t.Errorf("Got wrong arg count: %d", argCount)
	}
	if !isTenantQuery {
		t.Error("Expected tenant query flag to be true")
	}

	// Test stats
	hits, misses, size, hitRate := cache.Stats()
	if hits != 1 {
		t.Errorf("Expected 1 hit, got %d", hits)
	}
	if misses != 1 {
		t.Errorf("Expected 1 miss, got %d", misses)
	}
	if size != 1 {
		t.Errorf("Expected cache size 1, got %d", size)
	}
	if hitRate != 50.0 {
		t.Errorf("Expected 50%% hit rate, got %.2f%%", hitRate)
	}
}

func TestQueryCache_Eviction(t *testing.T) {
	cache := NewQueryCache(5)

	// Fill cache to capacity
	for i := 0; i < 5; i++ {
		query := "SELECT * FROM users" + string(rune('A'+i))
		cache.Put(query, query+" transformed", i, true)
	}

	hits, misses, size, _ := cache.Stats()
	if size != 5 {
		t.Errorf("Expected cache size 5, got %d", size)
	}

	// Add one more - should trigger eviction
	cache.Put("SELECT * FROM orders", "SELECT * FROM orders transformed", 6, true)

	_, _, size, _ = cache.Stats()
	if size > 5 {
		t.Errorf("Cache size %d exceeds max size 5", size)
	}

	// Clear and verify
	cache.Clear()
	hits, misses, size, _ = cache.Stats()
	if hits != 0 || misses != 0 || size != 0 {
		t.Errorf("Cache not properly cleared: hits=%d, misses=%d, size=%d", hits, misses, size)
	}
}

func TestQueryCache_Concurrency(t *testing.T) {
	cache := NewQueryCache(100)

	done := make(chan bool)

	// Start 10 concurrent goroutines doing gets and puts
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				query := "SELECT * FROM users WHERE id = " + string(rune('A'+id))

				// Try get
				cache.Get(query)

				// Put
				cache.Put(query, query+" transformed", id, true)

				// Get again
				cache.Get(query)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Just verify no panics occurred and cache has entries
	_, _, size, _ := cache.Stats()
	if size == 0 {
		t.Error("Expected cache to have entries after concurrent operations")
	}
}

func TestMemoryOptimization_StringBuilderPool(t *testing.T) {
	// Get a string builder from pool
	sb1 := getStringBuilder()
	if sb1 == nil {
		t.Fatal("getStringBuilder returned nil")
	}
	if sb1.Len() != 0 {
		t.Error("Expected empty string builder from pool")
	}

	// Use it
	sb1.WriteString("test")
	if sb1.String() != "test" {
		t.Error("String builder not working correctly")
	}

	// Return to pool
	putStringBuilder(sb1)

	// Get another - might be the same one
	sb2 := getStringBuilder()
	if sb2.Len() != 0 {
		t.Error("String builder not properly reset when returned from pool")
	}

	// Test with large builder (should not be pooled)
	sbLarge := getStringBuilder()
	for i := 0; i < 5000; i++ {
		sbLarge.WriteByte('x')
	}
	putStringBuilder(sbLarge) // This should not actually pool it

	sb3 := getStringBuilder()
	if sb3.Len() != 0 {
		t.Error("Got non-empty builder from pool")
	}
	putStringBuilder(sb3)
}

func TestMemoryOptimization_ArgsSlicePool(t *testing.T) {
	// Get a slice from pool
	slice1 := getArgsSlice()
	if slice1 == nil {
		t.Fatal("getArgsSlice returned nil")
	}
	if len(*slice1) != 0 {
		t.Error("Expected empty slice from pool")
	}

	// Use it
	*slice1 = append(*slice1, "arg1", "arg2")
	if len(*slice1) != 2 {
		t.Error("Slice append not working")
	}

	// Return to pool
	putArgsSlice(slice1)

	// Get another - might be the same one
	slice2 := getArgsSlice()
	if len(*slice2) != 0 {
		t.Error("Slice not properly reset when returned from pool")
	}

	// Test with large slice (should not be pooled)
	sliceLarge := getArgsSlice()
	for i := 0; i < 100; i++ {
		*sliceLarge = append(*sliceLarge, i)
	}
	putArgsSlice(sliceLarge) // Should not actually pool it

	slice3 := getArgsSlice()
	if len(*slice3) != 0 {
		t.Error("Got non-empty slice from pool")
	}
	putArgsSlice(slice3)
}

func BenchmarkQueryCache_Hit(b *testing.B) {
	cache := NewQueryCache(1000)
	query := "SELECT * FROM users WHERE id = $1"
	cache.Put(query, query+" transformed", 1, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(query)
	}
}

func BenchmarkQueryCache_Miss(b *testing.B) {
	cache := NewQueryCache(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("SELECT * FROM users WHERE id = " + string(rune(i%100)))
	}
}

func BenchmarkStringBuilder_WithPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb := getStringBuilder()
		sb.WriteString("SELECT * FROM users WHERE ")
		sb.WriteString("tenant_id = $1 AND ")
		sb.WriteString("id = $2")
		_ = sb.String()
		putStringBuilder(sb)
	}
}

func BenchmarkStringBuilder_WithoutPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb := &strings.Builder{}
		sb.WriteString("SELECT * FROM users WHERE ")
		sb.WriteString("tenant_id = $1 AND ")
		sb.WriteString("id = $2")
		_ = sb.String()
	}
}

func TestDB_CacheMethods(t *testing.T) {
	db, sqlDB := setupTestDB(t)
	defer sqlDB.Close()

	// Test ClearQueryCache
	db.ClearQueryCache()

	// Do some queries to populate cache
	ctx := WithTenant(context.Background(), "test-tenant")
	db.Query(ctx, "SELECT * FROM users WHERE id = $1", 1)
	db.Query(ctx, "SELECT * FROM orders WHERE user_id = $1", 1)

	// Check stats
	hits, misses, size, hitRate := db.QueryCacheStats()
	t.Logf("Cache stats: hits=%d, misses=%d, size=%d, hitRate=%.2f%%", hits, misses, size, hitRate)

	if size == 0 {
		t.Error("Expected cache to have entries after queries")
	}

	// Clear and verify
	db.ClearQueryCache()
	_, _, size, _ = db.QueryCacheStats()
	if size != 0 {
		t.Errorf("Expected empty cache after clear, got size %d", size)
	}
}

func setupTestDB(t *testing.T) (*DB, *sql.DB) {
	dsn := "postgres://tenantkit:tenantkit_secret@localhost:5432/tenantkit?sslmode=disable"
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skip("Skipping test: cannot connect to database:", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		t.Skip("Skipping test: database not available:", err)
	}

	db, err := Wrap(sqlDB, Config{
		TenantTables: []string{"users", "orders", "products"},
		TenantColumn: "tenant_id",
	})
	if err != nil {
		sqlDB.Close()
		t.Fatalf("Failed to wrap database: %v", err)
	}

	// Setup test tables
	_, _ = sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			name TEXT
		)
	`)
	_, _ = sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			user_id INTEGER
		)
	`)

	return db, sqlDB
}

func BenchmarkDB_WithCache_SELECT(b *testing.B) {
	db, sqlDB := setupBenchDB(b)
	defer sqlDB.Close()

	ctx := WithTenant(context.Background(), "bench-tenant")
	query := "SELECT * FROM users WHERE id = $1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Query(ctx, query, i%100)
	}
}

func BenchmarkDB_WithoutCache_SELECT(b *testing.B) {
	// This would require a version without cache, so we'll just clear cache each time
	db, sqlDB := setupBenchDB(b)
	defer sqlDB.Close()

	ctx := WithTenant(context.Background(), "bench-tenant")
	query := "SELECT * FROM users WHERE id = $1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.ClearQueryCache() // Force cache miss
		db.Query(ctx, query, i%100)
	}
}

func setupBenchDB(b *testing.B) (*DB, *sql.DB) {
	dsn := "postgres://tenantkit:tenantkit_secret@localhost:5432/tenantkit?sslmode=disable"
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		b.Skip("Skipping benchmark: cannot connect to database:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		b.Skip("Skipping benchmark: database not available:", err)
	}

	db, err := Wrap(sqlDB, Config{
		TenantTables: []string{"users", "orders"},
		TenantColumn: "tenant_id",
	})
	if err != nil {
		sqlDB.Close()
		b.Fatalf("Failed to wrap database: %v", err)
	}

	return db, sqlDB
}
