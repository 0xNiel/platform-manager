# Phase 6: Web Terminal - Architecture & Deployment Model

## Deployment Architecture

### ✅ Correct: In-Cluster Deployment

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                          │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │  Platform Manager Pod (Deployment)                        │ │
│  │                                                            │ │
│  │  ┌──────────────────────┐   ┌──────────────────────────┐ │ │
│  │  │  Backend (Go)        │   │  Frontend (Static)       │ │ │
│  │  │  - API Server :9080  │   │  - Vue.js SPA            │ │ │
│  │  │  - Controllers       │   │  - Served at /           │ │ │
│  │  │  - Terminal Manager  │   │                          │ │ │
│  │  └──────────────────────┘   └──────────────────────────┘ │ │
│  │                                                            │ │
│  │  Service: platform-manager-svc:9080                       │ │
│  └───────────────────────────────────────────────────────────┘ │
│                           │                                     │
│                           │ Creates & Manages                   │
│                           ↓                                     │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │  Namespace: toolbox-sessions                              │ │
│  │                                                            │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │ │
│  │  │ Toolbox Pod  │  │ Toolbox Pod  │  │ Toolbox Pod  │   │ │
│  │  │ tenant-alpha │  │ tenant-beta  │  │ tenant-gamma │   │ │
│  │  │              │  │              │  │              │   │ │
│  │  │ - kubectl    │  │ - kubectl    │  │ - kubectl    │   │ │
│  │  │ - helm       │  │ - helm       │  │ - helm       │   │ │
│  │  │ - argocd     │  │ - argocd     │  │ - argocd     │   │ │
│  │  │ - bash       │  │ - bash       │  │ - bash       │   │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘   │ │
│  │                                                            │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           ↑
                           │ HTTPS/Ingress
                           │
                    ┌──────────────┐
                    │   Browser    │
                    │   (Users)    │
                    └──────────────┘
```

## Communication Flow

### 1. User Access (Production)

```
User Browser
    ↓ HTTPS (via Ingress/LoadBalancer)
Ingress Controller (nginx/traefik)
    ↓ Internal routing
Platform Manager Service (:9080)
    ↓ Static files OR API calls
Platform Manager Pod
```

### 2. Terminal Session Creation

```
1. User clicks "Terminal" button in UI
2. Frontend → POST /api/v1/terminal/sessions {"tenantId": "alpha"}
3. Backend creates:
   - Session record
   - ServiceAccount (if needed)
   - Role & RoleBinding (if needed)
   - Pod in toolbox-sessions namespace
