package tenantkit

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// TestInterceptor_SystemQueriesBypass tests Rule 1: System queries auto-bypass
func TestInterceptor_SystemQueriesBypass(t *testing.T) {
	config := Config{
		TenantTables: []string{"users", "orders"},
		TenantColumn: "tenant_id",
	}
	interceptor, _ := NewInterceptor(config)

	testCases := []struct {
		name  string
		query string
	}{
		{"DDL ALTER", "ALTER TABLE users ADD COLUMN email VARCHAR(255)"},
		{"DDL CREATE", "CREATE TABLE products (id INT, name VARCHAR(255))"},
		{"DDL DROP", "DROP TABLE old_logs"},
		{"Health check SELECT 1", "SELECT 1"},
		{"Health check NOW", "SELECT NOW()"},
		{"System schema", "SELECT * FROM information_schema.tables"},
		{"PostgreSQL catalog", "SELECT * FROM pg_catalog.pg_tables"},
		{"VACUUM", "VACUUM ANALYZE users"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background() // No tenant context

			decision, err := interceptor.ShouldFilter(ctx, tc.query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if decision.RequiresFiltering {
				t.Errorf("System query should bypass, but RequiresFiltering=true")
			}

			if decision.Reason != ReasonSystemQuery {
				t.Errorf("Expected reason %s, got %s", ReasonSystemQuery, decision.Reason)
			}
		})
	}
}

// TestInterceptor_ExplicitBypassContext tests explicit bypass via context
func TestInterceptor_ExplicitBypassContext(t *testing.T) {
	config := Config{
		TenantTables: []string{"users", "orders"},
		TenantColumn: "tenant_id",
	}
	interceptor, _ := NewInterceptor(config)

	testCases := []struct {
		name  string
		query string
	}{
		{"Tenant table with bypass", "SELECT * FROM users"},
		{"Multiple tenant tables", "SELECT * FROM users JOIN orders ON users.id = orders.user_id"},
		{"Cross-tenant analytics", "SELECT tenant_id, COUNT(*) FROM orders GROUP BY tenant_id"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create context with explicit bypass
			ctx := WithoutTenantFiltering(context.Background())

			decision, err := interceptor.ShouldFilter(ctx, tc.query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if decision.RequiresFiltering {
				t.Errorf("Explicit bypass should skip filtering, but RequiresFiltering=true")
			}

			if decision.Reason != ReasonExplicitBypass {
				t.Errorf("Expected reason %s, got %s", ReasonExplicitBypass, decision.Reason)
			}
		})
	}
}

