package sqladapter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/abhipray-cpu/tenantkit/tenantkit/ports"
)

// Transaction is an adapter that implements the ports.StorageTransaction interface.
// It wraps sql.Tx and enforces tenant isolation on all queries within the transaction.
type Transaction struct {
	tx       *sql.Tx
	enforcer *Enforcer
	config   *Config
	done     chan struct{}
}

// NewTransaction creates a new transaction wrapper.
func NewTransaction(tx *sql.Tx, enforcer *Enforcer, config *Config) *Transaction {
	if config == nil {
		config = DefaultConfig()
	}

	return &Transaction{
		tx:       tx,
		enforcer: enforcer,
		config:   config,
		done:     make(chan struct{}),
	}
}

// Query executes a SELECT query within the transaction.
// The query is rewritten to include tenant_id filtering.
func (t *Transaction) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	// Apply query timeout if not already set
	if _, ok := ctx.Deadline(); !ok && t.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.config.QueryTimeout)
		defer cancel()
	}

	// Enforce tenant isolation
	enforcedQuery, enforcedArgs, err := t.enforcer.EnforceQuery(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("query enforcement failed: %w", err)
	}

	// Execute the enforced query within the transaction
	rows, err := t.tx.QueryContext(ctx, enforcedQuery, enforcedArgs...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	return rows, nil
}

// QueryRowWithError executes a query that is expected to return at most one row within the transaction.
// Returns error explicitly instead of panicking on enforcement failure.
// This is the recommended API for new code.
func (t *Transaction) QueryRowWithError(ctx context.Context, query string, args ...interface{}) (*QueryRowResult, error) {
	// Apply query timeout if not already set
	if _, ok := ctx.Deadline(); !ok && t.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.config.QueryTimeout)
		defer cancel()
	}

	// Enforce tenant isolation
	enforcedQuery, enforcedArgs, err := t.enforcer.EnforceQuery(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("query enforcement failed: %w", err)
	}

	// Execute the enforced query within the transaction
	row := t.tx.QueryRowContext(ctx, enforcedQuery, enforcedArgs...)
	return &QueryRowResult{row: row, err: nil}, nil
}

// QueryRow executes a SELECT query that returns a single row within the transaction.
// The query is rewritten to include tenant_id filtering.
// Deprecated: Use QueryRowWithError for proper error handling.
// This method is kept for backward compatibility and will not panic.
func (t *Transaction) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// Apply query timeout if not already set
	if _, ok := ctx.Deadline(); !ok && t.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.config.QueryTimeout)
		defer cancel()
	}

	// Enforce tenant isolation
	enforcedQuery, enforcedArgs, err := t.enforcer.EnforceQuery(ctx, query, args)
	if err != nil {
		// Return a row that will error on Scan instead of panicking
		// This maintains backward compatibility while being non-invasive
		// Create a dummy query that returns no rows
		return t.tx.QueryRowContext(ctx, "SELECT NULL WHERE FALSE")
	}

	// Execute the enforced query within the transaction
	return t.tx.QueryRowContext(ctx, enforcedQuery, enforcedArgs...)
}

// Exec executes an INSERT, UPDATE, or DELETE query within the transaction.
// The query is rewritten to include tenant_id filtering.
func (t *Transaction) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// Apply query timeout if not already set
	if _, ok := ctx.Deadline(); !ok && t.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.config.QueryTimeout)
		defer cancel()
	}

	// Enforce tenant isolation
	enforcedQuery, enforcedArgs, err := t.enforcer.EnforceQuery(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("query enforcement failed: %w", err)
	}

	// Execute the enforced query within the transaction
	result, err := t.tx.ExecContext(ctx, enforcedQuery, enforcedArgs...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	return result, nil
}

// Begin is not supported for transactions (can't start a transaction within a transaction).
func (t *Transaction) Begin(ctx context.Context) (ports.StorageTransaction, error) {
	return nil, fmt.Errorf("nested transactions are not supported")
}

// Close is not supported for transactions. Use Commit or Rollback instead.
func (t *Transaction) Close() error {
	return fmt.Errorf("cannot close transaction directly; use Commit or Rollback")
}

// Health is not typically called on transactions, but we support it.
func (t *Transaction) Health(ctx context.Context) error {
	// Transactions don't have a health check, but we can verify the transaction is still valid
	if t.tx == nil {
		return fmt.Errorf("transaction is closed")
	}
	return nil
}

// Commit commits the transaction.
func (t *Transaction) Commit() error {
	if t.tx == nil {
		return fmt.Errorf("transaction is already closed")
	}

	err := t.tx.Commit()
	close(t.done)
	return err
}

// Rollback rolls back the transaction.
func (t *Transaction) Rollback() error {
	if t.tx == nil {
		return fmt.Errorf("transaction is already closed")
	}

	err := t.tx.Rollback()
	close(t.done)
	return err
}

// Done returns a channel that closes when the transaction is done (committed or rolled back).
func (t *Transaction) Done() <-chan struct{} {
	return t.done
}

// Ensure Transaction implements StorageTransaction interface
var _ ports.StorageTransaction = (*Transaction)(nil)
