# Image URL to use all building/pushing image targets
IMG ?= controller:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet setup-envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# TODO(user): To use a different vendor for e2e tests, modify the setup under 'tests/e2e'.
# The default setup assumes Kind is pre-installed and builds/loads the Manager Docker image locally.
# CertManager is installed by default; skip with:
# - CERT_MANAGER_INSTALL_SKIP=true
KIND_CLUSTER ?= platform-manager-test-e2e

.PHONY: setup-test-e2e
setup-test-e2e: ## Set up a Kind cluster for e2e tests if it does not exist
	@command -v $(KIND) >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@case "$$($(KIND) get clusters)" in \
		*"$(KIND_CLUSTER)"*) \
			echo "Kind cluster '$(KIND_CLUSTER)' already exists. Skipping creation." ;; \
		*) \
			echo "Creating Kind cluster '$(KIND_CLUSTER)'..."; \
			$(KIND) create cluster --name $(KIND_CLUSTER) ;; \
	esac

.PHONY: test-e2e
test-e2e: setup-test-e2e manifests generate fmt vet ## Run the e2e tests. Expected an isolated environment using Kind.
	KIND_CLUSTER=$(KIND_CLUSTER) go test ./test/e2e/ -v -ginkgo.v
	$(MAKE) cleanup-test-e2e

