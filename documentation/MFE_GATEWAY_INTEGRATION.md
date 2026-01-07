# Gateway Integration Guide - Platform Manager MFE

Quick guide for integrating the Platform Manager MFE into your gateway application.

## Overview

The Platform Manager is a **single-spa micro-frontend** that your gateway loads dynamically at runtime.

## Prerequisites

- ✅ Platform Manager MFE deployed to Kubernetes
- ✅ Gateway application with single-spa integration
- ✅ SystemJS or similar module loader in gateway

## Step 1: Deploy the MFE

```bash
# Build and deploy
./helper-scripts/build-frontend-image.sh
kubectl apply -f config/frontend/deployment.yaml

# Get service URL (internal cluster URL)
echo "http://platform-manager-mfe.platform-system.svc.cluster.local"
```

## Step 2: Configure Gateway Import Map

Add the MFE to your gateway's SystemJS import map:

### Option A: Static Import Map

```html
<!-- In your gateway's index.html -->
<script type="systemjs-importmap">
{
  "imports": {
    "@platform/manager-mfe": "http://platform-manager-mfe.platform-system.svc.cluster.local/js/app.js",
    
    "vue": "https://cdn.jsdelivr.net/npm/vue@3.4.0/dist/vue.esm-browser.prod.js",
    "vue-router": "https://cdn.jsdelivr.net/npm/vue-router@4.2.5/dist/vue-router.esm-browser.js",
    "pinia": "https://cdn.jsdelivr.net/npm/pinia@2.1.7/dist/pinia.esm-browser.js",
    "single-spa-vue": "https://cdn.jsdelivr.net/npm/single-spa-vue@2.5.1/dist/esm/single-spa-vue.js"
  }
}
</script>
```

### Option B: Dynamic Import Map (Recommended)

```javascript
// In your gateway code
async function loadMFEManifest() {
  const response = await fetch('http://platform-manager-mfe/manifest.json');
  const manifest = await response.json();
  
  // Add to SystemJS import map
  System.applyImportMap({
    imports: {
      [manifest.name]: `http://platform-manager-mfe${manifest.main}`
    }
  });
  
  return manifest;
}

// Load before starting single-spa
await loadMFEManifest();
```

## Step 3: Register MFE with single-spa

```javascript
// In your gateway's app initialization
import { registerApplication, start } from 'single-spa';

registerApplication({
  name: '@platform/manager-mfe',
  app: () => System.import('@platform/manager-mfe'),
  activeWhen: ['/platform'],  // Adjust based on your routing
  customProps: {
    // Optional: Pass data to MFE
    apiBaseUrl: '/api',
    authToken: () => getAuthToken(),
  }
});

