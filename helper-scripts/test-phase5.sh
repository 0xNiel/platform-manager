#!/bin/bash

# Test script for Phase 5: Troubleshooting & Rule Engine

set -e

API_BASE="http://localhost:9080/api/v1"

echo "============================================"
echo "Phase 5: Troubleshooting & Rule Engine Tests"
echo "============================================"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass_count=0
fail_count=0

# Test function
test_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local expected_status="${4:-200}"
    
    echo -n "Testing: $name... "
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "$API_BASE$endpoint")
    elif [ "$method" = "POST" ]; then
        response=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE$endpoint")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "$expected_status" ]; then
        echo -e "${GREEN}✓ PASS${NC} (HTTP $http_code)"
        ((pass_count++))
        if [ -n "$body" ] && [ "$body" != "null" ]; then
            echo "$body" | jq -C '.' 2>/dev/null || echo "$body"
        fi
    else
        echo -e "${RED}✗ FAIL${NC} (Expected $expected_status, got $http_code)"
        ((fail_count++))
        echo "$body"
    fi
    echo ""
}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1. Troubleshooting API Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_endpoint "Get platform troubleshooting summary" "GET" "/troubleshooting/summary"
test_endpoint "Get all findings" "GET" "/troubleshooting/findings"
test_endpoint "Get rules list" "GET" "/troubleshooting/rules"
test_endpoint "Trigger manual scan" "POST" "/troubleshooting/scan" "202"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2. Filtered Findings Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_endpoint "Get critical findings only" "GET" "/troubleshooting/findings?severity=critical"
test_endpoint "Get findings for tenant-alpha" "GET" "/troubleshooting/findings?tenant=tenant-alpha"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3. Tenant-Specific Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_endpoint "Get tenant-alpha findings" "GET" "/troubleshooting/tenants/tenant-alpha"
test_endpoint "Get tenant-beta findings" "GET" "/troubleshooting/tenants/tenant-beta"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "4. Integration Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if any findings were detected
echo "Checking for detected issues..."
findings_count=$(curl -s "$API_BASE/troubleshooting/summary" | jq -r '.totalFindings')

if [ "$findings_count" != "null" ] && [ "$findings_count" != "" ]; then
    echo -e "${GREEN}✓${NC} Found $findings_count total findings"
    ((pass_count++))
    
    # Show severity breakdown
    critical=$(curl -s "$API_BASE/troubleshooting/summary" | jq -r '.critical')
    high=$(curl -s "$API_BASE/troubleshooting/summary" | jq -r '.high')
    medium=$(curl -s "$API_BASE/troubleshooting/summary" | jq -r '.medium')
    low=$(curl -s "$API_BASE/troubleshooting/summary" | jq -r '.low')
    
    echo "  Critical: $critical"
    echo "  High: $high"
    echo "  Medium: $medium"
    echo "  Low: $low"
else
    echo -e "${YELLOW}⚠${NC} No findings detected (might be normal if no issues exist)"
fi
echo ""

# Check if rules are registered
echo "Checking registered rules..."
rules_count=$(curl -s "$API_BASE/troubleshooting/rules" | jq -r '.total')

if [ "$rules_count" != "null" ] && [ "$rules_count" != "" ] && [ "$rules_count" -gt 0 ]; then
    echo -e "${GREEN}✓${NC} Found $rules_count registered rules"
    ((pass_count++))
    
    # List rules
    curl -s "$API_BASE/troubleshooting/rules" | jq -C '.rules[] | {id, name, severity}'
else
    echo -e "${RED}✗${NC} No rules registered!"
    ((fail_count++))
fi
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "Passed: ${GREEN}$pass_count${NC}"
echo -e "Failed: ${RED}$fail_count${NC}"
echo ""

if [ $fail_count -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi

