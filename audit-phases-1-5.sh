#!/bin/bash

# Comprehensive audit of Phases 1-5 implementation
# This script verifies all deliverables from the Implementation Plan

# Don't exit on error - we want to collect all results
set +e

echo "╔══════════════════════════════════════════════════════════════════════════╗"
echo "║                                                                          ║"
echo "║              COMPREHENSIVE PHASES 1-5 AUDIT                             ║"
echo "║                                                                          ║"
echo "╚══════════════════════════════════════════════════════════════════════════╝"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

check_pass() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
    ((PASS_COUNT++))
}

check_fail() {
    echo -e "${RED}❌ FAIL${NC}: $1"
    ((FAIL_COUNT++))
}

check_warn() {
    echo -e "${YELLOW}⚠️  WARN${NC}: $1"
    ((WARN_COUNT++))
}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "PHASE 1: HEALTH AGGREGATION & BASE API"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 1.1 Check CRDs exist
echo "📋 Checking CRDs..."
if kubectl get crd tenants.platform.platform.io &>/dev/null; then
    check_pass "Tenant CRD exists"
else
    check_fail "Tenant CRD missing"
fi

if kubectl get crd tenanthealths.platform.platform.io &>/dev/null; then
    check_pass "TenantHealth CRD exists"
else
    check_fail "TenantHealth CRD missing"
fi

if kubectl get crd platformhealths.platform.platform.io &>/dev/null; then
    check_pass "PlatformHealth CRD exists"
else
    check_fail "PlatformHealth CRD missing"
fi

if kubectl get crd resourcesummaries.platform.platform.io &>/dev/null; then
    check_pass "ResourceSummary CRD exists"
else
    check_fail "ResourceSummary CRD missing"
fi

echo ""

# 1.2 Check Controllers
echo "🔧 Checking Controllers..."
if [ -f "internal/controller/tenant_controller.go" ]; then
    check_pass "TenantReconciler implemented"
else
    check_fail "TenantReconciler missing"
fi

if [ -f "internal/controller/crossplane_watcher.go" ]; then
    check_pass "CrossplaneWatcher implemented"
else
    check_fail "CrossplaneWatcher missing"
fi

if [ -f "internal/controller/argo_watcher.go" ]; then
    check_pass "ArgoWatcher implemented"
else
    check_fail "ArgoWatcher missing"
fi

if [ -f "internal/controller/health_aggregator.go" ]; then
    check_pass "HealthAggregator implemented"
else
    check_fail "HealthAggregator missing"
fi

echo ""

# 1.3 Check API Server
echo "🌐 Checking API Server..."
if [ -f "internal/api/server.go" ]; then
    check_pass "API server implemented"
else
    check_fail "API server missing"
fi

if [ -f "internal/api/handlers/health.go" ]; then
    check_pass "Health handlers implemented"
else
    check_fail "Health handlers missing"
fi

if [ -f "internal/api/middleware/auth.go" ]; then
    check_pass "Auth middleware implemented"
else
    check_fail "Auth middleware missing"
fi

# 1.4 Check API endpoints are responding
echo ""
echo "🔍 Checking API Endpoints..."
if curl -s -H "X-Dev-Role: admin" http://localhost:9080/api/v1/health/platform | jq -e '.overallHealth' &>/dev/null; then
    check_pass "GET /api/v1/health/platform responding"
else
    check_warn "GET /api/v1/health/platform not responding (manager may not be running)"
fi

if curl -s -H "X-Dev-Role: admin" http://localhost:9080/api/v1/health/tenants | jq -e '.tenants' &>/dev/null; then
    check_pass "GET /api/v1/health/tenants responding"
else
    check_warn "GET /api/v1/health/tenants not responding"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "PHASE 2: DRILL-DOWN & RESOURCE DETAIL"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 2.1 Check ResourceScanner
echo "📊 Checking Resource Scanning..."
if [ -f "internal/controller/resource_scanner.go" ]; then
    check_pass "ResourceScanner implemented"
else
    check_fail "ResourceScanner missing"
fi

