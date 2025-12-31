# Phase 3 Quick Reference

## 📋 What We're Building

**6 Action Types**:
1. ArgoCD Sync - Trigger application synchronization
2. ArgoCD Refresh - Refresh application state
3. Crossplane Pause - Stop resource reconciliation
4. Crossplane Unpause - Resume resource reconciliation
5. Crossplane Reconcile - Force immediate reconciliation
6. Resource Delete - Delete any resource (with safeguards)

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Request Flow                            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  1. ExtractUser Middleware                                   │
│     Extract: X-Auth-Request-User, X-Auth-Request-Groups     │
│     Map groups → role (admin/infra/ml/readonly)            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  2. RequireCapability Middleware                            │
│     Check: Does user's role have required capability?       │
│     - admin → ALL capabilities                              │
│     - infra → sync, refresh, pause, reconcile              │
│     - ml → refresh only                                     │
│     - readonly → NONE                                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  3. Action Handler                                          │
│     - Validate request                                      │
│     - Fetch resource from K8s                              │
│     - Apply action (patch annotations/spec)                │
│     - Update resource                                       │
│     - Log audit event                                       │
│     - Return result                                         │
└─────────────────────────────────────────────────────────────┘
```

## 🔐 Authorization Matrix

| Role     | Sync Argo | Refresh Argo | Pause XP | Reconcile XP | Delete |
|----------|-----------|--------------|----------|--------------|--------|
| admin    | ✅        | ✅           | ✅       | ✅           | ✅     |
| infra    | ✅        | ✅           | ✅       | ✅           | ❌     |
| ml       | ❌        | ✅           | ❌       | ❌           | ❌     |
| readonly | ❌        | ❌           | ❌       | ❌           | ❌     |

## 🎯 API Endpoints

### ArgoCD Actions
```
POST /api/v1/actions/argo/sync
POST /api/v1/actions/argo/refresh
```

### Crossplane Actions
```
POST /api/v1/actions/crossplane/pause
POST /api/v1/actions/crossplane/unpause
POST /api/v1/actions/crossplane/reconcile
```

### Resource Actions
```
DELETE /api/v1/actions/resources
```

## 📝 Request Examples

### 1. Sync ArgoCD Application
```bash
curl -X POST http://localhost:9080/api/v1/actions/argo/sync \
  -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "alpha-ml-platform",
    "namespace": "argocd",
    "prune": false,
    "dryRun": false
  }'
```

**Response**:
```json
{
  "success": true,
  "message": "Sync operation initiated",
  "application": "alpha-ml-platform",
  "namespace": "argocd"
}
```

### 2. Pause Crossplane Resource
```bash
curl -X POST http://localhost:9080/api/v1/actions/crossplane/pause \
  -H "X-Dev-Role: infra" \
  -H "Content-Type: application/json" \
  -d '{
    "group": "iam.aws.upbound.io",
    "version": "v1beta1",
    "kind": "Role",
    "name": "tenant-alpha-lambda-role"
  }'
```

**Response**:
```json
{
  "success": true,
  "message": "Resource paused",
  "resource": {
    "group": "iam.aws.upbound.io",
    "version": "v1beta1",
    "kind": "Role",
    "name": "tenant-alpha-lambda-role"
  },
  "paused": true
}
```

### 3. Delete Resource
```bash
curl -X DELETE http://localhost:9080/api/v1/actions/resources \
  -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{
    "group": "apps",
    "version": "v1",
    "kind": "Deployment",
    "namespace": "tenant-alpha",
    "name": "test-deployment",
    "confirm": "test-deployment"
  }'
