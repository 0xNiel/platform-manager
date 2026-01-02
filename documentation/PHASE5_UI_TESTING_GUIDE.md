# Phase 5 UI Testing Guide

## 🎯 Purpose

This guide will help you test the Phase 5 Troubleshooting UI by creating real problems that trigger the rule engine, verifying the UI displays them correctly, and then cleaning everything up.

---

## 📋 Prerequisites

1. ✅ Backend manager running
2. ✅ Frontend dev server running
3. ✅ Kubernetes cluster accessible

---

## 🚀 Quick Start

### Option 1: Automated Test (Recommended)

```bash
# Start backend (Terminal 1)
./start-phase5.sh

# Start frontend (Terminal 2)
./start-frontend.sh

# Run UI test (Terminal 3)
./test-phase5-ui.sh
```

### Option 2: Manual Steps

Follow the detailed steps below for manual testing.

---

## 📝 Detailed Testing Steps

### Step 1: Start Services

#### Terminal 1: Backend
```bash
cd /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager

# Start the manager
./bin/manager \
  --api-bind-address=:9080 \
  --metrics-bind-address=:8443 \
  --rule-eval-interval=2m
```

**Expected output:**
```
INFO    Rule evaluator initialized    {"interval": "2m0s", "rules": 11}
INFO    Starting API server           {"addr": ":9080"}
```

#### Terminal 2: Frontend
```bash
cd /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager

# Start dev server
./start-frontend.sh
# OR
cd web && npm run serve
```

**Expected output:**
```
App running at:
- Local:   http://localhost:9083/
```

### Step 2: Verify Baseline

Open browser: **http://localhost:9083/troubleshooting**

**Expected State:**
- ✅ Summary cards show 0 findings
- ✅ "No Issues Found" message
- ✅ "All systems are operating normally"

### Step 3: Create Problems

Run the test script:
```bash
./test-phase5-ui.sh
```

Or manually create issues:

```bash
# 1. CrashLoop Pod
kubectl run crashloop-test \
  --image=busybox \
  --restart=Always \
  --namespace=tenant-alpha \
  --labels="platform.io/tenant=tenant-alpha" \
  --command -- sh -c "exit 1"

# 2. ImagePull Failure
kubectl run imagepull-test \
  --image=nonexistent:v999 \
  --namespace=tenant-alpha \
  --labels="platform.io/tenant=tenant-alpha"

# 3. Pending Pod
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: pending-test
  namespace: tenant-alpha
  labels:
    platform.io/tenant: tenant-alpha
spec:
  containers:
  - name: app
    image: nginx
    resources:
      requests:
        memory: "999Gi"  # Impossible
EOF
```

### Step 4: Wait for Detection

```bash
# Wait 1-2 minutes for pods to fail
sleep 60

# Check pod status
kubectl get pods -n tenant-alpha

# Trigger manual scan
curl -X POST http://localhost:9080/api/v1/troubleshooting/scan

# Wait for evaluation
sleep 15
```

### Step 5: Verify UI

**Refresh the Troubleshooting page** (or wait for auto-refresh)

#### ✅ Things to Check:

**Summary Cards:**
- [ ] Total Findings > 0
- [ ] Critical or High count > 0  
- [ ] Cards show correct numbers

**Top Issues Section:**
- [ ] Shows 3-5 findings
- [ ] Severity badges (red for high, orange for critical)
- [ ] Resource names displayed
- [ ] Tenant names shown

**Finding Cards:**
- [ ] Click expand button (▶) - card expands
- [ ] Title is descriptive
- [ ] Message explains the problem
- [ ] Recommendation section appears
- [ ] Shows first seen / last seen times
- [ ] Shows occurrence count

**Actions:**
- [ ] "View Resource" button present
- [ ] "Mark Resolved" button present
- [ ] Buttons have hover effects

**Filtering:**
- [ ] Select "High" severity - only high findings show
- [ ] Select "tenant-alpha" - only that tenant's findings show
- [ ] Type in search box - results filter
- [ ] Clear filters - all findings return

**Auto-Refresh:**
- [ ] Wait 30 seconds - page refreshes automatically
- [ ] Findings update without manual refresh

**Manual Scan:**
- [ ] Click "⚡ Trigger Scan" button
- [ ] Button shows "⚡ Scanning..." state
- [ ] Findings update after scan completes

---

## 🔍 Expected Findings

### 1. CrashLoopBackOff Finding

```
Severity: HIGH
Title: Pod crashloop-test is in CrashLoopBackOff
Resource: Pod/tenant-alpha/crashloop-test
Message: Container has crashed X times...
Recommendation: Check container logs: kubectl logs crashloop-test -n tenant-alpha
```

### 2. ImagePullBackOff Finding

```
Severity: HIGH
Title: Pod imagepull-test cannot pull image
Resource: Pod/tenant-alpha/imagepull-test
Message: Failed to pull image 'nonexistent:v999'...
Recommendation: Verify image exists and registry credentials are configured
```