// TestInterceptor_NonTenantTables tests non-tenant tables pass through
func TestInterceptor_NonTenantTables(t *testing.T) {
	config := Config{
		TenantTables: []string{"users", "orders"},
		TenantColumn: "tenant_id",
	}
	interceptor, _ := NewInterceptor(config)

	testCases := []struct {
		name   string
		query  string
		tables []string
	}{
		{"Reference table", "SELECT * FROM countries", []string{"countries"}},
		{"Configuration table", "SELECT * FROM app_settings", []string{"app_settings"}},
		{"Currency table", "SELECT * FROM currencies WHERE code = ?", []string{"currencies"}},
		{"Multiple non-tenant", "SELECT * FROM countries JOIN currencies ON countries.currency_code = currencies.code", []string{"countries", "currencies"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background() // No tenant context needed

			decision, err := interceptor.ShouldFilter(ctx, tc.query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if decision.RequiresFiltering {
				t.Errorf("Non-tenant tables should bypass, but RequiresFiltering=true")
			}

			if decision.Reason != ReasonNoTenantTables {
				t.Errorf("Expected reason %s, got %s", ReasonNoTenantTables, decision.Reason)
			}

			// Verify extracted tables
			sort.Strings(decision.ExtractedTables)
			sort.Strings(tc.tables)
			if !reflect.DeepEqual(decision.ExtractedTables, tc.tables) {
				t.Errorf("Expected tables %v, got %v", tc.tables, decision.ExtractedTables)
			}
		})
	}
}

// TestInterceptor_TenantTablesRequireContext tests tenant tables require tenant
func TestInterceptor_TenantTablesRequireContext(t *testing.T) {
	config := Config{
		TenantTables: []string{"users", "orders", "products"},
		TenantColumn: "tenant_id",
	}
	interceptor, _ := NewInterceptor(config)

	testCases := []struct {
		name  string
		query string
	}{
		{"Simple SELECT", "SELECT * FROM users"},
		{"SELECT with WHERE", "SELECT * FROM orders WHERE status = ?"},
		{"INSERT", "INSERT INTO products (name, price) VALUES (?, ?)"},
		{"UPDATE", "UPDATE users SET email = ? WHERE id = ?"},
		{"DELETE", "DELETE FROM orders WHERE created_at < ?"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Context WITH tenant
			ctx := WithTenant(context.Background(), "tenant-123")

			decision, err := interceptor.ShouldFilter(ctx, tc.query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !decision.RequiresFiltering {
				t.Errorf("Tenant table should require filtering, but RequiresFiltering=false")
			}

			if decision.TenantID != "tenant-123" {
				t.Errorf("Expected tenant_id 'tenant-123', got '%s'", decision.TenantID)
			}

			if decision.Reason != ReasonTenantTableAccess {
				t.Errorf("Expected reason %s, got %s", ReasonTenantTableAccess, decision.Reason)
			}
		})
	}
}

// TestInterceptor_MissingTenantError tests missing tenant context errors
func TestInterceptor_MissingTenantError(t *testing.T) {
	config := Config{
		TenantTables: []string{"users", "orders"},
		TenantColumn: "tenant_id",
	}
	interceptor, _ := NewInterceptor(config)

	testCases := []struct {
		name         string
		query        string
		expectedErr  error
		expectTables []string
	}{
		{
			name:         "SELECT from tenant table",
			query:        "SELECT * FROM users",
			expectedErr:  ErrMissingTenant,
			expectTables: []string{"users"},
		},
		{
			name:         "JOIN with tenant table",
			query:        "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
			expectedErr:  ErrMissingTenant,
			expectTables: []string{"users", "orders"},
		},
		{
			name:         "UPDATE tenant table",
			query:        "UPDATE orders SET status = ?",
			expectedErr:  ErrMissingTenant,
			expectTables: []string{"orders"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background() // NO tenant context

			decision, err := interceptor.ShouldFilter(ctx, tc.query)

			// Should return error
			if err == nil {
				t.Fatal("Expected error for missing tenant context, got nil")
			}

			// Should be TenantError
			var tenantErr *TenantError
			if !errors.As(err, &tenantErr) {
				t.Fatalf("Expected TenantError, got %T: %v", err, err)
			}

			// Should wrap ErrMissingTenant
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("Expected error to wrap %v, got %v", tc.expectedErr, err)
			}

			// Should include query in error
			querySnippet := tc.query
			if len(tc.query) > 20 {
				querySnippet = tc.query[:20]
			}
			if !strings.Contains(err.Error(), querySnippet) {
				t.Errorf("Error should include query snippet, got: %v", err)
			}

			// Should include tables in error
			sort.Strings(tenantErr.Tables)
			sort.Strings(tc.expectTables)
			if !reflect.DeepEqual(tenantErr.Tables, tc.expectTables) {
				t.Errorf("Expected tables %v in error, got %v", tc.expectTables, tenantErr.Tables)
			}

			// Decision should still be populated
			if decision == nil {
				t.Fatal("Decision should not be nil even on error")
			}
		})
	}
}

