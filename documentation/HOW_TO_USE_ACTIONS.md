# 🚀 How to Use Phase 3 Actions - Quick Guide

## Where to Find Action Buttons

### ❌ Dashboard View
**No action buttons here** - The dashboard is for viewing overall health and metrics only.

### ✅ Resources View
**This is where all the action happens!** Click **"Resources"** in the navigation bar.

---

## Step-by-Step Guide

### 1. Navigate to Resources
```
Top Navigation Bar → Click "Resources"
```

### 2. You'll See Resource Cards
Each card shows:
- **Resource Name** (e.g., "alpha-ml-platform")
- **Kind** (e.g., "Application", "Role", "Policy")
- **Category Badge** (ArgoCD, Crossplane, Kubernetes)
- **Status Badge** (Ready, Failed, Waiting, Paused)
- **Namespace** and **Tenant** info
- **Action Buttons** (at the bottom of each card)

### 3. Action Buttons Available

#### For ArgoCD Applications:
- 🔄 **Refresh** (Green) - Refresh app status from Git
- 🔁 **Sync** (Blue) - Deploy latest from Git repo

#### For Crossplane Resources:
- ⏸ **Pause** (Orange) - Temporarily pause reconciliation
- ▶ **Unpause** (Green) - Resume reconciliation
- ⚡ **Reconcile** (Blue) - Force immediate reconciliation

#### For Any Resource (Admin only):
- 🗑 **Delete** (Red) - Delete resource with confirmation

---

## Using the Filters

### Search Box
Type resource name, kind, or namespace to filter

### State Filter
- All States
- Ready
- Failed
- Waiting
- Paused

### Kind Filter
- All Kinds
- ArgoCD Applications
- IAM Roles
- IAM Policies
- Deployments

---

## Testing Actions

### Test 1: Refresh an ArgoCD App
1. Find an ArgoCD Application card (red "ARGOCD" badge)
2. Click the green **"Refresh"** button
3. Watch for green success toast in top-right corner
4. Resource card updates automatically

### Test 2: Sync an ArgoCD App
1. Find an ArgoCD Application
2. Click the blue **"Sync"** button
3. Confirm in the dialog
4. See success toast
5. Check ArgoCD UI to verify sync started

### Test 3: Pause a Crossplane Resource
1. Find a Crossplane resource (blue "CROSSPLANE" badge)
2. Click the orange **"Pause"** button
3. Status badge changes to "PAUSED" (gray)
4. Resource stops reconciling

### Test 4: Unpause
1. Find a paused Crossplane resource
2. Click the green **"Unpause"** button
3. Status returns to previous state
4. Reconciliation resumes

### Test 5: Force Reconcile
1. Find a Crossplane resource
2. Click the blue **"Reconcile"** button
3. Forces immediate reconciliation check
4. Useful for stuck resources

### Test 6: Delete (Admin Only)
1. Find any resource
2. Click the red **"Delete"** button
3. Type exact resource name to confirm
4. Resource is removed

---

## Toast Notifications

All actions show feedback in the **top-right corner**:

- ✅ **Green Toast** = Success
- ❌ **Red Toast** = Error (with error message)
- ⚠️ **Orange Toast** = Warning
- ℹ️ **Blue Toast** = Info

Toasts:
- Auto-dismiss after 5 seconds
- Click to dismiss immediately
- Stack vertically if multiple actions

---

## Role-Based Access

Different roles see different buttons:

| Role | Refresh | Sync | Pause/Unpause | Reconcile | Delete |
|------|---------|------|---------------|-----------|--------|
| **Admin** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Infra** | ✅ | ✅ | ✅ | ✅ | ❌ |
| **ML** | ✅ | ❌ | ❌ | ❌ | ❌ |
| **ReadOnly** | ❌ | ❌ | ❌ | ❌ | ❌ |

*Currently testing with `X-Dev-Role` header, so you'll see all buttons*

---

## Troubleshooting

### "Failed to load resources" Error
**Fixed!** The API response structure was corrected. Refresh the page.

### Action buttons not showing
- Make sure you're on the **Resources** view (not Dashboard)
- Check that resources are loaded (no error message)
- Try refreshing the page

### Action fails with "Forbidden"
- Check your role (currently should work with X-Dev-Role: admin)
- Verify the action is allowed for your role

### Toast doesn't appear
- Check browser console for errors
- Verify ToastContainer is in App.vue
- Action might have failed before reaching success

---

## Viewing Metrics

While testing actions, you can see metrics:

```bash
# View all action metrics
curl http://localhost:8443/metrics | grep platform_manager

# See action counts
curl -s http://localhost:8443/metrics | grep actions_total

# See auth denials
curl -s http://localhost:8443/metrics | grep denials_total
```

---

## Expected Behavior

### First Time on Resources Page
1. Shows "Loading resources..."
2. Fetches from `/api/v1/resources`
3. Displays resource cards with action buttons
4. No errors in console

### After Clicking an Action
1. Button shows loading spinner
2. API call is made
3. Success/error toast appears
4. Resources automatically refresh
5. Updated state is visible

### Visual Feedback
- Button shows spinner while loading
- Toast appears top-right
- Card updates with new status
- Smooth animations throughout

---

## Quick Commands to Populate Resources

If you see "No resources found":

```bash
# Apply sample tenants (creates ArgoCD apps)
kubectl apply -f hack/seed-tenants/tenant-alpha.yaml
kubectl apply -f hack/seed-tenants/tenant-beta.yaml
kubectl apply -f hack/seed-tenants/argo-apps.yaml

# Apply IAM resources (Crossplane)
kubectl apply -f hack/seed-tenants/iam-resources.yaml

# Wait for resources to be scanned
sleep 10

# Refresh the Resources page
```

---

## Summary

✅ **Actions are on Resources page** (not Dashboard)  
✅ **Error is fixed** (API response structure)  
✅ **6 action types available** (per resource type)  
✅ **Toast notifications work** (success/error feedback)  
✅ **Metrics are tracked** (view via curl)  

**Try it now:** Go to Resources → Click a Refresh button → See the green toast! 🎉

---

*Phase 3 is complete and working. Enjoy your new action buttons!*

