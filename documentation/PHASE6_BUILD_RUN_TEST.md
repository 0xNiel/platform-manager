# Phase 6: Web Terminal - Build, Run & Test Guide

This guide will walk you through building, running, and testing the Phase 6 terminal implementation.

---

## Quick Start (Automated)

### 1. Setup Everything
```bash
# Run the automated setup script
./setup-phase6-terminal.sh
```

This will:
- Build the toolbox Docker image
- Load it into Kind cluster
- Create `toolbox-sessions` namespace
- Create ServiceAccount with RBAC
- Grant manager permissions

### 2. Build & Run Manager
```bash
# Build the manager
make build

# Run with terminal enabled
./bin/manager --enable-terminal=true
```

### 3. Run Tests
```bash
# In a new terminal, run the test script
./test-phase6-terminal.sh
```

---

## Manual Step-by-Step Guide

### Step 1: Build the Toolbox Image

```bash
# Build the toolbox image
docker build -t platform-manager-toolbox:dev -f Dockerfile.toolbox .

# Load into Kind (if using Kind)
kind load docker-image platform-manager-toolbox:dev --name platform-manager

# Verify image is available
docker images | grep platform-manager-toolbox
```

**Expected output:**
```
platform-manager-toolbox   dev    abc123   2 minutes ago   150MB
```

---

### Step 2: Create Required Kubernetes Resources

#### Create Namespace
```bash
kubectl create namespace toolbox-sessions
```

#### Create ServiceAccount with RBAC
```bash
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: toolbox-session
  namespace: toolbox-sessions
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: toolbox-session
rules:
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps", "namespaces"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets", "daemonsets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["*.upbound.io", "*.crossplane.io"]
    resources: ["*"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["argoproj.io"]
    resources: ["applications", "appprojects"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["platform.io"]
    resources: ["*"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: toolbox-session
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: toolbox-session
subjects:
  - kind: ServiceAccount
    name: toolbox-session
    namespace: toolbox-sessions
EOF
```

#### Grant Manager Permissions
```bash
kubectl apply -f - <<EOF
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: platform-manager-toolbox
  namespace: toolbox-sessions
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["create", "get", "list", "watch", "delete"]
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create", "get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: platform-manager-toolbox
  namespace: toolbox-sessions
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: platform-manager-toolbox
subjects:
  - kind: ServiceAccount
    name: platform-manager-controller-manager
    namespace: platform-manager-system
EOF
```

**Verify resources:**
```bash
kubectl get namespace toolbox-sessions
kubectl get serviceaccount toolbox-session -n toolbox-sessions
kubectl get clusterrole toolbox-session
kubectl get clusterrolebinding toolbox-session
```

---

### Step 3: Build the Manager

```bash
# Build the manager binary
make build

# Or build with dependencies
make generate fmt vet build
```

**Expected output:**
```
go build -o bin/manager cmd/main.go
✓ Manager binary created at bin/manager
```

---

### Step 4: Run the Manager with Terminal Enabled

```bash
# Run with terminal enabled
./bin/manager \
  --enable-terminal=true \
  --terminal-namespace=toolbox-sessions \
  --terminal-image=platform-manager-toolbox:dev \
  --terminal-idle-timeout=10m

# Or use environment variables
export ENABLE_TERMINAL=true
./bin/manager
```

**Expected output:**
```
2026-01-05T10:30:00Z INFO setup Terminal feature enabled
2026-01-05T10:30:00Z INFO setup starting manager
2026-01-05T10:30:00Z INFO api-server Starting API server {"addr": ":9080"}
2026-01-05T10:30:00Z INFO api-server Registering terminal routes {"enabled": true}
```

---

### Step 5: Test the Terminal Endpoints

Open a **new terminal** and run these tests:

#### Test 1: Check Health
```bash
curl http://localhost:9080/healthz
# Expected: ok

curl http://localhost:9080/readyz
# Expected: ok
```

#### Test 2: Check User Capabilities
```bash
curl -H "X-Dev-Role: admin" http://localhost:9080/api/v1/auth/capabilities | jq .
```

**Expected output:**
```json
{
  "role": "admin",
  "capabilities": {
    "argo:refresh": true,
    "argo:sync": true,
    "crossplane:pause": true,
    "crossplane:reconcile": true,
    "resource:delete": true,
    "terminal:use": true
  },
  "features": {
    "canSyncArgo": true,
    "canRefreshArgo": true,
    "canPauseCrossplane": true,
    "canReconcileCrossplane": true,
    "canDeleteResource": true,
    "canUseTerminal": true
  }
}
```

