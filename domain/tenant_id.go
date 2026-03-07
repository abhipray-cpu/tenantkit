package domain

import (
	"errors"
	"regexp"
	"strings"
)

// TenantID is a value object representing a unique tenant identifier.
// It must be lowercase alphanumeric with dashes, 1-100 characters.
type TenantID struct {
	value string
}

// ErrInvalidTenantID is returned when TenantID is invalid
var ErrInvalidTenantID = errors.New("invalid tenant ID: must be 1-255 characters, alphanumeric with dashes, underscores, and dots")

// NewTenantID creates a new TenantID with validation.
// Accepts alphanumeric characters, dashes, underscores, and dots.
// This supports common formats: slugs (acme-corp), UUIDs (550e8400-...),
// org identifiers (org_123), and hierarchical IDs (tenant.prod).
func NewTenantID(id string) (TenantID, error) {
	if err := validateTenantID(id); err != nil {
		return TenantID{}, err
	}
	return TenantID{value: strings.ToLower(id)}, nil
}

// validateTenantID validates tenant ID format
func validateTenantID(id string) error {
	if len(id) < 1 || len(id) > 255 {
		return ErrInvalidTenantID
	}

	// Allow letters, numbers, dashes, underscores, and dots
	pattern := `^[a-zA-Z0-9._-]+$`
	if !regexp.MustCompile(pattern).MatchString(id) {
		return ErrInvalidTenantID
	}

	return nil
}

// String returns the string representation of the TenantID
func (tid TenantID) String() string {
	return tid.value
}

// Value returns the underlying string value
func (tid TenantID) Value() string {
	return tid.value
}

// Equal checks if two TenantIDs are equal
func (tid TenantID) Equal(other TenantID) bool {
	return tid.value == other.value
}
