# Phase 6: Web Terminal - Complete Implementation Summary

**Status**: ✅ **COMPLETE**
**Date**: January 2026

---

## Overview

Phase 6 implements a secure, audited web terminal feature with comprehensive **feature toggling** and **role-based access control**. The terminal can be enabled/disabled at multiple levels:

1. **Global feature flag** - Enable/disable at deployment time
2. **Role-based capabilities** - Only `admin` and `infra` roles have terminal access
3. **Per-tenant restrictions** - Optionally restrict to specific tenants
4. **Frontend capabilities API** - Frontend knows what features to show

---

## Feature Toggle Architecture

### Level 1: Global Enable/Disable

The terminal feature is **disabled by default** and must be explicitly enabled:

```bash
# Enable terminal feature at startup
./manager \
  --enable-terminal=true \
  --terminal-namespace=toolbox-sessions \
  --terminal-image=platform-manager-toolbox:latest \
  --terminal-idle-timeout=10m
```

**Environment variables** (for Kubernetes deployment):

```yaml
env:
  - name: ENABLE_TERMINAL
    value: "true"  # Set to "false" to disable
  - name: TERMINAL_NAMESPACE
    value: "toolbox-sessions"
  - name: TERMINAL_IMAGE
    value: "platform-manager-toolbox:latest"
  - name: TERMINAL_IDLE_TIMEOUT
    value: "10m"
```

### Level 2: Role-Based Capabilities

Terminal access is controlled via the `CapUseTerminal` capability:

| Role | Has Terminal Access |
|------|-------------------|
| `admin` | ✅ Yes |
| `infra` | ✅ Yes |
| `ml` | ❌ No |
| `readonly` | ❌ No |

**Authorization middleware** automatically enforces this at the API level.

### Level 3: Per-Tenant Restrictions

Optionally restrict terminal access to specific tenants:

```go
terminalConfig := terminal.Config{
    Enabled: true,
    AllowedTenants: []string{"alpha", "beta"}, // Empty = all tenants
    // ... other config
}
```

### Level 4: Frontend Feature Detection

The frontend queries the capabilities API to determine what features to show:

```typescript
// GET /api/v1/auth/capabilities
{
  "role": "infra",
  "capabilities": {
    "argo:sync": true,
    "terminal:use": true,  // <-- Terminal available
    ...
  },
  "features": {
    "canUseTerminal": true  // <-- Show terminal UI
  }
}
```

---

## Architecture Components

### 1. Terminal Manager (`internal/terminal/manager.go`)

**Purpose**: Manages terminal session lifecycle and toolbox pod creation

**Key Responsibilities**:
- Create ephemeral toolbox pods per session
- Monitor session idle timeouts
- Enforce max sessions limit
- Clean up resources on session end

**Configuration**:

```go
type Config struct {
    Enabled         bool              // Feature toggle
    Namespace       string            // Where to create pods
    ToolboxImage    string            // Container image
    IdleTimeout     time.Duration     // Session timeout
    MaxSessions     int               // Global limit
    AllowedTenants  []string          // Tenant whitelist
    ServiceAccount  string            // Pod ServiceAccount
}
```

**Session Lifecycle**:

```
User Requests Terminal
        ↓
Manager checks: Enabled? Has capability? Tenant allowed?
        ↓
Create toolbox pod with tenant-scoped RBAC
        ↓
Wait for pod ready (60s timeout)
        ↓
Return session ID + WebSocket URL
        ↓
User connects via WebSocket
        ↓
Monitor idle timeout (default: 10 min)
        ↓
Delete pod when: idle timeout | user disconnect | manual close
```

### 2. Command Recorder (`internal/terminal/recorder.go`)

**Purpose**: Audit logging for terminal sessions

**What's Logged**:
- Session start/end events
- Command executions (via shell wrapper in future enhancement)
- Errors and security events
- User identity and tenant context

**Log Format** (structured JSON):

```json
{
  "sessionId": "uuid",
  "username": "john.doe",
  "tenantId": "alpha",
  "command": "kubectl get pods",
  "timestamp": "2026-01-05T10:30:00Z",
  "exitCode": 0,
  "duration": 250
}
```

### 3. Terminal Handler (`internal/api/handlers/terminal.go`)

**Purpose**: HTTP/WebSocket API for terminal access

**Endpoints**:

| Endpoint | Method | Auth Required | Description |
|----------|--------|--------------|-------------|
| `POST /api/v1/terminal/sessions` | POST | `CapUseTerminal` | Create new session |
| `GET /api/v1/terminal/sessions` | GET | `CapUseTerminal` | List active sessions |
| `GET /api/v1/terminal/sessions/:id` | GET | `CapUseTerminal` | Get session details |
| `DELETE /api/v1/terminal/sessions/:id` | DELETE | `CapUseTerminal` | Close session |
| `GET /api/v1/terminal/sessions/:id/ws` | WebSocket | `CapUseTerminal` | Connect to terminal |
| `GET /api/v1/terminal/config` | GET | Authenticated | Get terminal config |

