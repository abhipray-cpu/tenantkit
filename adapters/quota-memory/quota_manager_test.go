package quota

import (
	"context"
	"sync"
	"testing"

	"github.com/abhipray-cpu/tenantkit/domain"
)

// Helper function to create a test context with tenant ID
func createTestContext(t *testing.T, tenantID string) context.Context {
	t.Helper()
	tenantCtx, err := domain.NewContext(tenantID, "user1", "req1")
	if err != nil {
		t.Fatalf("failed to create context: %v", err)
	}

	return tenantCtx.ToGoContext(context.Background())
}

func TestNewInMemoryQuotaManager(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	if qm == nil {
		t.Fatal("NewInMemoryQuotaManager returned nil")
	}

	// Verify default limits are set
	expectedLimits := map[string]int64{
		"api_requests_monthly": 100000,
		"api_requests_daily":   5000,
		"database_rows":        1000000,
		"storage_bytes":        1073741824,
	}

	for quotaType, expectedLimit := range expectedLimits {
		if qm.limits[quotaType] != expectedLimit {
			t.Errorf("expected limit for %s to be %d, got %d", quotaType, expectedLimit, qm.limits[quotaType])
		}
	}
}

func TestCheckQuota_NewTenant(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	ctx := createTestContext(t, "tenant1")

	t.Run("check quota with no usage", func(t *testing.T) {
		allowed, err := qm.CheckQuota(ctx, "api_requests_daily", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Error("expected quota to be allowed")
		}
	})

	t.Run("check with negative amount", func(t *testing.T) {
		allowed, err := qm.CheckQuota(ctx, "api_requests_daily", -100)
		if err == nil {
			t.Error("expected error for negative amount")
		}
		if allowed {
			t.Error("expected quota to not be allowed")
		}
	})

	t.Run("check unknown quota type", func(t *testing.T) {
		allowed, err := qm.CheckQuota(ctx, "unknown_quota", 100)
		if err == nil {
			t.Error("expected error for unknown quota type")
		}
		if allowed {
			t.Error("expected quota to not be allowed")
		}
	})
}

