#!/bin/bash

# Build script for microservices platform

set -e

echo "🏗️  Building Distributed Microservices Platform..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

# Set variables
REGISTRY=${DOCKER_REGISTRY:-localhost:5000}
VERSION=${VERSION:-latest}

echo "📦 Registry: $REGISTRY"
echo "🏷️  Version: $VERSION"

# Build services
SERVICES=("api-gateway" "user-service" "order-service" "product-service" "payment-service" "notification-service")

for service in "${SERVICES[@]}"; do
    echo "🔨 Building $service..."
    
    if [ "$service" = "api-gateway" ]; then
        docker build -t $REGISTRY/$service:$VERSION -f $service/Dockerfile .
    else
        docker build -t $REGISTRY/$service:$VERSION -f services/$service/Dockerfile .
    fi
    
    echo "✅ Built $service"
done

echo "🎉 All services built successfully!"

# Optional: Push to registry
if [ "$PUSH_IMAGES" = "true" ]; then
    echo "🚀 Pushing images to registry..."
    for service in "${SERVICES[@]}"; do
        docker push $REGISTRY/$service:$VERSION
        echo "✅ Pushed $service"
    done
    echo "🎉 All images pushed to registry!"
fi

echo "✨ Build completed successfully!"