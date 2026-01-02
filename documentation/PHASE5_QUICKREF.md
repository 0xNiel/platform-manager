# Phase 5 Quick Reference

## 🚀 Quick Start

```bash
# Start backend with rule evaluation
./bin/manager --rule-eval-interval=2m

# Test the API
./test-phase5.sh

# Access UI
http://localhost:9083/troubleshooting
```

## 📋 Built-in Rules (11 Total)

### Critical Severity
- `xr-failed-iam` - Crossplane resources failed due to IAM issues
- `provider-unhealthy` - Crossplane providers not healthy
- `iam-extra-privileges` - IAM roles with extra privileges (Phase 4 integration)

### High Severity
- `crashloop-backoff` - Pods crashing repeatedly (>5 restarts)
- `image-pull-failed` - Image pull failures (>10 min)
- `argo-sync-failed` - ArgoCD sync failures (>30 min)

### Medium Severity
- `paused-but-syncing` - Paused resources still managed by ArgoCD
- `stale-resource` - ResourceSummary not updated (>1 hour)
- `argo-out-of-sync-long` - Apps out of sync (>1 hour)
- `high-resource-usage` - Namespaces >80% of quota
- `pod-pending-long` - Pods pending (>15 min)

## 🔗 API Endpoints

```bash
# Platform Summary
GET /api/v1/troubleshooting/summary

# All Findings
GET /api/v1/troubleshooting/findings
GET /api/v1/troubleshooting/findings?severity=critical
GET /api/v1/troubleshooting/findings?tenant=tenant-alpha

# Tenant Findings
GET /api/v1/troubleshooting/tenants/{name}

# Single Finding
GET /api/v1/troubleshooting/findings/{id}

# Actions
POST /api/v1/troubleshooting/scan              # Trigger manual scan
POST /api/v1/troubleshooting/findings/{id}/resolve  # Mark resolved

# Rules
GET /api/v1/troubleshooting/rules              # List all rules
```

## 🎯 Common Use Cases

### Find All Critical Issues
```bash
curl "http://localhost:9080/api/v1/troubleshooting/findings?severity=critical" | jq
```

### Check Tenant Health
```bash
curl "http://localhost:9080/api/v1/troubleshooting/tenants/tenant-alpha" | jq
```

### Trigger Immediate Scan
```bash
curl -X POST "http://localhost:9080/api/v1/troubleshooting/scan"
```

### Get Summary Stats
```bash
curl "http://localhost:9080/api/v1/troubleshooting/summary" | jq '.critical, .high, .medium'
```

## 🧪 Testing Scenarios

### Test CrashLoop Detection
```bash
kubectl run crashtest --image=busybox --command -- sh -c "exit 1" -n tenant-alpha --labels="platform.io/tenant=tenant-alpha"
# Wait 2-3 minutes, check troubleshooting view
```

### Test ImagePull Detection
```bash
kubectl run imagefail --image=nonexistent:v999 -n tenant-beta --labels="platform.io/tenant=tenant-beta"
# Wait 10+ minutes, check troubleshooting view
```

### Test Resource Quota
```bash
# Create quota
kubectl create quota test-quota --hard=pods=2 -n tenant-alpha
# Create pods close to limit
kubectl run pod1 --image=nginx -n tenant-alpha
kubectl run pod2 --image=nginx -n tenant-alpha
# Check for high-resource-usage finding
```

## ⚙️ Configuration

### Evaluation Interval
```bash
# Fast (dev) - every minute
./bin/manager --rule-eval-interval=1m

# Default - every 2 minutes
./bin/manager --rule-eval-interval=2m

# Slow (prod) - every 10 minutes
./bin/manager --rule-eval-interval=10m
```

### Frontend Auto-Refresh
Edit `TroubleshootingView.vue`:
```typescript
// Change refresh interval (default: 30 seconds)
refreshInterval = window.setInterval(refreshFindings, 30000)  // 30s
refreshInterval = window.setInterval(refreshFindings, 60000)  // 60s
```

## 📊 Understanding Findings

### Finding Structure
```json
{
  "id": "a1b2c3d4",
  "ruleId": "crashloop-backoff",
  "ruleName": "Pod CrashLoopBackOff",
  "severity": "high",
  "status": "active",
  "title": "Pod crashloop-test is in CrashLoopBackOff",
  "message": "Container 'crash' has crashed 12 times...",
  "recommendation": "Check container logs: kubectl logs...",
  "resourceRef": {
    "kind": "Pod",
    "namespace": "tenant-alpha",
    "name": "crashloop-test"
  },
  "tenantName": "tenant-alpha",
  "firstSeen": "2026-01-02T19:00:00Z",
  "lastSeen": "2026-01-02T20:00:00Z",
  "occurrences": 6
}
```

