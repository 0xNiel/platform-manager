#!/bin/bash
# create-drift.sh - Create IAM drift for testing

echo "============================================"
echo "  Creating IAM Drift for Testing"
echo "============================================"
echo ""

# Check if awslocal is available
if ! command -v awslocal &> /dev/null; then
    echo "❌ awslocal command not found!"
    echo "   Install with: pip install awscli-local"
    exit 1
fi

# Check LocalStack
if ! curl -s http://localhost:4566/_localstack/health > /dev/null 2>&1; then
    echo "❌ LocalStack is not running!"
    exit 1
fi

echo "✓ Creating unauthorized inline policy on tenant-alpha-lambda-role..."
echo ""

awslocal iam put-role-policy \
  --role-name tenant-alpha-lambda-role \
  --policy-name unauthorized-s3-full-access \
  --policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Sid": "UnauthorizedAccess",
      "Effect": "Allow",
      "Action": "s3:*",
      "Resource": "*"
    }]
  }' 2>&1

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Drift created successfully!"
    echo ""
    echo "The role now has an EXTRA policy that is not in the Crossplane spec."
    echo "This should be detected as CRITICAL drift (extra_privileges)."
    echo ""
    echo "Next steps:"
    echo "1. Wait 1 minute for next drift scan, OR"
    echo "2. Trigger manual scan: curl -X POST http://localhost:9080/api/v1/iam/drift/scan"
    echo "3. Wait 10 seconds for scan to complete"
    echo "4. Check drift: ./test-phase4-api.sh"
    echo ""
else
    echo "❌ Failed to create drift"
    echo "   This might mean the role doesn't exist in LocalStack yet"
    echo "   Check: awslocal iam list-roles"
fi

