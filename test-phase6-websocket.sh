#!/bin/bash

# Phase 6: Web Terminal - WebSocket Testing Script
# This script tests actual terminal WebSocket connections

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

API_BASE="http://localhost:9080/api/v1"
WS_BASE="ws://localhost:9080/api/v1"

# Auth headers for OAuth2Proxy
AUTH_USER_HEADER="X-Auth-Request-User: admin"
AUTH_GROUPS_HEADER="X-Auth-Request-Groups: platform-admin"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║        Phase 6: Web Terminal - WebSocket Testing              ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Function to print section headers
print_section() {
    echo ""
    echo -e "${YELLOW}▶ $1${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Step 1: Check prerequisites
print_section "Step 1: Checking Prerequisites"

# Check if manager is running
if ! curl -s http://localhost:9080/healthz > /dev/null 2>&1; then
    echo -e "${RED}✗ Manager not running on port 9080${NC}"
    echo -e "${YELLOW}→ Please start the manager first:${NC}"
    echo -e "   ${GREEN}./start-phase5.sh${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Manager is running${NC}"
fi

# Check if Kind cluster is running
if ! kubectl cluster-info > /dev/null 2>&1; then
    echo -e "${RED}✗ Kind cluster not accessible${NC}"
    echo -e "${YELLOW}→ Please start the Kind cluster first:${NC}"
    echo -e "   ${GREEN}kind create cluster --name platform-manager --config hack/kind-config.yaml${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Kind cluster is accessible${NC}"
fi

# Check if wscat is installed
if ! command_exists wscat; then
    echo -e "${YELLOW}✗ wscat not found. Installing...${NC}"
    npm install -g wscat
    if ! command_exists wscat; then
        echo -e "${RED}✗ Failed to install wscat${NC}"
        echo -e "${YELLOW}→ Please install manually:${NC}"
        echo -e "   ${GREEN}npm install -g wscat${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ wscat installed${NC}"
else
    echo -e "${GREEN}✓ wscat is available${NC}"
fi

# Check if toolbox-sessions namespace exists
if ! kubectl get namespace toolbox-sessions > /dev/null 2>&1; then
    echo -e "${YELLOW}! toolbox-sessions namespace not found, creating...${NC}"
    kubectl create namespace toolbox-sessions
    echo -e "${GREEN}✓ Created toolbox-sessions namespace${NC}"
else
    echo -e "${GREEN}✓ toolbox-sessions namespace exists${NC}"
fi

# Step 2: Check terminal configuration
print_section "Step 2: Checking Terminal Configuration"

TERMINAL_CONFIG=$(curl -s -H "${AUTH_USER_HEADER}" \
                        -H "${AUTH_GROUPS_HEADER}" \
                        "${API_BASE}/terminal/config")

echo "Terminal Configuration:"
echo "$TERMINAL_CONFIG" | jq .

ENABLED=$(echo "$TERMINAL_CONFIG" | jq -r '.enabled')
HAS_ACCESS=$(echo "$TERMINAL_CONFIG" | jq -r '.hasAccess')

if [ "$ENABLED" != "true" ]; then
    echo -e "${RED}✗ Terminal feature is not enabled${NC}"
    echo -e "${YELLOW}→ Please restart the manager with terminal flags:${NC}"
    echo -e "   ${GREEN}--enable-terminal=true --terminal-namespace=toolbox-sessions${NC}"
    exit 1
fi

if [ "$HAS_ACCESS" != "true" ]; then
    echo -e "${RED}✗ User does not have terminal access${NC}"
    echo -e "${YELLOW}→ Check user roles and capabilities${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Terminal is enabled and user has access${NC}"

# Step 3: Create a terminal session
print_section "Step 3: Creating Terminal Session"

SESSION_RESPONSE=$(curl -s -X POST \
    -H "${AUTH_USER_HEADER}" \
    -H "${AUTH_GROUPS_HEADER}" \
    -H "Content-Type: application/json" \
    -d '{"tenantId":"tenant-alpha"}' \
    "${API_BASE}/terminal/sessions")

echo "Session Response:"
echo "$SESSION_RESPONSE" | jq .

SESSION_ID=$(echo "$SESSION_RESPONSE" | jq -r '.id')
POD_NAME=$(echo "$SESSION_RESPONSE" | jq -r '.podName')

if [ -z "$SESSION_ID" ] || [ "$SESSION_ID" == "null" ]; then
    echo -e "${RED}✗ Failed to create session${NC}"
    echo "Response: $SESSION_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✓ Session created: ${SESSION_ID}${NC}"
echo -e "  Pod: ${POD_NAME}"

# Step 4: Wait for pod to be ready
print_section "Step 4: Waiting for Toolbox Pod to be Ready"

echo "Waiting for pod ${POD_NAME} in toolbox-sessions namespace..."
kubectl wait --for=condition=ready pod/${POD_NAME} -n toolbox-sessions --timeout=60s || {
    echo -e "${RED}✗ Pod did not become ready in time${NC}"
    echo "Pod status:"
    kubectl get pod ${POD_NAME} -n toolbox-sessions
    echo "Pod events:"
    kubectl describe pod ${POD_NAME} -n toolbox-sessions | tail -20
    exit 1
}

echo -e "${GREEN}✓ Pod is ready${NC}"

# Step 5: List active sessions
print_section "Step 5: Listing Active Sessions"

SESSIONS=$(curl -s -H "${AUTH_USER_HEADER}" \
                 -H "${AUTH_GROUPS_HEADER}" \
                 "${API_BASE}/terminal/sessions")

echo "Active Sessions:"
echo "$SESSIONS" | jq .
echo -e "${GREEN}✓ Successfully listed sessions${NC}"

# Step 6: Manual WebSocket testing instructions
print_section "Step 6: WebSocket Connection Testing"

WS_URL="${WS_BASE}/terminal/sessions/${SESSION_ID}/ws"

echo -e "${YELLOW}WebSocket URL:${NC}"
echo -e "  ${WS_URL}"
echo ""
echo -e "${YELLOW}To test the WebSocket connection manually, run:${NC}"
echo -e "  ${GREEN}wscat -c \"${WS_URL}\" -H \"${AUTH_USER_HEADER}\" -H \"${AUTH_GROUPS_HEADER}\"${NC}"
echo ""
echo -e "${YELLOW}Once connected, try these commands:${NC}"
echo -e "  ${GREEN}ls -la${NC}"
echo -e "  ${GREEN}pwd${NC}"
echo -e "  ${GREEN}echo \$USER${NC}"
echo -e "  ${GREEN}kubectl get namespaces${NC}"
echo -e "  ${GREEN}helm version${NC}"
echo -e "  ${GREEN}argocd version --client${NC}"
echo ""
echo -e "${YELLOW}To disconnect: Press Ctrl+C${NC}"
echo ""

# Step 7: Keep session alive for manual testing
print_section "Step 7: Session Ready for Testing"

echo -e "${GREEN}Session ${SESSION_ID} is ready!${NC}"
echo ""
echo -e "${BLUE}Instructions:${NC}"
echo -e "  1. Open another terminal"
echo -e "  2. Run the wscat command shown above"
echo -e "  3. Execute commands in the terminal"
echo -e "  4. Verify command execution and output"
echo -e "  5. Press Ctrl+C in wscat to disconnect"
echo ""
echo -e "${YELLOW}This script will keep the session alive for 5 minutes...${NC}"
echo -e "${YELLOW}Press Ctrl+C to end the test early and clean up${NC}"
echo ""

# Trap to cleanup on exit
cleanup() {
    echo ""
    print_section "Cleanup: Deleting Terminal Session"
    curl -s -X DELETE \
        -H "${AUTH_USER_HEADER}" \
        -H "${AUTH_GROUPS_HEADER}" \
        "${API_BASE}/terminal/sessions/${SESSION_ID}" | jq .
    echo -e "${GREEN}✓ Session deleted${NC}"
    exit 0
}

trap cleanup INT TERM

# Wait for 5 minutes or until interrupted
for i in {1..60}; do
    echo -ne "  Time remaining: $(( (60 - i) * 5 ))s / 300s\r"
    sleep 5
done

echo ""
cleanup

