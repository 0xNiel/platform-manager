# Phase 6: Web Terminal - Quick Reference

## Enable/Disable Terminal

### At Startup (Recommended)

```bash
# Enable
./manager --enable-terminal=true --terminal-namespace=toolbox-sessions

# Disable (default)
./manager  # Terminal disabled by default
```

### Via Environment Variables

```yaml
env:
  - name: ENABLE_TERMINAL
    value: "true"  # or "false"
```

### Via Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: platform-manager-config
data:
  ENABLE_TERMINAL: "true"
  TERMINAL_NAMESPACE: "toolbox-sessions"
  TERMINAL_IMAGE: "platform-manager-toolbox:latest"
  TERMINAL_IDLE_TIMEOUT: "10m"
```

---

## Access Control Matrix

| Role | Terminal Access | Other Capabilities |
|------|----------------|-------------------|
| `admin` | ✅ Yes | All capabilities |
| `infra` | ✅ Yes | Argo sync, Crossplane pause/reconcile |
| `ml` | ❌ No | Argo refresh only |
| `readonly` | ❌ No | None |

**How it works**:
- Gateway assigns role based on OAuth2 groups
- Backend checks `CapUseTerminal` capability before granting access
- Frontend queries `/api/v1/auth/capabilities` to show/hide terminal button

---

## Quick Test Commands

```bash
# 1. Build and deploy
make docker-build-toolbox
make kind-load-toolbox

# 2. Create namespace
kubectl create ns toolbox-sessions
kubectl create sa toolbox-session -n toolbox-sessions

# 3. Start manager with terminal
go run ./cmd/main.go --enable-terminal=true

# 4. Check capabilities
curl -H "X-Dev-Role: admin" http://localhost:9080/api/v1/auth/capabilities

# 5. Create terminal session
curl -X POST -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{"tenantId":"alpha"}' \
  http://localhost:9080/api/v1/terminal/sessions

# 6. List sessions
curl -H "X-Dev-Role: admin" http://localhost:9080/api/v1/terminal/sessions

# 7. Delete session
curl -X DELETE -H "X-Dev-Role: admin" \
  http://localhost:9080/api/v1/terminal/sessions/SESSION_ID
```

---

## Frontend Integration

```typescript
// 1. Check if user can access terminal
const response = await fetch('/api/v1/auth/capabilities');
const data = await response.json();

if (data.features.canUseTerminal) {
  showTerminalButton();
}

// 2. Check if terminal is enabled globally
const configResponse = await fetch('/api/v1/terminal/config');
const config = await configResponse.json();

if (config.enabled && config.hasAccess) {
  enableTerminalFeature();
}

// 3. Create session
const session = await createTerminalSession('alpha');

// 4. Connect WebSocket
const ws = new WebSocket(`wss://platform.example.com${session.wsUrl}`);
terminal.onData((data) => ws.send(data));
ws.onmessage = (event) => terminal.write(event.data);
```

---

## Configuration Levels

```
┌─────────────────────────────────────────────────────┐
│ Level 1: Global Enable Flag                        │
│ --enable-terminal=true/false                        │
│ ↓ If disabled, all terminal APIs return 503        │
├─────────────────────────────────────────────────────┤
│ Level 2: Role-Based Capabilities                   │
│ Only admin/infra roles have CapUseTerminal         │
│ ↓ If user lacks capability, return 403             │
├─────────────────────────────────────────────────────┤
│ Level 3: Tenant Whitelist (Optional)               │
│ Config.AllowedTenants = ["alpha", "beta"]         │
│ ↓ If tenant not allowed, return 403                │
├─────────────────────────────────────────────────────┤
│ Level 4: Resource Limits                           │
│ Max sessions, idle timeouts, resource quotas       │
│ ↓ If limits exceeded, return 429/503               │
└─────────────────────────────────────────────────────┘
```

---

## API Endpoints

| Endpoint | Method | Auth | Purpose |
|----------|--------|------|---------|
| `/api/v1/auth/capabilities` | GET | Any | Get user capabilities |
| `/api/v1/terminal/config` | GET | Any | Get terminal config |
| `/api/v1/terminal/sessions` | POST | `CapUseTerminal` | Create session |
| `/api/v1/terminal/sessions` | GET | `CapUseTerminal` | List sessions |
| `/api/v1/terminal/sessions/:id` | GET | `CapUseTerminal` | Get session |
| `/api/v1/terminal/sessions/:id` | DELETE | `CapUseTerminal` | Close session |
| `/api/v1/terminal/sessions/:id/ws` | WebSocket | `CapUseTerminal` | Connect terminal |

---

## Security Checklist

- [ ] Terminal feature disabled by default
- [ ] Only admin/infra roles have access
- [ ] Toolbox pods run as non-root (UID 1000)
- [ ] No privilege escalation allowed
- [ ] All capabilities dropped
- [ ] Resource limits enforced
- [ ] Idle timeout configured
- [ ] ServiceAccount with minimal RBAC
- [ ] Audit logging enabled
- [ ] Network policies applied (optional)

---

## Troubleshooting

**Terminal button not showing in UI?**
→ Check `/api/v1/auth/capabilities` - is `terminal:use` in the list?

**"Terminal feature is disabled" error?**
→ Start manager with `--enable-terminal=true`

**Pod fails to start?**
→ Check: image exists, ServiceAccount exists, namespace exists

**WebSocket connection fails?**
→ Check: session exists, pod is running, gateway allows WebSocket upgrade

**Commands don't work in terminal?**
→ Check RBAC: `kubectl auth can-i --as=system:serviceaccount:toolbox-sessions:toolbox-session get pods`

---

## Per-Environment Configuration

**Development** (permissive):
```bash
--enable-terminal=true
--terminal-idle-timeout=30m  # Longer for debugging
```

**Production** (restricted):
```bash
--enable-terminal=false  # Disable unless needed
# Or enable with strict limits:
--enable-terminal=true
--terminal-idle-timeout=5m
# + Tenant whitelist in code
# + Restrictive NetworkPolicies
```

**QA/Staging** (moderate):
```bash
--enable-terminal=true
--terminal-idle-timeout=10m
```

---

## Files Created

```
internal/terminal/
  ├── types.go       # Config, Session, SessionOptions
  ├── manager.go     # Session lifecycle, pod management
  └── recorder.go    # Audit logging

internal/api/handlers/
  ├── terminal.go    # WebSocket handler
  └── auth.go        # Capabilities endpoint

internal/api/middleware/
  └── authz.go       # Updated with CapUseTerminal

cmd/main.go          # Added terminal flags

Dockerfile.toolbox   # Toolbox container image

documentation/
  ├── PHASE6_COMPLETE.md     # Full documentation
  └── PHASE6_QUICKREF.md     # This file
```

---

**See `PHASE6_COMPLETE.md` for full documentation.**

