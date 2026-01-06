#!/bin/bash
# test-phase4-api.sh - Test Phase 4 IAM Drift Detection APIs

echo "============================================"
echo "  Phase 4: IAM Drift Detection API Tests"
echo "============================================"
echo ""

API_BASE="http://localhost:9080/api/v1"

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TEST_NUM=0

run_test() {
    TEST_NUM=$((TEST_NUM + 1))
    local name="$1"
    local endpoint="$2"
    
    echo -e "${BLUE}Test $TEST_NUM: $name${NC}"
    echo "GET $API_BASE$endpoint"
    echo ""
    
    response=$(curl -s "$API_BASE$endpoint")
    
    if [ $? -eq 0 ] && [ -n "$response" ]; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        echo -e "${GREEN}✓ PASS${NC}"
    else
        echo -e "${RED}✗ FAIL - No response${NC}"
    fi
    echo ""
    echo "---"
    echo ""
}

# Check if manager is running
echo "Checking if manager is running..."
if ! curl -s http://localhost:9080/healthz > /dev/null 2>&1; then
    echo -e "${RED}❌ Manager is not running!${NC}"
    echo ""
    echo "Start it first with: ./start-phase4.sh"
    echo ""
    exit 1
fi
echo -e "${GREEN}✓ Manager is running${NC}"
echo ""
echo "============================================"
echo ""

# Run tests
run_test "Platform Drift Summary" "/iam/drift/platform"

run_test "All Tenants Drift" "/iam/drift/tenants"

run_test "Tenant Alpha Drift Details" "/iam/drift/tenants/alpha"

run_test "Tenant Beta Drift Details" "/iam/drift/tenants/beta"

# Test manual scan trigger
echo -e "${BLUE}Test $((TEST_NUM + 1)): Trigger Manual Scan${NC}"
echo "POST $API_BASE/iam/drift/scan"
echo ""
curl -s -X POST "$API_BASE/iam/drift/scan" | jq '.'
echo -e "${GREEN}✓ Scan triggered${NC}"
echo ""
echo "---"
echo ""

echo "============================================"
echo -e "${GREEN}✓ All API tests completed!${NC}"
echo "============================================"
echo ""
echo "Next steps:"
echo "1. Check manager logs: tail -f manager-phase4.log | grep drift"
echo "2. Create drift: awslocal iam put-role-policy ..."
echo "3. Re-run this script to see detected drift"
echo ""

