#!/bin/bash

# Test All API Endpoints
# Validates that all endpoints are working with real data from the cluster

set -e

API_URL="http://localhost:9080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║           Platform Manager API Endpoint Verification             ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

test_endpoint() {
    local name="$1"
    local method="$2"
    local path="$3"
    local expected_field="$4"
    
    printf "%-60s " "$name"
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "$API_URL$path")
        http_code=$(echo "$response" | tail -n1)
        body=$(echo "$response" | sed '$d')
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$API_URL$path")
        http_code=$(echo "$response" | tail -n1)
        body=$(echo "$response" | sed '$d')
    fi
    
    if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
        if [ -n "$expected_field" ]; then
            if echo "$body" | jq -e "$expected_field" > /dev/null 2>&1; then
                echo -e "${GREEN}✓ PASS${NC} (HTTP $http_code)"
                return 0
            else
                echo -e "${YELLOW}⚠ WARN${NC} (HTTP $http_code, missing field: $expected_field)"
                return 1
            fi
        else
            echo -e "${GREEN}✓ PASS${NC} (HTTP $http_code)"
            return 0
        fi
    else
        echo -e "${RED}✗ FAIL${NC} (HTTP $http_code)"
        return 1
    fi
}

PASS_COUNT=0
FAIL_COUNT=0

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1. HEALTH ENDPOINTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

test_endpoint "GET /api/v1/health/platform" "GET" "/api/v1/health/platform" ".overallHealth" && ((PASS_COUNT++)) || ((FAIL_COUNT++))
test_endpoint "GET /api/v1/health/tenants" "GET" "/api/v1/health/tenants" ".[0].name" && ((PASS_COUNT++)) || ((FAIL_COUNT++))
test_endpoint "GET /api/v1/health/tenants/tenant-alpha" "GET" "/api/v1/health/tenants/tenant-alpha" ".tenantRef" && ((PASS_COUNT++)) || ((FAIL_COUNT++))

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2. TENANT ENDPOINTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

test_endpoint "GET /api/v1/tenants" "GET" "/api/v1/tenants" ".[0].name" && ((PASS_COUNT++)) || ((FAIL_COUNT++))
test_endpoint "GET /api/v1/tenants/tenant-alpha" "GET" "/api/v1/tenants/tenant-alpha" ".name" && ((PASS_COUNT++)) || ((FAIL_COUNT++))
test_endpoint "GET /api/v1/tenants/tenant-alpha/resources" "GET" "/api/v1/tenants/tenant-alpha/resources" ".resources" && ((PASS_COUNT++)) || ((FAIL_COUNT++))

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3. RESOURCES ENDPOINTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

test_endpoint "GET /api/v1/resources" "GET" "/api/v1/resources" ".resources" && ((PASS_COUNT++)) || ((FAIL_COUNT++))
test_endpoint "GET /api/v1/resources?tenant=tenant-alpha" "GET" "/api/v1/resources?tenant=tenant-alpha" ".resources" && ((PASS_COUNT++)) || ((FAIL_COUNT++))
test_endpoint "GET /api/v1/resources?category=Kubernetes" "GET" "/api/v1/resources?category=Kubernetes" ".resources" && ((PASS_COUNT++)) || ((FAIL_COUNT++))

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "4. METRICS ENDPOINTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

test_endpoint "GET /api/v1/metrics/tenants/tenant-alpha" "GET" "/api/v1/metrics/tenants/tenant-alpha" ".tenantId" && ((PASS_COUNT++)) || ((FAIL_COUNT++))
test_endpoint "GET /api/v1/metrics/namespaces/tenant-alpha" "GET" "/api/v1/metrics/namespaces/tenant-alpha" ".namespace" && ((PASS_COUNT++)) || ((FAIL_COUNT++))

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "5. IAM DRIFT ENDPOINTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

test_endpoint "GET /api/v1/iam/drift/platform" "GET" "/api/v1/iam/drift/platform" ".summary" && ((PASS_COUNT++)) || ((FAIL_COUNT++))
test_endpoint "GET /api/v1/iam/drift/tenants" "GET" "/api/v1/iam/drift/tenants" ".tenants" && ((PASS_COUNT++)) || ((FAIL_COUNT++))
test_endpoint "GET /api/v1/iam/drift/tenants/tenant-alpha" "GET" "/api/v1/iam/drift/tenants/tenant-alpha" ".tenant" && ((PASS_COUNT++)) || ((FAIL_COUNT++))

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "6. TROUBLESHOOTING ENDPOINTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

test_endpoint "GET /api/v1/troubleshooting/summary" "GET" "/api/v1/troubleshooting/summary" ".totalFindings" && ((PASS_COUNT++)) || ((FAIL_COUNT++))
test_endpoint "GET /api/v1/troubleshooting/findings" "GET" "/api/v1/troubleshooting/findings" ".total" && ((PASS_COUNT++)) || ((FAIL_COUNT++))
test_endpoint "GET /api/v1/troubleshooting/rules" "GET" "/api/v1/troubleshooting/rules" ".rules" && ((PASS_COUNT++)) || ((FAIL_COUNT++))

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "7. ACTION ENDPOINTS (Read-only test - no actual actions)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo "   (Action endpoints require POST with data - skipping for safety)"
echo "   Available actions:"
echo "   • POST /api/v1/actions/argo/sync"
echo "   • POST /api/v1/actions/argo/refresh"
echo "   • POST /api/v1/actions/crossplane/pause"
echo "   • POST /api/v1/actions/crossplane/unpause"
echo "   • POST /api/v1/actions/crossplane/reconcile"
echo "   • DELETE /api/v1/actions/resources"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "SUMMARY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

TOTAL=$((PASS_COUNT + FAIL_COUNT))
echo ""
echo -e "Total Tests:  $TOTAL"
echo -e "${GREEN}Passed:       $PASS_COUNT${NC}"
if [ $FAIL_COUNT -gt 0 ]; then
    echo -e "${RED}Failed:       $FAIL_COUNT${NC}"
else
    echo -e "${GREEN}Failed:       0${NC}"
fi
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}✓ All endpoints are working with real data!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some endpoints failed - see details above${NC}"
    exit 1
fi