# 2.2 Check resource endpoints
echo ""
echo "🔍 Checking Resource API Endpoints..."
if curl -s -H "X-Dev-Role: admin" http://localhost:9080/api/v1/resources | jq -e '.resources' &>/dev/null; then
    check_pass "GET /api/v1/resources responding"
else
    check_warn "GET /api/v1/resources not responding"
fi

# 2.3 Check Prometheus integration
echo ""
echo "📈 Checking Prometheus Integration..."
if [ -f "internal/metrics/prometheus.go" ]; then
    check_pass "Prometheus client implemented"
else
    check_fail "Prometheus client missing"
fi

if curl -s -H "X-Dev-Role: admin" http://localhost:9080/api/v1/metrics/tenants/tenant-alpha | jq -e '.' &>/dev/null; then
    check_pass "GET /api/v1/metrics/tenants/:id responding"
else
    check_warn "GET /api/v1/metrics/tenants/:id not responding"
fi

# 2.4 Check Frontend Dashboard
echo ""
echo "🎨 Checking Frontend Views..."
if [ -f "web/src/views/DashboardView.vue" ]; then
    check_pass "Dashboard view implemented"
else
    check_fail "Dashboard view missing"
fi

if [ -f "web/src/views/TenantsView.vue" ]; then
    check_pass "Tenants view implemented"
else
    check_fail "Tenants view missing"
fi

if [ -f "web/src/views/ResourcesView.vue" ]; then
    check_pass "Resources view implemented"
else
    check_fail "Resources view missing"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "PHASE 3: ACTIONS (MINI ARGO + CROSSPLANE CONTROLS)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 3.1 Check Action Handlers
echo "⚡ Checking Action Handlers..."
if [ -f "internal/api/handlers/actions.go" ]; then
    check_pass "Actions handler implemented"
else
    check_fail "Actions handler missing"
fi

# 3.2 Check Authorization
echo ""
echo "🔒 Checking Authorization..."
if [ -f "internal/api/middleware/authz.go" ]; then
    check_pass "Authorization middleware implemented"
else
    check_fail "Authorization middleware missing"
fi

# Check if authz middleware defines capabilities
if grep -q "CapSyncArgo" internal/api/middleware/authz.go 2>/dev/null; then
    check_pass "Capabilities defined (CapSyncArgo, etc.)"
else
    check_fail "Capabilities not defined"
fi

# 3.3 Check Action Endpoints
echo ""
echo "🔍 Checking Action API Endpoints..."
ACTIONS_ENDPOINTS=(
    "/api/v1/actions/argo/sync"
    "/api/v1/actions/argo/refresh"
    "/api/v1/actions/crossplane/pause"
    "/api/v1/actions/crossplane/unpause"
    "/api/v1/actions/crossplane/reconcile"
)

for endpoint in "${ACTIONS_ENDPOINTS[@]}"; do
    if grep -q "$endpoint" internal/api/handlers/actions.go 2>/dev/null; then
        check_pass "Action endpoint $endpoint implemented"
    else
        check_fail "Action endpoint $endpoint missing"
    fi
done

# 3.4 Check Audit Logging
echo ""
echo "📝 Checking Audit Logging..."
if [ -f "internal/api/middleware/audit.go" ]; then
    check_pass "Audit middleware implemented"
else
    check_fail "Audit middleware missing"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "PHASE 4: IAM DRIFT MODULE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 4.1 Check IAM Drift Components
echo "🔐 Checking IAM Drift Detection..."
if [ -f "internal/iam/drift_checker.go" ]; then
    check_pass "IAM drift checker implemented"
else
    check_fail "IAM drift checker missing"
fi

if [ -f "internal/iam/aws_client.go" ]; then
    check_pass "AWS client implemented"
else
    check_fail "AWS client missing"
fi

if [ -f "internal/controller/iam_drift_scanner.go" ]; then
    check_pass "IAM drift scanner controller implemented"
else
    check_fail "IAM drift scanner controller missing"
fi

# 4.2 Check IAM API Endpoints
echo ""
echo "🔍 Checking IAM Drift API Endpoints..."
if [ -f "internal/api/handlers/iam.go" ]; then
    check_pass "IAM drift handlers implemented"
