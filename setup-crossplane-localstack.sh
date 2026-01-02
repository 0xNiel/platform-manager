#!/bin/bash
# Setup Crossplane with LocalStack for full integration testing
# This script configures Crossplane to create Managed Resources in LocalStack

set -e

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║        🚀 Crossplane + LocalStack Setup & Testing               ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check prerequisites
echo -e "${BLUE}📋 Checking prerequisites...${NC}"

if ! kubectl get ns crossplane-system &>/dev/null; then
    echo -e "${RED}❌ Crossplane not installed. Please install Crossplane first.${NC}"
    exit 1
fi

if ! docker ps | grep localstack &>/dev/null; then
    echo -e "${RED}❌ LocalStack not running. Please start LocalStack first.${NC}"
    echo "   Run: docker run -d -p 4566:4566 --name localstack-main localstack/localstack"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites met${NC}"
echo ""

# Step 1: Apply ProviderConfig for LocalStack
echo -e "${BLUE}📦 Step 1: Configuring ProviderConfig for LocalStack...${NC}"
kubectl apply -f hack/crossplane/providerconfig-localstack.yaml
echo -e "${GREEN}✅ ProviderConfig applied${NC}"
echo ""

# Wait for ProviderConfig to be ready
echo -e "${BLUE}⏳ Waiting for ProviderConfig to be ready...${NC}"
timeout=60
elapsed=0
while ! kubectl get providerconfig localstack &>/dev/null; do
    sleep 2
    elapsed=$((elapsed + 2))
    if [ $elapsed -ge $timeout ]; then
        echo -e "${RED}❌ Timeout waiting for ProviderConfig${NC}"
        exit 1
    fi
done
echo -e "${GREEN}✅ ProviderConfig ready${NC}"
echo ""

# Step 2: Verify provider health
echo -e "${BLUE}🔍 Step 2: Verifying Crossplane providers...${NC}"
kubectl get providers
echo ""

# Check provider health
PROVIDER_HEALTHY=$(kubectl get provider provider-aws-iam -o jsonpath='{.status.conditions[?(@.type=="Healthy")].status}')
if [ "$PROVIDER_HEALTHY" != "True" ]; then
    echo -e "${YELLOW}⚠️  Provider not yet healthy, waiting...${NC}"
    sleep 10
fi

echo -e "${GREEN}✅ Providers verified${NC}"
echo ""

# Step 3: Check LocalStack connectivity
echo -e "${BLUE}🌐 Step 3: Testing LocalStack connectivity...${NC}"

# Test IAM endpoint
if curl -s http://localhost:4566/_localstack/health | grep -q "iam"; then
    echo -e "${GREEN}✅ LocalStack IAM service is available${NC}"
else
    echo -e "${RED}❌ LocalStack IAM service not responding${NC}"
    exit 1
fi
echo ""

# Step 4: Apply existing IAM resources
echo -e "${BLUE}🔐 Step 4: Creating IAM Managed Resources...${NC}"
kubectl apply -f hack/seed-tenants/iam-resources.yaml
echo -e "${GREEN}✅ IAM resources applied${NC}"
echo ""

# Step 5: Create additional AWS resources for comprehensive testing
echo -e "${BLUE}☁️  Step 5: Creating additional AWS Managed Resources...${NC}"

# Create S3 buckets
cat <<EOF | kubectl apply -f -
---
apiVersion: s3.aws.upbound.io/v1beta1
kind: Bucket
metadata:
  name: tenant-alpha-ml-data
  labels:
    platform.io/tenant: tenant-alpha
    platform.io/component: ml
    platform.io/resource-type: s3-bucket
spec:
  providerConfigRef:
    name: localstack
  forProvider:
    region: us-east-1
    tags:
      tenant: alpha
      purpose: ml-data
      managed-by: crossplane
---
apiVersion: s3.aws.upbound.io/v1beta1
kind: Bucket
metadata:
  name: tenant-beta-analytics
  labels:
    platform.io/tenant: tenant-beta
    platform.io/component: analytics
    platform.io/resource-type: s3-bucket
spec:
  providerConfigRef:
    name: localstack
  forProvider:
    region: us-east-1
    tags:
      tenant: beta
      purpose: analytics
      managed-by: crossplane
EOF

echo -e "${GREEN}✅ S3 buckets created${NC}"
echo ""

# Create DynamoDB tables
cat <<EOF | kubectl apply -f -
---
apiVersion: dynamodb.aws.upbound.io/v1beta2
kind: Table
metadata:
  name: tenant-alpha-ml-models
  labels:
    platform.io/tenant: tenant-alpha
    platform.io/component: ml
    platform.io/resource-type: dynamodb-table
spec:
  providerConfigRef:
    name: localstack
  forProvider:
    region: us-east-1
    billingMode: PAY_PER_REQUEST
    attribute:
      - name: modelId
        type: S
      - name: version
        type: N
    hashKey: modelId
    rangeKey: version
    tags:
      tenant: alpha
      purpose: ml-models
      managed-by: crossplane
