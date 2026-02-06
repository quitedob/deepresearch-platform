#!/bin/bash

# AI Research Platform - Kubernetes Deployment Integration Test Script
# This script tests the Kubernetes deployment

set -e

echo "=== AI Research Platform Kubernetes Deployment Tests ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="ai-research-platform"
DEPLOYMENT_NAME="ai-research-api"
SERVICE_NAME="ai-research-api-service"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed"
    exit 1
fi

# Check if namespace exists
print_status "Checking if namespace exists..."
if ! kubectl get namespace $NAMESPACE &> /dev/null; then
    print_error "Namespace $NAMESPACE does not exist. Please deploy first."
    exit 1
fi

# Check deployment status
print_status "Checking deployment status..."
if ! kubectl get deployment $DEPLOYMENT_NAME -n $NAMESPACE &> /dev/null; then
    print_error "Deployment $DEPLOYMENT_NAME not found"
    exit 1
fi

# Wait for deployment to be ready
print_status "Waiting for deployment to be ready..."
kubectl wait --for=condition=available --timeout=300s \
    deployment/$DEPLOYMENT_NAME -n $NAMESPACE

# Check if pods are running
print_status "Checking pod status..."
RUNNING_PODS=$(kubectl get pods -n $NAMESPACE -l app=ai-research-api \
    --field-selector=status.phase=Running --no-headers | wc -l)

if [ $RUNNING_PODS -eq 0 ]; then
    print_error "No running pods found"
    kubectl get pods -n $NAMESPACE -l app=ai-research-api
    exit 1
fi

print_status "Found $RUNNING_PODS running pod(s)"

# Check service
print_status "Checking service..."
if ! kubectl get service $SERVICE_NAME -n $NAMESPACE &> /dev/null; then
    print_error "Service $SERVICE_NAME not found"
    exit 1
fi

# Port forward to test the service
print_status "Setting up port forward..."
kubectl port-forward -n $NAMESPACE svc/$SERVICE_NAME 8080:80 &
PORT_FORWARD_PID=$!

# Wait for port forward to be ready
sleep 5

# Function to cleanup port forward
cleanup() {
    print_status "Cleaning up port forward..."
    kill $PORT_FORWARD_PID 2>/dev/null || true
}

trap cleanup EXIT

# Test health endpoint
print_status "Testing health endpoint..."
MAX_RETRIES=10
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        print_status "Health check passed!"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo "Waiting for health endpoint... ($RETRY_COUNT/$MAX_RETRIES)"
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    print_error "Health endpoint did not respond"
    exit 1
fi

# Test readiness endpoint
print_status "Testing readiness endpoint..."
if curl -s http://localhost:8080/ready | grep -q "ready"; then
    print_status "Readiness check passed!"
else
    print_warning "Readiness check returned unexpected response"
fi

# Check HPA
print_status "Checking HorizontalPodAutoscaler..."
if kubectl get hpa -n $NAMESPACE &> /dev/null; then
    kubectl get hpa -n $NAMESPACE
else
    print_warning "HPA not found"
fi

# Check PVC
print_status "Checking PersistentVolumeClaims..."
kubectl get pvc -n $NAMESPACE

# Check ConfigMap
print_status "Checking ConfigMap..."
if kubectl get configmap ai-research-config -n $NAMESPACE &> /dev/null; then
    print_status "ConfigMap exists"
else
    print_error "ConfigMap not found"
    exit 1
fi

# Check Secrets
print_status "Checking Secrets..."
if kubectl get secret ai-research-secrets -n $NAMESPACE &> /dev/null; then
    print_status "Secrets exist"
else
    print_error "Secrets not found"
    exit 1
fi

# Check pod logs for errors
print_status "Checking pod logs for errors..."
POD_NAME=$(kubectl get pods -n $NAMESPACE -l app=ai-research-api \
    --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')

if kubectl logs $POD_NAME -n $NAMESPACE --tail=50 | grep -i "error\|fatal\|panic"; then
    print_warning "Found errors in pod logs"
else
    print_status "No errors found in recent logs"
fi

# Test environment variable injection
print_status "Testing environment variable injection..."
ENV_VARS=$(kubectl exec $POD_NAME -n $NAMESPACE -- env | grep -E "DATABASE_|REDIS_|SERVER_" || true)
if [ -n "$ENV_VARS" ]; then
    print_status "Environment variables are injected"
else
    print_warning "Could not verify environment variables"
fi

# Check resource usage
print_status "Checking resource usage..."
kubectl top pods -n $NAMESPACE -l app=ai-research-api || print_warning "Metrics not available"

# Test service discovery
print_status "Testing service discovery..."
kubectl exec $POD_NAME -n $NAMESPACE -- nslookup postgres-service || print_warning "DNS lookup failed"

# Check ingress
print_status "Checking Ingress..."
if kubectl get ingress -n $NAMESPACE &> /dev/null; then
    kubectl get ingress -n $NAMESPACE
else
    print_warning "Ingress not found"
fi

echo ""
print_status "=== Kubernetes deployment tests completed successfully ==="
