#!/bin/bash

# Start Phase 5 - Platform Manager with Rule Evaluator

set -e

echo "🚀 Starting Platform Manager with Phase 5 (Rule Engine)..."
echo ""

cd /Users/odnielgonzalez/Documents/2-WorkStuff--ai-platform-in-go/platform-manager

# Check if binary exists
if [ ! -f "./bin/manager" ]; then
    echo "❌ Binary not found. Building..."
    go build -o bin/manager ./cmd/main.go
fi

echo "✅ Starting manager with rule evaluation enabled..."
echo "   - API: http://localhost:9080"
echo "   - Metrics: http://localhost:8443"
echo "   - Rule Evaluation Interval: 2 minutes"
echo ""
echo "📊 Features enabled:"
echo "   ✓ Health Aggregation (Phase 1)"
echo "   ✓ Resource Scanner (Phase 2)"
echo "   ✓ Actions API (Phase 3)"
echo "   ✓ IAM Drift Detection (Phase 4)"
echo "   ✓ Rule Engine & Troubleshooting (Phase 5) ← NEW"
echo ""
echo "Press Ctrl+C to stop"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

./bin/manager \
  --api-bind-address=:9080 \
  --metrics-bind-address=:8443 \
  --health-probe-bind-address=:8081 \
  --iam-drift-scan-interval=5m \
  --rule-eval-interval=2m \
  --prometheus-url=http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090

