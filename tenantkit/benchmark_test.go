package tenantkit

import (
	"context"
	"testing"
)

func BenchmarkQueryTransformation_NoCache(b *testing.B) {
	db := &DB{
		interceptor: mustInterceptor(Config{
			TenantTables: []string{"users", "orders"},
			TenantColumn: "tenant_id",
		}),
		placeholderStyle: PlaceholderDollar,
		queryCache:       NewQueryCache(0), // Disabled cache
	}

	ctx := WithTenant(context.Background(), "tenant-1")
	_ = ctx

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = db.injectTenantFilter("SELECT * FROM users WHERE name = $1", "tenant-1", []interface{}{"Alice"})
	}
}

func BenchmarkQueryTransformation_CacheHit(b *testing.B) {
	db := &DB{
		interceptor: mustInterceptor(Config{
			TenantTables: []string{"users", "orders"},
			TenantColumn: "tenant_id",
		}),
		placeholderStyle: PlaceholderDollar,
		queryCache:       NewQueryCache(1000),
	}

	// Prime the cache
	db.injectTenantFilter("SELECT * FROM users WHERE name = $1", "tenant-1", []interface{}{"Alice"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = db.injectTenantFilter("SELECT * FROM users WHERE name = $1", "tenant-1", []interface{}{"Alice"})
	}
}

func BenchmarkQueryTransformation_QuestionMark(b *testing.B) {
	db := &DB{
		interceptor: mustInterceptor(Config{
			TenantTables: []string{"users", "orders"},
			TenantColumn: "tenant_id",
		}),
		placeholderStyle: PlaceholderQuestion,
		queryCache:       NewQueryCache(1000),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = db.injectTenantFilter("SELECT * FROM users WHERE name = ?", "tenant-1", []interface{}{"Alice"})
	}
}

func BenchmarkQueryTransformation_Insert(b *testing.B) {
	db := &DB{
		interceptor: mustInterceptor(Config{
			TenantTables: []string{"users"},
			TenantColumn: "tenant_id",
		}),
		placeholderStyle: PlaceholderDollar,
		queryCache:       NewQueryCache(1000),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = db.injectTenantFilter("INSERT INTO users (name, email) VALUES ($1, $2)", "tenant-1", []interface{}{"Alice", "alice@test.com"})
	}
}

func BenchmarkQueryTransformation_JoinQuery(b *testing.B) {
	db := &DB{
		interceptor: mustInterceptor(Config{
			TenantTables: []string{"users", "orders"},
			TenantColumn: "tenant_id",
		}),
		placeholderStyle: PlaceholderDollar,
		queryCache:       NewQueryCache(1000),
	}

	query := "SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id WHERE u.name = $1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = db.injectTenantFilter(query, "tenant-1", []interface{}{"Alice"})
	}
}

func BenchmarkContextWithTenant(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WithTenant(ctx, "tenant-1")
	}
}

func BenchmarkContextGetTenant(b *testing.B) {
	ctx := WithTenant(context.Background(), "tenant-1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetTenant(ctx)
	}
}

func BenchmarkQueryCache_Get(b *testing.B) {
	cache := NewQueryCache(1000)
	cache.Put("SELECT * FROM users", "SELECT * FROM users WHERE tenant_id = $1", 1, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("SELECT * FROM users")
	}
}

func BenchmarkQueryCache_Put(b *testing.B) {
	cache := NewQueryCache(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put("SELECT * FROM users", "SELECT * FROM users WHERE tenant_id = $1", 1, true)
	}
}

func BenchmarkInterceptorDecide(b *testing.B) {
	interceptor := mustInterceptor(Config{
		TenantTables: []string{"users", "orders", "products", "invoices"},
		TenantColumn: "tenant_id",
	})
	ctx := WithTenant(context.Background(), "tenant-1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		interceptor.ShouldFilter(ctx, "SELECT * FROM users WHERE id = $1")
	}
}

func BenchmarkInterceptorDecide_SystemQuery(b *testing.B) {
	interceptor := mustInterceptor(Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	})
	ctx := WithTenant(context.Background(), "tenant-1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		interceptor.ShouldFilter(ctx, "SELECT 1")
	}
}

// mustInterceptor creates an interceptor or panics (for benchmarks only)
func mustInterceptor(config Config) *Interceptor {
	i, err := NewInterceptor(config)
	if err != nil {
		panic(err)
	}
	return i
}
