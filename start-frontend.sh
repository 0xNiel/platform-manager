#!/bin/bash

# Start the Vue.js frontend dev server for Phase 5 UI testing

cd "$(dirname "$0")/web"

echo "🚀 Starting Frontend Dev Server for Phase 5 Testing"
echo ""
echo "The Troubleshooting view will be available at:"
echo "  👉 http://localhost:9083/troubleshooting"
echo ""
echo "Other views:"
echo "  • Dashboard: http://localhost:9083/"
echo "  • Resources: http://localhost:9083/resources"
echo "  • IAM Drift: http://localhost:9083/iam"
echo ""
echo "Press Ctrl+C to stop"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

npm run serve

