package domain

import "errors"

// Tenant errors
var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrTenantExists   = errors.New("tenant already exists")
)

// Context errors
var (
	ErrInvalidContext   = errors.New("invalid tenant context")
	ErrMissingTenantID  = errors.New("missing tenant ID in context")
	ErrMissingUserID    = errors.New("missing user ID in context")
	ErrMissingRequestID = errors.New("missing request ID in context")
)

// Query errors
var (
	ErrUnsafeQuery        = errors.New("unsafe query: missing tenant filter")
	ErrQueryParseFailed   = errors.New("failed to parse SQL query")
	ErrQueryRewriteFailed = errors.New("failed to rewrite SQL query")
)

// Storage errors
var (
	ErrStorageNotAvailable = errors.New("storage service not available")
	ErrTransactionFailed   = errors.New("transaction failed")
)

// Cache errors
var (
	ErrCacheNotAvailable = errors.New("cache service not available")
)

// Quota errors
var (
	ErrQuotaExceeded = errors.New("quota exceeded")
	ErrQuotaNotFound = errors.New("quota not found")
)

// Rate limiter errors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)
