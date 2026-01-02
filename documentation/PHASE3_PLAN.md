# Phase 3: Actions (Mini Argo + Crossplane Controls) - Implementation Plan

## Executive Summary

**Goal**: Enable mutating operations (sync, pause, delete) with proper authorization and audit logging.

**Current Status**: 
- ✅ Phase 0-2 completed (Backend skeleton, health aggregation, resource scanning)
- ✅ Auth middleware exists with role extraction (`middleware.ExtractUser`, `middleware.RequireRole`)
- ✅ ArgoCD and Crossplane watchers implemented
- ✅ Resource scanner tracking all resources

**What's Missing for Phase 3**:
1. Authorization middleware with capability-based access control
2. Action handlers for ArgoCD operations (sync, refresh)
3. Action handlers for Crossplane operations (pause, unpause, reconcile)
4. Resource deletion handler with safeguards
5. Audit logging for all actions
6. Integration with existing API server

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Request Flow                         │
├─────────────────────────────────────────────────────────────────┤
│  1. Request → ExtractUser middleware (get user context)         │
│  2. Request → RequireCapability middleware (check permission)   │
│  3. Request → Action Handler (validate + execute)               │
│  4. Action Handler → K8s API (patch/update resource)            │
│  5. Action Handler → Audit Log (record action)                  │
│  6. Response → JSON result                                      │
└─────────────────────────────────────────────────────────────────┘
```

### Authorization Model

**Roles** (already implemented in `middleware/auth.go`):
- `admin` - Full access
- `infra` - Infrastructure operations
- `ml` - Limited ML team access
- `readonly` - Read-only access

**Capabilities** (new):
- `argo:sync` - Sync ArgoCD applications
- `argo:refresh` - Refresh ArgoCD applications
- `crossplane:pause` - Pause Crossplane resources
- `crossplane:reconcile` - Force reconciliation
- `resource:delete` - Delete resources

**Role-to-Capability Mapping**:
```
admin     → ALL capabilities
infra     → argo:sync, argo:refresh, crossplane:pause, crossplane:reconcile
ml        → argo:refresh
readonly  → NONE (read-only)
```

---

## Implementation Tasks

### Task 1: Authorization Middleware Enhancement ⏳

**File**: `internal/api/middleware/authz.go` (NEW)

**What to build**:
```go
type Capability string

const (
    CapSyncArgo          Capability = "argo:sync"
    CapRefreshArgo       Capability = "argo:refresh"
    CapPauseCrossplane   Capability = "crossplane:pause"
    CapReconcileCrossplane Capability = "crossplane:reconcile"
    CapDeleteResource    Capability = "resource:delete"
)

var roleCapabilities = map[Role][]Capability{
    RoleAdmin: {
        CapSyncArgo,
        CapRefreshArgo,
        CapPauseCrossplane,
        CapReconcileCrossplane,
        CapDeleteResource,
    },
    RoleInfra: {
        CapSyncArgo,
        CapRefreshArgo,
        CapPauseCrossplane,
        CapReconcileCrossplane,
    },
    RoleML: {
        CapRefreshArgo,
    },
    RoleReadOnly: {},
}

// RequireCapability middleware checks if user has required capability
func RequireCapability(cap Capability) func(http.Handler) http.Handler
```

**Why**: Granular permission control beyond simple role checks.

**Testing**: Unit tests for each role/capability combination.

**Estimated time**: 2 hours

---

### Task 2: Audit Logging Infrastructure ⏳

**File**: `internal/api/middleware/audit.go` (NEW)

**What to build**:
```go
type AuditLog struct {
    Timestamp   time.Time
    User        string
    Action      string
    Resource    string
    Success     bool
    Error       string
    RequestID   string
}

// AuditLogger interface for recording actions
type AuditLogger interface {
    LogAction(ctx context.Context, action string, resource string, success bool, err error)
}

// DefaultAuditLogger logs to structured log
type DefaultAuditLogger struct {
    logger logr.Logger
}

