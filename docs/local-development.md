# Local development

The local setup is a kind cluster with ArgoCD and Crossplane. LocalStack stands in for AWS, and you run the manager and the UI from your machine. Everything is driven by `make`.

## Prerequisites

- Go 1.24+
- Docker (Docker Desktop on macOS, so that `host.docker.internal` resolves from inside kind)
- kind, kubectl, Helm
- Node 20 and npm
- LocalStack CLI, or just Docker
- `awslocal` (`pip install awscli-local`), used to create drift on purpose

## 1. Start LocalStack

Run it in its own terminal:

```sh
localstack start
# or
docker run --rm -p 4566:4566 localstack/localstack
```

The Crossplane AWS provider in the cluster reaches it at `http://host.docker.internal:4566` (`LOCALSTACK_ENDPOINT` in the Makefile). On Linux, change that to an address the kind node can reach.

## 2. Create the cluster

```sh
make dev-up
```

This creates a kind cluster named `platform-manager` and installs ArgoCD and Crossplane with the Upbound AWS IAM provider. It also points the provider at LocalStack and seeds three tenant namespaces, IAM roles and policies, and ArgoCD apps.

The AWS provider can take a minute or two to become healthy. If the IAM resources fail to apply on the first run, wait and run `make seed-iam`.

Check that Crossplane created the roles in LocalStack:

```sh
kubectl get roles.iam.aws.upbound.io
awslocal iam list-roles
```

## 3. Run the manager

```sh
DEV_MODE=true AWS_ENDPOINT=http://localhost:4566 make run-local
```

`run-local` installs the CRDs, applies the sample `Tenant` objects from `config/samples/`, and runs the manager with `go run`. The API is on `:9080` and the health probes are on `:8082`.

`DEV_MODE=true` lets you pick a role with the `X-Dev-Role` header:

```sh
curl -s localhost:9080/api/v1/health/platform | jq
curl -s -H 'X-Dev-Role: admin' localhost:9080/api/v1/auth/capabilities | jq
```

## 4. Run the UI

```sh
make web-install   # once
make web-dev
```

Open <http://localhost:9082>. The dev server proxies `/api` to `localhost:9080`.

## Seeding scenarios

The seed data starts out healthy. These targets break things so there is something for the UI to show:

| Command | What it creates |
|---------|-----------------|
| `make seed-failed-resources` | Pods and Deployments stuck in failed or waiting states, which the troubleshooting rules pick up |
| `make seed-paused-resources` | Pauses `tenant-alpha-lambda-role` with `crossplane.io/paused` |
| `make localstack-create-drift` | Attaches an `s3:*` inline policy to `tenant-alpha-lambda-role` directly in LocalStack, plus a role that Crossplane does not manage |

After creating drift, start a scan right away instead of waiting 5 minutes:

```sh
curl -X POST localhost:9080/api/v1/iam/drift/scan
curl -s localhost:9080/api/v1/iam/drift/tenants/tenant-alpha | jq
```

The inline policy shows up as `extra_privileges`. The unmanaged role does not show up yet, because orphan detection is not implemented.

`make clear-tenants` removes the seeded tenant resources.

## Web terminal

The terminal is off by default. To try it locally:

```sh
# Build the toolbox image and load it into kind
make kind-load-toolbox

# Create the toolbox namespace, ServiceAccount, and RBAC
./helper-scripts/setup-phase6-terminal.sh

# Run the manager with the terminal on
DEV_MODE=true AWS_ENDPOINT=http://localhost:4566 \
  go run ./cmd/main.go \
    --health-probe-bind-address=:8082 \
    --api-bind-address=:9080 \
    --enable-terminal \
    --terminal-image=platform-manager-toolbox:dev
```

Then open the terminal drawer in the UI. In development builds, the UI sends `X-Dev-Role: admin` on every request (`web/src/api/client.ts`), so you already have `terminal:use`.

## Tests and linting

```sh
make test       # unit and controller tests (Ginkgo + envtest)
make lint       # golangci-lint
make test-e2e   # builds the image and runs e2e tests in a separate kind cluster
```

`make test` downloads the envtest binaries into `bin/` on the first run. The same three targets run in GitHub Actions (`.github/workflows/`).

For the frontend:

```sh
cd web
npm run lint
npm run type-check
```

## Ports

| Port | Service |
|------|---------|
| 4566 | LocalStack |
| 8082 | Manager health probes (`run-local`) |
| 9080 | Manager API |
| 9082 | UI dev server |

## Tear down

```sh
make dev-down
```
