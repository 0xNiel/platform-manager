# Phase 4: Quick Start Guide

## ✅ Prerequisites Checklist

- [x] LocalStack running on `http://localhost:4566`
- [x] Kind cluster running
- [x] Crossplane AWS IAM Provider installed
- [x] Backend compiled successfully
- [x] Frontend compiled successfully

## 🚀 Start Phase 4

### Step 1: Start LocalStack (if not running)

```bash
# In a separate terminal
localstack start

# Or with Docker
docker run --rm -p 4566:4566 localstack/localstack
```

Verify LocalStack is ready:
```bash
curl http://localhost:4566/_localstack/health
```

### Step 2: Deploy IAM Test Resources

```bash
# Deploy Crossplane IAM resources
kubectl apply -f hack/seed-tenants/iam-resources.yaml

# Wait for reconciliation (30 seconds)
sleep 30

# Check that resources are created
kubectl get roles.iam.aws.upbound.io -A
kubectl get policies.iam.aws.upbound.io -A
```

Expected output:
```
NAME                         READY   SYNCED   ...
tenant-alpha-lambda-role     True    True     ...
tenant-beta-data-role        True    True     ...

NAME                           READY   SYNCED   ...
tenant-alpha-s3-policy         True    True     ...
tenant-beta-dynamodb-policy    True    True     ...
```

### Step 3: Start the Platform Manager

```bash
# Set environment variables for LocalStack
export AWS_ENDPOINT=http://localhost:4566
export AWS_REGION=us-east-1

# Start the manager
./bin/manager \
  --api-bind-address=:9080 \
  --metrics-bind-address=:8443 \
  --iam-drift-scan-interval=5m
```

You should see logs like:
```
INFO    starting manager
INFO    IAM drift scanner initialized    interval=5m0s
INFO    Starting IAM drift scanner      interval=5m0s
INFO    Starting drift scan
INFO    Drift scan completed            duration=2.5s  tenants=3  totalRoles=2  totalPolicies=2
INFO    Starting API server             addr=:9080
```

### Step 4: Test the API Endpoints

Open a new terminal and run:

```bash
# 1. Check platform drift summary
curl http://localhost:9080/api/v1/iam/drift/platform | jq

# Expected: Should show 2 roles, 2 policies, 0 drift
# {
#   "summary": {
#     "totalRoles": 2,
#     "totalPolicies": 2,
#     "rolesWithDrift": 0,
#     "policiesWithDrift": 0,
#     "criticalDrifts": 0
#   }
# }

# 2. Check tenant summaries
curl http://localhost:9080/api/v1/iam/drift/tenants | jq

# 3. Check tenant alpha details
curl http://localhost:9080/api/v1/iam/drift/tenants/alpha | jq

# 4. Trigger manual scan
curl -X POST http://localhost:9080/api/v1/iam/drift/scan
```

### Step 5: Create Drift and Test Detection

```bash
# Create drift by adding an extra inline policy to a role in LocalStack
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

# Trigger a new scan
curl -X POST http://localhost:9080/api/v1/iam/drift/scan

# Wait 10 seconds for scan to complete
sleep 10

# Check for detected drift
curl http://localhost:9080/api/v1/iam/drift/platform | jq

# Expected: Should now show 1 critical drift
# {
#   "summary": {
#     "totalRoles": 2,
#     "totalPolicies": 2,
#     "rolesWithDrift": 1,
#     "criticalDrifts": 1
#   }
# }

# Get detailed drift information
curl http://localhost:9080/api/v1/iam/drift/tenants/alpha | jq '.details[] | select(.hasDrift == true)'
```

### Step 6: Access the Frontend

```bash
# The frontend is already built in dist/
# You can serve it with any static server, or use the dev server:

cd web
npm run serve

# Open browser to http://localhost:8080
# Navigate to "IAM Drift" in the sidebar
```

You should see:
- Platform summary showing total roles/policies
- Tenant cards with drift indicators
- Click on a tenant to see detailed findings
- "Trigger Scan" button to manually refresh

### Step 7: Run Integration Tests

