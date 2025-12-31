#!/bin/bash
# Phase 3 Actions Testing Script

set -e

API_URL="http://localhost:9080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "======================================"
echo "Phase 3 Actions Testing"
echo "======================================"
echo ""

# Helper function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ PASS${NC}: $2"
    else
        echo -e "${RED}❌ FAIL${NC}: $2"
    fi
}

# Test 1: ArgoCD Refresh (ML role - should succeed)
echo "Test 1: ArgoCD Refresh (ML role)"
echo "======================================"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST ${API_URL}/api/v1/actions/argo/refresh \
  -H "X-Dev-Role: ml" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "alpha-ml-platform",
    "namespace": "argocd"
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ]; then
    print_result 0 "ML user can refresh ArgoCD app"
    echo "Response: $BODY" | jq . 2>/dev/null || echo "$BODY"
else
    print_result 1 "ML user refresh failed with HTTP $HTTP_CODE"
    echo "$BODY"
fi
echo ""

# Test 2: ArgoCD Sync (ML role - should FAIL)
echo "Test 2: ArgoCD Sync (ML role - should be forbidden)"
echo "======================================"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST ${API_URL}/api/v1/actions/argo/sync \
  -H "X-Dev-Role: ml" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "alpha-ml-platform",
    "namespace": "argocd",
    "prune": false,
    "dryRun": false
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "403" ]; then
    print_result 0 "ML user correctly blocked from sync"
    echo "Response: $BODY"
else
    print_result 1 "Expected 403, got HTTP $HTTP_CODE"
    echo "$BODY"
fi
echo ""

# Test 3: ArgoCD Sync (Admin role - should succeed)
echo "Test 3: ArgoCD Sync (Admin role)"
echo "======================================"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST ${API_URL}/api/v1/actions/argo/sync \
  -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "beta-data-pipeline",
    "namespace": "argocd",
    "prune": false,
    "dryRun": false
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ]; then
    print_result 0 "Admin user can sync ArgoCD app"
    echo "Response: $BODY" | jq . 2>/dev/null || echo "$BODY"
else
    print_result 1 "Admin sync failed with HTTP $HTTP_CODE"
    echo "$BODY"
fi
echo ""

# Test 4: List Crossplane IAM Roles
echo "Test 4: Check Crossplane IAM Roles"
echo "======================================"
kubectl get roles.iam.aws.upbound.io 2>/dev/null | head -5 || echo "No IAM roles found"
echo ""

# Test 5: Crossplane Pause (Infra role - should succeed)
echo "Test 5: Crossplane Pause (Infra role)"
echo "======================================"
ROLE_NAME=$(kubectl get roles.iam.aws.upbound.io -o name 2>/dev/null | head -1 | cut -d'/' -f2)
if [ -n "$ROLE_NAME" ]; then
    echo "Pausing role: $ROLE_NAME"
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST ${API_URL}/api/v1/actions/crossplane/pause \
      -H "X-Dev-Role: infra" \
      -H "Content-Type: application/json" \
      -d "{
        \"group\": \"iam.aws.upbound.io\",
        \"version\": \"v1beta1\",
        \"kind\": \"Role\",
        \"name\": \"$ROLE_NAME\"
      }")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | head -n-1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        print_result 0 "Infra user can pause Crossplane resource"
        echo "Response: $BODY" | jq . 2>/dev/null || echo "$BODY"
        
        # Verify pause annotation
        echo "Verifying pause annotation..."
        kubectl get role.iam.aws.upbound.io $ROLE_NAME -o yaml 2>/dev/null | grep "crossplane.io/paused" && print_result 0 "Pause annotation added" || print_result 1 "Pause annotation missing"
    else
        print_result 1 "Pause failed with HTTP $HTTP_CODE"
        echo "$BODY"
    fi
else
    echo "⚠️  No IAM roles found to test pause"
fi
echo ""

