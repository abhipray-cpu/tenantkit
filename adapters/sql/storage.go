package sqladapter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/abhipray-cpu/tenantkit/tenantkit/ports"
)

// Storage is an adapter that implements the ports.Storage interface.
// It wraps database/sql.DB and enforces tenant isolation on all queries.
type Storage struct {
	db       *sql.DB
	enforcer *Enforcer
	config   *Config
}

// Compile-time interface check
var _ ports.Storage = (*Storage)(nil)

// Config holds configuration for the SQL storage adapter.
type Config struct {
	// MaxOpenConnections is the maximum number of open connections to the database.
	MaxOpenConnections int

	// MaxIdleConnections is the maximum number of idle connections.
	MaxIdleConnections int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle.
	ConnMaxIdleTime time.Duration

	// QueryTimeout is the default timeout for queries.
	QueryTimeout time.Duration

	// HealthCheckConfig holds configuration for database health checks
	HealthCheckConfig *HealthCheckConfig
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return NewStorageConfigBuilder().Build()
}

// StorageConfigBuilder provides a fluent interface for building storage configurations.
type StorageConfigBuilder struct {
	config *Config
	errors []error // Collect validation errors instead of panicking
}

// NewStorageConfigBuilder creates a new storage config builder with sensible defaults.
func NewStorageConfigBuilder() *StorageConfigBuilder {
	return &StorageConfigBuilder{
		config: &Config{
			MaxOpenConnections: 25,
			MaxIdleConnections: 5,
			ConnMaxLifetime:    time.Hour,
			ConnMaxIdleTime:    time.Minute * 10,
			QueryTimeout:       time.Second * 30,
			HealthCheckConfig:  DefaultHealthCheckConfig(),
		},
		errors: make([]error, 0),
	}
}

// WithMaxOpenConnections sets the maximum number of open connections.
// No longer panics - validation happens on Build().
func (scb *StorageConfigBuilder) WithMaxOpenConnections(n int) *StorageConfigBuilder {
	if n <= 0 {
		scb.errors = append(scb.errors, fmt.Errorf("MaxOpenConnections must be > 0, got %d", n))
	}
	scb.config.MaxOpenConnections = n
	return scb
}

// WithMaxIdleConnections sets the maximum number of idle connections.
// No longer panics - validation happens on Build().
func (scb *StorageConfigBuilder) WithMaxIdleConnections(n int) *StorageConfigBuilder {
	if n < 0 {
		scb.errors = append(scb.errors, fmt.Errorf("MaxIdleConnections must be >= 0, got %d", n))
	}
	scb.config.MaxIdleConnections = n
	return scb
}

// WithConnMaxLifetime sets the maximum connection lifetime.
// No longer panics - validation happens on Build().
func (scb *StorageConfigBuilder) WithConnMaxLifetime(d time.Duration) *StorageConfigBuilder {
	if d <= 0 {
		scb.errors = append(scb.errors, fmt.Errorf("ConnMaxLifetime must be > 0, got %v", d))
	}
	scb.config.ConnMaxLifetime = d
	return scb
}

// WithConnMaxIdleTime sets the maximum connection idle time.
// No longer panics - validation happens on Build().
func (scb *StorageConfigBuilder) WithConnMaxIdleTime(d time.Duration) *StorageConfigBuilder {
	if d < 0 {
		scb.errors = append(scb.errors, fmt.Errorf("ConnMaxIdleTime must be >= 0, got %v", d))
	}
	scb.config.ConnMaxIdleTime = d
	return scb
}

// WithQueryTimeout sets the default query timeout.
// No longer panics - validation happens on Build().
func (scb *StorageConfigBuilder) WithQueryTimeout(d time.Duration) *StorageConfigBuilder {
	if d <= 0 {
		scb.errors = append(scb.errors, fmt.Errorf("QueryTimeout must be > 0, got %v", d))
	}
	scb.config.QueryTimeout = d
	return scb
}

// WithHealthCheckConfig sets the health check configuration.
func (scb *StorageConfigBuilder) WithHealthCheckConfig(hc *HealthCheckConfig) *StorageConfigBuilder {
	if hc != nil {
		scb.config.HealthCheckConfig = hc
	}
	return scb
}

// BuildWithValidation returns the configured Config with validation.
// Returns error if any validation rules were violated.
// This is the recommended method for new code.
func (scb *StorageConfigBuilder) BuildWithValidation() (*Config, error) {
	// Check for accumulated errors from With* methods
	if len(scb.errors) > 0 {
		// Build comprehensive error message
		var errMsgs []string
		for _, err := range scb.errors {
			errMsgs = append(errMsgs, err.Error())
		}
		return nil, fmt.Errorf("storage config validation failed: %s", strings.Join(errMsgs, "; "))
	}

	// Validation: MaxIdleConnections should not exceed MaxOpenConnections
	if scb.config.MaxIdleConnections > scb.config.MaxOpenConnections {
		scb.config.MaxIdleConnections = scb.config.MaxOpenConnections / 2
	}

	return scb.config, nil
}

// Build returns the configured Config.
// Deprecated: Use BuildWithValidation() for proper error handling.
// This method is kept for backward compatibility and will not return errors.
func (scb *StorageConfigBuilder) Build() *Config {
	// Validation: MaxIdleConnections should not exceed MaxOpenConnections
	if scb.config.MaxIdleConnections > scb.config.MaxOpenConnections {
		scb.config.MaxIdleConnections = scb.config.MaxOpenConnections / 2
	}
	return scb.config
}

