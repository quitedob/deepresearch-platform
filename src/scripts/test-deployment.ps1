# AI Research Platform - Deployment Integration Test Script (PowerShell)
# This script tests the deployment using docker-compose

$ErrorActionPreference = "Stop"

Write-Host "=== AI Research Platform Deployment Integration Tests ===" -ForegroundColor Green
Write-Host ""

function Print-Status {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Print-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Print-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

# Check if docker-compose is installed
if (-not (Get-Command docker-compose -ErrorAction SilentlyContinue)) {
    Print-Error "docker-compose is not installed"
    exit 1
}

# Check if .env file exists
if (-not (Test-Path .env)) {
    Print-Warning ".env file not found, creating from .env.example"
    Copy-Item .env.example .env
}

# Clean up any existing containers
Print-Status "Cleaning up existing containers..."
docker-compose down -v 2>$null

# Start services
Print-Status "Starting services with docker-compose..."
docker-compose up -d

# Wait for services to be healthy
Print-Status "Waiting for services to be healthy..."
Start-Sleep -Seconds 10

# Check if containers are running
Print-Status "Checking container status..."
$containers = docker-compose ps
if ($containers -notmatch "Up") {
    Print-Error "Containers are not running"
    docker-compose logs
    docker-compose down
    exit 1
}

# Wait for API to be ready
Print-Status "Waiting for API to be ready..."
$maxRetries = 30
$retryCount = 0
$apiReady = $false

while ($retryCount -lt $maxRetries) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 2
        if ($response.StatusCode -eq 200) {
            Print-Status "API is healthy!"
            $apiReady = $true
            break
        }
    }
    catch {
        # Continue waiting
    }
    
    $retryCount++
    Write-Host "Waiting for API... ($retryCount/$maxRetries)"
    Start-Sleep -Seconds 2
}

if (-not $apiReady) {
    Print-Error "API did not become healthy within timeout"
    docker-compose logs api
    docker-compose down
    exit 1
}

# Set environment variables for tests
$env:API_URL = "http://localhost:8080"
$env:DATABASE_HOST = "localhost"
$env:DATABASE_PORT = "5432"
$env:DATABASE_USER = "postgres"
$env:DATABASE_PASSWORD = "postgres"
$env:DATABASE_DBNAME = "ai_research_platform"
$env:SERVER_ENV = "development"

# Run integration tests
Print-Status "Running integration tests..."
go test -v ./test/integration/... -timeout 5m

$testExitCode = $LASTEXITCODE

# Show logs if tests failed
if ($testExitCode -ne 0) {
    Print-Error "Integration tests failed"
    Print-Status "Showing container logs..."
    docker-compose logs
}
else {
    Print-Status "All integration tests passed!"
}

# Cleanup
Print-Status "Cleaning up..."
docker-compose down -v

Write-Host ""
if ($testExitCode -eq 0) {
    Print-Status "=== Deployment tests completed successfully ==="
    exit 0
}
else {
    Print-Error "=== Deployment tests failed ==="
    exit 1
}