// TestInterceptor_MixedTables tests queries with both tenant and non-tenant tables
func TestInterceptor_MixedTables(t *testing.T) {
	config := Config{
		TenantTables: []string{"users", "orders"},
		TenantColumn: "tenant_id",
	}
	interceptor, _ := NewInterceptor(config)

	testCases := []struct {
		name          string
		query         string
		hasTenant     bool
		expectFilter  bool
		expectError   bool
		tenantTables  []string
		nonTenantTbls []string
	}{
		{
			name:          "User with country (has tenant)",
			query:         "SELECT * FROM users JOIN countries ON users.country_id = countries.id",
			hasTenant:     true,
			expectFilter:  true,
			expectError:   false,
			tenantTables:  []string{"users"},
			nonTenantTbls: []string{"countries"},
		},
		{
			name:          "User with country (no tenant)",
			query:         "SELECT * FROM users JOIN countries ON users.country_id = countries.id",
			hasTenant:     false,
			expectFilter:  false,
			expectError:   true,
			tenantTables:  []string{"users"},
			nonTenantTbls: []string{"countries"},
		},
		{
			name:          "Order with currency (has tenant)",
			query:         "SELECT * FROM orders o JOIN currencies c ON o.currency_code = c.code",
			hasTenant:     true,
			expectFilter:  true,
			expectError:   false,
			tenantTables:  []string{"orders"},
			nonTenantTbls: []string{"currencies"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var ctx context.Context
			if tc.hasTenant {
				ctx = WithTenant(context.Background(), "tenant-123")
			} else {
				ctx = context.Background()
			}

			decision, err := interceptor.ShouldFilter(ctx, tc.query)

			if tc.expectError {
				if err == nil {
					t.Fatal("Expected error for missing tenant, got nil")
				}
				if !errors.Is(err, ErrMissingTenant) {
					t.Errorf("Expected ErrMissingTenant, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if decision.RequiresFiltering != tc.expectFilter {
				t.Errorf("Expected RequiresFiltering=%v, got %v", tc.expectFilter, decision.RequiresFiltering)
			}

			// Verify tenant tables identified
			sort.Strings(decision.TenantTables)
			sort.Strings(tc.tenantTables)
			if !reflect.DeepEqual(decision.TenantTables, tc.tenantTables) {
				t.Errorf("Expected tenant tables %v, got %v", tc.tenantTables, decision.TenantTables)
			}
		})
	}
}

// TestInterceptor_ComplexJOINs tests complex multi-table JOINs
func TestInterceptor_ComplexJOINs(t *testing.T) {
	config := Config{
		TenantTables: []string{"users", "orders", "order_items"},
		TenantColumn: "tenant_id",
	}
	interceptor, _ := NewInterceptor(config)

	query := `
		SELECT u.name, o.total, oi.quantity, p.name
		FROM users u
		JOIN orders o ON u.id = o.user_id
		JOIN order_items oi ON o.id = oi.order_id
		JOIN products p ON oi.product_id = p.id
		WHERE o.status = ?
	`

	t.Run("With tenant context", func(t *testing.T) {
		ctx := WithTenant(context.Background(), "tenant-456")

		decision, err := interceptor.ShouldFilter(ctx, query)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !decision.RequiresFiltering {
			t.Error("Complex JOIN with tenant tables should require filtering")
		}

		// Should identify all tenant tables
		expectedTenantTables := []string{"users", "orders", "order_items"}
		sort.Strings(decision.TenantTables)
		sort.Strings(expectedTenantTables)
		if !reflect.DeepEqual(decision.TenantTables, expectedTenantTables) {
			t.Errorf("Expected tenant tables %v, got %v", expectedTenantTables, decision.TenantTables)
		}

		// Should extract all tables
		expectedAllTables := []string{"users", "orders", "order_items", "products"}
		sort.Strings(decision.ExtractedTables)
		sort.Strings(expectedAllTables)
		if !reflect.DeepEqual(decision.ExtractedTables, expectedAllTables) {
			t.Errorf("Expected all tables %v, got %v", expectedAllTables, decision.ExtractedTables)
		}
	})

	t.Run("Without tenant context", func(t *testing.T) {
		ctx := context.Background()

		_, err := interceptor.ShouldFilter(ctx, query)
		if err == nil {
			t.Fatal("Expected error for missing tenant, got nil")
		}

		if !errors.Is(err, ErrMissingTenant) {
			t.Errorf("Expected ErrMissingTenant, got %v", err)
		}
	})
}

// TestInterceptor_EdgeCases tests edge cases
func TestInterceptor_EdgeCases(t *testing.T) {
	config := Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	}
	interceptor, _ := NewInterceptor(config)

	testCases := []struct {
		name        string
		query       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Empty query",
			query:       "",
			expectError: false, // Empty query bypasses (no tables)
		},
		{
			name:        "Whitespace only",
			query:       "   \n\t  ",
			expectError: false, // Whitespace-only bypasses
		},
		{
			name:        "Comment only",
			query:       "-- This is a comment",
			expectError: false, // Comment bypasses (no tables)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			decision, err := interceptor.ShouldFilter(ctx, tc.query)

			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if decision.RequiresFiltering {
					t.Error("Edge case should bypass filtering")
				}
			}
		})
	}
}