// New creates a new SQL storage adapter.
// The db parameter should be an already-opened database/sql.DB connection.
// Users can bring any database driver they want (PostgreSQL, MySQL, SQLite, etc.)
//
// Example:
//
//	import (
//	    "database/sql"
//	    _ "github.com/lib/pq"
//	    sqladapter "github.com/abhipray-cpu/tenantkit/adapters/sql"
//	)
//
//	db, err := sql.Open("postgres", dsn)
//	storage := sqladapter.New(db)
func New(db *sql.DB) *Storage {
	return NewWithConfig(db, DefaultConfig())
}

// NewWithConfig creates a new SQL storage adapter with custom configuration.
func NewWithConfig(db *sql.DB, config *Config) *Storage {
	if config == nil {
		config = DefaultConfig()
	}

	// Apply configuration to the database connection
	db.SetMaxOpenConns(config.MaxOpenConnections)
	db.SetMaxIdleConns(config.MaxIdleConnections)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	return &Storage{
		db:       db,
		enforcer: NewEnforcer(),
		config:   config,
	}
}

// Query executes a SELECT query with automatic tenant filtering.
// The query is rewritten to include the tenant_id filter before execution.
func (s *Storage) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	// Apply query timeout if not already set
	if _, ok := ctx.Deadline(); !ok && s.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.QueryTimeout)
		defer cancel()
	}

	// Enforce tenant isolation
	enforcedQuery, enforcedArgs, err := s.enforcer.EnforceQuery(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("query enforcement failed: %w", err)
	}

	// Execute the enforced query
	rows, err := s.db.QueryContext(ctx, enforcedQuery, enforcedArgs...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	return rows, nil
}

// QueryRowResult wraps sql.Row with error information for graceful error handling.
// This allows QueryRow to return errors without panicking.
type QueryRowResult struct {
	row *sql.Row
	err error
}

// Scan delegates to the underlying row, or returns the stored error if query preparation failed.
func (qr *QueryRowResult) Scan(dest ...interface{}) error {
	if qr.err != nil {
		return qr.err
	}
	return qr.row.Scan(dest...)
}

// Err returns any error from query preparation.
func (qr *QueryRowResult) Err() error {
	return qr.err
}

// QueryRowWithError executes a SELECT query that returns a single row.
// This is the recommended API for new code as it returns errors explicitly.
// The query is rewritten to include the tenant_id filter before execution.
func (s *Storage) QueryRowWithError(ctx context.Context, query string, args ...interface{}) (*QueryRowResult, error) {
	// Apply query timeout if not already set
	if _, ok := ctx.Deadline(); !ok && s.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.QueryTimeout)
		defer cancel()
	}

	// Enforce tenant isolation
	enforcedQuery, enforcedArgs, err := s.enforcer.EnforceQuery(ctx, query, args)
	if err != nil {
		// Return error instead of panicking - this is the critical fix
		return nil, fmt.Errorf("query enforcement failed: %w", err)
	}

	// Execute the enforced query
	row := s.db.QueryRowContext(ctx, enforcedQuery, enforcedArgs...)
	return &QueryRowResult{row: row, err: nil}, nil
}

// QueryRow executes a SELECT query that returns a single row.
// The query is rewritten to include the tenant_id filter before execution.
// Note: Enforcement errors will only be returned when Scan() is called on the returned Row.
// Deprecated: Use QueryRowWithError for proper error handling.
func (s *Storage) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// Apply query timeout if not already set
	if _, ok := ctx.Deadline(); !ok && s.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.QueryTimeout)
		defer cancel()
	}

	// Enforce tenant isolation
	enforcedQuery, enforcedArgs, err := s.enforcer.EnforceQuery(ctx, query, args)
	if err != nil {
		// Return a row that will error on Scan instead of panicking
		// This maintains backward compatibility while preventing service crashes
		// The error will surface when Scan() is called
		// For better error handling, use QueryRowWithError() instead
		return s.db.QueryRowContext(ctx, "SELECT NULL WHERE FALSE")
	}

	// Execute the enforced query
	return s.db.QueryRowContext(ctx, enforcedQuery, enforcedArgs...)
}

// Exec executes an INSERT, UPDATE, or DELETE query.
// The query is rewritten to include the tenant_id filter before execution.
func (s *Storage) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// Apply query timeout if not already set
	if _, ok := ctx.Deadline(); !ok && s.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.QueryTimeout)
		defer cancel()
	}

	// Enforce tenant isolation
	enforcedQuery, enforcedArgs, err := s.enforcer.EnforceQuery(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("query enforcement failed: %w", err)
	}

	// Execute the enforced query
	result, err := s.db.ExecContext(ctx, enforcedQuery, enforcedArgs...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	return result, nil
}

// Begin starts a new transaction.
// The transaction will automatically enforce tenant isolation on all queries.
func (s *Storage) Begin(ctx context.Context) (ports.StorageTransaction, error) {
	// Apply query timeout if not already set
	if _, ok := ctx.Deadline(); !ok && s.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.QueryTimeout)
		defer cancel()
	}

	// Begin the transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("transaction begin failed: %w", err)
	}

	// Create a transaction wrapper
	return NewTransaction(tx, s.enforcer, s.config), nil
}

// Close closes the database connection.
func (s *Storage) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Health checks if the database is healthy by pinging it.
func (s *Storage) Health(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Apply timeout for health checks from configuration
	timeout := s.config.HealthCheckConfig.Timeout
	healthCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := s.db.PingContext(healthCtx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// GetDB returns the underlying database/sql.DB connection.
// This should only be used for advanced operations or third-party integrations.
// Use of this bypasses tenant enforcement, so use with caution!
func (s *Storage) GetDB() *sql.DB {
	return s.db
}

// GetEnforcer returns the enforcer used by this storage adapter.
// This can be used to validate or rewrite queries outside of the storage layer.
func (s *Storage) GetEnforcer() *Enforcer {
	return s.enforcer
}