// AuditAction middleware wrapper to audit all actions
func AuditAction(logger AuditLogger) func(http.Handler) http.Handler
```

**Why**: Track all mutating operations for security and compliance.

**Testing**: Verify audit logs are written for all actions.

**Estimated time**: 2 hours

---

### Task 3: ArgoCD Actions Handler ⏳

**File**: `internal/api/handlers/actions.go` (NEW)

**What to build**:

```go
type ActionsHandler struct {
    client       client.Client
    auditLogger  middleware.AuditLogger
}

// ArgoCD Sync
// POST /api/v1/actions/argo/sync
// Body: { "name": "app-name", "namespace": "argocd", "prune": false, "dryRun": false }
func (h *ActionsHandler) SyncArgoApp(w http.ResponseWriter, r *http.Request)

// ArgoCD Refresh
// POST /api/v1/actions/argo/refresh
// Body: { "name": "app-name", "namespace": "argocd" }
func (h *ActionsHandler) RefreshArgoApp(w http.ResponseWriter, r *http.Request)
```

**Implementation details**:

1. **Sync Operation**:
   - Fetch ArgoCD Application CR
   - Add/update `operation` field in Application spec
   - Set `syncStrategy`, `prune`, `dryRun` options
   - Return operation status

2. **Refresh Operation**:
   - Fetch ArgoCD Application CR
   - Add annotation: `argocd.argoproj.io/refresh: "normal"`
   - Return refresh status

**ArgoCD Application CRD Reference**:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
spec:
  operation:
    sync:
      syncStrategy:
        hook: {}
      prune: false
      dryRun: false
```

**Error handling**:
- Application not found → 404
- User lacks permission → 403
- Application already syncing → 409 Conflict
- K8s API error → 500

**Testing**: 
- Mock Application CR
- Test sync with different options
- Test concurrent sync attempts

**Estimated time**: 4 hours

---

### Task 4: Crossplane Actions Handler ⏳

**File**: `internal/api/handlers/actions.go` (extend)

**What to build**:

```go
// Crossplane Pause
// POST /api/v1/actions/crossplane/pause
// Body: { "group": "iam.aws.upbound.io", "version": "v1beta1", "kind": "Role", 
//         "namespace": "", "name": "tenant-alpha-lambda-role" }
func (h *ActionsHandler) PauseCrossplaneResource(w http.ResponseWriter, r *http.Request)

// Crossplane Unpause
// POST /api/v1/actions/crossplane/unpause
func (h *ActionsHandler) UnpauseCrossplaneResource(w http.ResponseWriter, r *http.Request)

// Crossplane Force Reconcile
// POST /api/v1/actions/crossplane/reconcile
func (h *ActionsHandler) ReconcileCrossplaneResource(w http.ResponseWriter, r *http.Request)
```

**Implementation details**:

1. **Pause Operation**:
   - Fetch resource by GVK
   - Add annotation: `crossplane.io/paused: "true"`
   - Update resource
   - Return updated state

2. **Unpause Operation**:
   - Fetch resource by GVK
   - Remove annotation: `crossplane.io/paused`
   - Update resource
   - Return updated state

3. **Force Reconcile**:
   - Fetch resource by GVK
   - Update annotation: `crossplane.io/reconcile: "now-<timestamp>"`
   - This triggers immediate reconciliation
   - Return reconcile request status

**Crossplane Pause Reference**:
```yaml
metadata:
  annotations:
    crossplane.io/paused: "true"  # Pauses reconciliation
```

**Error handling**:
- Resource not found → 404
- Not a Crossplane resource → 400 Bad Request
- User lacks permission → 403
- Resource already in desired state → 200 (idempotent)
- K8s API error → 500

**Testing**:
- Mock IAM Role CR
- Test pause/unpause/reconcile operations
- Verify annotations are set correctly

**Estimated time**: 4 hours

---

### Task 5: Resource Delete Handler with Safeguards ⏳

**File**: `internal/api/handlers/actions.go` (extend)

**What to build**:

```go
// Delete Resource
// DELETE /api/v1/actions/resources
// Body: { "group": "apps", "version": "v1", "kind": "Deployment",
//         "namespace": "tenant-alpha", "name": "ml-training-service" }
func (h *ActionsHandler) DeleteResource(w http.ResponseWriter, r *http.Request)
```

**Implementation details**:

