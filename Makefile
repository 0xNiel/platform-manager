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
	@echo ""
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

.PHONY: dev-status
dev-status: ## Check status of dev environment
	@echo "=== Cluster Status ==="
	@kubectl cluster-info --context kind-$(KIND_CLUSTER_NAME) 2>/dev/null || echo "Cluster not running"
	@echo ""
	@echo "=== Pods ==="
	@kubectl get pods -A --context kind-$(KIND_CLUSTER_NAME) 2>/dev/null || echo "Cannot connect to cluster"
	@echo ""
	@echo "=== Crossplane Providers ==="
	@kubectl get providers --context kind-$(KIND_CLUSTER_NAME) 2>/dev/null || echo "No providers found"
	@echo ""
	@echo "=== LocalStack ==="
	@curl -s $(LOCALSTACK_ENDPOINT)/_localstack/health 2>/dev/null | jq -r '.services | to_entries[] | "\(.key): \(.value)"' || echo "LocalStack not running"

# ==============================================================================
# Kind Cluster
# ==============================================================================

##@ Kind Cluster

.PHONY: kind-create
kind-create: ## Create Kind cluster
	@if kind get clusters 2>/dev/null | grep -q $(KIND_CLUSTER_NAME); then \
		echo "Cluster $(KIND_CLUSTER_NAME) already exists"; \
	else \
		kind create cluster --name $(KIND_CLUSTER_NAME) --config hack/kind-config.yaml; \
	fi
	@kubectl cluster-info --context kind-$(KIND_CLUSTER_NAME)

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
	@echo "Installing ArgoCD..."
	@kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
	@kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
	@echo "Waiting for ArgoCD to be ready..."
	@kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd
	@echo "✅ ArgoCD installed"
	@echo "   Get password: make argocd-password"
	@echo "   Port forward: make argocd-port-forward"

.PHONY: argocd-password
argocd-password: ## Get ArgoCD admin password
	@kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d && echo

.PHONY: argocd-port-forward
argocd-port-forward: ## Port forward ArgoCD UI (runs in foreground)
	@echo "ArgoCD UI available at: https://localhost:8080"
	@echo "Username: admin"
	@echo "Password: $$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d)"
	@echo ""
	@echo "Press Ctrl+C to stop port forwarding"
	@kubectl port-forward svc/argocd-server -n argocd 8080:443

# ==============================================================================
# Crossplane
# ==============================================================================

##@ Crossplane

.PHONY: install-crossplane
install-crossplane: ## Install Crossplane with AWS Provider
	@echo "Installing Crossplane..."
	@helm repo add crossplane-stable https://charts.crossplane.io/stable 2>/dev/null || true
	@helm repo update
	@helm upgrade --install crossplane crossplane-stable/crossplane \
		--namespace crossplane-system \
		--create-namespace \
		--wait
	@echo "Waiting for Crossplane to be ready..."
	@kubectl wait --for=condition=available --timeout=300s deployment/crossplane -n crossplane-system
	@echo "Installing AWS IAM Provider..."
	@kubectl apply -f hack/crossplane/provider-aws.yaml
	@echo "Waiting for AWS Provider to initialize (this may take 1-2 minutes)..."
	@sleep 20
	@kubectl wait --for=condition=healthy --timeout=300s provider.pkg.crossplane.io/provider-aws-iam 2>/dev/null || echo "Provider still initializing, continuing..."
	@echo "✅ Crossplane with AWS Provider installed"

.PHONY: setup-localstack-provider
setup-localstack-provider: ## Configure Crossplane to use LocalStack
	@echo "Waiting for Provider CRDs to be ready..."
	@sleep 10
	@echo "Creating LocalStack ProviderConfig..."
	@kubectl apply -f hack/crossplane/providerconfig-localstack.yaml
	@echo "✅ LocalStack ProviderConfig created"

.PHONY: crossplane-status
crossplane-status: ## Check Crossplane status
	@echo "=== Providers ==="
	@kubectl get providers
	@echo ""
	@echo "=== ProviderConfigs ==="
	@kubectl get providerconfigs
	@echo ""
	@echo "=== Provider Pods ==="
	@kubectl get pods -n crossplane-system

# ==============================================================================
# LocalStack
# ==============================================================================

##@ LocalStack

.PHONY: localstack-start
localstack-start: ## Start LocalStack (run in separate terminal)
	@echo "Starting LocalStack..."
	@echo "NOTE: Run this in a separate terminal or use Docker:"
	@echo ""
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
	@echo ""
	@echo "1. Adding extra inline policy to tenant-alpha-lambda-role..."
	@awslocal iam put-role-policy \
		--role-name tenant-alpha-lambda-role \
		--policy-name unauthorized-extra-policy \
		--policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}' 2>/dev/null || echo "   Role may not exist yet"
	@echo ""
	@echo "2. Creating orphaned role (not managed by Crossplane)..."
	@awslocal iam create-role \
		--role-name orphaned-manual-role \
		--assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ec2.amazonaws.com"},"Action":"sts:AssumeRole"}]}' 2>/dev/null || echo "   Role may already exist"
	@echo ""
	@echo "✅ Drift test resources created"
	@echo ""
	@echo "To verify:"
	@echo "  awslocal iam list-roles"
	@echo "  awslocal iam list-role-policies --role-name tenant-alpha-lambda-role"

.PHONY: localstack-list-iam
localstack-list-iam: ## List IAM resources in LocalStack
	@echo "=== IAM Roles ==="
	@awslocal iam list-roles --query 'Roles[].RoleName' --output table
	@echo ""
	@echo "=== IAM Policies ==="
	@awslocal iam list-policies --scope Local --query 'Policies[].PolicyName' --output table

# ==============================================================================
# Tenant Seeding
# ==============================================================================

##@ Test Data

.PHONY: seed-tenants
seed-tenants: ## Create fake tenants and resources for testing
	@echo "Seeding test tenants..."
	@kubectl apply -f hack/seed-tenants/namespaces.yaml
	@kubectl apply -f hack/seed-tenants/tenant-alpha.yaml
	@kubectl apply -f hack/seed-tenants/tenant-beta.yaml
	@echo "Waiting for Crossplane CRDs to be ready..."
	@sleep 5
	@kubectl apply -f hack/seed-tenants/iam-resources.yaml 2>/dev/null || echo "IAM CRDs not ready yet, try 'make seed-iam' later"
	@kubectl apply -f hack/seed-tenants/argo-apps.yaml 2>/dev/null || echo "ArgoCD CRDs not ready yet"
	@echo "✅ Test tenants seeded"

.PHONY: seed-iam
seed-iam: ## Seed IAM resources (run after Crossplane provider is ready)
	@kubectl apply -f hack/seed-tenants/iam-resources.yaml
	@echo "✅ IAM resources seeded"

.PHONY: seed-failed-resources
seed-failed-resources: ## Create resources in failed/waiting states for testing
	@kubectl apply -f hack/seed-tenants/failed-resources.yaml
	@echo "✅ Failed resources seeded"

.PHONY: seed-paused-resources
seed-paused-resources: ## Pause some Crossplane resources for testing
	@kubectl annotate --overwrite roles.iam.aws.upbound.io/tenant-alpha-lambda-role crossplane.io/paused="true" 2>/dev/null || echo "Resource not found"
	@echo "✅ Resources paused"

.PHONY: clear-tenants
clear-tenants: ## Remove test tenant resources
	@kubectl delete -f hack/seed-tenants/ --ignore-not-found
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

.PHONY: web-lint
web-lint: ## Lint frontend code
	cd web && npm run lint

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

