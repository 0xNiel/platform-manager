#!/bin/bash
# Quick test script for frontend Docker image

set -e

IMAGE="${IMAGE:-localhost:5001/platform-manager-frontend:latest}"
PORT="${PORT:-8080}"

echo "🧪 Testing Frontend Docker Image"
echo "================================="
echo "Image: $IMAGE"
echo "Port: $PORT"
echo ""

# Check if image exists
if ! docker image inspect "$IMAGE" &> /dev/null; then
    echo "❌ Image not found. Building..."
    ./helper-scripts/build-frontend-image.sh
fi

# Stop any existing container
docker rm -f frontend-test 2>/dev/null || true

# Run container
echo "🚀 Starting container..."
docker run -d \
    --name frontend-test \
    -p "${PORT}:8080" \
    "$IMAGE"

# Wait for container to be ready
echo "⏳ Waiting for container to be ready..."
sleep 5

# Test health endpoint
echo ""
echo "✅ Testing health endpoint..."
if curl -f -s "http://localhost:${PORT}/health" > /dev/null; then
    echo "✅ Health check passed"
else
    echo "❌ Health check failed"
    docker logs frontend-test
    docker rm -f frontend-test
    exit 1
fi

# Test main page
echo ""
echo "✅ Testing main page..."
if curl -f -s "http://localhost:${PORT}/" > /dev/null; then
    echo "✅ Main page loads"
else
    echo "❌ Main page failed to load"
    docker logs frontend-test
    docker rm -f frontend-test
    exit 1
fi

# Show logs
echo ""
echo "📋 Container logs:"
docker logs frontend-test | tail -20

# Show access info
echo ""
echo "✅ Frontend is running!"
echo ""
echo "Access at: http://localhost:${PORT}"
echo ""
echo "To stop:"
echo "  docker rm -f frontend-test"
echo ""
echo "To view logs:"
echo "  docker logs -f frontend-test"

