# Platform Manager Frontend - Docker Deployment Guide

This guide covers building and deploying the Vue.js frontend as a containerized application.

## Overview

The Platform Manager frontend is built as a **standalone Single Page Application (SPA)** using Vue 3, TypeScript, and Vue Router. It's deployed using nginx to serve static assets and proxy API requests to the backend.

## Architecture

```
┌─────────────────────────────────────────┐
│  Docker Container (nginx:1.25-alpine)   │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │  Static Assets (/dist)            │ │
│  │  - index.html                     │ │
│  │  - JS bundles (hashed)            │ │
│  │  - CSS (hashed)                   │ │
│  └───────────────────────────────────┘ │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │  nginx (port 8080)                │ │
│  │  - Serves static files            │ │
│  │  - SPA routing fallback           │ │
│  │  - Proxies /api/* to backend      │ │
│  │  - WebSocket support for terminal │ │
│  └───────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

## Prerequisites

- Docker 20.10+ with buildx support
- Docker registry (or local registry for testing)
- Kubernetes cluster with kubectl access (for deployment)

## Building the Image

### Quick Build (amd64)

```bash
cd /path/to/platform-manager
./helper-scripts/build-frontend-image.sh
```

### Custom Build

```bash
# Build for specific platform
PLATFORM=linux/amd64 ./helper-scripts/build-frontend-image.sh

# Build and push to registry
PUSH_IMAGE=true REGISTRY=your-registry.com ./helper-scripts/build-frontend-image.sh

# Custom image name and tag
IMAGE_NAME=my-frontend IMAGE_TAG=v1.0.0 ./helper-scripts/build-frontend-image.sh
```

### Multi-Architecture Build

For both amd64 and arm64:

```bash
cd web

docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag your-registry.com/platform-manager-frontend:latest \
  --push \
  -f Dockerfile \
  .
```

## Testing Locally

### Run Container Locally

```bash
# Run on port 8080
docker run -p 8080:8080 localhost:5001/platform-manager-frontend:latest

# Run with custom backend URL
docker run -p 8080:8080 \
  -e VUE_APP_API_BASE_URL=http://localhost:9080 \
  localhost:5001/platform-manager-frontend:latest

# Access at http://localhost:8080
```

### Test with Backend

```bash
# Terminal 1: Start backend
cd platform-manager
make run

# Terminal 2: Start frontend container
docker run -p 8080:8080 localhost:5001/platform-manager-frontend:latest

# Terminal 3: Test
curl http://localhost:8080/health  # Should return "healthy"
curl http://localhost:8080/api/tenants  # Proxied to backend
```

## Deploying to Kubernetes

### Prerequisites

1. Backend must be deployed first (or accessible)
2. Namespace `platform-system` must exist
3. Image must be accessible to cluster

### Deploy Frontend

```bash
# Create namespace if needed
kubectl create namespace platform-system

# Apply frontend deployment
kubectl apply -f config/frontend/deployment.yaml

# Verify deployment
kubectl -n platform-system get pods -l app=platform-manager-frontend
kubectl -n platform-system get svc platform-manager-frontend
```

### Check Deployment Status

```bash
# Check pods
kubectl -n platform-system get pods -l app=platform-manager-frontend

# Check logs
kubectl -n platform-system logs -l app=platform-manager-frontend --tail=50

# Check service
kubectl -n platform-system get svc platform-manager-frontend

# Port forward for testing
kubectl -n platform-system port-forward svc/platform-manager-frontend 8080:80
```

### Verify Functionality

```bash
# With port-forward active
curl http://localhost:8080/health

