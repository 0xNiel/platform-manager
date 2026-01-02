# 🧪 Phase 4 Testing - Step-by-Step Walkthrough

## ✅ Current Status

Your environment is ready:
- ✅ LocalStack IAM service: **RUNNING**
- ✅ Crossplane AWS Provider: **INSTALLED**
- ✅ IAM Test Resources: **4 READY** (2 roles + 2 policies)
  - `tenant-alpha-lambda-role` ✅
  - `tenant-beta-data-role` ✅
  - `tenant-alpha-s3-policy` ✅
  - `tenant-beta-dynamodb-policy` ✅
- ✅ Platform Manager Binary: **COMPILED** (81MB)
- ✅ Frontend: **BUILT** (1.3MB, zero errors)

---

## 📝 Testing Steps

### Step 1: Start the Platform Manager

Open Terminal 1 and run:

```bash
cd /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager

./start-phase4.sh
```

**Expected output:**
```
========================================
  Phase 4: IAM Drift Detection Testing
========================================

✓ Checking prerequisites...
  ✓ LocalStack is running
  ✓ Kubernetes cluster accessible
  ✓ Found 4 IAM resources

✓ Starting Platform Manager...
  API: http://localhost:9080
  Logs: ./manager-phase4.log

INFO    iam-drift-scanner    Creating AWS IAM client for LocalStack
INFO    setup               IAM drift scanner initialized    interval=1m0s
INFO    setup               Starting API server              addr=:9080
INFO    iam-drift-scanner   Starting IAM drift scanner
INFO    iam-drift-scanner   Starting drift scan
INFO    iam-drift-scanner   Drift scan completed            duration=...ms
```

**✋ Leave this terminal running!** The manager logs will appear here.

---

### Step 2: Test the IAM Drift APIs

Open Terminal 2 and run:

```bash
cd /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager

./test-phase4-api.sh
```

**Expected output:**
```
============================================
  Phase 4: IAM Drift Detection API Tests
============================================

✓ Manager is running

Test 1: Platform Drift Summary
GET http://localhost:9080/api/v1/iam/drift/platform

{
  "summary": {
    "totalTenants": 3,
    "totalRoles": 2,
    "totalPolicies": 2,
    "rolesWithDrift": 0,
    "policiesWithDrift": 0,
    "criticalDrifts": 0,
    "highDrifts": 0,
    "warningDrifts": 0
  },
  "lastScanTime": "2026-01-02T..."
}
✓ PASS

Test 2: All Tenants Drift
...
✓ All API tests completed!
```

**✅ If you see this, Phase 4 APIs are working!**

---

### Step 3: Create Drift for Testing

Still in Terminal 2:

```bash
./create-drift.sh
```

**What this does:**
- Adds an **unauthorized inline policy** to `tenant-alpha-lambda-role`
- This policy grants `s3:*` (full S3 access) which is NOT in the Crossplane spec
- Should be detected as **CRITICAL** drift (extra_privileges)

**Expected output:**
```
============================================
  Creating IAM Drift for Testing
============================================

✓ Creating unauthorized inline policy...

✅ Drift created successfully!

Next steps:
1. Wait 1 minute for next drift scan, OR
2. Trigger manual scan: curl -X POST http://localhost:9080/api/v1/iam/drift/scan
3. Wait 10 seconds
4. Check drift: ./test-phase4-api.sh
```

---

### Step 4: Trigger Drift Scan

```bash
# Trigger immediate scan
curl -X POST http://localhost:9080/api/v1/iam/drift/scan

# Wait for scan to complete
sleep 15

# Check for drift
curl -s http://localhost:9080/api/v1/iam/drift/platform | jq '.summary'
```

**Expected output:**
```json
{
  "totalRoles": 2,
  "totalPolicies": 2,
  "rolesWithDrift": 1,        ← Should be 1 now!
  "policiesWithDrift": 0,
  "criticalDrifts": 1,         ← Critical drift detected!
  "highDrifts": 0,
  "warningDrifts": 0
}
```

**✅ If `criticalDrifts: 1`, drift detection is working!**

---

### Step 5: View Detailed Drift Information

```bash
# Get detailed drift for tenant alpha
curl -s http://localhost:9080/api/v1/iam/drift/tenants/alpha | jq '.details[] | select(.hasDrift == true)'
```

**Expected output:**
```json
{
  "resourceType": "Role",
  "resourceName": "tenant-alpha-lambda-role",
  "hasDrift": true,
  "severity": "critical",
  "details": [
    {
      "type": "extra_privileges",
      "severity": "critical",
      "message": "Unexpected inline policy found in AWS: unauthorized-s3-full-access",
      "path": "inlinePolicies.unauthorized-s3-full-access"
    }
  ],
  "checkedAt": "2026-01-02T..."
}
```

**✅ This shows the exact drift detected!**

---

### Step 6: Test the Frontend

Open Terminal 3:

```bash
cd /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager/web

# Start dev server
npm run serve
```

Then open your browser to: **http://localhost:8080**

Click on **"IAM Drift"** in the sidebar.

**You should see:**
- 📊 Platform summary showing 1 critical drift
- 📋 Tenant cards with drift indicators
- 🔴 Tenant alpha marked with drift
- 📝 Click on alpha to see detailed findings

**✅ If the UI shows the drift, frontend is working!**

---

## 🎯 Success Checklist

Mark these off as you test:

- [ ] Manager starts without errors
- [ ] APIs return valid JSON
- [ ] Platform summary shows 2 roles, 2 policies
- [ ] Initially 0 drift
- [ ] create-drift.sh succeeds
- [ ] After scan, 1 critical drift detected
- [ ] Tenant details show the extra policy
- [ ] Frontend displays drift correctly
- [ ] Manual scan trigger works

---

## 📊 Monitoring

### Watch drift scans in real-time:

Terminal 4:
```bash
tail -f manager-phase4.log | grep drift
```

You'll see:
```
INFO    iam-drift-scanner   Starting drift scan
INFO    iam-drift-scanner   Drift scan completed    tenants=3  totalRoles=2  rolesWithDrift=1
```

---

## 🐛 Troubleshooting

### "Manager not running"
```bash
# Check if something is on port 9080
lsof -i :9080

# Check manager logs
tail -50 manager-phase4.log
```

### "0 roles found"
```bash
# Verify Crossplane synced to LocalStack
awslocal iam list-roles | jq '.Roles[] | .RoleName'

# Should show:
# - tenant-alpha-lambda-role
# - tenant-beta-data-role
```

### "Drift not detected"
```bash
# Check if policy was actually added
awslocal iam list-role-policies --role-name tenant-alpha-lambda-role

# Should show: unauthorized-s3-full-access

# Check manager logs for errors
tail -50 manager-phase4.log | grep -i error
```

---

## ✅ When Everything Works

You've successfully tested Phase 4 when:

1. ✅ Manager starts and shows "IAM drift scanner initialized"
2. ✅ APIs return drift summaries with correct counts
3. ✅ Drift creation adds critical findings
4. ✅ Frontend displays the drift data
5. ✅ Manual scan trigger works

---

## 🎉 Next Steps

Once Phase 4 is verified working:

1. **Clean up test drift:**
   ```bash
   awslocal iam delete-role-policy \
     --role-name tenant-alpha-lambda-role \
     --policy-name unauthorized-s3-full-access
   ```

2. **Run full test suite:**
   ```bash
   ./test-phase4-drift.sh
   ```

3. **Ready for Phase 5?**
   - Troubleshooting & Rule Engine
   - Automated issue detection
   - Root cause analysis

---

**Let me know if you hit any issues and I'll help you debug!** 🚀