```

## 🛡️ Safeguards

### Delete Safeguards
- ❌ Cannot delete resources in `kube-system`, `kube-public`, `default`
- ❌ Cannot delete Crossplane Providers
- ❌ Cannot delete Tenant CRs (unless force flag)
- ✅ Must provide `confirm` field matching resource name
- ✅ Only `admin` role can delete

### Concurrent Operation Protection
- ArgoCD: Check if operation already in progress
- Crossplane: Use optimistic locking (resourceVersion)

## 📊 Testing Verification

### Manual Testing Checklist
```bash
# 1. Test ArgoCD sync
curl -X POST http://localhost:9080/api/v1/actions/argo/sync \
  -H "X-Dev-Role: admin" -H "Content-Type: application/json" \
  -d '{"name": "alpha-ml-platform", "namespace": "argocd"}'

# Verify
kubectl get application alpha-ml-platform -n argocd -o yaml | grep operation

# 2. Test Crossplane pause
curl -X POST http://localhost:9080/api/v1/actions/crossplane/pause \
  -H "X-Dev-Role: infra" -H "Content-Type: application/json" \
  -d '{"group":"iam.aws.upbound.io","version":"v1beta1","kind":"Role","name":"tenant-alpha-lambda-role"}'

# Verify
kubectl get role.iam.aws.upbound.io tenant-alpha-lambda-role -o yaml | grep "crossplane.io/paused"

# 3. Test authorization (should fail)
curl -X DELETE http://localhost:9080/api/v1/actions/resources \
  -H "X-Dev-Role: readonly" -H "Content-Type: application/json" \
  -d '{"kind":"Deployment","name":"test"}'

# Expected: {"error": "Forbidden"}
```

## 📦 Files to Create

```
internal/api/middleware/
├── authz.go          # Capability-based authorization
├── authz_test.go     # Authorization tests
├── audit.go          # Audit logging
└── audit_test.go     # Audit tests

internal/api/handlers/
├── actions.go        # All action handlers
└── actions_test.go   # Action handler tests

test/e2e/
└── actions_test.go   # Integration tests

PHASE3_SUMMARY.md     # Completion documentation
PHASE3_PLAN.md        # This plan (already created)
PHASE3_CHECKLIST.md   # Task checklist (already created)
```

## ⏱️ Time Breakdown

| Day | Tasks | Hours |
|-----|-------|-------|
| 1 | Authorization + Audit | 4 |
| 2 | ArgoCD Actions | 3 |
| 2 | Crossplane Actions (Part 1) | 2 |
| 3 | Crossplane Actions (Part 2) + Delete | 3 |
| 3 | API Integration | 1 |
| 3 | Testing | 3 |
| 4 | More Testing + Documentation | 4 |

**Total**: ~20 hours (2.5 days)

## 🚀 Getting Started

1. **Create feature branch**:
   ```bash
   git checkout -b feature/phase3-actions
   ```

2. **Start with Task 1** - Authorization middleware:
   ```bash
   touch internal/api/middleware/authz.go
   touch internal/api/middleware/authz_test.go
   ```

3. **Follow the checklist** in `PHASE3_CHECKLIST.md`

4. **Test frequently**:
   ```bash
   go test ./internal/api/middleware/... -v
   ```

## ✅ Success Criteria

- [ ] All 6 action types working
- [ ] Authorization blocks unauthorized users
- [ ] Audit logs record all actions
- [ ] Safeguards prevent dangerous operations
- [ ] Unit tests >80% coverage
- [ ] E2E tests pass
- [ ] Documentation complete

## 🎉 What's Next After Phase 3?

**Phase 4: IAM Drift Module**
- Compare Crossplane IAM specs with AWS reality
- Detect privilege drift (extra/missing permissions)
- Visualize drift in frontend
- Use Phase 3 actions for remediation (pause → fix → unpause)

---

## 📚 Key Resources

- **Implementation Plan**: `PHASE3_PLAN.md` (detailed guide)
- **Task Checklist**: `PHASE3_CHECKLIST.md` (step-by-step)
- **Phase 2 Summary**: `PHASE2_SUMMARY.md` (what's already done)
- **Full Plan**: `IMPLEMENTATION_PLAN.md` (all phases)

---

**Ready to implement!** 🎯

Start with authorization middleware and work your way through the checklist.

