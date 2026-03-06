package quotaredis

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abhipray-cpu/tenantkit/domain"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setupTestRedis(t *testing.T) (*RedisQuotaManager, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	config := DefaultConfig()
	config.Limits = map[string]int64{
		"api_requests_daily": 5000,
		"storage_bytes":      1073741824,
		"concurrent_test":    100,
	}

	qm, err := NewRedisQuotaManager(client, config)
	if err != nil {
		mr.Close()
		t.Fatalf("Failed to create quota manager: %v", err)
	}

	return qm, mr
}

func createTenantContext(t *testing.T, tenantID string) context.Context {
	t.Helper()
	tc, err := domain.NewContext(tenantID, "test-user", "test-req")
	if err != nil {
		t.Fatalf("Failed to create tenant context: %v", err)
	}
	return tc.ToGoContext(context.Background())
}

func createQuotaManager(t *testing.T, addr string, config Config) *RedisQuotaManager {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	qm, err := NewRedisQuotaManager(client, config)
	if err != nil {
		t.Fatalf("Failed to create quota manager: %v", err)
	}

	return qm
}

// ---------------------------------------------------------------------------
// Port Interface Compliance Tests
// ---------------------------------------------------------------------------

func TestCheckQuota_Basic(t *testing.T) {
	qm, mr := setupTestRedis(t)
	defer mr.Close()
	defer qm.Close()

	ctx := createTenantContext(t, "tenant1")

	t.Run("within limit", func(t *testing.T) {
		allowed, err := qm.CheckQuota(ctx, "api_requests_daily", 100)
		if err != nil {
			t.Fatalf("CheckQuota error: %v", err)
		}
		if !allowed {
			t.Fatal("Expected quota to be allowed")
		}
	})

	t.Run("exceeds limit", func(t *testing.T) {
		allowed, err := qm.CheckQuota(ctx, "api_requests_daily", 6000)
		if err != nil {
			t.Fatalf("CheckQuota error: %v", err)
		}
		if allowed {
			t.Fatal("Expected quota to be denied")
		}
	})

	t.Run("check does not consume", func(t *testing.T) {
		allowed1, _ := qm.CheckQuota(ctx, "api_requests_daily", 5000)
		allowed2, _ := qm.CheckQuota(ctx, "api_requests_daily", 5000)
		if !allowed1 || !allowed2 {
			t.Fatal("CheckQuota should be read-only, second check should still pass")
		}
	})

	t.Run("negative amount returns error", func(t *testing.T) {
		_, err := qm.CheckQuota(ctx, "api_requests_daily", -1)
		if err == nil {
			t.Fatal("Expected error for negative amount")
		}
	})

	t.Run("unknown quota type returns error", func(t *testing.T) {
		_, err := qm.CheckQuota(ctx, "nonexistent_type", 1)
		if err == nil {
			t.Fatal("Expected error for unknown quota type")
		}
	})

	t.Run("missing tenant context returns error", func(t *testing.T) {
		_, err := qm.CheckQuota(context.Background(), "api_requests_daily", 1)
		if err == nil {
			t.Fatal("Expected error when tenant context is missing")
		}
	})
}

func TestConsumeQuota_Basic(t *testing.T) {
	qm, mr := setupTestRedis(t)
	defer mr.Close()
	defer qm.Close()

	ctx := createTenantContext(t, "tenant1")

	t.Run("consume returns correct remaining", func(t *testing.T) {
		remaining, err := qm.ConsumeQuota(ctx, "api_requests_daily", 100)
		if err != nil {
			t.Fatalf("ConsumeQuota error: %v", err)
		}
		if remaining != 4900 {
			t.Fatalf("Expected remaining 4900, got %d", remaining)
		}
	})

	t.Run("cumulative consumption", func(t *testing.T) {
		remaining, err := qm.ConsumeQuota(ctx, "api_requests_daily", 200)
		if err != nil {
			t.Fatalf("ConsumeQuota error: %v", err)
		}
		if remaining != 4700 {
			t.Fatalf("Expected remaining 4700, got %d", remaining)
		}
	})

	t.Run("exceed quota returns error", func(t *testing.T) {
		_, err := qm.ConsumeQuota(ctx, "api_requests_daily", 5000)
		if err == nil {
			t.Fatal("Expected error when exceeding quota")
		}
	})

	t.Run("negative amount returns error", func(t *testing.T) {
		_, err := qm.ConsumeQuota(ctx, "api_requests_daily", -1)
		if err == nil {
			t.Fatal("Expected error for negative amount")
		}
	})
}

