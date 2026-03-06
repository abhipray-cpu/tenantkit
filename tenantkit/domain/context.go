package domain

import (
	"context"
	"time"
)

// Context is a value object representing the tenant context for a request.
// It contains all tenant-specific information needed during request processing.
type Context struct {
	tenantID  TenantID
	userID    string
	requestID string
	timestamp time.Time
}

// NewContext creates a new tenant context
func NewContext(tenantID string, userID string, requestID string) (Context, error) {
	tid, err := NewTenantID(tenantID)
	if err != nil {
		return Context{}, err
	}

	if userID == "" {
		return Context{}, ErrMissingUserID
	}

	if requestID == "" {
		return Context{}, ErrMissingRequestID
	}

	return Context{
		tenantID:  tid,
		userID:    userID,
		requestID: requestID,
		timestamp: time.Now(),
	}, nil
}

// TenantID returns the tenant ID
func (c Context) TenantID() TenantID {
	return c.tenantID
}

// UserID returns the user ID
func (c Context) UserID() string {
	return c.userID
}

// RequestID returns the request ID
func (c Context) RequestID() string {
	return c.requestID
}

// Timestamp returns the context creation timestamp
func (c Context) Timestamp() time.Time {
	return c.timestamp
}

// WithUser returns a new context with updated user ID
func (c Context) WithUser(userID string) (Context, error) {
	if userID == "" {
		return Context{}, ErrMissingUserID
	}
	c.userID = userID
	c.timestamp = time.Now()
	return c, nil
}

// ContextKey is a type for storing context values
type ContextKey string

const (
	// TenantContextKey is the key for storing TenantContext in context.Context
	TenantContextKey ContextKey = "tenant_context"
)

// FromGoContext extracts the tenant context from a Go context
func FromGoContext(ctx context.Context) (Context, error) {
	// FIX BUG #24: Nil context check
	if ctx == nil {
		return Context{}, ErrInvalidContext
	}

	val := ctx.Value(TenantContextKey)
	if val == nil {
		return Context{}, ErrInvalidContext
	}

	tc, ok := val.(Context)
	if !ok {
		return Context{}, ErrInvalidContext
	}

	if tc.tenantID.Value() == "" {
		return Context{}, ErrMissingTenantID
	}

	return tc, nil
}

// ToGoContext returns a new Go context with the tenant context value
func (c Context) ToGoContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, TenantContextKey, c)
}
