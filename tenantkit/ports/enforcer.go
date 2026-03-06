package ports

import (
	"context"
)

// Enforcer is a port interface for enforcing tenant isolation on SQL queries.
// It rewrites queries to add tenant filters and validates that all queries
// are safe and properly scoped to the current tenant.
type Enforcer interface {
	// EnforceQuery rewrites a SQL query to add automatic tenant filtering.
	// Returns the rewritten query string and updated arguments.
	EnforceQuery(ctx context.Context, query string, args []interface{}) (string, []interface{}, error)

	// ValidateQuery validates that a query is safe and properly scoped.
	// Returns an error if the query is unsafe or invalid.
	ValidateQuery(ctx context.Context, query string) error

	// SupportedOperations returns the list of supported SQL operations (SELECT, INSERT, UPDATE, DELETE).
	SupportedOperations() []string
}
