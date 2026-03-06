#!/bin/bash

# TenantKit Docker Compose Startup Script
# This script brings up all containers and validates the setup

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$SCRIPT_DIR"

echo "════════════════════════════════════════════════════════"
echo "     🚀 TenantKit Docker Compose Startup"
echo "════════════════════════════════════════════════════════"
echo ""

# Check if Docker is running
echo "🔍 Checking Docker availability..."
if ! docker ps > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi
echo "✅ Docker is running"
echo ""

# Check if docker-compose is available
echo "🔍 Checking Docker Compose availability..."
if ! docker compose version > /dev/null 2>&1; then
    echo "❌ Docker Compose is not available. Please install Docker Compose and try again."
    exit 1
fi
echo "✅ Docker Compose is available"
echo ""

# Clean up old containers and volumes (optional)
read -p "🗑️  Do you want to clean up old containers and volumes? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🧹 Cleaning up old resources..."
    docker compose down -v || true
    echo "✅ Cleanup complete"
    echo ""
fi

# Build images
echo "🏗️  Building Docker images..."
docker compose build --no-cache
echo "✅ Build complete"
echo ""

# Start services
echo "⚙️  Starting services..."
docker compose up -d
echo "✅ Services started"
echo ""

# Wait for services to be healthy
echo "⏳ Waiting for services to be healthy..."
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    postgres_health=$(docker inspect --format='{{.State.Health.Status}}' tenantkit_postgres 2>/dev/null || echo "starting")
    redis_health=$(docker inspect --format='{{.State.Health.Status}}' tenantkit_redis 2>/dev/null || echo "starting")
    
    if [ "$postgres_health" = "healthy" ] && [ "$redis_health" = "healthy" ]; then
        echo "✅ All services are healthy"
        break
    fi
    
    echo "  PostgreSQL: $postgres_health, Redis: $redis_health"
    sleep 1
    ((attempt++))
done

if [ $attempt -eq $max_attempts ]; then
    echo "❌ Services did not become healthy in time"
    echo "Run 'docker compose logs' to see detailed logs"
    exit 1
fi
echo ""

# Verify database initialization
echo "🔍 Verifying database initialization..."
sleep 2
db_check=$(docker compose exec -T postgres psql -U tenantkit -d tenantkit -c "SELECT COUNT(*) FROM tenants;" 2>/dev/null || echo "0")
if [ "$db_check" -gt 0 ]; then
    echo "✅ Database initialized with tenants"
else
    echo "⚠️  Database check incomplete, but services are running"
fi
echo ""

# Display service information
echo "════════════════════════════════════════════════════════"
echo "     ✨ Services Running"
echo "════════════════════════════════════════════════════════"
echo ""
echo "📊 Service Status:"
docker compose ps
echo ""

echo "📋 Connection Information:"
echo "  PostgreSQL:  localhost:5432"
echo "             User: tenantkit"
echo "             Password: tenantkit_secure_password"
echo "             Database: tenantkit"
echo ""
echo "  Redis:       localhost:6379"
echo ""
echo "  Application: http://localhost:8080"
echo ""

echo "📝 Useful Commands:"
echo "  View logs:       docker compose logs -f"
echo "  View logs (app): docker compose logs -f app"
echo "  Stop services:   docker compose down"
echo "  Restart:         docker compose restart"
echo "  Database shell:  docker compose exec postgres psql -U tenantkit -d tenantkit"
echo "  Redis shell:     docker compose exec redis redis-cli"
echo ""

echo "🎯 Next Steps:"
echo "  1. Run integration tests: go test ./tests/integration/..."
echo "  2. Check application health: curl http://localhost:8080/health"
echo "  3. View logs: docker compose logs -f"
echo ""

echo "════════════════════════════════════════════════════════"
echo "     ✅ Setup Complete!"
echo "════════════════════════════════════════════════════════"
