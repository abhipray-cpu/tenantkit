.PHONY: help docker-up docker-down docker-restart docker-logs docker-clean docker-shell-db docker-shell-redis test lint build
.PHONY: test-unit test-integration test-e2e bench fuzz chaos pen perf load
.PHONY: validate validate-quota validate-multitenancy validate-noisy-neighbor validate-query-safety
.PHONY: phase0 phase1-test phase2-test phase3-test phase4-test phase5-test phase6-test
.PHONY: setup deps check security vet fmt ci ci-full

# Variables
DOCKER_COMPOSE := docker compose
GO := go
GOTEST := $(GO) test
GOBENCH := $(GO) test -bench=.
PROJECT := tenantkit
COVERAGE_DIR := ./coverage

help:
	@echo "🐳 TenantKit Docker Management"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-up        - Start all services with automatic setup"
	@echo "  make docker-down      - Stop all services"
	@echo "  make docker-restart   - Restart all services"
	@echo "  make docker-logs      - View logs from all services"
	@echo "  make docker-logs-app  - View logs from app only"
	@echo "  make docker-clean     - Remove all containers and volumes"
	@echo "  make docker-shell-db  - Open PostgreSQL shell"
	@echo "  make docker-shell-redis - Open Redis CLI"
	@echo ""
	@echo "Development Commands:"
	@echo "  make setup            - Initial project setup"
	@echo "  make test             - Run all tests"
	@echo "  make test-unit        - Run unit tests only"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-e2e         - Run E2E tests"
	@echo "  make lint             - Run linter"
	@echo "  make build            - Build all modules"
	@echo ""
	@echo "Specialized Testing:"
	@echo "  make bench            - Run benchmarks"
	@echo "  make fuzz             - Run fuzz tests"
	@echo "  make chaos            - Run chaos tests"
	@echo "  make pen              - Run penetration tests"
	@echo "  make perf             - Run performance tests"
	@echo "  make load             - Run load tests"
	@echo ""
	@echo "Validation (E2E Flows):"
	@echo "  make validate         - Run all validation tests"
	@echo "  make validate-quota   - Validate quota/restriction flow"
	@echo "  make validate-multitenancy - Validate multi-tenant system"
	@echo "  make validate-noisy-neighbor - Validate noisy neighbor protection"
	@echo "  make validate-query-safety - Validate query tampering scope"
	@echo ""
	@echo "Enhancement Phases:"
	@echo "  make phase0           - Phase 0: Foundation setup"
	@echo "  make phase1-test      - Phase 1: Critical fixes tests"
	@echo "  make phase2-test      - Phase 2: Distributed systems tests"
	@echo "  make phase3-test      - Phase 3: Resilience tests"
	@echo "  make phase4-test      - Phase 4: Query safety tests"
	@echo "  make phase5-test      - Phase 5: Comprehensive testing"
	@echo "  make phase6-test      - Phase 6: Validation"
	@echo ""

docker-up:
	@echo "🚀 Starting TenantKit services..."
	@bash scripts/startup.sh

docker-down:
	@echo "🛑 Stopping TenantKit services..."
	@docker compose down
	@echo "✅ Services stopped"

docker-restart:
	@echo "🔄 Restarting TenantKit services..."
	@docker compose restart
	@echo "✅ Services restarted"

docker-logs:
	@docker compose logs -f

docker-logs-app:
	@docker compose logs -f app

docker-clean:
	@echo "🗑️  Cleaning up containers and volumes..."
	@docker compose down -v
	@echo "✅ Cleanup complete"

docker-status:
	@echo "📊 Service Status:"
	@docker compose ps
	@echo ""
	@echo "📋 Network:"
	@docker network ls | grep tenantkit_network

docker-shell-db:
	@docker compose exec postgres psql -U tenantkit -d tenantkit

docker-shell-redis:
	@docker compose exec redis redis-cli

test:
	@echo "🧪 Running tests..."
	@go test -race -v ./...

lint:
	@echo "🔍 Running linter..."
	@golangci-lint run ./...