1. **Safeguards**:
   - Require `admin` role only
   - Deny deletion of:
     - Resources in `kube-system`, `kube-public`, `default` namespaces
     - Tenant CRs (require explicit flag)
     - Crossplane Providers
   - Require confirmation parameter: `confirm=<resource-name>`

2. **Deletion Logic**:
   - Fetch resource by GVK
   - Validate safeguards
   - Check if resource has finalizers (warn user)
   - Delete resource
   - Return deletion status

**Request validation**:
```go
// Safeguard check
func (h *ActionsHandler) canDelete(obj client.Object) error {
    // Check namespace
    ns := obj.GetNamespace()
    if ns == "kube-system" || ns == "kube-public" || ns == "default" {
        return errors.New("cannot delete resources in system namespaces")
    }
    
    // Check resource type
    gvk := obj.GetObjectKind().GroupVersionKind()
    if gvk.Kind == "Provider" && gvk.Group == "pkg.crossplane.io" {
        return errors.New("cannot delete Crossplane providers")
    }
    
    return nil
}
```

**Error handling**:
- Resource not found → 404
- User lacks permission → 403
- Safeguard violation → 403 Forbidden
- Missing confirmation → 400 Bad Request
- K8s API error → 500

**Testing**:
- Test successful deletion
- Test safeguard violations
- Test missing confirmation

**Estimated time**: 3 hours

---

### Task 6: API Router Integration ⏳

**File**: `internal/api/server.go` (UPDATE)

**What to add**:

```go
// In setupRouter() function, add action routes:
r.Route("/api/v1", func(r chi.Router) {
    // ... existing routes ...
    
    // Create actions handler
    actionsHandler := handlers.NewActionsHandler(s.client, auditLogger)
    
    // Action endpoints
    r.Route("/actions", func(r chi.Router) {
        // ArgoCD actions
        r.Route("/argo", func(r chi.Router) {
            r.With(middleware.RequireCapability(middleware.CapSyncArgo)).
                Post("/sync", actionsHandler.SyncArgoApp)
            r.With(middleware.RequireCapability(middleware.CapRefreshArgo)).
                Post("/refresh", actionsHandler.RefreshArgoApp)
        })
        
        // Crossplane actions
        r.Route("/crossplane", func(r chi.Router) {
            r.With(middleware.RequireCapability(middleware.CapPauseCrossplane)).
                Post("/pause", actionsHandler.PauseCrossplaneResource)
            r.With(middleware.RequireCapability(middleware.CapPauseCrossplane)).
                Post("/unpause", actionsHandler.UnpauseCrossplaneResource)
            r.With(middleware.RequireCapability(middleware.CapReconcileCrossplane)).
                Post("/reconcile", actionsHandler.ReconcileCrossplaneResource)
        })
        
        // Delete action
        r.With(middleware.RequireCapability(middleware.CapDeleteResource)).
            Delete("/resources", actionsHandler.DeleteResource)
    })
})
```

**Estimated time**: 1 hour

---

### Task 7: Testing Suite ⏳

**Files**: 
- `internal/api/handlers/actions_test.go` (NEW)
- `internal/api/middleware/authz_test.go` (NEW)

**Unit tests**:
```go
// Test ArgoCD sync action
func TestSyncArgoApp(t *testing.T) {
    // Test successful sync
    // Test with prune=true
    // Test with dryRun=true
    // Test app not found
    // Test concurrent sync
}

// Test Crossplane pause action
func TestPauseCrossplaneResource(t *testing.T) {
    // Test successful pause
    // Test unpause
    // Test already paused (idempotent)
    // Test resource not found
}

// Test delete with safeguards
func TestDeleteResource(t *testing.T) {
    // Test successful deletion
    // Test safeguard: system namespace
    // Test safeguard: provider deletion
    // Test missing confirmation
}

// Test authorization
func TestRequireCapability(t *testing.T) {
    // Test admin has all capabilities
    // Test infra has subset
    // Test ml has limited access
    // Test readonly has no access
}
```

**Integration tests** (`test/e2e/actions_test.go`):
```go
// Test sync ArgoCD application in live cluster
// Test pause Crossplane IAM role in LocalStack
// Test delete deployment
```

