package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// RLSConfig configures PostgreSQL Row-Level Security
type RLSConfig struct {
	TenantIDColumn string // Column name for tenant_id (default: "tenant_id")
	RLSPolicyName  string // RLS policy name prefix (default: "tenant_rls")
}

// RLSManager handles PostgreSQL Row-Level Security setup and enforcement
type RLSManager struct {
	conn     *pgx.Conn
	config   RLSConfig
	policies map[string]bool // Track installed policies
}

// NewRLSManager creates new RLS manager
func NewRLSManager(conn *pgx.Conn, config RLSConfig) *RLSManager {
	if config.TenantIDColumn == "" {
		config.TenantIDColumn = "tenant_id"
	}
	if config.RLSPolicyName == "" {
		config.RLSPolicyName = "tenant_rls"
	}

	return &RLSManager{
		conn:     conn,
		config:   config,
		policies: make(map[string]bool),
	}
}

// EnableRLS enables RLS on a table
func (r *RLSManager) EnableRLS(ctx context.Context, tableName string) error {
	sql := fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", tableName)
	_, err := r.conn.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to enable RLS on table %s: %w", tableName, err)
	}
	return nil
}

// ForceRLS forces RLS even for table owners (bypasses superuser/owner exemption)
func (r *RLSManager) ForceRLS(ctx context.Context, tableName string) error {
	sql := fmt.Sprintf("ALTER TABLE %s FORCE ROW LEVEL SECURITY;", tableName)
	_, err := r.conn.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to force RLS on table %s: %w", tableName, err)
	}
	return nil
}

// CreateTenantPolicy creates a tenant-scoped RLS policy
func (r *RLSManager) CreateTenantPolicy(ctx context.Context, tableName string) error {
	policyName := fmt.Sprintf("%s_%s", r.config.RLSPolicyName, tableName)

	// Using current_setting('app.current_tenant_id', true) - true means don't error if missing
	// The second parameter prevents errors when the setting isn't set
	sql := fmt.Sprintf(`
		CREATE POLICY %s ON %s
		USING (%s = current_setting('app.current_tenant_id', true))
		WITH CHECK (%s = current_setting('app.current_tenant_id', true));
	`, policyName, tableName, r.config.TenantIDColumn, r.config.TenantIDColumn)

	_, err := r.conn.Exec(ctx, sql)
	if err != nil {
		// Policy might already exist
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "42710" {
			r.policies[policyName] = true
			return nil
		}
		return fmt.Errorf("failed to create RLS policy on table %s: %w", tableName, err)
	}

	r.policies[policyName] = true
	return nil
}

// SetTenantContext sets the current tenant in PostgreSQL app context
func (r *RLSManager) SetTenantContext(ctx context.Context, tenantID string) error {
	// Use SET to set session variable
	sql := fmt.Sprintf("SELECT set_config('app.current_tenant_id', '%s', false);", tenantID)
	_, err := r.conn.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	return nil
}

// CreatePartition creates a tenant-specific partition
// Useful for large multi-tenant tables
func (r *RLSManager) CreatePartition(ctx context.Context, baseTable string, tenantID string) error {
	partitionName := fmt.Sprintf("%s_%s", baseTable, tenantID)

	sql := fmt.Sprintf(`
		CREATE TABLE %s PARTITION OF %s
		FOR VALUES IN ('%s');
	`, partitionName, baseTable, tenantID)

	_, err := r.conn.Exec(ctx, sql)
	if err != nil {
		// Partition might already exist
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "42710" {
			return nil
		}
		return fmt.Errorf("failed to create partition %s: %w", partitionName, err)
	}
	return nil
}

// VerifyTenantIsolation verifies that RLS is properly configured on a table
func (r *RLSManager) VerifyTenantIsolation(ctx context.Context, tableName string) (bool, error) {
	// Check if RLS is enabled (relrowsecurity is the correct column name)
	sql := fmt.Sprintf(`
		SELECT relrowsecurity FROM pg_class
		WHERE relname = '%s';
	`, tableName)

	var rlsEnabled bool
	err := r.conn.QueryRow(ctx, sql).Scan(&rlsEnabled)
	if err != nil {
		return false, fmt.Errorf("failed to verify RLS status: %w", err)
	}
	return rlsEnabled, nil
}

// GetActivePolicies returns list of active RLS policies
func (r *RLSManager) GetActivePolicies(ctx context.Context, tableName string) ([]string, error) {
	sql := fmt.Sprintf(`
		SELECT policyname FROM pg_policies
		WHERE tablename = '%s';
	`, tableName)

	rows, err := r.conn.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}
	defer rows.Close()

	var policies []string
	for rows.Next() {
		var policy string
		err := rows.Scan(&policy)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}
		policies = append(policies, policy)
	}

	return policies, nil
}
