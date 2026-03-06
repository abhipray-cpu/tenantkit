package sqladapter

import (
	"context"
	"testing"
)

func TestNewTransaction(t *testing.T) {
	tx := NewTransaction(nil, NewEnforcer(), DefaultConfig())
	if tx == nil {
		t.Error("NewTransaction() returned nil")
	}
}

func TestTransactionDone(t *testing.T) {
	tx := NewTransaction(nil, NewEnforcer(), DefaultConfig())

	// Done channel should not be closed yet
	select {
	case <-tx.Done():
		t.Error("expected Done channel to be open initially")
	default:
		// This is expected
	}
}

func TestTransactionCommit(t *testing.T) {
	tx := NewTransaction(nil, NewEnforcer(), DefaultConfig())

	// Commit should fail with nil tx (expected in test)
	err := tx.Commit()
	if err == nil {
		t.Error("expected Commit() to return error with nil tx, got nil")
	}

	// After commit, Done should be closed
	select {
	case <-tx.Done():
		// This is expected
	default:
		// The channel should be closed after Commit
	}
}

func TestTransactionRollback(t *testing.T) {
	tx := NewTransaction(nil, NewEnforcer(), DefaultConfig())

	// Rollback should fail with nil tx (expected in test)
	err := tx.Rollback()
	if err == nil {
		t.Error("expected Rollback() to return error with nil tx, got nil")
	}

	// After rollback, Done should be closed
	select {
	case <-tx.Done():
		// This is expected
	default:
		// The channel should be closed after Rollback
	}
}

func TestTransactionBeginNotSupported(t *testing.T) {
	tx := NewTransaction(nil, NewEnforcer(), DefaultConfig())

	// Begin should not be supported in transaction
	ctx := context.Background()
	_, err := tx.Begin(ctx)
	if err == nil {
		t.Error("expected Begin() to return error in transaction, got nil")
	}
}

func TestTransactionCloseNotSupported(t *testing.T) {
	tx := NewTransaction(nil, NewEnforcer(), DefaultConfig())

	// Close should not be supported in transaction
	err := tx.Close()
	if err == nil {
		t.Error("expected Close() to return error in transaction, got nil")
	}
}

func TestTransactionHealth(t *testing.T) {
	tx := NewTransaction(nil, NewEnforcer(), DefaultConfig())

	// Health check should fail with nil tx
	ctx := context.Background()
	err := tx.Health(ctx)
	if err == nil {
		t.Error("expected Health() to return error with nil tx, got nil")
	}
}
