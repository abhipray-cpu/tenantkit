#!/bin/bash
# TenantKit Docker Testing Infrastructure - Shutdown Script
# Stops all containers and optionally removes volumes

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🛑 Stopping TenantKit Docker Testing Infrastructure"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Navigate to project root
cd "$(dirname "$0")/.."

# Check if we should remove volumes
REMOVE_VOLUMES=false
if [ "$1" == "-v" ] || [ "$1" == "--volumes" ]; then
    REMOVE_VOLUMES=true
fi

# Stop all containers
echo "🛑 Stopping all containers..."
docker compose down

if [ "$REMOVE_VOLUMES" == "true" ]; then
    echo ""
    echo "🗑️  Removing volumes..."
    docker compose down -v
    echo "  ✅ Volumes removed"
    echo "  ⚠️  All data has been deleted!"
else
    echo ""
    echo "💾 Volumes preserved (data retained)"
    echo "  To remove volumes, run: $0 --volumes"
fi

echo ""
echo "✅ All containers stopped"
echo ""