#### Test 3: Check Terminal Configuration
```bash
curl -H "X-Dev-Role: admin" http://localhost:9080/api/v1/terminal/config | jq .
```

**Expected output:**
```json
{
  "enabled": true,
  "idleTimeout": 10,
  "maxSessions": 20,
  "hasAccess": true
}
```

#### Test 4: Create Terminal Session
```bash
curl -X POST \
  -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{"tenantId":"alpha"}' \
  http://localhost:9080/api/v1/terminal/sessions | jq .
```

**Expected output:**
```json
{
  "sessionId": "550e8400-e29b-41d4-a716-446655440000",
  "tenantId": "alpha",
  "podName": "toolbox-alpha-550e8400",
  "namespace": "toolbox-sessions",
  "createdAt": "2026-01-05T10:35:00Z",
  "wsUrl": "/api/v1/terminal/sessions/550e8400-e29b-41d4-a716-446655440000/ws"
}
```

Save the `sessionId` for next tests:
```bash
SESSION_ID="550e8400-e29b-41d4-a716-446655440000"  # Use your actual ID
```

#### Test 5: Verify Pod Was Created
```bash
kubectl get pods -n toolbox-sessions
```

**Expected output:**
```
NAME                     READY   STATUS    RESTARTS   AGE
toolbox-alpha-550e8400   1/1     Running   0          10s
```

#### Test 6: Check Pod Details
```bash
POD_NAME=$(kubectl get pods -n toolbox-sessions -o jsonpath='{.items[0].metadata.name}')

# Check labels
kubectl get pod $POD_NAME -n toolbox-sessions -o jsonpath='{.metadata.labels}' | jq .

# Check security context
kubectl get pod $POD_NAME -n toolbox-sessions -o jsonpath='{.spec.containers[0].securityContext}' | jq .
```

#### Test 7: Execute Command in Pod
```bash
kubectl exec $POD_NAME -n toolbox-sessions -- whoami
# Expected: toolbox

kubectl exec $POD_NAME -n toolbox-sessions -- kubectl version --client --short
# Expected: Client Version: v1.29.0

kubectl exec $POD_NAME -n toolbox-sessions -- aws --version
# Expected: aws-cli/2.x.x Python/3.x.x
```

#### Test 8: Test kubectl Permissions
```bash
kubectl exec $POD_NAME -n toolbox-sessions -- kubectl get namespaces
# Should list namespaces

kubectl exec $POD_NAME -n toolbox-sessions -- kubectl get pods -n platform-manager-system
# Should list pods
```

#### Test 9: List Active Sessions
```bash
curl -H "X-Dev-Role: admin" http://localhost:9080/api/v1/terminal/sessions | jq .
```

**Expected output:**
```json
{
  "sessions": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "dev-user",
      "tenantId": "alpha",
      "podName": "toolbox-alpha-550e8400",
      "namespace": "toolbox-sessions",
      "createdAt": "2026-01-05T10:35:00Z",
      "lastActivity": "2026-01-05T10:35:30Z",
      "isActive": true,
      "idleMinutes": 0
    }
  ],
  "count": 1
}
```

#### Test 10: Get Specific Session
```bash
curl -H "X-Dev-Role: admin" \
  http://localhost:9080/api/v1/terminal/sessions/$SESSION_ID | jq .
```

#### Test 11: Test Authorization (Should Fail)
```bash
# Try with readonly role - should get 403 or 503
curl -X POST \
  -H "X-Dev-Role: readonly" \
  -H "Content-Type: application/json" \
  -d '{"tenantId":"alpha"}' \
  http://localhost:9080/api/v1/terminal/sessions -w "\nHTTP Status: %{http_code}\n"

# Expected: HTTP Status: 403 or 503
```

#### Test 12: Delete Session
```bash
curl -X DELETE \
  -H "X-Dev-Role: admin" \
  http://localhost:9080/api/v1/terminal/sessions/$SESSION_ID -w "\nHTTP Status: %{http_code}\n"

# Expected: HTTP Status: 204
```

#### Test 13: Verify Pod Cleanup
```bash
# Wait a few seconds
sleep 5

# Pod should be gone
kubectl get pods -n toolbox-sessions
# Expected: No resources found
```

---

### Step 6: Test WebSocket Connection (Optional)

