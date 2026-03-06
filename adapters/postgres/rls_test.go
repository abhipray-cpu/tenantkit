package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// Helper to get test database connection
func getTestConn(t *testing.T) *pgx.Conn {
	// Get connection string from environment or use default
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		// Check if we have postgres available - if not, skip
		t.Skip("TEST_DATABASE_URL not set - skipping PostgreSQL tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}

	return conn
}

// Helper to cleanup tables
func cleanupTestTable(t *testing.T, conn *pgx.Conn, tableName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Drop table if exists
	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", tableName)
	_, err := conn.Exec(ctx, sql)
	if err != nil {
		t.Logf("Warning: Failed to cleanup table %s: %v", tableName, err)
	}
}

// Helper to create test table
func createTestTable(t *testing.T, conn *pgx.Conn, tableName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sql := fmt.Sprintf(`
		CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			tenant_id VARCHAR(255) NOT NULL,
			name VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`, tableName)

	_, err := conn.Exec(ctx, sql)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
}

// TestNewRLSManager tests RLSManager creation
func TestNewRLSManager(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	config := RLSConfig{
		TenantIDColumn: "tenant_id",
		RLSPolicyName:  "tenant_rls",
	}

	manager := NewRLSManager(conn, config)
	if manager == nil {
		t.Fatal("Expected non-nil RLSManager")
	}

	if manager.config.TenantIDColumn != "tenant_id" {
		t.Errorf("Expected TenantIDColumn=tenant_id, got %s", manager.config.TenantIDColumn)
	}

	if manager.config.RLSPolicyName != "tenant_rls" {
		t.Errorf("Expected RLSPolicyName=tenant_rls, got %s", manager.config.RLSPolicyName)
	}
}

// TestNewRLSManager_DefaultConfig tests default config values
func TestNewRLSManager_DefaultConfig(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	config := RLSConfig{}
	manager := NewRLSManager(conn, config)

	if manager.config.TenantIDColumn != "tenant_id" {
		t.Errorf("Expected default TenantIDColumn=tenant_id, got %s", manager.config.TenantIDColumn)
	}

	if manager.config.RLSPolicyName != "tenant_rls" {
		t.Errorf("Expected default RLSPolicyName=tenant_rls, got %s", manager.config.RLSPolicyName)
	}
}

// TestEnableRLS tests enabling RLS on a table
func TestEnableRLS(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	tableName := "test_rls_enable"
	defer cleanupTestTable(t, conn, tableName)
	createTestTable(t, conn, tableName)

	config := RLSConfig{}
	manager := NewRLSManager(conn, config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := manager.EnableRLS(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to enable RLS: %v", err)
	}

	// Verify RLS is enabled
	isEnabled, err := manager.VerifyTenantIsolation(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to verify RLS: %v", err)
	}

	if !isEnabled {
		t.Error("Expected RLS to be enabled")
	}
}

// TestCreateTenantPolicy tests creating RLS policies
func TestCreateTenantPolicy(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	tableName := "test_rls_policy"
	defer cleanupTestTable(t, conn, tableName)
	createTestTable(t, conn, tableName)

	config := RLSConfig{
		TenantIDColumn: "tenant_id",
		RLSPolicyName:  "tenant_rls",
	}
	manager := NewRLSManager(conn, config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First enable RLS
	err := manager.EnableRLS(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to enable RLS: %v", err)
	}

	// Create policy
	err = manager.CreateTenantPolicy(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to create tenant policy: %v", err)
	}

	// Verify policy exists
	policies, err := manager.GetActivePolicies(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to get policies: %v", err)
	}

	found := false
	for _, p := range policies {
		if p == "tenant_rls_test_rls_policy" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find tenant_rls_test_rls_policy")
	}
}

// TestCreateTenantPolicy_AlreadyExists tests handling of duplicate policy creation
func TestCreateTenantPolicy_AlreadyExists(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	tableName := "test_rls_policy_dup"
	defer cleanupTestTable(t, conn, tableName)
	createTestTable(t, conn, tableName)

	config := RLSConfig{}
	manager := NewRLSManager(conn, config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Enable RLS
	_ = manager.EnableRLS(ctx, tableName)

	// Create policy first time
	err1 := manager.CreateTenantPolicy(ctx, tableName)
	if err1 != nil {
		t.Fatalf("First policy creation failed: %v", err1)
	}

	// Create same policy again - should handle gracefully
	err2 := manager.CreateTenantPolicy(ctx, tableName)
	if err2 != nil {
		t.Fatalf("Second policy creation should succeed (duplicate): %v", err2)
	}
}

// TestSetTenantContext tests setting tenant context
func TestSetTenantContext(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	config := RLSConfig{}
	manager := NewRLSManager(conn, config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tenantID := "tenant-123"
	err := manager.SetTenantContext(ctx, tenantID)
	if err != nil {
		t.Fatalf("Failed to set tenant context: %v", err)
	}

	// Verify context was set (read it back)
	var retrievedTenantID string
	err = conn.QueryRow(ctx, "SELECT current_setting('app.current_tenant_id')").Scan(&retrievedTenantID)
	if err != nil {
		t.Fatalf("Failed to retrieve tenant context: %v", err)
	}

	if retrievedTenantID != tenantID {
		t.Errorf("Expected tenant context=%s, got %s", tenantID, retrievedTenantID)
	}
}

// TestSetTenantContext_MultipleTenants tests switching between tenant contexts
func TestSetTenantContext_MultipleTenants(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	config := RLSConfig{}
	manager := NewRLSManager(conn, config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tenants := []string{"tenant-1", "tenant-2", "tenant-3"}

	for _, tenantID := range tenants {
		err := manager.SetTenantContext(ctx, tenantID)
		if err != nil {
			t.Fatalf("Failed to set tenant context for %s: %v", tenantID, err)
		}

		var retrievedTenantID string
		err = conn.QueryRow(ctx, "SELECT current_setting('app.current_tenant_id')").Scan(&retrievedTenantID)
		if err != nil {
			t.Fatalf("Failed to retrieve tenant context for %s: %v", tenantID, err)
		}

		if retrievedTenantID != tenantID {
			t.Errorf("Expected tenant=%s, got %s", tenantID, retrievedTenantID)
		}
	}
}

// TestGetActivePolicies tests retrieving active policies
func TestGetActivePolicies(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	tableName := "test_get_policies"
	defer cleanupTestTable(t, conn, tableName)
	createTestTable(t, conn, tableName)

	config := RLSConfig{}
	manager := NewRLSManager(conn, config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Initially no policies
	policies, err := manager.GetActivePolicies(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to get policies: %v", err)
	}

	if len(policies) != 0 {
		t.Errorf("Expected 0 policies initially, got %d", len(policies))
	}

	// Enable RLS and create policy
	_ = manager.EnableRLS(ctx, tableName)
	_ = manager.CreateTenantPolicy(ctx, tableName)

	// Now should have 1 policy
	policies, err = manager.GetActivePolicies(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to get policies: %v", err)
	}

	if len(policies) != 1 {
		t.Errorf("Expected 1 policy, got %d", len(policies))
	}
}

// TestVerifyTenantIsolation tests RLS verification
func TestVerifyTenantIsolation(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	tableName := "test_verify_rls"
	defer cleanupTestTable(t, conn, tableName)
	createTestTable(t, conn, tableName)

	config := RLSConfig{}
	manager := NewRLSManager(conn, config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Before enabling RLS
	isEnabled, err := manager.VerifyTenantIsolation(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to verify RLS status: %v", err)
	}

	if isEnabled {
		t.Error("Expected RLS to be disabled initially")
	}

	// Enable RLS
	_ = manager.EnableRLS(ctx, tableName)

	// After enabling RLS
	isEnabled, err = manager.VerifyTenantIsolation(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to verify RLS status: %v", err)
	}

	if !isEnabled {
		t.Error("Expected RLS to be enabled after EnableRLS()")
	}
}

// TestCreatePartition tests partition creation
func TestCreatePartition(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	baseTableName := "test_partitioned_base"
	defer cleanupTestTable(t, conn, baseTableName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create partitioned base table
	sql := fmt.Sprintf(`
		CREATE TABLE %s (
			id SERIAL,
			tenant_id VARCHAR(255),
			name VARCHAR(255),
			PRIMARY KEY (id, tenant_id)
		) PARTITION BY LIST (tenant_id);
	`, baseTableName)

	_, err := conn.Exec(ctx, sql)
	if err != nil {
		t.Fatalf("Failed to create partitioned table: %v", err)
	}

	config := RLSConfig{}
	manager := NewRLSManager(conn, config)

	// Create partition for tenant-1
	tenantID := "tenant-1"
	err = manager.CreatePartition(ctx, baseTableName, tenantID)
	if err != nil {
		t.Fatalf("Failed to create partition: %v", err)
	}

	// Verify partition was created
	partitionName := fmt.Sprintf("%s_%s", baseTableName, tenantID)
	sql = fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = '%s');", partitionName)
	var exists bool
	err = conn.QueryRow(ctx, sql).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to verify partition: %v", err)
	}

	if !exists {
		t.Error("Expected partition to be created")
	}
}

// TestCreatePartition_AlreadyExists tests handling of duplicate partition
func TestCreatePartition_AlreadyExists(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	baseTableName := "test_partitioned_dup"
	defer cleanupTestTable(t, conn, baseTableName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create partitioned base table
	sql := fmt.Sprintf(`
		CREATE TABLE %s (
			id SERIAL,
			tenant_id VARCHAR(255),
			name VARCHAR(255),
			PRIMARY KEY (id, tenant_id)
		) PARTITION BY LIST (tenant_id);
	`, baseTableName)

	_, err := conn.Exec(ctx, sql)
	if err != nil {
		t.Fatalf("Failed to create partitioned table: %v", err)
	}

	config := RLSConfig{}
	manager := NewRLSManager(conn, config)

	tenantID := "tenant-1"

	// Create partition first time
	err1 := manager.CreatePartition(ctx, baseTableName, tenantID)
	if err1 != nil {
		t.Fatalf("First partition creation failed: %v", err1)
	}

	// Create same partition again - should handle gracefully
	err2 := manager.CreatePartition(ctx, baseTableName, tenantID)
	if err2 != nil {
		t.Fatalf("Second partition creation should succeed (duplicate): %v", err2)
	}
}

// TestEndToEnd_RLSSetup tests complete RLS setup workflow
func TestEndToEnd_RLSSetup(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	tableName := "test_e2e_rls"
	defer cleanupTestTable(t, conn, tableName)
	createTestTable(t, conn, tableName)

	config := RLSConfig{
		TenantIDColumn: "tenant_id",
		RLSPolicyName:  "e2e_policy",
	}
	manager := NewRLSManager(conn, config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 1: Enable RLS
	err := manager.EnableRLS(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to enable RLS: %v", err)
	}

	// Step 2: Create policy
	err = manager.CreateTenantPolicy(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	// Step 3: Set tenant context
	tenantID := "end-to-end-tenant"
	err = manager.SetTenantContext(ctx, tenantID)
	if err != nil {
		t.Fatalf("Failed to set tenant context: %v", err)
	}

	// Step 4: Verify isolation is enabled
	isEnabled, err := manager.VerifyTenantIsolation(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to verify isolation: %v", err)
	}

	if !isEnabled {
		t.Error("Expected RLS to be enabled")
	}

	// Step 5: Verify policies exist
	policies, err := manager.GetActivePolicies(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to get policies: %v", err)
	}

	if len(policies) != 1 {
		t.Errorf("Expected 1 policy, got %d", len(policies))
	}

	// Step 6: Verify context is set
	var currentTenant string
	err = conn.QueryRow(ctx, "SELECT current_setting('app.current_tenant_id')").Scan(&currentTenant)
	if err != nil {
		t.Fatalf("Failed to get current tenant: %v", err)
	}

	if currentTenant != tenantID {
		t.Errorf("Expected tenant=%s, got %s", tenantID, currentTenant)
	}
}

// TestRLSManager_ConcurrentContextSetting tests concurrent tenant context switching
func TestRLSManager_ConcurrentContextSetting(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	config := RLSConfig{}
	manager := NewRLSManager(conn, config)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Since pgx connections are not thread-safe, we test setting different contexts
	// in sequence which simulates tenant switching
	tenants := []string{"tenant-a", "tenant-b", "tenant-c"}

	for i := 0; i < 3; i++ {
		for _, tenantID := range tenants {
			err := manager.SetTenantContext(ctx, tenantID)
			if err != nil {
				t.Fatalf("Failed to set tenant context: %v", err)
			}

			var current string
			err = conn.QueryRow(ctx, "SELECT current_setting('app.current_tenant_id')").Scan(&current)
			if err != nil {
				t.Fatalf("Failed to get tenant context: %v", err)
			}

			if current != tenantID {
				t.Errorf("Expected tenant=%s, got %s", tenantID, current)
			}
		}
	}
}

// TestRLSManager_MultipleTablesWithPolicies tests RLS on multiple tables
func TestRLSManager_MultipleTablesWithPolicies(t *testing.T) {
	conn := getTestConn(t)
	if conn == nil {
		return
	}
	defer conn.Close(context.Background())

	tableNames := []string{"test_multi_1", "test_multi_2", "test_multi_3"}

	for _, tableName := range tableNames {
		defer cleanupTestTable(t, conn, tableName)
		createTestTable(t, conn, tableName)
	}

	config := RLSConfig{}
	manager := NewRLSManager(conn, config)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Enable RLS and create policies for all tables
	for _, tableName := range tableNames {
		err := manager.EnableRLS(ctx, tableName)
		if err != nil {
			t.Fatalf("Failed to enable RLS on %s: %v", tableName, err)
		}

		err = manager.CreateTenantPolicy(ctx, tableName)
		if err != nil {
			t.Fatalf("Failed to create policy on %s: %v", tableName, err)
		}
	}

	// Verify all tables have RLS enabled
	for _, tableName := range tableNames {
		isEnabled, err := manager.VerifyTenantIsolation(ctx, tableName)
		if err != nil {
			t.Fatalf("Failed to verify RLS on %s: %v", tableName, err)
		}

		if !isEnabled {
			t.Errorf("Expected RLS to be enabled on %s", tableName)
		}

		policies, err := manager.GetActivePolicies(ctx, tableName)
		if err != nil {
			t.Fatalf("Failed to get policies for %s: %v", tableName, err)
		}

		if len(policies) != 1 {
			t.Errorf("Expected 1 policy on %s, got %d", tableName, len(policies))
		}
	}
}
