// Package testutil provides testing utilities for TenantKit
// This file contains Docker container management utilities for integration testing
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// ContainerManager manages Docker containers for testing
type ContainerManager struct {
	projectDir     string
	composeFile    string
	containers     []string
	cleanupOnClose bool
}

// NewContainerManager creates a new container manager
func NewContainerManager(projectDir string) *ContainerManager {
	return &ContainerManager{
		projectDir:     projectDir,
		composeFile:    "docker-compose.yml",
		containers:     []string{},
		cleanupOnClose: true,
	}
}

// StartAll starts all containers defined in docker-compose.yml
func (cm *ContainerManager) StartAll(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "compose",
		"-f", cm.composeFile,
		"up", "-d", "--wait")
	cmd.Dir = cm.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// StartServices starts specific services from docker-compose.yml
func (cm *ContainerManager) StartServices(ctx context.Context, services ...string) error {
	args := []string{"compose", "-f", cm.composeFile, "up", "-d", "--wait"}
	args = append(args, services...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = cm.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// StopAll stops all containers
func (cm *ContainerManager) StopAll(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "compose",
		"-f", cm.composeFile,
		"down", "-v")
	cmd.Dir = cm.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RestartService restarts a specific service
func (cm *ContainerManager) RestartService(ctx context.Context, service string) error {
	cmd := exec.CommandContext(ctx, "docker", "compose",
		"-f", cm.composeFile,
		"restart", service)
	cmd.Dir = cm.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// WaitForService waits for a service to be healthy
func (cm *ContainerManager) WaitForService(ctx context.Context, service string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for service %s", service)
		}

		cmd := exec.CommandContext(ctx, "docker", "compose",
			"-f", cm.composeFile,
			"ps", "--format", "{{.Health}}", service)
		cmd.Dir = cm.projectDir

		output, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(output)) == "healthy" {
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}

// GetServicePort returns the mapped port for a service
func (cm *ContainerManager) GetServicePort(ctx context.Context, service, port string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "compose",
		"-f", cm.composeFile,
		"port", service, port)
	cmd.Dir = cm.projectDir

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Output is like "0.0.0.0:5432" - extract port
	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) >= 2 {
		return parts[len(parts)-1], nil
	}

	return "", fmt.Errorf("could not parse port from: %s", output)
}

// IsDockerAvailable checks if Docker is available
func IsDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	return cmd.Run() == nil
}

// SkipIfNoDocker skips the test if Docker is not available
func SkipIfNoDocker(t *testing.T) {
	t.Helper()
	if !IsDockerAvailable() {
		t.Skip("Docker is not available, skipping test")
	}
}

// WaitForPostgres waits for PostgreSQL to be ready
func WaitForPostgres(ctx context.Context, connStr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for PostgreSQL")
		}

		db, err := sql.Open("postgres", connStr)
		if err == nil {
			if err := db.PingContext(ctx); err == nil {
				db.Close()
				return nil
			}
			db.Close()
		}

		time.Sleep(1 * time.Second)
	}
}

// WaitForRedis waits for Redis to be ready
func WaitForRedis(ctx context.Context, addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for Redis")
		}

		cmd := exec.CommandContext(ctx, "redis-cli", "-h", addr, "ping")
		if output, err := cmd.Output(); err == nil && strings.TrimSpace(string(output)) == "PONG" {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// CleanupDatabase cleans up test data from the database
func CleanupDatabase(ctx context.Context, db *sql.DB, tables ...string) error {
	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}
	return nil
}

// TestDatabase wraps a database connection for testing
type TestDatabase struct {
	DB      *sql.DB
	ConnStr string
	Driver  string
}

// NewTestPostgres creates a test PostgreSQL connection
func NewTestPostgres(t *testing.T, connStr string) *TestDatabase {
	t.Helper()

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to open postgres: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping postgres: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return &TestDatabase{
		DB:      db,
		ConnStr: connStr,
		Driver:  "postgres",
	}
}

// Exec executes a query without returning rows
func (td *TestDatabase) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := td.DB.ExecContext(ctx, query, args...)
	return err
}

// Query executes a query and returns rows
func (td *TestDatabase) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return td.DB.QueryContext(ctx, query, args...)
}
