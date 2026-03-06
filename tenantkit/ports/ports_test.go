package ports

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	"github.com/abhipray-cpu/tenantkit/domain"
)

// Use domain errors
var (
	// Alias domain errors for test readability
	ErrMissingTenantID = domain.ErrMissingTenantID
)

// MockResolver implements Resolver interface for testing
type MockResolver struct {
	tenantID string
	err      error
}

func (m *MockResolver) Resolve(r *http.Request) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.tenantID, nil
}

// Test Resolver interface
func TestResolverInterface(t *testing.T) {
	t.Run("resolver implemented correctly", func(t *testing.T) {
		resolver := &MockResolver{tenantID: "test-tenant"}
		req, _ := http.NewRequest("GET", "http://example.com", nil)

		id, err := resolver.Resolve(req)
		if err != nil {
			t.Errorf("Resolve() error = %v", err)
		}
		if id != "test-tenant" {
			t.Errorf("Resolve() = %v, want test-tenant", id)
		}
	})

	t.Run("resolver error handling", func(t *testing.T) {
		resolver := &MockResolver{err: ErrMissingTenantID}
		req, _ := http.NewRequest("GET", "http://example.com", nil)

		_, err := resolver.Resolve(req)
		if err == nil {
			t.Error("Resolve() should return error")
		}
	})
}

// MockEnforcer implements Enforcer interface for testing
type MockEnforcer struct {
	err error
}

func (m *MockEnforcer) EnforceQuery(ctx context.Context, query string, args []interface{}) (string, []interface{}, error) {
	if m.err != nil {
		return "", nil, m.err
	}
	return query + " WHERE tenant_id = ?", append(args, "test-tenant"), nil
}

func (m *MockEnforcer) ValidateQuery(ctx context.Context, query string) error {
	return m.err
}

func (m *MockEnforcer) SupportedOperations() []string {
	return []string{"SELECT", "INSERT", "UPDATE", "DELETE"}
}

// Test Enforcer interface
func TestEnforcerInterface(t *testing.T) {
	t.Run("enforcer rewrite query", func(t *testing.T) {
		enforcer := &MockEnforcer{}
		ctx := context.Background()

		query, args, err := enforcer.EnforceQuery(ctx, "SELECT * FROM users", []interface{}{})
		if err != nil {
			t.Errorf("EnforceQuery() error = %v", err)
		}
		if query != "SELECT * FROM users WHERE tenant_id = ?" {
			t.Errorf("EnforceQuery() query = %v, want rewritten", query)
		}
		if len(args) != 1 {
			t.Errorf("EnforceQuery() args = %v, want 1 arg", len(args))
		}
	})

	t.Run("enforcer supported operations", func(t *testing.T) {
		enforcer := &MockEnforcer{}
		ops := enforcer.SupportedOperations()
		if len(ops) != 4 {
			t.Errorf("SupportedOperations() = %v, want 4 operations", len(ops))
		}
	})
}

// MockStorage implements Storage interface for testing
type MockStorage struct {
	err error
}

func (m *MockStorage) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, m.err
}

func (m *MockStorage) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

func (m *MockStorage) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, m.err
}

func (m *MockStorage) Begin(ctx context.Context) (StorageTransaction, error) {
	return &MockTransaction{}, m.err
}

func (m *MockStorage) Close() error {
	return m.err
}

func (m *MockStorage) Health(ctx context.Context) error {
	return m.err
}

// MockTransaction implements StorageTransaction interface
type MockTransaction struct {
	closed bool
}

func (m *MockTransaction) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *MockTransaction) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

func (m *MockTransaction) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (m *MockTransaction) Begin(ctx context.Context) (StorageTransaction, error) {
	return nil, nil
}

func (m *MockTransaction) Close() error {
	return nil
}

func (m *MockTransaction) Health(ctx context.Context) error {
	return nil
}

func (m *MockTransaction) Commit() error {
	m.closed = true
	return nil
}

func (m *MockTransaction) Rollback() error {
	m.closed = true
	return nil
}

func (m *MockTransaction) Done() <-chan struct{} {
	return make(chan struct{})
}

// Test Storage interface
func TestStorageInterface(t *testing.T) {
	t.Run("storage methods exist", func(t *testing.T) {
		storage := &MockStorage{}
		ctx := context.Background()

		// Test that all methods exist and are callable
		_, _ = storage.Query(ctx, "SELECT *")
		_ = storage.QueryRow(ctx, "SELECT *")
		_, _ = storage.Exec(ctx, "INSERT")
		_, _ = storage.Begin(ctx)
		_ = storage.Close()
		_ = storage.Health(ctx)
	})
}

// MockMetrics implements Metrics interface for testing
type MockMetrics struct {
	events []string
}