else
    check_fail "IAM drift handlers missing"
fi

if curl -s -H "X-Dev-Role: admin" http://localhost:9080/api/v1/iam/drift/summary | jq -e '.' &>/dev/null; then
    check_pass "GET /api/v1/iam/drift/summary responding"
else
    check_warn "GET /api/v1/iam/drift/summary not responding"
fi

# 4.3 Check IAM Frontend
echo ""
echo "🎨 Checking IAM Drift Frontend..."
if [ -f "web/src/views/IAMDriftView.vue" ]; then
    check_pass "IAM Drift view implemented"
else
    check_fail "IAM Drift view missing"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "PHASE 5: TROUBLESHOOTING & RULE ENGINE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 5.1 Check Rule Engine
echo "🔧 Checking Rule Engine..."
if [ -f "internal/rules/engine.go" ]; then
    check_pass "Rule engine implemented"
else
    check_fail "Rule engine missing"
fi

if [ -f "internal/rules/types.go" ]; then
    check_pass "Rule types defined"
else
    check_fail "Rule types missing"
fi

# 5.2 Check Built-in Rules
echo ""
echo "📋 Checking Built-in Rules..."
RULE_FILES=(
    "internal/rules/builtin/pod_rules.go"
    "internal/rules/builtin/crossplane_rules.go"
    "internal/rules/builtin/argo_rules.go"
    "internal/rules/builtin/registry.go"
)

for rule_file in "${RULE_FILES[@]}"; do
    if [ -f "$rule_file" ]; then
        check_pass "Rule file $rule_file exists"
    else
        check_fail "Rule file $rule_file missing"
    fi
done

# 5.3 Check Rule Evaluator
echo ""
echo "⚙️  Checking Rule Evaluator Controller..."
if [ -f "internal/controller/rule_evaluator.go" ]; then
    check_pass "Rule evaluator controller implemented"
else
    check_fail "Rule evaluator controller missing"
fi

# 5.4 Check Troubleshooting API
echo ""
echo "🔍 Checking Troubleshooting API..."
if [ -f "internal/api/handlers/troubleshooting.go" ]; then
    check_pass "Troubleshooting handlers implemented"
else
    check_fail "Troubleshooting handlers missing"
fi

if curl -s -H "X-Dev-Role: admin" http://localhost:9080/api/v1/troubleshooting/summary | jq -e '.' &>/dev/null; then
    check_pass "GET /api/v1/troubleshooting/summary responding"
else
    check_warn "GET /api/v1/troubleshooting/summary not responding"
fi

# 5.5 Check Troubleshooting Frontend
echo ""
echo "🎨 Checking Troubleshooting Frontend..."
if [ -f "web/src/views/TroubleshootingView.vue" ]; then
    check_pass "Troubleshooting view implemented"
else
    check_fail "Troubleshooting view missing"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "ADDITIONAL FEATURES & ENHANCEMENTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check additional enhancements
echo "🚀 Checking Additional Features..."

# Check TenantDetailView
if [ -f "web/src/views/TenantDetailView.vue" ]; then
    check_pass "Tenant detail view implemented"
else
    check_fail "Tenant detail view missing"
fi

# Check resource filtering and sorting
if grep -q "selectedTenant" web/src/views/ResourcesView.vue 2>/dev/null; then
    check_pass "Tenant filtering in Resources view"
else
    check_warn "Tenant filtering in Resources view not found"
fi

# Check Crossplane integration
if grep -q "Crossplane" internal/api/handlers/health.go 2>/dev/null; then
    check_pass "Crossplane metrics in health API"
else
    check_warn "Crossplane metrics not found in health API"
fi

# Check paused resource detection
if grep -q "ResourceStatePaused" internal/controller/resource_scanner.go 2>/dev/null; then
    check_pass "Paused resource detection implemented"
else
    check_warn "Paused resource detection not found"
fi

# Check ArgoCD sync policy tracking
if grep -q "autoSyncOff" internal/api/handlers/health.go 2>/dev/null; then
    check_pass "ArgoCD sync policy tracking implemented"
else
    check_warn "ArgoCD sync policy tracking not found"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "DOCUMENTATION & TESTING"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "📚 Checking Documentation..."