```bash
# Run the comprehensive test suite
./test-phase4-drift.sh
```

Expected output:
```
======================================
Phase 4: IAM Drift Detection Testing
======================================

Test 1: Get platform drift summary... ✓ PASS
Test 2: Get all tenants drift... ✓ PASS
Test 3: Get tenant alpha drift... ✓ PASS
Test 4: Trigger manual scan... ✓ PASS
Test 5: Platform summary shows IAM roles... ✓ PASS
Test 6: Platform summary shows IAM policies... ✓ PASS
Test 7: No drift detected initially... ✓ PASS
Test 8: Drift detected after modification... ✓ PASS
Test 9: Tenant alpha shows drift details... ✓ PASS
Test 10: TenantHealth populates IAM drift... ✓ PASS

Total tests: 10
Passed: 10
Failed: 0

✅ All tests passed!
```

## 🔍 Troubleshooting

### Issue: "Role not found in AWS"

**Cause**: Crossplane hasn't finished reconciling the IAM resources to LocalStack.

**Solution**:
```bash
# Check Crossplane status
kubectl get roles.iam.aws.upbound.io -A
# Look for READY=True and SYNCED=True

# Check LocalStack to see if roles exist
awslocal iam list-roles | jq '.Roles[] | .RoleName'
```

### Issue: "No IAM resources found"

**Cause**: IAM resources not deployed or provider not installed.

**Solution**:
```bash
# Check if AWS IAM Provider is installed
kubectl get providers | grep aws-iam

# If not, install it:
kubectl apply -f hack/crossplane/provider-aws.yaml
sleep 30

# Deploy IAM resources
kubectl apply -f hack/seed-tenants/iam-resources.yaml
```

### Issue: "Connection refused to AWS endpoint"

**Cause**: LocalStack not running or wrong endpoint.

**Solution**:
```bash
# Check LocalStack is running
curl http://localhost:4566/_localstack/health

# Verify AWS_ENDPOINT environment variable
echo $AWS_ENDPOINT
# Should be: http://localhost:4566

# Restart manager with correct endpoint
export AWS_ENDPOINT=http://localhost:4566
./bin/manager --api-bind-address=:9080
```

### Issue: "Frontend shows 'No data available'"

**Cause**: Backend not running or API not responding.

**Solution**:
```bash
# Check backend is running
curl http://localhost:9080/api/v1/iam/drift/platform

# Check for errors in manager logs
# Look for "Starting API server" and "IAM drift scanner initialized"
```

## 📊 Monitoring

### Watch Drift Scans in Real-Time

```bash
# Follow manager logs
./bin/manager --api-bind-address=:9080 2>&1 | grep "drift"

# You should see every 5 minutes:
# "Starting drift scan"
# "Drift scan completed"
```

### Check TenantHealth Integration

```bash
# View TenantHealth with IAM drift data
kubectl get tenanthealth alpha-health -o jsonpath='{.status.iamDrift}' | jq

# Expected:
# {
#   "totalRoles": 1,
#   "totalPolicies": 1,
#   "rolesWithDrift": 1,
#   "policiesWithDrift": 0,
#   "extraPrivileges": 1,
#   "lastChecked": "2026-01-02T16:50:00Z"
# }
```

## 🎯 Success Criteria

You'll know Phase 4 is working when:

- ✅ Manager starts without errors
- ✅ IAM drift scanner logs show periodic scans
- ✅ API endpoints return valid JSON responses
- ✅ Platform summary shows correct role/policy counts
- ✅ Creating drift in LocalStack is detected on next scan
- ✅ Frontend displays drift data correctly
- ✅ TenantHealth.IAMDrift is populated
- ✅ All integration tests pass

## 🚀 Next Steps

Once Phase 4 is verified working:

1. Test with different drift scenarios (policy changes, orphaned resources)
2. Test with multiple tenants
3. Test the manual scan trigger from the UI
4. Test auto-refresh in the frontend (30s interval)
5. Ready to move on to **Phase 5: Troubleshooting & Rule Engine**!

---

**Status**: Phase 4 is ready for testing! 🎉

