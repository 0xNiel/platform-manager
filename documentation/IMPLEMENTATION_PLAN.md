# Platform Manager - Implementation Plan

> A comprehensive guide to building "ArgoCD + Crossplane + IAM Doctor + Ops Toolbox" with a Go backend using Kubebuilder

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Decisions](#architecture-decisions)
3. [Technology Stack](#technology-stack)
4. [Project Structure](#project-structure)
5. [Phase 0: Development Environment & Test Infrastructure](#phase-0-development-environment--test-infrastructure) ✅ **COMPLETE**
6. [Phase 1: Backend Skeleton & Global Overview](#phase-1-backend-skeleton--global-overview) ✅ **COMPLETE**
7. [Phase 2: Drill-down & Resource Detail](#phase-2-drill-down--resource-detail) ⏳ **NEXT**
8. [Phase 3: Actions (Mini Argo + Crossplane Controls)](#phase-3-actions-mini-argo--crossplane-controls)
9. [Phase 4: IAM Drift Module](#phase-4-iam-drift-module)
10. [Phase 5: Troubleshooting & Rule Engine](#phase-5-troubleshooting--rule-engine)
11. [Phase 6: Web Terminal](#phase-6-web-terminal)
12. [Deployment & CI/CD](#deployment--cicd)
13. [Environment Configuration](#environment-configuration)

---

## Executive Summary

This platform manager will provide a unified control plane for:
- **Crossplane resources** (Compositions, Claims, XRs, Providers)
- **ArgoCD applications** (sync status, health, operations)
- **IAM drift detection** (comparing Crossplane spec → status → actual AWS state)
- **Resource health aggregation** (Ready/Failed/Waiting/Unknown/Paused states)
- **Web terminal** for secure, audited kubectl/aws CLI access
- **Multi-tenant views** with role-based access control

**Core Principle**: Start from a 30,000-foot view, drill down to individual resources.

### Key Configuration

| Aspect | Decision |
|--------|----------|
| **Frontend** | Vue 3 + single-spa (MFE compatible) |
| **Backend** | Go + Kubebuilder |
| **Auth** | Handled by OAuth2Proxy at gateway; ACR-based Access Control |
| **Cluster** | Single cluster per environment (5 environments) |
| **AWS Testing** | LocalStack Community |
| **Metrics** | Prometheus (already deployed) |

---

## Architecture Decisions

### Decision 1: Single Binary ✅

**Decision**: Start with a **single binary** using Kubebuilder's manager pattern. The manager runs both reconcilers AND an HTTP server.

```go
// manager runs both controllers and HTTP server
mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
    return httpServer.Start(ctx)
}))
```

**Rationale**: Simpler deployment, shared cache. Split later if scaling becomes an issue.

---

### Decision 2: Data Storage Strategy ✅

**Decision**: Progressive approach:
- **Phase 1-2**: Live K8s queries with in-memory aggregation cache
- **Phase 3+**: Introduce `ResourceSummary` and `TenantHealth` CRDs to persist state
- **Future**: Add SQLite/BadgerDB for historical timeline data if needed

---

### Decision 3: Mini ArgoCD UI Scope ✅

**Decision**: Don't recreate Argo's full UI. Instead:
- Integrate with Argo's **API/CRDs directly** for sync/refresh operations
- Show **summary views** only (sync status, health, last sync time)
- **Deep link to Argo UI** for detailed app debugging
- Focus on **platform-level context** that Argo doesn't provide (cross-tenant, Crossplane correlation)

---

### Decision 4: Terminal Security - Toolbox Pod Pattern ✅

**Decision**: Implement ephemeral "toolbox pods":
1. Backend spawns ephemeral pod per terminal session
2. Pod uses tenant-scoped ServiceAccount with minimal RBAC
3. AWS access via IRSA with tenant-specific role assumption
4. All commands logged via shell wrapper
5. Pod auto-deleted after idle timeout (10 min)
6. Only `admin` and `infra-engineer` roles can access terminal

---

### Decision 5: Authentication ✅

**Decision**: The gateway handles auth via **OAuth2Proxy**. The Platform Manager MFE doesn't need to implement auth—it receives user context from the gateway.

- Backend trusts headers set by OAuth2Proxy (e.g., `X-Auth-Request-User`, `X-Auth-Request-Groups`)
- Gateway uses **ACR-Based Access Control** (Access Control Rules)
- Role extraction from gateway-provided headers

---

### Decision 6: LocalStack Community Limitations ✅

**Available LocalStack Community APIs** (for local testing):

| Category | Services |
|----------|----------|
| **Core** | IAM, STS, S3, S3Control |
| **Compute** | Lambda, EC2 |
| **Database** | DynamoDB, DynamoDB Streams, Redshift, OpenSearch, ES |
| **Messaging** | SQS, SNS, Kinesis, Firehose, Events |
| **Networking** | Route53, Route53Resolver, API Gateway |
| **Storage** | S3, S3Control |
| **Security** | IAM, KMS, Secrets Manager, ACM |
| **Monitoring** | CloudWatch, Logs |
| **Config** | Config, SSM, CloudFormation |
| **Other** | Step Functions, SWF, SES, Transcribe, Support, Scheduler, Resource Groups |

**Note**: IAM policy simulation has quirks in Community edition. For drift testing, we'll:
1. Create IAM resources via Crossplane pointing to LocalStack
2. Use `awslocal` to manually drift policies (add/remove statements)
3. Verify drift detection logic

---

## Technology Stack

### Backend

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | **Go 1.22+** | Required for Kubebuilder, great K8s ecosystem |
| Controller Framework | **Kubebuilder v4** | Standard for K8s controllers |
| HTTP Framework | **Chi** or **Echo** | Lightweight, middleware-friendly |
| K8s Client | **controller-runtime client** | Caching, watches built-in |
| AWS SDK | **aws-sdk-go-v2** | Modern, context-aware |
| Metrics Queries | **prometheus/client_golang** | Query existing Prometheus |
| WebSocket | **gorilla/websocket** | For terminal sessions |
| Logging | **zap** (via controller-runtime) | Structured, fast |
| Testing | **ginkgo/gomega** + **envtest** | Kubebuilder standard |

### Frontend

| Component | Technology | Notes |
|-----------|------------|-------|
| Framework | **Vue 3** with TypeScript | Gateway requirement |
| MFE Framework | **single-spa** | Lifecycle methods (bootstrap, mount, unmount) |
| Bundler | **Webpack 5** | SystemJS-compatible output |
| State Management | **Pinia** | Vue 3 standard |
| HTTP Client | **axios** or **ky** | API communication |
| UI Components | **PrimeVue** or **Naive UI** | Vue 3 compatible |
| Terminal | **xterm.js** | WebSocket terminal |
| Charts | **Chart.js** or **ECharts** | Vue 3 compatible |

### Crossplane Providers to Watch

The controller will watch resources from these providers:

| Provider Family | Providers |
|-----------------|-----------|
| **AWS** | Full AWS family (IAM, S3, RDS, Lambda, etc.) |
| **Custom** | KnowledgeBases provider |
| **Infrastructure** | Ansible, OpenTofu, Terraform |
| **GitOps** | ArgoCD |
| **Observability** | Grafana |
| **Orchestration** | Helm, Kubernetes |
| **Secrets** | Vault |

### Development & Testing

| Component | Technology |
|-----------|------------|
| Local Cluster | Kind |
| GitOps | ArgoCD |
| Crossplane | Crossplane + Multiple Providers |
| AWS Mock | LocalStack Community |
| Build | Make + Ko (for images) |
| CI/CD | GitHub Actions (assumed) |

---

## Project Structure

```
platform-manager/
├── api/
│   └── v1alpha1/
│       ├── tenant_types.go           # Tenant CRD
│       ├── resourcesummary_types.go  # ResourceSummary CRD
│       ├── tenanthealth_types.go     # TenantHealth CRD
│       ├── platformhealth_types.go   # PlatformHealth CRD (singleton)
│       ├── groupversion_info.go
│       └── zz_generated.deepcopy.go
├── cmd/
│   └── main.go                       # Entrypoint
├── internal/
│   ├── controller/
│   │   ├── tenant_controller.go      # Watches Tenant CRD
│   │   ├── crossplane_watcher.go     # Watches XRDs, XRs, Claims
│   │   ├── argo_watcher.go           # Watches ArgoCD Applications
│   │   └── aggregator.go             # Builds health summaries
│   ├── api/
│   │   ├── server.go                 # HTTP server setup
│   │   ├── handlers/
│   │   │   ├── health.go             # /api/health/*
│   │   │   ├── tenants.go            # /api/tenants/*
│   │   │   ├── resources.go          # /api/resources/*
│   │   │   ├── iam.go                # /api/iam/*
│   │   │   ├── actions.go            # /api/actions/* (sync, pause, delete)
│   │   │   └── terminal.go           # /api/terminal (WebSocket)
│   │   └── middleware/
│   │       ├── auth.go               # Role extraction from gateway headers
│   │       └── audit.go              # Request logging
│   ├── iam/
│   │   ├── drift_checker.go          # Core drift detection logic
│   │   ├── aws_client.go             # AWS IAM API wrapper
│   │   └── types.go                  # DriftResult, Severity, etc.
│   ├── health/
│   │   ├── normalizer.go             # Normalize status to Ready/Failed/etc.
│   │   ├── aggregator.go             # Aggregate per-tenant and global
│   │   └── cache.go                  # In-memory health cache
│   ├── rules/
│   │   ├── engine.go                 # Rule evaluation engine
│   │   ├── rule.go                   # Rule interface
│   │   └── rules/
│   │       ├── iam_missing.go        # "XR failed due to IAM"
│   │       ├── crashloop.go          # "Pod in CrashLoopBackOff"
│   │       └── paused_but_syncing.go # "Paused but Argo still syncing"
│   ├── terminal/
│   │   ├── manager.go                # Toolbox pod lifecycle
│   │   ├── session.go                # WebSocket session handling
│   │   └── recorder.go               # Command audit logging
│   └── metrics/
│       └── prometheus.go             # Prometheus query client
├── config/
│   ├── crd/                          # Generated CRD YAMLs
│   ├── rbac/                         # RBAC for controller
│   ├── manager/                      # Deployment manifest
│   └── samples/                      # Sample Tenant CRs
├── hack/
│   ├── kind-config.yaml              # Kind cluster config
│   ├── crossplane/
│   │   ├── provider-aws.yaml         # AWS provider installation
│   │   └── providerconfig-localstack.yaml
│   └── seed-tenants/
│       ├── namespaces.yaml
│       ├── tenant-alpha.yaml         # Fake tenant resources
│       ├── tenant-beta.yaml
│       ├── iam-resources.yaml        # IAM roles/policies for testing
│       ├── argo-apps.yaml
│       └── failed-resources.yaml
├── web/                              # Vue 3 single-spa MFE
│   ├── package.json
│   ├── vue.config.js                 # Webpack configuration for single-spa
│   ├── src/
│   │   ├── main.ts                   # single-spa lifecycle exports
│   │   ├── App.vue
│   │   ├── router/
│   │   ├── stores/                   # Pinia stores
│   │   ├── components/
│   │   ├── views/
│   │   └── api/                      # API client
│   └── ...
├── test/
│   ├── e2e/
│   │   ├── suite_test.go
│   │   └── tenant_test.go
│   └── testdata/
│       └── fixtures/
├── Dockerfile
├── Dockerfile.toolbox                # Toolbox image for terminal
├── Makefile
├── go.mod
├── go.sum
├── PROJECT                           # Kubebuilder project file
├── .gitignore
└── README.md
```

---

## Phase 0: Development Environment & Test Infrastructure ✅ COMPLETE

**Goal**: Create a reproducible local development environment with Kind, ArgoCD, Crossplane, and LocalStack.

**Status**: ✅ **COMPLETE** (Completed December 2024)

**What was delivered**:
- Git repository initialized with comprehensive `.gitignore`
- Makefile with 40+ targets for development workflow
- Kind cluster configuration (single node, ports 9080-9082)
- Crossplane v1.20.0 with AWS IAM Provider pointing to LocalStack
- Seed tenant resources (alpha, beta, gamma) with deployments, IAM roles/policies, ArgoCD apps
- Vue 3 single-spa frontend skeleton with Dashboard, Tenants, Resources, IAM Drift views
- Dockerfiles for manager and toolbox images
- README with quick start guide

### 0.1 Initialize Git Repository (FIRST STEP)

```bash
# Initialize git repository
git init
git branch -M main

# Create .gitignore
cat > .gitignore << 'EOF'
# Binaries
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of go coverage tool
*.out
cover.out

# Go workspace file
go.work

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Kubernetes
*.kubeconfig

# LocalStack
.localstack/
volume/

# Node modules (frontend)
web/node_modules/
web/dist/

# Environment files
.env
.env.local
*.env

# Temporary files
tmp/
temp/

# Build artifacts
/platform-manager
EOF

# Create initial commit
git add .gitignore
git commit -m "Initial commit: add .gitignore"
```

### 0.2 Prerequisites

Ensure these are installed on your Mac:

```bash
# Required tools
brew install kind
brew install kubectl
brew install helm
brew install ko               # For building Go images
brew install argocd          # ArgoCD CLI
brew install awscli          # AWS CLI (for awslocal wrapper)
pip3 install localstack awscli-local  # LocalStack + awslocal

# Optional but recommended
brew install stern           # Log tailing
brew install k9s             # K8s TUI
brew install jq              # JSON processing

# Frontend development
brew install node            # Node.js (LTS)
npm install -g @vue/cli      # Vue CLI
```

### 0.3 Makefile

Create the following `Makefile`:

```makefile
# ==============================================================================
# Platform Manager Makefile
# ==============================================================================

# Variables
KIND_CLUSTER_NAME ?= platform-manager
LOCALSTACK_ENDPOINT ?= http://host.docker.internal:4566
AWS_REGION ?= us-east-1

# Image variables
IMG ?= platform-manager:dev
TOOLBOX_IMG ?= platform-manager-toolbox:dev
REGISTRY ?= your-registry.azurecr.io
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Go variables
GOBIN ?= $(shell go env GOPATH)/bin
ENVTEST_VERSION ?= release-0.17
ENVTEST_K8S_VERSION ?= 1.29.0

# ==============================================================================
# Git
# ==============================================================================

##@ Git

.PHONY: git-init
git-init: ## Initialize git repository with .gitignore
	@if [ ! -d .git ]; then \
		git init && \
		git branch -M main && \
		echo "✅ Git repository initialized"; \
	else \
		echo "Git repository already exists"; \
	fi

# ==============================================================================
# Development Environment
# ==============================================================================

##@ Development Environment

.PHONY: dev-up
dev-up: kind-create install-argocd install-crossplane setup-localstack-provider seed-tenants ## Create full dev environment
	@echo "✅ Development environment ready!"
	@echo ""
	@echo "📋 Quick Reference:"
	@echo "   ArgoCD UI: https://localhost:8080"
	@echo "   ArgoCD Password: $$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d)"
	@echo "   LocalStack: $(LOCALSTACK_ENDPOINT)"
	@echo ""
	@echo "🔧 Useful commands:"
	@echo "   make argocd-port-forward  # Access ArgoCD UI"
	@echo "   make localstack-health    # Check LocalStack"
	@echo "   kubectl get providers     # Check Crossplane providers"

.PHONY: dev-down
dev-down: ## Tear down dev environment
	kind delete cluster --name $(KIND_CLUSTER_NAME)
	@echo "✅ Development environment deleted"

.PHONY: dev-reset
dev-reset: dev-down dev-up ## Reset dev environment

# ==============================================================================
# Kind Cluster
# ==============================================================================

##@ Kind Cluster

.PHONY: kind-create
kind-create: ## Create Kind cluster
	@if kind get clusters | grep -q $(KIND_CLUSTER_NAME); then \
		echo "Cluster $(KIND_CLUSTER_NAME) already exists"; \
	else \
		kind create cluster --name $(KIND_CLUSTER_NAME) --config hack/kind-config.yaml; \
	fi
	kubectl cluster-info --context kind-$(KIND_CLUSTER_NAME)

.PHONY: kind-delete
kind-delete: ## Delete Kind cluster
	kind delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: kind-load-image
kind-load-image: docker-build ## Load image into Kind cluster
	kind load docker-image $(IMG) --name $(KIND_CLUSTER_NAME)

.PHONY: kind-load-toolbox
kind-load-toolbox: docker-build-toolbox ## Load toolbox image into Kind cluster
	kind load docker-image $(TOOLBOX_IMG) --name $(KIND_CLUSTER_NAME)

# ==============================================================================
# ArgoCD
# ==============================================================================

##@ ArgoCD

.PHONY: install-argocd
install-argocd: ## Install ArgoCD
	kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
	@echo "Waiting for ArgoCD to be ready..."
	kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd
	@echo "✅ ArgoCD installed"
	@echo "   Get password: kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d"
	@echo "   Port forward: kubectl port-forward svc/argocd-server -n argocd 8080:443"

.PHONY: argocd-password
argocd-password: ## Get ArgoCD admin password
	@kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d && echo

.PHONY: argocd-port-forward
argocd-port-forward: ## Port forward ArgoCD UI
	kubectl port-forward svc/argocd-server -n argocd 8080:443

# ==============================================================================
# Crossplane
# ==============================================================================

##@ Crossplane

.PHONY: install-crossplane
install-crossplane: ## Install Crossplane with AWS Provider
	helm repo add crossplane-stable https://charts.crossplane.io/stable || true
	helm repo update
	helm upgrade --install crossplane crossplane-stable/crossplane \
		--namespace crossplane-system \
		--create-namespace \
		--wait
	@echo "Waiting for Crossplane to be ready..."
	kubectl wait --for=condition=available --timeout=300s deployment/crossplane -n crossplane-system
	@echo "Installing AWS IAM Provider..."
	kubectl apply -f hack/crossplane/provider-aws.yaml
	@echo "Waiting for AWS Provider to be healthy (this may take a minute)..."
	@sleep 15
	kubectl wait --for=condition=healthy --timeout=300s provider.pkg.crossplane.io/provider-aws-iam || echo "Provider still initializing, continuing..."
	@echo "✅ Crossplane with AWS Provider installed"

.PHONY: setup-localstack-provider
setup-localstack-provider: ## Configure Crossplane to use LocalStack
	@echo "Waiting for Provider CRDs to be available..."
	@sleep 10
	@echo "Creating LocalStack ProviderConfig..."
	kubectl apply -f hack/crossplane/providerconfig-localstack.yaml
	@echo "✅ LocalStack ProviderConfig created"

# ==============================================================================
# LocalStack
# ==============================================================================

##@ LocalStack

.PHONY: localstack-start
localstack-start: ## Start LocalStack (run in separate terminal)
	@echo "Starting LocalStack..."
	@echo "NOTE: Run this in a separate terminal or use Docker:"
	@echo "  docker run --rm -p 4566:4566 -p 4510-4559:4510-4559 localstack/localstack"
	@echo ""
	@echo "Or if you have localstack CLI:"
	@echo "  localstack start"
	@echo ""
	@echo "Available services (Community):"
	@echo "  IAM, STS, S3, Lambda, EC2, DynamoDB, SQS, SNS, Kinesis,"
	@echo "  Route53, CloudWatch, KMS, Secrets Manager, and more"

.PHONY: localstack-health
localstack-health: ## Check LocalStack health
	@curl -s $(LOCALSTACK_ENDPOINT)/_localstack/health | jq .

.PHONY: localstack-status
localstack-status: ## Show LocalStack service status
	@localstack status services 2>/dev/null || curl -s $(LOCALSTACK_ENDPOINT)/_localstack/health | jq '.services'

.PHONY: localstack-create-drift
localstack-create-drift: ## Create IAM drift in LocalStack for testing
	@echo "Creating drift in LocalStack IAM resources..."
	@echo "1. Adding extra inline policy to tenant-alpha-lambda-role..."
	awslocal iam put-role-policy \
		--role-name tenant-alpha-lambda-role \
		--policy-name unauthorized-extra-policy \
		--policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}' 2>/dev/null || echo "Role may not exist yet"
	@echo "2. Creating orphaned role (not managed by Crossplane)..."
	awslocal iam create-role \
		--role-name orphaned-manual-role \
		--assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ec2.amazonaws.com"},"Action":"sts:AssumeRole"}]}' 2>/dev/null || echo "Role may already exist"
	@echo "✅ Drift test resources created"
	@echo ""
	@echo "To verify:"
	@echo "  awslocal iam list-roles"
	@echo "  awslocal iam list-role-policies --role-name tenant-alpha-lambda-role"

# ==============================================================================
# Tenant Seeding
# ==============================================================================

##@ Test Data

.PHONY: seed-tenants
seed-tenants: ## Create fake tenants and resources for testing
	kubectl apply -f hack/seed-tenants/namespaces.yaml
	kubectl apply -f hack/seed-tenants/tenant-alpha.yaml
	kubectl apply -f hack/seed-tenants/tenant-beta.yaml
	@echo "Waiting for Crossplane CRDs to be ready..."
	@sleep 5
	kubectl apply -f hack/seed-tenants/iam-resources.yaml || echo "IAM CRDs not ready yet, try again later"
	kubectl apply -f hack/seed-tenants/argo-apps.yaml
	@echo "✅ Test tenants seeded"

.PHONY: seed-failed-resources
seed-failed-resources: ## Create resources in failed/waiting states for testing
	kubectl apply -f hack/seed-tenants/failed-resources.yaml
	@echo "✅ Failed resources seeded"

.PHONY: seed-paused-resources
seed-paused-resources: ## Pause some Crossplane resources for testing
	kubectl annotate --overwrite roles.iam.aws.upbound.io/tenant-alpha-lambda-role crossplane.io/paused="true" || echo "Resource not found"
	@echo "✅ Resources paused"

.PHONY: clear-tenants
clear-tenants: ## Remove test tenant resources
	kubectl delete -f hack/seed-tenants/ --ignore-not-found
	@echo "✅ Test tenants cleared"

# ==============================================================================
# Building
# ==============================================================================

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary
	go build -o bin/manager cmd/main.go

.PHONY: run
run: generate fmt vet ## Run controller locally against the cluster
	go run ./cmd/main.go

.PHONY: docker-build
docker-build: ## Build Docker image (for local development - arm64)
	docker build -t $(IMG) .

.PHONY: docker-build-toolbox
docker-build-toolbox: ## Build toolbox Docker image
	docker build -t $(TOOLBOX_IMG) -f Dockerfile.toolbox .

.PHONY: docker-build-amd64
docker-build-amd64: ## Build Docker image for amd64 (deployment)
	docker buildx build --platform linux/amd64 -t $(IMG) --load .

.PHONY: docker-build-prod
docker-build-prod: ## Build and push production image (amd64)
	docker buildx build \
		--platform linux/amd64 \
		--tag $(REGISTRY)/platform-manager:$(VERSION) \
		--push \
		.

.PHONY: docker-push
docker-push: ## Push Docker image
	docker push $(IMG)

.PHONY: ko-build
ko-build: ## Build with Ko (faster iteration)
	KO_DOCKER_REPO=kind.local ko build ./cmd/ --bare

.PHONY: ko-build-amd64
ko-build-amd64: ## Build with Ko for amd64
	KO_DOCKER_REPO=$(REGISTRY) GOARCH=amd64 ko build ./cmd/ --bare --platform=linux/amd64

# ==============================================================================
# Frontend
# ==============================================================================

##@ Frontend

.PHONY: web-install
web-install: ## Install frontend dependencies
	cd web && npm install

.PHONY: web-dev
web-dev: ## Run frontend dev server
	cd web && npm run serve

.PHONY: web-build
web-build: ## Build frontend for production
	cd web && npm run build

.PHONY: web-build-mfe
web-build-mfe: ## Build frontend as single-spa MFE
	cd web && npm run build:mfe

# ==============================================================================
# Kubebuilder / Code Generation
# ==============================================================================

##@ Code Generation

.PHONY: manifests
manifests: controller-gen ## Generate CRD manifests
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code (DeepCopy, etc.)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

# ==============================================================================
# Testing
# ==============================================================================

##@ Testing

.PHONY: test
test: generate fmt vet envtest ## Run unit tests
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: test-e2e
test-e2e: ## Run e2e tests (requires dev-up)
	go test ./test/e2e/... -v -timeout 30m

.PHONY: test-integration
test-integration: ## Run integration tests
	go test ./internal/... -tags=integration -v

# ==============================================================================
# Deployment
# ==============================================================================

##@ Deployment

.PHONY: install
install: manifests kustomize ## Install CRDs into the cluster
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the cluster
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the cluster
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the cluster
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found -f -

.PHONY: deploy-dev
deploy-dev: docker-build kind-load-image deploy ## Build, load, and deploy to Kind

# ==============================================================================
# Tool Dependencies
# ==============================================================================

##@ Tool Dependencies

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
KUSTOMIZE ?= $(LOCALBIN)/kustomize
ENVTEST ?= $(LOCALBIN)/setup-envtest

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

.PHONY: kustomize
kustomize: $(KUSTOMIZE)
$(KUSTOMIZE): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5@latest

.PHONY: envtest
envtest: $(ENVTEST)
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)

# ==============================================================================
# Help
# ==============================================================================

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
```

### 0.4 Kind Cluster Configuration

Create `hack/kind-config.yaml`:

```yaml
# hack/kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: platform-manager
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
    extraPortMappings:
      # ArgoCD UI
      - containerPort: 30080
        hostPort: 8080
        protocol: TCP
      # Platform Manager API
      - containerPort: 30081
        hostPort: 8081
        protocol: TCP
      # Platform Manager Frontend (dev)
      - containerPort: 30082
        hostPort: 3000
        protocol: TCP
  - role: worker
  - role: worker
```

### 0.5 Crossplane AWS Provider Setup

Create `hack/crossplane/provider-aws.yaml`:

```yaml
# hack/crossplane/provider-aws.yaml
---
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-aws-iam
spec:
  package: xpkg.upbound.io/upbound/provider-aws-iam:v1.7.0
  runtimeConfigRef:
    name: provider-aws-iam
---
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: provider-aws-iam
spec:
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
            - name: package-runtime
              args:
                - --debug
              env:
                - name: AWS_ACCESS_KEY_ID
                  value: "test"
                - name: AWS_SECRET_ACCESS_KEY
                  value: "test"
```

Create `hack/crossplane/providerconfig-localstack.yaml`:

```yaml
# hack/crossplane/providerconfig-localstack.yaml
# ProviderConfig for LocalStack Community
# Available services: IAM, STS, S3, Lambda, EC2, DynamoDB, SQS, SNS, Kinesis,
# Route53, CloudWatch, KMS, Secrets Manager, CloudFormation, SSM, etc.
---
apiVersion: v1
kind: Secret
metadata:
  name: aws-localstack-creds
  namespace: crossplane-system
type: Opaque
stringData:
  credentials: |
    [default]
    aws_access_key_id = test
    aws_secret_access_key = test
---
apiVersion: aws.upbound.io/v1beta1
kind: ProviderConfig
metadata:
  name: localstack
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: aws-localstack-creds
      key: credentials
  endpoint:
    # LocalStack endpoint - host.docker.internal works from Kind on Mac
    url:
      type: Static
      static: http://host.docker.internal:4566
    hostnameImmutable: true
    # Services available in LocalStack Community
    services:
      - iam
      - sts
      - s3
      - s3control
      - lambda
      - ec2
      - dynamodb
      - sqs
      - sns
      - kinesis
      - firehose
      - events
      - route53
      - route53resolver
      - apigateway
      - cloudwatch
      - logs
      - kms
      - secretsmanager
      - acm
      - cloudformation
      - config
      - ssm
      - stepfunctions
      - redshift
      - opensearch
      - es
```

### 0.6 Seed Tenant Resources

Create `hack/seed-tenants/namespaces.yaml`:

```yaml
# hack/seed-tenants/namespaces.yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-alpha
  labels:
    platform.io/tenant: alpha
    platform.io/environment: dev
---
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-beta
  labels:
    platform.io/tenant: beta
    platform.io/environment: dev
---
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-gamma
  labels:
    platform.io/tenant: gamma
    platform.io/environment: dev
---
apiVersion: v1
kind: Namespace
metadata:
  name: toolbox-sessions
  labels:
    platform.io/component: toolbox
```

Create `hack/seed-tenants/tenant-alpha.yaml`:

```yaml
# hack/seed-tenants/tenant-alpha.yaml
# Tenant Alpha - ML workloads
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ml-training-service
  namespace: tenant-alpha
  labels:
    platform.io/tenant: alpha
    platform.io/component: ml
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ml-training-service
  template:
    metadata:
      labels:
        app: ml-training-service
        platform.io/tenant: alpha
    spec:
      containers:
        - name: trainer
          image: nginx:alpine  # Placeholder
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "200m"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: inference-api
  namespace: tenant-alpha
  labels:
    platform.io/tenant: alpha
    platform.io/component: ml
spec:
  replicas: 3
  selector:
    matchLabels:
      app: inference-api
  template:
    metadata:
      labels:
        app: inference-api
        platform.io/tenant: alpha
    spec:
      containers:
        - name: api
          image: nginx:alpine  # Placeholder
          resources:
            requests:
              memory: "64Mi"
              cpu: "50m"
            limits:
              memory: "128Mi"
              cpu: "100m"
```

Create `hack/seed-tenants/tenant-beta.yaml`:

```yaml
# hack/seed-tenants/tenant-beta.yaml
# Tenant Beta - Data pipeline workloads
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: etl-processor
  namespace: tenant-beta
  labels:
    platform.io/tenant: beta
    platform.io/component: data
spec:
  replicas: 1
  selector:
    matchLabels:
      app: etl-processor
  template:
    metadata:
      labels:
        app: etl-processor
        platform.io/tenant: beta
    spec:
      containers:
        - name: processor
          image: nginx:alpine
          resources:
            requests:
              memory: "256Mi"
              cpu: "200m"
            limits:
              memory: "512Mi"
              cpu: "500m"
```

Create `hack/seed-tenants/iam-resources.yaml`:

```yaml
# hack/seed-tenants/iam-resources.yaml
# Crossplane IAM resources pointing to LocalStack
---
apiVersion: iam.aws.upbound.io/v1beta1
kind: Role
metadata:
  name: tenant-alpha-lambda-role
  labels:
    platform.io/tenant: alpha
spec:
  providerConfigRef:
    name: localstack
  forProvider:
    assumeRolePolicy: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Principal": {
              "Service": "lambda.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
          }
        ]
      }
    tags:
      - key: tenant
        value: alpha
      - key: managed-by
        value: crossplane
---
apiVersion: iam.aws.upbound.io/v1beta1
kind: Policy
metadata:
  name: tenant-alpha-s3-policy
  labels:
    platform.io/tenant: alpha
spec:
  providerConfigRef:
    name: localstack
  forProvider:
    policy: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Action": [
              "s3:GetObject",
              "s3:PutObject"
            ],
            "Resource": "arn:aws:s3:::tenant-alpha-*/*"
          }
        ]
      }
    tags:
      - key: tenant
        value: alpha
---
apiVersion: iam.aws.upbound.io/v1beta1
kind: Role
metadata:
  name: tenant-beta-data-role
  labels:
    platform.io/tenant: beta
spec:
  providerConfigRef:
    name: localstack
  forProvider:
    assumeRolePolicy: |
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Principal": {
              "Service": "lambda.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
          }
        ]
      }
    tags:
      - key: tenant
        value: beta
```

Create `hack/seed-tenants/argo-apps.yaml`:

```yaml
# hack/seed-tenants/argo-apps.yaml
# ArgoCD Applications for testing
---
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: tenant-alpha
  namespace: argocd
spec:
  description: Tenant Alpha applications
  sourceRepos:
    - '*'
  destinations:
    - namespace: tenant-alpha
      server: https://kubernetes.default.svc
  clusterResourceWhitelist:
    - group: ''
      kind: Namespace
---
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: tenant-beta
  namespace: argocd
spec:
  description: Tenant Beta applications
  sourceRepos:
    - '*'
  destinations:
    - namespace: tenant-beta
      server: https://kubernetes.default.svc
---
# Sample Argo Application (synced, healthy)
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: alpha-ml-platform
  namespace: argocd
  labels:
    platform.io/tenant: alpha
spec:
  project: tenant-alpha
  source:
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    targetRevision: HEAD
    path: guestbook
  destination:
    server: https://kubernetes.default.svc
    namespace: tenant-alpha
  syncPolicy:
    automated:
      prune: false
      selfHeal: false
---
# Sample Argo Application (will be out of sync for testing)
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: beta-data-pipeline
  namespace: argocd
  labels:
    platform.io/tenant: beta
spec:
  project: tenant-beta
  source:
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    targetRevision: HEAD
    path: guestbook
  destination:
    server: https://kubernetes.default.svc
    namespace: tenant-beta
```

Create `hack/seed-tenants/failed-resources.yaml`:

```yaml
# hack/seed-tenants/failed-resources.yaml
# Resources intentionally configured to fail for testing
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: failing-deployment
  namespace: tenant-alpha
  labels:
    platform.io/tenant: alpha
    platform.io/test: "failed-state"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: failing-app
  template:
    metadata:
      labels:
        app: failing-app
    spec:
      containers:
        - name: fail
          image: nonexistent-image:v999.999.999  # Will fail ImagePullBackOff
          resources:
            requests:
              memory: "32Mi"
              cpu: "10m"
---
apiVersion: v1
kind: Pod
metadata:
  name: crashloop-pod
  namespace: tenant-beta
  labels:
    platform.io/tenant: beta
    platform.io/test: "failed-state"
spec:
  restartPolicy: Always
  containers:
    - name: crasher
      image: busybox
      command: ["sh", "-c", "exit 1"]  # Will CrashLoopBackOff
```

### 0.7 Vue 3 single-spa Frontend Setup

Create `web/package.json`:

```json
{
  "name": "@platform/manager-mfe",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "serve": "vue-cli-service serve",
    "build": "vue-cli-service build",
    "build:mfe": "vue-cli-service build --target lib --formats umd-min --name platform-manager src/main.ts",
    "lint": "vue-cli-service lint",
    "test:unit": "vue-cli-service test:unit"
  },
  "dependencies": {
    "axios": "^1.6.0",
    "chart.js": "^4.4.0",
    "pinia": "^2.1.7",
    "single-spa-vue": "^2.5.1",
    "vue": "^3.4.0",
    "vue-chartjs": "^5.3.0",
    "vue-router": "^4.2.5",
    "xterm": "^5.3.0",
    "xterm-addon-attach": "^0.9.0",
    "xterm-addon-fit": "^0.8.0"
  },
  "devDependencies": {
    "@types/node": "^20.10.0",
    "@typescript-eslint/eslint-plugin": "^6.13.0",
    "@typescript-eslint/parser": "^6.13.0",
    "@vue/cli-plugin-eslint": "^5.0.8",
    "@vue/cli-plugin-router": "^5.0.8",
    "@vue/cli-plugin-typescript": "^5.0.8",
    "@vue/cli-service": "^5.0.8",
    "@vue/eslint-config-typescript": "^12.0.0",
    "eslint": "^8.54.0",
    "eslint-plugin-vue": "^9.18.0",
    "sass": "^1.69.0",
    "sass-loader": "^13.3.0",
    "typescript": "^5.3.0"
  },
  "browserslist": [
    "> 1%",
    "last 2 versions",
    "not dead"
  ]
}
```

Create `web/vue.config.js`:

```javascript
// web/vue.config.js
const { defineConfig } = require('@vue/cli-service')

module.exports = defineConfig({
  transpileDependencies: true,
  
  // Configure for single-spa MFE
  configureWebpack: {
    output: {
      // Output as SystemJS module for single-spa
      libraryTarget: 'system',
      // Unique name for the MFE
      library: {
        type: 'system',
      },
    },
    externals: [
      // Externalize shared dependencies (loaded by shell)
      'vue',
      'vue-router',
      'pinia',
      /^@platform\/.+/,
    ],
  },
  
  // Disable chunk splitting for single-spa
  chainWebpack: (config) => {
    config.optimization.delete('splitChunks')
    
    // Disable HTML plugin for library mode
    config.plugins.delete('html')
    config.plugins.delete('preload')
    config.plugins.delete('prefetch')
  },
  
  // Dev server config
  devServer: {
    port: 3000,
    headers: {
      'Access-Control-Allow-Origin': '*',
    },
  },
})
```

Create `web/src/main.ts`:

```typescript
// web/src/main.ts
// single-spa Vue 3 entry point with lifecycle methods
import { h, createApp } from 'vue'
import { createPinia } from 'pinia'
import singleSpaVue from 'single-spa-vue'
import App from './App.vue'
import router from './router'

// Create the Vue lifecycle wrapper for single-spa
const vueLifecycles = singleSpaVue({
  createApp,
  appOptions: {
    render() {
      return h(App, {
        // Props passed from shell application
        // @ts-ignore
        ...this.customProps,
      })
    },
  },
  handleInstance: (app) => {
    app.use(createPinia())
    app.use(router)
  },
})

// Export single-spa lifecycle methods
export const bootstrap = vueLifecycles.bootstrap
export const mount = vueLifecycles.mount
export const unmount = vueLifecycles.unmount

// For standalone development
if (!window.singleSpaNavigate) {
  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  app.mount('#app')
}
```

Create `web/src/App.vue`:

```vue
<!-- web/src/App.vue -->
<template>
  <div id="platform-manager">
    <router-view />
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue'

export default defineComponent({
  name: 'PlatformManager',
})
</script>

<style lang="scss">
#platform-manager {
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
</style>
```

Create `web/src/router/index.ts`:

```typescript
// web/src/router/index.ts
import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    name: 'dashboard',
    component: () => import('../views/DashboardView.vue'),
  },
  {
    path: '/tenants',
    name: 'tenants',
    component: () => import('../views/TenantsView.vue'),
  },
  {
    path: '/tenants/:id',
    name: 'tenant-detail',
    component: () => import('../views/TenantDetailView.vue'),
    props: true,
  },
  {
    path: '/resources',
    name: 'resources',
    component: () => import('../views/ResourcesView.vue'),
  },
  {
    path: '/iam',
    name: 'iam-drift',
    component: () => import('../views/IAMDriftView.vue'),
  },
]

const router = createRouter({
  // Use base path for MFE
  history: createWebHistory(process.env.BASE_URL || '/platform/'),
  routes,
})

export default router
```

### 0.8 Phase 0 Deliverables Checklist

| Task | Status | Notes |
|------|--------|-------|
| Initialize git repository | ⬜ | `make git-init` or `git init` |
| Create .gitignore | ⬜ | See 0.1 |
| Install local prerequisites (kind, kubectl, helm, etc.) | ⬜ | One-time setup |
| Create Makefile with all targets | ⬜ | See 0.3 |
| Create Kind cluster config | ⬜ | See 0.4 |
| Create Crossplane provider configs | ⬜ | See 0.5 |
| Create seed tenant resources | ⬜ | See 0.6 |
| Create Vue 3 single-spa skeleton | ⬜ | See 0.7 |
| Start LocalStack (manually) | ⬜ | `localstack start` |
| Run `make dev-up` and verify everything works | ⬜ | Full integration test |
| Verify LocalStack connection from Kind | ⬜ | `awslocal iam list-roles` |
| Verify Crossplane can create resources in LocalStack | ⬜ | Check IAM role creation |
| Create initial git commit | ⬜ | After all files created |
| Initialize Kubebuilder project | ⬜ | Next step (Phase 1) |

### 0.9 Quick Start Commands

```bash
# 0. Initialize git (FIRST STEP)
git init
git branch -M main

# 1. Start LocalStack (in a separate terminal)
localstack start
# OR
docker run --rm -p 4566:4566 -p 4510-4559:4510-4559 localstack/localstack

# 2. Create the full dev environment
make dev-up

# 3. Verify everything is running
kubectl get pods -A

# 4. Check Crossplane resources
kubectl get providers
kubectl get providerconfigs

# 5. Check ArgoCD
make argocd-password  # Get the password
make argocd-port-forward  # Access UI at https://localhost:8080

# 6. Check LocalStack health
make localstack-health

# 7. Check available LocalStack services
make localstack-status

# 8. Verify Crossplane → LocalStack connection
kubectl get roles.iam.aws.upbound.io
awslocal iam list-roles

# 9. Create drift for testing
make localstack-create-drift

# 10. Commit Phase 0 setup
git add .
git commit -m "Phase 0: Development environment setup"
```

---

## Phase 1: Backend Skeleton & Global Overview

**Goal**: Build the core controller infrastructure and expose platform/tenant health endpoints.

### 1.1 Initialize Kubebuilder Project

```bash
# Initialize the project (from project root)
go mod init github.com/yourorg/platform-manager
kubebuilder init --domain platform.io --repo github.com/yourorg/platform-manager

# Create CRDs
kubebuilder create api --group platform --version v1alpha1 --kind Tenant --resource --controller
kubebuilder create api --group platform --version v1alpha1 --kind ResourceSummary --resource --controller=false
kubebuilder create api --group platform --version v1alpha1 --kind TenantHealth --resource --controller=false
kubebuilder create api --group platform --version v1alpha1 --kind PlatformHealth --resource --controller=false

# Commit
git add .
git commit -m "Phase 1: Initialize Kubebuilder project with CRDs"
```

### 1.2 CRD Definitions

**Tenant CRD** (`api/v1alpha1/tenant_types.go`):

```go
package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantSpec defines the desired state of Tenant
type TenantSpec struct {
    // DisplayName is the human-readable name
    DisplayName string `json:"displayName"`
    
    // Namespaces that belong to this tenant
    Namespaces []string `json:"namespaces,omitempty"`
    
    // LabelSelector to match resources belonging to this tenant
    // +optional
    LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
    
    // AWSAccounts associated with this tenant (for IAM tracking)
    // +optional
    AWSAccounts []string `json:"awsAccounts,omitempty"`
    
    // ArgoProjects that belong to this tenant
    // +optional
    ArgoProjects []string `json:"argoProjects,omitempty"`
    
    // Contacts for this tenant (for alerts/notifications)
    // +optional
    Contacts []TenantContact `json:"contacts,omitempty"`
}

type TenantContact struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Role  string `json:"role,omitempty"` // owner, developer, oncall
}

// TenantStatus defines the observed state of Tenant
type TenantStatus struct {
    // Phase represents the current phase (Active, Suspended, Deleting)
    Phase string `json:"phase,omitempty"`
    
    // LastReconciled timestamp
    LastReconciled *metav1.Time `json:"lastReconciled,omitempty"`
    
    // HealthRef points to the TenantHealth resource
    HealthRef string `json:"healthRef,omitempty"`
    
    // Conditions
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Tenant represents a logical tenant in the platform
type Tenant struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   TenantSpec   `json:"spec,omitempty"`
    Status TenantStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TenantList contains a list of Tenant
type TenantList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []Tenant `json:"items"`
}
```

**TenantHealth CRD** (`api/v1alpha1/tenanthealth_types.go`):

```go
package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/api/resource"
)

// TenantHealthSpec defines the desired state (mostly empty, driven by controller)
type TenantHealthSpec struct {
    // TenantRef is the name of the Tenant this health summary is for
    TenantRef string `json:"tenantRef"`
}

// ResourceStateCounts holds counts for each state
type ResourceStateCounts struct {
    Ready   int `json:"ready"`
    Failed  int `json:"failed"`
    Waiting int `json:"waiting"`
    Unknown int `json:"unknown"`
    Paused  int `json:"paused"`
    Total   int `json:"total"`
}

// IAMDriftSummary summarizes IAM drift
type IAMDriftSummary struct {
    TotalRoles         int `json:"totalRoles"`
    TotalPolicies      int `json:"totalPolicies"`
    RolesWithDrift     int `json:"rolesWithDrift"`
    PoliciesWithDrift  int `json:"policiesWithDrift"`
    ExtraPrivileges    int `json:"extraPrivileges"`
    MissingPrivileges  int `json:"missingPrivileges"`
}

// ArgoSummary summarizes ArgoCD application states
type ArgoSummary struct {
    TotalApps    int `json:"totalApps"`
    Synced       int `json:"synced"`
    OutOfSync    int `json:"outOfSync"`
    Healthy      int `json:"healthy"`
    Degraded     int `json:"degraded"`
    Progressing  int `json:"progressing"`
    Missing      int `json:"missing"`
}

// TenantHealthStatus defines the observed state
type TenantHealthStatus struct {
    // OverallHealth: Healthy, Degraded, Critical
    OverallHealth string `json:"overallHealth"`
    
    // CrossplaneResources state counts
    CrossplaneResources ResourceStateCounts `json:"crossplaneResources"`
    
    // KubernetesResources state counts (Deployments, Pods, etc.)
    KubernetesResources ResourceStateCounts `json:"kubernetesResources"`
    
    // IAMDrift summary
    IAMDrift IAMDriftSummary `json:"iamDrift"`
    
    // Argo summary
    Argo ArgoSummary `json:"argo"`
    
    // CPUUsage in millicores
    CPUUsage resource.Quantity `json:"cpuUsage,omitempty"`
    
    // MemoryUsage in bytes
    MemoryUsage resource.Quantity `json:"memoryUsage,omitempty"`
    
    // CPURequest in millicores
    CPURequest resource.Quantity `json:"cpuRequest,omitempty"`
    
    // MemoryRequest in bytes
    MemoryRequest resource.Quantity `json:"memoryRequest,omitempty"`
    
    // LastUpdated timestamp
    LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
    
    // TopIssues - top 5 issues for this tenant
    TopIssues []Issue `json:"topIssues,omitempty"`
}

type Issue struct {
    Severity    string `json:"severity"` // critical, warning, info
    Message     string `json:"message"`
    ResourceRef string `json:"resourceRef,omitempty"`
    Timestamp   metav1.Time `json:"timestamp"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenantRef`
//+kubebuilder:printcolumn:name="Health",type=string,JSONPath=`.status.overallHealth`
//+kubebuilder:printcolumn:name="Failed",type=integer,JSONPath=`.status.crossplaneResources.failed`
//+kubebuilder:printcolumn:name="IAM Drift",type=integer,JSONPath=`.status.iamDrift.rolesWithDrift`

// TenantHealth is the Schema for the tenanthealth API
type TenantHealth struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   TenantHealthSpec   `json:"spec,omitempty"`
    Status TenantHealthStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TenantHealthList contains a list of TenantHealth
type TenantHealthList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []TenantHealth `json:"items"`
}
```

### 1.3 Controller Architecture

**Watchers to implement**:

| Watcher | Watches | Purpose |
|---------|---------|---------|
| `TenantReconciler` | `Tenant` CRD | Manages tenant lifecycle, creates TenantHealth |
| `CrossplaneWatcher` | XRDs, XRs, Claims, Providers (all families) | Detects Crossplane resource health |
| `ArgoWatcher` | Applications, AppProjects | Detects Argo sync/health status |
| `HealthAggregator` | (scheduled) | Aggregates health into TenantHealth/PlatformHealth |

**Providers to watch** (dynamic discovery):
- AWS family providers (IAM, S3, Lambda, etc.)
- Custom KnowledgeBases provider
- Ansible, ArgoCD, Grafana, Helm, Kubernetes, OpenTofu, Terraform, Vault

### 1.4 API Endpoints (Phase 1)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `GET /api/v1/health/platform` | GET | Global platform health summary |
| `GET /api/v1/health/tenants` | GET | List all tenant health summaries |
| `GET /api/v1/health/tenants/:id` | GET | Single tenant health detail |
| `GET /api/v1/tenants` | GET | List all tenants |
| `GET /api/v1/tenants/:id` | GET | Get tenant details |

### 1.5 Authentication Middleware

Since auth is handled by OAuth2Proxy at the gateway level, extract user info from headers:

```go
// internal/api/middleware/auth.go
package middleware

import (
    "context"
    "net/http"
    "strings"
)

type Role string

const (
    RoleAdmin     Role = "admin"
    RoleInfra     Role = "infra"
    RoleML        Role = "ml"
    RoleReadOnly  Role = "readonly"
)

type UserInfo struct {
    Username string
    Email    string
    Groups   []string
    Role     Role
}

type contextKey string

const userInfoKey contextKey = "userInfo"

// ExtractUser extracts user info from OAuth2Proxy headers
func ExtractUser(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := UserInfo{
            Username: r.Header.Get("X-Auth-Request-User"),
            Email:    r.Header.Get("X-Auth-Request-Email"),
            Groups:   strings.Split(r.Header.Get("X-Auth-Request-Groups"), ","),
        }
        
        // Map groups to role (customize based on your ACR rules)
        user.Role = mapGroupsToRole(user.Groups)
        
        ctx := context.WithValue(r.Context(), userInfoKey, user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func mapGroupsToRole(groups []string) Role {
    for _, g := range groups {
        switch strings.TrimSpace(g) {
        case "platform-admin", "admin":
            return RoleAdmin
        case "platform-infra", "infra":
            return RoleInfra
        case "platform-ml", "ml":
            return RoleML
        }
    }
    return RoleReadOnly
}

func UserFromContext(ctx context.Context) UserInfo {
    if user, ok := ctx.Value(userInfoKey).(UserInfo); ok {
        return user
    }
    return UserInfo{Role: RoleReadOnly}
}
```

### 1.6 Phase 1 Deliverables

| Task | Priority | Effort |
|------|----------|--------|
| Kubebuilder project init | P0 | 1 day |
| Define Tenant, TenantHealth, PlatformHealth CRDs | P0 | 2 days |
| Implement TenantReconciler | P0 | 2 days |
| Implement CrossplaneWatcher (basic) | P0 | 3 days |
| Implement ArgoWatcher (basic) | P1 | 2 days |
| Implement HealthAggregator | P0 | 2 days |
| HTTP API server integration | P0 | 2 days |
| Implement health endpoints | P0 | 2 days |
| Implement auth middleware (header extraction) | P0 | 1 day |
| Unit tests for all controllers | P0 | 3 days |
| E2E tests with Kind | P1 | 2 days |
| Git commits throughout | P0 | - |

**Estimated time**: 3-4 weeks

---

## Phase 2: Drill-down & Resource Detail

**Goal**: Implement resource listing, filtering, and detailed views.

### 2.1 ResourceSummary CRD

We'll store normalized resource state in `ResourceSummary` objects:

```go
// ResourceSummarySpec captures the resource reference
type ResourceSummarySpec struct {
    // Resource reference
    Group     string `json:"group"`
    Version   string `json:"version"`
    Kind      string `json:"kind"`
    Namespace string `json:"namespace,omitempty"`
    Name      string `json:"name"`
    
    // TenantRef links to the owning tenant
    TenantRef string `json:"tenantRef"`
}

// ResourceSummaryStatus captures normalized health
type ResourceSummaryStatus struct {
    // State: Ready, Failed, Waiting, Unknown, Paused
    State string `json:"state"`
    
    // Conditions from the source resource (condensed)
    Conditions []ConditionSummary `json:"conditions,omitempty"`
    
    // Message explaining current state
    Message string `json:"message,omitempty"`
    
    // OwnerChain for traversing resource tree
    OwnerChain []OwnerRef `json:"ownerChain,omitempty"`
    
    // Age
    Age metav1.Duration `json:"age"`
    
    // LastTransition when state last changed
    LastTransition *metav1.Time `json:"lastTransition,omitempty"`
    
    // LastSeen when resource was last observed
    LastSeen *metav1.Time `json:"lastSeen,omitempty"`
}
```

### 2.2 API Endpoints (Phase 2)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `GET /api/v1/resources` | GET | List resources (with filters) |
| `GET /api/v1/resources/:group/:version/:kind/:namespace/:name` | GET | Get resource detail |
| `GET /api/v1/resources/:id/yaml` | GET | Get raw YAML |
| `GET /api/v1/resources/:id/events` | GET | Get K8s events for resource |
| `GET /api/v1/resources/:id/tree` | GET | Get ownership tree |
| `GET /api/v1/tenants/:id/resources` | GET | List resources for tenant |

**Query parameters for `/resources`**:
- `tenant`: Filter by tenant
- `state`: Filter by state (ready, failed, waiting, unknown, paused)
- `kind`: Filter by kind (Deployment, XR, Application, etc.)
- `provider`: Filter by Crossplane provider
- `search`: Free-text search by name

### 2.3 Prometheus Integration for Metrics

```go
// internal/metrics/prometheus.go
type PrometheusClient struct {
    api promv1.API
}

func (c *PrometheusClient) GetNamespaceMetrics(ctx context.Context, namespace string) (*NamespaceMetrics, error) {
    // Query CPU usage
    cpuQuery := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s"}[5m]))`, namespace)
    cpuResult, _, _ := c.api.Query(ctx, cpuQuery, time.Now())
    
    // Query memory usage
    memQuery := fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace="%s"})`, namespace)
    memResult, _, _ := c.api.Query(ctx, memQuery, time.Now())
    
    // Parse and return
    return &NamespaceMetrics{
        CPUUsage: parseQuantity(cpuResult),
        MemoryUsage: parseQuantity(memResult),
    }, nil
}
```

### 2.4 Phase 2 Deliverables

| Task | Priority | Effort |
|------|----------|--------|
| Define ResourceSummary CRD | P0 | 1 day |
| Implement resource scanning and normalization | P0 | 3 days |
| Implement resource listing API with filters | P0 | 2 days |
| Implement resource detail API | P0 | 2 days |
| Implement ownership tree traversal | P1 | 2 days |
| Integrate with existing Prometheus | P1 | 2 days |
| Add events fetching | P1 | 1 day |
| Frontend: Tenant detail page | P0 | 3 days |
| Frontend: Resource list with filters | P0 | 3 days |
| Frontend: Resource detail page | P0 | 2 days |

**Estimated time**: 3-4 weeks

---

## Phase 3: Actions (Mini Argo + Crossplane Controls)

**Goal**: Enable mutating operations (sync, pause, delete) with proper authorization.

### 3.1 Action Endpoints

| Endpoint | Method | Description | Roles |
|----------|--------|-------------|-------|
| `POST /api/v1/actions/argo/sync` | POST | Sync ArgoCD application | admin, infra |
| `POST /api/v1/actions/argo/refresh` | POST | Refresh ArgoCD application | admin, infra, ml |
| `POST /api/v1/actions/crossplane/pause` | POST | Pause Crossplane resource | admin, infra |
| `POST /api/v1/actions/crossplane/unpause` | POST | Unpause Crossplane resource | admin, infra |
| `POST /api/v1/actions/crossplane/reconcile` | POST | Force reconcile | admin, infra |
| `DELETE /api/v1/resources/:id` | DELETE | Delete resource | admin only |

### 3.2 Authorization Middleware

```go
// internal/api/middleware/authz.go
type Capability string

const (
    CapSyncArgo        Capability = "argo:sync"
    CapRefreshArgo     Capability = "argo:refresh"
    CapPauseCrossplane Capability = "crossplane:pause"
    CapDeleteResource  Capability = "resource:delete"
    CapUseTerminal     Capability = "terminal:use"
)

var roleCapabilities = map[Role][]Capability{
    RoleAdmin:    {CapSyncArgo, CapRefreshArgo, CapPauseCrossplane, CapDeleteResource, CapUseTerminal},
    RoleInfra:    {CapSyncArgo, CapRefreshArgo, CapPauseCrossplane, CapUseTerminal},
    RoleML:       {CapRefreshArgo},
    RoleReadOnly: {},
}

func RequireCapability(cap Capability) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := UserFromContext(r.Context())
            if !hasCapability(user.Role, cap) {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### 3.3 Phase 3 Deliverables

| Task | Priority | Effort |
|------|----------|--------|
| Implement role-based authorization | P0 | 2 days |
| Implement Argo sync/refresh handlers | P0 | 2 days |
| Implement Crossplane pause/unpause handlers | P0 | 2 days |
| Implement force reconcile handler | P1 | 1 day |
| Implement delete handler (with safeguards) | P1 | 2 days |
| Add audit logging | P0 | 2 days |
| Frontend: Add action buttons to UI | P0 | 2 days |
| Frontend: Confirmation dialogs | P0 | 1 day |
| Frontend: Role-based UI controls | P0 | 2 days |
| E2E tests for actions | P1 | 2 days |

**Estimated time**: 3 weeks

---

## Phase 4: IAM Drift Module

**Goal**: Detect and display drift between Crossplane IAM resources and actual AWS state.

### 4.1 Drift Checker Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     IAM Drift Checker                        │
├─────────────────────────────────────────────────────────────┤
│  1. List Crossplane IAM resources (Role, Policy, Attachment)│
│  2. For each resource:                                       │
│     a. Parse spec.forProvider (desired state)               │
│     b. Parse status.atProvider (observed state)             │
│     c. Fetch actual from AWS (live state)                   │
│  3. Compute diff:                                           │
│     - Desired vs Observed (Crossplane reconciliation lag)   │
│     - Observed vs Actual (out-of-band changes)              │
│  4. Store DriftResult per resource                          │
│  5. Aggregate into tenant/global summaries                  │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 LocalStack Community Limitations

For testing with LocalStack Community:
- IAM API is available (create/get/list roles, policies, attachments)
- Policy simulation has quirks
- We can create drift by using `awslocal` to modify resources directly

```bash
# Create drift for testing
make localstack-create-drift
```

### 4.3 Phase 4 Deliverables

| Task | Priority | Effort |
|------|----------|--------|
| Implement AWS IAM client wrapper | P0 | 2 days |
| Implement policy parsing and comparison | P0 | 3 days |
| Implement DriftChecker for Roles | P0 | 2 days |
| Implement DriftChecker for Policies | P0 | 2 days |
| Implement DriftChecker for Attachments | P1 | 2 days |
| Create scheduled drift scan job | P0 | 1 day |
| Implement drift API endpoints | P0 | 2 days |
| Aggregate drift into TenantHealth | P0 | 1 day |
| Frontend: IAM drift page | P0 | 3 days |
| Frontend: Diff visualization | P1 | 2 days |
| LocalStack integration tests | P1 | 2 days |

**Estimated time**: 3-4 weeks

---

## Phase 5: Troubleshooting & Rule Engine

**Goal**: Implement a rule-based system to detect and surface common issues.

### 5.1 Built-in Rules to Implement

| Rule | Severity | Trigger |
|------|----------|---------|
| `crashloop-backoff` | High | Pod in CrashLoopBackOff > 5 restarts |
| `image-pull-failed` | High | Pod with ImagePullBackOff > 10 min |
| `xr-failed-iam` | Critical | XR failed with IAM-related error |
| `paused-but-syncing` | Medium | Crossplane resource paused but Argo syncing |
| `argo-sync-failed` | High | Argo app in failed sync state > 30 min |
| `iam-extra-privileges` | Critical | IAM role has extra privileges vs desired |
| `stale-resource` | Medium | ResourceSummary not updated > 1 hour |
| `provider-unhealthy` | Critical | Crossplane Provider not healthy |
| `high-resource-usage` | Medium | Namespace > 80% of resource quota |

### 5.2 Phase 5 Deliverables

| Task | Priority | Effort |
|------|----------|--------|
| Implement Rule interface and Engine | P0 | 2 days |
| Implement all built-in rules | P0 | 5 days |
| Create scheduled rule evaluation | P0 | 1 day |
| API endpoint for troubleshooting | P0 | 1 day |
| Frontend: Troubleshooting tab | P0 | 3 days |
| Frontend: Findings display | P0 | 2 days |

**Estimated time**: 3 weeks

---

## Phase 6: Web Terminal

**Goal**: Implement secure, audited terminal access using the toolbox pod pattern.

### 6.1 Phase 6 Deliverables

| Task | Priority | Effort |
|------|----------|--------|
| Build toolbox Docker image | P0 | 2 days |
| Create tenant-scoped ServiceAccounts | P0 | 1 day |
| Implement TerminalManager | P0 | 3 days |
| Implement WebSocket handler | P0 | 2 days |
| Implement session lifecycle (create/delete) | P0 | 2 days |
| Implement idle timeout cleanup | P1 | 1 day |
| Implement audit log collection | P0 | 2 days |
| Frontend: xterm.js integration | P0 | 3 days |
| Frontend: Terminal tab in tenant detail | P0 | 2 days |
| RBAC for terminal access | P0 | 1 day |
| E2E tests | P1 | 2 days |

**Estimated time**: 3-4 weeks

---

## Deployment & CI/CD

### Multi-architecture Build

```makefile
# Build for production (amd64)
docker-build-prod:
	docker buildx build \
		--platform linux/amd64 \
		--tag $(REGISTRY)/platform-manager:$(VERSION) \
		--push \
		.

# Build multi-arch
docker-build-multi:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--tag $(REGISTRY)/platform-manager:$(VERSION) \
		--push \
		.
```

### Helm Chart Structure

```
charts/platform-manager/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── serviceaccount.yaml
│   ├── clusterrole.yaml
│   ├── clusterrolebinding.yaml
│   ├── configmap.yaml
│   └── _helpers.tpl
└── crds/
    ├── platform.io_tenants.yaml
    ├── platform.io_resourcesummaries.yaml
    └── platform.io_tenanthealth.yaml
```

---

## Environment Configuration

### Per-Environment Deployment

Since you have 5 environments, each with a single cluster:

| Environment | Cluster | Notes |
|-------------|---------|-------|
| dev | dev-cluster | LocalStack for AWS mocking |
| staging | staging-cluster | Real AWS, limited resources |
| qa | qa-cluster | Full testing |
| preprod | preprod-cluster | Production-like |
| prod | prod-cluster | Full production |

Each environment gets its own:
- Platform Manager deployment
- Tenant CRs for that environment's tenants
- TenantHealth/PlatformHealth CRs
- ResourceSummary CRs

### Environment-Specific Configuration

```yaml
# values-dev.yaml
aws:
  endpoint: "http://localstack:4566"  # LocalStack in dev
  region: us-east-1
prometheus:
  url: "http://prometheus:9090"
features:
  terminal: true
  iamDrift: true

# values-prod.yaml
aws:
  # Uses IRSA, no explicit endpoint
  region: us-east-1
prometheus:
  url: "http://thanos-query:9090"
features:
  terminal: true
  iamDrift: true
```

---

## Timeline Summary

| Phase | Duration | Dependencies |
|-------|----------|--------------|
| Phase 0: Dev Environment | 1 week | None |
| Phase 1: Backend Skeleton | 3-4 weeks | Phase 0 |
| Phase 2: Drill-down | 3-4 weeks | Phase 1 |
| Phase 3: Actions | 3 weeks | Phase 2 |
| Phase 4: IAM Drift | 3-4 weeks | Phase 1 (can parallel with 2-3) |
| Phase 5: Rule Engine | 3 weeks | Phase 2 |
| Phase 6: Web Terminal | 3-4 weeks | Phase 3 |

**Total estimated time**: 4-5 months for full implementation

**MVP (Phases 0-2)**: ~8 weeks
- Dev environment
- Health dashboard
- Resource listing and details

---

## Next Steps

1. ✅ Questions answered
2. **Run Phase 0** to set up the development environment
3. **Initialize Kubebuilder project** and start Phase 1

Start with:
```bash
# Initialize git
git init
git branch -M main

# Create initial structure
# (Create all hack/ files as shown above)

# Commit Phase 0 setup
git add .
git commit -m "Phase 0: Development environment setup"

# Start LocalStack
localstack start

# Create dev environment
make dev-up
```
