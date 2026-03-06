package tenantkit

import (
	"errors"
	"strings"
	"testing"
)

// TestErrMissingTenant_IsError verifies error is defined
func TestErrMissingTenant_IsError(t *testing.T) {
	if ErrMissingTenant == nil {
		t.Fatal("ErrMissingTenant should be defined")
	}

	msg := ErrMissingTenant.Error()
	if !strings.Contains(msg, "tenant") {
		t.Errorf("Error message should mention tenant, got: %s", msg)
	}
}

// TestErrInvalidTenant_IsError verifies error is defined
func TestErrInvalidTenant_IsError(t *testing.T) {
	if ErrInvalidTenant == nil {
		t.Fatal("ErrInvalidTenant should be defined")
	}

	msg := ErrInvalidTenant.Error()
	if !strings.Contains(msg, "tenant") {
		t.Errorf("Error message should mention tenant, got: %s", msg)
	}
}

// TestErrQueryParsing_IsError verifies error is defined
func TestErrQueryParsing_IsError(t *testing.T) {
	if ErrQueryParsing == nil {
		t.Fatal("ErrQueryParsing should be defined")
	}

	msg := ErrQueryParsing.Error()
	if !strings.Contains(msg, "query") && !strings.Contains(msg, "parse") {
		t.Errorf("Error message should mention query/parse, got: %s", msg)
	}
}

// TestTenantError_Error returns formatted message
func TestTenantError_Error(t *testing.T) {
	err := &TenantError{
		Query:  "SELECT * FROM users WHERE id = ?",
		Tables: []string{"users"},
		Err:    ErrMissingTenant,
	}

	msg := err.Error()

	if !strings.Contains(msg, "tenantkit") {
		t.Errorf("Error should contain package name, got: %s", msg)
	}
	if !strings.Contains(msg, "users") {
		t.Errorf("Error should contain table name, got: %s", msg)
	}
	if !strings.Contains(msg, "SELECT") {
		t.Errorf("Error should contain query snippet, got: %s", msg)
	}
}

// TestTenantError_ErrorTruncatesLongQuery
func TestTenantError_ErrorTruncatesLongQuery(t *testing.T) {
	longQuery := strings.Repeat("SELECT * FROM users WHERE id = ? AND ", 10) + "name = ?"

	err := &TenantError{
		Query:  longQuery,
		Tables: []string{"users"},
		Err:    ErrMissingTenant,
	}

	msg := err.Error()

	// Should be truncated (< 150 chars for query portion)
	if len(msg) > 300 {
		t.Errorf("Error message should be truncated, got length %d", len(msg))
	}

	// Should indicate truncation with ...
	if !strings.Contains(msg, "...") {
		t.Error("Long query should be truncated with ...")
	}
}

// TestTenantError_Unwrap returns wrapped error
func TestTenantError_Unwrap(t *testing.T) {
	innerErr := ErrMissingTenant

	err := &TenantError{
		Query:  "SELECT * FROM users",
		Tables: []string{"users"},
		Err:    innerErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != innerErr {
		t.Errorf("Expected Unwrap to return %v, got %v", innerErr, unwrapped)
	}
}

// TestTenantError_ErrorsIs works with errors.Is
func TestTenantError_ErrorsIs(t *testing.T) {
	err := &TenantError{
		Query:  "SELECT * FROM users",
		Tables: []string{"users"},
		Err:    ErrMissingTenant,
	}

	if !errors.Is(err, ErrMissingTenant) {
		t.Error("errors.Is should work with wrapped TenantError")
	}
}

// TestTenantError_EmptyTables handles empty table list
func TestTenantError_EmptyTables(t *testing.T) {
	err := &TenantError{
		Query:  "SELECT 1",
		Tables: []string{},
		Err:    ErrQueryParsing,
	}

	msg := err.Error()

	// Should not panic, should have some content
	if msg == "" {
		t.Error("Error message should not be empty")
	}
}

// TestTenantError_NilTables handles nil table list
func TestTenantError_NilTables(t *testing.T) {
	err := &TenantError{
		Query:  "SELECT 1",
		Tables: nil,
		Err:    ErrQueryParsing,
	}

	msg := err.Error()

	// Should not panic
	if msg == "" {
		t.Error("Error message should not be empty")
	}
}

// TestTenantError_MultipleTables shows all tables
func TestTenantError_MultipleTables(t *testing.T) {
	err := &TenantError{
		Query:  "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
		Tables: []string{"users", "orders"},
		Err:    ErrMissingTenant,
	}

	msg := err.Error()

	if !strings.Contains(msg, "users") {
		t.Error("Error should contain 'users' table")
	}
	if !strings.Contains(msg, "orders") {
		t.Error("Error should contain 'orders' table")
	}
}