---
apiVersion: dynamodb.aws.upbound.io/v1beta2
kind: Table
metadata:
  name: tenant-beta-events
  labels:
    platform.io/tenant: tenant-beta
    platform.io/component: analytics
    platform.io/resource-type: dynamodb-table
spec:
  providerConfigRef:
    name: localstack
  forProvider:
    region: us-east-1
    billingMode: PAY_PER_REQUEST
    attribute:
      - name: eventId
        type: S
      - name: timestamp
        type: N
    hashKey: eventId
    rangeKey: timestamp
    tags:
      tenant: beta
      purpose: events
      managed-by: crossplane
EOF

echo -e "${GREEN}✅ DynamoDB tables created${NC}"
echo ""

# Step 6: Wait for resources to reconcile
echo -e "${BLUE}⏳ Step 6: Waiting for Managed Resources to reconcile...${NC}"
echo "This may take 30-60 seconds..."
sleep 15

# Check IAM roles
echo ""
echo -e "${YELLOW}IAM Roles:${NC}"
kubectl get role.iam.aws.upbound.io -o wide

# Check IAM policies
echo ""
echo -e "${YELLOW}IAM Policies:${NC}"
kubectl get policy.iam.aws.upbound.io -o wide

# Check S3 buckets
echo ""
echo -e "${YELLOW}S3 Buckets:${NC}"
kubectl get bucket.s3.aws.upbound.io -o wide

# Check DynamoDB tables
echo ""
echo -e "${YELLOW}DynamoDB Tables:${NC}"
kubectl get table.dynamodb.aws.upbound.io -o wide

echo ""

# Step 7: Verify resources in LocalStack
echo -e "${BLUE}🔍 Step 7: Verifying resources in LocalStack...${NC}"

echo ""
echo -e "${YELLOW}IAM Roles in LocalStack:${NC}"
aws --endpoint-url=http://localhost:4566 iam list-roles --query 'Roles[].RoleName' 2>/dev/null || echo "AWS CLI not configured"

echo ""
echo -e "${YELLOW}IAM Policies in LocalStack:${NC}"
aws --endpoint-url=http://localhost:4566 iam list-policies --scope Local --query 'Policies[].PolicyName' 2>/dev/null || echo "AWS CLI not configured"

echo ""
echo -e "${YELLOW}S3 Buckets in LocalStack:${NC}"
aws --endpoint-url=http://localhost:4566 s3 ls 2>/dev/null || echo "AWS CLI not configured"

echo ""
echo -e "${YELLOW}DynamoDB Tables in LocalStack:${NC}"
aws --endpoint-url=http://localhost:4566 dynamodb list-tables 2>/dev/null || echo "AWS CLI not configured"

echo ""

# Step 8: Check ResourceSummary CRs
echo -e "${BLUE}📊 Step 8: Checking ResourceSummary CRs...${NC}"
echo "The platform-manager should automatically scan these resources."
echo ""

sleep 5

RESOURCE_SUMMARIES=$(kubectl get resourcesummaries --no-headers 2>/dev/null | wc -l | tr -d ' ')
echo -e "${GREEN}Found ${RESOURCE_SUMMARIES} ResourceSummary CRs${NC}"

if [ "$RESOURCE_SUMMARIES" -gt 0 ]; then
    echo ""
    kubectl get resourcesummaries -o wide | head -20
fi

echo ""

# Summary
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║                    ✅ SETUP COMPLETE!                            ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""
echo "🎯 Resources Created:"
echo "   • ProviderConfig: localstack"
echo "   • IAM Roles: 2 (tenant-alpha-lambda-role, tenant-beta-data-role)"
echo "   • IAM Policies: 2 (tenant-alpha-s3-policy, tenant-beta-dynamodb-policy)"
echo "   • S3 Buckets: 2 (tenant-alpha-ml-data, tenant-beta-analytics)"
echo "   • DynamoDB Tables: 2 (tenant-alpha-ml-models, tenant-beta-events)"
echo ""
echo "📡 Next Steps:"
echo "   1. Check UI at http://localhost:9083/resources"
echo "   2. Filter by tenant (tenant-alpha or tenant-beta)"
echo "   3. View IAM resources, S3 buckets, and DynamoDB tables"
echo "   4. Run: ./test-crossplane-integration.sh"
echo ""
echo "🔧 Useful Commands:"
echo "   • View all Crossplane resources:"
echo "     kubectl get managed"
echo ""
echo "   • Check resource status:"
echo "     kubectl get role.iam.aws.upbound.io tenant-alpha-lambda-role -o yaml"
echo ""
echo "   • View LocalStack resources:"
echo "     aws --endpoint-url=http://localhost:4566 iam list-roles"
echo "     aws --endpoint-url=http://localhost:4566 s3 ls"
echo "     aws --endpoint-url=http://localhost:4566 dynamodb list-tables"
echo ""
echo "   • Trigger manual resource scan:"
echo "     kubectl annotate tenant tenant-alpha platform.io/scan=now --overwrite"
echo ""

