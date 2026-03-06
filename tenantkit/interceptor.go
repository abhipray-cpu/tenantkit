package tenantkit

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Config holds the configuration for the interceptor
type Config struct {
	// TenantTables is the list of tables that require tenant filtering
	TenantTables []string

	// TenantColumn is the column name for tenant ID (default: "tenant_id")
	TenantColumn string

	// Logger is an optional structured logger. If nil, logging is disabled.
	// Libraries should never write to stdout unless explicitly configured.
	Logger *slog.Logger
}

// Decision represents the result of interceptor decision logic
type Decision struct {
	// RequiresFiltering indicates if the query requires tenant filtering
	RequiresFiltering bool

	// Reason explains why filtering is/isn't required
	Reason DecisionReason

	// TenantID is the tenant ID from context (if present)
	TenantID string

	// ExtractedTables are all tables found in the query
	ExtractedTables []string

	// TenantTables are tables that require tenant filtering
	TenantTables []string
}

// DecisionReason explains why a decision was made
type DecisionReason string

const (
	// ReasonSystemQuery indicates the query is a system query (DDL, health check, etc.)
	ReasonSystemQuery DecisionReason = "system_query"

	// ReasonExplicitBypass indicates context has explicit bypass flag
	ReasonExplicitBypass DecisionReason = "explicit_bypass"

	// ReasonNoTenantTables indicates query doesn't touch any tenant-scoped tables
	ReasonNoTenantTables DecisionReason = "no_tenant_tables"

	// ReasonTenantTableAccess indicates query accesses tenant-scoped tables
	ReasonTenantTableAccess DecisionReason = "tenant_table_access"
)

// Interceptor implements the Two-Rule System decision flow
type Interceptor struct {
	config         Config
	systemDetector *SystemQueryDetector
	tableExtractor *TableExtractor
	tenantTableMap map[string]bool // For fast tenant table lookup
}

// NewInterceptor creates a new interceptor with the given configuration.
// Returns an error if the configuration is invalid.
func NewInterceptor(config Config) (*Interceptor, error) {
	// Validate config
	if len(config.TenantTables) == 0 {
		return nil, fmt.Errorf("tenantkit: TenantTables cannot be empty")
	}

	// Apply defaults
	if config.TenantColumn == "" {
		config.TenantColumn = "tenant_id"
	}

	// Build tenant table lookup map (case-insensitive)
	tenantTableMap := make(map[string]bool)
	for _, table := range config.TenantTables {
		tenantTableMap[strings.ToLower(table)] = true
	}

	return &Interceptor{
		config:         config,
		systemDetector: NewSystemQueryDetector(),
		tableExtractor: NewTableExtractor(),
		tenantTableMap: tenantTableMap,
	}, nil
}

// ShouldFilter implements the Two-Rule System decision flow
//
// Decision Flow:
//  1. Is it a system query? (Rule 1) → Bypass
//  2. Does context have explicit bypass? → Bypass
//  3. Extract tables from query (Rule 2)
//  4. Any table in TenantTables? → NO: Bypass, YES: Require tenant
func (i *Interceptor) ShouldFilter(ctx context.Context, query string) (*Decision, error) {
	decision := &Decision{
		RequiresFiltering: false,
	}

	// RULE 1: Check if system query (DDL, health checks, etc.)
	if i.systemDetector.IsSystemQuery(query) {
		decision.Reason = ReasonSystemQuery
		return decision, nil
	}

	// Check for explicit bypass flag in context
	if shouldBypass(ctx) {
		decision.Reason = ReasonExplicitBypass
		return decision, nil
	}

	// RULE 2: Extract tables from query
	tables, err := i.tableExtractor.ExtractTables(query)
	if err != nil {
		// If extraction fails, be conservative - require tenant
		return nil, &TenantError{
			Query: query,
			Err:   fmt.Errorf("failed to extract tables: %w", err),
		}
	}

	decision.ExtractedTables = tables

	// If no tables found, bypass (e.g., empty query, comments only)
	if len(tables) == 0 {
		decision.Reason = ReasonNoTenantTables
		return decision, nil
	}

	// Check if any extracted table is a tenant table
	var tenantTables []string
	for _, table := range tables {
		tableLower := strings.ToLower(table)
		if i.tenantTableMap[tableLower] {
			tenantTables = append(tenantTables, table)
		}
	}

	decision.TenantTables = tenantTables

	// If no tenant tables touched, bypass
	if len(tenantTables) == 0 {
		decision.Reason = ReasonNoTenantTables
		return decision, nil
	}

	// Query touches tenant tables - require tenant context
	tenantID, ok := GetTenant(ctx)
	if !ok || tenantID == "" {
		// Missing tenant context - return error
		return decision, &TenantError{
			Query:  query,
			Tables: tenantTables,
			Err:    ErrMissingTenant,
		}
	}

	// All checks passed - require filtering
	decision.RequiresFiltering = true
	decision.TenantID = tenantID
	decision.Reason = ReasonTenantTableAccess

	return decision, nil
}
