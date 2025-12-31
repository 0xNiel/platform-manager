# Phase 3 Implementation Checklist

## Pre-requisites ✅
- [x] Phase 0 complete (dev environment)
- [x] Phase 1 complete (backend skeleton, auth middleware)
- [x] Phase 2 complete (resource scanning, drill-down)
- [x] ArgoCD installed and running
- [x] Crossplane with AWS provider installed
- [x] Seed tenant resources deployed

## Task Breakdown

### 1. Authorization Infrastructure (2 hours)
- [ ] Create `internal/api/middleware/authz.go`
  - [ ] Define `Capability` type and constants
  - [ ] Define `roleCapabilities` mapping
  - [ ] Implement `RequireCapability()` middleware
  - [ ] Implement `hasCapability()` helper
- [ ] Create `internal/api/middleware/authz_test.go`
  - [ ] Test admin has all capabilities
  - [ ] Test infra role capabilities
  - [ ] Test ml role capabilities
  - [ ] Test readonly has no capabilities
- [ ] Run tests: `go test ./internal/api/middleware/...`

### 2. Audit Logging (2 hours)
- [ ] Create `internal/api/middleware/audit.go`
  - [ ] Define `AuditLog` struct
  - [ ] Define `AuditLogger` interface
  - [ ] Implement `DefaultAuditLogger`
  - [ ] Implement `LogAction()` method
  - [ ] Implement `AuditAction()` middleware wrapper
- [ ] Create `internal/api/middleware/audit_test.go`
  - [ ] Test audit log creation
  - [ ] Test audit log fields
  - [ ] Test error logging
- [ ] Run tests: `go test ./internal/api/middleware/...`

### 3. Actions Handler Skeleton (1 hour)
- [ ] Create `internal/api/handlers/actions.go`
  - [ ] Define `ActionsHandler` struct
  - [ ] Implement `NewActionsHandler()` constructor
  - [ ] Define request/response types
  - [ ] Add error response helper functions

### 4. ArgoCD Sync Action (2 hours)
- [ ] Implement `SyncArgoApp()` handler
  - [ ] Parse request body
  - [ ] Validate ArgoCD Application exists
  - [ ] Check if already syncing
  - [ ] Set `operation.sync` in Application spec
  - [ ] Update Application CR
  - [ ] Log audit event
  - [ ] Return operation status
- [ ] Manual test:
  ```bash
  curl -X POST http://localhost:9080/api/v1/actions/argo/sync \
    -H "X-Dev-Role: admin" -H "Content-Type: application/json" \
    -d '{"name": "alpha-ml-platform", "namespace": "argocd"}'
  ```
- [ ] Verify: `kubectl get application alpha-ml-platform -n argocd -o yaml | grep operation`

### 5. ArgoCD Refresh Action (1 hour)
- [ ] Implement `RefreshArgoApp()` handler
  - [ ] Parse request body
  - [ ] Validate ArgoCD Application exists
  - [ ] Add refresh annotation
  - [ ] Update Application CR
  - [ ] Log audit event
  - [ ] Return refresh status
- [ ] Manual test:
  ```bash
  curl -X POST http://localhost:9080/api/v1/actions/argo/refresh \
    -H "X-Dev-Role: admin" -H "Content-Type: application/json" \
    -d '{"name": "alpha-ml-platform", "namespace": "argocd"}'
  ```
- [ ] Verify: `kubectl get application alpha-ml-platform -n argocd -o jsonpath='{.metadata.annotations}'`

### 6. Crossplane Pause Action (2 hours)
- [ ] Implement `PauseCrossplaneResource()` handler
  - [ ] Parse request body (GVK + name)
  - [ ] Fetch resource using dynamic client
  - [ ] Add `crossplane.io/paused: "true"` annotation
  - [ ] Update resource
  - [ ] Log audit event
  - [ ] Return paused status
- [ ] Manual test:
  ```bash
  curl -X POST http://localhost:9080/api/v1/actions/crossplane/pause \
    -H "X-Dev-Role: infra" -H "Content-Type: application/json" \
    -d '{
      "group": "iam.aws.upbound.io",
      "version": "v1beta1",
      "kind": "Role",
      "name": "tenant-alpha-lambda-role"
    }'
  ```
- [ ] Verify: `kubectl get role.iam.aws.upbound.io tenant-alpha-lambda-role -o yaml | grep crossplane.io/paused`

### 7. Crossplane Unpause Action (1 hour)
- [ ] Implement `UnpauseCrossplaneResource()` handler
  - [ ] Parse request body
  - [ ] Fetch resource
  - [ ] Remove `crossplane.io/paused` annotation
  - [ ] Update resource
  - [ ] Log audit event
  - [ ] Return unpaused status
