# Platform Manager - Micro-Frontend (MFE) Deployment Guide

This guide covers building and deploying the Platform Manager as a **Micro-Frontend (MFE)** that integrates with your gateway/shell application.

## Overview

The Platform Manager is built as a **single-spa compatible micro-frontend** using Vue 3, TypeScript, and the SystemJS module format. It's designed to be dynamically loaded by a shell/gateway application at runtime.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Gateway/Shell Application                              │
│  - Loads MFEs dynamically                               │
│  - Provides shared dependencies (vue, vue-router, etc)  │
│  - Handles routing & navigation                         │
│                                                         │
│  ┌──────────────────────────────────────────────┐      │
│  │  SystemJS Import Map                         │      │
│  │  {                                           │      │
│  │    "@platform/manager-mfe":                  │      │
│  │      "http://platform-mfe/js/app.js"         │      │
│  │  }                                           │      │
│  └──────────────────────────────────────────────┘      │
│                                                         │
│         ↓ loads dynamically                             │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│  Platform Manager MFE Container                         │
│  (nginx serving static SystemJS bundle)                 │
│                                                         │
│  ┌──────────────────────────────────────────────┐      │
│  │  /manifest.json     → MFE metadata           │      │
│  │  /js/app.js         → SystemJS bundle        │      │
│  │  /css/app.css       → Styles                 │      │
│  │  /health            → Health check           │      │
│  └──────────────────────────────────────────────┘      │
│                                                         │
│  Features:                                              │
│  - CORS enabled (allow gateway to load)                │
│  - Aggressive caching (1 year for JS/CSS)              │
│  - No routing (handled by gateway)                     │
│  - No API proxy (handled by gateway)                   │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│  Backend API Service                                    │
│  - Gateway proxies API calls here                      │
│  - WebSocket for terminal sessions                     │
└─────────────────────────────────────────────────────────┘
```

## Key Differences from Standalone SPA

| Aspect | Standalone SPA | MFE (Your Choice) |
|--------|----------------|-------------------|
| Build output | Complete app | SystemJS module |
| Entry point | index.html | app.js |
| Routing | Self-contained | Gateway handles |
| API proxy | nginx proxies | Gateway proxies |
| Dependencies | Bundled | Externalized (provided by gateway) |
| Deployment | Independent | Loaded by gateway |
| nginx config | Complex (SPA routing, proxy) | Simple (static files) |

## Prerequisites

- Docker 20.10+ with buildx support
- Docker registry access
- Kubernetes cluster with kubectl access
- **Gateway/shell application** that can load single-spa MFEs

## Building the MFE Image

### Quick Build (amd64)

```bash
cd /path/to/platform-manager
./helper-scripts/build-frontend-image.sh
```

This builds the MFE with:
- `BUILD_MODE=mfe` environment variable set
- SystemJS output format
- Externalized dependencies (vue, vue-router, pinia)
- Manifest file for discovery

### Custom Build

```bash
# Build for specific platform
PLATFORM=linux/amd64 ./helper-scripts/build-frontend-image.sh

# Build and push to registry
PUSH_IMAGE=true REGISTRY=your-registry.com ./helper-scripts/build-frontend-image.sh

# Custom image name and tag
IMAGE_NAME=my-mfe IMAGE_TAG=v1.0.0 ./helper-scripts/build-frontend-image.sh
```

### Using Makefile

```bash
# Build MFE
make web-docker-build

# Build multi-arch
make web-docker-build-multi

# Build and push
make web-docker-push
```

### Manual Docker Build

```bash
cd web

docker buildx build \
  --platform linux/amd64 \
  --tag localhost:5001/platform-manager-mfe:latest \
  --load \
  -f Dockerfile \
  .
```

## Testing the MFE Locally

### Run Container

```bash
# Using Makefile
make web-docker-run

# Or using script
docker run -p 8080:8080 localhost:5001/platform-manager-mfe:latest
```

### Verify MFE Assets

```bash
# Health check
curl http://localhost:8080/health
# Expected: "healthy"

# MFE manifest
curl http://localhost:8080/manifest.json
# Expected: JSON with name, version, main entry point

# MFE bundle (should be SystemJS format)
curl http://localhost:8080/js/app.js | head -20
# Expected: SystemJS module wrapper

# List all files
docker run --rm --entrypoint ls \
  localhost:5001/platform-manager-mfe:latest \
  -la /usr/share/nginx/html
