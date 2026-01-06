#!/bin/bash

# Phase 6: Web Terminal - Start Manager with Terminal Enabled
# This script starts the Platform Manager with terminal features enabled

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║      Phase 6: Starting Platform Manager with Terminal          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if already running
if lsof -Pi :9080 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo -e "${YELLOW}Warning: Port 9080 is already in use${NC}"
    echo -e "${YELLOW}Stopping existing manager...${NC}"
    pkill -f "bin/manager" || true
    sleep 2
fi

# Check if Kind cluster is running
if ! kubectl cluster-info > /dev/null 2>&1; then
    echo -e "${YELLOW}Warning: Kind cluster not accessible${NC}"
    echo -e "${YELLOW}Starting Kind cluster...${NC}"
    kind create cluster --name platform-manager --config hack/kind-config.yaml || true
fi

# Ensure toolbox-sessions namespace exists
echo -e "${GREEN}Ensuring toolbox-sessions namespace exists...${NC}"
kubectl create namespace toolbox-sessions --dry-run=client -o yaml | kubectl apply -f - || true

# Build the latest manager
echo -e "${GREEN}Building manager...${NC}"
make build

# Start the manager with terminal enabled
echo -e "${GREEN}Starting manager with terminal enabled...${NC}"
echo ""
echo -e "${YELLOW}Terminal Configuration:${NC}"
echo -e "  Enabled: ${GREEN}true${NC}"
echo -e "  Namespace: ${GREEN}toolbox-sessions${NC}"
echo -e "  Image: ${GREEN}platform-manager-toolbox:dev${NC}"
echo -e "  Idle Timeout: ${GREEN}10m${NC}"
echo -e "  Max Sessions: ${GREEN}20${NC}"
echo ""
echo -e "${BLUE}Manager starting... (output will be logged to manager-phase6.log)${NC}"
echo ""

# Run the manager with terminal flags
./bin/manager \
    --metrics-bind-address=:8081 \
    --health-probe-bind-address=:8082 \
    --leader-elect=false \
    --api-addr=:9080 \
    --enable-terminal=true \
    --terminal-namespace=toolbox-sessions \
    --terminal-image=platform-manager-toolbox:dev \
    --terminal-idle-timeout=10m \
    --terminal-max-sessions=20 \
    2>&1 | tee manager-phase6.log