- [ ] Manual test:
  ```bash
  curl -X POST http://localhost:9080/api/v1/actions/crossplane/unpause \
    -H "X-Dev-Role: infra" -H "Content-Type: application/json" \
    -d '{
      "group": "iam.aws.upbound.io",
      "version": "v1beta1",
      "kind": "Role",
      "name": "tenant-alpha-lambda-role"
    }'
  ```

### 8. Crossplane Force Reconcile (1 hour)
- [ ] Implement `ReconcileCrossplaneResource()` handler
  - [ ] Parse request body
  - [ ] Fetch resource
  - [ ] Add/update `crossplane.io/reconcile` annotation with timestamp
  - [ ] Update resource
  - [ ] Log audit event
  - [ ] Return reconcile request status
- [ ] Manual test:
  ```bash
  curl -X POST http://localhost:9080/api/v1/actions/crossplane/reconcile \
    -H "X-Dev-Role: infra" -H "Content-Type: application/json" \
    -d '{
      "group": "iam.aws.upbound.io",
      "version": "v1beta1",
      "kind": "Role",
      "name": "tenant-alpha-lambda-role"
    }'
  ```

### 9. Resource Delete with Safeguards (3 hours)
- [ ] Implement `DeleteResource()` handler
  - [ ] Parse request body
  - [ ] Validate confirmation matches resource name
  - [ ] Fetch resource
  - [ ] Check safeguards:
    - [ ] Not in system namespaces
    - [ ] Not a Crossplane Provider
    - [ ] Not a Tenant CR (unless force flag)
  - [ ] Delete resource
  - [ ] Log audit event
  - [ ] Return deletion status
- [ ] Implement `canDelete()` safeguard checker
- [ ] Manual test (success):
  ```bash
  # Create test deployment first
  kubectl create deployment test-delete --image=nginx -n tenant-alpha
  
  # Delete via API
  curl -X DELETE http://localhost:9080/api/v1/actions/resources \
    -H "X-Dev-Role: admin" -H "Content-Type: application/json" \
    -d '{
      "group": "apps",
      "version": "v1",
      "kind": "Deployment",
      "namespace": "tenant-alpha",
      "name": "test-delete",
      "confirm": "test-delete"
    }'
  ```
- [ ] Manual test (safeguard - should fail):
  ```bash
  curl -X DELETE http://localhost:9080/api/v1/actions/resources \
    -H "X-Dev-Role: admin" -H "Content-Type: application/json" \
    -d '{
      "group": "v1",
      "kind": "Service",
      "namespace": "kube-system",
      "name": "kube-dns",
      "confirm": "kube-dns"
    }'
  # Expected: 403 Forbidden
  ```

### 10. API Router Integration (1 hour)
- [ ] Update `internal/api/server.go`
  - [ ] Create `auditLogger` instance
  - [ ] Create `actionsHandler` instance
  - [ ] Add `/api/v1/actions/argo/*` routes with authz
  - [ ] Add `/api/v1/actions/crossplane/*` routes with authz
  - [ ] Add `/api/v1/actions/resources` DELETE route with authz
- [ ] Rebuild and restart manager:
  ```bash
  make build
  kubectl delete pod -n platform-manager-system -l control-plane=controller-manager
  ```
- [ ] Check manager logs: `kubectl logs -n platform-manager-system -l control-plane=controller-manager -f`

### 11. Unit Tests (4 hours)
- [ ] Create `internal/api/handlers/actions_test.go`
  - [ ] Test `SyncArgoApp` success
  - [ ] Test `SyncArgoApp` app not found
  - [ ] Test `SyncArgoApp` concurrent sync
  - [ ] Test `RefreshArgoApp` success
  - [ ] Test `PauseCrossplaneResource` success
  - [ ] Test `UnpauseCrossplaneResource` success
  - [ ] Test `ReconcileCrossplaneResource` success
  - [ ] Test `DeleteResource` success
  - [ ] Test `DeleteResource` safeguard violations
  - [ ] Test `DeleteResource` missing confirmation
- [ ] Run tests: `go test ./internal/api/handlers/... -v`
- [ ] Check coverage: `go test ./internal/api/handlers/... -coverprofile=cover.out`
- [ ] Target: >80% coverage