Install `wscat` for WebSocket testing:
```bash
npm install -g wscat
```

Create a session and connect:
```bash
# Create session
SESSION_RESPONSE=$(curl -X POST \
  -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{"tenantId":"alpha"}' \
  http://localhost:9080/api/v1/terminal/sessions)

SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.sessionId')

# Wait for pod to be ready
sleep 10

# Connect via WebSocket
wscat -c ws://localhost:9080/api/v1/terminal/sessions/$SESSION_ID/ws

# Once connected, you can type commands:
# > ls -la
# > whoami
# > kubectl get pods
# > exit
```

---

## Automated Test Script

Run the comprehensive test script:

```bash
./test-phase6-terminal.sh
```

This script will:
- Test all API endpoints
- Verify pod creation and configuration
- Test authorization and capabilities
- Test pod functionality (kubectl, aws CLI)
- Test session cleanup
- Display a test summary

**Expected output:**
```
==========================================
Phase 6: Web Terminal - Testing
==========================================

Testing against: http://localhost:9080
Test role: admin
Test tenant: alpha

...

==========================================
Test Summary
==========================================

Total tests run: 19
Tests passed: 19 ✓
Tests failed: 0

==========================================
All tests passed! ✓
==========================================
```

---

## Troubleshooting

### Issue: Terminal disabled error

**Error:**
```json
{"error": "Terminal feature is disabled"}
```

**Solution:**
```bash
# Restart manager with terminal enabled
./bin/manager --enable-terminal=true
```

---

### Issue: Pod fails to create

**Error:**
```
failed to create toolbox pod: pods is forbidden
```

**Solution:**
```bash
# Check RBAC permissions
kubectl auth can-i create pods \
  --as=system:serviceaccount:platform-manager-system:platform-manager-controller-manager \
  -n toolbox-sessions

# If false, reapply the Role/RoleBinding from Step 2
```

---

### Issue: Pod stuck in ImagePullBackOff

**Error:**
```bash
kubectl get pods -n toolbox-sessions
# NAME                     READY   STATUS             RESTARTS   AGE
# toolbox-alpha-xxx        0/1     ImagePullBackOff   0          30s
```

**Solution:**
```bash
# Check if image exists in Kind
docker exec -it platform-manager-control-plane crictl images | grep toolbox

# If not found, reload the image
kind load docker-image platform-manager-toolbox:dev --name platform-manager
```

---

### Issue: kubectl permissions not working in pod

**Error:**
```bash
kubectl exec $POD_NAME -n toolbox-sessions -- kubectl get pods
# Error from server (Forbidden): pods is forbidden
```

**Solution:**
```bash
# Check ClusterRoleBinding
kubectl get clusterrolebinding toolbox-session -o yaml

# Verify ServiceAccount
kubectl get pod $POD_NAME -n toolbox-sessions -o jsonpath='{.spec.serviceAccountName}'
# Should output: toolbox-session
```

---

## Next Steps

1. ✅ **You've tested terminal locally** - All endpoints work!
2. 📱 **Frontend Integration** - Add terminal UI component
3. 🚀 **Deploy to Dev Environment** - Test in real cluster
4. 🔐 **Review Security** - Ensure RBAC is appropriate for your org
5. 📊 **Add Monitoring** - Set up alerts for session metrics

---

## Additional Commands

### View Manager Logs
```bash
# If running in foreground, logs appear in terminal

# If running in background
tail -f manager.log
```

### View Terminal Session Logs
```bash
# Manager logs show terminal events
grep "terminal" manager.log

# Example entries:
# INFO terminal-manager terminal session created {"sessionID": "..."}
# INFO terminal session started {"audit": "..."}
# INFO terminal session ended {"reason": "user_requested"}
```

### Clean Up All Sessions
```bash
# Delete all pods in toolbox namespace
kubectl delete pods --all -n toolbox-sessions

# Or delete the entire namespace and recreate
kubectl delete namespace toolbox-sessions
kubectl create namespace toolbox-sessions
```

### Monitor Resource Usage
```bash
# Watch pods in toolbox namespace
watch kubectl get pods -n toolbox-sessions

# Check resource usage
kubectl top pods -n toolbox-sessions
```

---

**Ready to test? Run:**
```bash
./setup-phase6-terminal.sh && make build && ./bin/manager --enable-terminal=true
```

Then in another terminal:
```bash
./test-phase6-terminal.sh
```

Good luck! 🚀

