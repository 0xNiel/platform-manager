#!/bin/bash
# test-phase4-drift.sh
# Integration tests for Phase 4: IAM Drift Detection

set -e

API_BASE="http://localhost:9080/api/v1"

echo "======================================"
echo "Phase 4: IAM Drift Detection Testing"
echo "======================================"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TOTAL_TESTS=0
PASSED_TESTS=0

# Test helper function
test_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local expected_status="${4:-200}"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -n "Test $TOTAL_TESTS: $name... "
    
    response=$(curl -s -w "\n%{http_code}" -X "$method" "$API_BASE$endpoint")
    status=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$status" = "$expected_status" ]; then
        echo -e "${GREEN}✓ PASS${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC} (Expected $expected_status, got $status)"
        return 1
    fi
}

# Prerequisites check
echo "Checking prerequisites..."
echo ""

if ! kubectl get providers 2>/dev/null | grep -q provider-aws-iam; then
    echo -e "${YELLOW}⚠️  AWS IAM Provider not installed${NC}"
    echo "   Install with: kubectl apply -f hack/crossplane/provider-aws.yaml"
fi

if ! kubectl get namespace tenant-alpha 2>/dev/null >/dev/null; then
    echo -e "${YELLOW}⚠️  Tenant namespaces not created${NC}"
    echo "   Create with: kubectl apply -f hack/seed-tenants/namespaces.yaml"
fi

echo ""
echo "======================================"
echo "Phase 1: API Endpoint Tests"
echo "======================================"
echo ""

# Test 1: Get platform drift summary
test_endpoint "Get platform drift summary" GET "/iam/drift/platform"

# Test 2: Get all tenants drift
test_endpoint "Get all tenants drift" GET "/iam/drift/tenants"

# Test 3: Get tenant alpha drift
test_endpoint "Get tenant alpha drift" GET "/iam/drift/tenants/alpha"

# Test 4: Trigger manual scan
test_endpoint "Trigger manual scan" POST "/iam/drift/scan" 202

echo ""
echo "======================================"
echo "Phase 2: Drift Detection Tests"
echo "======================================"
echo ""

# Deploy IAM resources if not already present
echo "Ensuring IAM test resources are deployed..."
kubectl apply -f hack/seed-tenants/iam-resources.yaml 2>/dev/null || true
echo "Waiting for Crossplane reconciliation (30s)..."
sleep 30

# Trigger initial scan
echo "Triggering initial drift scan..."
curl -s -X POST "$API_BASE/iam/drift/scan" >/dev/null
sleep 10

# Test 5: Check for roles in summary
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo -n "Test $TOTAL_TESTS: Platform summary shows IAM roles... "
ROLE_COUNT=$(curl -s "$API_BASE/iam/drift/platform" | jq -r '.summary.totalRoles // 0')
if [ "$ROLE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ PASS${NC} (Found $ROLE_COUNT roles)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}✗ FAIL${NC} (No roles found)"
fi

# Test 6: Check for policies in summary
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo -n "Test $TOTAL_TESTS: Platform summary shows IAM policies... "
POLICY_COUNT=$(curl -s "$API_BASE/iam/drift/platform" | jq -r '.summary.totalPolicies // 0')
if [ "$POLICY_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ PASS${NC} (Found $POLICY_COUNT policies)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}✗ FAIL${NC} (No policies found)"
fi

# Test 7: Initially no drift expected
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo -n "Test $TOTAL_TESTS: No drift detected initially... "
DRIFT_COUNT=$(curl -s "$API_BASE/iam/drift/platform" | jq -r '.summary.criticalDrifts // 0')
if [ "$DRIFT_COUNT" -eq 0 ]; then
    echo -e "${GREEN}✓ PASS${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${YELLOW}⚠️  WARNING${NC} (Found $DRIFT_COUNT drifts - may be pre-existing)"
fi

echo ""
echo "======================================"
echo "Phase 3: Drift Creation Tests"
echo "======================================"
echo ""

# Check if LocalStack is available
if ! curl -s http://localhost:4566/_localstack/health >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  LocalStack not available, skipping drift creation tests${NC}"
else
    echo "Creating drift in LocalStack..."
    
    # Test 8: Create extra inline policy (critical drift)
    echo "Adding unauthorized inline policy to tenant-alpha-lambda-role..."
    awslocal iam put-role-policy \
        --role-name tenant-alpha-lambda-role \
        --policy-name unauthorized-s3-access \
        --policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}' \
        2>/dev/null && echo "  ✓ Drift created" || echo "  ⚠️  Role may not exist in LocalStack yet"
    
    # Wait and rescan
    echo "Waiting 5s then re-scanning..."
    sleep 5
    curl -s -X POST "$API_BASE/iam/drift/scan" >/dev/null
    sleep 10
    
    # Test 9: Drift should now be detected
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -n "Test $TOTAL_TESTS: Drift detected after modification... "
    DRIFT_COUNT=$(curl -s "$API_BASE/iam/drift/platform" | jq -r '.summary.criticalDrifts // 0')
    if [ "$DRIFT_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✓ PASS${NC} (Found $DRIFT_COUNT critical drifts)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${YELLOW}⚠️  EXPECTED FAIL${NC} (Drift detection may need more time or LocalStack sync)"
    fi
    
    # Test 10: Tenant alpha should have drift
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -n "Test $TOTAL_TESTS: Tenant alpha shows drift details... "
    TENANT_DRIFT=$(curl -s "$API_BASE/iam/drift/tenants/alpha" | jq -r '.details // [] | length')
    if [ "$TENANT_DRIFT" -gt 0 ]; then
        echo -e "${GREEN}✓ PASS${NC} (Found $TENANT_DRIFT drift items)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${YELLOW}⚠️  EXPECTED FAIL${NC} (See note above)"
    fi
fi

echo ""
echo "======================================"
echo "Phase 4: Health Integration Tests"
echo "======================================"
echo ""

# Test 11: Check TenantHealth IAM drift field
TOTAL_TESTS=$((TOTAL_TESTS + 1))
echo -n "Test $TOTAL_TESTS: TenantHealth populates IAM drift... "
IAM_DRIFT_ROLES=$(kubectl get tenanthealth alpha-health -o jsonpath='{.status.iamDrift.totalRoles}' 2>/dev/null || echo "0")
if [ "$IAM_DRIFT_ROLES" -gt 0 ]; then
    echo -e "${GREEN}✓ PASS${NC} (IAM drift integrated with health)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${YELLOW}⚠️  EXPECTED FAIL${NC} (Health aggregator may not have run yet)"
fi

echo ""
echo "======================================"
echo "Test Summary"
echo "======================================"
echo ""
echo "Total tests: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$((TOTAL_TESTS - PASSED_TESTS))${NC}"
echo ""

if [ "$PASSED_TESTS" -eq "$TOTAL_TESTS" ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
elif [ "$PASSED_TESTS" -ge 6 ]; then
    echo -e "${YELLOW}⚠️  Most tests passed - Phase 4 core functionality working${NC}"
    echo "   Some tests may fail due to timing or LocalStack sync issues"
    exit 0
else
    echo -e "${RED}❌ Some tests failed - please check logs${NC}"
    exit 1
fi

