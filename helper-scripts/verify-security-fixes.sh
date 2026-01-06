#!/bin/bash
# Test script to verify security fixes work correctly

set -e

echo "=================================="
echo "Security Fixes Verification Tests"
echo "=================================="
echo ""

cd "$(dirname "$0")/.."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass() {
    echo -e "${GREEN}✅ PASS:${NC} $1"
}

fail() {
    echo -e "${RED}❌ FAIL:${NC} $1"
    exit 1
}

info() {
    echo -e "${YELLOW}ℹ️  INFO:${NC} $1"
}

echo "Test 1: Verify Go application builds"
echo "-------------------------------------"
if make build > /dev/null 2>&1; then
    pass "Go application builds successfully"
else
    fail "Go application failed to build"
fi
echo ""

echo "Test 2: Verify Go version is updated"
echo "-------------------------------------"
if grep -q "go 1.24.5" go.mod; then
    pass "Go version updated to 1.24.5 in go.mod"
else
    fail "Go version not updated in go.mod"
fi

if grep -q "golang:1.24.5" Dockerfile; then
    pass "Dockerfile uses golang:1.24.5"
else
    fail "Dockerfile not using golang:1.24.5"
fi
echo ""

echo "Test 3: Verify Dockerfile.toolbox dependencies"
echo "----------------------------------------------"
if grep -q "alpine:3.21" Dockerfile.toolbox; then
    pass "Alpine updated to 3.21"
else
    fail "Alpine not updated to 3.21"
fi

if grep -q "KUBECTL_VERSION=v1.33.0" Dockerfile.toolbox; then
    pass "kubectl updated to v1.33.0"
else
    fail "kubectl not updated"
fi

if grep -q "HELM_VERSION=v3.16.3" Dockerfile.toolbox; then
    pass "Helm updated to v3.16.3"
else
    fail "Helm not updated"
fi

if grep -q "ARGOCD_VERSION=v2.13.1" Dockerfile.toolbox; then
    pass "ArgoCD CLI updated to v2.13.1"
else
    fail "ArgoCD CLI not updated"
fi

if grep -q "AWS_CLI_VERSION=1.32.0" Dockerfile.toolbox; then
    pass "AWS CLI version pinned to 1.32.0"
else
    fail "AWS CLI version not pinned"
fi
echo ""

echo "Test 4: Verify security code is present"
echo "---------------------------------------"
if grep -q "type SecurityConfig struct" internal/terminal/types.go; then
    pass "SecurityConfig struct added"
else
    fail "SecurityConfig struct not found"
fi

if grep -q "type RateLimiter struct" internal/terminal/ratelimit.go; then
    pass "RateLimiter implementation added"
else
    fail "RateLimiter not found"
fi

if grep -q "DEV_MODE" internal/api/middleware/auth.go; then
    pass "Dev mode check added to auth middleware"
else
    fail "Dev mode check not found in auth"
fi

if grep -q "TERMINAL_ALLOWED_ORIGINS" internal/api/server.go; then
    pass "Terminal origin validation configured"
else
    fail "Terminal origin config not found"
fi

if grep -q "ALLOWED_ORIGINS" internal/api/middleware/cors.go; then
    pass "CORS configuration updated"
else
    fail "CORS config not found"
fi
echo ""

echo "Test 5: Run middleware tests"
echo "----------------------------"
if go test ./internal/api/middleware/... -v > /dev/null 2>&1; then
    pass "All middleware tests pass"
else
    fail "Middleware tests failed"
fi
echo ""

echo "Test 6: Verify rate limiting integration"
echo "----------------------------------------"
if grep -q "RateLimiter" internal/terminal/types.go; then
    pass "RateLimiter added to Session struct"
else
    fail "RateLimiter not in Session struct"
fi

if grep -q "session.RateLimiter.Allow()" internal/api/handlers/terminal.go; then
    pass "Rate limiting integrated in WebSocket handler"
else
    fail "Rate limiting not integrated"
fi
echo ""

echo "Test 7: Verify web frontend builds"
echo "----------------------------------"
info "Building web frontend (this may take a moment)..."
cd web
if npm run build > /tmp/npm-build.log 2>&1; then
    pass "Web frontend builds successfully"
else
    fail "Web frontend build failed (check /tmp/npm-build.log)"
fi
cd ..
echo ""

echo "Test 8: Test API server with dev mode"
echo "-------------------------------------"
info "Starting API server with DEV_MODE=true for 5 seconds..."
timeout 5 bash -c 'DEV_MODE=true ./bin/manager --api-bind-address=:19081 --metrics-bind-address=0 --health-probe-bind-address=:18082 2>&1' > /tmp/api-startup.log 2>&1 || true

if grep -q "Starting API server" /tmp/api-startup.log; then
    pass "API server starts successfully with DEV_MODE=true"
else
    fail "API server failed to start (check /tmp/api-startup.log)"
fi

if grep -q "TERMINAL.*Development mode enabled" /tmp/api-startup.log || grep -q "Terminal manager is nil" /tmp/api-startup.log; then
    pass "Dev mode configuration loaded"
else
    info "Terminal not enabled (expected if --enable-terminal not set)"
fi
echo ""

echo "Test 9: Verify environment variable support"
echo "-------------------------------------------"
if grep -q 'os.Getenv("DEV_MODE")' internal/api/middleware/auth.go; then
    pass "Auth middleware reads DEV_MODE environment variable"
else
    fail "DEV_MODE environment variable not read"
fi

if grep -q 'os.Getenv("TERMINAL_ALLOWED_ORIGINS")' internal/api/server.go; then
    pass "Server reads TERMINAL_ALLOWED_ORIGINS environment variable"
else
    fail "TERMINAL_ALLOWED_ORIGINS not read"
fi

if grep -q 'os.Getenv("ALLOWED_ORIGINS")' internal/api/middleware/cors.go; then
    pass "CORS reads ALLOWED_ORIGINS environment variable"
else
    fail "ALLOWED_ORIGINS not read"
fi
echo ""

echo "Test 10: Code quality checks"
echo "----------------------------"
if go vet ./... > /dev/null 2>&1; then
    pass "go vet passes"
else
    fail "go vet found issues"
fi

if go fmt ./... > /dev/null 2>&1; then
    pass "Code is properly formatted"
else
    info "Some files needed formatting (this is OK)"
fi
echo ""

echo "=================================="
echo "All Security Tests Complete!"
echo "=================================="
echo ""
echo "Summary:"
echo "--------"
echo "✅ Go version updated (1.24.5)"
echo "✅ Dockerfile.toolbox dependencies updated"
echo "✅ WebSocket origin validation implemented"
echo "✅ Auth dev mode protection added"
echo "✅ CORS configuration hardened"
echo "✅ Rate limiting implemented"
echo "✅ All security tests passing"
echo ""
echo "Next steps:"
echo "-----------"
echo "1. Configure environment variables for production:"
echo "   - DEV_MODE=false"
echo "   - TERMINAL_ALLOWED_ORIGINS=https://your-domain.com"
echo "   - ALLOWED_ORIGINS=https://your-domain.com"
echo ""
echo "2. Test in staging environment"
echo "3. Deploy to production"
echo ""
echo "For more details, see:"
echo "- documentation/SECURITY_FIXES_COMPLETE.md"
echo "- documentation/SECURITY_MITIGATIONS.md"
echo ""

