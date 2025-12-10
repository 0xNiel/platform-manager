# Platform Manager

A unified control plane for managing Crossplane, ArgoCD, and AWS IAM resources across a multi-tenant Kubernetes platform.

## Features

- **Platform Dashboard**: 30,000-foot view of platform health
- **Tenant Management**: Multi-tenant resource tracking and health aggregation
- **Crossplane Integration**: Watch and manage Compositions, Claims, XRs, and Providers
- **ArgoCD Integration**: Sync status, health monitoring, and GitOps operations
- **IAM Drift Detection**: Track drift between Crossplane and AWS IAM resources
- **Web Terminal**: Secure, audited kubectl/aws CLI access
- **Rule-based Troubleshooting**: Automated issue detection and recommendations

## Quick Start

### Prerequisites

Ensure you have the following installed:

```bash
# Required tools
brew install kind kubectl helm ko argocd awscli
pip3 install localstack awscli-local

# Optional but recommended
brew install stern k9s jq

# Frontend development
brew install node
```

### Development Environment

1. **Start LocalStack** (in a separate terminal):
   ```bash
   localstack start
   # OR
   docker run --rm -p 4566:4566 -p 4510-4559:4510-4559 localstack/localstack
   ```

2. **Create the development environment**:
   ```bash
   make dev-up
   ```

3. **Verify everything is running**:
   ```bash
   make dev-status
   ```

4. **Access ArgoCD UI**:
   ```bash
   make argocd-port-forward
   # Open https://localhost:8080
   # Username: admin
   # Password: make argocd-password
   ```

### Project Structure

```
platform-manager/
├── api/v1alpha1/         # CRD definitions
├── cmd/                  # Main entrypoint
├── internal/
│   ├── controller/       # Kubernetes controllers
│   ├── api/              # HTTP API handlers
│   ├── iam/              # IAM drift detection
│   ├── health/           # Health aggregation
│   ├── rules/            # Troubleshooting rules
│   ├── terminal/         # Web terminal
│   └── metrics/          # Prometheus integration
├── config/               # Kubebuilder manifests
├── hack/                 # Development scripts
│   ├── kind-config.yaml
│   ├── crossplane/
│   └── seed-tenants/
├── web/                  # Vue 3 frontend (single-spa MFE)
└── test/                 # Tests
```

## Development

### Backend

```bash
# Run controller locally
make run

# Run tests
make test

# Generate CRDs
make manifests

# Build Docker image
make docker-build
```

### Frontend

```bash
# Install dependencies
make web-install

# Run dev server
make web-dev

# Build for production
make web-build

# Build as single-spa MFE
make web-build-mfe
```

### Testing with LocalStack

```bash
# Check LocalStack health
make localstack-health

# Create IAM drift for testing
make localstack-create-drift

# List IAM resources
make localstack-list-iam
```

## Architecture

### Backend

- **Go 1.22+** with Kubebuilder v4
- Controller-runtime for Kubernetes operations
- Chi/Echo for HTTP API
- aws-sdk-go-v2 for AWS operations
- gorilla/websocket for terminal

### Frontend

- **Vue 3** with TypeScript
- **single-spa** for MFE integration
- Pinia for state management
- xterm.js for terminal

### Crossplane Providers Watched

- AWS family (IAM, S3, Lambda, etc.)
- Custom KnowledgeBases
- Ansible, ArgoCD, Grafana, Helm, Kubernetes, OpenTofu, Terraform, Vault

## Deployment

### Kind (Development)

```bash
make deploy-dev
```

### Production (amd64)

```bash
make docker-build-prod
```

## Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `PROMETHEUS_URL` | Prometheus endpoint | `http://prometheus:9090` |
| `AWS_REGION` | AWS region | `us-east-1` |
| `LOG_LEVEL` | Logging level | `info` |

## License

Apache 2.0