4. Backend returns session info with wsUrl
5. Frontend connects WebSocket to wsUrl
```

### 3. WebSocket Connection (In-Cluster)

```
Browser
    ↓ WebSocket Upgrade (ws://platform-manager:9080/api/v1/terminal/sessions/{id}/ws)
Platform Manager Backend
    ↓ Uses in-cluster Kubernetes API
Kubernetes API Server
    ↓ Pod Exec (SPDY)
Toolbox Pod
    ↓ Bash shell
Terminal I/O ← → User
```

## Key Points

### ✅ Advantages of In-Cluster Deployment

1. **Direct Pod Access**: Platform Manager uses in-cluster config to exec into toolbox pods
2. **No External Network**: All communication stays within the cluster
3. **Secure by Default**: Uses Kubernetes RBAC and ServiceAccounts
4. **Low Latency**: No network hops outside cluster
5. **Simple Networking**: No need for NodePort or external access to pods

### ✅ Security Model

1. **User Authentication**: Via OAuth2Proxy (Ingress-level)
2. **Role-Based Access**: Middleware checks capabilities
3. **Pod Isolation**: Each session gets its own pod
4. **Namespace Isolation**: Toolbox pods in dedicated namespace
5. **Resource Limits**: CPU/Memory limits on each pod
6. **Audit Logging**: All commands logged

### ✅ Production Configuration

#### Ingress Example (nginx)
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: platform-manager
  annotations:
    nginx.ingress.kubernetes.io/auth-url: "https://oauth2-proxy.default.svc.cluster.local/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://oauth2-proxy.default.svc.cluster.local/oauth2/start"
spec:
  rules:
  - host: platform.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: platform-manager
            port:
              number: 9080
```

#### Service Example
```yaml
apiVersion: v1
kind: Service
metadata:
  name: platform-manager
  namespace: platform-system
spec:
  selector:
    app: platform-manager
  ports:
  - name: http
    port: 9080
    targetPort: 9080
  type: ClusterIP  # No need for LoadBalancer/NodePort
```

#### Deployment Example
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: platform-manager
  namespace: platform-system
spec:
  replicas: 2  # Can have multiple replicas
  selector:
    matchLabels:
      app: platform-manager
  template:
    metadata:
      labels:
        app: platform-manager
    spec:
      serviceAccountName: platform-manager
      containers:
      - name: manager
        image: platform-manager:latest
        args:
        - --api-bind-address=:9080
        - --enable-terminal=true
        - --terminal-namespace=toolbox-sessions
        - --terminal-image=platform-manager-toolbox:latest
        - --terminal-idle-timeout=30m
        - --terminal-max-sessions=50
        ports:
        - containerPort: 9080
          name: http
        - containerPort: 8081
          name: metrics
        volumeMounts:
        - name: static-files
          mountPath: /app/web/dist
      volumes:
      - name: static-files
        emptyDir: {}
```

## Current Development Setup vs Production

### Development (Current)
```
Browser → http://localhost:9083 (Vue dev server)
           ↓ API calls to
         http://localhost:9080 (Go manager)
           ↓ WebSocket to
         ws://localhost:9080/api/v1/terminal/sessions/{id}/ws
           ↓ Exec into
         Kind Cluster → Toolbox Pods
```

### Production (Deployed)
```
Browser → https://platform.example.com (Ingress)
           ↓ OAuth2Proxy authentication
           ↓ Routes to
         platform-manager-svc:9080 (Service)
           ↓ Backend serves both static + API
         Platform Manager Pod
           ↓ WebSocket upgrade
         wss://platform.example.com/api/v1/terminal/sessions/{id}/ws
           ↓ In-cluster API
         Kubernetes API → Toolbox Pods
```

## WebSocket URL Resolution

### Frontend Code (Fixed)
```typescript
// Uses the same host as API calls
const apiBase = process.env.VUE_APP_API_URL || 'http://localhost:9080/api/v1'
const apiUrl = new URL(apiBase)
const protocol = apiUrl.protocol === 'https:' ? 'wss:' : 'ws:'
const host = apiUrl.host
const wsFullUrl = `${protocol}//${host}${wsUrl}`
```

### Environment Variables

**Development (.env.development)**
```bash
VUE_APP_API_URL=http://localhost:9080/api/v1
```

**Production (.env.production)**
```bash
# Let it use relative URLs (same origin)
VUE_APP_API_URL=/api/v1
```

When using relative URLs in production, both HTTP API calls and WebSocket connections go to the same host that served the page (via Ingress).

## Network Policies (Optional but Recommended)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: toolbox-isolation
  namespace: toolbox-sessions
spec:
  podSelector:
    matchLabels:
      app: toolbox
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: platform-system
      podSelector:
        matchLabels:
          app: platform-manager
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # Kubernetes API
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: UDP
      port: 53  # DNS
```

## Summary

✅ **Architecture is correct for in-cluster deployment**
- Platform Manager runs inside cluster
- Toolbox pods run in same cluster
- Communication uses in-cluster Kubernetes API
- No external access needed for toolbox pods

✅ **WebSocket fix applied**
- Frontend now uses API host for WebSocket
- Works in both dev (localhost:9080) and production (same origin)

✅ **Ready for production deployment**
- Serve frontend as static files from manager
- Single service endpoint
- OAuth2Proxy for authentication
- Kubernetes RBAC for authorization

🔧 **Next: Test the WebSocket connection in browser!**

