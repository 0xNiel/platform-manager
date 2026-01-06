#!/bin/bash
# Phase 6 Terminal - Setup and Testing Script
# This script will set up everything needed to test the terminal feature

set -e

echo "=========================================="
echo "Phase 6: Web Terminal - Setup & Testing"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Step 1: Build the toolbox image
echo -e "${YELLOW}Step 1: Building toolbox image...${NC}"
docker build -t platform-manager-toolbox:dev -f Dockerfile.toolbox .
echo -e "${GREEN}✓ Toolbox image built${NC}"
echo ""

# Step 2: Load toolbox image into Kind
echo -e "${YELLOW}Step 2: Loading toolbox image into Kind...${NC}"
kind load docker-image platform-manager-toolbox:dev --name platform-manager
echo -e "${GREEN}✓ Toolbox image loaded into Kind${NC}"
echo ""

# Step 3: Create toolbox namespace
echo -e "${YELLOW}Step 3: Creating toolbox-sessions namespace...${NC}"
kubectl create namespace toolbox-sessions --dry-run=client -o yaml | kubectl apply -f -
echo -e "${GREEN}✓ Namespace created${NC}"
echo ""

# Step 4: Create ServiceAccount
echo -e "${YELLOW}Step 4: Creating toolbox-session ServiceAccount...${NC}"
cat <<EOF | kubectl apply -f -
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: toolbox-session
  namespace: toolbox-sessions
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: toolbox-session
rules:
  # Allow listing/reading resources across cluster
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps", "namespaces"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets", "daemonsets"]
    verbs: ["get", "list", "watch"]
  # Crossplane resources
  - apiGroups: ["*.upbound.io", "*.crossplane.io"]
    resources: ["*"]
    verbs: ["get", "list", "watch"]
  # ArgoCD resources
  - apiGroups: ["argoproj.io"]
    resources: ["applications", "appprojects"]
    verbs: ["get", "list", "watch"]
  # Platform Manager CRDs
  - apiGroups: ["platform.io"]
    resources: ["*"]
    verbs: ["get", "list", "watch"]
  # Logs
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: toolbox-session
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: toolbox-session
subjects:
  - kind: ServiceAccount
    name: toolbox-session
    namespace: toolbox-sessions
EOF
echo -e "${GREEN}✓ ServiceAccount and RBAC created${NC}"
echo ""

# Step 5: Grant manager permissions for toolbox namespace
echo -e "${YELLOW}Step 5: Granting manager permissions for toolbox namespace...${NC}"
cat <<EOF | kubectl apply -f -
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: platform-manager-toolbox
  namespace: toolbox-sessions
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["create", "get", "list", "watch", "delete"]
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create", "get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: platform-manager-toolbox
  namespace: toolbox-sessions
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: platform-manager-toolbox
subjects:
  - kind: ServiceAccount
    name: platform-manager-controller-manager
    namespace: platform-manager-system
EOF
echo -e "${GREEN}✓ Manager permissions granted${NC}"
echo ""

# Step 6: Verify setup
echo -e "${YELLOW}Step 6: Verifying setup...${NC}"
echo "  - Checking namespace..."
kubectl get namespace toolbox-sessions
echo "  - Checking ServiceAccount..."
kubectl get serviceaccount toolbox-session -n toolbox-sessions
echo "  - Checking ClusterRole..."
kubectl get clusterrole toolbox-session
echo "  - Checking ClusterRoleBinding..."
kubectl get clusterrolebinding toolbox-session
echo -e "${GREEN}✓ All resources verified${NC}"
echo ""

echo -e "${GREEN}=========================================="
echo "Setup Complete!"
echo "==========================================${NC}"
echo ""
echo "Next steps:"
echo "  1. Build the manager: make build"
echo "  2. Run with terminal enabled:"
echo "     ./bin/manager --enable-terminal=true"
echo ""
echo "Or use the test script:"
echo "  ./test-phase6-terminal.sh"
echo ""