build:
	@echo "🏗️  Building modules..."
	@go build -v ./...

rebuild:
	@echo "🏗️  Rebuilding Docker images..."
	@docker compose build --no-cache
	@echo "✅ Build complete"

health-check:
	@echo "🏥 Checking service health..."
	@echo "PostgreSQL:"
	@docker compose exec -T postgres pg_isready -U tenantkit || true
	@echo ""
	@echo "Redis:"
	@docker compose exec -T redis redis-cli ping || true
	@echo ""
	@echo "Application:"
	@curl -s http://localhost:8080/health || echo "App not responding"
	@echo ""

reset-db:
	@echo "🔄 Resetting database..."
	@docker compose exec -T postgres psql -U tenantkit -d tenantkit -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@docker compose exec -T postgres psql -U tenantkit -d tenantkit < scripts/init-db.sql
	@echo "✅ Database reset complete"

load-data:
	@echo "📥 Loading sample data..."
	@docker compose exec -T postgres psql -U tenantkit -d tenantkit < scripts/sample-data.sql || echo "No sample data file found"
	@echo "✅ Sample data loaded"

info:
	@echo ""
	@echo "════════════════════════════════════════════════════════"
	@echo "     📊 TenantKit Service Information"
	@echo "════════════════════════════════════════════════════════"
	@echo ""
	@echo "🌐 Service URLs:"
	@echo "  Application: http://localhost:8080"
	@echo ""
	@echo "🗄️  Database:"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  User: tenantkit"
	@echo "  DB: tenantkit"
	@echo ""
	@echo "🔴 Cache:"
	@echo "  Redis: localhost:6379"
	@echo ""
	@echo "📁 Configuration:"
	@echo "  Docker Compose: docker-compose.yml"
	@echo "  Environment: .env"
	@echo "  Database Init: scripts/init-db.sql"
	@echo "  Redis Config: scripts/redis.conf"
	@echo ""

#------------------------------------------------------------------------------
# Setup
#------------------------------------------------------------------------------

setup: ## Initial project setup
	@echo "🔧 Setting up TenantKit development environment..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "📦 Installing development tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "✅ Setup complete!"

deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod tidy

#------------------------------------------------------------------------------
# Testing - Unit & Integration
#------------------------------------------------------------------------------

test-unit: ## Run unit tests only (no Docker required)
	@echo "🧪 Running unit tests..."
	$(GOTEST) -race -v -short ./...

test-cover: ## Run tests with coverage
	@echo "📊 Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "✅ Coverage report: $(COVERAGE_DIR)/coverage.html"

test-integration: docker-up ## Run integration tests
	@echo "🧪 Running integration tests..."
	$(GOTEST) -race -v -tags=integration ./...

#------------------------------------------------------------------------------
# Testing - E2E Tests
#------------------------------------------------------------------------------

test-e2e: docker-up ## Run end-to-end tests
	@echo "🧪 Running E2E tests..."
	$(GOTEST) -race -v -tags=e2e ./tests/e2e/...

test-e2e-distributed: docker-up ## Run distributed systems E2E tests
	@echo "🧪 Running distributed systems tests..."
	$(GOTEST) -race -v -tags=e2e -run Distributed ./tests/e2e/...

#------------------------------------------------------------------------------
# Testing - Specialized Tests
#------------------------------------------------------------------------------

bench: ## Run benchmarks
	@echo "⚡ Running benchmarks..."
	$(GOBENCH) -benchmem ./...

load: docker-up ## Run load tests
	@echo "🏋️  Running load tests..."
	$(GOTEST) -race -v -tags=load -timeout 30m ./tests/load/...

fuzz: ## Run fuzz tests
	@echo "🔀 Running fuzz tests..."
	$(GOTEST) -fuzz=. -fuzztime=60s ./tests/fuzz/...

fuzz-sql: ## Run SQL parser fuzz tests
	@echo "🔀 Running SQL parser fuzz tests..."
	$(GOTEST) -fuzz=FuzzSQLParser -fuzztime=120s ./storage/...

