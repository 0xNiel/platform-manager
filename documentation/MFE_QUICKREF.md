# MFE Deployment - Quick Reference

## Build & Deploy MFE

```bash
# Build MFE image (amd64)
./helper-scripts/build-frontend-image.sh

# Or with Makefile
make web-docker-build

# Deploy to Kubernetes
kubectl apply -f config/frontend/deployment.yaml

# Or with Makefile
make web-deploy
```

## Test MFE Locally

```bash
# Run container
docker run -p 8080:8080 localhost:5001/platform-manager-mfe:latest

# Test endpoints
curl http://localhost:8080/health           # Health check
curl http://localhost:8080/manifest.json    # MFE metadata
curl http://localhost:8080/js/app.js | head # SystemJS bundle

# Check CORS
curl -I http://localhost:8080/js/app.js | grep -i access-control
```

## Gateway Integration (Essential)

### 1. Add to Import Map

```html
<script type="systemjs-importmap">
{
  "imports": {
    "@platform/manager-mfe": "http://platform-manager-mfe/js/app.js",
    "vue": "https://cdn.jsdelivr.net/npm/vue@3.4.0/dist/vue.esm-browser.prod.js",
    "vue-router": "https://cdn.jsdelivr.net/npm/vue-router@4.2.5/dist/vue-router.esm-browser.js",
    "pinia": "https://cdn.jsdelivr.net/npm/pinia@2.1.7/dist/pinia.esm-browser.js"
  }
}
</script>
```

### 2. Register with single-spa

```javascript
import { registerApplication, start } from 'single-spa';

registerApplication({
  name: '@platform/manager-mfe',
  app: () => System.import('@platform/manager-mfe'),
  activeWhen: ['/platform'],
});

start();
```

### 3. Proxy API Calls

```nginx
# In gateway nginx
location /api/ {
    proxy_pass http://platform-manager-backend:9080;
    proxy_http_version 1.1;
}
```

## Service URLs

```bash
# Internal cluster URL (for gateway)
http://platform-manager-mfe.platform-system.svc.cluster.local

# Short form (within same namespace)
http://platform-manager-mfe

# Port forward for testing
kubectl -n platform-system port-forward svc/platform-manager-mfe 8080:80
```

## Verify Deployment

```bash
# Check pods
kubectl -n platform-system get pods -l app=platform-manager-mfe

# Check logs
kubectl -n platform-system logs -l app=platform-manager-mfe

# Test from inside cluster
kubectl run -it --rm test --image=alpine -- sh
apk add curl
curl http://platform-manager-mfe/manifest.json
```

## Test in Browser

```javascript
// Load MFE
System.import('@platform/manager-mfe')
  .then(m => console.log('✅ Loaded', m))
  .catch(e => console.error('❌ Failed', e));

// Check registration
import { getAppNames, getAppStatus } from 'single-spa';
console.log(getAppNames());
console.log(getAppStatus('@platform/manager-mfe'));
```

## Key Differences: MFE vs SPA

| Aspect | MFE | Standalone SPA |
|--------|-----|----------------|
| Build | `BUILD_MODE=mfe` | Regular build |
| Output | SystemJS module | index.html + assets |
| Routing | Gateway handles | Self-contained |
| API | Gateway proxies | nginx proxies |
| nginx | Simple (static files) | Complex (SPA + proxy) |
| CORS | Required | Not needed |
| Dependencies | Externalized | Bundled |

## Makefile Commands

```bash
make web-docker-build        # Build MFE image
make web-docker-build-multi  # Build multi-arch
make web-docker-push         # Build and push
make web-docker-run          # Run locally
make web-docker-stop         # Stop container
make web-deploy              # Deploy to K8s
make web-deploy-status       # Check deployment
```

## Troubleshooting

```bash
# Check MFE manifest
curl http://platform-mfe/manifest.json

# Verify SystemJS format
curl http://platform-mfe/js/app.js | head -20

# Check CORS headers
curl -I http://platform-mfe/js/app.js | grep -i access-control

# View MFE logs
kubectl -n platform-system logs -l app=platform-manager-mfe -f

# Shell into container
kubectl -n platform-system exec -it <pod-name> -- sh

# List files in container
docker run --rm --entrypoint ls localhost:5001/platform-manager-mfe:latest -la /usr/share/nginx/html
```

## Common Issues

**MFE won't load:**
- Check import map in gateway
- Verify service URL is accessible
- Check CORS headers

**Dependencies missing:**
- Gateway must provide: vue, vue-router, pinia
- Check versions match

**API calls fail:**
- Gateway must proxy /api/* to backend
- Check backend service is running

**Routing doesn't work:**
- Check activeWhen in gateway
- Verify URL matches condition

## Files Changed for MFE

✅ `web/Dockerfile` - MFE build, simple nginx, CORS
✅ `helper-scripts/build-frontend-image.sh` - Updated for MFE
✅ `config/frontend/deployment.yaml` - Renamed to platform-manager-mfe
✅ `Makefile` - Updated targets for MFE

## Important Notes

- ⚠️ MFE **requires** a gateway/shell application
- ⚠️ Gateway **must** provide shared dependencies
- ⚠️ Gateway **must** proxy API calls
- ⚠️ Gateway **must** handle routing
- ✅ MFE is just static files (SystemJS bundle)
- ✅ No API proxy in MFE container
- ✅ No SPA routing in MFE container

## Documentation

- `MFE_DEPLOYMENT_GUIDE.md` - Complete MFE guide
- `MFE_GATEWAY_INTEGRATION.md` - Gateway setup
- This file - Quick reference

## Next Steps

1. ✅ Build: `./helper-scripts/build-frontend-image.sh`
2. ✅ Deploy: `kubectl apply -f config/frontend/deployment.yaml`
3. 🔧 Configure gateway import map
4. 🔧 Register with single-spa
5. 🔧 Configure API proxy
6. ✅ Test integration

## Gateway Checklist

- [ ] SystemJS loaded
- [ ] Import map includes MFE
- [ ] Import map includes vue, vue-router, pinia
- [ ] Registered with single-spa
- [ ] activeWhen configured
- [ ] API proxy configured
- [ ] Backend service accessible
- [ ] CORS allows loading
- [ ] Test: System.import('@platform/manager-mfe')

## Success Criteria

✅ MFE image builds successfully
✅ Container runs and serves files
✅ `/manifest.json` returns metadata
✅ `/js/app.js` returns SystemJS module
✅ CORS headers present
✅ Gateway can load MFE
✅ single-spa activates MFE on route
✅ API calls work via gateway proxy
✅ No console errors