func TestConsumeQuota_Basic(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	ctx := createTestContext(t, "tenant1")

	t.Run("consume quota successfully", func(t *testing.T) {
		remaining, err := qm.ConsumeQuota(ctx, "api_requests_daily", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedRemaining := int64(4900) // 5000 - 100
		if remaining != expectedRemaining {
			t.Errorf("expected remaining %d, got %d", expectedRemaining, remaining)
		}
	})

	t.Run("consume more quota", func(t *testing.T) {
		remaining, err := qm.ConsumeQuota(ctx, "api_requests_daily", 200)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedRemaining := int64(4700) // 4900 - 200
		if remaining != expectedRemaining {
			t.Errorf("expected remaining %d, got %d", expectedRemaining, remaining)
		}
	})

	t.Run("consume with negative amount", func(t *testing.T) {
		_, err := qm.ConsumeQuota(ctx, "api_requests_daily", -50)
		if err == nil {
			t.Error("expected error for negative amount")
		}
	})
}

func TestConsumeQuota_ExceedLimit(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	ctx := createTestContext(t, "tenant1")

	t.Run("exceed quota limit", func(t *testing.T) {
		// Consume most of the quota
		_, err := qm.ConsumeQuota(ctx, "api_requests_daily", 4900)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Try to consume more than remaining (100 left, trying 200)
		_, err = qm.ConsumeQuota(ctx, "api_requests_daily", 200)
		if err != domain.ErrQuotaExceeded {
			t.Errorf("expected ErrQuotaExceeded, got %v", err)
		}
	})
}

func TestGetUsage(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	ctx := createTestContext(t, "tenant1")

	t.Run("get usage with no consumption", func(t *testing.T) {
		used, limit, err := qm.GetUsage(ctx, "api_requests_daily")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if used != 0 {
			t.Errorf("expected used to be 0, got %d", used)
		}
		if limit != 5000 {
			t.Errorf("expected limit to be 5000, got %d", limit)
		}
	})

	t.Run("get usage after consumption", func(t *testing.T) {
		_, err := qm.ConsumeQuota(ctx, "api_requests_daily", 1000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		used, limit, err := qm.GetUsage(ctx, "api_requests_daily")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if used != 1000 {
			t.Errorf("expected used to be 1000, got %d", used)
		}
		if limit != 5000 {
			t.Errorf("expected limit to be 5000, got %d", limit)
		}
	})
}

func TestResetQuota(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	ctx := createTestContext(t, "tenant1")

	t.Run("reset quota after consumption", func(t *testing.T) {
		// Consume some quota
		_, err := qm.ConsumeQuota(ctx, "api_requests_daily", 2000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify consumption
		used, _, err := qm.GetUsage(ctx, "api_requests_daily")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if used != 2000 {
			t.Errorf("expected used to be 2000, got %d", used)
		}

		// Reset quota
		err = qm.ResetQuota(ctx, "api_requests_daily")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify reset
		used, _, err = qm.GetUsage(ctx, "api_requests_daily")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if used != 0 {
			t.Errorf("expected used to be 0 after reset, got %d", used)
		}
	})

	t.Run("reset non-existent quota", func(t *testing.T) {
		ctx2 := createTestContext(t, "tenant2")
		err := qm.ResetQuota(ctx2, "api_requests_daily")
		if err != nil {
			t.Errorf("expected no error when resetting non-existent quota, got %v", err)
		}
	})
}

func TestSetLimit(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	ctx := createTestContext(t, "tenant1")

	t.Run("set new limit", func(t *testing.T) {
		err := qm.SetLimit(ctx, "api_requests_daily", 10000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify new limit
		_, limit, err := qm.GetUsage(ctx, "api_requests_daily")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if limit != 10000 {
			t.Errorf("expected limit to be 10000, got %d", limit)
		}
	})

	t.Run("set limit with existing usage", func(t *testing.T) {
		// Consume some quota
		_, err := qm.ConsumeQuota(ctx, "api_requests_daily", 1000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Set new limit
		err = qm.SetLimit(ctx, "api_requests_daily", 20000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify limit changed but usage remained
		used, limit, err := qm.GetUsage(ctx, "api_requests_daily")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if used != 1000 {
			t.Errorf("expected used to be 1000, got %d", used)
		}
		if limit != 20000 {
			t.Errorf("expected limit to be 20000, got %d", limit)
		}
	})

	t.Run("set invalid limit", func(t *testing.T) {
		err := qm.SetLimit(ctx, "api_requests_daily", 0)
		if err == nil {
			t.Error("expected error for zero limit")
		}

		err = qm.SetLimit(ctx, "api_requests_daily", -100)
		if err == nil {
			t.Error("expected error for negative limit")
		}
	})
}

func TestMultipleTenants(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	ctx1 := createTestContext(t, "tenant1")
	ctx2 := createTestContext(t, "tenant2")

	t.Run("tenants have isolated quotas", func(t *testing.T) {
		// Tenant 1 consumes quota
		_, err := qm.ConsumeQuota(ctx1, "api_requests_daily", 1000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Tenant 2 consumes quota
		_, err = qm.ConsumeQuota(ctx2, "api_requests_daily", 2000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify tenant 1 usage
		used1, _, err := qm.GetUsage(ctx1, "api_requests_daily")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if used1 != 1000 {
			t.Errorf("expected tenant1 used to be 1000, got %d", used1)
		}

		// Verify tenant 2 usage
		used2, _, err := qm.GetUsage(ctx2, "api_requests_daily")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if used2 != 2000 {
			t.Errorf("expected tenant2 used to be 2000, got %d", used2)
		}
	})
}

func TestBulkResetQuotas(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	ctx := createTestContext(t, "tenant1")

	t.Run("reset multiple quotas", func(t *testing.T) {
		// Consume multiple quotas
		_, err := qm.ConsumeQuota(ctx, "api_requests_daily", 1000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = qm.ConsumeQuota(ctx, "api_requests_monthly", 5000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Bulk reset
		quotaTypes := []string{"api_requests_daily", "api_requests_monthly"}
		err = qm.BulkResetQuotas(ctx, quotaTypes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify both are reset
		used1, _, _ := qm.GetUsage(ctx, "api_requests_daily")
		used2, _, _ := qm.GetUsage(ctx, "api_requests_monthly")

		if used1 != 0 {
			t.Errorf("expected api_requests_daily to be 0, got %d", used1)
		}
		if used2 != 0 {
			t.Errorf("expected api_requests_monthly to be 0, got %d", used2)
		}
	})
}

func TestGetAllQuotas(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	ctx := createTestContext(t, "tenant1")

	t.Run("get all quotas with no usage", func(t *testing.T) {
		quotas, err := qm.GetAllQuotas(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(quotas) != 4 {
			t.Errorf("expected 4 quota types, got %d", len(quotas))
		}

		// All should have 0 usage
		for quotaType, values := range quotas {
			if values[0] != 0 {
				t.Errorf("expected %s usage to be 0, got %d", quotaType, values[0])
			}
		}
	})

	t.Run("get all quotas with usage", func(t *testing.T) {
		// Consume some quotas
		_, err := qm.ConsumeQuota(ctx, "api_requests_daily", 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = qm.ConsumeQuota(ctx, "storage_bytes", 1000000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		quotas, err := qm.GetAllQuotas(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify specific quotas
		if quotas["api_requests_daily"][0] != 100 {
			t.Errorf("expected api_requests_daily usage to be 100, got %d", quotas["api_requests_daily"][0])
		}
		if quotas["storage_bytes"][0] != 1000000 {
			t.Errorf("expected storage_bytes usage to be 1000000, got %d", quotas["storage_bytes"][0])
		}
	})
}

func TestGetStats(t *testing.T) {
	qm := NewInMemoryQuotaManager()

	t.Run("stats with no usage", func(t *testing.T) {
		stats := qm.GetStats()

		if stats.TotalQuotaTypes != 4 {
			t.Errorf("expected 4 quota types, got %d", stats.TotalQuotaTypes)
		}
		if stats.ActiveQuotas != 0 {
			t.Errorf("expected 0 active quotas, got %d", stats.ActiveQuotas)
		}
		if stats.TotalUsage != 0 {
			t.Errorf("expected 0 total usage, got %d", stats.TotalUsage)
		}
	})

	t.Run("stats with usage", func(t *testing.T) {
		ctx1 := createTestContext(t, "tenant1")
		ctx2 := createTestContext(t, "tenant2")

		// Multiple tenants consume quota
		qm.ConsumeQuota(ctx1, "api_requests_daily", 1000)
		qm.ConsumeQuota(ctx2, "api_requests_daily", 500)

		stats := qm.GetStats()

		if stats.ActiveQuotas != 2 {
			t.Errorf("expected 2 active quotas, got %d", stats.ActiveQuotas)
		}
		if stats.TotalUsage != 1500 {
			t.Errorf("expected 1500 total usage, got %d", stats.TotalUsage)
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	ctx := createTestContext(t, "tenant1")

	t.Run("concurrent consumption", func(t *testing.T) {
		var wg sync.WaitGroup
		goroutines := 10
		consumePerGoroutine := int64(100)

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := qm.ConsumeQuota(ctx, "api_requests_monthly", consumePerGoroutine)
				if err != nil && err != domain.ErrQuotaExceeded {
					t.Errorf("unexpected error: %v", err)
				}
			}()
		}

		wg.Wait()

		// Verify total usage
		used, _, err := qm.GetUsage(ctx, "api_requests_monthly")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedUsed := int64(goroutines) * consumePerGoroutine
		if used != expectedUsed {
			t.Errorf("expected used to be %d, got %d", expectedUsed, used)
		}
	})

	t.Run("concurrent check and consume", func(t *testing.T) {
		ctx2 := createTestContext(t, "tenant2")
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				allowed, _ := qm.CheckQuota(ctx2, "api_requests_daily", 100)
				if allowed {
					_, err := qm.ConsumeQuota(ctx2, "api_requests_daily", 100)
					if err == nil {
						mu.Lock()
						successCount++
						mu.Unlock()
					}
				}
			}()
		}

		wg.Wait()

		// Should be able to consume at most 50 times (5000 / 100)
		if successCount > 50 {
			t.Errorf("expected at most 50 successful consumptions, got %d", successCount)
		}
	})
}

func TestQuotaTypes(t *testing.T) {
	qm := NewInMemoryQuotaManager()
	ctx := createTestContext(t, "tenant1")

	quotaTypes := []string{
		"api_requests_monthly",
		"api_requests_daily",
		"database_rows",
		"storage_bytes",
	}

	for _, quotaType := range quotaTypes {
		t.Run(quotaType, func(t *testing.T) {
			// Should be able to check quota
			allowed, err := qm.CheckQuota(ctx, quotaType, 1)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", quotaType, err)
			}
			if !allowed {
				t.Errorf("expected %s to be allowed", quotaType)
			}

			// Should be able to consume quota
			_, err = qm.ConsumeQuota(ctx, quotaType, 1)
			if err != nil {
				t.Fatalf("unexpected error consuming %s: %v", quotaType, err)
			}
		})
	}
}
