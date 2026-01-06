#!/bin/bash
# Test WebSocket connection directly
set -e

API_URL="http://localhost:9080/api/v1"
USER_HEADER="X-Auth-Request-User: admin"
EMAIL_HEADER="X-Auth-Request-Email: admin@platform.io"
GROUPS_HEADER="X-Auth-Request-Groups: platform-admin"

echo "Creating session..."
SESSION_RESPONSE=$(curl -s "$API_URL/terminal/sessions" \
  -X POST \
  -H "$USER_HEADER" \
  -H "$EMAIL_HEADER" \
  -H "$GROUPS_HEADER" \
  -H "Content-Type: application/json" \
  -d '{"tenantId":"tenant-alpha"}')

SESSION_ID=$(echo "$SESSION_RESPONSE" | jq -r '.sessionId')
POD_NAME=$(echo "$SESSION_RESPONSE" | jq -r '.podName')
WS_URL=$(echo "$SESSION_RESPONSE" | jq -r '.wsUrl')

echo "Session ID: $SESSION_ID"
echo "Pod Name: $POD_NAME"
echo "WebSocket URL: $WS_URL"

# Wait for pod
echo "Waiting for pod to be ready..."
for i in {1..30}; do
  POD_STATUS=$(kubectl get pod "$POD_NAME" -n toolbox-sessions -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
  if [ "$POD_STATUS" == "Running" ]; then
    echo "Pod is ready!"
    sleep 2
    break
  fi
  sleep 1
done

# Watch logs in another terminal
echo ""
echo "=========================================="
echo "Opening log watcher in the background..."
echo "=========================================="
tail -f /tmp/manager-debug.log | grep -i "$SESSION_ID" &
LOG_PID=$!

echo ""
echo "=========================================="
echo "Connecting to WebSocket..."
echo "=========================================="
echo "Type commands and press Enter."
echo "You should see the bash prompt and output."
echo "Type 'exit' to quit."
echo ""

# Connect with wscat
WS_FULL_URL="ws://localhost:9080${WS_URL}"
echo "URL: $WS_FULL_URL"
echo ""

wscat -c "$WS_FULL_URL"

# Cleanup
kill $LOG_PID 2>/dev/null || true

echo ""
echo "Deleting session..."
curl -s "$API_URL/terminal/sessions/$SESSION_ID" \
  -X DELETE \
  -H "$USER_HEADER" \
  -H "$EMAIL_HEADER" \
  -H "$GROUPS_HEADER"
echo "Done!"