.PHONY: cleanup-test-e2e
cleanup-test-e2e: ## Tear down the Kind cluster used for e2e tests
	@$(KIND) delete cluster --name $(KIND_CLUSTER)

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	$(GOLANGCI_LINT) config verify

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name platform-manager-builder
	$(CONTAINER_TOOL) buildx use platform-manager-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm platform-manager-builder
	rm Dockerfile.cross

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > dist/install.yaml

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KIND ?= kind
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
KUSTOMIZE_VERSION ?= v5.6.0
CONTROLLER_TOOLS_VERSION ?= v0.18.0
#ENVTEST_VERSION is the version of controller-runtime release branch to fetch the envtest setup script (i.e. release-0.20)
ENVTEST_VERSION ?= $(shell go list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
#ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')
GOLANGCI_LINT_VERSION ?= v2.1.6

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

# ==============================================================================
# Platform Manager - Development Environment
# ==============================================================================

# Development cluster variables
KIND_CLUSTER_NAME ?= platform-manager
LOCALSTACK_ENDPOINT ?= http://host.docker.internal:4566
AWS_REGION ?= us-east-1
TOOLBOX_IMG ?= platform-manager-toolbox:dev
CROSSPLANE_VERSION ?= 1.20.0

##@ Development Environment

.PHONY: dev-up
dev-up: kind-create install-argocd install-crossplane setup-localstack-provider seed-tenants ## Create full dev environment
	@echo "[OK] Development environment ready!"
	@echo ""
	@echo "[INFO] Quick Reference:"
	@echo "   ArgoCD UI: https://localhost:9080 (or use: make argocd-port-forward)"
	@echo "   ArgoCD Password: $$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d)"
	@echo "   Platform API: http://localhost:9081"
	@echo "   Frontend Dev: http://localhost:9082"
	@echo "   LocalStack: $(LOCALSTACK_ENDPOINT)"
	@echo ""
	@echo "[TIP] Useful commands:"
	@echo "   make argocd-port-forward  # Access ArgoCD UI on localhost:8443"
	@echo "   make localstack-health    # Check LocalStack"
	@echo "   kubectl get providers     # Check Crossplane providers"

.PHONY: dev-down
dev-down: ## Tear down dev environment
	$(KIND) delete cluster --name $(KIND_CLUSTER_NAME)
	@echo "[OK] Development environment deleted"

.PHONY: dev-reset
dev-reset: dev-down dev-up ## Reset dev environment

##@ Kind Cluster

.PHONY: kind-create
kind-create: ## Create Kind cluster
	@if $(KIND) get clusters | grep -q $(KIND_CLUSTER_NAME); then \
		echo "Cluster $(KIND_CLUSTER_NAME) already exists"; \
	else \
		$(KIND) create cluster --name $(KIND_CLUSTER_NAME) --config hack/kind-config.yaml; \
	fi
	$(KUBECTL) cluster-info --context kind-$(KIND_CLUSTER_NAME)

.PHONY: kind-delete
kind-delete: ## Delete Kind cluster
	$(KIND) delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: kind-load-image
kind-load-image: docker-build ## Load image into Kind cluster
	$(KIND) load docker-image $(IMG) --name $(KIND_CLUSTER_NAME)

.PHONY: kind-load-toolbox
kind-load-toolbox: docker-build-toolbox ## Load toolbox image into Kind cluster
	$(KIND) load docker-image $(TOOLBOX_IMG) --name $(KIND_CLUSTER_NAME)

##@ ArgoCD

.PHONY: install-argocd
install-argocd: ## Install ArgoCD
	@echo "Installing ArgoCD..."
	$(KUBECTL) create namespace argocd --dry-run=client -o yaml | $(KUBECTL) apply -f -
	$(KUBECTL) apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
	@echo "Waiting for ArgoCD to be ready..."
	$(KUBECTL) wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd
	@echo "[OK] ArgoCD installed"
	@echo "   Get password: make argocd-password"
	@echo "   Port forward: make argocd-port-forward"

.PHONY: argocd-password
argocd-password: ## Get ArgoCD admin password
	@$(KUBECTL) -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d && echo

.PHONY: argocd-port-forward
argocd-port-forward: ## Port forward ArgoCD UI
	$(KUBECTL) port-forward svc/argocd-server -n argocd 8443:443

##@ Crossplane

.PHONY: install-crossplane
install-crossplane: ## Install Crossplane with AWS Provider
	@echo "Installing Crossplane v$(CROSSPLANE_VERSION)..."
	helm repo add crossplane-stable https://charts.crossplane.io/stable || true
	helm repo update
	helm upgrade --install crossplane crossplane-stable/crossplane \
		--namespace crossplane-system \
		--create-namespace \
		--version $(CROSSPLANE_VERSION) \
		--wait
	@echo "Waiting for Crossplane to be ready..."
	$(KUBECTL) wait --for=condition=available --timeout=300s deployment/crossplane -n crossplane-system
	@echo "Installing AWS IAM Provider..."
	$(KUBECTL) apply -f hack/crossplane/provider-aws.yaml
	@echo "Waiting for AWS Provider to initialize (this may take 1-2 minutes)..."
	@sleep 15
	$(KUBECTL) wait --for=condition=healthy --timeout=300s provider.pkg.crossplane.io/provider-aws-iam || echo "Provider still initializing, continuing..."
	@echo "[OK] Crossplane v$(CROSSPLANE_VERSION) with AWS Provider installed"

.PHONY: setup-localstack-provider
setup-localstack-provider: ## Configure Crossplane to use LocalStack
	@echo "Waiting for Provider CRDs to be ready..."
	@sleep 10
	@echo "Creating LocalStack ProviderConfig..."
	$(KUBECTL) apply -f hack/crossplane/providerconfig-localstack.yaml
	@echo "[OK] LocalStack ProviderConfig created"

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
	@echo "[OK] Drift test resources created"
	@echo ""
	@echo "To verify:"
	@echo "  awslocal iam list-roles"
	@echo "  awslocal iam list-role-policies --role-name tenant-alpha-lambda-role"

##@ Test Data

.PHONY: seed-tenants
seed-tenants: ## Create fake tenants and resources for testing
	@echo "Seeding test tenants..."
	$(KUBECTL) apply -f hack/seed-tenants/namespaces.yaml
	$(KUBECTL) apply -f hack/seed-tenants/tenant-alpha.yaml
	$(KUBECTL) apply -f hack/seed-tenants/tenant-beta.yaml
	@echo "Waiting for Crossplane CRDs to be ready..."
	@sleep 5
	$(KUBECTL) apply -f hack/seed-tenants/iam-resources.yaml || echo "IAM CRDs not ready yet, try 'make seed-iam' later"
	$(KUBECTL) apply -f hack/seed-tenants/argo-apps.yaml
	@echo "[OK] Test tenants seeded"

.PHONY: seed-iam
seed-iam: ## Apply IAM resources (after Crossplane CRDs are ready)
	$(KUBECTL) apply -f hack/seed-tenants/iam-resources.yaml
	@echo "[OK] IAM resources created"

.PHONY: seed-platform-tenants
seed-platform-tenants: ## Create Platform Manager Tenant CRDs
	$(KUBECTL) apply -f config/samples/platform_v1alpha1_tenant.yaml
	@echo "[OK] Platform Tenant CRDs created"

.PHONY: seed-failed-resources
seed-failed-resources: ## Create resources in failed/waiting states for testing
	$(KUBECTL) apply -f hack/seed-tenants/failed-resources.yaml
	@echo "[OK] Failed resources seeded"

.PHONY: seed-paused-resources
seed-paused-resources: ## Pause some Crossplane resources for testing
	$(KUBECTL) annotate --overwrite roles.iam.aws.upbound.io/tenant-alpha-lambda-role crossplane.io/paused="true" || echo "Resource not found"
	@echo "[OK] Resources paused"

.PHONY: clear-tenants
clear-tenants: ## Remove test tenant resources
	$(KUBECTL) delete -f hack/seed-tenants/ --ignore-not-found
	@echo "[OK] Test tenants cleared"

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

##@ Toolbox

.PHONY: docker-build-toolbox
docker-build-toolbox: ## Build toolbox Docker image
	$(CONTAINER_TOOL) build -t $(TOOLBOX_IMG) -f Dockerfile.toolbox .

##@ Quick Start

.PHONY: run-local
run-local: install seed-platform-tenants ## Run controller locally with CRDs installed
	go run ./cmd/main.go --health-probe-bind-address=:8082 --api-bind-address=:9080