func (m *MockMetrics) RecordRequest(ctx context.Context, method string, endpoint string, statusCode int, durationMS int64) error {
	m.events = append(m.events, "request")
	return nil
}

func (m *MockMetrics) RecordQuery(ctx context.Context, query string, rowsAffected int64, durationMS int64) error {
	m.events = append(m.events, "query")
	return nil
}

func (m *MockMetrics) RecordError(ctx context.Context, errorType string, message string) error {
	m.events = append(m.events, "error")
	return nil
}

func (m *MockMetrics) RecordQuotaUsage(ctx context.Context, quotaType string, used int64, limit int64) error {
	m.events = append(m.events, "quota")
	return nil
}

func (m *MockMetrics) RecordRateLimit(ctx context.Context, endpoint string, limited bool) error {
	m.events = append(m.events, "ratelimit")
	return nil
}

func (m *MockMetrics) RecordCacheHit(ctx context.Context, key string, hit bool) error {
	m.events = append(m.events, "cache")
	return nil
}

// Test Metrics interface
func TestMetricsInterface(t *testing.T) {
	t.Run("metrics recording", func(t *testing.T) {
		metrics := &MockMetrics{}
		ctx := context.Background()

		_ = metrics.RecordRequest(ctx, "GET", "/api/users", 200, 50)
		_ = metrics.RecordQuery(ctx, "SELECT", 10, 25)
		_ = metrics.RecordError(ctx, "timeout", "query timeout")
		_ = metrics.RecordQuotaUsage(ctx, "api_calls", 100, 1000)

		if len(metrics.events) != 4 {
			t.Errorf("events recorded = %d, want 4", len(metrics.events))
		}
	})
}

// MockLimiter implements Limiter interface for testing
type MockLimiter struct {
	allowed bool
	err     error
}

func (m *MockLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return m.allowed, m.err
}

func (m *MockLimiter) AllowN(ctx context.Context, key string, n int64) (bool, error) {
	return m.allowed, m.err
}

func (m *MockLimiter) Remaining(ctx context.Context, key string) (int64, error) {
	return 100, m.err
}

func (m *MockLimiter) Reset(ctx context.Context, key string) error {
	return m.err
}

func (m *MockLimiter) Health(ctx context.Context) error {
	return m.err
}

// Test Limiter interface
func TestLimiterInterface(t *testing.T) {
	t.Run("limiter allow", func(t *testing.T) {
		limiter := &MockLimiter{allowed: true}
		ctx := context.Background()

		allowed, err := limiter.Allow(ctx, "user-123")
		if err != nil {
			t.Errorf("Allow() error = %v", err)
		}
		if !allowed {
			t.Error("Allow() = false, want true")
		}
	})

	t.Run("limiter remaining", func(t *testing.T) {
		limiter := &MockLimiter{}
		ctx := context.Background()

		remaining, _ := limiter.Remaining(ctx, "user-123")
		if remaining != 100 {
			t.Errorf("Remaining() = %d, want 100", remaining)
		}
	})
}

// MockQuotaManager implements QuotaManager interface for testing
type MockQuotaManager struct {
	used  int64
	limit int64
}

func (m *MockQuotaManager) CheckQuota(ctx context.Context, quotaType string, amount int64) (bool, error) {
	return m.used+amount <= m.limit, nil
}

func (m *MockQuotaManager) ConsumeQuota(ctx context.Context, quotaType string, amount int64) (int64, error) {
	m.used += amount
	return m.limit - m.used, nil
}

func (m *MockQuotaManager) GetUsage(ctx context.Context, quotaType string) (int64, int64, error) {
	return m.used, m.limit, nil
}

func (m *MockQuotaManager) ResetQuota(ctx context.Context, quotaType string) error {
	m.used = 0
	return nil
}

func (m *MockQuotaManager) SetLimit(ctx context.Context, quotaType string, limit int64) error {
	m.limit = limit
	return nil
}

// Test QuotaManager interface
func TestQuotaManagerInterface(t *testing.T) {
	t.Run("quota check and consume", func(t *testing.T) {
		qm := &MockQuotaManager{used: 0, limit: 1000}
		ctx := context.Background()

		allowed, _ := qm.CheckQuota(ctx, "api_calls", 100)
		if !allowed {
			t.Error("CheckQuota() = false, want true")
		}

		remaining, _ := qm.ConsumeQuota(ctx, "api_calls", 100)
		if remaining != 900 {
			t.Errorf("ConsumeQuota() remaining = %d, want 900", remaining)
		}
	})

	t.Run("quota reset", func(t *testing.T) {
		qm := &MockQuotaManager{used: 500, limit: 1000}
		ctx := context.Background()

		_ = qm.ResetQuota(ctx, "api_calls")

		if qm.used != 0 {
			t.Errorf("used after Reset() = %d, want 0", qm.used)
		}
	})
}