**Estimated time**: 6 hours

---

### Task 8: Documentation and Examples ⏳

**File**: `PHASE3_SUMMARY.md` (NEW)

**What to document**:
1. All action endpoints with curl examples
2. Authorization model and capability matrix
3. Error codes and troubleshooting
4. Frontend integration guide

**Examples**:
```bash
# Sync ArgoCD application (admin or infra role)
curl -X POST http://localhost:9080/api/v1/actions/argo/sync \
  -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "alpha-ml-platform",
    "namespace": "argocd",
    "prune": false,
    "dryRun": false
  }'

# Pause Crossplane resource (admin or infra role)
curl -X POST http://localhost:9080/api/v1/actions/crossplane/pause \
  -H "X-Dev-Role: infra" \
  -H "Content-Type: application/json" \
  -d '{
    "group": "iam.aws.upbound.io",
    "version": "v1beta1",
    "kind": "Role",
    "name": "tenant-alpha-lambda-role"
  }'

# Delete resource (admin only)
curl -X DELETE http://localhost:9080/api/v1/actions/resources \
  -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{
    "group": "apps",
    "version": "v1",
    "kind": "Deployment",
    "namespace": "tenant-alpha",
    "name": "ml-training-service",
    "confirm": "ml-training-service"
  }'
```

**Estimated time**: 2 hours

---

## Implementation Order

### Week 1: Core Infrastructure
1. **Day 1-2**: Authorization middleware + Audit logging (Tasks 1-2)
   - Create `middleware/authz.go` with capability model
   - Create `middleware/audit.go` with audit logging
   - Write unit tests
   - ✅ Deliverable: Authorization and audit foundation

2. **Day 3-4**: ArgoCD Actions (Task 3)
   - Implement sync and refresh handlers
   - Add to router with authz
   - Write unit tests
   - Test with live ArgoCD
   - ✅ Deliverable: Working ArgoCD actions

3. **Day 5**: Crossplane Actions Part 1 (Task 4)
   - Implement pause/unpause handlers
   - Write unit tests
   - ✅ Deliverable: Pause/unpause operations

### Week 2: Completion and Testing
4. **Day 6**: Crossplane Actions Part 2 + Delete (Tasks 4-5)
   - Implement reconcile handler
   - Implement delete with safeguards
   - Write unit tests
   - ✅ Deliverable: All action handlers complete

5. **Day 7**: API Integration (Task 6)
   - Wire up all handlers to router
   - Test all endpoints manually
   - ✅ Deliverable: Complete API

6. **Day 8-9**: Integration Testing (Task 7)
   - Write e2e tests
   - Test in live environment
   - Fix any issues
   - ✅ Deliverable: Tested and verified

7. **Day 10**: Documentation (Task 8)
   - Write curl examples
   - Document authorization model
   - Create frontend integration guide
   - ✅ Deliverable: Complete Phase 3

---

## Testing Strategy

### Local Testing Commands

```bash
# 1. Ensure dev environment is running
make dev-up

# 2. Port forward API server
kubectl port-forward -n platform-manager-system deployment/platform-manager 9080:9080

# 3. Test ArgoCD sync (with dev header)
curl -X POST http://localhost:9080/api/v1/actions/argo/sync \
  -H "X-Dev-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{"name": "alpha-ml-platform", "namespace": "argocd"}'

# 4. Check ArgoCD application status
kubectl get application alpha-ml-platform -n argocd -o yaml | grep operation

# 5. Test Crossplane pause
curl -X POST http://localhost:9080/api/v1/actions/crossplane/pause \
  -H "X-Dev-Role: infra" \
  -H "Content-Type: application/json" \
  -d '{
    "group": "iam.aws.upbound.io",
    "version": "v1beta1",
    "kind": "Role",
    "name": "tenant-alpha-lambda-role"
  }'

# 6. Verify pause annotation
kubectl get role.iam.aws.upbound.io tenant-alpha-lambda-role -o yaml | grep crossplane.io/paused

# 7. Test authorization (should fail)
curl -X DELETE http://localhost:9080/api/v1/actions/resources \
  -H "X-Dev-Role: readonly" \
  -H "Content-Type: application/json" \
  -d '{"kind": "Deployment", "name": "test"}'
# Expected: 403 Forbidden
```

