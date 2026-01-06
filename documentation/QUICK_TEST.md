# 🚀 Phase 4 Quick Test Guide

## ✅ Prerequisites Status
- ✅ LocalStack running with IAM service
- ✅ 2 IAM Roles in AWS: `tenant-alpha-lambda-role`, `tenant-beta-data-role`
- ✅ 2 IAM Policies in AWS
- ✅ Manager binary compiled and ready

---

## 🏃 Quick Start (5 minutes)

### Terminal 1: Start Manager
```bash
cd ~/Documents/2-WorkStuff--ai-platform-in-go/platform-manager
./start-phase4.sh
```
**Wait for:** `INFO iam-drift-scanner Starting IAM drift scanner`

### Terminal 2: Test APIs
```bash
./test-phase4-api.sh
```
**Expected:** All tests PASS, 0 drift initially

### Terminal 3: Create Drift
```bash
./create-drift.sh
curl -X POST http://localhost:9080/api/v1/iam/drift/scan
sleep 15
./test-phase4-api.sh
```
**Expected:** 1 CRITICAL drift detected in tenant-alpha

### Browser: View Frontend
```bash
cd web && npm run serve
```
Open: http://localhost:8080 → Click "IAM Drift"

---

## 📋 Quick Verification

```bash
# Check platform summary
curl -s http://localhost:9080/api/v1/iam/drift/platform | jq '.summary'

# Should show:
# "rolesWithDrift": 1
# "criticalDrifts": 1
```

---

## 🎯 Success = All Green ✅

- Manager starts without errors ✅
- APIs return JSON ✅  
- Drift detected after create-drift.sh ✅
- Frontend shows drift ✅

**Full walkthrough:** See `TESTING_WALKTHROUGH.md`