### Severity Levels
- **Critical** 🔴 - Immediate action required (security, provider failures)
- **High** 🟠 - Important issues (crashes, sync failures)
- **Medium** 🟡 - Moderate issues (out of sync, resource usage)
- **Low** 🔵 - Minor issues
- **Info** ⚪ - Informational only

## 🔍 Troubleshooting the Troubleshooter

### No Findings Showing
```bash
# Check if rule evaluator is running
kubectl logs -l app=platform-manager | grep "rule evaluation"

# Check if resources have tenant labels
kubectl get pods -A --show-labels | grep platform.io/tenant

# Trigger manual scan
curl -X POST http://localhost:9080/api/v1/troubleshooting/scan

# Check API directly
curl http://localhost:9080/api/v1/troubleshooting/summary | jq
```

### Rules Not Registered
```bash
# Check rules list
curl http://localhost:9080/api/v1/troubleshooting/rules | jq '.total'
# Should show 11

# Check manager logs
kubectl logs -l app=platform-manager | grep "Rule evaluator initialized"
```

### Findings Not Updating
```bash
# Check evaluation interval
kubectl logs -l app=platform-manager | grep "rule-eval-interval"

# Check for errors
kubectl logs -l app=platform-manager | grep "rule evaluation failed"
```

## 📝 Adding Custom Rules

### 1. Create New Rule
```go
// internal/rules/builtin/custom_rules.go
type MyCustomRule struct{}

func (r *MyCustomRule) ID() string { return "my-custom-rule" }
func (r *MyCustomRule) Name() string { return "My Custom Check" }
func (r *MyCustomRule) Description() string { return "Checks for..." }
func (r *MyCustomRule) Severity() rules.Severity { return rules.SeverityMedium }

func (r *MyCustomRule) AppliesTo(resource *unstructured.Unstructured) bool {
    return resource.GetKind() == "MyResourceType"
}

func (r *MyCustomRule) Evaluate(ctx rules.RuleContext) ([]rules.Finding, error) {
    // Your detection logic here
    return []rules.Finding{}, nil
}
```

### 2. Register Rule
```go
// internal/rules/builtin/registry.go
func GetAllRules() []rules.Rule {
    return []rules.Rule{
        // ... existing rules ...
        &MyCustomRule{},
    }
}
```

### 3. Rebuild & Deploy
```bash
go build -o bin/manager ./cmd/main.go
kubectl rollout restart deployment/platform-manager
```

## 🎨 UI Customization

### Change Severity Colors
Edit `TroubleshootingView.vue`:
```scss
.severity-badge.critical {
  background: #ef4444;  // Change to your color
}
```

### Modify Finding Card Layout
Edit the `.finding-card` section in `TroubleshootingView.vue`

### Adjust Items Per Page
```typescript
const itemsPerPage = 20  // Change to desired number
```

## 🏗️ Architecture

```
┌──────────────────────────────────────────────┐
│              Rule Evaluator                   │
│  (Runs every 2 minutes)                      │
└────────────┬─────────────────────────────────┘
             │
             ├─> Scan Pods
             ├─> Scan ArgoCD Apps
             ├─> Scan Crossplane Resources
             ├─> Scan Providers
             └─> Scan ResourceQuotas
             │
             ├─> Apply Rules (11 total)
             ├─> Generate Findings
             ├─> Update Cache
             └─> Build Summaries
             │
             ▼
┌──────────────────────────────────────────────┐
│           REST API                            │
│  /api/v1/troubleshooting/*                   │
└────────────┬─────────────────────────────────┘
             │
             ▼
┌──────────────────────────────────────────────┐
│           Frontend                            │
│  Vue 3 + Auto-refresh (30s)                 │
└──────────────────────────────────────────────┘
```

## 📚 Related Documentation

- **Implementation Plan**: `documentation/IMPLEMENTATION_PLAN.md` (Section 5)
- **Complete Summary**: `documentation/PHASE5_COMPLETE.md`
- **Phase 1-4 Context**: `documentation/PHASES_1-3_COMPLETE.md`, `documentation/PHASE4_COMPLETE_SUMMARY.md`

## 🆘 Need Help?

1. Check logs: `kubectl logs -l app=platform-manager | grep rule`
2. Run test suite: `./test-phase5.sh`
3. Verify API: `curl http://localhost:9080/api/v1/troubleshooting/summary`
4. Check frontend: Open browser console at `/troubleshooting`

---

**Phase 5 Status:** ✅ Complete - All 11 rules operational, UI functional, API tested