chaos: docker-up ## Run chaos tests
	@echo "🌪️  Running chaos tests..."
	$(GOTEST) -race -v -tags=chaos -timeout 30m ./tests/chaos/...

pen: docker-up ## Run penetration tests
	@echo "🔓 Running penetration tests..."
	$(GOTEST) -race -v -tags=pen -timeout 30m ./tests/pen/...

perf: docker-up ## Run performance tests
	@echo "📈 Running performance tests..."
	$(GOTEST) -race -v -tags=perf -timeout 30m ./tests/perf/...

#------------------------------------------------------------------------------
# Validation - E2E Flows (Phase 6)
#------------------------------------------------------------------------------

validate: validate-quota validate-multitenancy validate-noisy-neighbor validate-query-safety ## Run all validation tests
	@echo "✅ All validation tests complete!"

validate-quota: docker-up ## Validate quota/restriction definition flow
	@echo "📋 Validating quota definition flow..."
	$(GOTEST) -race -v -tags=validate -run QuotaFlow ./tests/validate/...

validate-multitenancy: docker-up ## Validate multi-tenant system flow
	@echo "🏢 Validating multi-tenant system flow..."
	$(GOTEST) -race -v -tags=validate -run MultiTenancy ./tests/validate/...

validate-noisy-neighbor: docker-up ## Validate noisy neighbor solution
	@echo "🔕 Validating noisy neighbor protection..."
	$(GOTEST) -race -v -tags=validate -run NoisyNeighbor ./tests/validate/...

validate-query-safety: docker-up ## Validate query tampering is only for multi-tenancy
	@echo "🔒 Validating query safety (multi-tenancy only)..."
	$(GOTEST) -race -v -tags=validate -run QuerySafety ./tests/validate/...

#------------------------------------------------------------------------------
# Code Quality
#------------------------------------------------------------------------------

fmt: ## Format code
	@echo "✨ Formatting code..."
	$(GO) fmt ./...

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	$(GO) vet ./...

security: ## Run security scanner
	@echo "🔐 Running security scanner..."
	gosec ./...

check: lint vet security ## Run all code quality checks
	@echo "✅ All quality checks passed!"

#------------------------------------------------------------------------------
# Enhancement Phases (from ENHANCEMENT_PLAN.md)
#------------------------------------------------------------------------------

phase0: setup docker-up ## Phase 0: Foundation setup
	@echo "✅ Phase 0 complete: Foundation ready!"

phase1-test: ## Phase 1: Run critical fixes tests
	@echo "🧪 Running Phase 1 tests (Critical Fixes)..."
	$(GOTEST) -race -v -run 'TestPanic|TestBypass' ./...

phase2-test: docker-up ## Phase 2: Run distributed systems tests
	@echo "🧪 Running Phase 2 tests (Distributed Systems)..."
	$(GOTEST) -race -v -run 'TestRedis|TestDistributed|TestSync' ./...

phase3-test: docker-up ## Phase 3: Run resilience tests
	@echo "🧪 Running Phase 3 tests (Resilience)..."
	$(GOTEST) -race -v -run 'TestCircuitBreaker|TestShutdown|TestRecovery' ./...

phase4-test: ## Phase 4: Run query safety tests
	@echo "🧪 Running Phase 4 tests (Query Safety)..."
	$(GOTEST) -race -v -run 'TestDDL|TestQuery|TestBypass' ./...

phase5-test: test bench fuzz ## Phase 5: Run comprehensive testing
	@echo "✅ Phase 5: Comprehensive testing complete!"

phase6-test: validate ## Phase 6: Run validation tests
	@echo "✅ Phase 6: Validation complete!"

#------------------------------------------------------------------------------
# CI/CD
#------------------------------------------------------------------------------

ci: deps lint test-cover ## CI pipeline
	@echo "✅ CI pipeline complete!"

ci-full: deps lint test-cover test-integration test-e2e ## Full CI pipeline
	@echo "✅ Full CI pipeline complete!"

.DEFAULT_GOAL := help

