# Frontend Docker Deployment Checklist

## Pre-Deployment

- [ ] Node.js dependencies installed locally (`cd web && npm install`)
- [ ] Frontend builds successfully (`cd web && npm run build`)
- [ ] Docker installed and running
- [ ] Docker buildx available (`docker buildx version`)
- [ ] Access to Docker registry (if pushing)
- [ ] Kubernetes cluster access (if deploying to K8s)

## Build Phase

- [ ] **Build Docker image**
  ```bash
  ./helper-scripts/build-frontend-image.sh
  ```
  Expected output: Image built successfully

- [ ] **Verify image exists**
  ```bash
  docker images | grep platform-manager-frontend
  ```
  Should show: `localhost:5001/platform-manager-frontend   latest   ...`

- [ ] **Check image size**
  ```bash
  docker images localhost:5001/platform-manager-frontend:latest
  ```
  Expected: ~25-30 MB

## Local Testing

- [ ] **Run container locally**
  ```bash
  ./helper-scripts/test-frontend-docker.sh
  ```
  Expected: Container runs, health check passes

- [ ] **Test health endpoint**
  ```bash
  curl http://localhost:8080/health
  ```
  Expected: `healthy`

- [ ] **Test main page**
  ```bash
  curl -I http://localhost:8080/
  ```
  Expected: `200 OK` with `Content-Type: text/html`

- [ ] **Open in browser**
  ```bash
  open http://localhost:8080
  ```
  Expected: Platform Manager UI loads

- [ ] **Check API routing** (if backend running)
  ```bash
  curl http://localhost:8080/api/tenants
  ```
  Expected: Proxied to backend, returns data or auth error

- [ ] **Stop test container**
  ```bash
  docker rm -f frontend-test
  ```

## Registry Push (Optional)

- [ ] **Configure registry**
  ```bash
  export REGISTRY=your-registry.com
  export FRONTEND_IMG=${REGISTRY}/platform-manager-frontend:latest
  ```

- [ ] **Build and push**
  ```bash
  PUSH_IMAGE=true ./helper-scripts/build-frontend-image.sh
  ```

- [ ] **Verify in registry**
  ```bash
  docker pull ${FRONTEND_IMG}
  ```

## Kubernetes Deployment

### Prerequisites

- [ ] **Namespace exists**
  ```bash
  kubectl create namespace platform-system --dry-run=client -o yaml | kubectl apply -f -
  ```

- [ ] **Backend is deployed** (or update service name)
  ```bash
  kubectl -n platform-system get svc platform-manager-backend
  ```
  If backend has different name, update `config/frontend/deployment.yaml`

- [ ] **Update backend service name** in deployment.yaml (if needed)
  Edit: `config/frontend/deployment.yaml`
  Find: `proxy_pass http://platform-manager-backend:9080;`
  Change to your backend service name

### Deploy

- [ ] **Apply deployment**
  ```bash
  kubectl apply -f config/frontend/deployment.yaml
  ```
  Expected: deployment, service, and ingress created

- [ ] **Wait for pods to be ready**
  ```bash
  kubectl -n platform-system get pods -l app=platform-manager-frontend -w
  ```
  Wait until: `STATUS: Running` and `READY: 1/1`

- [ ] **Check deployment status**
  ```bash
  kubectl -n platform-system rollout status deployment/platform-manager-frontend
  ```
  Expected: `deployment "platform-manager-frontend" successfully rolled out`

- [ ] **Check pod logs**
  ```bash
  kubectl -n platform-system logs -l app=platform-manager-frontend --tail=50
  ```
  Expected: nginx startup logs, no errors

### Verification

- [ ] **Port forward to test**
  ```bash
  kubectl -n platform-system port-forward svc/platform-manager-frontend 8080:80
  ```

- [ ] **Test health endpoint**
  ```bash
  curl http://localhost:8080/health
  ```
  Expected: `healthy`

- [ ] **Open in browser**
  ```bash
  open http://localhost:8080
  ```
  Expected: UI loads

- [ ] **Test navigation**
  - [ ] Dashboard loads
  - [ ] Tenants page loads
  - [ ] Resources page loads
  - [ ] No console errors

- [ ] **Test API integration** (if backend deployed)
  - [ ] Tenants list shows data
  - [ ] Resource counts display
  - [ ] Health checks work
  - [ ] Actions trigger (if applicable)

## Ingress Configuration (Optional)

