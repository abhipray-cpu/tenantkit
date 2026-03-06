package sqladapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/abhipray-cpu/tenantkit/domain"
	"github.com/abhipray-cpu/tenantkit/tenantkit/ports"
)

// Compile-time interface check
var _ ports.Enforcer = (*Enforcer)(nil)

// Enforcer implements the ports.Enforcer interface by rewriting SQL queries
// to automatically add tenant filtering.
type Enforcer struct {
	tenantColumn string
}

// NewEnforcer creates a new SQL enforcer for tenant isolation.
func NewEnforcer() *Enforcer {
	return &Enforcer{
		tenantColumn: "tenant_id",
	}
}

// NewEnforcerWithColumn creates an enforcer with a custom tenant column name.
func NewEnforcerWithColumn(column string) *Enforcer {
	return &Enforcer{
		tenantColumn: column,
	}
}

// EnforceQuery rewrites a SQL query to add automatic tenant filtering.
func (e *Enforcer) EnforceQuery(ctx context.Context, query string, args []interface{}) (string, []interface{}, error) {
	tenantCtx, err := domain.FromGoContext(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract tenant context: %w", err)
	}

	if err := e.ValidateQuery(ctx, query); err != nil {
		return "", nil, fmt.Errorf("query validation failed: %w", err)
	}

	tenantID := tenantCtx.TenantID().Value()
	rewritten, err := e.rewriteQuery(query, tenantID)
	if err != nil {
		return "", nil, fmt.Errorf("query rewrite failed: %w", err)
	}

	return rewritten, args, nil
}

// ValidateQuery validates that a query is safe and properly scoped.
func (e *Enforcer) ValidateQuery(_ context.Context, query string) error {
	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}

	upper := strings.ToUpper(strings.TrimSpace(query))
	dangerous := []string{"DROP TABLE", "DROP DATABASE", "TRUNCATE", "ALTER TABLE"}
	for _, d := range dangerous {
		if strings.Contains(upper, d) {
			return fmt.Errorf("dangerous operation not allowed: %s", d)
		}
	}

	return nil
}

// SupportedOperations returns the list of supported SQL operations.
func (e *Enforcer) SupportedOperations() []string {
	return []string{"SELECT", "INSERT", "UPDATE", "DELETE"}
}

func (e *Enforcer) rewriteQuery(query, tenantID string) (string, error) {
	upper := strings.ToUpper(strings.TrimSpace(query))
	tenantCondition := fmt.Sprintf("%s = '%s'", e.tenantColumn, tenantID)

	switch {
	case strings.HasPrefix(upper, "SELECT"), strings.HasPrefix(upper, "DELETE"):
		wherePos := strings.Index(upper, " WHERE ")
		if wherePos != -1 {
			return query[:wherePos+7] + tenantCondition + " AND (" + strings.TrimSpace(query[wherePos+7:]) + ")", nil
		}
		for _, kw := range []string{" ORDER BY ", " GROUP BY ", " LIMIT ", " HAVING "} {
			if pos := strings.Index(upper, kw); pos != -1 {
				return query[:pos] + " WHERE " + tenantCondition + query[pos:], nil
			}
		}
		return query + " WHERE " + tenantCondition, nil

	case strings.HasPrefix(upper, "UPDATE"):
		wherePos := strings.Index(upper, " WHERE ")
		if wherePos != -1 {
			return query[:wherePos+7] + tenantCondition + " AND (" + strings.TrimSpace(query[wherePos+7:]) + ")", nil
		}
		return query + " WHERE " + tenantCondition, nil

	case strings.HasPrefix(upper, "INSERT"):
		return query, nil

	default:
		return query, nil
	}
}

// VerifyTenantIsolation verifies that a rewritten query properly isolates by tenant ID.
func (e *Enforcer) VerifyTenantIsolation(query string, expectedTenantID string) bool {
	return e.extractTenantIDFromQuery(query) == expectedTenantID
}

func (e *Enforcer) extractTenantIDFromQuery(query string) string {
	lower := strings.ToLower(query)
	idx := strings.Index(lower, e.tenantColumn)
	if idx == -1 {
		return ""
	}
	remaining := lower[idx:]
	eqIdx := strings.Index(remaining, "=")
	if eqIdx == -1 {
		return ""
	}
	afterEq := remaining[eqIdx+1:]
	quoteIdx := strings.Index(afterEq, "'")
	if quoteIdx == -1 {
		return ""
	}
	afterQuote := afterEq[quoteIdx+1:]
	endQuoteIdx := strings.Index(afterQuote, "'")
	if endQuoteIdx == -1 {
		return ""
	}
	return afterQuote[:endQuoteIdx]
}
