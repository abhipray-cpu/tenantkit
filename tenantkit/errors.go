package tenantkit

import (
	"errors"
	"fmt"
)

// Predefined error variables for common tenant-related errors
var (
	// ErrMissingTenant indicates a query requires tenant context but none was provided
	ErrMissingTenant = errors.New("tenantkit: tenant context required for this query")

	// ErrInvalidTenant indicates the tenant ID format is invalid
	ErrInvalidTenant = errors.New("tenantkit: invalid tenant ID")

	// ErrQueryParsing indicates the query could not be parsed for tenant filtering
	ErrQueryParsing = errors.New("tenantkit: failed to parse query")
)

// TenantError provides detailed context about a tenant-related error.
// It includes the query that failed, the tables involved, and the underlying error.
type TenantError struct {
	// Query is the SQL query that caused the error (may be truncated)
	Query string

	// Tables are the table names extracted from the query
	Tables []string

	// Err is the underlying error
	Err error
}

// Error implements the error interface, returning a formatted error message
// with query and table context.
func (e *TenantError) Error() string {
	truncatedQuery := truncate(e.Query, 100)
	return fmt.Sprintf("tenantkit: %v (query: %s, tables: %v)",
		e.Err, truncatedQuery, e.Tables)
}

// Unwrap returns the underlying error, enabling errors.Is and errors.As
func (e *TenantError) Unwrap() error {
	return e.Err
}

// truncate shortens a string to the specified maximum length,
// adding "..." if truncation occurs
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
