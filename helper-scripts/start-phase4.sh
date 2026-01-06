#!/bin/bash
# start-phase4.sh - Start Platform Manager for Phase 4 Testing

echo "========================================"
echo "  Phase 4: IAM Drift Detection Testing"
echo "========================================"
echo ""

# Check prerequisites
echo "✓ Checking prerequisites..."

# LocalStack
if ! curl -s http://localhost:4566/_localstack/health > /dev/null 2>&1; then
    echo "❌ LocalStack is not running!"
    echo "   Start it with: localstack start"
    exit 1
fi
echo "  ✓ LocalStack is running"

# Kubernetes
if ! kubectl cluster-info > /dev/null 2>&1; then
    echo "❌ Kubernetes cluster not accessible!"
    exit 1
fi
echo "  ✓ Kubernetes cluster accessible"

# IAM Resources
IAM_COUNT=$(kubectl get roles.iam.aws.upbound.io,policies.iam.aws.upbound.io -A 2>/dev/null | grep -c True || echo 0)
echo "  ✓ Found $IAM_COUNT IAM resources"

echo ""
echo "✓ Starting Platform Manager..."
echo "  API: http://localhost:9080"
echo "  Logs: ./manager-phase4.log"
echo ""

# Set environment
export AWS_ENDPOINT=http://localhost:4566
export AWS_REGION=us-east-1

# Start manager
exec ./bin/manager \
  --api-bind-address=:9080 \
  --health-probe-bind-address=:8082 \
  --metrics-bind-address=0 \
  --iam-drift-scan-interval=1m \
  2>&1 | tee manager-phase4.log

