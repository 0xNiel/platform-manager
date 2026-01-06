# Phase 6: WebSocket Testing Guide

## Overview
This guide walks through testing the actual WebSocket terminal connections using `wscat` and the browser.

---

## Prerequisites

Before running WebSocket tests, ensure:

1. **Manager is running with terminal enabled**
2. **Kind cluster is accessible**
3. **Toolbox image is loaded in Kind**
4. **wscat is installed** (for CLI testing)

---

## Step 1: Restart Manager with Terminal Enabled

The manager needs to be restarted with terminal feature flags.

### Option A: Using the start script (Recommended)

```bash
# Stop current manager (if running)
pkill -f "bin/manager"

# Start with terminal enabled
./start-phase6.sh
```

This will start the manager in the foreground with terminal enabled. You'll see logs indicating:
- `Terminal feature enabled`
- `Registering terminal routes`

### Option B: Manual start

```bash
# Build
make build

# Run with terminal flags
./bin/manager \
    --api-addr=:9080 \
    --enable-terminal=true \
    --terminal-namespace=toolbox-sessions \
    --terminal-image=platform-manager-toolbox:dev \
    --terminal-idle-timeout=10m \
    --terminal-max-sessions=20 \
    2>&1 | tee manager-phase6.log
```

---

## Step 2: Run Automated WebSocket Tests

Open a **new terminal** (keep the manager running in the first one).

### Run the test script:

```bash
./test-phase6-websocket.sh
```

This script will:
1. ✓ Check prerequisites (manager, Kind, wscat)
2. ✓ Verify terminal configuration
3. ✓ Create a terminal session
4. ✓ Wait for the toolbox pod to be ready
5. ✓ List active sessions
6. ⏸️ Wait for manual WebSocket testing (5 minutes)

### When the script pauses, you'll see:

```
WebSocket URL:
  ws://localhost:9080/api/v1/terminal/sessions/{SESSION_ID}/ws

To test the WebSocket connection manually, run:
  wscat -c "ws://..." -H "X-Forwarded-User: admin" -H "X-Forwarded-Groups: platform-admins"

Once connected, try these commands:
  ls -la
  pwd
  kubectl get namespaces
  helm version
  argocd version --client
```

### Open a **third terminal** and run the wscat command shown

You should see:
```
Connected (press CTRL+C to quit)
>
```

Now type commands and see them execute in the terminal!

---

## Step 3: Browser Testing

While the WebSocket test is running, test the frontend:

1. **Open browser** to `http://localhost:9083`
2. **Look for the terminal icon** (💻) in the nav bar (top right, after "Troubleshooting")
3. **Click the terminal icon** - the drawer should slide up from the bottom
4. **Terminal should auto-connect** and show:
   - Connection status
   - Tenant selector (if multiple tenants)
   - Terminal prompt

5. **Try commands:**
   ```bash
   ls -la
   pwd
   echo $USER
   kubectl get pods -n tenant-alpha
   helm version
   ```

6. **Test navigation:**
   - Navigate to different pages (Dashboard, Tenants, Resources)
   - Terminal drawer should persist across all pages
   - Terminal session should remain connected

7. **Test terminal controls:**
   - Resize the drawer (drag the handle at the top)
   - Minimize the terminal (click the icon again)
   - Reopen and verify session persists

---

## Step 4: Verify Session Management

### Check active sessions via API:

```bash
curl -s -H "X-Forwarded-User: admin" \
     -H "X-Forwarded-Groups: platform-admins" \
     http://localhost:9080/api/v1/terminal/sessions | jq .
```

### Check pods in Kubernetes:

```bash
kubectl get pods -n toolbox-sessions
kubectl describe pod -n toolbox-sessions
```

### Check audit logs:

```bash
ls -la /var/log/platform-manager/terminal-audit/
# or if running locally, check where logs are being written
```

---

## Expected Behaviors

### ✓ Successful Connection
- Terminal drawer opens smoothly
- Connection status shows "Connected" (green dot in nav icon)
- Commands execute and show output
- Session persists across page navigation

### ✓ Session Management
- Can create multiple sessions (up to max limit)
- Each session has its own toolbox pod
- Sessions timeout after idle period (10m by default)

### ✓ Security
- Only admin and infra roles can access terminal
- Read-only and ML roles see no terminal icon
- Sessions are isolated per tenant

### ✗ Common Issues

**"Forbidden: insufficient permissions"**
- Check that X-Forwarded-User header is set
- Verify user role has `CapUseTerminal` capability

**"Failed to create session"**
- Check toolbox image is loaded in Kind: `docker exec -it platform-manager-control-plane crictl images | grep toolbox`
- Check namespace exists: `kubectl get ns toolbox-sessions`
- Check RBAC: `kubectl get sa,role,rolebinding -n toolbox-sessions`

**WebSocket connection failed**
- Check manager logs for errors
- Verify pod is running: `kubectl get pods -n toolbox-sessions`
- Check pod logs: `kubectl logs -n toolbox-sessions <pod-name>`

---

## Cleanup

After testing:

1. **Stop WebSocket test** - Press Ctrl+C in the third terminal (wscat)
2. **Stop automated test** - Press Ctrl+C in the second terminal (test script will cleanup)
3. **Stop manager** - Press Ctrl+C in the first terminal

Or manually delete sessions:

```bash
# Delete a specific session
curl -X DELETE \
  -H "X-Forwarded-User: admin" \
  -H "X-Forwarded-Groups: platform-admins" \
  http://localhost:9080/api/v1/terminal/sessions/{SESSION_ID}

# Delete all pods
kubectl delete pods -n toolbox-sessions --all
```

---

## Next Steps

After successful WebSocket testing:
- **Step 3**: Add Prometheus metrics for terminal sessions
- **Step 4**: Enhance auditing with shell wrapper for command tracking

---

## Troubleshooting

### Manager won't start with terminal enabled

Check logs for:
```
Failed to initialize terminal manager
```

Possible causes:
- Invalid kubeconfig
- Missing RBAC permissions
- Invalid image name

### Pod stuck in ImagePullBackOff

```bash
kubectl describe pod -n toolbox-sessions <pod-name>
```

Solution: Reload toolbox image into Kind
```bash
kind load docker-image platform-manager-toolbox:dev --name platform-manager
```

### Commands not executing in terminal

Check pod logs:
```bash
kubectl logs -n toolbox-sessions <pod-name>
```

Check WebSocket connection in browser DevTools:
- Network tab → WS → Click connection → Messages

---

## Success Criteria

WebSocket testing is complete when:
- [x] Manager starts with terminal enabled
- [x] Terminal config API returns `enabled: true, hasAccess: true`
- [x] Can create terminal sessions via API
- [x] Toolbox pods start successfully
- [x] Can connect via wscat and execute commands
- [x] Browser terminal drawer connects and executes commands
- [x] Terminal persists across page navigation
- [x] Sessions timeout after idle period
- [x] Audit logs are created

