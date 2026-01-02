# Phase 5 UI Testing - Quick Start

## 🚀 Run the Test

```bash
# Terminal 1: Start backend
./start-phase5.sh

# Terminal 2: Start frontend
./start-frontend.sh

# Terminal 3: Run test
./test-phase5-ui.sh
```

Then open: **http://localhost:9083/troubleshooting**

## 📋 What the Test Does

1. ✅ Creates 3 failing pods (crashloop, imagepull, pending)
2. ✅ Creates resource quota with high usage
3. ✅ Waits for rule engine to detect issues
4. ✅ Pauses for you to verify UI
5. ✅ Cleans up all test resources
6. ✅ Verifies findings clear

## 🎯 What to Check in UI

- [ ] Summary cards show finding counts
- [ ] Top Issues section displays problems
- [ ] Expandable finding cards work
- [ ] Severity badges colored correctly (red/orange/yellow)
- [ ] Recommendations shown
- [ ] Filtering works (severity, tenant, search)
- [ ] Manual scan button works
- [ ] Auto-refresh every 30 seconds
- [ ] Dark theme consistent

## ⏱️ Timeline

- **0:00** - Test creates problems
- **0:30** - Pods start failing
- **1:30** - Rule engine detects issues
- **2:00** - UI shows findings ← **Verify here**
- **3:00** - Cleanup runs
- **4:00** - Findings cleared

## 🔍 Expected Findings

You should see 3-4 findings:
- 🔴 **HIGH:** Pod CrashLoopBackOff  
- 🔴 **HIGH:** Image Pull Failed  
- 🟡 **MEDIUM:** Pod Stuck Pending  
- 🟡 **MEDIUM:** High Resource Usage (maybe)

## 🧹 If Something Goes Wrong

```bash
# Clean up manually
kubectl delete pods -n tenant-alpha --selector=phase5-test=true
kubectl delete resourcequota -n tenant-alpha phase5-test-quota

# Restart manager
pkill -f "bin/manager"
./start-phase5.sh
```

## 📚 Full Documentation

See: `documentation/PHASE5_UI_TESTING_GUIDE.md`

---

**Status:** Ready to test!  
**Duration:** ~5 minutes

