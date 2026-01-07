# Frontend Docker - Quick Reference

## Build Image (amd64)

```bash
# Quick build
./helper-scripts/build-frontend-image.sh

# Build and push
PUSH_IMAGE=true REGISTRY=your-registry.com ./helper-scripts/build-frontend-image.sh
```

## Test Locally

```bash
# Build and test
./helper-scripts/test-frontend-docker.sh

# Or manually
docker run -p 8080:8080 localhost:5001/platform-manager-frontend:latest
open http://localhost:8080
```

## Deploy to Kubernetes

```bash
# Apply deployment
kubectl apply -f config/frontend/deployment.yaml

# Check status
kubectl -n platform-system get pods -l app=platform-manager-frontend

# Port forward
kubectl -n platform-system port-forward svc/platform-manager-frontend 8080:80
```

## Common Commands

```bash
# Build for specific platform
docker buildx build --platform linux/amd64 -t my-image:latest web/

# Build multi-arch
docker buildx build --platform linux/amd64,linux/arm64 -t my-image:latest --push web/

# Check image size
docker images localhost:5001/platform-manager-frontend:latest

# Run with backend
docker run -p 8080:8080 \
  -e VUE_APP_API_BASE_URL=http://host.docker.internal:9080 \
  localhost:5001/platform-manager-frontend:latest

# View logs
docker logs <container-id>

# Shell into container
docker run -it --entrypoint sh localhost:5001/platform-manager-frontend:latest
```

## Troubleshooting

```bash
# Build with no cache
docker buildx build --no-cache -f web/Dockerfile web/

# Check what's in the image
docker run --rm --entrypoint ls localhost:5001/platform-manager-frontend:latest -la /usr/share/nginx/html

# View nginx config
docker run --rm --entrypoint cat localhost:5001/platform-manager-frontend:latest /etc/nginx/conf.d/default.conf

# Test health endpoint
curl http://localhost:8080/health
```

## Files Created

- `web/Dockerfile` - Multi-stage Docker build
- `web/.dockerignore` - Exclude unnecessary files
- `config/frontend/deployment.yaml` - Kubernetes manifests
- `helper-scripts/build-frontend-image.sh` - Build script
- `helper-scripts/test-frontend-docker.sh` - Test script
- `documentation/FRONTEND_DOCKER_DEPLOYMENT.md` - Full documentation

## Image Details

- **Base**: nginx:1.25-alpine
- **Size**: ~25-30 MB
- **Port**: 8080
- **User**: nginx (101)
- **Health**: `/health`

## Next Steps

1. ✅ Build the image
2. ✅ Test locally
3. Update backend service name in deployment.yaml
4. Deploy to Kubernetes
5. Configure ingress for external access

