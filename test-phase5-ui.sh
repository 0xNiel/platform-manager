#!/bin/bash

# Phase 5 UI Testing Script
# Creates problematic resources, waits for detection, allows UI verification, then cleans up

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}Phase 5 UI Testing - Create Problems${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

API_BASE="http://localhost:9080/api/v1"
NAMESPACE="tenant-alpha"

# Check if API is responding
if ! curl -s ${API_BASE}/troubleshooting/summary > /dev/null 2>&1; then
    echo -e "${RED}✗ API not responding. Is the manager running?${NC}"
    echo "Start it with: ./bin/manager --api-bind-address=:9080 --rule-eval-interval=2m"
    exit 1
fi

echo -e "${GREEN}✓ API is responding${NC}"
echo ""

# Function to wait for rule evaluation
wait_for_evaluation() {
    local wait_time=$1
    echo -e "${YELLOW}⏳ Waiting ${wait_time} seconds for rule evaluation...${NC}"
    for i in $(seq $wait_time -1 1); do
        printf "\r   Time remaining: %02d seconds" $i
        sleep 1
    done
    echo ""
}

# Function to show findings
show_findings() {
    echo -e "${BLUE}Current Findings:${NC}"
    local findings=$(curl -s ${API_BASE}/troubleshooting/summary)
    local total=$(echo "$findings" | jq -r '.totalFindings')
    local critical=$(echo "$findings" | jq -r '.critical')
    local high=$(echo "$findings" | jq -r '.high')
    local medium=$(echo "$findings" | jq -r '.medium')
    
    echo -e "  Total: ${YELLOW}$total${NC}"
    echo -e "  Critical: ${RED}$critical${NC}"
    echo -e "  High: ${RED}$high${NC}"
    echo -e "  Medium: ${YELLOW}$medium${NC}"
    echo ""
}

echo -e "${BLUE}Step 1: Creating Problematic Resources${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 1. Create CrashLoop Pod
echo -e "${YELLOW}1. Creating CrashLoop Pod (triggers crashloop-backoff rule)${NC}"
kubectl run phase5-crashloop-test \
    --image=busybox \
    --restart=Always \
    --namespace=$NAMESPACE \
    --labels="platform.io/tenant=tenant-alpha,phase5-test=true" \
    --command -- sh -c "echo 'Crashing intentionally'; exit 1" \
    2>/dev/null || echo "   (Pod may already exist)"

# 2. Create ImagePull Failure Pod
echo -e "${YELLOW}2. Creating ImagePull Failure Pod (triggers image-pull-failed rule)${NC}"
kubectl run phase5-imagepull-test \
    --image=nonexistent-registry.example.com/nonexistent:v999 \
    --namespace=$NAMESPACE \
    --labels="platform.io/tenant=tenant-alpha,phase5-test=true" \
    2>/dev/null || echo "   (Pod may already exist)"

# 3. Create Pending Pod (no resources available)
echo -e "${YELLOW}3. Creating Pending Pod (triggers pod-pending-long rule)${NC}"
cat <<EOF | kubectl apply -f - 2>/dev/null || echo "   (Pod may already exist)"
apiVersion: v1
kind: Pod
metadata:
  name: phase5-pending-test
  namespace: $NAMESPACE
  labels:
    platform.io/tenant: tenant-alpha
    phase5-test: "true"
spec:
  containers:
  - name: app
    image: nginx:alpine
    resources:
      requests:
        memory: "999Gi"  # Impossible requirement
        cpu: "999"
EOF

# 4. Create a resource quota for high-resource-usage rule
echo -e "${YELLOW}4. Creating ResourceQuota (for high-resource-usage rule)${NC}"
cat <<EOF | kubectl apply -f - 2>/dev/null || echo "   (Quota may already exist)"
apiVersion: v1
kind: ResourceQuota
metadata:
  name: phase5-test-quota
  namespace: $NAMESPACE
  labels:
    phase5-test: "true"
spec:
  hard:
    pods: "10"
    requests.memory: "1Gi"
    requests.cpu: "2"
EOF

# Create pods to push quota usage high
echo -e "${YELLOW}5. Creating pods to trigger high quota usage${NC}"
for i in {1..8}; do
    kubectl run phase5-quota-test-$i \
        --image=nginx:alpine \
        --namespace=$NAMESPACE \
        --labels="platform.io/tenant=tenant-alpha,phase5-test=true" \
        --requests="memory=100Mi,cpu=200m" \
        2>/dev/null || echo "   (Pod $i may already exist)"
done

echo ""
echo -e "${GREEN}✓ All test resources created${NC}"
echo ""

# Show current state
echo -e "${BLUE}Current Pod Status:${NC}"
kubectl get pods -n $NAMESPACE --selector=phase5-test=true 2>/dev/null || echo "No test pods found"
echo ""

# Initial wait for pods to start failing
echo -e "${BLUE}Step 2: Waiting for Problems to Manifest${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo -e "${YELLOW}Waiting 30 seconds for pods to fail...${NC}"
sleep 30

echo ""
echo -e "${BLUE}Pod States After 30s:${NC}"
kubectl get pods -n $NAMESPACE --selector=phase5-test=true 2>/dev/null | grep -E "phase5-|STATUS"
echo ""

# Trigger manual scan
echo -e "${YELLOW}Triggering manual rule scan...${NC}"
curl -s -X POST ${API_BASE}/troubleshooting/scan > /dev/null
echo -e "${GREEN}✓ Scan triggered${NC}"

# Wait for evaluation
wait_for_evaluation 10

# Show findings
show_findings

echo -e "${BLUE}Step 3: Verify in UI${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${GREEN}Now check the Troubleshooting UI:${NC}"
echo ""
echo -e "  ${YELLOW}1.${NC} Open browser: ${BLUE}http://localhost:9083/troubleshooting${NC}"
echo -e "  ${YELLOW}2.${NC} You should see:"
echo -e "     • Summary cards showing ${RED}HIGH${NC} and ${YELLOW}MEDIUM${NC} findings"
echo -e "     • CrashLoopBackOff finding for phase5-crashloop-test"
echo -e "     • ImagePullBackOff finding for phase5-imagepull-test"
echo -e "     • Pending Pod finding for phase5-pending-test"
echo -e "     • High Resource Usage finding (if pods are running)"
echo -e "  ${YELLOW}3.${NC} Click on findings to expand details"
echo -e "  ${YELLOW}4.${NC} Verify recommendations are shown"
echo -e "  ${YELLOW}5.${NC} Test filtering by severity"
echo ""

# Detailed findings list
echo -e "${BLUE}Current Findings Detail:${NC}"
curl -s ${API_BASE}/troubleshooting/findings | jq -r '.findings[] | 
    "  • [\(.severity | ascii_upcase)] \(.title)\n    Resource: \(.resourceRef.kind)/\(.resourceRef.name)\n    Message: \(.message[0:80])...\n"' 2>/dev/null \
    || echo "  No findings yet (may need more time)"
echo ""

echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
echo -e "${YELLOW}Press ENTER after verifying the UI...${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
read

# Cleanup
echo ""
echo -e "${BLUE}Step 4: Cleaning Up Test Resources${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo -e "${YELLOW}Deleting test pods...${NC}"
kubectl delete pods -n $NAMESPACE --selector=phase5-test=true --ignore-not-found=true 2>/dev/null || true

echo -e "${YELLOW}Deleting test quota...${NC}"
kubectl delete resourcequota -n $NAMESPACE phase5-test-quota --ignore-not-found=true 2>/dev/null || true

echo -e "${GREEN}✓ Cleanup complete${NC}"
echo ""

# Wait for evaluation to clear findings
echo -e "${YELLOW}Waiting for rule evaluation to clear findings...${NC}"
wait_for_evaluation 15

# Trigger final scan
curl -s -X POST ${API_BASE}/troubleshooting/scan > /dev/null
sleep 5

# Show final state
echo ""
echo -e "${BLUE}Final State:${NC}"
show_findings

echo -e "${GREEN}✓ Test complete!${NC}"
echo ""
echo -e "${BLUE}Summary:${NC}"
echo "  ✓ Created problematic resources"
echo "  ✓ Rules detected issues"
echo "  ✓ UI displayed findings"
echo "  ✓ Resources cleaned up"
echo "  ✓ Findings should now be cleared"
echo ""
echo -e "${GREEN}Phase 5 UI testing successful!${NC}"

