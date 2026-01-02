# Phase 4 Testing Session - January 2, 2026

## ✅ Setup Completed Successfully

### 1. Crossplane AWS IAM Provider
- **Status**: ✅ INSTALLED and HEALTHY
- **Provider**: `provider-aws-iam` v1.7.0
- **ProviderConfig**: `localstack` configured for http://localhost:4566

### 2. IAM Test Resources Deployed
```bash
✅ role.iam.aws.upbound.io/tenant-alpha-lambda-role   (READY/SYNCED)
✅ role.iam.aws.upbound.io/tenant-beta-data-role      (READY/SYNCED)
✅ policy.iam.aws.upbound.io/tenant-alpha-s3-policy   (READY/SYNCED)
✅ policy.iam.aws.upbound.io/tenant-beta-dynamodb-policy (READY/SYNCED)
```

### 3. Platform Manager Running
- **Status**: ✅ RUNNING
- **API Server**: http://localhost:9080
- **Health Probe**: :8082
- **IAM Drift Scanner**: Active (1m interval)
- **First Scan**: Completed in 136ms

### 4. Frontend Built
- **Status**: ✅ BUILD SUCCESSFUL
- **Size**: 1.3MB
- **Location**: `web/dist/`
- **Zero TypeScript errors**

---

## 🧪 Testing the IAM Drift API

Now that the manager is running, let's test the Phase 4 endpoints:

```bash
# Terminal 1 - Platform Manager is running
# Logs showing:
# - IAM drift scanner initialized
# - API server started on :9080
# - Drift scans running every 1 minute

# Terminal 2 - Run these tests:

# Test 1: Platform Drift Summary
curl -s http://localhost:9080/api/v1/iam/drift/platform | jq

# Expected Response:
{
  "summary": {
    "totalTenants": 3,
    "totalRoles": 2,
    "totalPolicies": 2,
    "rolesWithDrift": 0,
    "policiesWithDrift": 0,
    "criticalDrifts": 0,
    "highDrifts": 0,
    "warningDrifts": 0,
    "lastChecked": "2026-01-02T17:13:12Z",
    "tenantSummaries": {...}
  },
  "lastScanTime": "2026-01-02T17:13:12Z"
}

# Test 2: All Tenants Drift
curl -s http://localhost:9080/api/v1/iam/drift/tenants | jq

# Test 3: Tenant Alpha Drift Details
curl -s http://localhost:9080/api/v1/iam/drift/tenants/alpha | jq

# Test 4: Trigger Manual Scan
curl -X POST http://localhost:9080/api/v1/iam/drift/scan
# Response: {"status": "scanning", "message": "Drift scan initiated"}
```

---

## 🎯 Next Testing Steps

### Step 1: Verify IAM Resources in LocalStack

```bash
# Check if Crossplane created the roles in LocalStack
awslocal iam list-roles | jq '.Roles[] | {RoleName, CreateDate}'

# Check if policies were created
awslocal iam list-policies --scope Local | jq '.Policies[] | {PolicyName, Arn}'

# Expected to see:
# - tenant-alpha-lambda-role
# - tenant-beta-data-role
# - tenant-alpha-s3-policy  
# - tenant-beta-dynamodb-policy
```

### Step 2: Create Drift for Testing

```bash
# Add an unauthorized inline policy to a role (Critical Drift)
awslocal iam put-role-policy \
  --role-name tenant-alpha-lambda-role \
  --policy-name unauthorized-s3-access \
  --policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Action": "s3:*",
      "Resource": "*"
    }]
  }'

# Wait for next drift scan (1 minute) or trigger manual scan
curl -X POST http://localhost:9080/api/v1/iam/drift/scan

# Wait 10 seconds for scan to complete
sleep 10

# Check drift again
curl -s http://localhost:9080/api/v1/iam/drift/platform | jq '.summary'

# Expected: Should now show criticalDrifts: 1
```

### Step 3: Test Frontend

```bash
# Option 1: Serve production build
cd web/dist
python3 -m http.server 8080

# Option 2: Dev server
cd web
npm run serve

# Then open browser to:
# http://localhost:8080

# Navigate to "IAM Drift" view and verify:
# - Platform summary shows correct counts
# - Tenant cards display
# - Clicking a tenant shows drift details
# - "Trigger Scan" button works
```

---

## 📊 Current State

### What's Working ✅
1. ✅ Manager compiled and running
2. ✅ IAM drift scanner initialized
3. ✅ API server listening on :9080
4. ✅ Drift scans executing (every 1m)
5. ✅ Crossplane IAM resources deployed and synced
6. ✅ LocalStack IAM service running
7. ✅ Frontend built with zero errors

### What Needs Verification 🔍
1. 🔍 IAM resources in LocalStack (check with awslocal)
2. 🔍 Drift scanner finding the IAM resources
3. 🔍 API endpoints returning data
4. 🔍 Frontend displaying drift information
5. 🔍 Drift detection after manual changes

---

## 🐛 Known Issues

### Issue: Drift scanner shows 0 roles/0 policies

**Root Cause**: The IAM resources might not have the correct labels for tenant matching, OR Crossplane hasn't synced them to LocalStack yet.

**Debug Steps**:
```bash
# 1. Check if resources have tenant labels
kubectl get roles.iam.aws.upbound.io,policies.iam.aws.upbound.io -A \
  --show-labels | grep "platform.io/tenant"

# 2. Check Crossplane status
kubectl get roles.iam.aws.upbound.io tenant-alpha-lambda-role -o yaml | \
  yq '.status.conditions'

# 3. Check LocalStack directly
awslocal iam list-roles

# 4. Check manager logs for drift scanner
tail -f manager.log | grep "iam-drift"
```

**Expected Fix**: The scanner looks for resources with `platform.io/tenant` labels. If the IAM resources have these labels and are synced to LocalStack, the next scan should pick them up.

---

## 🎉 Success Criteria

Phase 4 will be considered fully working when:

- [ ] Platform drift API returns > 0 total roles
- [ ] Platform drift API returns > 0 total policies  
- [ ] Creating drift in LocalStack is detected
- [ ] Frontend displays drift data
- [ ] Manual scan trigger works
- [ ] TenantHealth.IAMDrift is populated

---

**Current Status**: Manager running, waiting for API verification...

**Next Command to Run**:
```bash
curl -s http://localhost:9080/api/v1/iam/drift/platform | jq
```

