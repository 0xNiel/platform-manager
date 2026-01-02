# 🎯 Phase 4 Ready to Test - Final Instructions

## ✅ Everything Is Ready!

All Phase 4 components have been successfully implemented and compiled:
- ✅ Backend with IAM drift detection
- ✅ Frontend with beautiful UI  
- ✅ IAM test resources deployed to cluster
- ✅ LocalStack is running

---

## 🚀 Start Testing - Run These Commands

### In Terminal 1 - Start the Platform Manager:

```bash
cd /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager

# Set environment variables
export AWS_ENDPOINT=http://localhost:4566
export AWS_REGION=us-east-1

# Start the manager
./bin/manager \
  --api-bind-address=:9080 \
  --health-probe-bind-address=:8082 \
  --metrics-bind-address=0 \
  --iam-drift-scan-interval=2m

# You should see logs like:
# INFO    IAM drift scanner initialized    interval=2m0s
# INFO    Starting API server              addr=:9080
# INFO    IAM drift scanner: Starting drift scan
# INFO    Drift scan completed             duration=...ms tenants=3
```

### In Terminal 2 - Test the APIs:

```bash
# Test 1: Platform drift summary
curl -s http://localhost:9080/api/v1/iam/drift/platform | jq

# Test 2: All tenants drift
curl -s http://localhost:9080/api/v1/iam/drift/tenants | jq

# Test 3: Tenant alpha details
curl -s http://localhost:9080/api/v1/iam/drift/tenants/alpha | jq

# Test 4: Trigger manual scan
curl -X POST http://localhost:9080/api/v1/iam/drift/scan

# Test 5: Check LocalStack has the roles
awslocal iam list-roles | jq '.Roles[] | {RoleName, Path}'
```

### In Terminal 3 - Create Drift and Test Detection:

```bash
# Add unauthorized policy to create drift
awslocal iam put-role-policy \
  --role-name tenant-alpha-lambda-role \
  --policy-name unauthorized-access \
  --policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Action": "s3:*",
      "Resource": "*"
    }]
  }'

# Trigger scan
curl -X POST http://localhost:9080/api/v1/iam/drift/scan

# Wait 15 seconds for scan
sleep 15

# Check for drift (should show 1 critical drift)
curl -s http://localhost:9080/api/v1/iam/drift/platform | jq '.summary'
```

---

## 🎨 Test the Frontend

```bash
# Open a new terminal
cd /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager/web

# Serve the built frontend
npm run serve

# OR serve the dist folder
cd dist && python3 -m http.server 8080
```

Then open your browser to `http://localhost:8080` and click on "IAM Drift" in the sidebar.

---

## 📋 Full Integration Test Script

Or just run the automated test script:

```bash
cd /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager

# Make sure manager is running first, then:
./test-phase4-drift.sh
```

---

## 🐛 Troubleshooting

### If API returns 404:
- Check manager is running: `lsof -i :9080`
- Check manager logs for errors
- Verify health endpoint: `curl http://localhost:8082/healthz`

### If drift scanner shows 0 roles:
- Check resources are synced: `kubectl get roles.iam.aws.upbound.io -A`
- Check LocalStack: `awslocal iam list-roles`
- Wait for next scan (2 minutes) or trigger manual scan

### If LocalStack connection fails:
- Verify LocalStack is running: `curl http://localhost:4566/_localstack/health`
- Check AWS_ENDPOINT is set: `echo $AWS_ENDPOINT`
- Restart manager with correct environment variables

---

## 📊 Expected Results

### Without Drift (Initial State):
```json
{
  "summary": {
    "totalRoles": 2,
    "totalPolicies": 2,
    "rolesWithDrift": 0,
    "policiesWithDrift": 0,
    "criticalDrifts": 0
  }
}
```

### With Drift (After Manual Modification):
```json
{
  "summary": {
    "totalRoles": 2,
    "totalPolicies": 2,
    "rolesWithDrift": 1,
    "policiesWithDrift": 0,
    "criticalDrifts": 1
  }
}
```

---

## ✅ Phase 4 Complete!

Once you verify the above tests work, Phase 4 is **100% complete** and ready for production!

All documentation is in:
- `documentation/PHASE4_COMPLETE.md` - Technical details
- `documentation/PHASE4_QUICKSTART.md` - Step-by-step guide
- `documentation/PHASE4_TESTING_SESSION.md` - This testing session
- `test-phase4-drift.sh` - Automated test script

**Ready for Phase 5?** Let me know when you want to start the Troubleshooting & Rule Engine! 🚀