- [ ] **Update ingress host** in `config/frontend/deployment.yaml`
  ```yaml
  spec:
    rules:
    - host: platform-manager.your-domain.com  # Change this
  ```

- [ ] **Apply ingress**
  ```bash
  kubectl apply -f config/frontend/deployment.yaml
  ```

- [ ] **Configure DNS**
  Point `platform-manager.your-domain.com` to ingress IP

- [ ] **Test external access**
  ```bash
  curl https://platform-manager.your-domain.com/health
  ```

- [ ] **Verify TLS** (if configured)
  ```bash
  curl -v https://platform-manager.your-domain.com 2>&1 | grep -i certificate
  ```

## Production Checklist

- [ ] **Resource limits configured**
  Check in `config/frontend/deployment.yaml`:
  - CPU limits: 200m
  - Memory limits: 128Mi

- [ ] **Replicas set appropriately**
  Default: 2 replicas (adjust based on load)

- [ ] **Health checks configured**
  - Liveness probe: ✅
  - Readiness probe: ✅

- [ ] **Security context set**
  - Non-root user: ✅
  - Read-only root filesystem: ❌ (nginx needs cache)
  - Drop all capabilities: ✅

- [ ] **Monitoring configured** (if applicable)
  - Prometheus metrics
  - Log aggregation
  - Error tracking

- [ ] **Backup strategy** (for data/config)
  - ConfigMaps backed up
  - Secrets backed up
  - Deployment manifests in Git

## Troubleshooting Commands

If something goes wrong:

```bash
# Check pod status
kubectl -n platform-system get pods -l app=platform-manager-frontend

# Describe pod (shows events)
kubectl -n platform-system describe pod -l app=platform-manager-frontend

# Check logs
kubectl -n platform-system logs -l app=platform-manager-frontend --tail=100

# Get into container (if needed)
kubectl -n platform-system exec -it <pod-name> -- sh

# Check service
kubectl -n platform-system get svc platform-manager-frontend
kubectl -n platform-system describe svc platform-manager-frontend

# Check endpoints (should show pod IPs)
kubectl -n platform-system get endpoints platform-manager-frontend

# Force restart
kubectl -n platform-system rollout restart deployment/platform-manager-frontend

# Delete and recreate
kubectl delete -f config/frontend/deployment.yaml
kubectl apply -f config/frontend/deployment.yaml
```

## Rollback Procedure

If deployment fails:

```bash
# Check rollout history
kubectl -n platform-system rollout history deployment/platform-manager-frontend

# Rollback to previous version
kubectl -n platform-system rollout undo deployment/platform-manager-frontend

# Rollback to specific revision
kubectl -n platform-system rollout undo deployment/platform-manager-frontend --to-revision=2
```

## Success Criteria

✅ All checks passed when:
- Image builds without errors
- Container runs locally
- Health endpoint returns 200
- UI loads in browser
- Pods are running in K8s
- Service is accessible
- API calls work (if backend deployed)
- No errors in logs

## Common Issues and Solutions

### Issue: npm install fails during build
**Solution**: Clear Docker cache
```bash
docker buildx build --no-cache -f web/Dockerfile web/
```

### Issue: Image too large
**Solution**: Check `.dockerignore` includes `node_modules/`, `dist/`

### Issue: Container fails to start
**Solution**: Check logs
```bash
docker logs <container-id>
```

### Issue: API calls return 502
**Solution**: Verify backend service name in nginx config

### Issue: WebSocket connection fails
**Solution**: Check nginx timeout settings (should be high)

### Issue: Pod won't start (CrashLoopBackOff)
**Solution**: Check image pull, check logs
```bash
kubectl -n platform-system describe pod <pod-name>
kubectl -n platform-system logs <pod-name>
```

### Issue: Service not accessible
**Solution**: Check service and endpoints
```bash
kubectl -n platform-system get svc,endpoints
```

## Next Steps After Successful Deployment

1. [ ] Set up monitoring alerts
2. [ ] Configure autoscaling (HPA)
3. [ ] Add network policies
4. [ ] Implement backup strategy
5. [ ] Document runbook
6. [ ] Train team on deployment process

## Additional Resources

- Full Guide: `documentation/FRONTEND_DOCKER_DEPLOYMENT.md`
- Quick Ref: `documentation/FRONTEND_DOCKER_QUICKREF.md`
- Summary: `documentation/FRONTEND_DOCKER_SETUP_COMPLETE.md`
- This Checklist: `documentation/FRONTEND_DEPLOYMENT_CHECKLIST.md`

