#!/bin/bash

# Deployment script for Kubernetes

set -e

echo "🚀 Deploying Distributed Microservices Platform to Kubernetes..."

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed. Please install kubectl and try again."
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info > /dev/null 2>&1; then
    echo "❌ Cannot connect to Kubernetes cluster. Please check your kubeconfig."
    exit 1
fi

echo "📝 Deploying namespace..."
kubectl apply -f k8s/namespace.yaml

echo "🔧 Deploying configmaps and secrets..."
if [ -d "k8s/configmaps" ]; then
    kubectl apply -f k8s/configmaps/
fi

if [ -d "k8s/secrets" ]; then
    kubectl apply -f k8s/secrets/
fi

echo "🗄️  Deploying persistent volumes..."
if [ -f "k8s/storage/postgres-pv.yaml" ]; then
    kubectl apply -f k8s/storage/
fi

echo "🎯 Deploying services..."
kubectl apply -f k8s/services/

echo "🚀 Deploying applications..."
kubectl apply -f k8s/deployments/

echo "📊 Deploying monitoring stack..."
kubectl apply -f monitoring/monitoring-stack.yaml

# Optional: Deploy Istio service mesh
if [ "$DEPLOY_ISTIO" = "true" ]; then
    echo "🕸️  Deploying Istio service mesh..."
    
    # Check if Istio is installed
    if ! command -v istioctl &> /dev/null; then
        echo "⚠️  istioctl is not installed. Skipping Istio deployment."
    else
        # Install Istio if not already installed
        istioctl install --set values.defaultRevision=default -y
        
        # Deploy Istio configurations
        kubectl apply -f istio/
        
        echo "✅ Istio service mesh deployed successfully!"
    fi
fi

echo "⏳ Waiting for deployments to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment --all -n microservices

echo "📋 Deployment status:"
kubectl get pods -n microservices

echo "🌐 Service endpoints:"
kubectl get services -n microservices

echo "✨ Deployment completed successfully!"

# Show access information
API_GATEWAY_IP=$(kubectl get service api-gateway -n microservices -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
if [ -n "$API_GATEWAY_IP" ]; then
    echo "🎯 API Gateway is accessible at: http://$API_GATEWAY_IP"
else
    echo "🎯 API Gateway service is running. Use 'kubectl port-forward' to access it locally:"
    echo "   kubectl port-forward -n microservices service/api-gateway 8080:80"
fi

PROMETHEUS_IP=$(kubectl get service prometheus -n microservices -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
if [ -n "$PROMETHEUS_IP" ]; then
    echo "📊 Prometheus is accessible at: http://$PROMETHEUS_IP:9090"
else
    echo "📊 To access Prometheus locally:"
    echo "   kubectl port-forward -n microservices service/prometheus 9090:9090"
fi

JAEGER_IP=$(kubectl get service jaeger -n microservices -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
if [ -n "$JAEGER_IP" ]; then
    echo "🔍 Jaeger UI is accessible at: http://$JAEGER_IP:16686"
else
    echo "🔍 To access Jaeger UI locally:"
    echo "   kubectl port-forward -n microservices service/jaeger 16686:16686"
fi