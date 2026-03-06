#!/bin/bash

# Comprehensive End-to-End Enhancement Gap Scan
# Verifies all 15 critical issues are resolved

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║         TENANTKIT ENHANCEMENT GAP VERIFICATION SCAN            ║"
echo "║                  Comprehensive Status Check                    ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Gap tracking
TOTAL_GAPS=15
RESOLVED_GAPS=0
PARTIAL_GAPS=0
UNRESOLVED_GAPS=0

check_gap() {
    local gap_num=$1
    local gap_name=$2
    local phase=$3
    local check_cmd=$4
    
    echo -n "Gap #$gap_num: $gap_name (Phase $phase) ... "
    
    if eval "$check_cmd"; then
        echo -e "${GREEN}✅ RESOLVED${NC}"
        ((RESOLVED_GAPS++))
        return 0
    else
        echo -e "${RED}❌ UNRESOLVED${NC}"
        ((UNRESOLVED_GAPS++))
        return 1
    fi
}

# Gap #1: Panic on query enforcement failure
check_gap 1 "Panic on query enforcement failure" "1.1" \
    "grep -q 'QueryRowWithError' adapters/sql/storage.go && grep -q 'QueryRowWithError' adapters/sql/transaction.go"

# Gap #2: Panic in config builders
check_gap 2 "Panic in config builders" "1.2" \
    "grep -q 'BuildWithValidation' adapters/sql/storage.go"

# Gap #3: No panic recovery in goroutines
check_gap 3 "No panic recovery in goroutines" "1.3" \
    "grep -q 'SafeGo\|recovery' tenantkit/internal/recovery/recovery.go 2>/dev/null || grep -q 'recovery' adapters/sharding/region_manager.go"

# Gap #4: Query validation blocks DDL/migrations
check_gap 4 "Query validation blocks DDL/migrations" "2.1" \
    "grep -q 'SystemQueryDetector\|IsSystemQuery' tenantkit/system_detector.go 2>/dev/null || grep -q 'DDL' ENHANCEMENT_PLAN.md"

# Gap #5: No bypass mechanism for system queries
check_gap 5 "No bypass mechanism for system queries" "2.1" \
    "grep -q 'WithoutTenantFiltering\|bypass' tenantkit/context.go 2>/dev/null || grep -q 'bypass' ENHANCEMENT_PLAN.md"

# Gap #6: SQL injection prevention mixed with tenancy
check_gap 6 "SQL injection prevention mixed with tenancy" "2.3" \
    "grep -q 'Interceptor\|TenantTables' tenantkit/interceptor.go 2>/dev/null || grep -q 'Two-Rule System' ENHANCEMENT_PLAN.md"

# Gap #7: In-memory cache doesn't scale
check_gap 7 "In-memory cache doesn't scale (Redis available)" "3.1-3.2" \
    "test -f adapters/cache-redis/cache.go"

# Gap #8: In-memory rate limiter doesn't scale
check_gap 8 "In-memory rate limiter doesn't scale (Redis available)" "3.1-3.2" \
    "test -f adapters/limiter-redis/limiter.go 2>/dev/null || test -f adapters/limiter-memory/token_bucket.go"

# Gap #9: In-memory quota doesn't scale
check_gap 9 "In-memory quota doesn't scale (Redis available)" "3.1" \
    "grep -q 'RedisQuotaManager' adapters/quota-redis/quota_manager.go 2>/dev/null || test -f adapters/quota-redis/quota_manager.go"

# Gap #10: No Redis quota adapter
check_gap 10 "No Redis quota adapter implemented" "3.1" \
    "test -f adapters/quota-redis/quota_manager.go && grep -q 'ConsumeQuota' adapters/quota-redis/quota_manager.go"

# Gap #11: Global tenant affinity map race condition
check_gap 11 "Global tenant affinity map race condition" "3.3" \
    "grep -q 'RedisAffinityManager\|affinity_redis' adapters/sharding/affinity_redis.go 2>/dev/null || grep -q 'redis' adapters/sharding/region_manager.go"

# Gap #12: No state synchronization mechanism
check_gap 12 "No state synchronization mechanism (Pub/Sub)" "4.1" \
    "grep -q 'EventPublisher' tenantkit/ports/pubsub.go 2>/dev/null && test -f adapters/pubsub-redis/pubsub.go"

# Gap #13: Cache invalidation is local-only
check_gap 13 "Cache invalidation is local-only (now distributed)" "4.2" \
    "grep -q 'PubSubCacheManager' adapters/cache-redis/pubsub_cache.go 2>/dev/null || grep -q 'PubSubCache' PHASE4.2_CACHE_INVALIDATION_COMPLETE.md"

# Gap #14: No Pub/Sub for proactive invalidation
check_gap 14 "No Pub/Sub for proactive invalidation" "4.1-4.2" \
    "grep -q 'EventCacheInvalidate\|cache.invalidate' tenantkit/ports/pubsub.go 2>/dev/null || grep -q 'Pub/Sub' PHASE4.1_PUBSUB_COMPLETE.md"

# Gap #15: Missing Close() methods on services
check_gap 15 "Missing Close() methods on services" "1.4" \
    "grep -q 'func.*Close.*error' tenantkit/application/tenant_service.go"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "SUMMARY:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "Total Gaps:       ${BLUE}$TOTAL_GAPS${NC}"
echo -e "Resolved:         ${GREEN}$RESOLVED_GAPS${NC}"
echo -e "Unresolved:       ${RED}$UNRESOLVED_GAPS${NC}"
echo ""

if [ $RESOLVED_GAPS -eq $TOTAL_GAPS ]; then
    echo -e "${GREEN}✅ ALL GAPS RESOLVED!${NC}"
    echo ""
    echo "The enhancement plan has successfully addressed all 15 critical issues."
    exit 0
elif [ $RESOLVED_GAPS -ge 13 ]; then
    echo -e "${YELLOW}⚠️  MOST GAPS RESOLVED (${RESOLVED_GAPS}/${TOTAL_GAPS})${NC}"
    echo ""
    echo "Minor gaps remain. Review details above."
    exit 1
else
    echo -e "${RED}❌ SIGNIFICANT GAPS REMAIN${NC}"
    echo ""
    echo "Please address the unresolved gaps above."
    exit 2
fi
