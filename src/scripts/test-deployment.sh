#!/bin/bash

# AI Research Platform - Deployment Integration Test Script
# This script tests the deployment using docker-compose

set -e

echo "=== AI Research Platform Deployment Integration Tests ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    print_error "docker-compose is not installed"
    exit 1
fi

# Check if .env file exists
if [ ! -f .env ]; then
    print_warning ".env file not found, creating from .env.example"
    cp .env .env
fi

# Clean up any existing containers
print_status "Cleaning up existing containers..."
docker-compose down -v 2>/dev/null || true

# Start services
print_status "Starting services with docker-compose..."
docker-compose up -d

# Wait for services to be healthy
print_status "Waiting for services to be healthy..."
sleep 10

# Check if containers are running
print_status "Checking container status..."
if ! docker-compose ps | grep -q "Up"; then
    print_error "Containers are not running"
    docker-compose logs
    docker-compose down
    exit 1
fi

# Wait for API to be ready
print_status "Waiting for API to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        print_status "API is healthy!"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo "Waiting for API... ($RETRY_COUNT/$MAX_RETRIES)"
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    print_error "API did not become healthy within timeout"
    docker-compose logs api
    docker-compose down
    exit 1
fi

# Run integration tests
print_status "Running integration tests..."
export API_URL="http://localhost:8080"
export DATABASE_HOST="localhost"
export DATABASE_PORT="5432"
export DATABASE_USER="postgres"
export DATABASE_PASSWORD="postgres"
export DATABASE_DBNAME="ai_research_platform"
export SERVER_ENV="development"

go test -v ./test/integration/... -timeout 5m

TEST_EXIT_CODE=$?

# Show logs if tests failed
if [ $TEST_EXIT_CODE -ne 0 ]; then
    print_error "Integration tests failed"
    print_status "Showing container logs..."
    docker-compose logs
else
    print_status "All integration tests passed!"
fi

# Cleanup
print_status "Cleaning up..."
docker-compose down -v

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo ""
    print_status "=== Deployment tests completed successfully ==="
    exit 0
else
    echo ""
    print_error "=== Deployment tests failed ==="
    exit 1
fi
