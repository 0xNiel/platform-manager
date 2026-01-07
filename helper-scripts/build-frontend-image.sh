#!/bin/bash
# Build and push frontend Docker image for amd64 platform (MFE mode)

set -e

# Configuration
IMAGE_NAME="${IMAGE_NAME:-platform-manager-mfe}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REGISTRY="${REGISTRY:-localhost:5001}"
PLATFORM="${PLATFORM:-linux/amd64}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🐳 Building Platform Manager MFE Docker Image${NC}"
echo "=================================================="
echo "Image: ${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
echo "Platform: ${PLATFORM}"
echo "Mode: Micro-Frontend (SystemJS)"
echo ""

# Navigate to web directory
cd "$(dirname "$0")/.."

# Build the image
echo -e "${BLUE}📦 Building Docker image (MFE mode)...${NC}"
docker buildx build \
    --platform "${PLATFORM}" \
    --tag "${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}" \
    --file web/Dockerfile \
    --load \
    web/

echo ""
echo -e "${GREEN}✅ Build complete!${NC}"

# Optional: Push to registry
if [ "${PUSH_IMAGE}" = "true" ]; then
    echo ""
    echo -e "${BLUE}📤 Pushing image to registry...${NC}"
    docker push "${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
    echo -e "${GREEN}✅ Push complete!${NC}"
fi

# Show image info
echo ""
echo -e "${BLUE}📊 Image Information:${NC}"
docker images "${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"

echo ""
echo -e "${GREEN}🎉 Done!${NC}"
echo ""
echo "MFE Details:"
echo "  Type: SystemJS module for single-spa"
echo "  Main entry: /js/app.js"
echo "  Manifest: /manifest.json"
echo ""
echo "To run locally:"
echo "  docker run -p 8080:8080 ${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
echo ""
echo "To test MFE bundle:"
echo "  curl http://localhost:8080/manifest.json"
echo "  curl http://localhost:8080/js/app.js"
echo ""
echo "To push to registry:"
echo "  PUSH_IMAGE=true ./helper-scripts/build-frontend-image.sh"
