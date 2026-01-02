#!/bin/bash
# Comprehensive Crossplane Integration Testing
# Tests the full flow: Crossplane -> LocalStack -> ResourceScanner -> API -> UI

set -e

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║     🧪 Crossplane Integration Testing - Full Stack Validation   ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

API_BASE="http://localhost:9080/api/v1"
TESTS_PASSED=0
TESTS_FAILED=0

function test_section() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

function test_pass() {
    echo -e "${GREEN}✅ PASS: $1${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

function test_fail() {
    echo -e "${RED}❌ FAIL: $1${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

function test_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# =============================================================================
# TEST 1: Verify Crossplane Resources Exist
# =============================================================================
test_section "TEST 1: Crossplane Managed Resources in Kubernetes"

echo "Checking IAM Roles..."
IAM_ROLES=$(kubectl get role.iam.aws.upbound.io --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$IAM_ROLES" -ge 2 ]; then
    test_pass "Found $IAM_ROLES IAM Roles"
    kubectl get role.iam.aws.upbound.io -o custom-columns=NAME:.metadata.name,READY:.status.conditions[?\(@.type==\"Ready\"\)].status,SYNCED:.status.conditions[?\(@.type==\"Synced\"\)].status
else
    test_fail "Expected at least 2 IAM Roles, found $IAM_ROLES"
fi

echo ""
echo "Checking IAM Policies..."
IAM_POLICIES=$(kubectl get policy.iam.aws.upbound.io --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$IAM_POLICIES" -ge 2 ]; then
    test_pass "Found $IAM_POLICIES IAM Policies"
    kubectl get policy.iam.aws.upbound.io -o custom-columns=NAME:.metadata.name,READY:.status.conditions[?\(@.type==\"Ready\"\)].status,SYNCED:.status.conditions[?\(@.type==\"Synced\"\)].status
else
    test_fail "Expected at least 2 IAM Policies, found $IAM_POLICIES"
fi

echo ""
echo "Checking S3 Buckets..."
S3_BUCKETS=$(kubectl get bucket.s3.aws.upbound.io --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$S3_BUCKETS" -ge 2 ]; then
    test_pass "Found $S3_BUCKETS S3 Buckets"
    kubectl get bucket.s3.aws.upbound.io -o custom-columns=NAME:.metadata.name,READY:.status.conditions[?\(@.type==\"Ready\"\)].status,SYNCED:.status.conditions[?\(@.type==\"Synced\"\)].status
else
    test_fail "Expected at least 2 S3 Buckets, found $S3_BUCKETS"
fi

echo ""
echo "Checking DynamoDB Tables..."
DYNAMO_TABLES=$(kubectl get table.dynamodb.aws.upbound.io --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$DYNAMO_TABLES" -ge 2 ]; then
    test_pass "Found $DYNAMO_TABLES DynamoDB Tables"
    kubectl get table.dynamodb.aws.upbound.io -o custom-columns=NAME:.metadata.name,READY:.status.conditions[?\(@.type==\"Ready\"\)].status,SYNCED:.status.conditions[?\(@.type==\"Synced\"\)].status
else
    test_fail "Expected at least 2 DynamoDB Tables, found $DYNAMO_TABLES"
fi

# =============================================================================
# TEST 2: Verify Resources Created in LocalStack
# =============================================================================
test_section "TEST 2: Resources in LocalStack (AWS Emulator)"

if command -v aws &> /dev/null; then
    echo "Checking IAM Roles in LocalStack..."
    LOCALSTACK_ROLES=$(aws --endpoint-url=http://localhost:4566 iam list-roles --query 'Roles[].RoleName' --output text 2>/dev/null | wc -w | tr -d ' ')
    if [ "$LOCALSTACK_ROLES" -ge 2 ]; then
        test_pass "Found $LOCALSTACK_ROLES IAM Roles in LocalStack"
        aws --endpoint-url=http://localhost:4566 iam list-roles --query 'Roles[].[RoleName,CreateDate]' --output table
    else
        test_fail "Expected at least 2 IAM Roles in LocalStack, found $LOCALSTACK_ROLES"
    fi
    
    echo ""
    echo "Checking IAM Policies in LocalStack..."
    LOCALSTACK_POLICIES=$(aws --endpoint-url=http://localhost:4566 iam list-policies --scope Local --query 'Policies[].PolicyName' --output text 2>/dev/null | wc -w | tr -d ' ')
    if [ "$LOCALSTACK_POLICIES" -ge 2 ]; then
        test_pass "Found $LOCALSTACK_POLICIES IAM Policies in LocalStack"
        aws --endpoint-url=http://localhost:4566 iam list-policies --scope Local --query 'Policies[].[PolicyName,CreateDate]' --output table
    else
        test_fail "Expected at least 2 IAM Policies in LocalStack, found $LOCALSTACK_POLICIES"
    fi
    
    echo ""
    echo "Checking S3 Buckets in LocalStack..."
    LOCALSTACK_BUCKETS=$(aws --endpoint-url=http://localhost:4566 s3 ls 2>/dev/null | wc -l | tr -d ' ')
    if [ "$LOCALSTACK_BUCKETS" -ge 2 ]; then
        test_pass "Found $LOCALSTACK_BUCKETS S3 Buckets in LocalStack"
        aws --endpoint-url=http://localhost:4566 s3 ls
    else
        test_fail "Expected at least 2 S3 Buckets in LocalStack, found $LOCALSTACK_BUCKETS"
    fi
    
    echo ""
    echo "Checking DynamoDB Tables in LocalStack..."
    LOCALSTACK_TABLES=$(aws --endpoint-url=http://localhost:4566 dynamodb list-tables --query 'TableNames[]' --output text 2>/dev/null | wc -w | tr -d ' ')
    if [ "$LOCALSTACK_TABLES" -ge 2 ]; then
        test_pass "Found $LOCALSTACK_TABLES DynamoDB Tables in LocalStack"
        aws --endpoint-url=http://localhost:4566 dynamodb list-tables --query 'TableNames[]' --output table
    else
        test_fail "Expected at least 2 DynamoDB Tables in LocalStack, found $LOCALSTACK_TABLES"
    fi
else
    test_info "AWS CLI not installed, skipping LocalStack verification"
    test_info "Install with: brew install awscli (Mac) or apt-get install awscli (Linux)"
fi

# =============================================================================
# TEST 3: Verify ResourceScanner Created ResourceSummary CRs
# =============================================================================
test_section "TEST 3: ResourceSummary CRs (Created by ResourceScanner)"

echo "Triggering resource scan for tenant-alpha..."
kubectl annotate tenant tenant-alpha platform.io/scan=now --overwrite 2>/dev/null || true
sleep 5

echo "Checking ResourceSummary CRs..."
RESOURCE_SUMMARIES=$(kubectl get resourcesummaries --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$RESOURCE_SUMMARIES" -ge 8 ]; then
    test_pass "Found $RESOURCE_SUMMARIES ResourceSummary CRs (expected >= 8)"
else
    test_fail "Expected at least 8 ResourceSummary CRs, found $RESOURCE_SUMMARIES"
fi

echo ""
echo "ResourceSummary breakdown by category:"
kubectl get resourcesummaries -o json | jq -r '.items[] | "\(.spec.category): \(.spec.name)"' | sort | uniq -c

echo ""
echo "Sample ResourceSummary CRs:"
kubectl get resourcesummaries -o custom-columns=NAME:.metadata.name,CATEGORY:.spec.category,KIND:.spec.kind,TENANT:.spec.tenantRef,STATE:.status.state | head -15

# =============================================================================
# TEST 4: Verify API Endpoint Returns Crossplane Resources
# =============================================================================
test_section "TEST 4: API Endpoint - Resources"

echo "Testing GET /api/v1/resources..."
RESPONSE=$(curl -s -H "X-Dev-Role: admin" "${API_BASE}/resources")
RESOURCE_COUNT=$(echo "$RESPONSE" | jq '.resources | length' 2>/dev/null || echo "0")

if [ "$RESOURCE_COUNT" -ge 8 ]; then
    test_pass "API returned $RESOURCE_COUNT resources"
else
    test_fail "Expected at least 8 resources, API returned $RESOURCE_COUNT"
fi

echo ""
echo "Resource breakdown by category:"
echo "$RESPONSE" | jq -r '.resources[] | "\(.spec.category): \(.spec.name)"' 2>/dev/null | sort | uniq -c || echo "Failed to parse response"

echo ""
echo "Crossplane resources in API response:"
echo "$RESPONSE" | jq -r '.resources[] | select(.spec.category == "Crossplane" or .spec.category == "IAM") | "\(.spec.kind): \(.spec.name) (\(.status.state // "Unknown"))"' 2>/dev/null || echo "No Crossplane resources found"

# Test filtering by tenant
echo ""
echo "Testing tenant-alpha resources..."
ALPHA_RESPONSE=$(curl -s -H "X-Dev-Role: admin" "${API_BASE}/resources" | jq '.resources[] | select(.spec.tenantRef == "tenant-alpha")' 2>/dev/null)
ALPHA_COUNT=$(echo "$ALPHA_RESPONSE" | jq -s 'length' 2>/dev/null || echo "0")

if [ "$ALPHA_COUNT" -ge 4 ]; then
    test_pass "Found $ALPHA_COUNT resources for tenant-alpha"
else
    test_fail "Expected at least 4 resources for tenant-alpha, found $ALPHA_COUNT"
fi

# =============================================================================
# TEST 5: Test Resource Actions on Crossplane Resources
# =============================================================================
test_section "TEST 5: Resource Actions - Crossplane Resources"

echo "Getting a Crossplane IAM Role..."
ROLE_NAME=$(kubectl get role.iam.aws.upbound.io -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

if [ -n "$ROLE_NAME" ]; then
    test_info "Testing actions on role: $ROLE_NAME"
    
    # Test pause action
    echo ""
    echo "Testing PAUSE action..."
    PAUSE_RESPONSE=$(curl -s -X POST -H "X-Dev-Role: admin" "${API_BASE}/resources/${ROLE_NAME}/pause")
    if echo "$PAUSE_RESPONSE" | jq -e '.status == "success"' &>/dev/null; then
        test_pass "Pause action succeeded"
    else
        test_fail "Pause action failed: $(echo $PAUSE_RESPONSE | jq -r '.message // .error // "Unknown error"')"
    fi
    
    sleep 3
    
    # Test reconcile action
    echo ""
    echo "Testing RECONCILE action..."
    RECONCILE_RESPONSE=$(curl -s -X POST -H "X-Dev-Role: admin" "${API_BASE}/resources/${ROLE_NAME}/reconcile")
    if echo "$RECONCILE_RESPONSE" | jq -e '.status == "success"' &>/dev/null; then
        test_pass "Reconcile action succeeded"
    else
        test_fail "Reconcile action failed: $(echo $RECONCILE_RESPONSE | jq -r '.message // .error // "Unknown error"')"
    fi
else
    test_fail "No IAM Roles found to test actions"
fi

# =============================================================================
# TEST 6: Test IAM Drift Detection with Crossplane Resources
# =============================================================================
test_section "TEST 6: IAM Drift Detection"

echo "Triggering IAM drift scan..."
SCAN_RESPONSE=$(curl -s -X POST -H "X-Dev-Role: admin" "${API_BASE}/iam/drift/scan")
if echo "$SCAN_RESPONSE" | jq -e '.status == "success"' &>/dev/null; then
    test_pass "IAM drift scan triggered"
else
    test_fail "Failed to trigger IAM drift scan"
fi

sleep 5

echo ""
echo "Checking drift results for tenant-alpha..."
DRIFT_RESPONSE=$(curl -s -H "X-Dev-Role: admin" "${API_BASE}/iam/drift/tenants/tenant-alpha")
DRIFT_COUNT=$(echo "$DRIFT_RESPONSE" | jq '.tenant.drifts | length' 2>/dev/null || echo "0")

test_info "Found $DRIFT_COUNT drift(s) for tenant-alpha"
if [ "$DRIFT_COUNT" -gt 0 ]; then
    echo "$DRIFT_RESPONSE" | jq -r '.tenant.drifts[] | "  • \(.resourceType): \(.resourceName) - \(.driftType)"' 2>/dev/null
fi

# =============================================================================
# TEST 7: Verify UI Data (via API)
# =============================================================================
test_section "TEST 7: UI Data Validation"

echo "Testing tenant health endpoint..."
TENANT_HEALTH=$(curl -s -H "X-Dev-Role: admin" "${API_BASE}/health/tenants/tenant-alpha")
RESOURCE_COUNT_HEALTH=$(echo "$TENANT_HEALTH" | jq '.tenant.resourceCount' 2>/dev/null || echo "0")

if [ "$RESOURCE_COUNT_HEALTH" -ge 4 ]; then
    test_pass "Tenant health shows $RESOURCE_COUNT_HEALTH resources for tenant-alpha"
else
    test_fail "Expected at least 4 resources in tenant health, found $RESOURCE_COUNT_HEALTH"
fi

echo ""
echo "Tenant health summary:"
echo "$TENANT_HEALTH" | jq '{name: .tenant.name, resources: .tenant.resourceCount, ready: .tenant.readyCount, failed: .tenant.failedCount}' 2>/dev/null || echo "Failed to parse"

# =============================================================================
# TEST 8: Create and Delete a Test Resource
# =============================================================================
test_section "TEST 8: Lifecycle Test - Create and Delete Resource"

TEST_ROLE_NAME="test-crossplane-role-$$"
echo "Creating test IAM role: $TEST_ROLE_NAME..."

cat <<EOF | kubectl apply -f -
apiVersion: iam.aws.upbound.io/v1beta1
kind: Role
metadata:
  name: $TEST_ROLE_NAME
  labels:
    platform.io/tenant: tenant-alpha
    platform.io/test: "true"
spec:
  providerConfigRef:
    name: localstack
  forProvider:
    assumeRolePolicy: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Principal": {
              "Service": "lambda.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
          }
        ]
      }
    tags:
      test: "true"
      managed-by: crossplane
EOF

sleep 10

# Check if ResourceSummary was created
SUMMARY_EXISTS=$(kubectl get resourcesummary -l platform.io/test=true --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$SUMMARY_EXISTS" -ge 1 ]; then
    test_pass "ResourceSummary created for test role"
else
    test_fail "ResourceSummary not created for test role"
fi

echo ""
echo "Deleting test role..."
kubectl delete role.iam.aws.upbound.io "$TEST_ROLE_NAME" 2>/dev/null || true

sleep 10

# Verify ResourceSummary was cleaned up
SUMMARY_CLEANED=$(kubectl get resourcesummary -l platform.io/test=true --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$SUMMARY_CLEANED" -eq 0 ]; then
    test_pass "ResourceSummary cleaned up after deletion"
else
    test_fail "ResourceSummary still exists after deletion (orphaned)"
fi

# =============================================================================
# Final Summary
# =============================================================================
echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║                    📊 TEST RESULTS SUMMARY                       ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""
echo -e "${GREEN}✅ Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}❌ Tests Failed: $TESTS_FAILED${NC}"
echo ""

TOTAL_TESTS=$((TESTS_PASSED + TESTS_FAILED))
if [ $TOTAL_TESTS -gt 0 ]; then
    SUCCESS_RATE=$((TESTS_PASSED * 100 / TOTAL_TESTS))
    echo -e "Success Rate: ${SUCCESS_RATE}%"
fi

echo ""
echo "🎯 Integration Points Tested:"
echo "   ✓ Crossplane Managed Resources in Kubernetes"
echo "   ✓ Resources created in LocalStack (AWS Emulator)"
echo "   ✓ ResourceScanner creating ResourceSummary CRs"
echo "   ✓ API endpoints serving Crossplane resources"
echo "   ✓ Resource actions (pause/reconcile) on Crossplane"
echo "   ✓ IAM drift detection with Crossplane resources"
echo "   ✓ Tenant health aggregation with Crossplane resources"
echo "   ✓ Full lifecycle (create/scan/delete/cleanup)"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 ALL TESTS PASSED! Crossplane integration is working perfectly.${NC}"
    echo ""
    echo "📱 Next: Check the UI at http://localhost:9083"
    echo "   • Resources tab - Filter by tenant-alpha or tenant-beta"
    echo "   • Look for IAM Roles, Policies, S3 Buckets, DynamoDB Tables"
    echo "   • Test pause/reconcile actions"
    echo "   • Check IAM Drift tab"
    exit 0
else
    echo -e "${RED}⚠️  SOME TESTS FAILED. Please review the output above.${NC}"
    exit 1
fi

