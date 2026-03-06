package ports

import (
	"context"
	"database/sql"
	"time"
)

// Storage is a port interface for data storage operations.
// It abstracts the underlying storage mechanism (SQL, NoSQL, etc.).
// All queries are automatically tenant-scoped by the enforcer.
type Storage interface {
	// Query executes a SELECT query and returns rows.
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// QueryRow executes a SELECT query that returns a single row.
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row

	// Exec executes an INSERT, UPDATE, or DELETE query.
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// Begin starts a new transaction.
	Begin(ctx context.Context) (StorageTransaction, error)

	// Close closes the storage connection.
	Close() error

	// Health checks if the storage is available.
	Health(ctx context.Context) error
}

// StorageTransaction represents a database transaction.
// It supports the same operations as Storage plus Commit and Rollback.
type StorageTransaction interface {
	Storage

	// Commit commits the transaction.
	Commit() error

	// Rollback rolls back the transaction.
	Rollback() error

	// Done returns a channel that closes when the transaction is done.
	Done() <-chan struct{}
}

// StorageConfig contains configuration for storage adapters.
type StorageConfig struct {
	// ConnectionString is the database connection string.
	ConnectionString string

	// MaxOpenConnections is the maximum number of open connections.
	MaxOpenConnections int

	// MaxIdleConnections is the maximum number of idle connections.
	MaxIdleConnections int

	// ConnMaxLifetime is the maximum lifetime of a connection.
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum idle time for a connection.
	ConnMaxIdleTime time.Duration

	// Timeout is the query timeout.
	Timeout time.Duration
}