// TestInterceptor_EmptyTenantID tests empty tenant ID is rejected
func TestInterceptor_EmptyTenantID(t *testing.T) {
	config := Config{
		TenantTables: []string{"users"},
		TenantColumn: "tenant_id",
	}
	interceptor, _ := NewInterceptor(config)

	// Create context with empty tenant ID
	ctx := WithTenant(context.Background(), "")

	_, err := interceptor.ShouldFilter(ctx, "SELECT * FROM users")

	if err == nil {
		t.Fatal("Expected error for empty tenant ID, got nil")
	}

	if !errors.Is(err, ErrMissingTenant) {
		t.Errorf("Expected ErrMissingTenant for empty tenant ID, got %v", err)
	}
}

// TestInterceptor_ConfigValidation tests config validation
func TestInterceptor_ConfigValidation(t *testing.T) {
	testCases := []struct {
		name      string
		config    Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid config",
			config: Config{
				TenantTables: []string{"users", "orders"},
				TenantColumn: "tenant_id",
			},
			expectErr: false,
		},
		{
			name: "Empty TenantTables",
			config: Config{
				TenantTables: []string{},
				TenantColumn: "tenant_id",
			},
			expectErr: true,
			errMsg:    "TenantTables cannot be empty",
		},
		{
			name: "Nil TenantTables",
			config: Config{
				TenantTables: nil,
				TenantColumn: "tenant_id",
			},
			expectErr: true,
			errMsg:    "TenantTables cannot be empty",
		},
		{
			name: "Empty TenantColumn uses default",
			config: Config{
				TenantTables: []string{"users"},
				TenantColumn: "",
			},
			expectErr: false, // Should use default "tenant_id"
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			interceptor, err := NewInterceptor(tc.config)

			if tc.expectErr {
				if err == nil {
					t.Error("Expected error, but got nil")
				} else if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tc.errMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify default column name
			if tc.config.TenantColumn == "" {
				if interceptor.config.TenantColumn != "tenant_id" {
					t.Errorf("Expected default TenantColumn 'tenant_id', got '%s'", interceptor.config.TenantColumn)
				}
			}
		})
	}
}

// TestInterceptor_CaseSensitivity tests table name case handling
func TestInterceptor_CaseSensitivity(t *testing.T) {
	config := Config{
		TenantTables: []string{"users", "orders"}, // Lowercase
		TenantColumn: "tenant_id",
	}
	interceptor, _ := NewInterceptor(config)

	testCases := []struct {
		name         string
		query        string
		expectFilter bool
		expectError  bool
	}{
		{
			name:         "Uppercase table name",
			query:        "SELECT * FROM USERS",
			expectFilter: true,
		},
		{
			name:         "Mixed case",
			query:        "SELECT * FROM Users",
			expectFilter: true,
		},
		{
			name:         "Lowercase (exact match)",
			query:        "SELECT * FROM users",
			expectFilter: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := WithTenant(context.Background(), "tenant-789")

			decision, err := interceptor.ShouldFilter(ctx, tc.query)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if decision.RequiresFiltering != tc.expectFilter {
				t.Errorf("Expected RequiresFiltering=%v, got %v", tc.expectFilter, decision.RequiresFiltering)
			}
		})
	}
}
