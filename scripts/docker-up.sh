#!/bin/bash
# TenantKit Docker Testing Infrastructure - Startup Script
# Starts all containers for comprehensive multi-region, multi-instance testing

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 Starting TenantKit Docker Testing Infrastructure"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Navigate to project root
cd "$(dirname "$0")/.."

# Build test application images
echo "📦 Building test application images..."
docker-compose build

# Start infrastructure (databases, redis, monitoring)
echo ""
echo "🗄️  Starting databases and infrastructure..."
echo "  • PostgreSQL (us-east-1) - Port 5432"
echo "  • PostgreSQL (us-west-1) - Port 5433"
echo "  • PostgreSQL (eu-west-1) - Port 5434"
echo "  • MySQL                  - Port 3306"
echo "  • Redis                  - Port 6379"
echo "  • Redis Sentinel         - Port 26379"
echo "  • Prometheus             - Port 9090"
echo "  • Grafana                - Port 3000"

docker-compose up -d postgres postgres-us-west postgres-eu-west mysql redis redis-sentinel prometheus grafana

# Wait for databases to be healthy
echo ""
echo "⏳ Waiting for databases to be healthy..."
for i in {1..30}; do
  if docker-compose exec -T postgres pg_isready -h localhost > /dev/null 2>&1 && \
     docker-compose exec -T postgres-us-west pg_isready -h localhost > /dev/null 2>&1 && \
     docker-compose exec -T postgres-eu-west pg_isready -h localhost > /dev/null 2>&1; then
    echo "✅ Databases are healthy"
    break
  fi
  echo "  Attempt $i/30 - Still waiting..."
  sleep 1
done

# Start application instances
echo ""
echo "🚀 Starting application instances..."
echo "  • app-1 - Port 8081 (metrics: 9101)"
echo "  • app-2 - Port 8082 (metrics: 9102)"
echo "  • app-3 - Port 8083 (metrics: 9103)"

docker-compose up -d app-1 app-2 app-3

# Wait for apps to be ready
echo ""
echo "⏳ Waiting for application instances to be ready..."
sleep 5

# Start chaos tools
echo ""
echo "🌪️  Starting chaos testing tools..."
docker-compose up -d toxiproxy

# Start exporters
echo ""
echo "📊 Starting metric exporters..."
docker-compose up -d redis-exporter postgres-exporter

# Verify all services
echo ""
echo "✅ Verifying services..."
docker-compose ps

# Health check
echo ""
echo "🏥 Running health checks..."

# Check PostgreSQL regions
for port in 5432 5433 5434; do
    if pg_isready -h localhost -p $port -U tenantkit > /dev/null 2>&1; then
        echo "  ✅ PostgreSQL on port $port: healthy"
    else
        echo "  ⚠️  PostgreSQL on port $port: not responding"
    fi
done

# Check Redis
if redis-cli -h localhost -p 6379 ping > /dev/null 2>&1; then
    echo "  ✅ Redis: healthy"
else
    echo "  ⚠️  Redis: not responding"
fi

# Check app instances
for port in 8081 8082 8083; do
    if curl -s http://localhost:$port/health > /dev/null 2>&1; then
        echo "  ✅ App instance on port $port: healthy"
    else
        echo "  ⚠️  App instance on port $port: not responding (may not have health endpoint)"
    fi
done

# Show connection info
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ TenantKit Testing Infrastructure Ready!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📊 Application Instances:"
echo "  • app-1: http://localhost:8081 (metrics: :9101)"
echo "  • app-2: http://localhost:8082 (metrics: :9102)"
echo "  • app-3: http://localhost:8083 (metrics: :9103)"
echo ""
echo "🗄️  Databases:"
echo "  • PostgreSQL (us-east-1): localhost:5432"
echo "  • PostgreSQL (us-west-1): localhost:5433"
echo "  • PostgreSQL (eu-west-1): localhost:5434"
echo "  • MySQL:                  localhost:3306"
echo ""
echo "🔄 Infrastructure:"
echo "  • Redis:          localhost:6379"
echo "  • Redis Sentinel: localhost:26379"
echo "  • Toxiproxy:      localhost:8474"
echo ""
echo "📈 Monitoring:"
echo "  • Prometheus: http://localhost:9090"
echo "  • Grafana:    http://localhost:3000 (admin/admin)"
echo ""
echo "🔗 Connection Strings (for tests):"
echo "  export POSTGRES_URL='postgres://tenantkit:tenantkit_secret@localhost:5432/tenantkit?sslmode=disable'"
echo "  export POSTGRES_US_WEST_URL='postgres://tenantkit:tenantkit_secret@localhost:5433/tenantkit?sslmode=disable'"
echo "  export POSTGRES_EU_WEST_URL='postgres://tenantkit:tenantkit_secret@localhost:5434/tenantkit?sslmode=disable'"
echo "  export REDIS_URL='redis://localhost:6379'"
echo "  export MYSQL_URL='tenantkit:tenantkit_secret@tcp(localhost:3306)/tenantkit'"
echo ""
echo "🧪 Run tests:"
echo "  • All tests:         ./scripts/docker-test.sh all"
echo "  • Feature tests:     ./scripts/docker-test.sh feature"
echo "  • Performance tests: ./scripts/docker-test.sh performance"
echo "  • Security tests:    ./scripts/docker-test.sh security"
echo "  • Chaos tests:       ./scripts/docker-test.sh chaos"
echo "  • Scenario tests:    ./scripts/docker-test.sh scenarios"
echo "  • Integration tests: ./scripts/docker-test.sh integration"
echo ""
echo "🔄 Validate synchronization:"
echo "  ./scripts/validate-sync.sh"
echo ""
echo "🛑 Stop all containers:"
echo "  ./scripts/docker-down.sh"
echo ""