// Start single-spa
start({
  urlRerouteOnly: true,
});
```

## Step 4: Configure Gateway Routing

### Option A: Path-based Activation

```javascript
registerApplication({
  name: '@platform/manager-mfe',
  app: () => System.import('@platform/manager-mfe'),
  // Activates when URL path starts with /platform
  activeWhen: ['/platform'],
});
```

### Option B: Custom Activation Function

```javascript
registerApplication({
  name: '@platform/manager-mfe',
  app: () => System.import('@platform/manager-mfe'),
  activeWhen: (location) => {
    // Custom logic
    return location.pathname.startsWith('/platform') ||
           location.pathname === '/tenants' ||
           location.pathname === '/resources';
  },
});
```

### Option C: Always Active

```javascript
registerApplication({
  name: '@platform/manager-mfe',
  app: () => System.import('@platform/manager-mfe'),
  activeWhen: () => true,  // Always loaded
});
```

## Step 5: Configure API Proxy in Gateway

The MFE makes API calls to `/api/*`. Your gateway must proxy these to the backend.

### nginx Configuration (if gateway uses nginx)

```nginx
# In gateway's nginx config
location /api/ {
    proxy_pass http://platform-manager-backend.platform-system.svc.cluster.local:9080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    proxy_set_header Host $host;
    proxy_cache_bypass $http_upgrade;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}

# WebSocket for terminal
location /api/terminal/ {
    proxy_pass http://platform-manager-backend.platform-system.svc.cluster.local:9080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 3600s;
    proxy_send_timeout 3600s;
}
```

### Express/Node.js Gateway

```javascript
const { createProxyMiddleware } = require('http-proxy-middleware');

app.use('/api', createProxyMiddleware({
  target: 'http://platform-manager-backend.platform-system.svc.cluster.local:9080',
  changeOrigin: true,
  ws: true,  // Enable WebSocket proxy
}));
```

### Spring Cloud Gateway

```yaml
spring:
  cloud:
    gateway:
      routes:
        - id: platform-api
          uri: http://platform-manager-backend.platform-system.svc.cluster.local:9080
          predicates:
            - Path=/api/**
```

## Step 6: Provide Shared Dependencies

Your gateway must provide these dependencies that the MFE expects:

### Required Dependencies

```javascript
// In gateway's import map
{
  "imports": {
    // Core Vue
    "vue": "https://cdn.jsdelivr.net/npm/vue@3.4.0/dist/vue.esm-browser.prod.js",
    
    // Vue Router
    "vue-router": "https://cdn.jsdelivr.net/npm/vue-router@4.2.5/dist/vue-router.esm-browser.js",
    
    // State management
    "pinia": "https://cdn.jsdelivr.net/npm/pinia@2.1.7/dist/pinia.esm-browser.js",
    
    // single-spa integration
    "single-spa-vue": "https://cdn.jsdelivr.net/npm/single-spa-vue@2.5.1/dist/esm/single-spa-vue.js",
    
    // The MFE itself
    "@platform/manager-mfe": "http://platform-manager-mfe/js/app.js"
  }
}
```

### Version Compatibility

Ensure versions match what the MFE expects:

```bash
# Check MFE's dependencies
cat web/package.json | jq '.dependencies'
```

Expected versions:
- Vue: ^3.4.0
- Vue Router: ^4.2.5
- Pinia: ^2.1.7
- single-spa-vue: ^2.5.1

## Step 7: Test Integration

### Test MFE Loading

```javascript
// In browser console (on gateway page)
System.import('@platform/manager-mfe')
  .then(module => {
    console.log('✅ MFE loaded successfully');
    console.log('Lifecycle:', {
      bootstrap: typeof module.bootstrap,
      mount: typeof module.mount,
      unmount: typeof module.unmount
    });
  })
  .catch(err => {
    console.error('❌ Failed to load MFE:', err);
  });
```

### Verify single-spa Registration

```javascript
// In browser console
import { getAppNames, getAppStatus } from 'single-spa';

console.log('Registered apps:', getAppNames());
console.log('MFE status:', getAppStatus('@platform/manager-mfe'));
```

### Test Navigation

```javascript
// Navigate to MFE route
window.location.href = '/platform';

// Or programmatically
import { navigateToUrl } from 'single-spa';
navigateToUrl('/platform');
```

## Complete Gateway Example

Here's a minimal working gateway:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Platform Gateway</title>
  
  <!-- SystemJS -->
  <script src="https://cdn.jsdelivr.net/npm/systemjs@6.14.1/dist/system.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/systemjs@6.14.1/dist/extras/amd.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/systemjs@6.14.1/dist/extras/named-exports.min.js"></script>
  
  <!-- Import Map -->
  <script type="systemjs-importmap">
  {
    "imports": {
      "single-spa": "https://cdn.jsdelivr.net/npm/single-spa@5.9.5/lib/system/single-spa.min.js",
      "vue": "https://cdn.jsdelivr.net/npm/vue@3.4.0/dist/vue.esm-browser.prod.js",
      "vue-router": "https://cdn.jsdelivr.net/npm/vue-router@4.2.5/dist/vue-router.esm-browser.js",
      "pinia": "https://cdn.jsdelivr.net/npm/pinia@2.1.7/dist/pinia.esm-browser.js",
      "single-spa-vue": "https://cdn.jsdelivr.net/npm/single-spa-vue@2.5.1/dist/esm/single-spa-vue.js",
      "@platform/manager-mfe": "http://platform-manager-mfe.platform-system.svc.cluster.local/js/app.js"
    }
  }
  </script>
</head>
<body>
  <nav>
    <a href="/">Home</a>
    <a href="/platform">Platform Manager</a>
  </nav>
  
  <div id="single-spa-application:@platform/manager-mfe"></div>
  
  <script>
    System.import('single-spa').then(({ registerApplication, start }) => {
      registerApplication({
        name: '@platform/manager-mfe',
        app: () => System.import('@platform/manager-mfe'),
        activeWhen: ['/platform'],
      });
      
      start();
    });
  </script>
</body>
</html>
```

## Troubleshooting

### MFE Not Loading

**Check import map:**
```javascript
console.log(System.getConfig());
```

**Check network:**
```bash
# From gateway pod
curl http://platform-manager-mfe.platform-system.svc.cluster.local/manifest.json
```

**Check CORS:**
```javascript
fetch('http://platform-manager-mfe/js/app.js')
  .then(r => console.log('✅ CORS OK'))
  .catch(e => console.error('❌ CORS issue:', e));
```

### Dependencies Missing

**Error: "vue is not defined"**
- Add vue to gateway's import map
- Check version compatibility

**Error: "Cannot find '@platform/manager-mfe'"**
- Verify MFE is in import map
- Check service URL is accessible

### Routing Issues

**MFE not activating:**
- Check `activeWhen` condition
- Verify URL matches condition
- Check browser console for errors

**Wrong MFE loads:**
- Review all registered apps
- Check for conflicting `activeWhen` conditions

### API Calls Failing

**404 on /api/***:
- Verify gateway proxies `/api/*` to backend
- Check backend service is running

**CORS errors:**
- Gateway proxy should handle CORS
- Backend should allow gateway origin

## Production Considerations

### Security

```nginx
# Restrict CORS to gateway only
add_header Access-Control-Allow-Origin "https://gateway.example.com" always;
```

### Caching

```html
<!-- Preload MFE for better performance -->
<link rel="preload" href="http://platform-mfe/js/app.js" as="script" crossorigin>
```

### Monitoring

```javascript
// Log MFE lifecycle events
registerApplication({
  name: '@platform/manager-mfe',
  app: async () => {
    console.time('mfe-load');
    const module = await System.import('@platform/manager-mfe');
    console.timeEnd('mfe-load');
    return module;
  },
  activeWhen: ['/platform'],
});
```

### Error Handling

```javascript
import { addErrorHandler } from 'single-spa';

addErrorHandler(err => {
  if (err.appOrParcelName === '@platform/manager-mfe') {
    console.error('Platform Manager MFE error:', err);
    // Send to error tracking service
  }
});
```

## Gateway Configuration Checklist

- [ ] SystemJS loaded in gateway
- [ ] Import map includes MFE and dependencies
- [ ] MFE registered with single-spa
- [ ] Routing configured (`activeWhen`)
- [ ] API proxy configured
- [ ] Shared dependencies provided (vue, vue-router, pinia)
- [ ] CORS allows MFE loading
- [ ] Network access from gateway to MFE service
- [ ] Backend API accessible via gateway proxy

## Summary

Your gateway needs to:

1. **Load** the MFE via SystemJS
2. **Register** it with single-spa
3. **Activate** it based on route
4. **Provide** shared dependencies
5. **Proxy** API calls to backend

The MFE container:
- ✅ Serves SystemJS bundle
- ✅ Enables CORS
- ✅ Provides manifest
- ✅ Caches aggressively

## Next Steps

1. Deploy MFE: `kubectl apply -f config/frontend/deployment.yaml`
2. Get service URL
3. Update gateway import map
4. Register with single-spa
5. Test integration
6. Configure API proxy
7. Go live!

## Support

- Full guide: `documentation/MFE_DEPLOYMENT_GUIDE.md`
- Gateway integration: This file
- Troubleshooting: Check MFE logs, gateway logs, browser console