### 12. Integration Tests (2 hours)
- [ ] Create `test/e2e/actions_test.go`
  - [ ] Test sync ArgoCD application in live cluster
  - [ ] Test refresh ArgoCD application
  - [ ] Test pause Crossplane IAM role
  - [ ] Test unpause Crossplane IAM role
  - [ ] Test reconcile Crossplane IAM role
  - [ ] Test delete deployment
  - [ ] Test authorization (readonly user should fail)
- [ ] Run e2e tests:
  ```bash
  make test-e2e
  ```

### 13. Authorization Tests (1 hour)
- [ ] Test admin role (should pass all):
  ```bash
  # Test each action with X-Dev-Role: admin
  ```
- [ ] Test infra role (should pass most):
  ```bash
  # Test argo sync - should pass
  # Test crossplane pause - should pass
  # Test delete - should FAIL
  ```
- [ ] Test ml role (limited access):
  ```bash
  # Test argo refresh - should pass
  # Test argo sync - should FAIL
  # Test crossplane pause - should FAIL
  ```
- [ ] Test readonly role (should fail all):
  ```bash
  # Test any action - should FAIL with 403
  ```

### 14. Documentation (2 hours)
- [ ] Create `PHASE3_SUMMARY.md`
  - [ ] Document all action endpoints
  - [ ] Add curl examples for each action
  - [ ] Document authorization model
  - [ ] Add troubleshooting section
  - [ ] Add frontend integration guide
- [ ] Update `README.md` with Phase 3 completion
- [ ] Add action examples to `IMPLEMENTATION_PLAN.md`

### 15. Final Verification (1 hour)
- [ ] Run full test suite: `make test`
- [ ] Manual testing of all 6 action types
- [ ] Check audit logs are written
- [ ] Verify authorization blocks unauthorized users
- [ ] Verify safeguards prevent dangerous operations
- [ ] Check API documentation is complete

## Definition of Done

Phase 3 is complete when ALL of these are true:

- [ ] All 6 action types implemented (sync, refresh, pause, unpause, reconcile, delete)
- [ ] Authorization middleware with capabilities works
- [ ] Audit logging records all actions
- [ ] Unit tests pass with >80% coverage
- [ ] E2E tests pass in live environment
- [ ] Manual testing successful for all actions
- [ ] Documentation complete with examples
- [ ] Code committed to git
- [ ] `PHASE3_SUMMARY.md` created

## Time Estimate

Total: **24 hours** (3 days at 8 hours/day)

| Category | Hours |
|----------|-------|
| Authorization & Audit | 4 |
| Action Handlers | 10 |
| API Integration | 1 |
| Testing | 7 |
| Documentation | 2 |

## Git Commits Strategy

Make frequent, atomic commits:

```bash
# After task 1-2
git add internal/api/middleware/
git commit -m "feat(phase3): add authorization and audit middleware"

# After task 3-5
git add internal/api/handlers/actions.go
git commit -m "feat(phase3): implement ArgoCD sync and refresh actions"

# After task 6-8
git commit -m "feat(phase3): implement Crossplane pause, unpause, and reconcile actions"

# After task 9
git commit -m "feat(phase3): implement resource delete with safeguards"

# After task 10
git add internal/api/server.go
git commit -m "feat(phase3): integrate action handlers into API router"

# After task 11-12
git add internal/api/handlers/actions_test.go test/e2e/actions_test.go
git commit -m "test(phase3): add comprehensive action handler tests"

# After task 14
git add PHASE3_SUMMARY.md README.md
git commit -m "docs(phase3): add Phase 3 documentation and examples"
```

## Quick Start

```bash
# 1. Create feature branch
git checkout -b feature/phase3-actions

# 2. Start with authorization middleware
touch internal/api/middleware/authz.go
touch internal/api/middleware/authz_test.go

# 3. Follow checklist from top to bottom

# 4. Test frequently
go test ./internal/api/middleware/... -v
go test ./internal/api/handlers/... -v
```

## Troubleshooting

### Issue: "Application not found"
**Solution**: Check ArgoCD namespace and application name:
```bash
kubectl get applications -n argocd
```

### Issue: "Resource not found" for Crossplane
**Solution**: Check resource exists and GVK is correct:
```bash
kubectl get roles.iam.aws.upbound.io
kubectl api-resources | grep iam
```

### Issue: "Forbidden" error
**Solution**: Check X-Dev-Role header is set:
```bash
curl -H "X-Dev-Role: admin" ...
```

### Issue: Audit logs not showing
**Solution**: Check manager logs:
```bash
kubectl logs -n platform-manager-system -l control-plane=controller-manager | grep audit
```

---

Ready to start? Begin with **Task 1: Authorization Infrastructure** ✨