### 3. Pod Pending Finding

```
Severity: MEDIUM
Title: Pod pending-test stuck in Pending state
Resource: Pod/tenant-alpha/pending-test
Message: Pod has been pending for X minutes...
Recommendation: Check pod events: kubectl describe pod pending-test
```

---

## 🧹 Cleanup

### Automated (Using test-phase5-ui.sh)

The script will prompt you:
```
Press ENTER after verifying the UI...
```

Then it automatically cleans up.

### Manual Cleanup

```bash
# Delete test pods
kubectl delete pod crashloop-test -n tenant-alpha
kubectl delete pod imagepull-test -n tenant-alpha
kubectl delete pod pending-test -n tenant-alpha

# Wait for evaluation (2 minutes) or trigger scan
curl -X POST http://localhost:9080/api/v1/troubleshooting/scan
sleep 10

# Verify cleanup
curl http://localhost:9080/api/v1/troubleshooting/summary | jq
```

**Expected:** `totalFindings` should return to 0

---

## 🎨 UI Visual Checklist

### Dark Theme Consistency
- [ ] Background: Dark blue/gray
- [ ] Text: Light gray/white
- [ ] Cards: Darker background with borders
- [ ] Severity badges: Colored (red/orange/yellow/blue)

### Layout
- [ ] Summary cards in grid (responsive)
- [ ] Findings list below summary
- [ ] Filters above findings list
- [ ] Expandable card design

### Colors
- [ ] Critical: Red (#ef4444)
- [ ] High: Orange (#f97316)
- [ ] Medium: Yellow (#eab308)
- [ ] Low: Blue (#3b82f6)
- [ ] Info: Gray (#6b7280)

### Interactions
- [ ] Hover effects on buttons
- [ ] Smooth expand/collapse animation
- [ ] Loading spinner during scans
- [ ] Disabled state on buttons

---

## 📊 API Verification

While testing UI, verify API responses:

```bash
# Get summary
curl http://localhost:9080/api/v1/troubleshooting/summary | jq

# Get all findings
curl http://localhost:9080/api/v1/troubleshooting/findings | jq

# Get critical only
curl "http://localhost:9080/api/v1/troubleshooting/findings?severity=critical" | jq

# Get tenant-specific
curl http://localhost:9080/api/v1/troubleshooting/tenants/tenant-alpha | jq
```

---

## 🐛 Troubleshooting

### No Findings Showing

**Problem:** UI shows 0 findings even after creating problems

**Solutions:**
1. Wait 2 minutes for automatic evaluation
2. Trigger manual scan: Click "⚡ Trigger Scan" button
3. Check pod status: `kubectl get pods -n tenant-alpha`
4. Verify pods have labels: `kubectl get pods -n tenant-alpha --show-labels`
5. Check backend logs: `tail -f /tmp/phase5-*.log`

### Findings Not Clearing After Cleanup

**Problem:** Findings still show after deleting pods

**Solutions:**
1. Wait for next evaluation cycle (up to 2 minutes)
2. Trigger manual scan
3. Check pods are actually deleted: `kubectl get pods -n tenant-alpha`

### UI Not Loading

**Problem:** Browser shows error or blank page

**Solutions:**
1. Check frontend is running: `curl http://localhost:9083`
2. Check browser console for errors (F12)
3. Rebuild frontend: `cd web && npm run build`
4. Clear browser cache

### API 404 Errors

**Problem:** API endpoints return 404

**Solutions:**
1. Verify manager is running: `curl http://localhost:9080/healthz`
2. Check rule evaluator initialized: `grep "Rule evaluator" /tmp/phase5-*.log`
3. Rebuild backend: `go build -o bin/manager ./cmd/main.go`

---

## ✅ Success Criteria

- [x] Backend running with rule evaluator
- [x] Frontend accessible at localhost:9083
- [x] Can create problematic resources
- [x] Rules detect issues within 2 minutes
- [x] UI displays findings correctly
- [x] All UI elements functional (expand, filter, search)
- [x] Manual scan triggers work
- [x] Cleanup removes findings
- [x] System returns to 0 findings after cleanup

---

## 📸 Screenshots to Verify

Take screenshots of:
1. **Clean state** - 0 findings, "No Issues Found"
2. **With findings** - Summary cards showing counts
3. **Expanded finding** - Full details visible
4. **Filtered view** - Only high severity shown
5. **After cleanup** - Back to 0 findings

---

## 🎯 Next Steps

After successful UI testing:
1. ✅ Document any UI issues found
2. ✅ Test on different browsers (Chrome, Firefox, Safari)
3. ✅ Test responsive design (mobile view)
4. ✅ Prepare for production deployment
5. ✅ Move to Phase 6 (Web Terminal)

---

**Test Duration:** ~10-15 minutes  
**Difficulty:** Easy  
**Automation:** Available via `test-phase5-ui.sh`


