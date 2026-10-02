# Deployment

The tested path today is running the manager locally against a kind cluster ([local-development.md](local-development.md)). The in-cluster manifests in `config/` came from Kubebuilder's scaffolding and deploy the controller. They still need the additions listed under [What the manifests are missing](#what-the-manifests-are-missing) before the full app works in a cluster.

## Backend

### Build and deploy

```sh
make docker-build docker-push IMG=<registry>/platform-manager:<tag>
make install                                   # CRDs
make deploy IMG=<registry>/platform-manager:<tag>
```

`make deploy` applies `config/default` into the `platform-manager-system` namespace, with the manager Deployment, its RBAC, and the metrics Service.

### Configuration

Flags (see `cmd/main.go`):

| Flag | Default | Purpose |
|------|---------|---------|
| `--api-bind-address` | `:9080` | REST and WebSocket API |
| `--health-probe-bind-address` | `:8081` | `/healthz` and `/readyz` for the kubelet |
| `--metrics-bind-address` | `0` (off) | controller-runtime metrics |
| `--prometheus-url` | kube-prometheus-stack service in `monitoring` | Source for CPU and memory metrics |
| `--iam-drift-scan-interval` | `5m` | How often to compare IAM with AWS |
| `--rule-eval-interval` | `2m` | How often to run troubleshooting rules |
| `--enable-terminal` | `false` | Turn on the web terminal |
| `--terminal-namespace` | `toolbox-sessions` | Where toolbox pods run |
| `--terminal-image` | `platform-manager-toolbox:latest` | Toolbox image |
| `--terminal-idle-timeout` | `10m` | Idle timeout per session |
| `--leader-elect` | `false` | Turn on when running more than one replica |

Environment variables:

| Variable | Purpose |
|----------|---------|
| `ALLOWED_ORIGINS` | Comma-separated origins allowed by CORS, for example `https://portal.example.com` |
| `TERMINAL_ALLOWED_ORIGINS` | Comma-separated origins allowed to open the terminal WebSocket. If empty, every WebSocket is rejected. |
| `AWS_REGION` | Defaults to `us-east-1` |
| `AWS_ENDPOINT` | Only for LocalStack. Leave unset in a real cluster. |
| `DEV_MODE` | Must be unset in any shared environment. See [security.md](security.md#development-mode). |

### AWS access

The drift scanner only calls read APIs: `GetRole`, `GetRolePolicy`, `ListRolePolicies`, `ListAttachedRolePolicies`, `GetPolicy`, `GetPolicyVersion`, `ListRoles`, and `ListPolicies`. On EKS, give the manager's ServiceAccount an IRSA role that allows only those actions. With no `AWS_ENDPOINT`, the SDK's default credential chain picks up IRSA.

### Web terminal

Build the toolbox image and make it pullable from the cluster:

```sh
make docker-build-toolbox TOOLBOX_IMG=<registry>/platform-manager-toolbox:<tag>
```

The terminal also needs:

1. The `toolbox-sessions` namespace.
2. The `toolbox-session` ServiceAccount with a read-only ClusterRole.
3. A Role in `toolbox-sessions` that lets the manager create, get, list, watch, and delete pods, and create `pods/exec`.

`helper-scripts/setup-phase6-terminal.sh` creates all three. Read it before running it against a real cluster.

### What the manifests are missing

- A Service for the API port (9080). The gateway needs one to route to the backend.
- `env` entries for `ALLOWED_ORIGINS`, `TERMINAL_ALLOWED_ORIGINS`, and `AWS_REGION`, and an IRSA annotation on the ServiceAccount.
- The terminal namespace and RBAC described above.
- A NetworkPolicy so that only the gateway can reach port 9080. This one matters for security: the API trusts identity headers, so anything else that can reach the port can impersonate a user.

## Frontend

The frontend builds two ways from the same code.

| Mode | Command | Output |
|------|---------|--------|
| Standalone SPA | `make web-build` | `web/dist/` with an `index.html`, served by any static host |
| Micro-frontend | `make web-build-mfe` or `make web-docker-build` | A SystemJS module for a single-spa shell |

`web/src/main.ts` exports single-spa `bootstrap`, `mount`, and `unmount`. If no single-spa shell is present, it mounts itself on `#app`, so one bundle works in both places.

### API base URL

The client reads `VUE_APP_API_URL` at build time and falls back to `http://localhost:9080/api/v1`. The `web/Dockerfile` does not set it, so the image currently calls localhost. Before you deploy it, add `ENV VUE_APP_API_URL=/api/v1` (or your gateway's path) to the builder stage.

### Container image

```sh
make web-docker-build MFE_IMG=<registry>/platform-manager-mfe:<tag>
make web-docker-push  MFE_IMG=<registry>/platform-manager-mfe:<tag>
```

The image is `nginx:1.25-alpine` running as the `nginx` user on port 8080. It serves:

| Path | Content |
|------|---------|
| `/js/app.<hash>.js` | The SystemJS module, plus lazy-loaded route chunks |
| `/css/app.<hash>.css` | Styles |
| `/manifest.json` | Name, version, and entry point, so a gateway can discover the module |
| `/health` | Returns `healthy` |

`manifest.json` currently says the entry point is `/js/app.js`. Vue CLI hashes filenames, so that path does not exist in the image. Either set `filenameHashing: false` for the MFE build in `vue.config.js`, or write the hashed name into the manifest at build time.

`config/frontend/deployment.yaml` deploys it as `platform-manager-mfe` in the `platform-system` namespace, with a Service and an Ingress. Change the image and the Ingress host before you apply it with `make web-deploy`.

### Loading it into a single-spa shell

The MFE bundle leaves `vue`, `vue-router`, and `pinia` out, so the shell's import map has to provide them:

```html
<script type="systemjs-importmap">
{
  "imports": {
    "@platform/manager-mfe": "https://<mfe-host>/js/app.<hash>.js",
    "vue": "https://cdn.jsdelivr.net/npm/vue@3.4.0/dist/vue.esm-browser.prod.js",
    "vue-router": "https://cdn.jsdelivr.net/npm/vue-router@4.2.5/dist/vue-router.esm-browser.js",
    "pinia": "https://cdn.jsdelivr.net/npm/pinia@2.1.7/dist/pinia.esm-browser.js"
  }
}
</script>
```

Register it with the shell:

```js
import { registerApplication, start } from 'single-spa'

registerApplication({
  name: '@platform/manager-mfe',
  app: () => System.import('@platform/manager-mfe'),
  activeWhen: ['/platform'],
})

start({ urlRerouteOnly: true })
```

The gateway then has to proxy API calls to the backend and allow WebSocket upgrades for the terminal. With nginx:

```nginx
location /api/ {
    proxy_pass http://<backend-service>:9080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 3600s;   # keep terminal sessions open
}
```

OAuth2 Proxy sits in front of this location and adds the `X-Auth-Request-*` headers. Configure it to strip any copies of those headers that come from the client.