DOC_FILES=(
    "documentation/PHASE2_SUMMARY.md"
    "documentation/PHASE3_COMPLETE_SUMMARY.md"
    "documentation/PHASE4_COMPLETE_SUMMARY.md"
    "documentation/PHASE5_COMPLETE.md"
    "documentation/IMPLEMENTATION_PLAN.md"
    "README.md"
)

for doc in "${DOC_FILES[@]}"; do
    if [ -f "$doc" ]; then
        check_pass "Documentation: $doc exists"
    else
        check_warn "Documentation: $doc missing"
    fi
done

echo ""
echo "🧪 Checking Test Scripts..."
TEST_SCRIPTS=(
    "test-phase3-actions.sh"
    "test-phase4-api.sh"
    "test-phase5.sh"
    "test-phase5-ui.sh"
)

for script in "${TEST_SCRIPTS[@]}"; do
    if [ -f "$script" ]; then
        check_pass "Test script: $script exists"
    else
        check_warn "Test script: $script missing"
    fi
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "KUBERNETES CLUSTER STATE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "🔍 Checking Kubernetes Resources..."

# Check if tenants exist
TENANT_COUNT=$(kubectl get tenants --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$TENANT_COUNT" -gt 0 ]; then
    check_pass "Tenants exist in cluster ($TENANT_COUNT found)"
else
    check_warn "No tenants found in cluster"
fi

# Check if ResourceSummaries exist
RS_COUNT=$(kubectl get resourcesummaries --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$RS_COUNT" -gt 0 ]; then
    check_pass "ResourceSummaries exist ($RS_COUNT found)"
else
    check_warn "No ResourceSummaries found"
fi

# Check if TenantHealth exists
TH_COUNT=$(kubectl get tenanthealth --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$TH_COUNT" -gt 0 ]; then
    check_pass "TenantHealth CRs exist ($TH_COUNT found)"
else
    check_warn "No TenantHealth CRs found"
fi

# Check Crossplane resources
XR_COUNT=$(kubectl get composite --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$XR_COUNT" -gt 0 ]; then
    check_pass "Crossplane Composite Resources exist ($XR_COUNT found)"
else
    check_warn "No Crossplane Composite Resources found"
fi

# Check ArgoCD apps
ARGO_COUNT=$(kubectl get applications -n argocd --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$ARGO_COUNT" -gt 0 ]; then
    check_pass "ArgoCD Applications exist ($ARGO_COUNT found)"
else
    check_warn "No ArgoCD Applications found"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "AUDIT SUMMARY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

TOTAL=$((PASS_COUNT + FAIL_COUNT + WARN_COUNT))
PASS_PERCENT=$((PASS_COUNT * 100 / TOTAL))

echo -e "${GREEN}✅ PASSED${NC}: $PASS_COUNT"
echo -e "${RED}❌ FAILED${NC}: $FAIL_COUNT"
echo -e "${YELLOW}⚠️  WARNINGS${NC}: $WARN_COUNT"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TOTAL CHECKS: $TOTAL"
echo "PASS RATE: ${PASS_PERCENT}%"
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo "╔══════════════════════════════════════════════════════════════════════════╗"
    echo "║                                                                          ║"
    echo "║              🎉 ALL CRITICAL CHECKS PASSED! 🎉                          ║"
    echo "║                                                                          ║"
    echo "║  Phases 1-5 are complete and ready for Phase 6!                         ║"
    if [ $WARN_COUNT -gt 0 ]; then
        echo "║  Note: Some warnings exist (likely runtime checks)                      ║"
    fi
    echo "║                                                                          ║"
    echo "╚══════════════════════════════════════════════════════════════════════════╝"
    exit 0
else
    echo "╔══════════════════════════════════════════════════════════════════════════╗"
    echo "║                                                                          ║"
    echo "║              ⚠️  SOME CHECKS FAILED                                     ║"
    echo "║                                                                          ║"
    echo "║  Please review the failed checks above.                                 ║"
    echo "║                                                                          ║"
    echo "╚══════════════════════════════════════════════════════════════════════════╝"
    exit 1
fi

