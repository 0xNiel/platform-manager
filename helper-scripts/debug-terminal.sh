#!/bin/bash
set -e

echo "=================================="
echo "Phase 6 Terminal Debug Script"
echo "=================================="
echo ""

API_URL="http://localhost:9080/api/v1"
USER_HEADER="X-Auth-Request-User: admin"
EMAIL_HEADER="X-Auth-Request-Email: admin@platform.io"
GROUPS_HEADER="X-Auth-Request-Groups: platform-admin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Step 1: Check terminal config"
echo "------------------------------"
CONFIG=$(curl -s "$API_URL/terminal/config" \
  -H "$USER_HEADER" \
  -H "$EMAIL_HEADER" \
  -H "$GROUPS_HEADER")
echo "Config: $CONFIG"
echo ""

ENABLED=$(echo "$CONFIG" | jq -r '.enabled')
HAS_ACCESS=$(echo "$CONFIG" | jq -r '.hasAccess')

if [ "$ENABLED" != "true" ]; then
  echo -e "${RED}❌ Terminal is not enabled${NC}"
  exit 1
fi

if [ "$HAS_ACCESS" != "true" ]; then
  echo -e "${RED}❌ User does not have access${NC}"
  exit 1
fi

echo -e "${GREEN}✅ Terminal is enabled and user has access${NC}"
echo ""

echo "Step 2: Create terminal session"
echo "--------------------------------"
SESSION_RESPONSE=$(curl -s "$API_URL/terminal/sessions" \
  -X POST \
  -H "$USER_HEADER" \
  -H "$EMAIL_HEADER" \
  -H "$GROUPS_HEADER" \
  -H "Content-Type: application/json" \
  -d '{"tenantId":"tenant-alpha"}')

echo "Session response: $SESSION_RESPONSE"
echo ""

SESSION_ID=$(echo "$SESSION_RESPONSE" | jq -r '.sessionId')
POD_NAME=$(echo "$SESSION_RESPONSE" | jq -r '.podName')
NAMESPACE=$(echo "$SESSION_RESPONSE" | jq -r '.namespace')
WS_URL=$(echo "$SESSION_RESPONSE" | jq -r '.wsUrl')

if [ "$SESSION_ID" == "null" ] || [ -z "$SESSION_ID" ]; then
  echo -e "${RED}❌ Failed to create session${NC}"
  exit 1
fi

echo -e "${GREEN}✅ Session created${NC}"
echo "   Session ID: $SESSION_ID"
echo "   Pod Name: $POD_NAME"
echo "   Namespace: $NAMESPACE"
echo "   WebSocket URL: $WS_URL"
echo ""