```

### Test CORS Headers

```bash
# CORS should be enabled for gateway to load
curl -I http://localhost:8080/js/app.js | grep -i "access-control"
# Expected: Access-Control-Allow-Origin: *
```

## Deploying to Kubernetes

### Deploy MFE Service

```bash
# Apply deployment
kubectl apply -f config/frontend/deployment.yaml

# Or using Makefile
make web-deploy
```

### Verify Deployment

```bash
# Check pods
kubectl -n platform-system get pods -l app=platform-manager-mfe

# Check service
kubectl -n platform-system get svc platform-manager-mfe

# Check logs
kubectl -n platform-system logs -l app=platform-manager-mfe --tail=50

# Port forward for testing
kubectl -n platform-system port-forward svc/platform-manager-mfe 8080:80
```

### Get Service URL

```bash
# Internal cluster URL (for gateway to use)
echo "http://platform-manager-mfe.platform-system.svc.cluster.local"

# Or via service
kubectl -n platform-system get svc platform-manager-mfe -o jsonpath='{.spec.clusterIP}'
```

## Integrating with Gateway

Your gateway needs to:

### 1. Configure Import Map

Add the MFE to your gateway's import map:

```html
<script type="systemjs-importmap">
{
  "imports": {
    "@platform/manager-mfe": "http://platform-manager-mfe.platform-system.svc.cluster.local/js/app.js"
  }
}
</script>
```

Or dynamically from manifest:

```javascript
// Fetch MFE manifest
const manifest = await fetch('http://platform-manager-mfe/manifest.json');
const { name, main } = await manifest.json();

// Add to import map
System.import.addImportMap({
  imports: {
    [name]: `http://platform-manager-mfe${main}`
  }
});
```

### 2. Register with single-spa

```javascript
import { registerApplication, start } from 'single-spa';

registerApplication({
  name: '@platform/manager-mfe',
  app: () => System.import('@platform/manager-mfe'),
  activeWhen: ['/platform'],  // Or your routing logic
});