### Verification Checklist

- [ ] ArgoCD sync operation is triggered
- [ ] ArgoCD refresh annotation is added
- [ ] Crossplane pause annotation is set
- [ ] Crossplane unpause removes annotation
- [ ] Reconcile annotation triggers immediate reconciliation
- [ ] Delete removes resource
- [ ] Safeguards prevent system resource deletion
- [ ] Authorization blocks unauthorized users
- [ ] Audit logs record all actions
- [ ] Error messages are clear and actionable

---

## Frontend Integration Preview

### Action Buttons to Add

1. **Tenant Resources View** (for each resource):
   - "Sync" button (if ArgoCD Application)
   - "Refresh" button (if ArgoCD Application)
   - "Pause" button (if Crossplane resource)
   - "Reconcile" button (if Crossplane resource)
   - "Delete" button (admin only)

2. **Button State Logic**:
   ```typescript
   // Show sync button only for ArgoCD apps
   if (resource.spec.category === 'ArgoCD' && 
       resource.spec.kind === 'Application' &&
       userRole in ['admin', 'infra']) {
     showSyncButton = true;
   }
   
   // Show pause button only for Crossplane resources
   if (resource.spec.category === 'Crossplane' &&
       userRole in ['admin', 'infra']) {
     showPauseButton = resource.status.state !== 'Paused';
     showUnpauseButton = resource.status.state === 'Paused';
   }
   ```

3. **Confirmation Dialogs**:
   - Sync: "Are you sure you want to sync this application?"
   - Delete: "Type the resource name to confirm deletion"

4. **Action Feedback**:
   - Show loading spinner during action
   - Show success toast on completion
   - Show error toast with message on failure
   - Refresh resource list after action

---

## Success Criteria

Phase 3 is complete when:

- ✅ All 6 action types are implemented and working
- ✅ Authorization prevents unauthorized actions
- ✅ Audit logs record all operations
- ✅ Safeguards prevent dangerous deletions
- ✅ Unit tests achieve >80% coverage
- ✅ E2E tests verify live functionality
- ✅ Documentation includes curl examples
- ✅ Manual testing verifies all operations

---

## Risk Mitigation

### Risk 1: ArgoCD CRD Version Mismatch
**Mitigation**: Check ArgoCD version in cluster, use correct API version

### Risk 2: Crossplane Provider CRD Variations
**Mitigation**: Use dynamic client with GVK lookup, test with multiple provider types

### Risk 3: Concurrent Operations
**Mitigation**: Use optimistic locking (resourceVersion), handle conflicts gracefully

### Risk 4: Audit Log Volume
**Mitigation**: Start with structured logging, plan for external audit storage (Phase 5+)

---

## Dependencies

**External**:
- ArgoCD installed and running (✅ from Phase 0)
- Crossplane installed with providers (✅ from Phase 0)
- Kubernetes 1.29+ (✅ from Phase 0)

**Internal**:
- Auth middleware exists (✅ from Phase 1)
- Resource scanning tracks all resources (✅ from Phase 2)
- ArgoCD and Crossplane watchers exist (✅ from Phase 2)

---

## Next Phase Preview: Phase 4 - IAM Drift Module

After completing Phase 3, we'll implement:
- AWS IAM policy comparison logic
- Drift detection between Crossplane spec and AWS reality
- Drift visualization in frontend
- Remediation suggestions

Phase 3 actions will enable **manual remediation** of drift (pause drifted resource, fix, unpause).

---

## Questions Before Starting?

1. Should we implement rate limiting on action endpoints?
2. Should we add bulk operations (sync multiple apps)?
3. Should delete operations be soft-deletes (add deletion annotation)?
4. Should we implement action history/timeline per resource?

**Recommendation**: Keep it simple for Phase 3, add enhancements in later phases.

---

## Get Started

```bash
# 1. Verify Phase 2 is complete
kubectl get resourcesummaries
kubectl get tenanthealth

# 2. Create feature branch
git checkout -b feature/phase3-actions

# 3. Start with Task 1
# Create internal/api/middleware/authz.go
```

Ready to implement! 🚀