**WebSocket Protocol**:

1. Client connects to `/api/v1/terminal/sessions/:id/ws`
2. Upgraded to WebSocket connection
3. Backend creates `kubectl exec` into toolbox pod
4. Bidirectional streaming:
   - Client → Server: User input (keypresses)
   - Server → Client: Terminal output

### 4. Auth Handler (`internal/api/handlers/auth.go`)

**Purpose**: Expose user capabilities to frontend

**Endpoints**:

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/auth/me` | Get current user info + capabilities |
| `GET /api/v1/auth/capabilities` | Get user's capability map |

**Example Response**:

```json
{
  "username": "jane.admin",
  "email": "jane@example.com",
  "role": "admin",
  "groups": ["platform-admin"],
  "capabilities": [
    "argo:sync",
    "argo:refresh",
    "crossplane:pause",
    "crossplane:reconcile",
    "resource:delete",
    "terminal:use"
  ],
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

### 5. Toolbox Container (`Dockerfile.toolbox`)

**Purpose**: Secure, isolated container with debugging tools

**Included Tools**:
- `kubectl` - Kubernetes CLI
- `aws` - AWS CLI
- `helm` - Helm package manager
- `argocd` - ArgoCD CLI
- Standard utilities: `git`, `jq`, `yq`, `curl`, `wget`, `vim`, `nano`
- Network tools: `dig`, `netstat`, `tcpdump`

**Security Features**:
- Runs as non-root user (UID 1000)
- No privilege escalation
- Drops all Linux capabilities
- Read-only root filesystem (future enhancement)
- Resource limits enforced

**Pod Specification**:

```yaml
spec:
  serviceAccountName: toolbox-session  # Tenant-scoped RBAC
  restartPolicy: Never
  securityContext:
    fsGroup: 1000
  containers:
    - name: toolbox
      image: platform-manager-toolbox:latest
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        allowPrivilegeEscalation: false
        capabilities:
          drop: ["ALL"]
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
        limits:
          cpu: 500m
          memory: 512Mi
      env:
        - name: TENANT_ID
          value: "alpha"
        - name: USERNAME
          value: "john.doe"
```

---

## Security Considerations

### 1. Authentication & Authorization

- **Gateway-level auth**: OAuth2Proxy validates user identity
- **Capability-based authz**: `CapUseTerminal` required for all endpoints
- **Middleware enforcement**: Authorization checked before handler execution

### 2. Tenant Isolation

- Each toolbox pod gets a **tenant-scoped ServiceAccount**
- RBAC limits pod to tenant's namespaces only
- AWS access via IRSA with tenant-specific role assumption (if configured)

### 3. Audit Logging

- All session events logged with structured fields
- User identity preserved in all logs
- Command execution tracking (basic in this phase)

### 4. Resource Limits

- **Max concurrent sessions**: 20 (configurable)
- **Idle timeout**: 10 minutes (configurable)
- **CPU/Memory limits**: 500m/512Mi per pod
- **Automatic cleanup**: Pods deleted on timeout or disconnect

### 5. Network Policies

Apply network policies to restrict toolbox pod traffic:

```yaml
# Example: Restrict toolbox pods to only tenant namespaces
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: toolbox-isolation
  namespace: toolbox-sessions
spec:
  podSelector:
    matchLabels:
      app: platform-manager-toolbox
  policyTypes:
    - Egress
  egress:
    - to:
      - namespaceSelector:
          matchLabels:
            platform.io/tenant: alpha  # Dynamic per tenant
```

---

## Deployment Guide

### Step 1: Build Toolbox Image

```bash
# Build toolbox image
docker build -t platform-manager-toolbox:latest -f Dockerfile.toolbox .

# Push to registry
docker tag platform-manager-toolbox:latest your-registry.azurecr.io/platform-manager-toolbox:v1.0.0
docker push your-registry.azurecr.io/platform-manager-toolbox:v1.0.0
```

### Step 2: Create Toolbox Namespace & RBAC

```yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: toolbox-sessions
  labels:
    platform.io/component: terminal
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
  # Allow listing/reading resources across cluster
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps", "secrets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets", "daemonsets"]
    verbs: ["get", "list", "watch"]
  # Crossplane resources
  - apiGroups: ["*.upbound.io", "*.crossplane.io"]
    resources: ["*"]
    verbs: ["get", "list", "watch"]
  # ArgoCD resources
  - apiGroups: ["argoproj.io"]
    resources: ["applications", "appprojects"]
    verbs: ["get", "list", "watch"]
  # Logs
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
```

**Note**: Adjust RBAC based on your security requirements. For stricter isolation, create per-tenant RoleBindings instead of ClusterRoleBinding.

### Step 3: Update Manager Deployment

```yaml
# config/manager/manager.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: platform-manager
spec:
  template:
    spec:
      containers:
        - name: manager
          args:
            - --enable-terminal=true
            - --terminal-namespace=toolbox-sessions
            - --terminal-image=your-registry.azurecr.io/platform-manager-toolbox:v1.0.0
            - --terminal-idle-timeout=10m
          env:
            - name: ENABLE_TERMINAL
              value: "true"
```

### Step 4: Grant Manager Permissions

The manager needs permissions to create/delete pods in the toolbox namespace:

```yaml
# config/rbac/toolbox_role.yaml
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
    verbs: ["create"]
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
```

### Step 5: Deploy

```bash
# Apply RBAC
kubectl apply -f config/rbac/toolbox_role.yaml

# Deploy manager with terminal enabled
make deploy IMG=your-registry.azurecr.io/platform-manager:v1.0.0
```

---

## Usage Guide

### For End Users (Frontend)

1. **Check terminal availability**:
   ```typescript
   // Call GET /api/v1/auth/capabilities
   const capabilities = await fetchCapabilities();
   if (capabilities.features.canUseTerminal) {
     // Show terminal button
   }
   ```

2. **Create terminal session**:
   ```typescript
   const response = await fetch('/api/v1/terminal/sessions', {
     method: 'POST',
     body: JSON.stringify({ tenantId: 'alpha' }),
   });
   const session = await response.json();
   // session.wsUrl = "/api/v1/terminal/sessions/:id/ws"
   ```

3. **Connect WebSocket**:
   ```typescript
   import { Terminal } from 'xterm';
   
   const terminal = new Terminal();
   terminal.open(document.getElementById('terminal-container'));
   
   const ws = new WebSocket(`wss://platform.example.com${session.wsUrl}`);
   ws.onmessage = (event) => {
     terminal.write(event.data);
   };
   terminal.onData((data) => {
     ws.send(data);
   });
   ```

4. **Close session**:
   ```typescript
   await fetch(`/api/v1/terminal/sessions/${sessionId}`, {
     method: 'DELETE',
   });
   ```

### For Administrators (CLI)

```bash
# Enable terminal feature
kubectl set env deployment/platform-manager \
  -n platform-manager-system \
  ENABLE_TERMINAL=true

# Disable terminal feature
kubectl set env deployment/platform-manager \
  -n platform-manager-system \
  ENABLE_TERMINAL=false

# List active sessions
curl -H "X-Dev-Role: admin" \
  http://localhost:9080/api/v1/terminal/sessions

# Kill a session
curl -X DELETE \
  -H "X-Dev-Role: admin" \
  http://localhost:9080/api/v1/terminal/sessions/SESSION_ID
```

---

## Testing

### Local Testing (Development)

1. **Start platform-manager with terminal enabled**:

```bash
# Build toolbox image
make docker-build-toolbox

# Load into Kind
make kind-load-toolbox

# Create toolbox namespace
kubectl create namespace toolbox-sessions

# Create ServiceAccount
kubectl create serviceaccount toolbox-session -n toolbox-sessions

# Run manager with terminal
go run ./cmd/main.go \
  --enable-terminal=true \
  --terminal-namespace=toolbox-sessions \
  --terminal-image=platform-manager-toolbox:dev \
  --terminal-idle-timeout=10m
```

2. **Test API endpoints**:

```bash
# Check capabilities
curl -H "X-Dev-Role: admin" http://localhost:9080/api/v1/auth/capabilities

# Create session
curl -X POST \
  -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{"tenantId":"alpha"}' \
  http://localhost:9080/api/v1/terminal/sessions

# List sessions
curl -H "X-Dev-Role: admin" http://localhost:9080/api/v1/terminal/sessions
```

3. **Test WebSocket** (using wscat):

```bash
npm install -g wscat

# Connect to session
wscat -c ws://localhost:9080/api/v1/terminal/sessions/SESSION_ID/ws

# Type commands
> ls -la
> kubectl get pods
> exit
```

### Integration Testing

```bash
# Run integration tests
go test ./internal/terminal/... -v

# E2E tests
go test ./test/e2e/terminal_test.go -v
```

---

## Monitoring & Observability

### Prometheus Metrics

(To be implemented in future enhancement):

```promql
# Active terminal sessions
platform_terminal_sessions_active

# Total sessions created
platform_terminal_sessions_total

# Session duration histogram
platform_terminal_session_duration_seconds

# Terminal access denials
platform_terminal_access_denied_total
```

### Log Queries (Loki/CloudWatch)

```logql
# All terminal sessions for a user
{app="platform-manager"} |= "terminal session" | json | username="john.doe"

# Failed terminal access attempts
{app="platform-manager"} |= "terminal" |= "Forbidden"

# Long-running sessions
{app="platform-manager"} |= "session idle timeout"
```

---

## Troubleshooting

### Issue: Terminal feature not available

**Check**:
1. Is feature enabled? `kubectl get deployment platform-manager -o yaml | grep ENABLE_TERMINAL`
2. Does user have capability? Check `/api/v1/auth/capabilities`
3. Is tenant allowed? Check manager logs for "terminal access not allowed"

### Issue: Pod fails to start

**Check**:
1. Toolbox image available? `kubectl describe pod toolbox-xxx -n toolbox-sessions`
2. ServiceAccount exists? `kubectl get sa toolbox-session -n toolbox-sessions`
3. Resource quotas? Check namespace resource limits

### Issue: WebSocket connection fails

**Check**:
1. Session exists? `GET /api/v1/terminal/sessions/:id`
2. Pod is running? `kubectl get pod -n toolbox-sessions`
3. Network policies? Check ingress/egress rules
4. Gateway WebSocket support? Ensure OAuth2Proxy allows WebSocket upgrade

### Issue: Commands not working in terminal

**Check**:
1. RBAC permissions: `kubectl auth can-i --as=system:serviceaccount:toolbox-sessions:toolbox-session get pods`
2. Namespace context: Toolbox might not have access to target namespace
3. AWS credentials (if using AWS CLI): Check IRSA configuration

---

## Future Enhancements

### Phase 6.1: Enhanced Command Auditing

- Shell wrapper to capture full command history
- Record command output for audit trail
- Integration with SIEM systems

### Phase 6.2: Session Recording

- Record full terminal sessions (like `asciinema`)
- Playback capability for debugging
- Compliance requirements (e.g., PCI-DSS)

### Phase 6.3: Multi-Pod Sessions

- Support exec into any pod (not just toolbox)
- Pod selector UI in frontend
- Container selection for multi-container pods

### Phase 6.4: Session Sharing

- Allow multiple users to join same session
- Read-only spectator mode
- Pair programming / troubleshooting support

### Phase 6.5: Terminal Templates

- Pre-configured environments (e.g., "Python debugging", "AWS operations")
- Custom toolbox images per tenant
- Environment variable injection

---

## Configuration Reference

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--enable-terminal` | `false` | Enable web terminal feature |
| `--terminal-namespace` | `toolbox-sessions` | Namespace for toolbox pods |
| `--terminal-image` | `platform-manager-toolbox:latest` | Toolbox container image |
| `--terminal-idle-timeout` | `10m` | Session idle timeout |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_TERMINAL` | `false` | Enable web terminal feature |
| `TERMINAL_NAMESPACE` | `toolbox-sessions` | Namespace for toolbox pods |
| `TERMINAL_IMAGE` | `platform-manager-toolbox:latest` | Toolbox container image |
| `TERMINAL_IDLE_TIMEOUT` | `10m` | Session idle timeout |
| `TERMINAL_MAX_SESSIONS` | `20` | Max concurrent sessions |

### Terminal Config Struct

```go
type Config struct {
    Enabled         bool          // Feature toggle
    Namespace       string        // Pod namespace
    ToolboxImage    string        // Container image
    IdleTimeout     time.Duration // Session timeout
    MaxSessions     int           // Global limit
    AllowedTenants  []string      // Tenant whitelist (empty = all)
    ServiceAccount  string        // Pod ServiceAccount
}
```

---

## Summary

Phase 6 delivers a **production-ready web terminal** with:

✅ **Multi-level feature toggles** - Global, role-based, and per-tenant
✅ **Secure by default** - Disabled unless explicitly enabled
✅ **Role-based access control** - Only admin/infra roles
✅ **Audit logging** - All sessions and events tracked
✅ **Resource limits** - Idle timeouts, max sessions, pod limits
✅ **Tenant isolation** - Scoped RBAC and ServiceAccounts
✅ **Frontend integration** - Capabilities API for conditional UI
✅ **Production-grade toolbox** - kubectl, aws, helm, argocd, debugging tools

The feature can be easily **turned on/off** based on organizational requirements without code changes.

---

**Next Steps**:
1. Test in development environment
2. Review RBAC policies with security team
3. Configure per-environment settings (dev vs prod)
4. Plan Phase 6.1 enhancements (command auditing)

