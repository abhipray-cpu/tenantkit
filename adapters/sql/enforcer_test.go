package sqladapter

import (
	"context"
	"testing"

	"github.com/abhipray-cpu/tenantkit/domain"
)

func TestNewEnforcer(t *testing.T) {
	enforcer := NewEnforcer()
	if enforcer == nil {
		t.Error("NewEnforcer() returned nil")
	}
}

func TestEnforceQuery(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantErr   bool
		checkFunc func(t *testing.T, query string)
	}{
		{
			name:    "SELECT with tenant context",
			query:   "SELECT * FROM users WHERE id = 1",
			wantErr: false,
			checkFunc: func(t *testing.T, query string) {
				if !contains(query, "tenant_id") {
					t.Errorf("expected rewritten query to contain tenant_id, got: %s", query)
				}
			},
		},
		{
			name:    "UPDATE with tenant context",
			query:   "UPDATE users SET name = 'test' WHERE id = 1",
			wantErr: false,
			checkFunc: func(t *testing.T, query string) {
				if !contains(query, "tenant_id") {
					t.Errorf("expected rewritten query to contain tenant_id, got: %s", query)
				}
			},
		},
		{
			name:    "DELETE with tenant context",
			query:   "DELETE FROM users WHERE id = 1",
			wantErr: false,
			checkFunc: func(t *testing.T, query string) {
				if !contains(query, "tenant_id") {
					t.Errorf("expected rewritten query to contain tenant_id, got: %s", query)
				}
			},
		},
		{
			name:    "INSERT passes through unchanged",
			query:   "INSERT INTO users (name) VALUES ('test')",
			wantErr: false,
			checkFunc: func(t *testing.T, query string) {
				// INSERT queries are passed through unchanged by the enforcer;
				// tenant column injection for INSERTs is handled by the DB wrapper layer
				if query != "INSERT INTO users (name) VALUES ('test')" {
					t.Errorf("expected INSERT query to pass through unchanged, got: %s", query)
				}
			},
		},
	}

	enforcer := NewEnforcer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a tenant context and add it to Go context
			tc, err := domain.NewContext("tenant123", "user1", "req123")
			if err != nil {
				t.Fatalf("failed to create tenant context: %v", err)
			}
			ctx := tc.ToGoContext(context.Background())

			rewritten, args, err := enforcer.EnforceQuery(ctx, tt.query, []interface{}{})

			if (err != nil) != tt.wantErr {
				t.Errorf("EnforceQuery() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				tt.checkFunc(t, rewritten)
				if len(args) != 0 {
					t.Errorf("expected args to be unchanged, got: %v", args)
				}
			}
		})
	}
}

func TestEnforceQueryMissingContext(t *testing.T) {
	enforcer := NewEnforcer()

	// Try to enforce query without tenant context
	ctx := context.Background()
	_, _, err := enforcer.EnforceQuery(ctx, "SELECT * FROM users", []interface{}{})

	if err == nil {
		t.Error("expected error when tenant context is missing, got nil")
	}
}

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "valid SELECT",
			query:   "SELECT * FROM users",
			wantErr: false,
		},
		{
			name:    "valid INSERT",
			query:   "INSERT INTO users (name) VALUES ('test')",
			wantErr: false,
		},
		{
			name:    "valid UPDATE",
			query:   "UPDATE users SET name = 'test'",
			wantErr: false,
		},
		{
			name:    "valid DELETE",
			query:   "DELETE FROM users",
			wantErr: false,
		},
		{
			name:    "dangerous DROP",
			query:   "DROP TABLE users",
			wantErr: true,
		},
		{
			name:    "dangerous TRUNCATE",
			query:   "TRUNCATE TABLE users",
			wantErr: true,
		},
		{
			name:    "dangerous ALTER",
			query:   "ALTER TABLE users ADD COLUMN admin BOOLEAN",
			wantErr: true,
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: true,
		},
	}

	enforcer := NewEnforcer()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := enforcer.ValidateQuery(ctx, tt.query)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateQuery() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSupportedOperations(t *testing.T) {
	enforcer := NewEnforcer()
	ops := enforcer.SupportedOperations()

	expected := []string{"SELECT", "INSERT", "UPDATE", "DELETE"}
	if len(ops) != len(expected) {
		t.Errorf("expected %d operations, got %d", len(expected), len(ops))
		return
	}

	for _, exp := range expected {
		found := false
		for _, op := range ops {
			if op == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected operation %s not found", exp)
		}
	}
}

func TestEnforceQueryWithContext(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		tenantID  string
		wantErr   bool
		checkFunc func(t *testing.T, query string)
	}{
		{
			name:     "valid SELECT",
			query:    "SELECT * FROM users",
			tenantID: "tenant123",
			wantErr:  false,
			checkFunc: func(t *testing.T, query string) {
				if !contains(query, "tenant_id") {
					t.Errorf("expected rewritten query to contain tenant_id, got: %s", query)
				}
			},
		},
		{
			name:     "dangerous query",
			query:    "DROP TABLE users",
			tenantID: "tenant123",
			wantErr:  true,
		},
	}

	enforcer := NewEnforcer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc, tcErr := domain.NewContext(tt.tenantID, "test-user", "req-1")
			if tcErr != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("failed to create tenant context: %v", tcErr)
			}
			ctx := tc.ToGoContext(context.Background())

			rewritten, _, err := enforcer.EnforceQuery(ctx, tt.query, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("EnforceQuery() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, rewritten)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