# Test 6: Crossplane Unpause (Infra role)
echo "Test 6: Crossplane Unpause (Infra role)"
echo "======================================"
if [ -n "$ROLE_NAME" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST ${API_URL}/api/v1/actions/crossplane/unpause \
      -H "X-Dev-Role: infra" \
      -H "Content-Type: application/json" \
      -d "{
        \"group\": \"iam.aws.upbound.io\",
        \"version\": \"v1beta1\",
        \"kind\": \"Role\",
        \"name\": \"$ROLE_NAME\"
      }")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | head -n-1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        print_result 0 "Infra user can unpause Crossplane resource"
        echo "Response: $BODY" | jq . 2>/dev/null || echo "$BODY"
        
        # Verify pause annotation removed
        echo "Verifying pause annotation removed..."
        kubectl get role.iam.aws.upbound.io $ROLE_NAME -o yaml 2>/dev/null | grep -q "crossplane.io/paused" && print_result 1 "Pause annotation still present" || print_result 0 "Pause annotation removed"
    else
        print_result 1 "Unpause failed with HTTP $HTTP_CODE"
        echo "$BODY"
    fi
else
    echo "⚠️  No IAM roles found to test unpause"
fi
echo ""

# Test 7: Crossplane Force Reconcile (Infra role)
echo "Test 7: Crossplane Force Reconcile (Infra role)"
echo "======================================"
if [ -n "$ROLE_NAME" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST ${API_URL}/api/v1/actions/crossplane/reconcile \
      -H "X-Dev-Role: infra" \
      -H "Content-Type: application/json" \
      -d "{
        \"group\": \"iam.aws.upbound.io\",
        \"version\": \"v1beta1\",
        \"kind\": \"Role\",
        \"name\": \"$ROLE_NAME\"
      }")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | head -n-1)
    
    if [ "$HTTP_CODE" = "200" ]; then
        print_result 0 "Infra user can force reconcile"
        echo "Response: $BODY" | jq . 2>/dev/null || echo "$BODY"
    else
        print_result 1 "Reconcile failed with HTTP $HTTP_CODE"
        echo "$BODY"
    fi
else
    echo "⚠️  No IAM roles found to test reconcile"
fi
echo ""

# Test 8: Create test deployment for delete testing
echo "Test 8: Resource Delete (Admin role)"
echo "======================================"
echo "Creating test deployment..."
kubectl create deployment test-delete --image=nginx -n tenant-alpha --dry-run=client -o yaml | kubectl apply -f - 2>/dev/null
sleep 2

RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE ${API_URL}/api/v1/actions/resources \
  -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{
    "group": "apps",
    "version": "v1",
    "kind": "Deployment",
    "namespace": "tenant-alpha",
    "name": "test-delete",
    "confirm": "test-delete"
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ]; then
    print_result 0 "Admin can delete resources"
    echo "Response: $BODY" | jq . 2>/dev/null || echo "$BODY"
    
    # Verify deletion
    sleep 1
    kubectl get deployment test-delete -n tenant-alpha 2>/dev/null && print_result 1 "Deployment still exists" || print_result 0 "Deployment deleted successfully"
else
    print_result 1 "Delete failed with HTTP $HTTP_CODE"
    echo "$BODY"
fi
echo ""

# Test 9: Delete safeguard test (system namespace)
echo "Test 9: Delete Safeguard - System Namespace (should FAIL)"
echo "======================================"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE ${API_URL}/api/v1/actions/resources \
  -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{
    "group": "v1",
    "version": "v1",
    "kind": "Service",
    "namespace": "kube-system",
    "name": "kube-dns",
    "confirm": "kube-dns"
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "403" ]; then
    print_result 0 "Safeguard correctly blocked system namespace deletion"
    echo "Response: $BODY"
else
    print_result 1 "Safeguard failed - expected 403, got HTTP $HTTP_CODE"
    echo "$BODY"
fi
echo ""

# Test 10: Readonly user (should fail everything)
echo "Test 10: Readonly User - Should be blocked"
echo "======================================"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST ${API_URL}/api/v1/actions/argo/refresh \
  -H "X-Dev-Role: readonly" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "alpha-ml-platform",
    "namespace": "argocd"
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "403" ]; then
    print_result 0 "Readonly user correctly blocked"
else
    print_result 1 "Readonly user should be blocked, got HTTP $HTTP_CODE"
fi
echo ""

# Summary
echo "======================================"
echo "Testing Complete!"
echo "======================================"
echo ""
echo "Check audit logs with:"
echo "  ps aux | grep 'bin/manager' | grep -v grep"
echo "  tail -f /tmp/manager.log (if redirected)"
echo ""