# Open in browser
open http://localhost:8080
```

## Configuration

### Environment Variables

The frontend container supports these environment variables:

- `VUE_APP_API_BASE_URL` - Backend API URL (default: uses relative paths)

### Backend Connection

The nginx configuration proxies these paths to the backend:

- `/api/*` - REST API endpoints
- `/api/terminal/*` - WebSocket connections for terminal

**Update the backend service name** in `config/frontend/deployment.yaml`:

```yaml
env:
- name: VUE_APP_API_BASE_URL
  value: "http://your-backend-service:9080"  # Update this
```

Or in nginx config (`web/Dockerfile`):

```nginx
location /api/ {
    proxy_pass http://your-backend-service:9080;
    ...
}
```

## Dockerfile Details

### Multi-Stage Build

1. **Builder Stage** (`node:20-alpine`)
   - Installs npm dependencies
   - Builds production bundle
   - Output: `/app/dist/` directory

2. **Production Stage** (`nginx:1.25-alpine`)
   - Copies built assets
   - Configures nginx
   - Runs as non-root user
   - Exposes port 8080

### Key Features

- ✅ Multi-architecture support (amd64/arm64)
- ✅ Non-root user (nginx:101)
- ✅ Health check endpoint (`/health`)
- ✅ Gzip compression
- ✅ Security headers
- ✅ SPA routing (history mode)
- ✅ Static asset caching
- ✅ API proxy with WebSocket support

## Troubleshooting

### Build Errors

**Error: npm install fails**
```bash
# Clear npm cache and retry
docker buildx build --no-cache -f web/Dockerfile web/
```

**Error: Memory limit exceeded**
```bash
# Increase Docker memory limit in Docker Desktop settings
# Or add to Dockerfile:
ENV NODE_OPTIONS="--max-old-space-size=4096"
```

### Runtime Errors

**API calls fail (404/CORS)**
- Check backend service name in nginx config
- Verify backend is running: `kubectl get pods -n platform-system`
- Check proxy configuration in nginx

**WebSocket connection fails**
- Ensure `/api/terminal/*` proxy is configured correctly
- Check backend WebSocket endpoint: `/api/terminal/:id/connect`
- Verify timeout settings in nginx (should be high for WebSockets)

**Static assets not loading**
- Check nginx logs: `kubectl logs -l app=platform-manager-frontend`
- Verify dist directory was copied: `docker run --rm --entrypoint ls <image> /usr/share/nginx/html`

### Logs

```bash
# Frontend container logs
kubectl -n platform-system logs -l app=platform-manager-frontend -f

# nginx access logs
kubectl -n platform-system exec -it <pod-name> -- tail -f /var/log/nginx/access.log

# nginx error logs
kubectl -n platform-system exec -it <pod-name> -- tail -f /var/log/nginx/error.log
```

## Performance Optimization

### Build Time

- Use `.dockerignore` to exclude unnecessary files
- Use `npm ci` instead of `npm install` (faster, deterministic)
- Cache Docker layers (dependencies change less than source)

### Runtime Performance

- Gzip compression enabled (reduces transfer size by ~70%)
- Static assets cached for 1 year (with immutable cache-control)
- CDN-friendly (all assets have content hashes)

### Image Size

Current image size: ~25-30 MB (compressed)

- Uses Alpine Linux (minimal base)
- Multi-stage build (no dev dependencies)
- Only production assets included

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Build Frontend
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3
    
    - name: Build and push
      run: |
        cd platform-manager
        REGISTRY=${{ secrets.REGISTRY }} \
        IMAGE_TAG=${{ github.sha }} \
        PUSH_IMAGE=true \
        ./helper-scripts/build-frontend-image.sh
```

## Security Considerations

1. **Non-root user**: Runs as nginx (UID 101)
2. **Security headers**: X-Frame-Options, X-Content-Type-Options, etc.
3. **Read-only root filesystem**: Considered but disabled (nginx needs cache dir)
4. **Minimal attack surface**: Alpine base, no shell in production stage
5. **No secrets in image**: Use environment variables or Kubernetes secrets

## MFE vs Standalone Deployment

### Current: Standalone SPA (Recommended)
- ✅ Simple to deploy
- ✅ Self-contained
- ✅ No external dependencies
- ✅ Works independently

### Alternative: Micro-Frontend (MFE)
If you want to deploy as a micro-frontend:

```bash
# Build as MFE
cd web
BUILD_MODE=mfe npm run build

# Update Dockerfile to serve as library
# Requires shell application to load it
```

**When to use MFE:**
- Multiple teams managing different UIs
- Need to compose multiple frontend apps
- Dynamic loading of features

**When to use Standalone (current):**
- Single application
- Simpler deployment
- Better initial load performance

## Next Steps

1. **Set up Ingress**: Configure external access (see `config/frontend/deployment.yaml`)
2. **Add TLS**: Use cert-manager for HTTPS
3. **CDN Integration**: For production, consider CloudFront/CloudFlare
4. **Monitoring**: Add Prometheus metrics, logging

## References

- Vue CLI: https://cli.vuejs.org/
- nginx Docker: https://hub.docker.com/_/nginx
- Docker Multi-arch: https://docs.docker.com/build/building/multi-platform/

