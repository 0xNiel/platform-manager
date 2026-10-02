# API reference

The API server listens on `:9080` (set with `--api-bind-address`). Every endpoint returns JSON. Routes are registered in `internal/api/server.go` and in each handler's `RegisterRoutes`.

## Authentication

The API expects to run behind OAuth2 Proxy, and it reads identity from these headers:

| Header | Use |
|--------|-----|
| `X-Auth-Request-User` | Username, recorded in audit logs |
| `X-Auth-Request-Email` | Email |
| `X-Auth-Request-Groups` | Comma-separated groups, mapped to a role |

A request without these headers runs as `readonly`. With `DEV_MODE=true`, you can set the role directly with `X-Dev-Role: admin|infra|ml|readonly`. That is for local development only. See [security.md](security.md).

```sh
curl -H 'X-Dev-Role: admin' localhost:9080/api/v1/auth/me
```

## Probes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Liveness |
| GET | `/readyz` | Readiness |

## Auth

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/auth/me` | Current user, role, and groups |
| GET | `/api/v1/auth/capabilities` | Capabilities granted to the current role |

## Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health/platform` | Platform-wide rollup |
| GET | `/api/v1/health/tenants` | `TenantHealth` for every tenant |
| GET | `/api/v1/health/tenants/{id}` | `TenantHealth` for one tenant |

## Tenants and resources

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tenants` | List tenants |
| GET | `/api/v1/tenants/{id}` | One tenant |
| GET | `/api/v1/tenants/{id}/resources` | A tenant's resources. Supports `state`, `kind`, and `category` filters. |
| GET | `/api/v1/resources` | All tracked resources. Filters: `tenant`, `state`, `kind`, `category`, `provider`, `search`. |
| GET | `/api/v1/resources/{name}` | One resource. Add `yaml=true`, `events=true`, or `tree=true` for extra detail. |

## Metrics

These endpoints query Prometheus (`--prometheus-url`). If Prometheus is unreachable, they return an error. Nothing else depends on them.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/metrics/namespaces/{namespace}` | CPU and memory for a namespace |
| GET | `/api/v1/metrics/tenants/{id}` | CPU and memory summed across a tenant's namespaces |
| GET | `/api/v1/metrics/pods/{namespace}/{podName}` | CPU and memory for one pod |

## Actions

Each action needs the capability listed. A denied request returns `403` and increments `platform_manager_auth_denials_total`. Once a request passes the check, the handler writes an audit log entry with the user, action, resource, and result, whether it succeeds or fails. Denied requests show up in the metric but not in the audit log.

| Method | Path | Capability | Body |
|--------|------|------------|------|
| POST | `/api/v1/actions/argo/sync` | `argo:sync` | `{"name", "namespace", "prune", "force", "dryRun"}` |
| POST | `/api/v1/actions/argo/refresh` | `argo:refresh` | `{"name", "namespace"}` |
| POST | `/api/v1/actions/crossplane/pause` | `crossplane:pause` | resource reference |
| POST | `/api/v1/actions/crossplane/unpause` | `crossplane:pause` | resource reference |
| POST | `/api/v1/actions/crossplane/reconcile` | `crossplane:reconcile` | resource reference |
| DELETE | `/api/v1/actions/resources` | `resource:delete` | resource reference plus `"confirm"` |

A resource reference looks like this:

```json
{
  "group": "iam.aws.upbound.io",
  "version": "v1beta1",
  "kind": "Role",
  "name": "tenant-alpha-lambda-role"
}
```

To delete, `confirm` must equal `name`. That stops a client bug from deleting the wrong thing.

Pausing sets `crossplane.io/paused: "true"` on the resource. Reconcile updates an annotation, which makes Crossplane reconcile it right away.

## IAM drift

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/iam/drift/platform` | Platform drift summary and last scan time |
| GET | `/api/v1/iam/drift/tenants` | Drift summary for each tenant |
| GET | `/api/v1/iam/drift/tenants/{tenant}` | Detailed findings for one tenant, with expected and actual documents |
| POST | `/api/v1/iam/drift/scan` | Start a scan now instead of waiting for the next interval |

## Troubleshooting

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/troubleshooting/summary` | Finding counts by severity and tenant |
| GET | `/api/v1/troubleshooting/findings` | Active findings. Filters: `tenant`, `severity`. |
| GET | `/api/v1/troubleshooting/findings/{id}` | One finding |
| GET | `/api/v1/troubleshooting/tenants/{name}` | Findings for one tenant |
| POST | `/api/v1/troubleshooting/findings/{id}/resolve` | Mark a finding resolved |
| POST | `/api/v1/troubleshooting/scan` | Run every rule now |
| GET | `/api/v1/troubleshooting/rules` | List registered rules |

## Terminal

These routes exist only when the manager starts with `--enable-terminal`.

| Method | Path | Capability | Description |
|--------|------|------------|-------------|
| GET | `/api/v1/terminal/config` | none | Whether the terminal is on and whether the caller can use it |
| POST | `/api/v1/terminal/sessions` | `terminal:use` | Create a session. Returns `sessionId` and `wsUrl`. |
| GET | `/api/v1/terminal/sessions` | `terminal:use` | List active sessions |
| GET | `/api/v1/terminal/sessions/{sessionId}` | `terminal:use` | Session status |
| DELETE | `/api/v1/terminal/sessions/{sessionId}` | `terminal:use` | End a session |
| GET (WebSocket) | `/api/v1/terminal/sessions/{sessionId}/ws` | valid session ID and allowed `Origin` | Terminal stream |

The WebSocket carries raw terminal bytes in both directions.

## Prometheus metrics

The manager exports these alongside the standard controller-runtime metrics:

| Metric | Type | Labels |
|--------|------|--------|
| `platform_manager_actions_total` | counter | action, status, user_role |
| `platform_manager_auth_denials_total` | counter | action, user_role |
| `platform_manager_action_duration_seconds` | histogram | action |
