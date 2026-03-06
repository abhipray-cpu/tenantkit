package sqlx

import (
	"context"
	"testing"

	"github.com/abhipray-cpu/tenantkit/domain"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestNew(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	tdb := New(db, nil)
	if tdb == nil {
		t.Fatal("expected non-nil DB")
	}

	if tdb.tenantColumn != "tenant_id" {
		t.Errorf("expected tenant_id, got %s", tdb.tenantColumn)
	}
}

func TestNewWithConfig(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		TenantColumn: "org_id",
		SkipTables:   []string{"migrations", "system_config"},
	}

	tdb := New(db, cfg)
	if tdb.tenantColumn != "org_id" {
		t.Errorf("expected org_id, got %s", tdb.tenantColumn)
	}

	if !tdb.skipTables["migrations"] {
		t.Error("expected migrations to be skipped")
	}

	if !tdb.skipTables["system_config"] {
		t.Error("expected system_config to be skipped")
	}
}

func TestOpen(t *testing.T) {
	tdb, err := Open("sqlite3", ":memory:", nil)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer tdb.Close()

	if tdb == nil {
		t.Fatal("expected non-nil DB")
	}
}

func TestConnect(t *testing.T) {
	tdb, err := Connect("sqlite3", ":memory:", nil)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer tdb.Close()

	if tdb == nil {
		t.Fatal("expected non-nil DB")
	}

	// Test ping
	if err := tdb.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestGetTenantID(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	tdb := New(db, nil)

	tests := []struct {
		name    string
		ctx     context.Context
		want    string
		wantErr bool
	}{
		{
			name:    "nil context",
			ctx:     nil,
			wantErr: true,
		},
		{
			name:    "empty context",
			ctx:     context.Background(),
			wantErr: true,
		},
		{
			name: "valid tenant context",
			ctx: func() context.Context {
				tc, _ := domain.NewContext("t123", "u1", "r1")
				return tc.ToGoContext(context.Background())
			}(),
			want:    "t123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tdb.getTenantID(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTenantID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getTenantID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldSkip(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	cfg := &Config{
		SkipTables: []string{"migrations", "system_config"},
	}
	tdb := New(db, cfg)

	tests := []struct {
		name  string
		ctx   context.Context
		query string
		want  bool
	}{
		{
			name:  "skip flag set",
			ctx:   SkipTenant(context.Background()),
			query: "SELECT * FROM users",
			want:  true,
		},
		{
			name:  "skip table - migrations",
			ctx:   context.Background(),
			query: "SELECT * FROM migrations",
			want:  true,
		},
		{
			name:  "skip table - system_config",
			ctx:   context.Background(),
			query: "INSERT INTO system_config VALUES (?)",
			want:  true,
		},
		{
			name:  "normal query",
			ctx:   context.Background(),
			query: "SELECT * FROM users",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tdb.shouldSkip(tt.ctx, tt.query)
			if got != tt.want {
				t.Errorf("shouldSkip() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInjectTenantCondition(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	tdb := New(db, nil)

	tests := []struct {
		name         string
		query        string
		tenantID     string
		wantQuery    string
		wantPosition int
	}{
		{
			name:         "simple SELECT without WHERE",
			query:        "SELECT * FROM users",
			tenantID:     "t123",
			wantQuery:    "SELECT * FROM users WHERE tenant_id = ? ",
			wantPosition: 0,
		},
		{
			name:         "SELECT with WHERE",
			query:        "SELECT * FROM users WHERE active = 1",
			tenantID:     "t123",
			wantQuery:    "SELECT * FROM users WHERE tenant_id = ? AND active = 1",
			wantPosition: 0,
		},
		{
			name:         "SELECT with ORDER BY",
			query:        "SELECT * FROM users ORDER BY name",
			tenantID:     "t123",
			wantQuery:    "SELECT * FROM users  WHERE tenant_id = ? ORDER BY name",
			wantPosition: 0,
		},
		{
			name:         "SELECT with LIMIT",
			query:        "SELECT * FROM users LIMIT 10",
			tenantID:     "t123",
			wantQuery:    "SELECT * FROM users  WHERE tenant_id = ? LIMIT 10",
			wantPosition: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuery, gotPosition := tdb.injectTenantCondition(tt.query, tt.tenantID)
			if gotQuery != tt.wantQuery {
				t.Errorf("injectTenantCondition() query = %v, want %v", gotQuery, tt.wantQuery)
			}
			if gotPosition != tt.wantPosition {
				t.Errorf("injectTenantCondition() position = %v, want %v", gotPosition, tt.wantPosition)
			}
		})
	}
}

func TestInjectTenantIntoInsert(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	tdb := New(db, nil)

	tests := []struct {
		name      string
		query     string
		tenantID  string
		wantQuery string
	}{
		{
			name:      "simple INSERT",
			query:     "INSERT INTO users (name, email) VALUES (?, ?)",
			tenantID:  "t123",
			wantQuery: "INSERT INTO users (name, email, tenant_id) VALUES (?, ?, ?)",
		},
		{
			name:      "INSERT with multiple values",
			query:     "INSERT INTO users (name) VALUES (?)",
			tenantID:  "t123",
			wantQuery: "INSERT INTO users (name, tenant_id) VALUES (?, ?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tdb.injectTenantIntoInsert(tt.query, tt.tenantID)
			if got != tt.wantQuery {
				t.Errorf("injectTenantIntoInsert() = %v, want %v", got, tt.wantQuery)
			}
		})
	}
}

func TestWithoutTenant(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	tdb := New(db, nil)
	noTenantDB := tdb.WithoutTenant()

	if !noTenantDB.skipTables["*"] {
		t.Error("expected * to be in skip tables")
	}
}

func TestSkipTenant(t *testing.T) {
	ctx := context.Background()
	skipCtx := SkipTenant(ctx)

	skip, ok := skipCtx.Value(SkipTenantKey).(bool)
	if !ok || !skip {
		t.Error("expected skip flag to be set")
	}
}

func TestWithTenant(t *testing.T) {
	ctx := context.Background()
	tenantCtx, err := WithTenant(ctx, "t123")
	if err != nil {
		t.Fatalf("WithTenant() error = %v", err)
	}

	tc, err := domain.FromGoContext(tenantCtx)
	if err != nil {
		t.Fatalf("FromGoContext() error = %v", err)
	}

	if tc.TenantID().Value() != "t123" {
		t.Errorf("TenantID() = %v, want t123", tc.TenantID().Value())
	}
}

func TestWithTenantInvalidID(t *testing.T) {
	ctx := context.Background()
	_, err := WithTenant(ctx, "")
	if err == nil {
		t.Error("expected error for empty tenant ID")
	}
}
