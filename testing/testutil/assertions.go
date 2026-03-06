// Package testutil provides testing utilities for TenantKit
// This file contains assertion helpers for testing
package testutil

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v %v", err, msgAndArgs)
	}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error but got nil %v", msgAndArgs)
	}
}

// AssertErrorContains fails if err is nil or doesn't contain the substring
func AssertErrorContains(t *testing.T, err error, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q but got nil %v", substr, msgAndArgs)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("expected error containing %q but got: %v %v", substr, err, msgAndArgs)
	}
}

// AssertEqual fails if expected != actual
func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v but got %v %v", expected, actual, msgAndArgs)
	}
}

// AssertNotEqual fails if expected == actual
func AssertNotEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected values to be different but both were: %v %v", expected, msgAndArgs)
	}
}

// AssertTrue fails if condition is false
func AssertTrue(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if !condition {
		t.Fatalf("expected true but got false %v", msgAndArgs)
	}
}

// AssertFalse fails if condition is true
func AssertFalse(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if condition {
		t.Fatalf("expected false but got true %v", msgAndArgs)
	}
}

// AssertNil fails if value is not nil
func AssertNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if value != nil && !reflect.ValueOf(value).IsNil() {
		t.Fatalf("expected nil but got: %v %v", value, msgAndArgs)
	}
}

// AssertNotNil fails if value is nil
func AssertNotNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if value == nil || reflect.ValueOf(value).IsNil() {
		t.Fatalf("expected non-nil value %v", msgAndArgs)
	}
}

// AssertContains fails if str doesn't contain substr
func AssertContains(t *testing.T, str, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if !strings.Contains(str, substr) {
		t.Fatalf("expected %q to contain %q %v", str, substr, msgAndArgs)
	}
}

// AssertNotContains fails if str contains substr
func AssertNotContains(t *testing.T, str, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if strings.Contains(str, substr) {
		t.Fatalf("expected %q to not contain %q %v", str, substr, msgAndArgs)
	}
}

// AssertLen fails if the length of value is not expected
func AssertLen(t *testing.T, value interface{}, expected int, msgAndArgs ...interface{}) {
	t.Helper()
	v := reflect.ValueOf(value)
	if v.Len() != expected {
		t.Fatalf("expected length %d but got %d %v", expected, v.Len(), msgAndArgs)
	}
}

// AssertEmpty fails if value is not empty
func AssertEmpty(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	v := reflect.ValueOf(value)
	if v.Len() != 0 {
		t.Fatalf("expected empty but got length %d %v", v.Len(), msgAndArgs)
	}
}

// AssertNotEmpty fails if value is empty
func AssertNotEmpty(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	v := reflect.ValueOf(value)
	if v.Len() == 0 {
		t.Fatalf("expected non-empty value %v", msgAndArgs)
	}
}

// AssertPanics fails if the function doesn't panic
func AssertPanics(t *testing.T, fn func(), msgAndArgs ...interface{}) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic but none occurred %v", msgAndArgs)
		}
	}()
	fn()
}

// AssertNoPanic fails if the function panics
func AssertNoPanic(t *testing.T, fn func(), msgAndArgs ...interface{}) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v %v", r, msgAndArgs)
		}
	}()
	fn()
}

// AssertDuration fails if the duration is not within the expected range
func AssertDuration(t *testing.T, actual, min, max time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	if actual < min || actual > max {
		t.Fatalf("expected duration between %v and %v but got %v %v", min, max, actual, msgAndArgs)
	}
}

// AssertJSON fails if the values are not equal when serialized as JSON
func AssertJSON(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("failed to marshal expected: %v", err)
	}

	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("failed to marshal actual: %v", err)
	}

	if string(expectedJSON) != string(actualJSON) {
		t.Fatalf("expected JSON %s but got %s %v", expectedJSON, actualJSON, msgAndArgs)
	}
}

// RequireNoError is like AssertNoError but stops the test immediately
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		t.Fatalf("required no error but got: %v %v", err, msgAndArgs)
	}
}

// RequireError is like AssertError but stops the test immediately
func RequireError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		t.Fatalf("required error but got nil %v", msgAndArgs)
	}
}

// BenchmarkResult holds benchmark results for comparison
type BenchmarkResult struct {
	Name        string
	Operations  int
	NsPerOp     int64
	BytesPerOp  int64
	AllocsPerOp int64
	Duration    time.Duration
}

// AssertPerformance validates that the operation meets performance requirements
func AssertPerformance(t *testing.T, result BenchmarkResult, maxNsPerOp int64, msgAndArgs ...interface{}) {
	t.Helper()
	if result.NsPerOp > maxNsPerOp {
		t.Fatalf("performance requirement not met: %d ns/op exceeds max %d ns/op %v",
			result.NsPerOp, maxNsPerOp, msgAndArgs)
	}
}

// AssertResponseTime validates that a response time is within acceptable limits
func AssertResponseTime(t *testing.T, actual, max time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	if actual > max {
		t.Fatalf("response time %v exceeds maximum %v %v", actual, max, msgAndArgs)
	}
}