func TestGetUsage_Basic(t *testing.T) {
	qm, mr := setupTestRedis(t)
	defer mr.Close()
	defer qm.Close()

	ctx := createTenantContext(t, "tenant1")

	t.Run("initial usage is zero", func(t *testing.T) {
		used, limit, err := qm.GetUsage(ctx, "api_requests_daily")
		if err != nil {
			t.Fatalf("GetUsage error: %v", err)
		}
		if used != 0 {
			t.Fatalf("Expected used 0, got %d", used)
		}
		if limit != 5000 {
			t.Fatalf("Expected limit 5000, got %d", limit)
		}
	})

	t.Run("usage after consumption", func(t *testing.T) {
		_, err := qm.ConsumeQuota(ctx, "api_requests_daily", 150)
		if err != nil {
			t.Fatalf("ConsumeQuota error: %v", err)
		}
		used, limit, err := qm.GetUsage(ctx, "api_requests_daily")
		if err != nil {
			t.Fatalf("GetUsage error: %v", err)
		}
		if used != 150 {
			t.Fatalf("Expected used 150, got %d", used)
		}
		if limit != 5000 {
			t.Fatalf("Expected limit 5000, got %d", limit)
		}
	})
}

func TestResetQuota_Basic(t *testing.T) {
	qm, mr := setupTestRedis(t)
	defer mr.Close()
	defer qm.Close()

	ctx := createTenantContext(t, "tenant1")

	_, err := qm.ConsumeQuota(ctx, "api_requests_daily", 500)
	if err != nil {
		t.Fatalf("ConsumeQuota error: %v", err)
	}

	err = qm.ResetQuota(ctx, "api_requests_daily")
	if err != nil {
		t.Fatalf("ResetQuota error: %v", err)
	}

	used, _, err := qm.GetUsage(ctx, "api_requests_daily")
	if err != nil {
		t.Fatalf("GetUsage error: %v", err)
	}
	if used != 0 {
		t.Fatalf("Expected used 0 after reset, got %d", used)
	}

	remaining, err := qm.ConsumeQuota(ctx, "api_requests_daily", 5000)
	if err != nil {
		t.Fatalf("Should succeed after reset: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("Expected remaining 0, got %d", remaining)
	}
}

func TestSetLimit_Basic(t *testing.T) {
	qm, mr := setupTestRedis(t)
	defer mr.Close()
	defer qm.Close()

	ctx := createTenantContext(t, "tenant1")

	t.Run("set per-tenant limit overrides config", func(t *testing.T) {
		err := qm.SetLimit(ctx, "api_requests_daily", 10000)
		if err != nil {
			t.Fatalf("SetLimit error: %v", err)
		}
		remaining, err := qm.ConsumeQuota(ctx, "api_requests_daily", 8000)
		if err != nil {
			t.Fatalf("Should succeed with new limit: %v", err)
		}
		if remaining != 2000 {
			t.Fatalf("Expected remaining 2000, got %d", remaining)
		}
	})

	t.Run("other tenant still has default limit", func(t *testing.T) {
		ctx2 := createTenantContext(t, "tenant2")
		_, limit, err := qm.GetUsage(ctx2, "api_requests_daily")
		if err != nil {
			t.Fatalf("GetUsage error: %v", err)
		}
		if limit != 5000 {
			t.Fatalf("Expected default limit 5000 for tenant2, got %d", limit)
		}
	})

	t.Run("negative limit returns error", func(t *testing.T) {
		err := qm.SetLimit(ctx, "api_requests_daily", -1)
		if err == nil {
			t.Fatal("Expected error for negative limit")
		}
	})
}

// ---------------------------------------------------------------------------
// Tenant Isolation Tests
// ---------------------------------------------------------------------------

func TestTenantIsolation(t *testing.T) {
	qm, mr := setupTestRedis(t)
	defer mr.Close()
	defer qm.Close()

	ctx1 := createTenantContext(t, "tenant1")
	ctx2 := createTenantContext(t, "tenant2")

	t.Run("tenants have independent quotas", func(t *testing.T) {
		_, err := qm.ConsumeQuota(ctx1, "api_requests_daily", 1000)
		if err != nil {
			t.Fatalf("Tenant1 consume error: %v", err)
		}
		remaining, err := qm.ConsumeQuota(ctx2, "api_requests_daily", 500)
		if err != nil {
			t.Fatalf("Tenant2 consume error: %v", err)
		}
		if remaining != 4500 {
			t.Fatalf("Expected tenant2 remaining 4500, got %d", remaining)
		}
	})

	t.Run("resetting one tenant does not affect another", func(t *testing.T) {
		err := qm.ResetQuota(ctx1, "api_requests_daily")
		if err != nil {
			t.Fatalf("ResetQuota error: %v", err)
		}
		used2, _, _ := qm.GetUsage(ctx2, "api_requests_daily")
		if used2 != 500 {
			t.Fatalf("Expected tenant2 used 500, got %d", used2)
		}
		used1, _, _ := qm.GetUsage(ctx1, "api_requests_daily")
		if used1 != 0 {
			t.Fatalf("Expected tenant1 used 0 after reset, got %d", used1)
		}
	})

	t.Run("per-tenant limit does not affect other tenants", func(t *testing.T) {
		err := qm.SetLimit(ctx1, "api_requests_daily", 50000)
		if err != nil {
			t.Fatalf("SetLimit error: %v", err)
		}
		_, err = qm.ConsumeQuota(ctx1, "api_requests_daily", 10000)
		if err != nil {
			t.Fatalf("Tenant1 should succeed with premium limit: %v", err)
		}
		_, err = qm.ConsumeQuota(ctx2, "api_requests_daily", 5000)
		if err == nil {
			t.Fatal("Tenant2 should fail — already consumed 500 of 5000")
		}
	})
}

// ---------------------------------------------------------------------------
// Distributed Consistency Tests
// ---------------------------------------------------------------------------

func TestDistributed_SharedQuotaAcrossInstances(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	config := DefaultConfig()
	config.Limits = map[string]int64{"shared_quota": 100}

	inst1 := createQuotaManager(t, mr.Addr(), config)
	defer inst1.Close()
	inst2 := createQuotaManager(t, mr.Addr(), config)
	defer inst2.Close()
	inst3 := createQuotaManager(t, mr.Addr(), config)
	defer inst3.Close()

	ctx := createTenantContext(t, "tenant1")

	_, err = inst1.ConsumeQuota(ctx, "shared_quota", 40)
	if err != nil {
		t.Fatalf("Instance 1 should succeed: %v", err)
	}

	_, err = inst2.ConsumeQuota(ctx, "shared_quota", 40)
	if err != nil {
		t.Fatalf("Instance 2 should succeed: %v", err)
	}

	_, err = inst3.ConsumeQuota(ctx, "shared_quota", 40)
	if err == nil {
		t.Fatal("Instance 3 should fail — quota exceeded across instances")
	}

	remaining, err := inst3.ConsumeQuota(ctx, "shared_quota", 20)
	if err != nil {
		t.Fatalf("20 units should work: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("Expected 0 remaining, got %d", remaining)
	}

	for i, inst := range []*RedisQuotaManager{inst1, inst2, inst3} {
		used, limit, err := inst.GetUsage(ctx, "shared_quota")
		if err != nil {
			t.Fatalf("Instance %d GetUsage error: %v", i+1, err)
		}
		if used != 100 {
			t.Errorf("Instance %d: expected used 100, got %d", i+1, used)
		}
		if limit != 100 {
			t.Errorf("Instance %d: expected limit 100, got %d", i+1, limit)
		}
	}
}

func TestDistributed_ConcurrentConsumptionAcrossInstances(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	config := DefaultConfig()
	config.Limits = map[string]int64{"concurrent_quota": 1000}

	numInstances := 5
	instances := make([]*RedisQuotaManager, numInstances)
	for i := 0; i < numInstances; i++ {
		instances[i] = createQuotaManager(t, mr.Addr(), config)
		defer instances[i].Close()
	}

	ctx := createTenantContext(t, "tenant1")

	var wg sync.WaitGroup
	var totalSuccess int64
	var totalFailed int64

	// 5 instances * 50 goroutines * 10 units = 2500 attempted, limit = 1000
	for _, inst := range instances {
		for g := 0; g < 50; g++ {
			wg.Add(1)
			go func(qm *RedisQuotaManager) {
				defer wg.Done()
				_, err := qm.ConsumeQuota(ctx, "concurrent_quota", 10)
				if err != nil {
					atomic.AddInt64(&totalFailed, 1)
				} else {
					atomic.AddInt64(&totalSuccess, 1)
				}
			}(inst)
		}
	}

	wg.Wait()

	if totalSuccess != 100 {
		t.Errorf("Expected exactly 100 successes (1000/10), got %d", totalSuccess)
	}
	if totalFailed != 150 {
		t.Errorf("Expected 150 failures (250-100), got %d", totalFailed)
	}

	used, _, _ := instances[0].GetUsage(ctx, "concurrent_quota")
	if used != 1000 {
		t.Errorf("Expected final usage 1000, got %d", used)
	}
}

func TestDistributed_CheckQuotaConsistency(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	config := DefaultConfig()
	config.Limits = map[string]int64{"check_test": 100}

	inst1 := createQuotaManager(t, mr.Addr(), config)
	defer inst1.Close()
	inst2 := createQuotaManager(t, mr.Addr(), config)
	defer inst2.Close()

	ctx := createTenantContext(t, "tenant1")

	_, err = inst1.ConsumeQuota(ctx, "check_test", 80)
	if err != nil {
		t.Fatalf("Consume error: %v", err)
	}

	allowed, err := inst2.CheckQuota(ctx, "check_test", 30)
	if err != nil {
		t.Fatalf("CheckQuota error: %v", err)
	}
	if allowed {
		t.Fatal("CheckQuota should return false — only 20 remaining but asked for 30")
	}

	allowed, err = inst2.CheckQuota(ctx, "check_test", 20)
	if err != nil {
		t.Fatalf("CheckQuota error: %v", err)
	}
	if !allowed {
		t.Fatal("CheckQuota should return true — exactly 20 remaining")
	}
}

func TestDistributed_SetLimitPropagation(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	config := DefaultConfig()
	config.Limits = map[string]int64{"limit_test": 100}

	inst1 := createQuotaManager(t, mr.Addr(), config)
	defer inst1.Close()
	inst2 := createQuotaManager(t, mr.Addr(), config)
	defer inst2.Close()
	inst3 := createQuotaManager(t, mr.Addr(), config)
	defer inst3.Close()

	ctx := createTenantContext(t, "tenant1")

	err = inst1.SetLimit(ctx, "limit_test", 500)
	if err != nil {
		t.Fatalf("SetLimit error: %v", err)
	}

	_, limit, err := inst2.GetUsage(ctx, "limit_test")
	if err != nil {
		t.Fatalf("GetUsage error: %v", err)
	}
	if limit != 500 {
		t.Fatalf("Instance 2 should see limit 500, got %d", limit)
	}

	remaining, err := inst3.ConsumeQuota(ctx, "limit_test", 400)
	if err != nil {
		t.Fatalf("Should succeed with 500 limit: %v", err)
	}
	if remaining != 100 {
		t.Fatalf("Expected remaining 100, got %d", remaining)
	}

	_, err = inst2.ConsumeQuota(ctx, "limit_test", 200)
	if err == nil {
		t.Fatal("Should fail — only 100 remaining")
	}
}

func TestDistributed_ResetAcrossInstances(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	config := DefaultConfig()
	config.Limits = map[string]int64{"reset_test": 100}

	inst1 := createQuotaManager(t, mr.Addr(), config)
	defer inst1.Close()
	inst2 := createQuotaManager(t, mr.Addr(), config)
	defer inst2.Close()

	ctx := createTenantContext(t, "tenant1")

	_, _ = inst1.ConsumeQuota(ctx, "reset_test", 50)
	_, _ = inst2.ConsumeQuota(ctx, "reset_test", 50)

	_, err = inst1.ConsumeQuota(ctx, "reset_test", 1)
	if err == nil {
		t.Fatal("Quota should be exhausted")
	}

	err = inst2.ResetQuota(ctx, "reset_test")
	if err != nil {
		t.Fatalf("ResetQuota error: %v", err)
	}

	remaining, err := inst1.ConsumeQuota(ctx, "reset_test", 100)
	if err != nil {
		t.Fatalf("Should succeed after reset: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("Expected remaining 0, got %d", remaining)
	}
}

func TestDistributed_MultiTenantIsolation(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	config := DefaultConfig()
	config.Limits = map[string]int64{"isolated_quota": 100}

	inst1 := createQuotaManager(t, mr.Addr(), config)
	defer inst1.Close()
	inst2 := createQuotaManager(t, mr.Addr(), config)
	defer inst2.Close()

	ctxA := createTenantContext(t, "tenantA")
	ctxB := createTenantContext(t, "tenantB")

	_, err = inst1.ConsumeQuota(ctxA, "isolated_quota", 100)
	if err != nil {
		t.Fatalf("TenantA consume error: %v", err)
	}

	remaining, err := inst2.ConsumeQuota(ctxB, "isolated_quota", 50)
	if err != nil {
		t.Fatalf("TenantB should succeed: %v", err)
	}
	if remaining != 50 {
		t.Fatalf("Expected tenantB remaining 50, got %d", remaining)
	}

	_, err = inst2.ConsumeQuota(ctxA, "isolated_quota", 1)
	if err == nil {
		t.Fatal("TenantA should be denied from any instance")
	}
}

func TestDistributed_ConcurrentSetLimitAndConsume(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	config := DefaultConfig()
	config.Limits = map[string]int64{"dynamic_limit": 50}

	inst1 := createQuotaManager(t, mr.Addr(), config)
	defer inst1.Close()
	inst2 := createQuotaManager(t, mr.Addr(), config)
	defer inst2.Close()

	ctx := createTenantContext(t, "tenant1")

	err = inst1.SetLimit(ctx, "dynamic_limit", 1000)
	if err != nil {
		t.Fatalf("SetLimit error: %v", err)
	}

	var wg sync.WaitGroup
	var success int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := inst2.ConsumeQuota(ctx, "dynamic_limit", 10)
			if err == nil {
				atomic.AddInt64(&success, 1)
			}
		}()
	}
	wg.Wait()

	if success != 100 {
		t.Errorf("Expected 100 successes (1000 limit, 10 per goroutine), got %d", success)
	}
}

func TestDistributed_ManyTenantsManyConcurrent(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	config := DefaultConfig()
	config.Limits = map[string]int64{"multi_tenant_quota": 100}

	numInstances := 3
	instances := make([]*RedisQuotaManager, numInstances)
	for i := 0; i < numInstances; i++ {
		instances[i] = createQuotaManager(t, mr.Addr(), config)
		defer instances[i].Close()
	}

	numTenants := 10
	var wg sync.WaitGroup
	results := make([]int64, numTenants)

	for tenant := 0; tenant < numTenants; tenant++ {
		ctx := createTenantContext(t, fmt.Sprintf("tenant-%d", tenant))
		for _, inst := range instances {
			for g := 0; g < 10; g++ {
				wg.Add(1)
				go func(qm *RedisQuotaManager, tenantIdx int) {
					defer wg.Done()
					_, err := qm.ConsumeQuota(ctx, "multi_tenant_quota", 10)
					if err == nil {
						atomic.AddInt64(&results[tenantIdx], 1)
					}
				}(inst, tenant)
			}
		}
	}

	wg.Wait()

	for i, successes := range results {
		if successes != 10 {
			t.Errorf("Tenant-%d: expected 10 successes, got %d", i, successes)
		}
	}
}

// ---------------------------------------------------------------------------
// TTL & Expiration Tests
// ---------------------------------------------------------------------------

func TestTTL_QuotaKeyExpires(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	config := DefaultConfig()
	config.Limits = map[string]int64{"ttl_test": 100}
	config.ResetTTL = 1 * time.Second

	qm, err := NewRedisQuotaManager(client, config)
	if err != nil {
		t.Fatalf("Failed to create quota manager: %v", err)
	}
	defer qm.Close()

	ctx := createTenantContext(t, "tenant1")

	_, err = qm.ConsumeQuota(ctx, "ttl_test", 100)
	if err != nil {
		t.Fatalf("ConsumeQuota error: %v", err)
	}

	_, err = qm.ConsumeQuota(ctx, "ttl_test", 1)
	if err == nil {
		t.Fatal("Expected quota exhausted")
	}

	mr.FastForward(2 * time.Second)

	used, _, err := qm.GetUsage(ctx, "ttl_test")
	if err != nil {
		t.Fatalf("GetUsage error: %v", err)
	}
	if used != 0 {
		t.Fatalf("Expected 0 usage after TTL expiry, got %d", used)
	}
}

// ---------------------------------------------------------------------------
// Constructor Validation Tests
// ---------------------------------------------------------------------------

func TestNewRedisQuotaManager_Validation(t *testing.T) {
	tests := []struct {
		name      string
		client    redis.UniversalClient
		config    Config
		wantError bool
	}{
		{
			name:      "nil client",
			client:    nil,
			config:    DefaultConfig(),
			wantError: true,
		},
		{
			name:      "valid config",
			client:    redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
			config:    DefaultConfig(),
			wantError: false,
		},
		{
			name:   "empty prefix uses default",
			client: redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
			config: Config{
				Prefix:   "",
				Limits:   map[string]int64{},
				ResetTTL: time.Hour,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qm, err := NewRedisQuotaManager(tt.client, tt.config)
			if tt.wantError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if qm != nil {
					qm.Close()
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Health Check Tests
// ---------------------------------------------------------------------------

func TestHealth(t *testing.T) {
	qm, mr := setupTestRedis(t)
	defer qm.Close()

	err := qm.Health(context.Background())
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	mr.Close()

	err = qm.Health(context.Background())
	if err == nil {
		t.Fatal("Expected health check to fail after Redis shutdown")
	}
}

// ---------------------------------------------------------------------------
// Edge Cases
// ---------------------------------------------------------------------------

func TestEdge_ConsumeExactLimit(t *testing.T) {
	qm, mr := setupTestRedis(t)
	defer mr.Close()
	defer qm.Close()

	ctx := createTenantContext(t, "tenant1")

	remaining, err := qm.ConsumeQuota(ctx, "concurrent_test", 100)
	if err != nil {
		t.Fatalf("Consuming exact limit should succeed: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("Expected remaining 0, got %d", remaining)
	}

	_, err = qm.ConsumeQuota(ctx, "concurrent_test", 1)
	if err == nil {
		t.Fatal("Should fail after exact limit consumed")
	}
}

func TestEdge_ZeroAmount(t *testing.T) {
	qm, mr := setupTestRedis(t)
	defer mr.Close()
	defer qm.Close()

	ctx := createTenantContext(t, "tenant1")

	remaining, err := qm.ConsumeQuota(ctx, "api_requests_daily", 0)
	if err != nil {
		t.Fatalf("Zero amount should succeed: %v", err)
	}
	if remaining != 5000 {
		t.Fatalf("Expected remaining 5000, got %d", remaining)
	}
}

func TestEdge_CheckAfterExhaustion(t *testing.T) {
	qm, mr := setupTestRedis(t)
	defer mr.Close()
	defer qm.Close()

	ctx := createTenantContext(t, "tenant1")

	_, _ = qm.ConsumeQuota(ctx, "concurrent_test", 100)

	allowed, err := qm.CheckQuota(ctx, "concurrent_test", 1)
	if err != nil {
		t.Fatalf("CheckQuota error: %v", err)
	}
	if allowed {
		t.Fatal("Should not be allowed after exhaustion")
	}

	allowed, err = qm.CheckQuota(ctx, "concurrent_test", 0)
	if err != nil {
		t.Fatalf("CheckQuota error: %v", err)
	}
	if !allowed {
		t.Fatal("Zero amount should always be allowed")
	}
}
