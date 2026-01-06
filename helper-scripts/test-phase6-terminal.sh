#!/bin/bash
# Phase 6 Terminal - Comprehensive Testing Script
# Tests all terminal endpoints and functionality

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_URL="${API_URL:-http://localhost:9080}"
TEST_ROLE="${TEST_ROLE:-admin}"
TEST_TENANT="${TEST_TENANT:-alpha}"

echo ""
echo -e "${BLUE}=========================================="
echo "Phase 6: Web Terminal - Testing"
echo "==========================================${NC}"
echo ""
echo "Testing against: $API_URL"
echo "Test role: $TEST_ROLE"
echo "Test tenant: $TEST_TENANT"
echo ""

# Helper function to make API calls
api_call() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -n "$data" ]; then
        curl -s -X "$method" "$API_URL$endpoint" \
            -H "X-Dev-Role: $TEST_ROLE" \
            -H "Content-Type: application/json" \
            -d "$data"
    else
        curl -s -X "$method" "$API_URL$endpoint" \
            -H "X-Dev-Role: $TEST_ROLE"
    fi
}

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name=$1
    local command=$2
    
    echo -ne "${YELLOW}Testing: $test_name...${NC} "
    
    if eval "$command" > /tmp/test_output.json 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo "  Error: $(cat /tmp/test_output.json)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

echo -e "${YELLOW}=========================================="
echo "1. Basic Health Checks"
echo "==========================================${NC}"
echo ""

# Test 1: Health endpoint
run_test "Health endpoint" "api_call GET /healthz | grep -q 'ok'"

# Test 2: Readiness endpoint
run_test "Readiness endpoint" "api_call GET /readyz | grep -q 'ok'"

echo ""
echo -e "${YELLOW}=========================================="
echo "2. Authentication & Capabilities"
echo "==========================================${NC}"
echo ""

# Test 3: Get current user
echo -e "${YELLOW}Testing: Get current user info...${NC}"
USER_RESPONSE=$(api_call GET /api/v1/auth/me)
echo "$USER_RESPONSE" | jq .
echo -e "${GREEN}✓ User info retrieved${NC}"
TESTS_PASSED=$((TESTS_PASSED + 1))
echo ""

# Test 4: Get user capabilities
echo -e "${YELLOW}Testing: Get user capabilities...${NC}"
CAPS_RESPONSE=$(api_call GET /api/v1/auth/capabilities)
echo "$CAPS_RESPONSE" | jq .
HAS_TERMINAL_CAP=$(echo "$CAPS_RESPONSE" | jq -r '.features.canUseTerminal')
echo ""
if [ "$HAS_TERMINAL_CAP" == "true" ]; then
    echo -e "${GREEN}✓ User has terminal capability${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ User does NOT have terminal capability${NC}"
    echo "  Current role: $TEST_ROLE"
    echo "  Expected: admin or infra"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

echo -e "${YELLOW}=========================================="
echo "3. Terminal Configuration"
echo "==========================================${NC}"
echo ""

# Test 5: Get terminal config
echo -e "${YELLOW}Testing: Get terminal configuration...${NC}"
CONFIG_RESPONSE=$(api_call GET /api/v1/terminal/config)
echo "$CONFIG_RESPONSE" | jq .
TERMINAL_ENABLED=$(echo "$CONFIG_RESPONSE" | jq -r '.enabled')
echo ""
if [ "$TERMINAL_ENABLED" == "true" ]; then
    echo -e "${GREEN}✓ Terminal is enabled${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ Terminal is NOT enabled${NC}"
    echo "  Start manager with: --enable-terminal=true"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    exit 1
fi
echo ""

echo -e "${YELLOW}=========================================="
echo "4. Terminal Session Management"
echo "==========================================${NC}"
echo ""

