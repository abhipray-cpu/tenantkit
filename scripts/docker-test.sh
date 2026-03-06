#!/bin/bash
# TenantKit Docker Testing Infrastructure - Test Execution Script
# Runs comprehensive test suite against real Docker containers

set -e

TEST_CATEGORY=${1:-all}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 TenantKit Test Suite: $TEST_CATEGORY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Navigate to project root
cd "$(dirname "$0")/.."

# Ensure containers are running
echo "🔍 Checking if containers are running..."
if ! docker compose ps | grep -q "Up"; then
    echo "⚠️  Containers not running. Starting infrastructure..."
    ./scripts/docker-up.sh
else
    echo "✅ Containers are running"
fi

# Export connection strings for tests
export POSTGRES_URL="postgres://tenantkit:tenantkit_secret@localhost:5432/tenantkit?sslmode=disable"
export POSTGRES_US_WEST_URL="postgres://tenantkit:tenantkit_secret@localhost:5433/tenantkit?sslmode=disable"
export POSTGRES_EU_WEST_URL="postgres://tenantkit:tenantkit_secret@localhost:5434/tenantkit?sslmode=disable"
export MYSQL_URL="tenantkit:tenantkit_secret@tcp(localhost:3306)/tenantkit"
export REDIS_URL="redis://localhost:6379"
export APP1_URL="http://localhost:8081"
export APP2_URL="http://localhost:8082"
export APP3_URL="http://localhost:8083"

echo ""
echo "🔗 Environment configured:"
echo "  • PostgreSQL (us-east-1): $POSTGRES_URL"
echo "  • PostgreSQL (us-west-1): $POSTGRES_US_WEST_URL"
echo "  • PostgreSQL (eu-west-1): $POSTGRES_EU_WEST_URL"
echo "  • Redis: $REDIS_URL"
echo "  • App Instances: 8081, 8082, 8083"
echo ""

# Test execution based on category
case $TEST_CATEGORY in
    feature)
        echo "🧪 Running Feature Tests (76 tests)..."
        echo "  • Phase 1: Crash Prevention (16 tests)"
        echo "  • Phase 2: Query Safety (16 tests)"
        echo "  • Phase 3: Distributed Systems (24 tests)"
        echo "  • Phase 4: State Synchronization (20 tests)"
        echo ""
        go test -v -count=1 -timeout=15m ./testing/feature/...
        ;;
    
    performance)
        echo "📊 Running Performance Tests (30 tests)..."
        echo "  • Micro-benchmarks (18 benchmarks)"
        echo "  • Macro load tests (12 scenarios)"
        echo ""
        go test -v -count=1 -timeout=30m ./testing/performance/...
        ;;
    
    benchmarks)
        echo "⚡ Running Benchmarks..."
        echo ""
        go test -bench=. -benchmem -benchtime=10s -timeout=30m ./testing/performance/...
        ;;
    
    security)
        echo "🔒 Running Security Tests (19 tests)..."
        echo "  • Tenant Isolation (8 tests)"
        echo "  • SQL Injection (6 tests)"
        echo "  • Rate Limit Bypass (5 tests)"
        echo ""
        go test -v -count=1 -timeout=10m ./testing/security/...
        ;;
    
    chaos)
        echo "🌪️  Running Chaos Tests (18 tests)..."
        echo "  • Failure Scenarios (12 tests)"
        echo "  • Resource Exhaustion (6 tests)"
        echo ""
        go test -v -count=1 -timeout=20m ./testing/chaos/...
        ;;
    
    scenarios)
        echo "🎬 Running Real-World Scenario Tests (22 tests)..."
        echo "  • SaaS Multi-Instance (8 tests)"
        echo "  • E-Commerce Distributed (5 tests)"
        echo "  • Analytics Multi-Region (4 tests)"
        echo "  • Operational Scenarios (5 tests)"
        echo ""
        go test -v -count=1 -timeout=30m ./testing/scenarios/...
        ;;
    
    integration)
        echo "🔗 Running Integration Tests (15 tests)..."
        echo "  • Multi-Region Integration (8 tests)"
        echo "  • Multi-Instance Sync (7 tests)"
        echo ""
        go test -v -count=1 -timeout=15m ./testing/integration/...
        ;;
    
    all)
        echo "🚀 Running ALL Tests (265+ tests)..."
        echo ""
        echo "Test Breakdown:"
        echo "  • Feature Tests: 76"
        echo "  • Performance Tests: 30"
        echo "  • Security Tests: 19"
        echo "  • Chaos Tests: 18"
        echo "  • Scenario Tests: 22"
        echo "  • Integration Tests: 15"
        echo "  • Documentation: 85+"
        echo "  ─────────────────────"
        echo "  • Total: 265+ tests"
        echo ""
        
        START_TIME=$(date +%s)
        
        # Run all test categories
        go test -v -count=1 -timeout=60m ./testing/...
        
        END_TIME=$(date +%s)
        DURATION=$((END_TIME - START_TIME))
        MINUTES=$((DURATION / 60))
        SECONDS=$((DURATION % 60))
        
        echo ""
        echo "⏱️  Total test duration: ${MINUTES}m ${SECONDS}s"
        ;;
    
    quick)
        echo "⚡ Running Quick Test Suite (smoke tests)..."
        echo ""
        go test -v -count=1 -timeout=5m -short ./testing/...
        ;;
    
    *)
        echo "❌ Unknown test category: $TEST_CATEGORY"
        echo ""
        echo "Usage: $0 [CATEGORY]"
        echo ""
        echo "Categories:"
        echo "  feature      - Feature tests (76 tests, ~15 min)"
        echo "  performance  - Performance tests (30 tests, ~30 min)"
        echo "  benchmarks   - Run benchmarks only"
        echo "  security     - Security tests (19 tests, ~10 min)"
        echo "  chaos        - Chaos tests (18 tests, ~20 min)"
        echo "  scenarios    - Scenario tests (22 tests, ~30 min)"
        echo "  integration  - Integration tests (15 tests, ~15 min)"
        echo "  all          - All tests (265+ tests, ~60 min)"
        echo "  quick        - Quick smoke tests (~5 min)"
        echo ""
        exit 1
        ;;
esac

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Tests Completed: $TEST_CATEGORY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
