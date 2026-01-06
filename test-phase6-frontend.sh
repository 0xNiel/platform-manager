#!/bin/bash
# Quick test script for Phase 6 terminal frontend

echo "=========================================="
echo "Phase 6: Testing Terminal Frontend"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Backend Manager:${NC}"
if ps aux | grep "[b]in/manager" > /dev/null; then
    echo -e "${GREEN}✓ Running on :9080${NC}"
else
    echo -e "${YELLOW}✗ Not running - start with: ./bin/manager --enable-terminal=true${NC}"
fi
echo ""

echo -e "${BLUE}Frontend Dev Server:${NC}"
if lsof -ti:9083 > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Running on :9083${NC}"
else
    echo -e "${YELLOW}✗ Not running - start with: cd web && npm run serve${NC}"
fi
echo ""

echo -e "${BLUE}Test Instructions:${NC}"
echo "1. Open browser: http://localhost:9083"
echo "2. Look for 💻 button in bottom-right corner"
echo "3. Click button to open terminal drawer"
echo "4. Click 'New Session' and select a tenant"
echo "5. Wait for connection (green dot appears)"
echo "6. Type commands: whoami, kubectl get ns, etc."
echo "7. Navigate between pages - terminal stays open!"
echo "8. Resize by dragging top edge of terminal"
echo ""

echo -e "${BLUE}Quick Browser Test:${NC}"
if command -v open > /dev/null 2>&1; then
    echo "Opening browser..."
    open http://localhost:9083
elif command -v xdg-open > /dev/null 2>&1; then
    echo "Opening browser..."
    xdg-open http://localhost:9083
else
    echo "Open manually: http://localhost:9083"
fi
echo ""

echo -e "${GREEN}=========================================="
echo "Ready to test!"
echo "==========================================${NC}"