echo "Step 3: Wait for pod to be ready"
echo "---------------------------------"
for i in {1..30}; do
  POD_STATUS=$(kubectl get pod "$POD_NAME" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
  echo "Attempt $i/30: Pod status = $POD_STATUS"
  
  if [ "$POD_STATUS" == "Running" ]; then
    echo -e "${GREEN}✅ Pod is running${NC}"
    sleep 2  # Give it a moment to fully initialize
    break
  fi
  
  if [ $i -eq 30 ]; then
    echo -e "${RED}❌ Pod failed to start within 30 seconds${NC}"
    kubectl get pod "$POD_NAME" -n "$NAMESPACE"
    kubectl describe pod "$POD_NAME" -n "$NAMESPACE"
    exit 1
  fi
  
  sleep 1
done
echo ""

echo "Step 4: Check pod details"
echo "-------------------------"
kubectl get pod "$POD_NAME" -n "$NAMESPACE" -o yaml | grep -A 20 "containers:"
echo ""

echo "Step 5: Test direct exec to pod"
echo "--------------------------------"
echo "Running: kubectl exec -it $POD_NAME -n $NAMESPACE -- echo 'test'"
kubectl exec "$POD_NAME" -n "$NAMESPACE" -- echo "Direct exec test successful"
echo -e "${GREEN}✅ Direct kubectl exec works${NC}"
echo ""

echo "Step 6: Test interactive bash"
echo "------------------------------"
echo "Running: kubectl exec -it $POD_NAME -n $NAMESPACE -- /bin/bash -i -c 'echo \$PS1'"
PS1_OUTPUT=$(kubectl exec "$POD_NAME" -n "$NAMESPACE" -- /bin/bash -i -c 'echo $PS1' 2>&1)
echo "PS1 (prompt): $PS1_OUTPUT"
echo ""

echo "Step 7: Simulate what the backend does"
echo "---------------------------------------"
echo "Testing: kubectl exec with stdin, stdout, stderr, tty"
# Create a test script
cat > /tmp/test-terminal.sh << 'EOF'
#!/bin/bash
# Send a newline to trigger prompt
echo ""
# Wait a moment
sleep 0.5
# Send a command
echo "ls -la"
sleep 0.5
echo "exit"
EOF
chmod +x /tmp/test-terminal.sh

echo "Executing interactive bash in pod..."
timeout 5 kubectl exec -it "$POD_NAME" -n "$NAMESPACE" -- /bin/bash -i < /tmp/test-terminal.sh 2>&1 || true
echo ""

echo "Step 8: Check pod logs"
echo "----------------------"
POD_LOGS=$(kubectl logs "$POD_NAME" -n "$NAMESPACE" --tail=50 2>&1 || echo "No logs")
echo "Pod logs:"
echo "$POD_LOGS"
echo ""

echo "Step 9: Check backend logs for this session"
echo "--------------------------------------------"
echo "Backend logs (last 50 lines, filtered for session):"
tail -50 /tmp/manager-tty-fix.log | grep -i "$SESSION_ID" || echo "No session-specific logs found"
echo ""

echo "Step 10: Test WebSocket connection (if wscat is available)"
echo "-----------------------------------------------------------"
if command -v wscat &> /dev/null; then
  WS_FULL_URL="ws://localhost:9080${WS_URL}"
  echo "WebSocket URL: $WS_FULL_URL"
  echo "Sending test input to WebSocket..."
  
  # Create test input file
  echo -e "\n" > /tmp/ws-test-input.txt
  echo "ls -la" >> /tmp/ws-test-input.txt
  echo "exit" >> /tmp/ws-test-input.txt
  
  timeout 5 wscat -c "$WS_FULL_URL" < /tmp/ws-test-input.txt 2>&1 || echo "WebSocket test completed"
else
  echo -e "${YELLOW}⚠️  wscat not installed, skipping WebSocket test${NC}"
  echo "Install with: npm install -g wscat"
fi
echo ""

echo "Step 11: Check manager WebSocket handler code"
echo "----------------------------------------------"
echo "Checking if TerminalSizeQueue is set..."
grep -A 5 "TerminalSizeQueue" /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager/internal/api/handlers/terminal.go || echo "Not found in code"
echo ""

echo "Step 12: Cleanup - Delete session"
echo "----------------------------------"
curl -s "$API_URL/terminal/sessions/$SESSION_ID" \
  -X DELETE \
  -H "$USER_HEADER" \
  -H "$EMAIL_HEADER" \
  -H "$GROUPS_HEADER"
echo -e "${GREEN}✅ Session deleted${NC}"
echo ""

echo "=================================="
echo "Debug Report Complete"
echo "=================================="
echo ""
echo "Summary:"
echo "--------"
echo "1. Terminal config: $ENABLED / Access: $HAS_ACCESS"
echo "2. Session created: $SESSION_ID"
echo "3. Pod created: $POD_NAME"
echo "4. Pod status: Running"
echo "5. Direct exec: Works"
echo "6. Interactive bash: Check output above"
echo ""
echo "Next steps:"
echo "-----------"
echo "1. Review the 'Test interactive bash' output (Step 7)"
echo "2. Check if PS1 (prompt) is set (Step 6)"
echo "3. Review pod logs for any errors (Step 8)"
echo "4. Check backend logs for WebSocket activity (Step 9)"
echo ""

