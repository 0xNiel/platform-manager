# Phase 6: Web Terminal - NOTES FOR USER

## How to Turn Terminal On/Off

You asked for a way to turn the terminal feature on and off depending on the access you give to users. Here's what was implemented:

### ✅ Multi-Level Control System

#### 1. **Global Enable/Disable** (Deployment Level)
The terminal is **DISABLED by default** and must be explicitly enabled:

```bash
# Enable at startup
./manager --enable-terminal=true

# Or via environment variable
export ENABLE_TERMINAL=true
```

```yaml
# In Kubernetes deployment
env:
  - name: ENABLE_TERMINAL
    value: "true"  # Change to "false" to disable globally
```

#### 2. **Role-Based Access** (Authorization Level)
Only certain roles can use the terminal:

- ✅ `admin` - Full terminal access
- ✅ `infra` - Full terminal access
- ❌ `ml` - No terminal access
- ❌ `readonly` - No terminal access

**This is enforced automatically** via the `CapUseTerminal` capability in the authorization middleware.

#### 3. **Per-Tenant Restrictions** (Optional Fine-Grained Control)
You can optionally restrict terminal to specific tenants:

```go
// In your configuration
terminalConfig := terminal.Config{
    Enabled: true,
    AllowedTenants: []string{"alpha", "beta"},  // Only these tenants
    // Empty slice = all tenants allowed
}
```

#### 4. **Frontend Feature Detection** (User Experience)
The frontend can query capabilities to show/hide the terminal button:

```typescript
// Frontend checks capabilities
GET /api/v1/auth/capabilities
{
  "features": {
    "canUseTerminal": true  // Shows/hides terminal button
  }
}
```

---

## Quick Commands

### Enable Terminal Globally
```bash
# At startup
./manager --enable-terminal=true

# In Kubernetes
kubectl set env deployment/platform-manager ENABLE_TERMINAL=true
```

### Disable Terminal Globally
```bash
# At startup (default)
./manager  # No flag = disabled

# In Kubernetes
kubectl set env deployment/platform-manager ENABLE_TERMINAL=false
```

### Check Who Has Access
```bash
# Check your capabilities
curl http://localhost:9080/api/v1/auth/capabilities \
  -H "X-Auth-Request-User: your-username" \
  -H "X-Auth-Request-Groups: platform-admin"
```

---

## Configuration Matrix

| Want to... | How to Do It |
|-----------|--------------|
| **Disable terminal completely** | Don't set `--enable-terminal` flag (it's off by default) |
| **Enable for admin/infra only** | Set `--enable-terminal=true` (role check is automatic) |
| **Enable for specific tenants** | Set `AllowedTenants` in Config |
| **Temporarily disable** | Set environment variable `ENABLE_TERMINAL=false` |
| **Change timeout** | Set `--terminal-idle-timeout=5m` |
| **Limit max sessions** | Set `MaxSessions` in Config (default: 20) |

---

## Environment-Specific Recommendations

### Development
```bash
--enable-terminal=true
--terminal-idle-timeout=30m  # Longer for debugging
```

### Production
```bash
# Option 1: Disable entirely (safest)
# (Don't set --enable-terminal flag)

# Option 2: Enable with strict controls
--enable-terminal=true
--terminal-idle-timeout=5m  # Shorter timeout
# + Set AllowedTenants to specific list
# + Apply NetworkPolicies
```

### QA/Staging
```bash
--enable-terminal=true
--terminal-idle-timeout=10m
```

---

## How Users Know If They Have Access

1. **User visits platform UI**
2. **Frontend calls** `/api/v1/auth/capabilities`
3. **If `canUseTerminal: true`** → Terminal button shows
4. **If `canUseTerminal: false`** → Terminal button hidden

**No error messages, just feature not visible** if they don't have access.

---

## Security Notes

✅ **Disabled by default** - Feature must be explicitly enabled  
✅ **Role-based** - Only admin/infra can access  
✅ **Audited** - All sessions logged  
✅ **Time-limited** - Sessions auto-close after idle timeout  
✅ **Resource-limited** - Max sessions, CPU/memory limits  
✅ **Tenant-isolated** - Each session scoped to tenant RBAC  

---

## Deployment Checklist

When deploying to a new environment:

- [ ] Decide if terminal should be enabled
- [ ] Set `ENABLE_TERMINAL` environment variable appropriately
- [ ] Build and push `platform-manager-toolbox` image
- [ ] Create `toolbox-sessions` namespace
- [ ] Create `toolbox-session` ServiceAccount with appropriate RBAC
- [ ] Test with admin user
- [ ] Verify readonly user cannot access
- [ ] Document decision in environment runbook

---

## See Also

- **[PHASE6_COMPLETE.md](PHASE6_COMPLETE.md)** - Full technical documentation
- **[PHASE6_QUICKREF.md](PHASE6_QUICKREF.md)** - Quick reference guide
- **[PHASE6_SUMMARY.md](PHASE6_SUMMARY.md)** - Implementation summary

---

**Bottom Line**: The terminal is **OFF by default**. To turn it ON, set `--enable-terminal=true` flag or `ENABLE_TERMINAL=true` environment variable. Only admin and infra roles will have access automatically.

