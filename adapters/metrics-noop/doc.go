package noop

// Package noop provides a no-operation metrics adapter implementing [ports.Metrics].
//
// Use this adapter when you don't need metrics collection, or as a default
// during development and testing.
//
// All methods are no-ops that return immediately.