start();
```

### 3. Provide Shared Dependencies

Your gateway should provide these shared dependencies:

```javascript
// In your gateway's import map
{
  "imports": {
    "vue": "https://cdn.jsdelivr.net/npm/vue@3.4.0/dist/vue.esm-browser.js",
    "vue-router": "https://cdn.jsdelivr.net/npm/vue-router@4.2.5/dist/vue-router.esm-browser.js",
    "pinia": "https://cdn.jsdelivr.net/npm/pinia@2.1.7/dist/pinia.esm-browser.js",
    "single-spa-vue": "https://cdn.jsdelivr.net/npm/single-spa-vue@2.5.1/dist/esm/single-spa-vue.js"
  }
}
```

### 4. Configure API Proxy

Your gateway should proxy API calls to the backend:

```nginx
# In gateway's nginx config
location /api/ {
    proxy_pass http://platform-manager-backend:9080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    # ... other headers
}
```

## MFE Configuration

### Environment Variables

The MFE build respects:

- `BUILD_MODE=mfe` - Enables MFE build mode (set in Dockerfile)
- `VUE_APP_API_BASE_URL` - API base (should be relative in MFE, gateway handles)

### Externalized Dependencies

These are NOT bundled in the MFE (gateway provides them):

- `vue`
- `vue-router`
- `pinia`
- Other `@platform/*` packages

Configured in `vue.config.js`:

```javascript
externals: [
  'vue',
  'vue-router', 
  'pinia',
  /^@platform\/.+/,
]
```

### Output Format

The MFE outputs as:

- **Format**: SystemJS module
- **Library target**: `system`
- **No HTML**: Gateway loads it
- **No chunk splitting**: Single bundle

## MFE Manifest

The container serves a manifest at `/manifest.json`:

```json
{
  "name": "@platform/manager-mfe",
  "version": "0.1.0",
  "main": "/js/app.js",
  "type": "module",
  "format": "system"
}
```

Your gateway can fetch this to discover:
- MFE name/version
- Entry point path
- Module format

## Testing Integration

### Test MFE Loading

```javascript
// In browser console (on gateway page)
System.import('@platform/manager-mfe')
  .then(module => {
    console.log('MFE loaded:', module);
  })
  .catch(err => {
    console.error('Failed to load MFE:', err);
  });
```

### Test MFE Lifecycle

```javascript
// single-spa exposes lifecycle functions
const mfe = await System.import('@platform/manager-mfe');
console.log('Bootstrap:', mfe.bootstrap);
console.log('Mount:', mfe.mount);
console.log('Unmount:', mfe.unmount);
```

## Troubleshooting

### MFE Won't Load

**Check CORS headers:**
```bash
curl -I http://platform-mfe/js/app.js | grep -i access-control
```
Expected: `Access-Control-Allow-Origin: *`

**Check SystemJS format:**
```bash
curl http://platform-mfe/js/app.js | head -1
```
Should start with SystemJS wrapper

**Check gateway import map:**
```javascript
// In browser console
console.log(System.getConfig());
```

### Missing Dependencies

**Error: "vue is not defined"**
- Gateway must provide vue in import map
- Check gateway's shared dependencies

**Error: "Cannot find module '@platform/...'"**
- Gateway must provide other platform packages
- Or remove from externals in `vue.config.js`

### API Calls Fail

**MFE should NOT proxy API calls**
- Gateway handles all API routing
- MFE makes relative calls: `/api/tenants`
- Gateway proxies to backend

### Routing Issues

**MFE should NOT handle routing**
- Gateway controls all routing
- single-spa activates MFE based on route
- Check `activeWhen` in gateway's `registerApplication()`

## Performance Optimization

### Caching

The MFE container sets:
- **1 year cache** for JS/CSS (immutable)
- **No cache** for manifest.json
- **ETag support** for efficient updates

### CDN Integration

For production, use CDN:

```javascript
// In gateway import map
{
  "@platform/manager-mfe": "https://cdn.example.com/platform-mfe/v1.0.0/js/app.js"
}
```

### Preloading

Gateway can preload MFE:

```html
<link rel="preload" href="http://platform-mfe/js/app.js" as="script" crossorigin>
```

## Security Considerations

### CORS

- MFE enables CORS to allow gateway loading
- Restrict origins in production:
  ```nginx
  add_header Access-Control-Allow-Origin "https://gateway.example.com" always;
  ```

### Content Security Policy

Gateway should set CSP:
```
Content-Security-Policy: 
  script-src 'self' http://platform-mfe;
  connect-src 'self' http://backend-api;
```

### No Secrets in MFE

- MFE is publicly accessible
- Gateway handles authentication
- No API keys or secrets in MFE code

## CI/CD Integration

### Build Pipeline

```yaml
# Example GitHub Actions
name: Build MFE
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Build MFE image
      run: |
        REGISTRY=${{ secrets.REGISTRY }} \
        IMAGE_TAG=${{ github.sha }} \
        PUSH_IMAGE=true \
        ./helper-scripts/build-frontend-image.sh
```

### Version Management

Tag images with version:
```bash
IMAGE_TAG=v1.2.3 ./helper-scripts/build-frontend-image.sh
```

Update gateway's import map:
```javascript
{
  "@platform/manager-mfe": "http://platform-mfe/v1.2.3/js/app.js"
}
```

## Comparison: SPA vs MFE

### When to Use MFE (Your Case)

✅ **Use MFE when:**
- Gateway/shell app exists
- Multiple teams manage different UIs
- Need to deploy features independently
- Want to share common dependencies
- Dynamic composition required

### When to Use Standalone SPA

❌ **Don't use MFE when:**
- No gateway/shell exists
- Single team, single app
- Simpler deployment preferred
- No need for composition

## Summary

### What Changed for MFE

| File | Change |
|------|--------|
| `web/Dockerfile` | ✅ Sets `BUILD_MODE=mfe`, simpler nginx (no proxy), CORS headers |
| `helper-scripts/build-frontend-image.sh` | ✅ Updated image name, added manifest check |
| `config/frontend/deployment.yaml` | ✅ Renamed to `platform-manager-mfe`, reduced resources |
| `Makefile` | ✅ Updated targets for MFE |

### What Stayed the Same

- ✅ Multi-architecture support
- ✅ Non-root user
- ✅ Health checks
- ✅ Security hardening
- ✅ Gzip compression

### Gateway Integration Checklist

- [ ] Gateway import map configured
- [ ] MFE registered with single-spa
- [ ] Shared dependencies provided
- [ ] API proxy configured in gateway
- [ ] Routing activates MFE correctly
- [ ] CORS allows gateway origin

## Next Steps

1. **Build MFE image**: `./helper-scripts/build-frontend-image.sh`
2. **Deploy to K8s**: `make web-deploy`
3. **Get service URL**: For gateway configuration
4. **Configure gateway**: Add to import map
5. **Register MFE**: With single-spa in gateway
6. **Test integration**: Load MFE from gateway

## References

- single-spa: https://single-spa.js.org/
- SystemJS: https://github.com/systemjs/systemjs
- Import Maps: https://github.com/WICG/import-maps
- Vue + single-spa: https://single-spa.js.org/docs/ecosystem-vue/