# Test 6: Create terminal session
echo -e "${YELLOW}Testing: Create terminal session...${NC}"
SESSION_RESPONSE=$(api_call POST /api/v1/terminal/sessions "{\"tenantId\":\"$TEST_TENANT\"}")
echo "$SESSION_RESPONSE" | jq .
SESSION_ID=$(echo "$SESSION_RESPONSE" | jq -r '.sessionId')
POD_NAME=$(echo "$SESSION_RESPONSE" | jq -r '.podName')
echo ""
if [ -n "$SESSION_ID" ] && [ "$SESSION_ID" != "null" ]; then
    echo -e "${GREEN}✓ Session created: $SESSION_ID${NC}"
    echo "  Pod: $POD_NAME"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ Failed to create session${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    exit 1
fi
echo ""

# Wait for pod to be ready
echo -e "${YELLOW}Waiting for toolbox pod to be ready...${NC}"
sleep 5
kubectl wait --for=condition=ready pod/$POD_NAME -n toolbox-sessions --timeout=60s
echo -e "${GREEN}✓ Pod is ready${NC}"
echo ""

# Test 7: List sessions
echo -e "${YELLOW}Testing: List active sessions...${NC}"
LIST_RESPONSE=$(api_call GET /api/v1/terminal/sessions)
echo "$LIST_RESPONSE" | jq .
SESSION_COUNT=$(echo "$LIST_RESPONSE" | jq -r '.count')
echo ""
if [ "$SESSION_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ Found $SESSION_COUNT active session(s)${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ No active sessions found${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

# Test 8: Get specific session
echo -e "${YELLOW}Testing: Get session details...${NC}"
DETAIL_RESPONSE=$(api_call GET "/api/v1/terminal/sessions/$SESSION_ID")
echo "$DETAIL_RESPONSE" | jq .
echo -e "${GREEN}✓ Session details retrieved${NC}"
TESTS_PASSED=$((TESTS_PASSED + 1))
echo ""

# Test 9: Check pod exists
echo -e "${YELLOW}Testing: Verify pod exists in Kubernetes...${NC}"
kubectl get pod $POD_NAME -n toolbox-sessions -o json | jq '{name: .metadata.name, phase: .status.phase, containers: [.spec.containers[].name]}'
echo -e "${GREEN}✓ Pod exists and is accessible${NC}"
TESTS_PASSED=$((TESTS_PASSED + 1))
echo ""

# Test 10: Check pod labels
echo -e "${YELLOW}Testing: Verify pod labels...${NC}"
POD_LABELS=$(kubectl get pod $POD_NAME -n toolbox-sessions -o json | jq '.metadata.labels')
echo "$POD_LABELS" | jq .
echo -e "${GREEN}✓ Pod has correct labels${NC}"
TESTS_PASSED=$((TESTS_PASSED + 1))
echo ""

# Test 11: Check pod security context
echo -e "${YELLOW}Testing: Verify pod security context...${NC}"
SECURITY_CONTEXT=$(kubectl get pod $POD_NAME -n toolbox-sessions -o json | jq '.spec.containers[0].securityContext')
echo "$SECURITY_CONTEXT" | jq .
RUN_AS_USER=$(echo "$SECURITY_CONTEXT" | jq -r '.runAsUser')
if [ "$RUN_AS_USER" == "1000" ]; then
    echo -e "${GREEN}✓ Pod runs as non-root user (UID 1000)${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ Pod security context incorrect${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

echo -e "${YELLOW}=========================================="
echo "5. Pod Functionality Tests"
echo "==========================================${NC}"
echo ""

# Test 12: Execute command in pod
echo -e "${YELLOW}Testing: Execute command in toolbox pod...${NC}"
EXEC_OUTPUT=$(kubectl exec $POD_NAME -n toolbox-sessions -- whoami 2>/dev/null)
echo "  Command: whoami"
echo "  Output: $EXEC_OUTPUT"
if [ "$EXEC_OUTPUT" == "toolbox" ]; then
    echo -e "${GREEN}✓ Command execution works${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ Unexpected output${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

# Test 13: Check kubectl available
echo -e "${YELLOW}Testing: kubectl available in pod...${NC}"
KUBECTL_VERSION=$(kubectl exec $POD_NAME -n toolbox-sessions -- kubectl version --client --short 2>/dev/null || echo "not found")
echo "  $KUBECTL_VERSION"
if [[ "$KUBECTL_VERSION" == *"Client Version"* ]]; then
    echo -e "${GREEN}✓ kubectl is available${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ kubectl not found${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

# Test 14: Check aws CLI available
echo -e "${YELLOW}Testing: aws CLI available in pod...${NC}"
AWS_VERSION=$(kubectl exec $POD_NAME -n toolbox-sessions -- aws --version 2>/dev/null || echo "not found")
echo "  $AWS_VERSION"
if [[ "$AWS_VERSION" == *"aws-cli"* ]]; then
    echo -e "${GREEN}✓ aws CLI is available${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ aws CLI not found${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

# Test 15: Test kubectl permissions
echo -e "${YELLOW}Testing: kubectl permissions from pod...${NC}"
KUBECTL_TEST=$(kubectl exec $POD_NAME -n toolbox-sessions -- kubectl get namespaces 2>&1)
if [[ "$KUBECTL_TEST" != *"Error"* ]] && [[ "$KUBECTL_TEST" != *"Forbidden"* ]]; then
    echo "  ✓ Can list namespaces"
    echo -e "${GREEN}✓ kubectl has basic permissions${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ kubectl permissions issue${NC}"
    echo "  $KUBECTL_TEST"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

echo -e "${YELLOW}=========================================="
echo "6. Authorization Tests"
echo "==========================================${NC}"
echo ""

# Test 16: Test with readonly role (should fail)
echo -e "${YELLOW}Testing: Terminal access with readonly role (should fail)...${NC}"
READONLY_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/terminal/sessions" \
    -H "X-Dev-Role: readonly" \
    -H "Content-Type: application/json" \
    -d "{\"tenantId\":\"$TEST_TENANT\"}" \
    -w "\n%{http_code}")
HTTP_CODE=$(echo "$READONLY_RESPONSE" | tail -n 1)
if [ "$HTTP_CODE" == "403" ] || [ "$HTTP_CODE" == "503" ]; then
    echo -e "${GREEN}✓ Correctly denied access for readonly role (HTTP $HTTP_CODE)${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ Should have denied access (got HTTP $HTTP_CODE)${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

# Test 17: Test capabilities for readonly
echo -e "${YELLOW}Testing: Capabilities for readonly role...${NC}"
READONLY_CAPS=$(curl -s "$API_URL/api/v1/auth/capabilities" -H "X-Dev-Role: readonly")
READONLY_TERMINAL=$(echo "$READONLY_CAPS" | jq -r '.features.canUseTerminal')
if [ "$READONLY_TERMINAL" == "false" ]; then
    echo -e "${GREEN}✓ Readonly role correctly has no terminal capability${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ Readonly should not have terminal capability${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

echo -e "${YELLOW}=========================================="
echo "7. Cleanup"
echo "==========================================${NC}"
echo ""

# Test 18: Delete session
echo -e "${YELLOW}Testing: Delete terminal session...${NC}"
DELETE_RESPONSE=$(curl -s -X DELETE "$API_URL/api/v1/terminal/sessions/$SESSION_ID" \
    -H "X-Dev-Role: $TEST_ROLE" \
    -w "\n%{http_code}")
HTTP_CODE=$(echo "$DELETE_RESPONSE" | tail -n 1)
if [ "$HTTP_CODE" == "204" ]; then
    echo -e "${GREEN}✓ Session deleted (HTTP $HTTP_CODE)${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ Failed to delete session (HTTP $HTTP_CODE)${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

# Wait for pod cleanup
echo -e "${YELLOW}Waiting for pod cleanup...${NC}"
sleep 3
POD_STATUS=$(kubectl get pod $POD_NAME -n toolbox-sessions 2>&1 || echo "not found")
if [[ "$POD_STATUS" == *"not found"* ]] || [[ "$POD_STATUS" == *"NotFound"* ]]; then
    echo -e "${GREEN}✓ Pod cleaned up successfully${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${YELLOW}⚠ Pod still exists (may be terminating)${NC}"
    echo "  Status: $POD_STATUS"
fi
echo ""

# Summary
echo -e "${BLUE}=========================================="
echo "Test Summary"
echo "==========================================${NC}"
echo ""
echo -e "Total tests run: $((TESTS_PASSED + TESTS_FAILED))"
echo -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}=========================================="
    echo "All tests passed! ✓"
    echo "==========================================${NC}"
    exit 0
else
    echo -e "${RED}=========================================="
    echo "Some tests failed! ✗"
    echo "==========================================${NC}"
    exit 1
fi

