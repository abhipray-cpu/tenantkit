package domain

// Package domain provides core domain types for TenantKit.
//
// The primary type is [TenantID], a validated string type that ensures tenant
// identifiers conform to naming rules. TenantIDs are case-insensitive and
// support alphanumeric characters, hyphens, underscores, and dots.
//
// The package also provides context helpers for threading tenant information
// through Go contexts:
//
//	tctx := domain.NewContext("acme-corp", "user-123", "req-abc")
//	ctx := tctx.ToGoContext(context.Background())
//
// # Error Sentinels
//
// Standard error values (ErrTenantNotFound, ErrInvalidTenantID, etc.) are
// provided for consistent error handling across adapters.
