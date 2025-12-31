# Phase 3 Enhancement Plan - Frontend + Metrics

## Goal
Complete Phase 3 by adding frontend integration and Prometheus metrics for production readiness.

---

## Part 1: Frontend Integration (Vue 3)

### 1.1 API Client Extensions
**File**: `web/src/api/client.ts`

Add action API methods:
```typescript
export const actionsApi = {
  // ArgoCD Actions
  syncArgoApp(name: string, namespace: string, options: { prune?: boolean; dryRun?: boolean }): Promise<ActionResponse>
  refreshArgoApp(name: string, namespace: string): Promise<ActionResponse>
  
  // Crossplane Actions
  pauseCrossplaneResource(resource: CrossplaneResource): Promise<ActionResponse>
  unpauseCrossplaneResource(resource: CrossplaneResource): Promise<ActionResponse>
  reconcileCrossplaneResource(resource: CrossplaneResource): Promise<ActionResponse>
  
  // Delete Action
  deleteResource(resource: ResourceIdentifier, confirm: string): Promise<ActionResponse>
}
```

### 1.2 New Components
1. **ActionButton.vue** - Reusable action button with loading state
2. **ConfirmDialog.vue** - Confirmation dialog for destructive actions
3. **ActionMenu.vue** - Dropdown menu for resource actions

### 1.3 Enhanced Views
1. **TenantDetailView.vue** - Add action buttons to resource cards
2. **ResourcesView.vue** - Add bulk actions
3. **DashboardView.vue** - Show action metrics

### 1.4 Features
- Action buttons based on user role
- Loading states during API calls
- Success/error toasts
- Confirmation dialogs for destructive actions
- Real-time resource refresh after actions

---

## Part 2: Prometheus Metrics

### 2.1 Metrics to Track
```
# Counter metrics
platform_manager_actions_total{action="argo_sync", status="success|failure", user_role="admin|infra|ml"}
platform_manager_actions_total{action="argo_refresh", status="success|failure", user_role="admin|infra|ml"}
platform_manager_actions_total{action="crossplane_pause", status="success|failure", user_role="admin|infra"}
platform_manager_actions_total{action="crossplane_unpause", status="success|failure", user_role="admin|infra"}
platform_manager_actions_total{action="crossplane_reconcile", status="success|failure", user_role="admin|infra"}
platform_manager_actions_total{action="resource_delete", status="success|failure", user_role="admin"}

# Authorization denials
platform_manager_auth_denials_total{action="argo_sync", user_role="ml|readonly"}

# Histogram for action duration
platform_manager_action_duration_seconds{action="argo_sync"}
```

### 2.2 Implementation
**File**: `internal/metrics/actions.go`

Create metrics collector:
```go
type ActionMetrics struct {
    actionsTotal *prometheus.CounterVec
    authDenials *prometheus.CounterVec
    actionDuration *prometheus.HistogramVec
}

func (m *ActionMetrics) RecordAction(action string, userRole string, success bool, duration time.Duration)
func (m *ActionMetrics) RecordAuthDenial(action string, userRole string)
```

### 2.3 Middleware Integration
Add metrics recording to:
- Action handlers (success/failure)
- Authorization middleware (denials)
- Audit logger (wrap with metrics)

---

## Implementation Timeline

### Phase A: Frontend (4-5 hours)
1. **Hour 1**: API client extensions
2. **Hour 2**: Reusable action components
3. **Hour 2-3**: Integrate into existing views
4. **Hour 4**: Testing and polish
5. **Hour 5**: Documentation

### Phase B: Metrics (2-3 hours)
1. **Hour 1**: Metrics infrastructure
2. **Hour 2**: Integration with handlers
3. **Hour 3**: Testing and Grafana dashboard

**Total: 6-8 hours**

---

## Testing Strategy

### Frontend Testing
1. Manual testing with all 4 roles
2. Test each action type
3. Verify loading states
4. Verify error handling
5. Verify confirmation dialogs

### Metrics Testing
1. Execute actions and verify metrics
2. Check metric labels
3. Verify histogram buckets
4. Test authorization denials
5. Create Grafana dashboard

---

## Deliverables

### Frontend
- ✅ Action buttons on all resource views
- ✅ Role-based UI controls
- ✅ Confirmation dialogs
- ✅ Loading states and feedback
- ✅ Error handling with toasts

### Metrics
- ✅ Prometheus metrics for all actions
- ✅ Authorization denial tracking
- ✅ Action duration histograms
- ✅ Grafana dashboard JSON
- ✅ Metrics documentation

---

## Success Criteria

1. ✅ All 6 action types accessible from UI
2. ✅ Actions work for authorized users
3. ✅ Actions blocked for unauthorized users
4. ✅ Metrics appear in Prometheus
5. ✅ Grafana dashboard displays action stats
6. ✅ No console errors
7. ✅ Responsive UI
8. ✅ Audit logs still working

Let's implement this!

